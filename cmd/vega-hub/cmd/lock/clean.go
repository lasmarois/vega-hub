package lock

import (
	"fmt"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/hub"
	"github.com/spf13/cobra"
)

// CleanResult contains the result of cleaning stale locks
type CleanResult struct {
	Cleaned int `json:"cleaned"`
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove all stale locks",
	Long: `Remove all stale locks (locks held by dead processes or too old).

A lock is considered stale if:
  - The owning process (PID) is no longer running
  - The lock has been held longer than the stale threshold (15 minutes)

This is safe to run at any time as it only removes locks that are no longer valid.

Examples:
  vega-hub lock clean`,
	Run: runClean,
}

func init() {
	LockCmd.AddCommand(cleanCmd)
}

func runClean(c *cobra.Command, args []string) {
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, nil)
	}

	lm := hub.NewLockManager(vegaDir)

	// First, show what's stale
	locks, err := lm.ListLocks()
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "list_failed",
			"Failed to list locks",
			map[string]string{"error": err.Error()},
			nil)
	}

	staleCount := 0
	for _, l := range locks {
		if l.IsStale() {
			staleCount++
			cli.Info("Stale lock: %s-%s (PID %d)", l.LockType, l.Resource, l.PID)
		}
	}

	if staleCount == 0 {
		cli.Output(cli.Result{
			Success: true,
			Action:  "lock_clean",
			Message: "No stale locks found",
			Data:    CleanResult{Cleaned: 0},
		})
		return
	}

	// Clean stale locks
	cleaned, err := lm.CleanStaleLocks()
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "clean_failed",
			"Failed to clean stale locks",
			map[string]string{"error": err.Error()},
			nil)
	}

	result := CleanResult{Cleaned: cleaned}

	cli.Output(cli.Result{
		Success: true,
		Action:  "lock_clean",
		Message: fmt.Sprintf("Cleaned %d stale lock(s)", cleaned),
		Data:    result,
	})
}
