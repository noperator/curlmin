package main

import (
	"fmt"
	"log"
	"net/http"
)

const (
	requiredCookie      = "session=abc123"
	requiredAuthToken   = "Bearer xyz789"
	requiredQueryParam  = "auth_key=def456"
	successResponseBody = "Authentication successful!"
	failureResponseBody = "Authentication failed!"
)

func main() {
	// Create a handler for the test endpoint
	http.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		// Check for required cookie
		cookieFound := false
		for _, cookie := range r.Cookies() {
			if cookie.Name == "session" && cookie.Value == "abc123" {
				cookieFound = true
				break
			}
		}

		// Check for required auth token
		authHeader := r.Header.Get("Authorization")
		authTokenValid := authHeader == requiredAuthToken

		// Check for required query parameter
		queryParamValid := false
		queryParams := r.URL.Query()
		if queryParams.Get("auth_key") == "def456" {
			queryParamValid = true
		}

		// Log the request details for debugging
		fmt.Printf("Request received:\n")
		fmt.Printf("  Method: %s\n", r.Method)
		fmt.Printf("  URL: %s\n", r.URL.String())
		fmt.Printf("  Headers: %v\n", r.Header)
		fmt.Printf("  Cookies: %v\n", r.Cookies())
		fmt.Printf("  Auth validation: Cookie=%v, Token=%v, QueryParam=%v\n",
			cookieFound, authTokenValid, queryParamValid)

		// Check if all required elements are present
		if cookieFound && authTokenValid && queryParamValid {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(successResponseBody))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(failureResponseBody))
		}
	})

	// Start the server
	fmt.Println("Starting test server on http://localhost:8080")
	fmt.Println("Required authentication:")
	fmt.Printf("  Cookie: %s\n", requiredCookie)
	fmt.Printf("  Auth Token: %s\n", requiredAuthToken)
	fmt.Printf("  Query Parameter: %s\n", requiredQueryParam)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
