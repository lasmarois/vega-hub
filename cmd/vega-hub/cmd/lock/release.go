package lock

import (
	"fmt"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/hub"
	"github.com/spf13/cobra"
)

var (
	releaseResource string
	releaseLockType string
	releaseForce    bool
)

// ReleaseResult contains the result of releasing a lock
type ReleaseResult struct {
	Resource string       `json:"resource"`
	LockType hub.LockType `json:"lock_type"`
	Released bool         `json:"released"`
}

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Force release a lock",
	Long: `Force release a resource lock (escape hatch for stuck locks).

WARNING: Force-releasing a lock held by a running process can cause corruption.
Use this only when you're sure the owning process has crashed or hung.

Lock types:
  worktree-base  - Protects worktree-base operations
  branch         - Protects branch creation
  merge          - Protects merge operations
  goal-state     - Protects goal state transitions

Examples:
  vega-hub lock release --resource my-project --type worktree-base --force
  vega-hub lock release --resource my-project-goal123 --type branch --force`,
	Run: runRelease,
}

func init() {
	LockCmd.AddCommand(releaseCmd)
	releaseCmd.Flags().StringVar(&releaseResource, "resource", "", "Resource name (required)")
	releaseCmd.Flags().StringVar(&releaseLockType, "type", "worktree-base", "Lock type (worktree-base, branch, merge, goal-state)")
	releaseCmd.Flags().BoolVarP(&releaseForce, "force", "f", false, "Force release (required for safety)")
	releaseCmd.MarkFlagRequired("resource")
}

func runRelease(c *cobra.Command, args []string) {
	if !releaseForce {
		cli.OutputError(cli.ExitValidationError, "force_required",
			"The --force flag is required to release locks",
			nil,
			[]cli.ErrorOption{
				{Flag: "force", Description: "Confirm you want to force-release the lock"},
			})
	}

	if releaseResource == "" {
		cli.OutputError(cli.ExitValidationError, "resource_required",
			"Resource name is required",
			nil,
			[]cli.ErrorOption{
				{Flag: "resource", Description: "Specify the resource name"},
			})
	}

	// Validate lock type
	lockType := hub.LockType(releaseLockType)
	validTypes := map[hub.LockType]bool{
		hub.LockWorktreeBase: true,
		hub.LockBranch:       true,
		hub.LockMerge:        true,
		hub.LockGoalState:    true,
		hub.LockProject:      true,
	}
	if !validTypes[lockType] {
		cli.OutputError(cli.ExitValidationError, "invalid_lock_type",
			fmt.Sprintf("Invalid lock type: %s", releaseLockType),
			map[string]string{"valid_types": "worktree-base, branch, merge, goal-state, project"},
			nil)
	}

	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, nil)
	}

	lm := hub.NewLockManager(vegaDir)

	// Show lock info before releasing
	locks, _ := lm.ListLocks()
	var targetLock *hub.LockInfo
	for _, l := range locks {
		if l.LockType == lockType && l.Resource == releaseResource {
			targetLock = l
			break
		}
	}

	if targetLock == nil {
		cli.OutputError(cli.ExitNotFound, "lock_not_found",
			fmt.Sprintf("No lock found for %s-%s", lockType, releaseResource),
			nil,
			[]cli.ErrorOption{
				{Action: "list", Description: "Run 'vega-hub lock list' to see active locks"},
			})
	}

	// Warn if lock doesn't appear stale
	if !targetLock.IsStale() {
		cli.Warn("Lock appears to be held by a running process (PID %d)", targetLock.PID)
		cli.Warn("Force-releasing may cause data corruption!")
	}

	// Force release
	if err := lm.ForceRelease(lockType, releaseResource); err != nil {
		cli.OutputError(cli.ExitInternalError, "release_failed",
			"Failed to release lock",
			map[string]string{"error": err.Error()},
			nil)
	}

	result := ReleaseResult{
		Resource: releaseResource,
		LockType: lockType,
		Released: true,
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "lock_release",
		Message: fmt.Sprintf("Force-released lock: %s-%s", lockType, releaseResource),
		Data:    result,
	})
}
