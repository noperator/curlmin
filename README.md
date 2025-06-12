# curlmin - Curl Request Minimizer

A CLI tool that minimizes curl commands by removing unnecessary headers, cookies, and query parameters while ensuring the response remains the same.

How it works:

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

Minimize everything by default (headers, cookies, and query parameters), or choose which items you want to minimize. You can also match on status code, or other body features (bytes, lines, words) besides content.

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
```

Use the provided test server to see how it works. Consider using the `-v` flag with `curlmin` so you can watch it progressively strip down the curl command.

```
# start test server requiring a few auth-related items
go run testserver/server.go
Starting test server on http://localhost:8080
Required authentication:
  Cookie: session=abc123
  Auth Token: Bearer xyz789
  Query Parameter: auth_key=def456

# generate the following test curl command and pass to curlmin:
# curl \
#     -H 'Authorization: Bearer xyz789' \
#     -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' \
#     -H 'Accept: text/html,application/xhtml+xml,application/xml' \
#     -H 'Accept-Language: en-US,en;q=0.9' \
#     -H 'Cache-Control: max-age=0' \
#     -H 'Connection: keep-alive' \
#     -H 'Upgrade-Insecure-Requests: 1' \
#     -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' \
#     -H 'Cookie: _fbp=fb.1.1623456789.1234567890' \
#     -H 'Cookie: _gat=1; thisis=notneeded' \
#     -b 'preference=dark; language=en; theme=blue' \
#     'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin'
curlmin "$(go run testserver/cmd/generate_test_curl.go | grep -v '#')"

# prints this resulting minimized command
curl -H 'Authorization: Bearer xyz789' -H 'Cookie: session=abc123' 'http://localhost:8080/api/test?auth_key=def456'
```

### Troubleshooting

Since this tool actually executes the curl command to check the server response, that remote server actually needs to be _running_. So if you see this error, make sure you can actually reach the server you're validating the command against.

```
# didn't start test server (see example above)
curlmin "$(go run testserver/cmd/generate_test_curl.go | grep -v '#')"
Error minimizing curl command: failed to get baseline response: failed to execute curl command: exit status 7, stderr:
exit status 1
```

## Back matter

### See also

- https://github.com/portswigger/request-minimizer

### To-do

- [ ] optional delay between requests
- [ ] detect session expiration
- [ ] consolidate testing logic
- [ ] recognize `-` for reading from stdin
- [ ] document library usage

### License

This project is licensed under the [MIT License](LICENSE.md).
