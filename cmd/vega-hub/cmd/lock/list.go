package lock

import (
	"fmt"
	"time"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/hub"
	"github.com/spf13/cobra"
)

// LockListResult contains the list of current locks
type LockListResult struct {
	Locks []LockDetails `json:"locks"`
	Count int           `json:"count"`
}

// LockDetails contains detailed information about a lock
type LockDetails struct {
	Resource   string         `json:"resource"`
	LockType   hub.LockType   `json:"lock_type"`
	PID        int            `json:"pid"`
	Hostname   string         `json:"hostname"`
	AcquiredAt time.Time      `json:"acquired_at"`
	HeldFor    string         `json:"held_for"`
	Owner      string         `json:"owner,omitempty"`
	GoalID     string         `json:"goal_id,omitempty"`
	Project    string         `json:"project,omitempty"`
	IsStale    bool           `json:"is_stale"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all current locks",
	Long: `List all current resource locks with their details.

Shows PID, timestamp, and stale status for each lock.

Examples:
  vega-hub lock list
  vega-hub lock list --json`,
	Run: runList,
}

func init() {
	LockCmd.AddCommand(listCmd)
}

func runList(c *cobra.Command, args []string) {
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, nil)
	}

	lm := hub.NewLockManager(vegaDir)
	locks, err := lm.ListLocks()
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "list_failed",
			"Failed to list locks",
			map[string]string{"error": err.Error()},
			nil)
	}

	result := LockListResult{
		Locks: make([]LockDetails, 0, len(locks)),
		Count: len(locks),
	}

	for _, l := range locks {
		result.Locks = append(result.Locks, LockDetails{
			Resource:   l.Resource,
			LockType:   l.LockType,
			PID:        l.PID,
			Hostname:   l.Hostname,
			AcquiredAt: l.AcquiredAt,
			HeldFor:    time.Since(l.AcquiredAt).Round(time.Second).String(),
			Owner:      l.Owner,
			GoalID:     l.GoalID,
			Project:    l.Project,
			IsStale:    l.IsStale(),
		})
	}

	if len(locks) == 0 {
		cli.Output(cli.Result{
			Success: true,
			Action:  "lock_list",
			Message: "No active locks",
			Data:    result,
		})
		return
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "lock_list",
		Message: fmt.Sprintf("Found %d active lock(s)", len(locks)),
		Data:    result,
	})

	// Human-readable output
	if !cli.JSONOutput {
		fmt.Println()
		for _, l := range result.Locks {
			staleStr := ""
			if l.IsStale {
				staleStr = " [STALE]"
			}
			fmt.Printf("  %s-%s%s\n", l.LockType, l.Resource, staleStr)
			fmt.Printf("    PID: %d  |  Owner: %s  |  Held for: %s\n", l.PID, orDefault(l.Owner, "-"), l.HeldFor)
			if l.GoalID != "" {
				fmt.Printf("    Goal: %s  |  Project: %s\n", l.GoalID, l.Project)
			}
		}
	}
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
