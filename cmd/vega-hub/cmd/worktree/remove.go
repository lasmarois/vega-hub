package worktree

import (
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/spf13/cobra"
)

var (
	removeForce       bool
	removeDeleteBranch bool
)

// RemoveResult contains the result of removing a worktree
type RemoveResult struct {
	GoalID        string `json:"goal_id"`
	Path          string `json:"path"`
	Branch        string `json:"branch"`
	BranchDeleted bool   `json:"branch_deleted"`
}

var removeCmd = &cobra.Command{
	Use:   "remove <goal-id>",
	Short: "Remove a worktree for a goal",
	Long: `Remove the git worktree associated with a goal.

Examples:
  vega-hub worktree remove f3a8b2c
  vega-hub worktree remove f3a8b2c --force
  vega-hub worktree remove f3a8b2c --delete-branch

By default, this command will:
  - Check for uncommitted changes (fails if found)
  - Remove the worktree directory
  - Keep the branch (for resuming later)

Use --force to remove even with uncommitted changes.
Use --delete-branch to also delete the goal branch.`,
	Args: cobra.ExactArgs(1),
	Run:  runRemove,
}

func init() {
	WorktreeCmd.AddCommand(removeCmd)
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Remove even with uncommitted changes")
	removeCmd.Flags().BoolVar(&removeDeleteBranch, "delete-branch", false, "Also delete the goal branch")
}

func runRemove(c *cobra.Command, args []string) {
	goalID := args[0]

	// Get vega-missile directory
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	// Find the worktree
	worktreePath, project, err := findWorktreeForGoal(vegaDir, goalID)
	if err != nil || worktreePath == "" {
		cli.OutputError(cli.ExitNotFound, "worktree_not_found",
			fmt.Sprintf("No worktree found for goal %s", goalID),
			map[string]string{"goal_id": goalID},
			[]cli.ErrorOption{
				{Action: "status", Description: fmt.Sprintf("Run: vega-hub worktree status %s", goalID)},
			})
	}

	// Get current branch before removal
	branch := getCurrentBranch(worktreePath)

	// Check for uncommitted changes (unless --force)
	if !removeForce {
		uncommitted := countUncommittedFiles(worktreePath)
		if uncommitted > 0 {
			cli.OutputError(cli.ExitStateError, "uncommitted_changes",
				fmt.Sprintf("Worktree has %d uncommitted files", uncommitted),
				map[string]string{
					"goal_id":           goalID,
					"path":              worktreePath,
					"uncommitted_files": fmt.Sprintf("%d", uncommitted),
				},
				[]cli.ErrorOption{
					{Flag: "force", Description: "Remove anyway (discards changes)"},
					{Action: "commit", Description: "Commit changes first, then retry"},
				})
		}
	}

	// Get project base for git operations
	projectBase := filepath.Join(vegaDir, "workspaces", project, "worktree-base")

	// Remove the worktree
	var removeArgs []string
	if removeForce {
		removeArgs = []string{"-C", projectBase, "worktree", "remove", "--force", worktreePath}
	} else {
		removeArgs = []string{"-C", projectBase, "worktree", "remove", worktreePath}
	}

	cmd := exec.Command("git", removeArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		cli.OutputError(cli.ExitStateError, "worktree_remove_failed",
			"Failed to remove worktree",
			map[string]string{
				"path":  worktreePath,
				"error": fmt.Sprintf("%s: %s", err, string(output)),
			},
			[]cli.ErrorOption{
				{Flag: "force", Description: "Try with --force"},
			})
	}

	// Delete branch if requested
	branchDeleted := false
	if removeDeleteBranch && branch != "" {
		cmd = exec.Command("git", "-C", projectBase, "branch", "-D", branch)
		if err := cmd.Run(); err != nil {
			cli.Warn("Failed to delete branch %s: %v", branch, err)
		} else {
			branchDeleted = true
		}
	}

	// Success output
	result := RemoveResult{
		GoalID:        goalID,
		Path:          worktreePath,
		Branch:        branch,
		BranchDeleted: branchDeleted,
	}

	message := fmt.Sprintf("Removed worktree for goal %s", goalID)
	if branchDeleted {
		message += " (branch deleted)"
	} else if branch != "" {
		message += fmt.Sprintf(" (branch '%s' preserved)", branch)
	}

	nextSteps := []string{}
	if !branchDeleted && branch != "" {
		nextSteps = append(nextSteps, fmt.Sprintf("Resume later: vega-hub worktree create %s", goalID))
		nextSteps = append(nextSteps, fmt.Sprintf("Delete branch: git -C %s branch -D %s", projectBase, branch))
	}

	cli.Output(cli.Result{
		Success:   true,
		Action:    "worktree_remove",
		Message:   message,
		Data:      result,
		NextSteps: nextSteps,
	})

	// Human-readable summary
	if !cli.JSONOutput {
		fmt.Printf("\n  Removed: %s\n", worktreePath)
		if branchDeleted {
			fmt.Printf("  Branch deleted: %s\n", branch)
		} else if branch != "" {
			fmt.Printf("  Branch preserved: %s\n", branch)
		}
	}
}
