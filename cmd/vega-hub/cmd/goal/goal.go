package goal

import (
	"github.com/spf13/cobra"
)

// GoalCmd represents the goal command
var GoalCmd = &cobra.Command{
	Use:   "goal",
	Short: "Manage goals",
	Long: `Manage goals in the vega-missile system.

Available subcommands:
  list      List all goals
  create    Create a new goal with worktree
  complete  Complete a goal (merge, cleanup)
  ice       Pause a goal for later
  cleanup   Delete branch after MR/PR merged`,
}

func init() {
	// Subcommands are added in their respective files
}
