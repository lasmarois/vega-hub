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

	"github.com/google/uuid"
	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/lasmarois/vega-hub/internal/hub"
	"github.com/spf13/cobra"
)

// stateManager is initialized during create for state transitions
var stateManager *goals.StateManager

var (
	createBaseBranch    string
	createNoWorktree    bool
	createSkipPreflight bool
	createParent        string
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
	ParentID     string `json:"parent_id,omitempty"`
}

var createCmd = &cobra.Command{
	Use:   "create <title> <project>",
	Short: "Create a new goal with worktree",
	Long: `Create a goal file, add to registry, and set up a worktree.

Examples:
  vega-hub goal create "Add user authentication" my-api
  vega-hub goal create "Fix login bug" my-api --base-branch dev
  vega-hub goal create "Research caching" my-api --no-worktree
  vega-hub goal create "Design API" my-api --parent abc123  # Create child goal

The goal ID is a 7-character hash generated from a UUID.
For child goals (--parent), the ID is hierarchical: parent-id.N (e.g., abc123.1)
The worktree is created at workspaces/<project>/goal-<id>-<slug>/

Hierarchy:
  Goals can have parent-child relationships up to 3 levels deep.
  Child goals inherit the project from their parent by default.`,
	Args: cobra.RangeArgs(1, 2),
	Run:  runCreate,
}

func init() {
	GoalCmd.AddCommand(createCmd)
	createCmd.Flags().StringVar(&createBaseBranch, "base-branch", "", "Base branch (default: from project config)")
	createCmd.Flags().BoolVar(&createNoWorktree, "no-worktree", false, "Create goal file and registry only, no worktree")
	createCmd.Flags().BoolVar(&createSkipPreflight, "skip-preflight", false, "Skip pre-flight checks (escape hatch)")
	createCmd.Flags().StringVar(&createParent, "parent", "", "Parent goal ID to create this as a child (hierarchical goal)")
}

func runCreate(c *cobra.Command, args []string) {
	title := args[0]
	
	// Project can be inferred from parent or must be provided
	var project string
	if len(args) > 1 {
		project = args[1]
	}

	// Get vega-missile directory
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	// Handle parent goal (hierarchical creation)
	var parentDetail *goals.GoalDetail
	if createParent != "" {
		parser := goals.NewParser(vegaDir)
		hm := goals.NewHierarchyManager(vegaDir)
		
		// Validate parent can have children
		if err := hm.ValidateParentForChildCreation(createParent); err != nil {
			cli.OutputError(cli.ExitValidationError, "invalid_parent", err.Error(),
				map[string]string{"parent_id": createParent}, nil)
		}
		
		// Get parent details for project inheritance
		parentDetail, err = parser.ParseGoalDetail(createParent)
		if err != nil {
			cli.OutputError(cli.ExitNotFound, "parent_not_found",
				fmt.Sprintf("Parent goal '%s' not found", createParent),
				map[string]string{"parent_id": createParent}, nil)
		}
		
		// Inherit project from parent if not specified
		if project == "" {
			if len(parentDetail.Projects) > 0 {
				project = parentDetail.Projects[0]
			} else {
				cli.OutputError(cli.ExitValidationError, "no_project",
					"No project specified and parent has no projects",
					map[string]string{"parent_id": createParent}, nil)
			}
		}
	}

	// Project is required at this point
	if project == "" {
		cli.OutputError(cli.ExitValidationError, "project_required",
			"Project is required (or use --parent to inherit from parent goal)",
			nil, nil)
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
	
	_ = parentDetail // Used later for setting up hierarchy

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

	// Generate goal ID
	// For child goals: hierarchical ID (parent-id.N)
	// For root goals: 7-char hash from UUID
	var goalID string
	hm := goals.NewHierarchyManager(vegaDir)
	
	if createParent != "" {
		// Generate hierarchical child ID
		goalID, err = hm.GenerateChildID(createParent)
		if err != nil {
			cli.OutputError(cli.ExitInternalError, "id_generation_failed",
				"Failed to generate hierarchical goal ID",
				map[string]string{"error": err.Error(), "parent_id": createParent},
				nil)
		}
	} else {
		// Generate unique root goal ID (7-char hash from UUID)
		// Uses UUID v4 for randomness, truncated to 7 chars for readability
		// Collision probability is negligible for typical goal counts (<1000)
		goalID, err = generateUniqueGoalID(vegaDir)
		if err != nil {
			cli.OutputError(cli.ExitInternalError, "id_generation_failed",
				"Failed to generate unique goal ID",
				map[string]string{"error": err.Error()},
				nil)
		}
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

	// Create goal folder first (needed for state file)
	goalDir := filepath.Join(vegaDir, "goals", "active", goalID)
	if err := os.MkdirAll(goalDir, 0755); err != nil {
		cli.OutputError(cli.ExitInternalError, "goal_dir_failed",
			"Failed to create goal directory",
			map[string]string{
				"path":  goalDir,
				"error": err.Error(),
			},
			nil)
	}
	rollback = append(rollback, func() { os.RemoveAll(goalDir) })

	// Initialize state manager and transition to pending state
	stateManager = goals.NewStateManager(vegaDir)
	if err := stateManager.Transition(goalID, goals.StatePending, "Goal created", map[string]string{
		"title":   title,
		"project": project,
	}); err != nil {
		doRollback()
		cli.OutputError(cli.ExitInternalError, "state_transition_failed",
			"Failed to initialize goal state",
			map[string]string{"error": err.Error()},
			nil)
	}
	
	goalFile := filepath.Join(goalDir, goalID+".md")
	if err := createGoalFile(goalFile, goalID, title, project); err != nil {
		doRollback()
		cli.OutputError(cli.ExitInternalError, "goal_file_failed",
			"Failed to create goal file",
			map[string]string{
				"path":  goalFile,
				"error": err.Error(),
			},
			nil)
	}

	// Update registry.jsonl
	if err := addGoalToRegistry(vegaDir, goalID, title, project); err != nil {
		doRollback()
		cli.OutputError(cli.ExitInternalError, "registry_update_failed",
			"Failed to update registry",
			map[string]string{
				"error": err.Error(),
			},
			nil)
	}
	rollback = append(rollback, func() { removeGoalFromRegistry(vegaDir, goalID) })

	// Create worktree (unless --no-worktree)
	var worktreePath string
	if !createNoWorktree {
		// Run pre-flight checks (unless --skip-preflight)
		if !createSkipPreflight {
			checker := hub.NewPreflightChecker(projectBase, baseBranch, goalBranch)
			preflightResult := checker.RunAll()

			if !preflightResult.Ready {
				// Transition to failed state
				stateManager.Transition(goalID, goals.StateFailed, "Pre-flight checks failed", map[string]string{
					"blocking_issues": strings.Join(preflightResult.BlockingIssues, ", "),
				})
				doRollback()

				// Build details map for error output
				details := map[string]string{
					"blocking_issues": strings.Join(preflightResult.BlockingIssues, ", "),
				}
				for i, cmd := range preflightResult.FixCommands {
					details[fmt.Sprintf("fix_%d", i+1)] = cmd
				}
				for name, check := range preflightResult.Checks {
					if !check.Passed {
						details[name] = check.Error
					}
				}

				cli.OutputError(cli.ExitValidationError, "preflight_failed",
					"Pre-flight checks failed",
					details,
					[]cli.ErrorOption{
						{Flag: "skip-preflight", Description: "Skip pre-flight checks (escape hatch)"},
					})
			}
		}

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

		// Acquire locks for branch creation (worktree-base + branch)
		lockManager := hub.NewLockManager(vegaDir)
		
		// Lock worktree-base for branch creation
		worktreeLock, err := lockManager.AcquireWorktreeBase(project, "goal-create-"+goalID)
		if err != nil {
			stateManager.Transition(goalID, goals.StateFailed, "Failed to acquire worktree lock", map[string]string{
				"error": err.Error(),
			})
			doRollback()
			cli.OutputError(cli.ExitStateError, "lock_failed",
				"Failed to acquire worktree lock",
				map[string]string{
					"project": project,
					"error":   err.Error(),
				},
				[]cli.ErrorOption{
					{Action: "check", Description: "Run 'vega-hub lock list' to see active locks"},
					{Action: "release", Description: "Run 'vega-hub lock release --resource <name> --type worktree-base --force' if lock is stale"},
				})
		}
		defer worktreeLock.Release()

		// Also lock branch creation for this specific goal
		branchLock, err := lockManager.AcquireBranch(project, goalID, "goal-create-"+goalID)
		if err != nil {
			stateManager.Transition(goalID, goals.StateFailed, "Failed to acquire branch lock", map[string]string{
				"error": err.Error(),
			})
			doRollback()
			cli.OutputError(cli.ExitStateError, "lock_failed",
				"Failed to acquire branch lock",
				map[string]string{
					"project": project,
					"goal_id": goalID,
					"error":   err.Error(),
				},
				[]cli.ErrorOption{
					{Action: "check", Description: "Run 'vega-hub lock list' to see active locks"},
				})
		}
		defer branchLock.Release()

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

	// Set up hierarchy relationship if this is a child goal
	if createParent != "" {
		if err := hm.CreateChildGoal(goalID, createParent); err != nil {
			cli.Warn("Failed to set up hierarchy relationship: %v", err)
		}
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
		ParentID:     createParent,
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
		if createParent != "" {
			fmt.Printf("\n  Parent:      %s\n", createParent)
		}
		fmt.Printf("  Project:     %s\n", project)
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

// addGoalToRegistry adds a new goal to the JSONL registry
func addGoalToRegistry(vegaDir, id, title, project string) error {
	registry := goals.NewRegistry(vegaDir)
	now := time.Now().Format(time.RFC3339)
	return registry.Add(goals.RegistryEntry{
		ID:        id,
		Title:     title,
		Projects:  []string{project},
		Status:    "active",
		Phase:     "1/?",
		CreatedAt: now,
		UpdatedAt: now,
	})
}

// removeGoalFromRegistry removes a goal from the registry (for rollback)
func removeGoalFromRegistry(vegaDir, id string) error {
	registry := goals.NewRegistry(vegaDir)
	return registry.Delete(id)
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
