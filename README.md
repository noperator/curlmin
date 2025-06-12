# curlmin - Curl Request Minimizer

A CLI tool that minimizes curl commands by removing unnecessary headers, cookies, and query parameters while ensuring the response remains the same.

## How it works

1. Parses the curl command into a syntax tree 🌳
2. Makes a baseline request to get the expected response 📜
3. Iteratively removes headers, cookies, and query parameters one by one ✂️
4. After each removal, makes a new request and compares the response to the baseline  🧐
5. If the response is the same, removes the unnecessary element 🚮
6. Returns final minimized curl command 🎁

## Getting started

### Install

```
go install github.com/noperator/curlmin/cmd/curlmin@latest
```

### Usage

Minimize everything by default (headers, cookies, and query parameters), or choose which items you want to minimize.

```
Usage of curlmin:
  -body
    	Compare body content (default true)
  -bytes
    	Compare byte count
  -cookies
    	Minimize cookies (default true)
  -headers
    	Minimize headers (default true)
  -lines
    	Compare line count
  -params
    	Minimize query parameters (default true)
  -status
    	Compare status code
  -v	Verbose output
  -words
    	Compare word count

# start test server
go run testserver/server.go
Starting test server on http://localhost:8080
Required authentication:
  Cookie: session=abc123
  Auth Token: Bearer xyz789
  Query Parameter: auth_key=def456

# pass curl command to curlmin
curlmin "curl -H 'Authorization: Bearer xyz789' -H 'User-Agent: Mozilla/5.0' -H 'Accept: text/html' -H 'Cookie: session=abc123' -H 'Cookie: _ga=GA1.2.1234567890.1623456789' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&utm_source=test'"

# prints this minimized command
curl -H 'Authorization: Bearer xyz789' -H 'Cookie: session=abc123' 'http://localhost:8080/api/test?auth_key=def456'
```

### Troubleshooting

Since this tool actually executes the curl command to check the server response, that remote server actually needs to be _running_. So if you see this error, make sure you can actually reach the server you're validating the command against.

```
curlmin "$(go run testserver/cmd/generate_test_curl.go | grep -v '#')"
Error minimizing curl command: failed to get baseline response: failed to execute curl command: exit status 7, stderr:
exit status 1
```


### Use as a library

```go
package main

import (
	"fmt"
	"github.com/noperator/curlmin/pkg/curlmin"
)

func main() {
	// Create a minimizer with default options
	minimizer := curlmin.New(curlmin.DefaultOptions())

	// Minimize a curl command
	curlCmd := `curl -H 'Authorization: Bearer xyz789' -H 'User-Agent: Mozilla/5.0' -H 'Cookie: session=abc123' 'http://example.com/api?param1=value1&param2=value2'`
	minimizedCmd, err := minimizer.MinimizeCurlCommand(curlCmd)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Minimized command: %s\n", minimizedCmd)
}
```

### Testing

Use the provided test suite to validate the minimization process.

```bash
cd pkg/curlmin
go test -v
```

## Back matter

### See also

- https://github.com/portswigger/request-minimizer

### To-do

- [ ] optional delay between requests
- [ ] detect session expiration

### License

This project is licensed under the [MIT License](LICENSE.md).
