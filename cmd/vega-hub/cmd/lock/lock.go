package lock

import (
	"github.com/spf13/cobra"
)

// LockCmd is the parent command for lock operations
var LockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Manage resource locks",
	Long: `Manage resource locks used to prevent concurrent operations on shared resources.

vega-hub uses filesystem-based locks to protect:
  - worktree-base: git pull, branch creation, merge operations
  - branch: branch creation for specific goals
  - merge: merge operations
  - goal-state: goal state transitions

Locks include PID and timestamp for stale detection. Stale locks (process no longer 
running or held too long) are automatically stolen when new lock acquisition is attempted.

Examples:
  vega-hub lock list
  vega-hub lock release --resource my-project --type worktree-base --force
  vega-hub lock clean`,
}

func init() {
	// Subcommands are added in their respective files
}
