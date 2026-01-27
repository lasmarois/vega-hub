package goal

import (
	"fmt"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/hub"
	"github.com/spf13/cobra"
)

var (
	cleanupDeleteBranch bool
	cleanupForce        bool
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup <goal-id> <project>",
	Short: "Clean up a goal's worktree and optionally delete branch",
	Long: `Remove a goal's worktree directory and optionally delete the associated branch.

This is useful for:
- Cleaning up after a goal is completed
- Removing stuck or abandoned goals
- Recovering disk space

Use --delete-branch to also remove the local and remote branch.
Use --force to skip the uncommitted changes check.`,
	Args: cobra.ExactArgs(2),
	RunE: runCleanup,
}

func init() {
	GoalCmd.AddCommand(cleanupCmd)
	cleanupCmd.Flags().BoolVar(&cleanupDeleteBranch, "delete-branch", false, "Also delete the goal's branch (local and remote)")
	cleanupCmd.Flags().BoolVar(&cleanupForce, "force", false, "Force cleanup even with uncommitted changes")
}

func runCleanup(cmd *cobra.Command, args []string) error {
	goalID := args[0]
	project := args[1]

	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		return err
	}

	manager := hub.NewCleanupManager(vegaDir)
	if err := manager.CleanupGoal(goalID, project, cleanupDeleteBranch, cleanupForce); err != nil {
		cli.OutputError(cli.ExitStateError, "cleanup_failed",
			fmt.Sprintf("Failed to cleanup goal %s", goalID),
			map[string]string{
				"goal_id": goalID,
				"project": project,
				"error":   err.Error(),
			},
			[]cli.ErrorOption{
				{Flag: "force", Description: "Force cleanup even with uncommitted changes"},
			})
		return nil
	}

	// Success output
	message := fmt.Sprintf("Cleaned up worktree for goal %s", goalID)
	if cleanupDeleteBranch {
		message += " and deleted branch"
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "goal_cleanup",
		Message: message,
		Data: map[string]string{
			"goal_id":        goalID,
			"project":        project,
			"branch_deleted": fmt.Sprintf("%v", cleanupDeleteBranch),
		},
	})

	return nil
}
