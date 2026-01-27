package goal

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/spf13/cobra"
)

var (
	iceForce bool
)

// IceResult contains the result of icing a goal
type IceResult struct {
	GoalID          string `json:"goal_id"`
	Title           string `json:"title"`
	Project         string `json:"project"`
	Reason          string `json:"reason"`
	BranchPreserved string `json:"branch_preserved"`
	WorktreeRemoved bool   `json:"worktree_removed"`
	ResumeCommand   string `json:"resume_command"`
}

var iceCmd = &cobra.Command{
	Use:   "ice <goal-id> <project> <reason>",
	Short: "Ice (pause) a goal for later",
	Long: `Ice a goal: remove worktree but preserve branch for later resumption.

Examples:
  vega-hub goal ice f3a8b2c my-api "Blocked on API design"
  vega-hub goal ice f3a8b2c my-api "Deprioritized for Q2"
  vega-hub goal ice f3a8b2c my-api "Paused" --force

This command will:
  1. Remove the worktree directory (branch is preserved)
  2. Update the goal file status to "Iced"
  3. Update REGISTRY.md (Active -> Iced with reason)
  4. Update the project config

NOTE: Executor should archive planning files before running this command.

To resume an iced goal later:
  cd workspaces/<project>/worktree-base
  git worktree add ../goal-<id>-<slug> goal-<id>-<slug>
  # Then update registry status back to Active`,
	Args: cobra.ExactArgs(3),
	Run:  runIce,
}

func init() {
	GoalCmd.AddCommand(iceCmd)
	iceCmd.Flags().BoolVarP(&iceForce, "force", "f", false, "Skip safety checks for uncommitted changes")
}

func runIce(c *cobra.Command, args []string) {
	goalID := args[0]
	project := args[1]
	reason := args[2]

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

	// Find the worktree directory
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
	if !iceForce {
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

	cli.Info("Icing goal %s: %s", goalID, goalTitle)
	cli.Info("  Project: %s", project)
	cli.Info("  Reason: %s", reason)
	cli.Info("  Worktree: %s", worktreeDir)

	// Initialize state manager
	sm := goals.NewStateManager(vegaDir)

	result := IceResult{
		GoalID:          goalID,
		Title:           goalTitle,
		Project:         project,
		Reason:          reason,
		BranchPreserved: branchName,
	}

	// Step 1: Remove worktree (keep branch)
	cli.Info("Removing worktree (keeping branch)...")
	if err := removeWorktreeComplete(projectBase, worktreeDir); err != nil {
		cli.Warn("Could not remove worktree cleanly: %v", err)
	}
	result.WorktreeRemoved = true

	// Step 2: Update goal file status
	cli.Info("Updating goal file status...")
	if err := updateGoalFileToIced(goalFile, reason); err != nil {
		cli.Warn("Could not update goal file: %v", err)
	}

	// Step 3: Update registry (Active -> Iced)
	registryPath := filepath.Join(vegaDir, "goals", "REGISTRY.md")
	if err := iceGoalInRegistry(registryPath, goalID, goalTitle, project, reason); err != nil {
		cli.Warn("Could not update registry: %v", err)
	}

	// Step 4: Update project config
	projectConfig := filepath.Join(vegaDir, "projects", project+".md")
	if err := removeGoalFromProjectConfig(projectConfig, goalID); err != nil {
		cli.Warn("Could not update project config: %v", err)
	}

	// Step 5: Transition state to iced
	if err := sm.Transition(goalID, goals.StateIced, reason, map[string]string{
		"branch": branchName,
	}); err != nil {
		cli.Warn("Failed to transition to iced state: %v", err)
	}

	// Build resume command
	result.ResumeCommand = fmt.Sprintf("cd %s && git worktree add ../%s %s", projectBase, branchName, branchName)

	// Success output
	cli.Output(cli.Result{
		Success: true,
		Action:  "goal_ice",
		Message: fmt.Sprintf("Iced goal %s: %s", goalID, goalTitle),
		Data:    result,
		NextSteps: []string{
			fmt.Sprintf("Branch preserved: %s", branchName),
			fmt.Sprintf("To resume: %s", result.ResumeCommand),
		},
	})
}

// updateGoalFileToIced updates the goal file status to Iced and adds reason
func updateGoalFileToIced(goalFile, reason string) error {
	content, err := os.ReadFile(goalFile)
	if err != nil {
		return err
	}

	text := string(content)

	// Update status line
	statusRe := regexp.MustCompile(`\*\*Status\*\*:\s*Active`)
	text = statusRe.ReplaceAllString(text, "**Status**: Iced")

	// Add iced note at the end
	today := time.Now().Format("2006-01-02")
	icedNote := fmt.Sprintf("\n### Iced\n\n**Date**: %s\n**Reason**: %s\n", today, reason)
	text = strings.TrimRight(text, "\n") + "\n" + icedNote

	return os.WriteFile(goalFile, []byte(text), 0644)
}

// iceGoalInRegistry updates REGISTRY.md: remove from Active, add to Iced
func iceGoalInRegistry(registryPath, goalID, goalTitle, project, reason string) error {
	content, err := os.ReadFile(registryPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	addedToIced := false

	// First pass: remove from Active Goals
	activeGoalPattern := regexp.MustCompile(fmt.Sprintf(`^\| %s \|.*\| Active \|`, regexp.QuoteMeta(goalID)))
	for _, line := range lines {
		if activeGoalPattern.MatchString(line) {
			continue // Skip this line (remove from active)
		}
		newLines = append(newLines, line)
	}

	// Second pass: add to Iced Goals section
	var finalLines []string
	for i, line := range newLines {
		finalLines = append(finalLines, line)

		// Look for Iced Goals table header
		if !addedToIced && strings.Contains(line, "| ID | Title | Project(s) | Reason |") {
			// Next line should be the separator
			if i+1 < len(newLines) && strings.Contains(newLines[i+1], "|---") {
				finalLines = append(finalLines, newLines[i+1])
				// Add our iced goal entry
				icedRow := fmt.Sprintf("| %s | %s | %s | %s |", goalID, goalTitle, project, reason)
				finalLines = append(finalLines, icedRow)
				addedToIced = true
				// Skip the separator in the main loop since we already added it
				newLines = append(newLines[:i+1], newLines[i+2:]...)
			}
		}
	}

	if !addedToIced {
		finalLines = newLines // Fall back to just removing from active
	}

	return os.WriteFile(registryPath, []byte(strings.Join(finalLines, "\n")), 0644)
}

// removeGoalFromProjectConfig removes a goal from the project's Active Goals section
func removeGoalFromProjectConfig(configPath, goalID string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string

	// Pattern to match active goal entries: "- <id>: <title>" or "- #<id>: <title>"
	activePattern := regexp.MustCompile(fmt.Sprintf(`^- #?%s:`, regexp.QuoteMeta(goalID)))

	for _, line := range lines {
		if activePattern.MatchString(line) {
			continue // Skip this line
		}
		newLines = append(newLines, line)
	}

	return os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0644)
}
