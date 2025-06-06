package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/noperator/curlmin/pkg/curlmin"
)

func main() {
	minimizeHeaders := flag.Bool("headers", true, "Minimize headers")
	minimizeCookies := flag.Bool("cookies", true, "Minimize cookies")
	minimizeParams := flag.Bool("params", true, "Minimize query parameters")
	verbose := flag.Bool("v", false, "Verbose output")
	flag.Parse()

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
