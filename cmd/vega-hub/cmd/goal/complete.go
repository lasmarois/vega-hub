package goal

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/spf13/cobra"
)

var (
	completeNoMerge bool
	completeForce   bool
)

// CompleteResult contains the result of completing a goal
type CompleteResult struct {
	GoalID       string `json:"goal_id"`
	Title        string `json:"title"`
	Project      string `json:"project"`
	Merged       bool   `json:"merged"`
	MergedTo     string `json:"merged_to,omitempty"`
	MergedFrom   string `json:"merged_from,omitempty"`
	WorktreeRemoved bool `json:"worktree_removed"`
	BranchDeleted   bool `json:"branch_deleted"`
	GoalArchived    bool `json:"goal_archived"`
	HistoryFile     string `json:"history_file"`
}

var completeCmd = &cobra.Command{
	Use:   "complete <goal-id> <project>",
	Short: "Complete a goal (merge, cleanup)",
	Long: `Complete a goal: merge branch to base, remove worktree, archive goal.

Examples:
  vega-hub goal complete f3a8b2c my-api
  vega-hub goal complete f3a8b2c my-api --no-merge
  vega-hub goal complete f3a8b2c my-api --force

This command will:
  1. Merge the goal branch to the project's base branch (unless --no-merge)
  2. Remove the worktree directory
  3. Delete the branch (unless --no-merge)
  4. Move the goal file to goals/history/
  5. Update REGISTRY.md (Active -> Completed)
  6. Update the project config

NOTE: Executor should archive planning files before running this command.`,
	Args: cobra.ExactArgs(2),
	Run:  runComplete,
}

func init() {
	GoalCmd.AddCommand(completeCmd)
	completeCmd.Flags().BoolVar(&completeNoMerge, "no-merge", false, "Skip merging (use when creating MR/PR instead)")
	completeCmd.Flags().BoolVarP(&completeForce, "force", "f", false, "Skip safety checks for uncommitted changes")
}

func runComplete(c *cobra.Command, args []string) {
	goalID := args[0]
	project := args[1]

	// Get vega-missile directory
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	// Validate goal exists and is active
	goalFile := filepath.Join(vegaDir, "goals", "active", goalID+".md")
	if _, err := os.Stat(goalFile); os.IsNotExist(err) {
		cli.OutputError(cli.ExitNotFound, "goal_not_found",
			fmt.Sprintf("Active goal '%s' not found", goalID),
			map[string]string{
				"expected_path": goalFile,
				"goal_id":       goalID,
			},
			[]cli.ErrorOption{
				{Action: "check", Description: "Verify goal ID is correct and goal is active"},
			})
	}

	// Get goal title from file
	goalTitle := getGoalTitle(goalFile, goalID)

	// Get project base folder and branch
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

	baseBranch, err := getProjectBaseBranch(vegaDir, project)
	if err != nil {
		cli.OutputError(cli.ExitStateError, "no_base_branch",
			fmt.Sprintf("Could not determine base branch for project '%s'", project),
			map[string]string{"error": err.Error()},
			nil)
	}

	// Find the worktree directory (pattern: goal-<id>-*)
	worktreeDir, err := findWorktreeDir(vegaDir, project, goalID)
	if err != nil {
		cli.OutputError(cli.ExitNotFound, "worktree_not_found",
			err.Error(),
			map[string]string{
				"goal_id": goalID,
				"project": project,
			},
			[]cli.ErrorOption{
				{Action: "check", Description: "Verify worktree exists for this goal"},
			})
	}

	// Get the branch name from the worktree
	branchName, err := getWorktreeBranch(worktreeDir)
	if err != nil {
		cli.OutputError(cli.ExitStateError, "branch_detection_failed",
			"Could not determine branch name from worktree",
			map[string]string{
				"worktree": worktreeDir,
				"error":    err.Error(),
			},
			nil)
	}

	// Safety check: verify worktree is clean (unless --force)
	if !completeForce {
		if err := checkWorktreeClean(worktreeDir); err != nil {
			cli.OutputError(cli.ExitStateError, "uncommitted_changes",
				"Worktree has uncommitted changes",
				map[string]string{
					"worktree": worktreeDir,
					"details":  err.Error(),
				},
				[]cli.ErrorOption{
					{Flag: "force", Description: "Discard changes and proceed (destructive)"},
					{Action: "commit", Description: "Commit or stash changes first"},
				})
		}
	}

	cli.Info("Completing goal %s: %s", goalID, goalTitle)
	cli.Info("  Project: %s", project)
	cli.Info("  Worktree: %s", worktreeDir)
	cli.Info("  Branch: %s -> %s", branchName, baseBranch)

	result := CompleteResult{
		GoalID:  goalID,
		Title:   goalTitle,
		Project: project,
	}

	// Step 1: Merge branch (unless --no-merge)
	if !completeNoMerge {
		cli.Info("Merging %s to %s...", branchName, baseBranch)
		if err := mergeBranch(projectBase, branchName, baseBranch, goalID, goalTitle); err != nil {
			cli.OutputError(cli.ExitConflict, "merge_failed",
				"Merge failed",
				map[string]string{
					"source":      branchName,
					"target":      baseBranch,
					"error":       err.Error(),
				},
				[]cli.ErrorOption{
					{Action: "resolve", Description: "Resolve conflicts manually, then run with --no-merge"},
					{Flag: "no-merge", Description: "Skip merge (create MR/PR instead)"},
				})
		}
		result.Merged = true
		result.MergedTo = baseBranch
		result.MergedFrom = branchName
	} else {
		cli.Info("Skipping merge (--no-merge specified)")
		cli.Info("Remember to create MR/PR for branch: %s", branchName)
	}

	// Step 2: Remove worktree
	cli.Info("Removing worktree...")
	if err := removeWorktreeComplete(projectBase, worktreeDir); err != nil {
		cli.Warn("Could not remove worktree cleanly: %v", err)
	}
	result.WorktreeRemoved = true

	// Step 3: Delete branch (unless --no-merge)
	if !completeNoMerge {
		cli.Info("Deleting branch %s...", branchName)
		if err := deleteBranch(projectBase, branchName); err != nil {
			cli.Warn("Could not delete branch: %v", err)
		} else {
			result.BranchDeleted = true
		}
	}

	// Step 4: Move goal file to history
	cli.Info("Moving goal to history...")
	historyDir := filepath.Join(vegaDir, "goals", "history")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		cli.Warn("Could not create history directory: %v", err)
	}
	historyFile := filepath.Join(historyDir, goalID+".md")
	if err := os.Rename(goalFile, historyFile); err != nil {
		cli.Warn("Could not move goal to history: %v", err)
	} else {
		result.GoalArchived = true
		result.HistoryFile = historyFile
	}

	// Step 5: Update registry
	registryPath := filepath.Join(vegaDir, "goals", "REGISTRY.md")
	if err := completeGoalInRegistry(registryPath, goalID, goalTitle, project); err != nil {
		cli.Warn("Could not update registry: %v", err)
	}

	// Step 6: Update project config
	projectConfig := filepath.Join(vegaDir, "projects", project+".md")
	if err := completeGoalInProjectConfig(projectConfig, goalID, goalTitle); err != nil {
		cli.Warn("Could not update project config: %v", err)
	}

	// Success output
	nextSteps := []string{}
	if completeNoMerge {
		nextSteps = append(nextSteps, fmt.Sprintf("Create MR/PR for branch: %s", branchName))
	}

	cli.Output(cli.Result{
		Success:   true,
		Action:    "goal_complete",
		Message:   fmt.Sprintf("Completed goal %s: %s", goalID, goalTitle),
		Data:      result,
		NextSteps: nextSteps,
	})
}

// getGoalTitle extracts the title from a goal file
func getGoalTitle(goalFile, goalID string) string {
	file, err := os.Open(goalFile)
	if err != nil {
		return "goal-" + goalID
	}
	defer file.Close()

	// Pattern: # Goal <id>: <title> (supports both hash and numeric IDs)
	re := regexp.MustCompile(`^# Goal #?[a-f0-9]+: (.+)$`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			return matches[1]
		}
	}

	return "goal-" + goalID
}

// findWorktreeDir finds the worktree directory for a goal
func findWorktreeDir(vegaDir, project, goalID string) (string, error) {
	pattern := filepath.Join(vegaDir, "workspaces", project, fmt.Sprintf("goal-%s-*", goalID))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}

	// Filter to only directories
	var dirs []string
	for _, m := range matches {
		if info, err := os.Stat(m); err == nil && info.IsDir() {
			dirs = append(dirs, m)
		}
	}

	if len(dirs) == 0 {
		return "", fmt.Errorf("no worktree found matching pattern: goal-%s-*", goalID)
	}
	if len(dirs) > 1 {
		return "", fmt.Errorf("multiple worktrees found for goal %s: %v", goalID, dirs)
	}

	return dirs[0], nil
}

// getWorktreeBranch gets the current branch name from a worktree
func getWorktreeBranch(worktreeDir string) (string, error) {
	cmd := exec.Command("git", "-C", worktreeDir, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		// Fallback: extract from directory name
		return filepath.Base(worktreeDir), nil
	}
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return filepath.Base(worktreeDir), nil
	}
	return branch, nil
}

// checkWorktreeClean verifies the worktree has no uncommitted changes
func checkWorktreeClean(worktreeDir string) error {
	cmd := exec.Command("git", "-C", worktreeDir, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}
	if len(strings.TrimSpace(string(output))) > 0 {
		return fmt.Errorf("uncommitted changes:\n%s", string(output))
	}
	return nil
}

// mergeBranch merges the source branch to target branch
func mergeBranch(projectBase, sourceBranch, targetBranch, goalID, goalTitle string) error {
	// Checkout target branch
	cmd := exec.Command("git", "-C", projectBase, "checkout", targetBranch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("checkout %s: %s", targetBranch, string(output))
	}

	// Merge source branch
	mergeMsg := fmt.Sprintf("Merge goal %s: %s", goalID, goalTitle)
	cmd = exec.Command("git", "-C", projectBase, "merge", sourceBranch, "-m", mergeMsg)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("merge: %s", string(output))
	}

	return nil
}

// removeWorktreeComplete removes a worktree, with fallback to force removal
func removeWorktreeComplete(projectBase, worktreeDir string) error {
	// Try normal removal first
	cmd := exec.Command("git", "-C", projectBase, "worktree", "remove", worktreeDir, "--force")
	if err := cmd.Run(); err != nil {
		// Fallback: remove directory and prune
		os.RemoveAll(worktreeDir)
		exec.Command("git", "-C", projectBase, "worktree", "prune").Run()
	}
	return nil
}

// deleteBranch deletes a branch (tries -d first, then -D)
func deleteBranch(projectBase, branchName string) error {
	// Try safe delete first
	cmd := exec.Command("git", "-C", projectBase, "branch", "-d", branchName)
	if err := cmd.Run(); err != nil {
		// Try force delete
		cmd = exec.Command("git", "-C", projectBase, "branch", "-D", branchName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("could not delete branch: %w", err)
		}
	}
	return nil
}

// completeGoalInRegistry updates REGISTRY.md: remove from Active, add to Completed
func completeGoalInRegistry(registryPath, goalID, goalTitle, project string) error {
	content, err := os.ReadFile(registryPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	today := time.Now().Format("2006-01-02")
	addedToCompleted := false

	// First pass: remove from Active Goals
	activeGoalPattern := regexp.MustCompile(fmt.Sprintf(`^\| %s \|.*\| Active \|`, regexp.QuoteMeta(goalID)))
	for _, line := range lines {
		if activeGoalPattern.MatchString(line) {
			continue // Skip this line (remove from active)
		}
		newLines = append(newLines, line)
	}

	// Second pass: add to Completed Goals section
	var finalLines []string
	for i, line := range newLines {
		finalLines = append(finalLines, line)

		// Look for Completed Goals table header
		if !addedToCompleted && strings.Contains(line, "| ID | Title | Project(s) | Completed |") {
			// Next line should be the separator
			if i+1 < len(newLines) && strings.Contains(newLines[i+1], "|---") {
				finalLines = append(finalLines, newLines[i+1])
				// Add our completed goal entry
				completedRow := fmt.Sprintf("| %s | %s | %s | %s |", goalID, goalTitle, project, today)
				finalLines = append(finalLines, completedRow)
				addedToCompleted = true
				// Skip the separator in the main loop since we already added it
				newLines = append(newLines[:i+1], newLines[i+2:]...)
			}
		}
	}

	if !addedToCompleted {
		finalLines = newLines // Fall back to just removing from active
	}

	return os.WriteFile(registryPath, []byte(strings.Join(finalLines, "\n")), 0644)
}

// completeGoalInProjectConfig updates project config: remove from Active, add to Completed
func completeGoalInProjectConfig(configPath, goalID, goalTitle string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	today := time.Now().Format("2006-01-02")
	addedToCompleted := false

	// Pattern to match active goal entries: "- <id>: <title>" or "- #<id>: <title>"
	activePattern := regexp.MustCompile(fmt.Sprintf(`^- #?%s:`, regexp.QuoteMeta(goalID)))

	for i, line := range lines {
		// Skip active goal entries for this goal
		if activePattern.MatchString(line) {
			continue
		}
		newLines = append(newLines, line)

		// Look for Completed Goals table header in project config
		if !addedToCompleted && strings.Contains(line, "| ID | Title | Completed |") {
			// Next line should be the separator
			if i+1 < len(lines) && strings.Contains(lines[i+1], "|---") {
				newLines = append(newLines, lines[i+1])
				// Add our completed goal entry
				completedRow := fmt.Sprintf("| %s | %s | %s |", goalID, goalTitle, today)
				newLines = append(newLines, completedRow)
				addedToCompleted = true
				// We need to handle this carefully in the iteration
			}
		}
	}

	return os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0644)
}
