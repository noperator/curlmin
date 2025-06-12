package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/noperator/curlmin/pkg/curlmin"
)

// isFlagPassed checks if a flag was explicitly passed on the command line
func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func main() {
	// Minimization options
	minimizeHeaders := flag.Bool("headers", true, "Minimize headers")
	minimizeCookies := flag.Bool("cookies", true, "Minimize cookies")
	minimizeParams := flag.Bool("params", true, "Minimize query parameters")
	verbose := flag.Bool("v", false, "Verbose output")

	// Response comparison options
	compareStatusCode := flag.Bool("status", false, "Compare status code")
	compareBodyContent := flag.Bool("body", true, "Compare body content")
	compareWordCount := flag.Bool("words", false, "Compare word count")
	compareLineCount := flag.Bool("lines", false, "Compare line count")
	compareByteCount := flag.Bool("bytes", false, "Compare byte count")

	flag.Parse()

	// If any other comparison option is set, disable the default body comparison
	if *compareStatusCode || *compareWordCount || *compareLineCount || *compareByteCount {
		// The flag package sets compareBodyContent to true by default
		// If the user didn't explicitly set it to true, we should disable it
		if flag.Lookup("body").Value.String() == "true" && !isFlagPassed("body") {
			*compareBodyContent = false
		}
	}

	var curlCmd string
	args := flag.Args()

	if len(args) > 0 {
		curlCmd = strings.Join(args, " ")
	} else {
		stdinBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
		curlCmd = string(stdinBytes)
	}

	// Print the original curl command if verbose
	if *verbose {
		fmt.Println("Original curl command:")
		fmt.Println(curlCmd)
		fmt.Println()
	}

	options := curlmin.Options{
		MinimizeHeaders: *minimizeHeaders,
		MinimizeCookies: *minimizeCookies,
		MinimizeParams:  *minimizeParams,
		Verbose:         *verbose,
		// Response comparison options
		CompareStatusCode:  *compareStatusCode,
		CompareBodyContent: *compareBodyContent,
		CompareWordCount:   *compareWordCount,
		CompareLineCount:   *compareLineCount,
		CompareByteCount:   *compareByteCount,
	}

	min := curlmin.New(options)

	minimizedCmd, err := min.MinimizeCurlCommand(curlCmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error minimizing curl command: %v\n", err)
		os.Exit(1)
	}

	// Print the minimized curl command
	if *verbose {
		fmt.Println("Minimized curl command:")
	}
	fmt.Println(minimizedCmd)
}
