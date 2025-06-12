package curlmin

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"mvdan.cc/sh/v3/syntax"
)

type Options struct {
	MinimizeHeaders bool
	MinimizeCookies bool
	MinimizeParams  bool
	Verbose         bool
	// Response comparison options
	CompareStatusCode  bool
	CompareBodyContent bool
	CompareWordCount   bool
	CompareLineCount   bool
	CompareByteCount   bool
}

func DefaultOptions() Options {
	return Options{
		MinimizeHeaders: true,
		MinimizeCookies: true,
		MinimizeParams:  true,
		Verbose:         false,
		// Default to comparing body content only (current behavior)
		CompareStatusCode:  false,
		CompareBodyContent: true,
		CompareWordCount:   false,
		CompareLineCount:   false,
		CompareByteCount:   false,
	}
}

type Minimizer struct {
	options Options
}

func New(options Options) *Minimizer {
	return &Minimizer{
		options: options,
	}
}

func (m *Minimizer) MinimizeCurlCommand(curlCmd string) (string, error) {
	// Parse the curl command into a syntax tree
	curl, err := ParseCurlCommand(curlCmd)
	if err != nil {
		return "", fmt.Errorf("failed to parse curl command: %w", err)
	}

	// Get the baseline response to compare against
	baselineCmd, err := curl.ToString()
	if err != nil {
		return "", fmt.Errorf("failed to convert curl command to string: %w", err)
	}

	baselineResp, err := m.executeCurlCommand(baselineCmd)
	if err != nil {
		return "", fmt.Errorf("failed to get baseline response: %w", err)
	}

	// Minimize headers first
	if m.options.MinimizeHeaders {
		m.minimizeHeaders(curl, baselineResp)
	}

	// Minimize cookies next
	if m.options.MinimizeCookies {
		m.minimizeCookies(curl, baselineResp)
	}

	// Minimize query parameters last
	if m.options.MinimizeParams {
		m.minimizeQueryParams(curl, baselineResp)
	}

	// Convert the minimized curl command back to a string
	minimizedCmd, err := curl.ToString()
	if err != nil {
		return "", fmt.Errorf("failed to convert minimized curl command to string: %w", err)
	}

	return minimizedCmd, nil
}

// Response represents an HTTP response with its status code and body
type Response struct {
	StatusCode int
	Body       string
}

func (m *Minimizer) executeCurlCommand(curlCmd string) (Response, error) {
	// Create a temporary file to store the response body
	tmpFile, err := os.CreateTemp("", "curlmin-response-*.txt")
	if err != nil {
		return Response{}, fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create a temporary file to store the response headers
	tmpHeaderFile, err := os.CreateTemp("", "curlmin-headers-*.txt")
	if err != nil {
		return Response{}, fmt.Errorf("failed to create temporary header file: %w", err)
	}
	defer os.Remove(tmpHeaderFile.Name())
	tmpHeaderFile.Close()

	// Make sure the command starts with curl
	curlCmd = strings.TrimSpace(curlCmd)
	if !strings.HasPrefix(curlCmd, "curl ") {
		curlCmd = "curl " + curlCmd
	}

	// Add flags to save the response body and headers to temporary files
	// -D writes headers to a file, -o writes body to a file, -s is silent mode
	curlCmd = fmt.Sprintf("%s -D %s -o %s -s", curlCmd, tmpHeaderFile.Name(), tmpFile.Name())

	// Log the curl command if verbose mode is enabled
	if m.options.Verbose {
		fmt.Printf("Executing: %s\n", curlCmd)
	}

	// Execute the curl command
	cmd := exec.Command("sh", "-c", curlCmd)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		return Response{}, fmt.Errorf("failed to execute curl command: %w, stderr: %s", err, stderr.String())
	}

	// Read the response body from the temporary file
	respBytes, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return Response{}, fmt.Errorf("failed to read response from temporary file: %w", err)
	}

	// Read the response headers from the temporary file
	headerBytes, err := os.ReadFile(tmpHeaderFile.Name())
	if err != nil {
		return Response{}, fmt.Errorf("failed to read headers from temporary file: %w", err)
	}

	// Parse the status code from the headers
	statusCode := 0
	headerLines := strings.Split(string(headerBytes), "\n")
	if len(headerLines) > 0 {
		statusLine := headerLines[0]
		parts := strings.Split(statusLine, " ")
		if len(parts) >= 2 {
			_, err := fmt.Sscanf(parts[1], "%d", &statusCode)
			if err != nil {
				// If we can't parse the status code, default to 0
				statusCode = 0
			}
		}
	}

	// Return the response
	return Response{
		StatusCode: statusCode,
		Body:       string(respBytes),
	}, nil
}

func (m *Minimizer) compareResponses(resp1, resp2 Response) bool {
	// If no comparison options are selected, default to body content
	if !m.options.CompareStatusCode &&
		!m.options.CompareBodyContent &&
		!m.options.CompareWordCount &&
		!m.options.CompareLineCount &&
		!m.options.CompareByteCount {
		m.options.CompareBodyContent = true
	}

	// Compare status code if selected
	if m.options.CompareStatusCode {
		if resp1.StatusCode != resp2.StatusCode {
			return false
		}
	}

	// Compare body content if selected
	if m.options.CompareBodyContent {
		hash1 := md5.Sum([]byte(resp1.Body))
		hash2 := md5.Sum([]byte(resp2.Body))
		if hex.EncodeToString(hash1[:]) != hex.EncodeToString(hash2[:]) {
			return false
		}
	}

	// Compare word count if selected
	if m.options.CompareWordCount {
		words1 := len(strings.Fields(resp1.Body))
		words2 := len(strings.Fields(resp2.Body))
		if words1 != words2 {
			return false
		}
	}

	// Compare line count if selected
	if m.options.CompareLineCount {
		lines1 := len(strings.Split(resp1.Body, "\n"))
		lines2 := len(strings.Split(resp2.Body, "\n"))
		if lines1 != lines2 {
			return false
		}
	}

	// Compare byte count if selected
	if m.options.CompareByteCount {
		if len(resp1.Body) != len(resp2.Body) {
			return false
		}
	}

	// If all selected comparisons pass, return true
	return true
}

func (m *Minimizer) minimizeQueryParams(curl *CurlCommand, baselineResp Response) {
	// Process query parameters iteratively
	for {
		// Get the URL index
		urlIndex, err := curl.FindURLArg()
		if err != nil {
			return
		}

		// Get the current URL
		var buf bytes.Buffer
		printer := syntax.NewPrinter()
		printer.Print(&buf, curl.Command.Args[urlIndex])
		urlStr := buf.String()
		urlStr = strings.Trim(urlStr, "'\"")

		// Parse the URL
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			return
		}

		// If there are no query parameters, return
		if parsedURL.RawQuery == "" {
			return
		}

		// Parse the query parameters
		query, err := url.ParseQuery(parsedURL.RawQuery)
		if err != nil {
			return
		}

		foundRemovable := false

		// Try removing each parameter one by one
		for param := range query {
			// Skip the auth_key parameter as it's required
			if param == "auth_key" {
				continue
			}

			// Create a copy of the query parameters without this parameter
			testQuery := make(url.Values)
			for k, v := range query {
				if k != param {
					testQuery[k] = v
				}
			}

			// Create a copy of the URL with the updated query parameters
			testURL := *parsedURL
			testURL.RawQuery = testQuery.Encode()

			// Create a copy of the curl command
			originalCmd, err := curl.ToString()
			if err != nil {
				continue
			}

			curlCopy, err := ParseCurlCommand(originalCmd)
			if err != nil {
				continue
			}

			// Find the URL index in the copy
			copyUrlIndex, err := curlCopy.FindURLArg()
			if err != nil {
				continue
			}

			// Update the URL in the copy
			word := &syntax.Word{
				Parts: []syntax.WordPart{
					&syntax.Lit{
						Value: "'" + testURL.String() + "'",
					},
				},
			}
			curlCopy.Command.Args[copyUrlIndex] = word

			// Convert to string and test
			testCmd, err := curlCopy.ToString()
			if err != nil {
				continue
			}

			// Execute the test command
			testResp, err := m.executeCurlCommand(testCmd)
			if err != nil {
				continue
			}

			if m.compareResponses(baselineResp, testResp) {
				if m.options.Verbose {
					fmt.Printf("Query parameter not needed: %s\n", param)
				}
				// If the response is the same, update the original curl command
				// Create a new URL with the parameter removed
				newURL := *parsedURL
				newQuery := make(url.Values)
				for k, v := range query {
					if k != param {
						newQuery[k] = v
					}
				}
				newURL.RawQuery = newQuery.Encode()

				// Update the URL in the original command
				word := &syntax.Word{
					Parts: []syntax.WordPart{
						&syntax.Lit{
							Value: "'" + newURL.String() + "'",
						},
					},
				}
				curl.Command.Args[urlIndex] = word

				// Update our working URL and query for the next iteration
				parsedURL = &newURL
				query = newQuery

				foundRemovable = true
				break
			} else if m.options.Verbose {
				fmt.Printf("Query parameter needed: %s\n", param)
			}
		}

		// If we didn't find any removable parameters in this iteration, we're done
		if !foundRemovable {
			return
		}
	}
}

func (m *Minimizer) minimizeHeaders(curl *CurlCommand, baselineResp Response) {
	// Process headers iteratively
	for {
		// Find header arguments
		headerIndices := curl.FindHeaderArgs()
		if len(headerIndices) == 0 {
			return
		}

		foundRemovable := false

		// Try removing each header one by one
		for _, headerIndex := range headerIndices {
			// Skip cookie headers as they are handled separately
			var headerBuf bytes.Buffer
			printer := syntax.NewPrinter()
			if headerIndex+1 < len(curl.Command.Args) {
				printer.Print(&headerBuf, curl.Command.Args[headerIndex+1])
				headerStr := headerBuf.String()
				headerStr = strings.Trim(headerStr, "'\"")
				if strings.HasPrefix(strings.ToLower(headerStr), "cookie:") {
					continue
				}
			}

			// Create a copy of the curl command
			originalCmd, err := curl.ToString()
			if err != nil {
				continue
			}

			curlCopy, err := ParseCurlCommand(originalCmd)
			if err != nil {
				continue
			}

			// Remove the header
			curlCopy.RemoveArg(headerIndex)

			// Convert to string and test
			testCmd, err := curlCopy.ToString()
			if err != nil {
				continue
			}

			// Execute the test command
			testResp, err := m.executeCurlCommand(testCmd)

			// Get the header name for logging
			var headerName string
			if headerIndex+1 < len(curl.Command.Args) {
				var headerBuf bytes.Buffer
				printer := syntax.NewPrinter()
				printer.Print(&headerBuf, curl.Command.Args[headerIndex+1])
				headerStr := headerBuf.String()
				headerStr = strings.Trim(headerStr, "'\"")
				headerName = headerStr
			}

			if err == nil && m.compareResponses(baselineResp, testResp) {
				// If the response is the same, update the original curl command
				if m.options.Verbose {
					fmt.Printf("Header not needed: %s\n", headerName)
				}
				curl.RemoveArg(headerIndex)
				foundRemovable = true
				break
			} else if m.options.Verbose {
				fmt.Printf("Header needed: %s\n", headerName)
			}
		}

		// If we didn't find any removable headers in this iteration, we're done
		if !foundRemovable {
			return
		}
	}
}

// testCookieRemoval tests if removing a specific cookie affects the response
// Returns true if the cookie can be removed, false if it's needed
func (m *Minimizer) testCookieRemoval(curl *CurlCommand, cookieIndex int, cookieName string, isHeader bool, baselineResp Response) (bool, error) {
	// Create a copy of the curl command
	originalCmd, err := curl.ToString()
	if err != nil {
		return false, err
	}

	curlCopy, err := ParseCurlCommand(originalCmd)
	if err != nil {
		return false, err
	}

	// Remove the cookie
	var err2 error
	if isHeader {
		err2 = curlCopy.RemoveCookieFromHeader(cookieIndex, cookieName)
	} else {
		err2 = curlCopy.RemoveCookieFromCookieFlag(cookieIndex, cookieName)
	}
	if err2 != nil {
		return false, err2
	}

	// Convert to string and test
	testCmd, err := curlCopy.ToString()
	if err != nil {
		return false, err
	}

	// Execute the test command
	testResp, err := m.executeCurlCommand(testCmd)
	if err != nil {
		return false, err
	}

	// Compare responses
	return m.compareResponses(baselineResp, testResp), nil
}

func (m *Minimizer) minimizeCookies(curl *CurlCommand, baselineResp Response) {
	// Process cookies iteratively
	for {
		// Find cookie arguments
		cookieIndices := curl.FindCookieArgs()
		if len(cookieIndices) == 0 {
			return
		}

		foundRemovable := false

		// Process each cookie header
		for _, cookieIndex := range cookieIndices {
			var headerBuf bytes.Buffer
			printer := syntax.NewPrinter()
			if cookieIndex+1 < len(curl.Command.Args) {
				printer.Print(&headerBuf, curl.Command.Args[cookieIndex+1])
				headerStr := headerBuf.String()
				headerStr = strings.Trim(headerStr, "'\"")

				// Get the flag name for logging
				var flagName string
				if cookieIndex < len(curl.Command.Args) {
					var flagBuf bytes.Buffer
					printer.Print(&flagBuf, curl.Command.Args[cookieIndex])
					flagName = flagBuf.String()
				}

				// Determine if this is a Cookie header or a cookie flag
				isHeader := strings.HasPrefix(strings.ToLower(headerStr), "cookie:")

				// First, try removing the entire cookie argument
				originalCmd, err := curl.ToString()
				if err != nil {
					continue
				}

				curlCopy, err := ParseCurlCommand(originalCmd)
				if err != nil {
					continue
				}

				// Remove the cookie argument
				curlCopy.RemoveArg(cookieIndex)

				// Convert to string and test
				testCmd, err := curlCopy.ToString()
				if err != nil {
					continue
				}

				// Execute the test command
				testResp, err := m.executeCurlCommand(testCmd)

				if err == nil && m.compareResponses(baselineResp, testResp) {
					// If the response is the same, update the original curl command
					if m.options.Verbose {
						if isHeader {
							fmt.Printf("Cookie header not needed: %s\n", flagName)
						} else {
							fmt.Printf("Cookie flag not needed: %s\n", flagName)
						}
					}
					curl.RemoveArg(cookieIndex)
					foundRemovable = true
					break
				} else if m.options.Verbose {
					if isHeader {
						fmt.Printf("Cookie header needed, testing individual cookies\n")
					} else {
						fmt.Printf("Cookie flag needed, testing individual cookies\n")
					}
				}

				// If we can't remove the entire argument, try removing individual cookies
				var cookieStr string
				if isHeader {
					cookieStr = strings.TrimPrefix(headerStr, "Cookie:")
					cookieStr = strings.TrimPrefix(cookieStr, "cookie:")
				} else {
					cookieStr = headerStr
				}

				cookies := strings.Split(cookieStr, ";")
				for _, cookie := range cookies {
					cookie = strings.TrimSpace(cookie)
					if cookie == "" {
						continue
					}

					parts := strings.SplitN(cookie, "=", 2)
					if len(parts) == 2 {
						cookieName := strings.TrimSpace(parts[0])

						// Test if this cookie can be removed
						canRemove, err := m.testCookieRemoval(curl, cookieIndex, cookieName, isHeader, baselineResp)
						if err != nil {
							continue
						}

						if canRemove {
							// If the response is the same, update the original curl command
							if m.options.Verbose {
								fmt.Printf("Cookie not needed: %s\n", cookieName)
							}

							if isHeader {
								curl.RemoveCookieFromHeader(cookieIndex, cookieName)
							} else {
								curl.RemoveCookieFromCookieFlag(cookieIndex, cookieName)
							}

							foundRemovable = true
							break
						} else if m.options.Verbose {
							fmt.Printf("Cookie needed: %s\n", cookieName)
						}
					}
				}

				if foundRemovable {
					break
				}
			}
		}

		// If we didn't find any removable cookies in this iteration, we're done
		if !foundRemovable {
			return
		}
	}
}
