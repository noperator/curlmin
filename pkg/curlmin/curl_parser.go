package curlmin

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

// CurlCommand represents a curl command with its syntax tree
type CurlCommand struct {
	Program *syntax.File
	Command *syntax.CallExpr
}

// ParseCurlCommand parses a curl command string into a syntax tree
func ParseCurlCommand(curlCmd string) (*CurlCommand, error) {
	// Make sure the command starts with curl
	curlCmd = strings.TrimSpace(curlCmd)
	if !strings.HasPrefix(curlCmd, "curl ") {
		curlCmd = "curl " + curlCmd
	}

	parser := syntax.NewParser()
	reader := strings.NewReader(curlCmd)
	prog, err := parser.Parse(reader, "")
	if err != nil {
		return nil, fmt.Errorf("failed to parse shell command: %w", err)
	}

	if len(prog.Stmts) == 0 {
		return nil, fmt.Errorf("no statements found in command")
	}

	// Get the first statement
	stmt := prog.Stmts[0]

	// Check if it's a command
	if stmt.Cmd == nil {
		return nil, fmt.Errorf("not a command")
	}

	// Try to get it as a CallExpr (command with arguments)
	callExpr, ok := stmt.Cmd.(*syntax.CallExpr)
	if !ok {
		return nil, fmt.Errorf("not a call expression")
	}

	// Verify it's a curl command
	if len(callExpr.Args) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	firstArg := callExpr.Args[0]
	var buf bytes.Buffer
	printer := syntax.NewPrinter()
	err = printer.Print(&buf, firstArg)
	if err != nil {
		return nil, fmt.Errorf("failed to print first argument: %w", err)
	}

	if !strings.Contains(strings.ToLower(buf.String()), "curl") {
		return nil, fmt.Errorf("not a curl command")
	}

	return &CurlCommand{
		Program: prog,
		Command: callExpr,
	}, nil
}

// FindHeaderArgs finds all header arguments (-H) in the curl command
func (c *CurlCommand) FindHeaderArgs() []int {
	var headerIndices []int
	for i, arg := range c.Command.Args {
		if i == 0 {
			continue // Skip the curl command itself
		}

		var buf bytes.Buffer
		printer := syntax.NewPrinter()
		printer.Print(&buf, arg)
		argStr := buf.String()

		// Check if it's a header flag
		if strings.TrimSpace(argStr) == "-H" || strings.TrimSpace(argStr) == "--header" {
			if i+1 < len(c.Command.Args) {
				headerIndices = append(headerIndices, i)
			}
		}
	}
	return headerIndices
}

// FindCookieArgs finds all cookie arguments (-b, --cookie, or -H "Cookie:") in the curl command
func (c *CurlCommand) FindCookieArgs() []int {
	var cookieIndices []int
	for i, arg := range c.Command.Args {
		if i == 0 {
			continue // Skip the curl command itself
		}

		var buf bytes.Buffer
		printer := syntax.NewPrinter()
		printer.Print(&buf, arg)
		argStr := buf.String()

		// Check if it's a cookie flag
		if strings.TrimSpace(argStr) == "-b" || strings.TrimSpace(argStr) == "--cookie" {
			if i+1 < len(c.Command.Args) {
				cookieIndices = append(cookieIndices, i)
			}
		} else if strings.TrimSpace(argStr) == "-H" || strings.TrimSpace(argStr) == "--header" {
			if i+1 < len(c.Command.Args) {
				var headerBuf bytes.Buffer
				printer.Print(&headerBuf, c.Command.Args[i+1])
				headerStr := headerBuf.String()
				headerStr = strings.Trim(headerStr, "'\"")
				if strings.HasPrefix(strings.ToLower(headerStr), "cookie:") {
					cookieIndices = append(cookieIndices, i)
				}
			}
		}
	}
	return cookieIndices
}

// FindURLArg finds the URL argument in the curl command
func (c *CurlCommand) FindURLArg() (int, error) {
	// First, look for arguments that don't start with a dash and aren't preceded by a flag
	for i, arg := range c.Command.Args {
		if i == 0 || i == len(c.Command.Args)-1 {
			continue // Skip the curl command itself and the last argument (which can't be followed by a value)
		}

		var buf bytes.Buffer
		printer := syntax.NewPrinter()
		printer.Print(&buf, arg)
		argStr := buf.String()
		argStr = strings.Trim(argStr, "'\"")

		// Check if it's a flag that expects a value
		if strings.HasPrefix(argStr, "-") {
			// Skip this argument and its value
			i++
			continue
		}

		// Check if the previous argument is a flag
		var prevBuf bytes.Buffer
		printer.Print(&prevBuf, c.Command.Args[i-1])
		prevStr := prevBuf.String()
		prevStr = strings.Trim(prevStr, "'\"")

		if strings.HasPrefix(prevStr, "-") {
			// This is a value for a flag, not a URL
			continue
		}

		// Try to parse it as a URL
		_, err := url.Parse(argStr)
		if err == nil {
			return i, nil
		}
	}

	// If we didn't find a URL yet, look for the last argument
	lastIndex := len(c.Command.Args) - 1
	if lastIndex > 0 {
		var buf bytes.Buffer
		printer := syntax.NewPrinter()
		printer.Print(&buf, c.Command.Args[lastIndex])
		argStr := buf.String()
		argStr = strings.Trim(argStr, "'\"")

		// Check if it's not a flag
		if !strings.HasPrefix(argStr, "-") {
			// Try to parse it as a URL
			_, err := url.Parse(argStr)
			if err == nil {
				return lastIndex, nil
			}
		}
	}

	return -1, fmt.Errorf("could not find URL in curl command")
}

// FindQueryParams finds query parameters in the URL
func (c *CurlCommand) FindQueryParams() (map[string]string, error) {
	urlIndex, err := c.FindURLArg()
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	printer := syntax.NewPrinter()
	printer.Print(&buf, c.Command.Args[urlIndex])
	urlStr := buf.String()
	urlStr = strings.Trim(urlStr, "'\"")

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	if parsedURL.RawQuery == "" {
		return nil, nil
	}

	queryParams := make(map[string]string)
	query, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		return nil, err
	}

	for k, v := range query {
		if len(v) > 0 {
			queryParams[k] = v[0]
		}
	}

	return queryParams, nil
}

// RemoveArg removes an argument and its value from the curl command
func (c *CurlCommand) RemoveArg(index int) {
	if index < 1 || index >= len(c.Command.Args) {
		return
	}

	// Check if this is a flag with a value
	if index+1 < len(c.Command.Args) {
		var buf bytes.Buffer
		printer := syntax.NewPrinter()
		printer.Print(&buf, c.Command.Args[index])
		argStr := buf.String()

		var nextBuf bytes.Buffer
		printer.Print(&nextBuf, c.Command.Args[index+1])
		nextStr := nextBuf.String()

		// If this is a flag and the next arg doesn't start with a dash, remove both
		if strings.HasPrefix(argStr, "-") && !strings.HasPrefix(nextStr, "-") {
			c.Command.Args = append(c.Command.Args[:index], c.Command.Args[index+2:]...)
			return
		}
	}

	// Otherwise just remove this arg
	c.Command.Args = append(c.Command.Args[:index], c.Command.Args[index+1:]...)
}

// RemoveQueryParam removes a query parameter from the URL
func (c *CurlCommand) RemoveQueryParam(param string) error {
	urlIndex, err := c.FindURLArg()
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	printer := syntax.NewPrinter()
	printer.Print(&buf, c.Command.Args[urlIndex])
	urlStr := buf.String()
	urlStr = strings.Trim(urlStr, "'\"")

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	if parsedURL.RawQuery == "" {
		return nil
	}

	query, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		return err
	}

	query.Del(param)
	parsedURL.RawQuery = query.Encode()

	// Create a new word node with the updated URL
	word := &syntax.Word{
		Parts: []syntax.WordPart{
			&syntax.Lit{
				Value: "'" + parsedURL.String() + "'",
			},
		},
	}

	c.Command.Args[urlIndex] = word
	return nil
}

// RemoveCookieFromHeader removes a specific cookie from a Cookie header
func (c *CurlCommand) RemoveCookieFromHeader(headerIndex int, cookieName string) error {
	if headerIndex < 1 || headerIndex >= len(c.Command.Args)-1 {
		return fmt.Errorf("invalid header index")
	}

	var headerBuf bytes.Buffer
	printer := syntax.NewPrinter()
	printer.Print(&headerBuf, c.Command.Args[headerIndex+1])
	headerStr := headerBuf.String()
	headerStr = strings.Trim(headerStr, "'\"")

	if !strings.HasPrefix(strings.ToLower(headerStr), "cookie:") {
		return fmt.Errorf("not a cookie header")
	}

	cookieStr := strings.TrimPrefix(headerStr, "Cookie:")
	cookieStr = strings.TrimPrefix(cookieStr, "cookie:")
	cookies := strings.Split(cookieStr, ";")

	var newCookies []string
	for _, cookie := range cookies {
		cookie = strings.TrimSpace(cookie)
		if cookie == "" {
			continue
		}

		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) == 2 {
			cookieNamePart := strings.TrimSpace(parts[0])
			if cookieNamePart != cookieName {
				newCookies = append(newCookies, cookie)
			}
		}
	}

	if len(newCookies) == 0 {
		// If no cookies left, remove the entire header
		c.RemoveArg(headerIndex)
		return nil
	}

	// Create a new word node with the updated cookie header
	word := &syntax.Word{
		Parts: []syntax.WordPart{
			&syntax.Lit{
				Value: "'Cookie: " + strings.Join(newCookies, "; ") + "'",
			},
		},
	}

	c.Command.Args[headerIndex+1] = word
	return nil
}

// ToString converts the curl command back to a string
func (c *CurlCommand) ToString() (string, error) {
	var buf bytes.Buffer
	printer := syntax.NewPrinter()
	err := printer.Print(&buf, c.Program)
	if err != nil {
		return "", fmt.Errorf("failed to print command: %w", err)
	}
	return buf.String(), nil
}
