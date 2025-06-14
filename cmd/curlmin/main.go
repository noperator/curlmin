package main

import (
	"fmt"
	"io"
	"os"

	"github.com/noperator/curlmin/pkg/curlmin"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

var (
	// Input options
	commandStr  string
	commandFile string

	// Minimization options
	minimizeHeaders bool
	minimizeCookies bool
	minimizeParams  bool
	verbose         bool

	// Response comparison options
	compareStatusCode  bool
	compareBodyContent bool
	compareWordCount   bool
	compareLineCount   bool
	compareByteCount   bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:                   "curlmin",
	Short:                 "Minimize curl commands by removing unnecessary options",
	Long:                  `curlmin is a tool that minimizes curl commands by removing unnecessary options while preserving the same behavior.`,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		// If any other comparison option is set, disable the default body comparison
		if compareStatusCode || compareWordCount || compareLineCount || compareByteCount {
			// Check if body flag was explicitly set
			bodyFlagExplicitlySet := false
			cmd.Flags().Visit(func(f *pflag.Flag) {
				if f.Name == "body" {
					bodyFlagExplicitlySet = true
				}
			})

			if cmd.Flags().Lookup("body").Value.String() == "true" && !bodyFlagExplicitlySet {
				compareBodyContent = false
			}
		}

		var curlCmd string

		// Determine the source of the curl command
		if commandStr != "" {
			// Use the command string provided via -command/-c flag
			curlCmd = commandStr
		} else if commandFile != "" {
			// Read the command from the file provided via -file/-f flag
			var fileBytes []byte
			var err error

			if commandFile == "-" {
				// Read from stdin if file is "-"
				fileBytes, err = io.ReadAll(os.Stdin)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading from stdin: %v\n", err)
					os.Exit(1)
				}
			} else {
				// Read from the specified file
				fileBytes, err = os.ReadFile(commandFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading from file %s: %v\n", commandFile, err)
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
			cmd.Help()
			os.Exit(1)
		}

		// Print the original curl command if verbose
		if verbose {
			fmt.Println("Original curl command:")
			fmt.Println(curlCmd)
			fmt.Println()
		}

		options := curlmin.Options{
			MinimizeHeaders: minimizeHeaders,
			MinimizeCookies: minimizeCookies,
			MinimizeParams:  minimizeParams,
			Verbose:         verbose,
			// Response comparison options
			CompareStatusCode:  compareStatusCode,
			CompareBodyContent: compareBodyContent,
			CompareWordCount:   compareWordCount,
			CompareLineCount:   compareLineCount,
			CompareByteCount:   compareByteCount,
		}

		min := curlmin.New(options)

		minimizedCmd, err := min.MinimizeCurlCommand(curlCmd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error minimizing curl command: %v\n", err)
			os.Exit(1)
		}

		// Print the minimized curl command
		if verbose {
			fmt.Println("Minimized curl command:")
		}
		fmt.Println(minimizedCmd)
	},
}

func init() {
	// Input options group
	rootCmd.Flags().StringVarP(&commandStr, "command", "c", "", "Curl command as a string")
	rootCmd.Flags().StringVarP(&commandFile, "file", "f", "", "File containing the curl command")

	// Mark flags with their group
	for _, name := range []string{"command", "file"} {
		flag := rootCmd.Flags().Lookup(name)
		if flag != nil {
			flag.Annotations = make(map[string][]string)
			flag.Annotations["group"] = []string{"Input"}
		}
	}

	// Comparison options group
	rootCmd.Flags().BoolVar(&compareStatusCode, "status", false, "Compare status code")
	rootCmd.Flags().BoolVar(&compareBodyContent, "body", true, "Compare body content")
	rootCmd.Flags().BoolVar(&compareWordCount, "words", false, "Compare word count")
	rootCmd.Flags().BoolVar(&compareLineCount, "lines", false, "Compare line count")
	rootCmd.Flags().BoolVar(&compareByteCount, "bytes", false, "Compare byte count")

	// Mark flags with their group
	for _, name := range []string{"status", "body", "words", "lines", "bytes"} {
		flag := rootCmd.Flags().Lookup(name)
		if flag != nil {
			flag.Annotations = make(map[string][]string)
			flag.Annotations["group"] = []string{"Comparison"}
		}
	}

	// Minimization options group
	rootCmd.Flags().BoolVar(&minimizeHeaders, "headers", true, "Minimize headers")
	rootCmd.Flags().BoolVar(&minimizeCookies, "cookies", true, "Minimize cookies")
	rootCmd.Flags().BoolVar(&minimizeParams, "params", true, "Minimize query parameters")

	// Mark flags with their group
	for _, name := range []string{"headers", "cookies", "params"} {
		flag := rootCmd.Flags().Lookup(name)
		if flag != nil {
			flag.Annotations = make(map[string][]string)
			flag.Annotations["group"] = []string{"Minimization"}
		}
	}

	// Flags group (for flags that don't fit in other categories)
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// Set up custom help template to display grouped flags
	cobra.AddTemplateFunc("FlagsInGroup", FlagsInGroup)
	cobra.AddTemplateFunc("FilterFlags", FilterFlags)
	rootCmd.SetUsageTemplate(usageTemplate)
}

// FlagsInGroup returns all flags in a specific group
func FlagsInGroup(cmd *cobra.Command, group string) *pflag.FlagSet {
	fs := pflag.NewFlagSet(group, pflag.ContinueOnError)

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if annotations := f.Annotations["group"]; len(annotations) > 0 {
			if annotations[0] == group {
				fs.AddFlag(f)
			}
		}
	})

	return fs
}

// FilterFlags returns a FlagSet with only the flags that are not in any group
func FilterFlags(cmd *cobra.Command) *pflag.FlagSet {
	fs := pflag.NewFlagSet("filtered", pflag.ContinueOnError)

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Check if this flag is in any group
		inGroup := false
		if annotations := f.Annotations["group"]; len(annotations) > 0 {
			inGroup = true
		}

		// If it's not in any group, add it to our filtered set
		if !inGroup && f.Name != "help" {
			fs.AddFlag(f)
		}
	})

	// Add help flag separately to ensure it's only added once
	if helpFlag := cmd.Flags().Lookup("help"); helpFlag != nil {
		fs.AddFlag(helpFlag)
	}

	return fs
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

// Custom usage template with grouped flags
const usageTemplate = `Usage:
  {{.UseLine}}

{{if FlagsInGroup . "Input"}}Input:
{{(FlagsInGroup . "Input").FlagUsages | trimTrailingWhitespaces}}

{{end}}{{if FlagsInGroup . "Comparison"}}Comparison:
{{(FlagsInGroup . "Comparison").FlagUsages | trimTrailingWhitespaces}}

{{end}}{{if FlagsInGroup . "Minimization"}}Minimization:
{{(FlagsInGroup . "Minimization").FlagUsages | trimTrailingWhitespaces}}

{{end}}Flags:
{{(FilterFlags .).FlagUsages | trimTrailingWhitespaces}}
`
