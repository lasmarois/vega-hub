package worktree

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
	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/spf13/cobra"
)

var (
	createBaseBranch string
)

// CreateResult contains the result of creating a worktree
type CreateResult struct {
	GoalID       string `json:"goal_id"`
	Project      string `json:"project"`
	Path         string `json:"path"`
	Branch       string `json:"branch"`
	BaseBranch   string `json:"base_branch"`
}

var createCmd = &cobra.Command{
	Use:   "create <goal-id>",
	Short: "Create a worktree for an existing goal",
	Long: `Create a git worktree for a goal that doesn't have one.

Examples:
  vega-hub worktree create f3a8b2c
  vega-hub worktree create f3a8b2c --base-branch dev

This is useful for:
  - Resuming work on an iced goal
  - Creating a worktree after goal was created with --no-worktree
  - Re-creating a worktree that was removed

The goal must already exist in the registry.`,
	Args: cobra.ExactArgs(1),
	Run:  runCreate,
}

func init() {
	WorktreeCmd.AddCommand(createCmd)
	createCmd.Flags().StringVar(&createBaseBranch, "base-branch", "", "Base branch (default: from project config)")
}

func runCreate(c *cobra.Command, args []string) {
	goalID := args[0]

	// Get vega-missile directory
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	// Check if worktree already exists
	existingPath, _, _ := findWorktreeForGoal(vegaDir, goalID)
	if existingPath != "" {
		cli.OutputError(cli.ExitConflict, "worktree_exists",
			fmt.Sprintf("Worktree already exists for goal %s", goalID),
			map[string]string{
				"goal_id": goalID,
				"path":    existingPath,
			},
			[]cli.ErrorOption{
				{Action: "status", Description: fmt.Sprintf("Run: vega-hub worktree status %s", goalID)},
			})
	}

	// Parse registry to find the goal and its project
	parser := goals.NewParser(vegaDir)
	allGoals, err := parser.ParseRegistry()
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "registry_parse_failed",
			"Failed to parse goal registry",
			map[string]string{"error": err.Error()},
			nil)
	}

	// Find the goal
	var goal *goals.Goal
	for i := range allGoals {
		if allGoals[i].ID == goalID {
			goal = &allGoals[i]
			break
		}
	}

	if goal == nil {
		cli.OutputError(cli.ExitNotFound, "goal_not_found",
			fmt.Sprintf("Goal %s not found in registry", goalID),
			map[string]string{"goal_id": goalID},
			[]cli.ErrorOption{
				{Action: "list", Description: "Run: vega-hub goal list"},
			})
	}

	if len(goal.Projects) == 0 {
		cli.OutputError(cli.ExitStateError, "no_project",
			fmt.Sprintf("Goal %s has no associated project", goalID),
			map[string]string{"goal_id": goalID},
			nil)
	}

	project := goal.Projects[0]

	// Validate project exists
	projectBase := filepath.Join(vegaDir, "workspaces", project, "worktree-base")
	if _, err := os.Stat(projectBase); os.IsNotExist(err) {
		cli.OutputError(cli.ExitNotFound, "project_not_found",
			fmt.Sprintf("Project '%s' not found", project),
			map[string]string{
				"project": project,
				"path":    projectBase,
			},
			nil)
	}

	// Get base branch
	baseBranch := createBaseBranch
	if baseBranch == "" {
		proj, err := parser.ParseProject(project)
		if err != nil {
			cli.OutputError(cli.ExitStateError, "project_config_failed",
				fmt.Sprintf("Failed to read project config for '%s'", project),
				map[string]string{"error": err.Error()},
				[]cli.ErrorOption{
					{Flag: "base-branch", Description: "Specify base branch explicitly"},
				})
		}
		baseBranch = proj.BaseBranch
		if baseBranch == "" {
			baseBranch = "main" // fallback
		}
	}

	// Verify base branch exists
	if err := verifyBranchExists(projectBase, baseBranch); err != nil {
		cli.OutputError(cli.ExitStateError, "branch_not_found",
			fmt.Sprintf("Base branch '%s' not found in project '%s'", baseBranch, project),
			map[string]string{
				"branch":  baseBranch,
				"project": project,
			},
			[]cli.ErrorOption{
				{Action: "fetch", Description: fmt.Sprintf("Run: git -C %s fetch --all", projectBase)},
				{Flag: "base-branch", Description: "Specify a different base branch"},
			})
	}

	// Create branch name from goal title
	slug := slugify(goal.Title)
	goalBranch := fmt.Sprintf("goal-%s-%s", goalID, slug)

	// Check if branch already exists (from previous worktree)
	branchExists := verifyBranchExists(projectBase, goalBranch) == nil

	// Create worktree path
	worktreePath := filepath.Join(vegaDir, "workspaces", project, fmt.Sprintf("goal-%s-%s", goalID, slug))

	// Create the worktree
	var cmd *exec.Cmd
	if branchExists {
		// Attach to existing branch
		cmd = exec.Command("git", "-C", projectBase, "worktree", "add", worktreePath, goalBranch)
	} else {
		// Create new branch from base
		cmd = exec.Command("git", "-C", projectBase, "worktree", "add", worktreePath, "-b", goalBranch, baseBranch)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		cli.OutputError(cli.ExitStateError, "worktree_create_failed",
			"Failed to create worktree",
			map[string]string{
				"path":   worktreePath,
				"branch": goalBranch,
				"error":  fmt.Sprintf("%s: %s", err, string(output)),
			},
			[]cli.ErrorOption{
				{Action: "check", Description: "Verify git status in project base"},
			})
	}

	// Copy hooks and rules to worktree
	if err := setupWorktreeEnvironment(vegaDir, worktreePath); err != nil {
		cli.Warn("Failed to copy hooks to worktree: %v", err)
	}

	// Write worktree metadata to goal file
	if err := writeWorktreeToGoalFile(vegaDir, goalID, project, worktreePath, goalBranch, baseBranch); err != nil {
		cli.Warn("Failed to write worktree metadata to goal file: %v", err)
	}

	// Success output
	result := CreateResult{
		GoalID:     goalID,
		Project:    project,
		Path:       worktreePath,
		Branch:     goalBranch,
		BaseBranch: baseBranch,
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "worktree_create",
		Message: fmt.Sprintf("Created worktree for goal %s", goalID),
		Data:    result,
		NextSteps: []string{
			fmt.Sprintf("Spawn executor: vega-hub executor spawn %s", goalID),
		},
	})

	// Human-readable summary
	if !cli.JSONOutput {
		fmt.Printf("\n  Path: %s\n", worktreePath)
		fmt.Printf("  Branch: %s\n", goalBranch)
		fmt.Printf("  Base: %s\n", baseBranch)
		if branchExists {
			fmt.Println("  (attached to existing branch)")
		}
	}
}

// verifyBranchExists checks if a branch exists in the git repo
func verifyBranchExists(repoPath, branch string) error {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--verify", branch)
	if err := cmd.Run(); err != nil {
		// Try with remote prefix
		cmd = exec.Command("git", "-C", repoPath, "rev-parse", "--verify", "origin/"+branch)
		return cmd.Run()
	}
	return nil
}

// slugify converts a title to a branch-safe slug
func slugify(title string) string {
	slug := strings.ToLower(title)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug = re.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	// Limit length to 30 chars
	if len(slug) > 30 {
		slug = slug[:30]
		slug = strings.TrimRight(slug, "-")
	}
	return slug
}

// setupWorktreeEnvironment copies hooks and rules to the worktree
func setupWorktreeEnvironment(vegaDir, worktreePath string) error {
	templateDir := filepath.Join(vegaDir, "templates", "project-init", ".claude")
	destDir := filepath.Join(worktreePath, ".claude")

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Copy hooks directory
	hooksSource := filepath.Join(templateDir, "hooks")
	hooksDest := filepath.Join(destDir, "hooks")
	if err := copyDir(hooksSource, hooksDest); err != nil {
		return fmt.Errorf("copying hooks: %w", err)
	}

	// Copy settings.local.json if exists
	settingsSource := filepath.Join(templateDir, "settings.local.json")
	settingsDest := filepath.Join(destDir, "settings.local.json")
	if _, err := os.Stat(settingsSource); err == nil {
		if err := copyFile(settingsSource, settingsDest); err != nil {
			return fmt.Errorf("copying settings: %w", err)
		}
	}

	// Copy rules/vega-missile/ if exists
	rulesSource := filepath.Join(templateDir, "rules", "vega-missile")
	rulesDest := filepath.Join(destDir, "rules", "vega-missile")
	if _, err := os.Stat(rulesSource); err == nil {
		if err := copyDir(rulesSource, rulesDest); err != nil {
			return fmt.Errorf("copying rules: %w", err)
		}
	}

	return nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, content, info.Mode())
}

// writeWorktreeToGoalFile appends or updates the Worktree section in the goal markdown file
func writeWorktreeToGoalFile(vegaDir, goalID, project, worktreePath, branch, baseBranch string) error {
	// Find goal file - check active, then iced
	goalPath := filepath.Join(vegaDir, "goals", "active", goalID+".md")
	if _, err := os.Stat(goalPath); os.IsNotExist(err) {
		goalPath = filepath.Join(vegaDir, "goals", "iced", goalID+".md")
		if _, err := os.Stat(goalPath); os.IsNotExist(err) {
			return fmt.Errorf("goal file not found for %s", goalID)
		}
	}

	// Read the current file
	content, err := os.ReadFile(goalPath)
	if err != nil {
		return fmt.Errorf("failed to read goal file: %w", err)
	}

	// Calculate relative path from vega-missile root
	relativePath := strings.TrimPrefix(worktreePath, vegaDir+"/")

	// Build the worktree section
	today := time.Now().Format("2006-01-02")
	worktreeSection := fmt.Sprintf(`## Worktree
- **Branch**: %s
- **Project**: %s
- **Path**: %s
- **Base Branch**: %s
- **Created**: %s
`, branch, project, relativePath, baseBranch, today)

	// Check if there's already a Worktree section
	lines := strings.Split(string(content), "\n")
	var newLines []string
	inWorktreeSection := false
	worktreeSectionFound := false

	for _, line := range lines {
		if strings.HasPrefix(line, "## Worktree") {
			// Replace existing worktree section
			inWorktreeSection = true
			worktreeSectionFound = true
			// Add the new worktree section (without trailing newline, we'll add lines)
			newLines = append(newLines, strings.TrimSuffix(worktreeSection, "\n"))
			continue
		}

		if inWorktreeSection {
			// Skip until we hit the next ## section or end
			if strings.HasPrefix(line, "## ") {
				inWorktreeSection = false
				newLines = append(newLines, line)
			}
			// Skip lines within the old worktree section
			continue
		}

		newLines = append(newLines, line)
	}

	// If no worktree section found, append it before ## Status or at the end
	if !worktreeSectionFound {
		newLines = insertWorktreeSection(lines, worktreeSection)
	}

	// Write back the file
	newContent := strings.Join(newLines, "\n")
	if err := os.WriteFile(goalPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write goal file: %w", err)
	}

	return nil
}

// insertWorktreeSection inserts the worktree section in the appropriate place
func insertWorktreeSection(lines []string, worktreeSection string) []string {
	var result []string
	inserted := false

	// Try to insert before "## Status" or "## Executor Activity"
	for i, line := range lines {
		if !inserted && (strings.HasPrefix(line, "## Status") || strings.HasPrefix(line, "## Executor Activity")) {
			// Insert worktree section before this line
			result = append(result, "")
			result = append(result, strings.Split(strings.TrimSuffix(worktreeSection, "\n"), "\n")...)
			result = append(result, "")
			inserted = true
		}
		result = append(result, line)

		// If we're at the end and haven't inserted yet
		if i == len(lines)-1 && !inserted {
			result = append(result, "")
			result = append(result, strings.Split(strings.TrimSuffix(worktreeSection, "\n"), "\n")...)
		}
	}

	return result
}

// removeWorktreeFromGoalFile removes the Worktree section from the goal file
// This can be called when removing a worktree to clear the metadata
func removeWorktreeFromGoalFile(vegaDir, goalID string) error {
	// Find goal file - check active, then iced, then history
	goalPath := filepath.Join(vegaDir, "goals", "active", goalID+".md")
	if _, err := os.Stat(goalPath); os.IsNotExist(err) {
		goalPath = filepath.Join(vegaDir, "goals", "iced", goalID+".md")
		if _, err := os.Stat(goalPath); os.IsNotExist(err) {
			goalPath = filepath.Join(vegaDir, "goals", "history", goalID+".md")
			if _, err := os.Stat(goalPath); os.IsNotExist(err) {
				return nil // Goal file doesn't exist, nothing to do
			}
		}
	}

	file, err := os.Open(goalPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var newLines []string
	scanner := bufio.NewScanner(file)
	inWorktreeSection := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "## Worktree") {
			inWorktreeSection = true
			continue
		}

		if inWorktreeSection {
			if strings.HasPrefix(line, "## ") {
				inWorktreeSection = false
				newLines = append(newLines, line)
			}
			// Skip lines within worktree section
			continue
		}

		newLines = append(newLines, line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Write back
	newContent := strings.Join(newLines, "\n")
	return os.WriteFile(goalPath, []byte(newContent), 0644)
}
