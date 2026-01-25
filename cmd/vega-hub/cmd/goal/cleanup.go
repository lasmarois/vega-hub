package goal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/spf13/cobra"
)

// CleanupResult contains the result of cleaning up a goal
type CleanupResult struct {
	GoalID        string `json:"goal_id"`
	Project       string `json:"project"`
	Branch        string `json:"branch"`
	BranchDeleted bool   `json:"branch_deleted"`
	BranchExisted bool   `json:"branch_existed"`
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup <goal-id> <project>",
	Short: "Clean up a completed goal's branch",
	Long: `Clean up a goal's branch after MR/PR has been merged.

This command is used after:
  1. goal complete --no-merge (which preserves the branch)
  2. MR/PR created and merged externally

Examples:
  vega-hub goal cleanup f3a8b2c my-api
  vega-hub goal cleanup f3a8b2c my-api --json

This command will:
  1. Verify the goal exists in history (completed)
  2. Find and delete the goal branch

NOTE: Only use this after your MR/PR has been merged.`,
	Args: cobra.ExactArgs(2),
	Run:  runCleanup,
}

func init() {
	GoalCmd.AddCommand(cleanupCmd)
}

func runCleanup(c *cobra.Command, args []string) {
	goalID := args[0]
	project := args[1]

	// Get vega-missile directory
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	// Check goal is in history (completed)
	historyFile := filepath.Join(vegaDir, "goals", "history", goalID+".md")
	activeFile := filepath.Join(vegaDir, "goals", "active", goalID+".md")
	icedFile := filepath.Join(vegaDir, "goals", "iced", goalID+".md")

	if _, err := os.Stat(activeFile); err == nil {
		cli.OutputError(cli.ExitStateError, "goal_still_active",
			fmt.Sprintf("Goal '%s' is still active", goalID),
			map[string]string{
				"goal_id": goalID,
				"status":  "active",
			},
			[]cli.ErrorOption{
				{Action: "complete", Description: "Run: vega-hub goal complete " + goalID + " " + project},
			})
	}

	if _, err := os.Stat(icedFile); err == nil {
		cli.OutputError(cli.ExitStateError, "goal_is_iced",
			fmt.Sprintf("Goal '%s' is iced (paused), not completed", goalID),
			map[string]string{
				"goal_id": goalID,
				"status":  "iced",
			},
			[]cli.ErrorOption{
				{Action: "resume", Description: "Resume the goal first, then complete it"},
			})
	}

	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		cli.OutputError(cli.ExitNotFound, "goal_not_found",
			fmt.Sprintf("Goal '%s' not found in history", goalID),
			map[string]string{
				"goal_id":       goalID,
				"expected_path": historyFile,
			},
			[]cli.ErrorOption{
				{Action: "check", Description: "Verify goal ID is correct"},
				{Action: "list", Description: "Run: vega-hub goal list --status completed"},
			})
	}

	// Get project base folder
	projectBase := filepath.Join(vegaDir, "workspaces", project, "worktree-base")
	if _, err := os.Stat(projectBase); os.IsNotExist(err) {
		cli.OutputError(cli.ExitNotFound, "project_not_found",
			fmt.Sprintf("Project '%s' not found", project),
			map[string]string{
				"expected_path": projectBase,
				"project":       project,
			},
			nil)
	}

	// Find the branch name (pattern: goal-<id>-*)
	branchName, err := findGoalBranch(projectBase, goalID)
	if err != nil {
		cli.OutputError(cli.ExitNotFound, "branch_not_found",
			fmt.Sprintf("No branch found for goal '%s'", goalID),
			map[string]string{
				"goal_id": goalID,
				"project": project,
				"error":   err.Error(),
			},
			[]cli.ErrorOption{
				{Action: "check", Description: "Branch may have already been deleted"},
			})
	}

	result := CleanupResult{
		GoalID:        goalID,
		Project:       project,
		Branch:        branchName,
		BranchExisted: true,
	}

	cli.Info("Cleaning up goal %s", goalID)
	cli.Info("  Project: %s", project)
	cli.Info("  Branch: %s", branchName)

	// Delete the branch
	cli.Info("Deleting branch %s...", branchName)
	if err := deleteBranch(projectBase, branchName); err != nil {
		cli.OutputError(cli.ExitStateError, "branch_delete_failed",
			"Could not delete branch",
			map[string]string{
				"branch": branchName,
				"error":  err.Error(),
			},
			[]cli.ErrorOption{
				{Action: "manual", Description: fmt.Sprintf("Run: cd %s && git branch -D %s", projectBase, branchName)},
			})
	}
	result.BranchDeleted = true

	cli.Output(cli.Result{
		Success: true,
		Action:  "goal_cleanup",
		Message: fmt.Sprintf("Cleaned up branch for goal %s", goalID),
		Data:    result,
	})
}

// findGoalBranch finds a branch matching the goal ID pattern
func findGoalBranch(projectBase, goalID string) (string, error) {
	// List all branches
	cmd := exec.Command("git", "-C", projectBase, "branch", "--list", fmt.Sprintf("goal-%s-*", goalID))
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git branch list failed: %w", err)
	}

	branches := strings.TrimSpace(string(output))
	if branches == "" {
		return "", fmt.Errorf("no branch found matching pattern goal-%s-*", goalID)
	}

	// Parse branches (may have * prefix for current branch)
	lines := strings.Split(branches, "\n")
	for _, line := range lines {
		branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if branch != "" {
			return branch, nil
		}
	}

	return "", fmt.Errorf("no valid branch found")
}
