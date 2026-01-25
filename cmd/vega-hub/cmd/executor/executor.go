package executor

import (
	"github.com/spf13/cobra"
)

// ExecutorCmd represents the executor command
var ExecutorCmd = &cobra.Command{
	Use:   "executor",
	Short: "Manage executors",
	Long: `Manage Claude Code executors in the vega-missile system.

Available subcommands:
  spawn     Spawn a new executor for a goal
  list      List active executors
  stop      Stop an executor`,
}

func init() {
	// Subcommands are added in their respective files
}
