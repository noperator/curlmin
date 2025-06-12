package main

import (
	"fmt"
)

func main() {
	// Base URL with required query parameter
	baseURL := "http://localhost:8080/api/test?auth_key=def456"

	// Add unnecessary query parameters
	baseURL += "&timestamp=1623456789"
	baseURL += "&tracking_id=abcdef123456"
	baseURL += "&utm_source=test"
	baseURL += "&utm_medium=cli"
	baseURL += "&utm_campaign=curlmin"

	// Required headers
	requiredHeaders := []string{
		"-H 'Authorization: Bearer xyz789'",
	}

	// Unnecessary headers
	unnecessaryHeaders := []string{
		"-H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36'",
		"-H 'Accept: text/html,application/xhtml+xml,application/xml'",
		"-H 'Accept-Language: en-US,en;q=0.9'",
		"-H 'Cache-Control: max-age=0'",
		"-H 'Connection: keep-alive'",
		"-H 'Upgrade-Insecure-Requests: 1'",
	}

	// Combined cookies in a single header (with required session cookie in the middle)
	combinedCookieHeader := "-H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789'"

	// Additional cookies in separate headers
	additionalCookies := []string{
		"-H 'Cookie: _fbp=fb.1.1623456789.1234567890'",
		"-H 'Cookie: _gat=1; thisis=notneeded'",
	}

	// Cookie flag with multiple cookies
	cookieFlag := "-b 'preference=dark; language=en; theme=blue'"

	// Build the curl command
	curlCmd := "curl"

	// Add all headers
	for _, header := range append(requiredHeaders, unnecessaryHeaders...) {
		curlCmd += " " + header
	}

	// Add the combined cookie header
	curlCmd += " " + combinedCookieHeader

	// Add additional cookie headers
	for _, cookie := range additionalCookies {
		curlCmd += " " + cookie
	}

	// Add the cookie flag
	curlCmd += " " + cookieFlag

	// Add the URL
	curlCmd += " '" + baseURL + "'"

	// Print the curl command
	fmt.Println("# Test curl command with required and unnecessary elements")
	fmt.Println(curlCmd)

	// Print instructions
	fmt.Println("\n# To use this command with curlmin:")
	fmt.Println("# 1. Start the test server: go run testserver/server.go")
	fmt.Println("# 2. In another terminal, run one of the following:")
	fmt.Println("#    - Using --command/-c flag: go run cmd/curlmin/main.go --command \"$(go run testserver/cmd/generate_test_curl.go)\"")
	fmt.Println("#      or with the shorthand option: go run cmd/curlmin/main.go -c \"$(go run testserver/cmd/generate_test_curl.go)\"")
	fmt.Println("#    - Using a file: ")
	fmt.Println("#      a. Save the output to a file: go run testserver/cmd/generate_test_curl.go > test_curl.txt")
	fmt.Println("#      b. Run curlmin with the file: go run cmd/curlmin/main.go --file test_curl.txt")
	fmt.Println("#         or with the shorthand option: go run cmd/curlmin/main.go -f test_curl.txt")
}
