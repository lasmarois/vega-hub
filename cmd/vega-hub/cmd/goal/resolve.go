package goal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/lasmarois/vega-hub/internal/hub"
	"github.com/spf13/cobra"
)

var (
	resolveAbort   bool
	resolveMessage string
)

var resolveCmd = &cobra.Command{
	Use:   "resolve-conflict <goal-id> <project>",
	Short: "Resolve a merge conflict and continue",
	Long: `After manually resolving conflicts in the worktree, use this command to mark them as resolved and continue.

Use --abort to abort the merge instead of resolving.

Steps to resolve:
1. cd into the goal's worktree
2. Edit conflicting files, removing conflict markers
3. Run: vega-hub goal resolve-conflict <goal-id> <project>

The command will:
- Verify all conflicts are resolved
- Stage the resolved files
- Complete the merge commit
- Update goal state from CONFLICT to WORKING`,
	Args: cobra.ExactArgs(2),
	RunE: runResolve,
}

func init() {
	GoalCmd.AddCommand(resolveCmd)
	resolveCmd.Flags().BoolVar(&resolveAbort, "abort", false, "Abort the merge instead of resolving")
	resolveCmd.Flags().StringVarP(&resolveMessage, "message", "m", "", "Commit message for merge (default: auto-generated)")
}

func runResolve(cmd *cobra.Command, args []string) error {
	goalID := args[0]
	project := args[1]

	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		return err
	}

	// Find worktree path
	workspacesDir := filepath.Join(vegaDir, "workspaces", project)
	entries, err := os.ReadDir(workspacesDir)
	if err != nil {
		cli.OutputError(cli.ExitNotFound, "project_not_found",
			fmt.Sprintf("Project '%s' not found", project),
			map[string]string{"project": project},
			nil)
		return nil
	}

	var worktreePath string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), fmt.Sprintf("goal-%s-", goalID)) {
			worktreePath = filepath.Join(workspacesDir, entry.Name())
			break
		}
	}

	if worktreePath == "" {
		cli.OutputError(cli.ExitNotFound, "worktree_not_found",
			fmt.Sprintf("Worktree not found for goal %s", goalID),
			map[string]string{"goal_id": goalID, "project": project},
			nil)
		return nil
	}

	checker := hub.NewConflictChecker(worktreePath)
	stateManager := goals.NewStateManager(vegaDir)

	if resolveAbort {
		// Abort the merge
		if err := checker.AbortMerge(); err != nil {
			cli.OutputError(cli.ExitStateError, "abort_failed",
				"Failed to abort merge",
				map[string]string{"error": err.Error()},
				nil)
			return nil
		}

		// Update state back to working
		stateManager.Transition(goalID, goals.StateWorking, "Merge aborted by user", nil)

		cli.Output(cli.Result{
			Success: true,
			Action:  "resolve_conflict",
			Message: fmt.Sprintf("Aborted merge for goal %s", goalID),
		})
		return nil
	}

	// Check if there are still conflicts
	if checker.IsConflicted() {
		details, _ := checker.DetectConflicts()
		files := []string{}
		if details != nil {
			files = details.ConflictingFiles
		}

		cli.OutputError(cli.ExitValidationError, "unresolved_conflicts",
			"There are still unresolved conflicts",
			map[string]string{
				"conflicting_files": strings.Join(files, ", "),
			},
			[]cli.ErrorOption{
				{Action: "edit", Description: "Edit the files to resolve conflicts manually"},
				{Flag: "abort", Description: "Abort the merge instead"},
			})
		return nil
	}

	// Mark all files as resolved
	if err := checker.MarkResolved(); err != nil {
		cli.OutputError(cli.ExitStateError, "mark_resolved_failed",
			"Failed to mark files as resolved",
			map[string]string{"error": err.Error()},
			nil)
		return nil
	}

	// Generate commit message if not provided
	message := resolveMessage
	if message == "" {
		message = fmt.Sprintf("Resolve merge conflicts for goal %s", goalID)
	}

	// Complete the merge
	if err := checker.ContinueMerge(message); err != nil {
		cli.OutputError(cli.ExitStateError, "merge_failed",
			"Failed to complete merge",
			map[string]string{"error": err.Error()},
			nil)
		return nil
	}

	// Update state from CONFLICT to WORKING (or MERGING if we're in completion flow)
	currentState, _ := stateManager.GetState(goalID)
	if currentState == goals.StateConflict {
		stateManager.Transition(goalID, goals.StateWorking, "Conflicts resolved", nil)
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "resolve_conflict",
		Message: fmt.Sprintf("Resolved conflicts for goal %s", goalID),
		Data: map[string]string{
			"goal_id":        goalID,
			"project":        project,
			"commit_message": message,
		},
	})

	return nil
}
