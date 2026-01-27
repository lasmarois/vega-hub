package goal

import (
	"fmt"
	"strings"
	"time"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/spf13/cobra"
)

var (
	stateHistory bool
	setStateForce bool
)

// StateResult contains the result of querying goal state
type StateResult struct {
	GoalID  string           `json:"goal_id"`
	State   goals.GoalState  `json:"state"`
	Since   *time.Time       `json:"since,omitempty"`
	Reason  string           `json:"reason,omitempty"`
	History []goals.StateEvent `json:"history,omitempty"`
}

// SetStateResult contains the result of setting goal state
type SetStateResult struct {
	GoalID    string          `json:"goal_id"`
	PrevState goals.GoalState `json:"prev_state"`
	NewState  goals.GoalState `json:"new_state"`
	Forced    bool            `json:"forced"`
}

var stateCmd = &cobra.Command{
	Use:   "state <goal-id>",
	Short: "Show current state of a goal",
	Long: `Show the current state and optionally full state history for a goal.

Examples:
  vega-hub goal state f3a8b2c              # Show current state
  vega-hub goal state f3a8b2c --history    # Show full state history

States:
  pending    - Created, not yet branched
  branching  - Creating branch/worktree
  working    - Active development
  pushing    - Committing/pushing changes
  merging    - Merging to target branch
  done       - Successfully completed
  iced       - Paused/frozen
  failed     - Failed (recoverable)
  conflict   - Merge conflict detected`,
	Args: cobra.ExactArgs(1),
	Run:  runState,
}

var setStateCmd = &cobra.Command{
	Use:   "set-state <goal-id> <state>",
	Short: "Force-set a goal's state (escape hatch)",
	Long: `Force-set a goal's state, bypassing normal transition validation.

This is an escape hatch for recovering from stuck or inconsistent states.
Use with caution - prefer normal workflows when possible.

Examples:
  vega-hub goal set-state f3a8b2c working --force
  vega-hub goal set-state f3a8b2c pending --force

Valid states: pending, branching, working, pushing, merging, done, iced, failed, conflict`,
	Args: cobra.ExactArgs(2),
	Run:  runSetState,
}

func init() {
	GoalCmd.AddCommand(stateCmd)
	GoalCmd.AddCommand(setStateCmd)
	
	stateCmd.Flags().BoolVar(&stateHistory, "history", false, "Show full state history")
	setStateCmd.Flags().BoolVar(&setStateForce, "force", false, "Required to confirm force-setting state")
}

func runState(c *cobra.Command, args []string) {
	goalID := args[0]

	// Get vega-missile directory
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	sm := goals.NewStateManager(vegaDir)

	if stateHistory {
		// Get full history
		history, err := sm.GetHistory(goalID)
		if err != nil {
			cli.OutputError(cli.ExitNotFound, "state_not_found",
				fmt.Sprintf("Could not read state for goal '%s'", goalID),
				map[string]string{
					"goal_id": goalID,
					"error":   err.Error(),
				},
				nil)
		}

		result := StateResult{
			GoalID:  goalID,
			History: history,
		}

		if len(history) > 0 {
			lastEvent := history[len(history)-1]
			result.State = lastEvent.State
			result.Since = &lastEvent.Timestamp
			result.Reason = lastEvent.Reason
		} else {
			result.State = goals.StateWorking // Legacy goal
		}

		cli.Output(cli.Result{
			Success: true,
			Action:  "goal_state",
			Message: fmt.Sprintf("Goal %s: %s", goalID, result.State),
			Data:    result,
		})

		// Human-readable history
		if !cli.JSONOutput && len(history) > 0 {
			fmt.Println("\nState History:")
			for _, event := range history {
				ts := event.Timestamp.Local().Format("2006-01-02 15:04:05")
				if event.PrevState != "" {
					fmt.Printf("  %s: %s → %s", ts, event.PrevState, event.State)
				} else {
					fmt.Printf("  %s: %s (initial)", ts, event.State)
				}
				if event.Reason != "" {
					fmt.Printf(" - %s", event.Reason)
				}
				if event.User != "" {
					fmt.Printf(" [%s]", event.User)
				}
				fmt.Println()
			}
		}
	} else {
		// Get current state only
		state, err := sm.GetState(goalID)
		if err != nil {
			cli.OutputError(cli.ExitNotFound, "state_not_found",
				fmt.Sprintf("Could not read state for goal '%s'", goalID),
				map[string]string{
					"goal_id": goalID,
					"error":   err.Error(),
				},
				nil)
		}

		// Get last event for additional context
		lastEvent, _ := sm.GetLastEvent(goalID)

		result := StateResult{
			GoalID: goalID,
			State:  state,
		}

		if lastEvent != nil {
			result.Since = &lastEvent.Timestamp
			result.Reason = lastEvent.Reason
		}

		cli.Output(cli.Result{
			Success: true,
			Action:  "goal_state",
			Message: fmt.Sprintf("Goal %s: %s", goalID, state),
			Data:    result,
		})

		// Human-readable output
		if !cli.JSONOutput {
			fmt.Printf("\n  State: %s\n", state)
			if result.Since != nil {
				fmt.Printf("  Since: %s\n", result.Since.Local().Format("2006-01-02 15:04:05"))
			}
			if result.Reason != "" {
				fmt.Printf("  Reason: %s\n", result.Reason)
			}
			if state.NeedsAttention() {
				fmt.Println("\n  ⚠ This goal needs attention!")
			}
		}
	}
}

func runSetState(c *cobra.Command, args []string) {
	goalID := args[0]
	newStateStr := args[1]

	if !setStateForce {
		cli.OutputError(cli.ExitValidationError, "force_required",
			"--force flag is required to force-set state",
			map[string]string{
				"goal_id": goalID,
				"state":   newStateStr,
			},
			[]cli.ErrorOption{
				{Flag: "force", Description: "Confirm you want to bypass normal transition validation"},
			})
	}

	newState := goals.GoalState(newStateStr)
	if !newState.IsValid() {
		validStates := make([]string, 0)
		for _, s := range goals.AllStates() {
			validStates = append(validStates, string(s))
		}
		cli.OutputError(cli.ExitValidationError, "invalid_state",
			fmt.Sprintf("Invalid state: %s", newStateStr),
			map[string]string{
				"valid_states": strings.Join(validStates, ", "),
			},
			nil)
	}

	// Get vega-missile directory
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	sm := goals.NewStateManager(vegaDir)

	// Get current state for logging
	prevState, _ := sm.GetState(goalID)

	// Force the state
	reason := "Manual state override via CLI"
	if err := sm.ForceState(goalID, newState, reason); err != nil {
		cli.OutputError(cli.ExitInternalError, "force_state_failed",
			fmt.Sprintf("Failed to set state for goal '%s'", goalID),
			map[string]string{
				"goal_id": goalID,
				"error":   err.Error(),
			},
			nil)
	}

	result := SetStateResult{
		GoalID:    goalID,
		PrevState: prevState,
		NewState:  newState,
		Forced:    true,
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "goal_set_state",
		Message: fmt.Sprintf("Forced goal %s state: %s → %s", goalID, prevState, newState),
		Data:    result,
	})

	// Human-readable output
	if !cli.JSONOutput {
		fmt.Printf("\n  Previous: %s\n", prevState)
		fmt.Printf("  New:      %s\n", newState)
		fmt.Println("\n  ⚠ State was force-set (transition validation bypassed)")
	}
}
