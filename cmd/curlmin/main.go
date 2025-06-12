package main

import (
	"fmt"
	"io"
	"os"

	"github.com/noperator/curlmin/pkg/curlmin"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

func main() {
	// Input options
	commandStr := pflag.StringP("command", "c", "", "Curl command as a string")
	commandFile := pflag.StringP("file", "f", "", "File containing the curl command")

	// Minimization options
	minimizeHeaders := pflag.Bool("headers", true, "Minimize headers")
	minimizeCookies := pflag.Bool("cookies", true, "Minimize cookies")
	minimizeParams := pflag.Bool("params", true, "Minimize query parameters")
	verbose := pflag.BoolP("verbose", "v", false, "Verbose output")

	// Response comparison options
	compareStatusCode := pflag.Bool("status", false, "Compare status code")
	compareBodyContent := pflag.Bool("body", true, "Compare body content")
	compareWordCount := pflag.Bool("words", false, "Compare word count")
	compareLineCount := pflag.Bool("lines", false, "Compare line count")
	compareByteCount := pflag.Bool("bytes", false, "Compare byte count")

	pflag.Parse()

	// If any other comparison option is set, disable the default body comparison
	if *compareStatusCode || *compareWordCount || *compareLineCount || *compareByteCount {
		// The pflag package sets compareBodyContent to true by default
		// If the user didn't explicitly set it to true, we should disable it
		bodyFlagExplicitlySet := false
		pflag.Visit(func(f *pflag.Flag) {
			if f.Name == "body" {
				bodyFlagExplicitlySet = true
			}
		})

		if pflag.Lookup("body").Value.String() == "true" && !bodyFlagExplicitlySet {
			*compareBodyContent = false
		}
	}

	var curlCmd string

	// Determine the source of the curl command
	if *commandStr != "" {
		// Use the command string provided via -command/-c flag
		curlCmd = *commandStr
	} else if *commandFile != "" {
		// Read the command from the file provided via -file/-f flag
		var fileBytes []byte
		var err error

		if *commandFile == "-" {
			// Read from stdin if file is "-"
			fileBytes, err = io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Read from the specified file
			fileBytes, err = os.ReadFile(*commandFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading from file %s: %v\n", *commandFile, err)
				os.Exit(1)
			}
		}
		curlCmd = string(fileBytes)
	} else if stdinAvailable() {
		// If no command source is specified but stdin is available, read from stdin
		fileBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
			os.Exit(1)
		}
		curlCmd = string(fileBytes)
	} else {
		// If no command source is specified and stdin is not available, show usage and exit
		fmt.Fprintf(os.Stderr, "Error: either --command/-c or --file/-f is required, or pipe input via stdin\n\n")
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		pflag.PrintDefaults()
		os.Exit(1)
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

// stdinAvailable checks if stdin is available (not a terminal and has data to read)
func stdinAvailable() bool {
	// Check if stdin is a terminal
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return false
	}

	// Check if there's data available to read
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	return (stat.Mode() & os.ModeCharDevice) == 0
}
