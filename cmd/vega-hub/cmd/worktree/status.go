package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/spf13/cobra"
)

// StatusResult contains worktree status information
type StatusResult struct {
	GoalID           string `json:"goal_id"`
	Path             string `json:"path"`
	Branch           string `json:"branch"`
	BaseBranch       string `json:"base_branch,omitempty"`
	Project          string `json:"project"`
	Exists           bool   `json:"exists"`
	UncommittedFiles int    `json:"uncommitted_files"`
	LastCommit       string `json:"last_commit,omitempty"`
	LastCommitMsg    string `json:"last_commit_message,omitempty"`
}

var statusCmd = &cobra.Command{
	Use:   "status <goal-id>",
	Short: "Show worktree status for a goal",
	Long: `Display detailed status information about a goal's worktree.

Examples:
  vega-hub worktree status f3a8b2c
  vega-hub worktree status f3a8b2c --json

Shows:
  - Worktree path and existence
  - Current branch and base branch
  - Number of uncommitted changes
  - Last commit information`,
	Args: cobra.ExactArgs(1),
	Run:  runStatus,
}

func init() {
	WorktreeCmd.AddCommand(statusCmd)
}

func runStatus(c *cobra.Command, args []string) {
	goalID := args[0]

	// Get vega-missile directory
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	// Find the worktree for this goal
	worktreePath, project, err := findWorktreeForGoal(vegaDir, goalID)

	result := StatusResult{
		GoalID:  goalID,
		Project: project,
	}

	if err != nil || worktreePath == "" {
		// Worktree doesn't exist
		result.Exists = false
		result.Path = ""

		cli.Output(cli.Result{
			Success: true,
			Action:  "worktree_status",
			Message: fmt.Sprintf("No worktree exists for goal %s", goalID),
			Data:    result,
			NextSteps: []string{
				fmt.Sprintf("Create worktree: vega-hub worktree create %s", goalID),
			},
		})
		return
	}

	result.Exists = true
	result.Path = worktreePath

	// Get current branch
	result.Branch = getCurrentBranch(worktreePath)

	// Get base branch from goal info
	parser := goals.NewParser(vegaDir)
	if proj, err := parser.ParseProject(project); err == nil {
		result.BaseBranch = proj.BaseBranch
	}

	// Count uncommitted changes
	result.UncommittedFiles = countUncommittedFiles(worktreePath)

	// Get last commit info
	result.LastCommit, result.LastCommitMsg = getLastCommit(worktreePath)

	// Output
	cli.Output(cli.Result{
		Success: true,
		Action:  "worktree_status",
		Message: fmt.Sprintf("Worktree status for goal %s", goalID),
		Data:    result,
	})

	// Human-readable summary
	if !cli.JSONOutput {
		fmt.Printf("\n  Path: %s\n", result.Path)
		fmt.Printf("  Branch: %s\n", result.Branch)
		if result.BaseBranch != "" {
			fmt.Printf("  Base branch: %s\n", result.BaseBranch)
		}
		fmt.Printf("  Uncommitted files: %d\n", result.UncommittedFiles)
		if result.LastCommit != "" {
			fmt.Printf("  Last commit: %s %s\n", result.LastCommit[:7], result.LastCommitMsg)
		}
	}
}

// findWorktreeForGoal searches for a worktree matching the goal ID
// Returns (path, project, error)
func findWorktreeForGoal(vegaDir, goalID string) (string, string, error) {
	workspacesDir := filepath.Join(vegaDir, "workspaces")

	projects, err := os.ReadDir(workspacesDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to read workspaces: %w", err)
	}

	goalPrefix := fmt.Sprintf("goal-%s-", goalID)

	for _, project := range projects {
		if !project.IsDir() {
			continue
		}

		projectPath := filepath.Join(workspacesDir, project.Name())
		entries, err := os.ReadDir(projectPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), goalPrefix) {
				return filepath.Join(projectPath, entry.Name()), project.Name(), nil
			}
		}
	}

	return "", "", fmt.Errorf("no worktree found for goal %s", goalID)
}

// getCurrentBranch returns the current branch name
func getCurrentBranch(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// countUncommittedFiles returns the number of uncommitted files
func countUncommittedFiles(repoPath string) int {
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}

// getLastCommit returns the last commit hash and message
func getLastCommit(repoPath string) (string, string) {
	cmd := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%H|%s")
	output, err := cmd.Output()
	if err != nil {
		return "", ""
	}

	parts := strings.SplitN(strings.TrimSpace(string(output)), "|", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}
