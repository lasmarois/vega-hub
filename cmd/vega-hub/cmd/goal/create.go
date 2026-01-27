package goal

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/spf13/cobra"
)

// stateManager is initialized during create for state transitions
var stateManager *goals.StateManager

var (
	createBaseBranch string
	createNoWorktree bool
)

// CreateResult contains the result of creating a goal
type CreateResult struct {
	GoalID       string `json:"goal_id"`
	Title        string `json:"title"`
	Project      string `json:"project"`
	BaseBranch   string `json:"base_branch"`
	GoalBranch   string `json:"goal_branch"`
	WorktreePath string `json:"worktree_path,omitempty"`
	GoalFile     string `json:"goal_file"`
}

var createCmd = &cobra.Command{
	Use:   "create <title> <project>",
	Short: "Create a new goal with worktree",
	Long: `Create a goal file, add to registry, and set up a worktree.

Examples:
  vega-hub goal create "Add user authentication" my-api
  vega-hub goal create "Fix login bug" my-api --base-branch dev
  vega-hub goal create "Research caching" my-api --no-worktree

The goal ID is a 7-character hash generated from a UUID.
The worktree is created at workspaces/<project>/goal-<id>-<slug>/`,
	Args: cobra.ExactArgs(2),
	Run:  runCreate,
}

func init() {
	GoalCmd.AddCommand(createCmd)
	createCmd.Flags().StringVar(&createBaseBranch, "base-branch", "", "Base branch (default: from project config)")
	createCmd.Flags().BoolVar(&createNoWorktree, "no-worktree", false, "Create goal file and registry only, no worktree")
}

func runCreate(c *cobra.Command, args []string) {
	title := args[0]
	project := args[1]

	// Get vega-missile directory
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	// Validate project exists
	projectBase := filepath.Join(vegaDir, "workspaces", project, "worktree-base")
	if _, err := os.Stat(projectBase); os.IsNotExist(err) {
		cli.OutputError(cli.ExitNotFound, "project_not_found",
			fmt.Sprintf("Project '%s' not found", project),
			map[string]string{
				"expected_path": projectBase,
				"project":       project,
			},
			[]cli.ErrorOption{
				{Action: "onboard", Description: fmt.Sprintf("Run: scripts/onboard-project.sh %s <git-url>", project)},
			})
	}

	// Get base branch from flag or project config
	baseBranch := createBaseBranch
	if baseBranch == "" {
		baseBranch, err = getProjectBaseBranch(vegaDir, project)
		if err != nil {
			cli.OutputError(cli.ExitStateError, "no_base_branch",
				fmt.Sprintf("Could not determine base branch for project '%s'", project),
				map[string]string{
					"project":     project,
					"config_path": filepath.Join(vegaDir, "projects", project+".md"),
				},
				[]cli.ErrorOption{
					{Flag: "base-branch", Description: "Specify base branch explicitly"},
				})
		}
	}

	// Verify base branch exists in project repo
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

	// Generate unique goal ID (7-char hash from UUID)
	// Uses UUID v4 for randomness, truncated to 7 chars for readability
	// Collision probability is negligible for typical goal counts (<1000)
	goalID, err := generateUniqueGoalID(vegaDir)
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "id_generation_failed",
			"Failed to generate unique goal ID",
			map[string]string{"error": err.Error()},
			nil)
	}

	// Initialize state manager for tracking goal state
	stateManager = goals.NewStateManager(vegaDir)

	// Transition to pending state (goal creation starting)
	if err := stateManager.Transition(goalID, goals.StatePending, "Goal created", map[string]string{
		"title":   title,
		"project": project,
	}); err != nil {
		cli.OutputError(cli.ExitInternalError, "state_transition_failed",
			"Failed to initialize goal state",
			map[string]string{"error": err.Error()},
			nil)
	}

	// Create slug from title for branch name
	slug := slugify(title)
	goalBranch := fmt.Sprintf("goal-%s-%s", goalID, slug)

	// Track created resources for rollback on error
	var rollback []func()
	doRollback := func() {
		for i := len(rollback) - 1; i >= 0; i-- {
			rollback[i]()
		}
	}

	// Create goal file
	goalFile := filepath.Join(vegaDir, "goals", "active", goalID+".md")
	if err := createGoalFile(goalFile, goalID, title, project); err != nil {
		cli.OutputError(cli.ExitInternalError, "goal_file_failed",
			"Failed to create goal file",
			map[string]string{
				"path":  goalFile,
				"error": err.Error(),
			},
			nil)
	}
	rollback = append(rollback, func() { os.Remove(goalFile) })

	// Update REGISTRY.md
	registryPath := filepath.Join(vegaDir, "goals", "REGISTRY.md")
	if err := addGoalToRegistry(registryPath, goalID, title, project); err != nil {
		doRollback()
		cli.OutputError(cli.ExitInternalError, "registry_update_failed",
			"Failed to update registry",
			map[string]string{
				"path":  registryPath,
				"error": err.Error(),
			},
			nil)
	}
	rollback = append(rollback, func() { removeGoalFromRegistry(registryPath, goalID) })

	// Create worktree (unless --no-worktree)
	var worktreePath string
	if !createNoWorktree {
		// Transition to branching state
		if err := stateManager.Transition(goalID, goals.StateBranching, "Creating worktree", map[string]string{
			"branch":      goalBranch,
			"base_branch": baseBranch,
		}); err != nil {
			doRollback()
			cli.OutputError(cli.ExitInternalError, "state_transition_failed",
				"Failed to transition to branching state",
				map[string]string{"error": err.Error()},
				nil)
		}

		worktreePath = filepath.Join(vegaDir, "workspaces", project, fmt.Sprintf("goal-%s-%s", goalID, slug))
		if err := createWorktree(projectBase, worktreePath, goalBranch, baseBranch); err != nil {
			// Transition to failed state
			stateManager.Transition(goalID, goals.StateFailed, "Worktree creation failed", map[string]string{
				"error": err.Error(),
			})
			doRollback()
			cli.OutputError(cli.ExitStateError, "worktree_failed",
				"Failed to create worktree",
				map[string]string{
					"path":        worktreePath,
					"branch":      goalBranch,
					"base_branch": baseBranch,
					"error":       err.Error(),
				},
				[]cli.ErrorOption{
					{Action: "check", Description: "Verify git status in project base"},
					{Flag: "no-worktree", Description: "Create goal without worktree"},
				})
		}
		rollback = append(rollback, func() { removeWorktree(projectBase, worktreePath, goalBranch) })

		// Copy hooks and rules to worktree
		if err := setupWorktreeEnvironment(vegaDir, worktreePath); err != nil {
			// Non-fatal: warn but continue
			cli.Warn("Failed to copy hooks to worktree: %v", err)
		}

		// Transition to working state - goal is ready for development
		if err := stateManager.Transition(goalID, goals.StateWorking, "Worktree ready", map[string]string{
			"worktree": worktreePath,
		}); err != nil {
			cli.Warn("Failed to transition to working state: %v", err)
		}
	} else {
		// No worktree - transition directly to working (for research/planning goals)
		if err := stateManager.Transition(goalID, goals.StateBranching, "No worktree requested", nil); err != nil {
			cli.Warn("Failed to transition to branching state: %v", err)
		}
		if err := stateManager.Transition(goalID, goals.StateWorking, "Goal ready (no worktree)", nil); err != nil {
			cli.Warn("Failed to transition to working state: %v", err)
		}
	}

	// Update project config
	projectConfig := filepath.Join(vegaDir, "projects", project+".md")
	if err := addGoalToProjectConfig(projectConfig, goalID, title); err != nil {
		// Non-fatal: warn but continue
		cli.Warn("Failed to update project config: %v", err)
	}

	// Success output
	result := CreateResult{
		GoalID:       goalID,
		Title:        title,
		Project:      project,
		BaseBranch:   baseBranch,
		GoalBranch:   goalBranch,
		WorktreePath: worktreePath,
		GoalFile:     goalFile,
	}

	nextSteps := []string{
		fmt.Sprintf("Edit goal file: %s", goalFile),
	}
	if worktreePath != "" {
		nextSteps = append(nextSteps, fmt.Sprintf("Spawn executor: vega-hub executor spawn %s", goalID))
	}

	cli.Output(cli.Result{
		Success:   true,
		Action:    "goal_create",
		Message:   fmt.Sprintf("Created goal %s: %s", goalID, title),
		Data:      result,
		NextSteps: nextSteps,
	})

	// Human-readable summary (non-JSON mode)
	if !cli.JSONOutput {
		fmt.Printf("\n  Project:     %s\n", project)
		fmt.Printf("  Base branch: %s\n", baseBranch)
		fmt.Printf("  Goal branch: %s\n", goalBranch)
		if worktreePath != "" {
			fmt.Printf("  Worktree:    %s\n", worktreePath)
		}
	}
}

// getProjectBaseBranch reads the base branch from projects/<project>.md
// Format expected: "**Base Branch**: `<branch>`" or "Base Branch: <branch>"
func getProjectBaseBranch(vegaDir, project string) (string, error) {
	configPath := filepath.Join(vegaDir, "projects", project+".md")
	file, err := os.Open(configPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Pattern matches "Base Branch: master" or "**Base Branch**: `master`"
	re := regexp.MustCompile(`(?i)(?:\*\*)?Base Branch(?:\*\*)?:?\s*` + "`?" + `([a-zA-Z0-9_/-]+)` + "`?")

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("base branch not found in %s", configPath)
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

// generateUniqueGoalID generates a 7-character hash ID and verifies uniqueness
func generateUniqueGoalID(vegaDir string) (string, error) {
	parser := goals.NewParser(vegaDir)
	existingGoals, err := parser.ParseRegistry()
	if err != nil {
		return "", err
	}

	// Build set of existing IDs
	existingIDs := make(map[string]bool)
	for _, g := range existingGoals {
		existingIDs[g.ID] = true
	}

	// Try up to 10 times to generate a unique ID
	// With 7-char hex (16^7 = 268M combinations), collision is extremely unlikely
	// But we check anyway for safety
	const maxAttempts = 10
	for i := 0; i < maxAttempts; i++ {
		id := strings.ToLower(uuid.New().String()[:7])
		if !existingIDs[id] {
			return id, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique ID after %d attempts", maxAttempts)
}

// slugify converts a title to a branch-safe slug
func slugify(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)
	// Replace spaces and special chars with dashes
	re := regexp.MustCompile(`[^a-z0-9]+`)
	slug = re.ReplaceAllString(slug, "-")
	// Remove leading/trailing dashes
	slug = strings.Trim(slug, "-")
	// Limit length to keep branch names reasonable
	// 30 chars is enough to be descriptive but not unwieldy
	if len(slug) > 30 {
		slug = slug[:30]
		// Don't end on a dash
		slug = strings.TrimRight(slug, "-")
	}
	return slug
}

// createGoalFile creates a goal markdown file from the template
func createGoalFile(path, id, title, project string) error {
	content := fmt.Sprintf(`# Goal %s: %s

## Overview

<Brief description of the goal>

## Project(s)

- **%s**: <what this project does for the goal>

## Phases

### Phase 1: <Phase Title>
- [ ] Task 1
- [ ] Task 2

### Phase 2: <Phase Title>
- [ ] Task 1
- [ ] Task 2

## Acceptance Criteria

- [ ] Criterion 1
- [ ] Criterion 2

## Status

**Current Phase**: 1
**Status**: Active
**Assigned To**: -

## Notes

<Any additional context, blockers, or decisions>
`, id, title, project)

	return os.WriteFile(path, []byte(content), 0644)
}

// addGoalToRegistry adds a new goal to the Active Goals section of REGISTRY.md
func addGoalToRegistry(path, id, title, project string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	inserted := false

	for i, line := range lines {
		newLines = append(newLines, line)

		// Find the Active Goals table header row (|---...|)
		// Insert our new goal right after it
		if !inserted && strings.Contains(line, "|----") {
			// Check if previous line is the Active Goals header
			if i > 0 && strings.Contains(lines[i-1], "| ID |") {
				// Check if we're in the Active Goals section
				for j := i - 1; j >= 0; j-- {
					if strings.Contains(lines[j], "## Active Goals") {
						// Insert new goal row
						newRow := fmt.Sprintf("| %s | %s | %s | Active | 1/? |", id, title, project)
						newLines = append(newLines, newRow)
						inserted = true
						break
					}
					if strings.HasPrefix(lines[j], "## ") {
						// Hit another section, stop looking
						break
					}
				}
			}
		}
	}

	if !inserted {
		return fmt.Errorf("could not find Active Goals table in registry")
	}

	return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
}

// removeGoalFromRegistry removes a goal from the registry (for rollback)
func removeGoalFromRegistry(path, id string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string

	for _, line := range lines {
		// Skip lines that start with this goal ID
		if strings.HasPrefix(strings.TrimSpace(line), "| "+id+" |") {
			continue
		}
		newLines = append(newLines, line)
	}

	return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
}

// createWorktree creates a git worktree for the goal
func createWorktree(projectBase, worktreePath, branch, baseBranch string) error {
	// Create worktree with new branch based on base branch
	cmd := exec.Command("git", "-C", projectBase, "worktree", "add", worktreePath, "-b", branch, baseBranch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

// removeWorktree removes a worktree and its branch (for rollback)
func removeWorktree(projectBase, worktreePath, branch string) error {
	// Remove worktree
	exec.Command("git", "-C", projectBase, "worktree", "remove", "--force", worktreePath).Run()
	// Delete branch
	exec.Command("git", "-C", projectBase, "branch", "-D", branch).Run()
	return nil
}

// setupWorktreeEnvironment copies hooks and rules to the worktree
func setupWorktreeEnvironment(vegaDir, worktreePath string) error {
	// Source: templates/project-init/.claude/
	templateDir := filepath.Join(vegaDir, "templates", "project-init", ".claude")

	// Destination: worktree/.claude/
	destDir := filepath.Join(worktreePath, ".claude")

	// Create .claude directory if it doesn't exist
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

// addGoalToProjectConfig adds a goal to the project's Active Goals section
func addGoalToProjectConfig(path, id, title string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	inActiveGoals := false

	for _, line := range lines {
		newLines = append(newLines, line)

		if strings.Contains(line, "## Active Goals") {
			inActiveGoals = true
			continue
		}

		// Insert after "## Active Goals" section header
		if inActiveGoals && !strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "(none)") && line != "" {
			// Found first non-list item after Active Goals, insert before it
			goalLine := fmt.Sprintf("- %s: %s", id, title)
			// Insert before current line
			newLines = newLines[:len(newLines)-1]
			newLines = append(newLines, goalLine)
			newLines = append(newLines, line)
			inActiveGoals = false
		} else if inActiveGoals && strings.HasPrefix(line, "(none)") {
			// Replace "(none)" with the goal
			newLines = newLines[:len(newLines)-1]
			newLines = append(newLines, fmt.Sprintf("- %s: %s", id, title))
			inActiveGoals = false
		}
	}

	return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0644)
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

	// Preserve original file permissions
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, content, info.Mode())
}
