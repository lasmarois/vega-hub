package goals

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGoalStateIsValid(t *testing.T) {
	tests := []struct {
		state GoalState
		valid bool
	}{
		{StatePending, true},
		{StateBranching, true},
		{StateWorking, true},
		{StatePushing, true},
		{StateMerging, true},
		{StateDone, true},
		{StateIced, true},
		{StateFailed, true},
		{StateConflict, true},
		{GoalState("invalid"), false},
		{GoalState(""), false},
		{GoalState("PENDING"), false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsValid(); got != tt.valid {
				t.Errorf("GoalState(%q).IsValid() = %v, want %v", tt.state, got, tt.valid)
			}
		})
	}
}

func TestGoalStateIsTerminal(t *testing.T) {
	tests := []struct {
		state    GoalState
		terminal bool
	}{
		{StatePending, false},
		{StateWorking, false},
		{StateDone, true},
		{StateIced, true},
		{StateFailed, false},
		{StateConflict, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.IsTerminal(); got != tt.terminal {
				t.Errorf("GoalState(%q).IsTerminal() = %v, want %v", tt.state, got, tt.terminal)
			}
		})
	}
}

func TestGoalStateNeedsAttention(t *testing.T) {
	tests := []struct {
		state         GoalState
		needsAttention bool
	}{
		{StateWorking, false},
		{StateDone, false},
		{StateFailed, true},
		{StateConflict, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.NeedsAttention(); got != tt.needsAttention {
				t.Errorf("GoalState(%q).NeedsAttention() = %v, want %v", tt.state, got, tt.needsAttention)
			}
		})
	}
}

func TestGoalStateToHumanStatus(t *testing.T) {
	tests := []struct {
		state  GoalState
		status string
	}{
		{StatePending, "Active"},
		{StateBranching, "Active"},
		{StateWorking, "Active"},
		{StatePushing, "Active"},
		{StateMerging, "Active"},
		{StateIced, "Iced"},
		{StateDone, "Completed"},
		{StateFailed, "Needs Attention"},
		{StateConflict, "Needs Attention"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			if got := tt.state.ToHumanStatus(); got != tt.status {
				t.Errorf("GoalState(%q).ToHumanStatus() = %q, want %q", tt.state, got, tt.status)
			}
		})
	}
}

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from  GoalState
		to    GoalState
		valid bool
	}{
		// Valid transitions
		{StatePending, StateBranching, true},
		{StatePending, StateFailed, true},
		{StateBranching, StateWorking, true},
		{StateBranching, StateFailed, true},
		{StateWorking, StatePushing, true},
		{StateWorking, StateIced, true},
		{StateWorking, StateFailed, true},
		{StatePushing, StateMerging, true},
		{StatePushing, StateFailed, true},
		{StatePushing, StateWorking, true}, // Can go back if push fails
		{StateMerging, StateDone, true},
		{StateMerging, StateConflict, true},
		{StateMerging, StateFailed, true},
		{StateConflict, StateMerging, true}, // Retry after resolution
		{StateConflict, StateFailed, true},
		{StateConflict, StateWorking, true},
		{StateIced, StateWorking, true}, // Resume
		{StateFailed, StatePending, true}, // Retry
		{StateFailed, StateWorking, true},
		{StateFailed, StateBranching, true},

		// Invalid transitions
		{StatePending, StateWorking, false},      // Must go through branching
		{StatePending, StateDone, false},
		{StateBranching, StateDone, false},
		{StateWorking, StateDone, false},         // Must go through pushing/merging
		{StateWorking, StateMerging, false},
		{StateDone, StateWorking, false},         // Terminal state
		{StateDone, StatePending, false},
		{StateIced, StateDone, false},
		{StateIced, StatePending, false},
	}

	for _, tt := range tests {
		name := string(tt.from) + "_to_" + string(tt.to)
		t.Run(name, func(t *testing.T) {
			if got := CanTransition(tt.from, tt.to); got != tt.valid {
				t.Errorf("CanTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.valid)
			}
		})
	}
}

func TestStateManager(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "vega-state-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create goals directory structure
	goalsDir := filepath.Join(tmpDir, "goals", "active")
	if err := os.MkdirAll(goalsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a dummy goal file
	goalID := "abc1234"
	goalFile := filepath.Join(goalsDir, goalID+".md")
	if err := os.WriteFile(goalFile, []byte("# Goal"), 0644); err != nil {
		t.Fatal(err)
	}

	sm := NewStateManager(tmpDir)

	t.Run("initial_state", func(t *testing.T) {
		// No state file yet, should return working (legacy behavior)
		state, err := sm.GetState(goalID)
		if err != nil {
			t.Fatalf("GetState failed: %v", err)
		}
		if state != StateWorking {
			t.Errorf("expected StateWorking for new goal, got %s", state)
		}
	})

	t.Run("first_transition_pending", func(t *testing.T) {
		// First transition must be to pending or working
		err := sm.Transition(goalID, StatePending, "Goal created", nil)
		if err != nil {
			t.Fatalf("Transition to pending failed: %v", err)
		}

		state, err := sm.GetState(goalID)
		if err != nil {
			t.Fatalf("GetState failed: %v", err)
		}
		if state != StatePending {
			t.Errorf("expected StatePending, got %s", state)
		}
	})

	t.Run("valid_transition", func(t *testing.T) {
		err := sm.Transition(goalID, StateBranching, "Creating worktree", nil)
		if err != nil {
			t.Fatalf("Transition failed: %v", err)
		}

		state, err := sm.GetState(goalID)
		if err != nil {
			t.Fatalf("GetState failed: %v", err)
		}
		if state != StateBranching {
			t.Errorf("expected StateBranching, got %s", state)
		}
	})

	t.Run("invalid_transition", func(t *testing.T) {
		// Try to go directly to done (invalid)
		err := sm.Transition(goalID, StateDone, "Skip to done", nil)
		if err == nil {
			t.Error("expected error for invalid transition, got nil")
		}

		// Check it's an InvalidTransitionError
		if _, ok := err.(*InvalidTransitionError); !ok {
			t.Errorf("expected InvalidTransitionError, got %T", err)
		}

		// State should not have changed
		state, _ := sm.GetState(goalID)
		if state != StateBranching {
			t.Errorf("state should still be branching, got %s", state)
		}
	})

	t.Run("transition_to_working", func(t *testing.T) {
		err := sm.Transition(goalID, StateWorking, "Worktree ready", map[string]string{
			"branch": "goal-abc1234-test",
		})
		if err != nil {
			t.Fatalf("Transition failed: %v", err)
		}
	})

	t.Run("get_history", func(t *testing.T) {
		history, err := sm.GetHistory(goalID)
		if err != nil {
			t.Fatalf("GetHistory failed: %v", err)
		}

		if len(history) != 3 {
			t.Fatalf("expected 3 events, got %d", len(history))
		}

		// Verify event order
		if history[0].State != StatePending {
			t.Errorf("first event should be pending, got %s", history[0].State)
		}
		if history[1].State != StateBranching {
			t.Errorf("second event should be branching, got %s", history[1].State)
		}
		if history[2].State != StateWorking {
			t.Errorf("third event should be working, got %s", history[2].State)
		}

		// Verify prev_state tracking
		if history[1].PrevState != StatePending {
			t.Errorf("expected prev_state=pending, got %s", history[1].PrevState)
		}
		if history[2].PrevState != StateBranching {
			t.Errorf("expected prev_state=branching, got %s", history[2].PrevState)
		}

		// Verify details
		if history[2].Details["branch"] != "goal-abc1234-test" {
			t.Errorf("expected branch detail, got %v", history[2].Details)
		}
	})

	t.Run("force_state", func(t *testing.T) {
		// Force to an invalid transition
		err := sm.ForceState(goalID, StateDone, "Manual override")
		if err != nil {
			t.Fatalf("ForceState failed: %v", err)
		}

		state, _ := sm.GetState(goalID)
		if state != StateDone {
			t.Errorf("expected done after force, got %s", state)
		}

		// Check that it was marked as forced
		history, _ := sm.GetHistory(goalID)
		lastEvent := history[len(history)-1]
		if lastEvent.Details["forced"] != "true" {
			t.Error("forced event should have forced=true in details")
		}
	})
}

func TestStateManagerGoalsInState(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vega-state-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	goalsDir := filepath.Join(tmpDir, "goals", "active")
	if err := os.MkdirAll(goalsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sm := NewStateManager(tmpDir)

	// Create multiple goals in different states
	goals := []struct {
		id    string
		state GoalState
	}{
		{"aaa1111", StateWorking},
		{"bbb2222", StateWorking},
		{"ccc3333", StatePushing},
	}

	for _, g := range goals {
		// Create goal file
		goalFile := filepath.Join(goalsDir, g.id+".md")
		os.WriteFile(goalFile, []byte("# Goal"), 0644)

		// Set state
		sm.Transition(g.id, StatePending, "created", nil)
		sm.Transition(g.id, StateBranching, "branching", nil)
		sm.Transition(g.id, StateWorking, "working", nil)
		if g.state == StatePushing {
			sm.Transition(g.id, StatePushing, "pushing", nil)
		}
	}

	// Query goals in working state
	working, err := sm.GoalsInState(StateWorking)
	if err != nil {
		t.Fatalf("GoalsInState failed: %v", err)
	}
	if len(working) != 2 {
		t.Errorf("expected 2 goals in working state, got %d", len(working))
	}

	// Query goals in pushing state
	pushing, err := sm.GoalsInState(StatePushing)
	if err != nil {
		t.Fatalf("GoalsInState failed: %v", err)
	}
	if len(pushing) != 1 {
		t.Errorf("expected 1 goal in pushing state, got %d", len(pushing))
	}
}

func TestStateManagerStuckGoals(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vega-state-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	goalsDir := filepath.Join(tmpDir, "goals", "active")
	if err := os.MkdirAll(goalsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sm := NewStateManager(tmpDir)

	// Create a goal and manually write an old event
	goalID := "abc1234" // 7 hex chars
	goalFile := filepath.Join(goalsDir, goalID+".md")
	os.WriteFile(goalFile, []byte("# Goal"), 0644)

	// Write state file with old timestamp
	stateFile := filepath.Join(goalsDir, goalID+".state.jsonl")
	oldTime := time.Now().UTC().Add(-2 * time.Hour)
	event := StateEvent{
		Timestamp: oldTime,
		State:     StateBranching,
		Reason:    "Creating worktree",
	}
	data, _ := json.Marshal(event)
	os.WriteFile(stateFile, append(data, '\n'), 0644)

	// Query stuck goals (stuck for > 1 hour)
	stuck, err := sm.StuckGoals(1 * time.Hour)
	if err != nil {
		t.Fatalf("StuckGoals failed: %v", err)
	}

	if len(stuck) != 1 {
		t.Fatalf("expected 1 stuck goal, got %d", len(stuck))
	}

	if stuck[0].GoalID != goalID {
		t.Errorf("expected goal %s, got %s", goalID, stuck[0].GoalID)
	}
	if stuck[0].State != StateBranching {
		t.Errorf("expected state branching, got %s", stuck[0].State)
	}
}

func TestIsValidGoalID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"abc1234", true},  // 7 char hex
		{"0000000", true},  // 7 zeros
		{"abcdefg", false}, // g is not hex
		{"ABC1234", false}, // uppercase
		{"1", true},        // legacy numeric
		{"10", true},
		{"999", true},
		{"12345", false},   // too long for numeric
		{"", false},
		{"abc123", false},  // 6 chars
		{"abc12345", false}, // 8 chars
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			if got := isValidGoalID(tt.id); got != tt.valid {
				t.Errorf("isValidGoalID(%q) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}

func TestSyncGoalFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vega-sync-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	goalsDir := filepath.Join(tmpDir, "goals", "active")
	if err := os.MkdirAll(goalsDir, 0755); err != nil {
		t.Fatal(err)
	}

	goalID := "abc1234"
	sm := NewStateManager(tmpDir)

	// Create a goal file with initial Active status
	goalFileContent := `# Goal abc1234: Test Goal

## Overview

Test goal overview.

## Status

**Current Phase**: 1
**Status**: Active
**Assigned To**: -

## Notes

Some notes here.
`
	goalFile := filepath.Join(goalsDir, goalID+".md")
	if err := os.WriteFile(goalFile, []byte(goalFileContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Initialize state
	if err := sm.Transition(goalID, StatePending, "Goal created", nil); err != nil {
		t.Fatalf("Initial transition failed: %v", err)
	}

	t.Run("transition_to_working_syncs_active", func(t *testing.T) {
		sm.Transition(goalID, StateBranching, "Creating worktree", nil)
		sm.Transition(goalID, StateWorking, "Worktree ready", nil)

		// Read the goal file and verify status is still Active
		content, err := os.ReadFile(goalFile)
		if err != nil {
			t.Fatalf("Failed to read goal file: %v", err)
		}
		if !contains(string(content), "**Status**: Active") {
			t.Errorf("Goal file should have Status: Active, got:\n%s", content)
		}
	})

	t.Run("transition_to_iced_syncs_iced", func(t *testing.T) {
		sm.Transition(goalID, StateIced, "Blocked on dependency", nil)

		content, err := os.ReadFile(goalFile)
		if err != nil {
			t.Fatalf("Failed to read goal file: %v", err)
		}
		if !contains(string(content), "**Status**: Iced") {
			t.Errorf("Goal file should have Status: Iced, got:\n%s", content)
		}
	})

	t.Run("transition_to_working_from_iced_syncs_active", func(t *testing.T) {
		sm.Transition(goalID, StateWorking, "Resumed", nil)

		content, err := os.ReadFile(goalFile)
		if err != nil {
			t.Fatalf("Failed to read goal file: %v", err)
		}
		if !contains(string(content), "**Status**: Active") {
			t.Errorf("Goal file should have Status: Active, got:\n%s", content)
		}
	})

	t.Run("force_state_to_failed_syncs_needs_attention", func(t *testing.T) {
		sm.ForceState(goalID, StateFailed, "Something went wrong")

		content, err := os.ReadFile(goalFile)
		if err != nil {
			t.Fatalf("Failed to read goal file: %v", err)
		}
		if !contains(string(content), "**Status**: Needs Attention") {
			t.Errorf("Goal file should have Status: Needs Attention, got:\n%s", content)
		}
	})

	t.Run("force_state_to_done_syncs_completed", func(t *testing.T) {
		sm.ForceState(goalID, StateDone, "Manual completion")

		content, err := os.ReadFile(goalFile)
		if err != nil {
			t.Fatalf("Failed to read goal file: %v", err)
		}
		if !contains(string(content), "**Status**: Completed") {
			t.Errorf("Goal file should have Status: Completed, got:\n%s", content)
		}
	})
}

func TestSyncGoalFileExplicit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vega-sync-explicit-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	goalsDir := filepath.Join(tmpDir, "goals", "active")
	if err := os.MkdirAll(goalsDir, 0755); err != nil {
		t.Fatal(err)
	}

	goalID := "def5678"
	sm := NewStateManager(tmpDir)

	// Create a goal file with no status line initially
	goalFileContent := `# Goal def5678: Another Test

## Overview

Overview text.

## Status

**Current Phase**: 1
**Status**:Active
**Assigned To**: -
`
	goalFile := filepath.Join(goalsDir, goalID+".md")
	if err := os.WriteFile(goalFile, []byte(goalFileContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Set up state manually without triggering auto-sync
	stateFile := filepath.Join(goalsDir, goalID+".state.jsonl")
	event := StateEvent{
		Timestamp: time.Now().UTC(),
		State:     StateIced,
		Reason:    "Test",
	}
	data, _ := json.Marshal(event)
	os.WriteFile(stateFile, append(data, '\n'), 0644)

	// Call SyncGoalFile explicitly
	if err := sm.SyncGoalFile(goalID); err != nil {
		t.Fatalf("SyncGoalFile failed: %v", err)
	}

	// Verify the status was updated (note: no space after colon in original)
	content, err := os.ReadFile(goalFile)
	if err != nil {
		t.Fatalf("Failed to read goal file: %v", err)
	}
	if !contains(string(content), "**Status**: Iced") {
		t.Errorf("Goal file should have Status: Iced after sync, got:\n%s", content)
	}
}

// contains checks if substr is in s
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
