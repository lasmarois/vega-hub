package cmd

import (
	"encoding/json"
	"fmt"
	"os"
)

// Exit codes following AI-friendly error categories
const (
	ExitSuccess         = 0
	ExitValidationError = 1 // Invalid input (user's fault)
	ExitStateError      = 2 // Invalid state (e.g., uncommitted changes)
	ExitNotFound        = 3 // Resource doesn't exist
	ExitConflict        = 4 // Operation would cause conflict
	ExitInternalError   = 5 // Bug or unexpected condition
)

// Result represents a command result for JSON output
type Result struct {
	Success   bool        `json:"success"`
	Action    string      `json:"action,omitempty"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	NextSteps []string    `json:"next_steps,omitempty"`
}

// ErrorInfo provides structured error information
type ErrorInfo struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
	Options []ErrorOption     `json:"options,omitempty"`
}

// ErrorOption provides actionable options for error recovery
type ErrorOption struct {
	Flag        string `json:"flag,omitempty"`
	Action      string `json:"action,omitempty"`
	Description string `json:"description"`
}

// Output prints the result in the appropriate format
func Output(result Result) {
	if jsonOutput {
		outputJSON(result)
	} else {
		outputHuman(result)
	}
}

// OutputSuccess prints a success message
func OutputSuccess(action, message string, data interface{}) {
	Output(Result{
		Success: true,
		Action:  action,
		Message: message,
		Data:    data,
	})
}

// OutputError prints an error and exits with the appropriate code
func OutputError(exitCode int, code, message string, details map[string]string, options []ErrorOption) {
	result := Result{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
			Options: options,
		},
	}

	Output(result)
	os.Exit(exitCode)
}

func outputJSON(result Result) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(result)
}

func outputHuman(result Result) {
	if quietMode && result.Success {
		// In quiet mode, only output essential data for success
		if result.Data != nil {
			fmt.Println(result.Data)
		}
		return
	}

	if result.Success {
		if result.Message != "" {
			fmt.Printf("✓ %s\n", result.Message)
		}
		if result.Data != nil {
			// Pretty print data based on type
			switch v := result.Data.(type) {
			case string:
				fmt.Println(v)
			case map[string]interface{}:
				for k, val := range v {
					fmt.Printf("  %s: %v\n", k, val)
				}
			default:
				fmt.Printf("%v\n", v)
			}
		}
		if len(result.NextSteps) > 0 {
			fmt.Println("\nNext steps:")
			for _, step := range result.NextSteps {
				fmt.Printf("  → %s\n", step)
			}
		}
	} else if result.Error != nil {
		fmt.Fprintf(os.Stderr, "✗ Error: %s\n", result.Error.Message)
		if len(result.Error.Details) > 0 {
			for k, v := range result.Error.Details {
				fmt.Fprintf(os.Stderr, "  %s: %s\n", k, v)
			}
		}
		if len(result.Error.Options) > 0 {
			fmt.Fprintln(os.Stderr, "\nOptions:")
			for _, opt := range result.Error.Options {
				if opt.Flag != "" {
					fmt.Fprintf(os.Stderr, "  --%s: %s\n", opt.Flag, opt.Description)
				} else if opt.Action != "" {
					fmt.Fprintf(os.Stderr, "  %s: %s\n", opt.Action, opt.Description)
				}
			}
		}
	}
}

// Info prints an informational message (not shown in quiet mode)
func Info(format string, args ...interface{}) {
	if !quietMode {
		fmt.Printf(format+"\n", args...)
	}
}

// Warn prints a warning message (always shown)
func Warn(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "⚠ "+format+"\n", args...)
}
