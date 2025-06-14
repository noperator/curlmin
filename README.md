# curlmin - Curl Request Minimizer

A CLI tool that minimizes curl commands by removing unnecessary headers, cookies, and query parameters while ensuring the response remains the same. This is especially handy when copying a network request "as cURL" in Chrome DevTools' Network panel (Right-click page > Inspect > Network > Right-click request > Copy > Copy as cURL) as shown in this example:

https://github.com/user-attachments/assets/c1acc4de-7836-494d-800e-1921ac93c8db

## Description

I use Chrome's "Copy as cURL" _a lot_ (so much, in fact, that I wrote [sol](https://github.com/noperator/sol) partially just to help me auto-format long curl commands). I often have this problem where the copied curl command contains a bunch of garbage (namely, extra headers and cookies for tracking purposes) that isn't at all relevant to the actual request being made. After years of manually trimming out cookies in order to see which ones are actually necessary to maintain a stateful authenticated session, I finally decided to make a tool to automate the minification of a curl command.

### How it works

1. Parses the curl command into a syntax tree 🌳
2. Makes a baseline request to get the expected response 📜
3. Iteratively removes headers, cookies, and query parameters one by one ✂️
4. After each removal, makes a new request and compares the response to the baseline  🧐
5. If the response is the same, removes the unnecessary element 🚮
6. Returns final minimized curl command 🎁

### Features

- Choose which request elements you want to **minimize**: headers, cookies, or query parameters. Minimizes all by default.
- Choose which features of the response you want to **compare** against the baseline request: status code, body content, or body line/word/byte count. Compares body content by default.

## Getting started

### Install

```
go install github.com/noperator/curlmin/cmd/curlmin@latest
```

### Usage

```
Usage of curlmin:
      --body             Compare body content (default true)
      --bytes            Compare byte count
  -c, --command string   Curl command as a string
      --cookies          Minimize cookies (default true)
  -f, --file string      File containing the curl command
      --headers          Minimize headers (default true)
      --lines            Compare line count
      --params           Minimize query parameters (default true)
      --status           Compare status code
  -v, --verbose          Verbose output
      --words            Compare word count
```

You can provide the curl command in one of three ways:
1. Use `--command` to specify the curl command as a string
2. Use `--file` to read the curl command from a file (`--file -` will read from stdin)
3. Pipe the curl command directly to curlmin (e.g., `cat curl.sh | curlmin`)

In this example, we start with a big ol' curl command with a bunch of unnecessary headers, cookies, and query parameters, and then use curlmin to strip it down to the minimal necessary request elements that result in the same response:

```
$ go run testserver/cmd/generate_test_curl.go | grep '^[^#]' | tee curl.sh 
curl \
    -H 'Authorization: Bearer xyz789' \
    -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' \
    -H 'Accept: text/html,application/xhtml+xml,application/xml' \
    -H 'Accept-Language: en-US,en;q=0.9' \
    -H 'Cache-Control: max-age=0' \
    -H 'Connection: keep-alive' \
    -H 'Upgrade-Insecure-Requests: 1' \
    -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' \
    -H 'Cookie: _fbp=fb.1.1623456789.1234567890' \
    -H 'Cookie: _gat=1; thisis=notneeded' \
    -b 'preference=dark; language=en; theme=blue' \
    'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin'

$ curlmin -f curl.sh
curl \
    -H 'Authorization: Bearer xyz789' \
    -H 'Cookie: session=abc123' \
    'http://localhost:8080/api/test?auth_key=def456'
```

Note that in this example we're using the provided test server which requires a specific header, cookie, and query parameter to be set. You can start the server like this:

```
$ go run testserver/server.go
Starting test server on http://localhost:8080
Required authentication:
  Cookie: session=abc123
  Auth Token: Bearer xyz789
  Query Parameter: auth_key=def456
```

If you use curlmin's `--verbose` option, you can follow how it iteratively removes an element from a curl command, executes the command, and examines the response to determine whether to keep that element or not.

<details><summary>Verbose output</summary>
<p>

```
$ curlmin -v -f curl.sh
Original curl command:
curl -H 'Authorization: Bearer xyz789' -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' -H 'Accept: text/html,application/xhtml+xml,application/xml' -H 'Accept-Language: en-US,en;q=0.9' -H 'Cache-Control: max-age=0' -H 'Connection: keep-alive' -H 'Upgrade-Insecure-Requests: 1' -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin'

Executing: curl -H 'Authorization: Bearer xyz789' -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' -H 'Accept: text/html,application/xhtml+xml,application/xml' -H 'Accept-Language: en-US,en;q=0.9' -H 'Cache-Control: max-age=0' -H 'Connection: keep-alive' -H 'Upgrade-Insecure-Requests: 1' -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-3643437334.txt -o /tmp/curlmin-response-963623028.txt -s
Executing: curl -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36' -H 'Accept: text/html,application/xhtml+xml,application/xml' -H 'Accept-Language: en-US,en;q=0.9' -H 'Cache-Control: max-age=0' -H 'Connection: keep-alive' -H 'Upgrade-Insecure-Requests: 1' -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-337543815.txt -o /tmp/curlmin-response-984025244.txt -s
Header needed: Authorization: Bearer xyz789
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Accept: text/html,application/xhtml+xml,application/xml' -H 'Accept-Language: en-US,en;q=0.9' -H 'Cache-Control: max-age=0' -H 'Connection: keep-alive' -H 'Upgrade-Insecure-Requests: 1' -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-4216696553.txt -o /tmp/curlmin-response-2384003786.txt -s
Header not needed: User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36
Executing: curl -H 'Accept: text/html,application/xhtml+xml,application/xml' -H 'Accept-Language: en-US,en;q=0.9' -H 'Cache-Control: max-age=0' -H 'Connection: keep-alive' -H 'Upgrade-Insecure-Requests: 1' -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-3133278322.txt -o /tmp/curlmin-response-3049459802.txt -s
Header needed: Authorization: Bearer xyz789
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Accept-Language: en-US,en;q=0.9' -H 'Cache-Control: max-age=0' -H 'Connection: keep-alive' -H 'Upgrade-Insecure-Requests: 1' -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-2046388633.txt -o /tmp/curlmin-response-647469596.txt -s
Header not needed: Accept: text/html,application/xhtml+xml,application/xml
Executing: curl -H 'Accept-Language: en-US,en;q=0.9' -H 'Cache-Control: max-age=0' -H 'Connection: keep-alive' -H 'Upgrade-Insecure-Requests: 1' -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-1254716396.txt -o /tmp/curlmin-response-2981810659.txt -s
Header needed: Authorization: Bearer xyz789
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cache-Control: max-age=0' -H 'Connection: keep-alive' -H 'Upgrade-Insecure-Requests: 1' -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-2938036561.txt -o /tmp/curlmin-response-1208700683.txt -s
Header not needed: Accept-Language: en-US,en;q=0.9
Executing: curl -H 'Cache-Control: max-age=0' -H 'Connection: keep-alive' -H 'Upgrade-Insecure-Requests: 1' -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-2936420885.txt -o /tmp/curlmin-response-3761155716.txt -s
Header needed: Authorization: Bearer xyz789
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Connection: keep-alive' -H 'Upgrade-Insecure-Requests: 1' -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-2126919866.txt -o /tmp/curlmin-response-1661365263.txt -s
Header not needed: Cache-Control: max-age=0
Executing: curl -H 'Connection: keep-alive' -H 'Upgrade-Insecure-Requests: 1' -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-184484295.txt -o /tmp/curlmin-response-2601466044.txt -s
Header needed: Authorization: Bearer xyz789
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Upgrade-Insecure-Requests: 1' -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-2231888437.txt -o /tmp/curlmin-response-3663833958.txt -s
Header not needed: Connection: keep-alive
Executing: curl -H 'Upgrade-Insecure-Requests: 1' -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-2382836929.txt -o /tmp/curlmin-response-2631633639.txt -s
Header needed: Authorization: Bearer xyz789
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-3755165625.txt -o /tmp/curlmin-response-4149399765.txt -s
Header not needed: Upgrade-Insecure-Requests: 1
Executing: curl -H 'Cookie: _ga=GA1.2.1234567890.1623456789; session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-1892301372.txt -o /tmp/curlmin-response-135692561.txt -s
Header needed: Authorization: Bearer xyz789
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-447112710.txt -o /tmp/curlmin-response-425483109.txt -s
Cookie header needed, testing individual cookies
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: session=abc123; _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-3581151498.txt -o /tmp/curlmin-response-1104187038.txt -s
Cookie not needed: _ga
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-355344628.txt -o /tmp/curlmin-response-707357954.txt -s
Cookie header needed, testing individual cookies
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: _gid=GA1.2.9876543210.1623456789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-3695396543.txt -o /tmp/curlmin-response-1925809169.txt -s
Cookie needed: session
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: session=abc123' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-630525039.txt -o /tmp/curlmin-response-1322865396.txt -s
Cookie not needed: _gid
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-3308648748.txt -o /tmp/curlmin-response-959214987.txt -s
Cookie header needed, testing individual cookies
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: _fbp=fb.1.1623456789.1234567890' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-2235045407.txt -o /tmp/curlmin-response-968194517.txt -s
Cookie needed: session
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: session=abc123' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-686363476.txt -o /tmp/curlmin-response-2586551186.txt -s
Cookie header not needed: -H
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-4049849043.txt -o /tmp/curlmin-response-2842975052.txt -s
Cookie header needed, testing individual cookies
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: _gat=1; thisis=notneeded' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-4192806095.txt -o /tmp/curlmin-response-2231182901.txt -s
Cookie needed: session
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: session=abc123' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-850279218.txt -o /tmp/curlmin-response-2331635687.txt -s
Cookie header not needed: -H
Executing: curl -H 'Authorization: Bearer xyz789' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-39224845.txt -o /tmp/curlmin-response-560093416.txt -s
Cookie header needed, testing individual cookies
Executing: curl -H 'Authorization: Bearer xyz789' -b 'preference=dark; language=en; theme=blue' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-3542029077.txt -o /tmp/curlmin-response-2471057670.txt -s
Cookie needed: session
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: session=abc123' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-2066167408.txt -o /tmp/curlmin-response-1534794634.txt -s
Cookie flag not needed: -b
Executing: curl -H 'Authorization: Bearer xyz789' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-2657689963.txt -o /tmp/curlmin-response-2137903034.txt -s
Cookie header needed, testing individual cookies
Executing: curl -H 'Authorization: Bearer xyz789' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_source=test&utm_medium=cli&utm_campaign=curlmin' -D /tmp/curlmin-headers-1573734881.txt -o /tmp/curlmin-response-4058415586.txt -s
Cookie needed: session
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: session=abc123' 'http://localhost:8080/api/test?auth_key=def456&timestamp=1623456789&tracking_id=abcdef123456&utm_medium=cli&utm_source=test' -D /tmp/curlmin-headers-2805583773.txt -o /tmp/curlmin-response-2704891021.txt -s
Query parameter not needed: utm_campaign
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: session=abc123' 'http://localhost:8080/api/test?auth_key=def456&tracking_id=abcdef123456&utm_medium=cli&utm_source=test' -D /tmp/curlmin-headers-323062671.txt -o /tmp/curlmin-response-1395711224.txt -s
Query parameter not needed: timestamp
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: session=abc123' 'http://localhost:8080/api/test?auth_key=def456&utm_medium=cli&utm_source=test' -D /tmp/curlmin-headers-4071408259.txt -o /tmp/curlmin-response-3483987195.txt -s
Query parameter not needed: tracking_id
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: session=abc123' 'http://localhost:8080/api/test?auth_key=def456&utm_medium=cli' -D /tmp/curlmin-headers-3552221131.txt -o /tmp/curlmin-response-2371034657.txt -s
Query parameter not needed: utm_source
Executing: curl -H 'Authorization: Bearer xyz789' -H 'Cookie: session=abc123' 'http://localhost:8080/api/test?auth_key=def456' -D /tmp/curlmin-headers-347212372.txt -o /tmp/curlmin-response-3259833644.txt -s
Query parameter not needed: utm_medium
Minimized curl command:
curl -H 'Authorization: Bearer xyz789' -H 'Cookie: session=abc123' 'http://localhost:8080/api/test?auth_key=def456'
```

</p>
</details>

### Troubleshooting

Since this tool actually executes the curl command to check the server response, that remote server actually needs to be _running_. If you see the following error, make sure you can actually reach the server you're validating the command against. Here's what we'd see if we ran the example above without first starting the test server:

```
$ curlmin -f curl.sh
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
- [x] recognize `-` for reading from stdin
- [ ] document library usage
- [ ] group cli options

### License

This project is licensed under the [MIT License](LICENSE.md).
