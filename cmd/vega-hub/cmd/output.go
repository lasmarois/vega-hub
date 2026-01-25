package cmd

import (
	"github.com/lasmarois/vega-hub/internal/cli"
)

// Re-export types and functions from internal/cli for backward compatibility

// Exit codes
const (
	ExitSuccess         = cli.ExitSuccess
	ExitValidationError = cli.ExitValidationError
	ExitStateError      = cli.ExitStateError
	ExitNotFound        = cli.ExitNotFound
	ExitConflict        = cli.ExitConflict
	ExitInternalError   = cli.ExitInternalError
)

// Type aliases
type Result = cli.Result
type ErrorInfo = cli.ErrorInfo
type ErrorOption = cli.ErrorOption

// Function wrappers
var (
	Output        = cli.Output
	OutputSuccess = cli.OutputSuccess
	OutputError   = cli.OutputError
	Info          = cli.Info
	Warn          = cli.Warn
)

// GetVegaDir returns the vega-missile directory
func GetVegaDir() (string, error) {
	return cli.GetVegaDir()
}

// IsJSONOutput returns true if JSON output mode is enabled
func IsJSONOutput() bool {
	return cli.JSONOutput
}
