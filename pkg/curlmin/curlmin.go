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
}

func DefaultOptions() Options {
	return Options{
		MinimizeHeaders: true,
		MinimizeCookies: true,
		MinimizeParams:  true,
		Verbose:         false,
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

func (m *Minimizer) executeCurlCommand(curlCmd string) (string, error) {
	// Create a temporary file to store the response
	tmpFile, err := os.CreateTemp("", "curlmin-response-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Make sure the command starts with curl
	curlCmd = strings.TrimSpace(curlCmd)
	if !strings.HasPrefix(curlCmd, "curl ") {
		curlCmd = "curl " + curlCmd
	}

	// Add the -o flag to save the response to the temporary file
	curlCmd = fmt.Sprintf("%s -o %s -s", curlCmd, tmpFile.Name())

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
		return "", fmt.Errorf("failed to execute curl command: %w, stderr: %s", err, stderr.String())
	}

	// Read the response from the temporary file
	respBytes, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read response from temporary file: %w", err)
	}

	// Return the response as a string
	return string(respBytes), nil
}

func (m *Minimizer) compareResponses(resp1, resp2 string) bool {
	// Calculate MD5 hashes of the responses
	hash1 := md5.Sum([]byte(resp1))
	hash2 := md5.Sum([]byte(resp2))

	// Compare the hashes
	return hex.EncodeToString(hash1[:]) == hex.EncodeToString(hash2[:])
}

func (m *Minimizer) minimizeQueryParams(curl *CurlCommand, baselineResp string) {
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

func (m *Minimizer) minimizeHeaders(curl *CurlCommand, baselineResp string) {
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

func (m *Minimizer) minimizeCookies(curl *CurlCommand, baselineResp string) {
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

				// If it's a Cookie header, extract and test each cookie
				if strings.HasPrefix(strings.ToLower(headerStr), "cookie:") {
					cookieStr := strings.TrimPrefix(headerStr, "Cookie:")
					cookieStr = strings.TrimPrefix(cookieStr, "cookie:")
					cookies := strings.Split(cookieStr, ";")

					for _, cookie := range cookies {
						cookie = strings.TrimSpace(cookie)
						if cookie == "" {
							continue
						}

						parts := strings.SplitN(cookie, "=", 2)
						if len(parts) == 2 {
							cookieName := strings.TrimSpace(parts[0])

							// Create a copy of the curl command
							originalCmd, err := curl.ToString()
							if err != nil {
								continue
							}

							curlCopy, err := ParseCurlCommand(originalCmd)
							if err != nil {
								continue
							}

							// Remove the cookie
							err = curlCopy.RemoveCookieFromHeader(cookieIndex, cookieName)
							if err != nil {
								continue
							}

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
									fmt.Printf("Cookie not needed: %s\n", cookieName)
								}
								curl.RemoveCookieFromHeader(cookieIndex, cookieName)
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
				} else {
					// For -b/--cookie flags, try removing the entire argument
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

					// Get the cookie flag for logging
					var cookieFlag string
					if cookieIndex < len(curl.Command.Args) {
						var flagBuf bytes.Buffer
						printer := syntax.NewPrinter()
						printer.Print(&flagBuf, curl.Command.Args[cookieIndex])
						cookieFlag = flagBuf.String()
					}

					if err == nil && m.compareResponses(baselineResp, testResp) {
						// If the response is the same, update the original curl command
						if m.options.Verbose {
							fmt.Printf("Cookie flag not needed: %s\n", cookieFlag)
						}
						curl.RemoveArg(cookieIndex)
						foundRemovable = true
						break
					} else if m.options.Verbose {
						fmt.Printf("Cookie flag needed: %s\n", cookieFlag)
					}
				}
			}
		}

		// If we didn't find any removable cookies in this iteration, we're done
		if !foundRemovable {
			return
		}
	}
}
