package goals

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// GoalState represents the lifecycle state of a goal
type GoalState string

const (
	StatePending   GoalState = "pending"   // Created, not yet branched
	StateBranching GoalState = "branching" // Creating branch/worktree
	StateWorking   GoalState = "working"   // Active development
	StatePushing   GoalState = "pushing"   // Committing/pushing changes
	StateMerging   GoalState = "merging"   // Merging to target branch
	StateDone      GoalState = "done"      // Successfully completed
	StateIced      GoalState = "iced"      // Paused/frozen
	StateFailed    GoalState = "failed"    // Failed (recoverable)
	StateConflict  GoalState = "conflict"  // Merge conflict detected
)

// AllStates returns all valid goal states
func AllStates() []GoalState {
	return []GoalState{
		StatePending,
		StateBranching,
		StateWorking,
		StatePushing,
		StateMerging,
		StateDone,
		StateIced,
		StateFailed,
		StateConflict,
	}
}

// IsValid checks if a state string is a valid GoalState
func (s GoalState) IsValid() bool {
	for _, valid := range AllStates() {
		if s == valid {
			return true
		}
	}
	return false
}

// IsTerminal returns true if the state is a terminal state (done, iced)
func (s GoalState) IsTerminal() bool {
	return s == StateDone || s == StateIced
}

// NeedsAttention returns true if the state requires user intervention
func (s GoalState) NeedsAttention() bool {
	return s == StateFailed || s == StateConflict
}

// ToHumanStatus converts state to human-readable status for markdown
func (s GoalState) ToHumanStatus() string {
	switch s {
	case StatePending, StateBranching, StateWorking, StatePushing, StateMerging:
		return "Active"
	case StateIced:
		return "Iced"
	case StateDone:
		return "Completed"
	case StateFailed, StateConflict:
		return "Needs Attention"
	default:
		return "Unknown"
	}
}

// validTransitions defines the allowed state transitions
// Key: from state, Value: list of allowed to states
var validTransitions = map[GoalState][]GoalState{
	StatePending:   {StateBranching, StateFailed},
	StateBranching: {StateWorking, StateFailed},
	StateWorking:   {StatePushing, StateIced, StateFailed},
	StatePushing:   {StateMerging, StateFailed, StateWorking}, // Can go back to working if push fails non-fatally
	StateMerging:   {StateDone, StateConflict, StateFailed},
	StateConflict:  {StateMerging, StateFailed, StateWorking}, // After resolution, retry merge or go back to working
	StateIced:      {StateWorking},                            // Resume
	StateFailed:    {StatePending, StateWorking, StateBranching}, // Retry from various points
	StateDone:      {}, // Terminal, no transitions out
}

// CanTransition checks if a transition from one state to another is valid
func CanTransition(from, to GoalState) bool {
	allowed, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// StateEvent represents a single state change event
type StateEvent struct {
	Timestamp time.Time         `json:"ts"`
	State     GoalState         `json:"state"`
	PrevState GoalState         `json:"prev_state,omitempty"`
	Reason    string            `json:"reason,omitempty"`
	User      string            `json:"user,omitempty"`
	Details   map[string]string `json:"details,omitempty"`
}

// StateManager handles goal state persistence and transitions
type StateManager struct {
	dir string // vega-missile directory
	mu  sync.RWMutex
}

// NewStateManager creates a new StateManager
func NewStateManager(dir string) *StateManager {
	return &StateManager{dir: dir}
}

// stateFilePath returns the path to a goal's state file
func (m *StateManager) stateFilePath(goalID string) string {
	// Check folder structure first (goals/active/<id>/<id>.state.jsonl)
	// Then fall back to flat structure (goals/active/<id>.state.jsonl)
	
	dirs := []string{"active", "iced", "history"}
	for _, dir := range dirs {
		// Folder structure
		folderPath := filepath.Join(m.dir, "goals", dir, goalID, goalID+".state.jsonl")
		if _, err := os.Stat(folderPath); err == nil {
			return folderPath
		}
		// Flat structure (legacy)
		flatPath := filepath.Join(m.dir, "goals", dir, goalID+".state.jsonl")
		if _, err := os.Stat(flatPath); err == nil {
			return flatPath
		}
	}
	
	// Default to folder structure in active for new goals
	return filepath.Join(m.dir, "goals", "active", goalID, goalID+".state.jsonl")
}

// stateFilePathForWrite returns the path for writing (always in the goal's current location)
func (m *StateManager) stateFilePathForWrite(goalID string) (string, error) {
	// Check where the goal file is to determine where state should go
	// Supports both folder structure (goals/<dir>/<id>/<id>.md) and flat (goals/<dir>/<id>.md)
	
	dirs := []string{"active", "iced", "history"}
	for _, dir := range dirs {
		// Folder structure (preferred)
		folderGoal := filepath.Join(m.dir, "goals", dir, goalID, goalID+".md")
		if _, err := os.Stat(folderGoal); err == nil {
			return filepath.Join(m.dir, "goals", dir, goalID, goalID+".state.jsonl"), nil
		}
		// Flat structure (legacy)
		flatGoal := filepath.Join(m.dir, "goals", dir, goalID+".md")
		if _, err := os.Stat(flatGoal); err == nil {
			return filepath.Join(m.dir, "goals", dir, goalID+".state.jsonl"), nil
		}
	}
	
	// Goal doesn't exist yet, default to folder structure in active
	return filepath.Join(m.dir, "goals", "active", goalID, goalID+".state.jsonl"), nil
}

// GetState returns the current state of a goal
func (m *StateManager) GetState(goalID string) (GoalState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	events, err := m.readEvents(goalID)
	if err != nil {
		if os.IsNotExist(err) {
			// No state file = legacy goal, assume working
			return StateWorking, nil
		}
		return "", fmt.Errorf("reading state: %w", err)
	}
	
	if len(events) == 0 {
		return StateWorking, nil // Empty file = assume working
	}
	
	return events[len(events)-1].State, nil
}

// GetHistory returns the full state history for a goal
func (m *StateManager) GetHistory(goalID string) ([]StateEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.readEvents(goalID)
}

// GetLastEvent returns the most recent state event for a goal
func (m *StateManager) GetLastEvent(goalID string) (*StateEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	events, err := m.readEvents(goalID)
	if err != nil {
		return nil, err
	}
	
	if len(events) == 0 {
		return nil, nil
	}
	
	return &events[len(events)-1], nil
}

// Transition changes the state of a goal with validation
func (m *StateManager) Transition(goalID string, newState GoalState, reason string, details map[string]string) error {
	return m.TransitionWithUser(goalID, newState, reason, "", details)
}

// TransitionWithUser changes the state of a goal with validation and user tracking
func (m *StateManager) TransitionWithUser(goalID string, newState GoalState, reason, user string, details map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !newState.IsValid() {
		return fmt.Errorf("invalid state: %s", newState)
	}
	
	// Get current state
	events, err := m.readEvents(goalID)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading current state: %w", err)
	}
	
	var currentState GoalState
	if len(events) == 0 {
		// No previous state - only allow pending or working (for legacy/backfill)
		if newState != StatePending && newState != StateWorking {
			return fmt.Errorf("first state must be 'pending' or 'working', got '%s'", newState)
		}
		currentState = "" // No previous state
	} else {
		currentState = events[len(events)-1].State
		
		// Validate transition
		if !CanTransition(currentState, newState) {
			return &InvalidTransitionError{
				GoalID: goalID,
				From:   currentState,
				To:     newState,
			}
		}
	}
	
	// Create event
	event := StateEvent{
		Timestamp: time.Now().UTC(),
		State:     newState,
		PrevState: currentState,
		Reason:    reason,
		User:      user,
		Details:   details,
	}
	
	// Append to file
	if err := m.appendEvent(goalID, event); err != nil {
		return err
	}
	
	// Sync goal markdown file status (best effort - don't fail transition on sync error)
	// Skip sync for pending state as goal file may not exist yet
	if newState != StatePending {
		if syncErr := m.syncGoalFileInternal(goalID, newState); syncErr != nil {
			// Log but don't fail - state file is the source of truth
			// The markdown file sync is for human readability
			_ = syncErr // Silently ignore sync errors
		}
	}
	
	return nil
}

// ForceState sets a goal's state without validation (escape hatch)
func (m *StateManager) ForceState(goalID string, newState GoalState, reason string) error {
	return m.ForceStateWithUser(goalID, newState, reason, "")
}

// ForceStateWithUser sets a goal's state without validation (escape hatch) with user tracking
func (m *StateManager) ForceStateWithUser(goalID string, newState GoalState, reason, user string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if !newState.IsValid() {
		return fmt.Errorf("invalid state: %s", newState)
	}
	
	// Get current state for logging
	events, _ := m.readEvents(goalID)
	var currentState GoalState
	if len(events) > 0 {
		currentState = events[len(events)-1].State
	}
	
	event := StateEvent{
		Timestamp: time.Now().UTC(),
		State:     newState,
		PrevState: currentState,
		Reason:    fmt.Sprintf("[FORCED] %s", reason),
		User:      user,
		Details:   map[string]string{"forced": "true"},
	}
	
	if err := m.appendEvent(goalID, event); err != nil {
		return err
	}
	
	// Sync goal markdown file status (best effort)
	_ = m.syncGoalFileInternal(goalID, newState)
	
	return nil
}

// GoalsInState returns all goal IDs currently in a specific state
func (m *StateManager) GoalsInState(state GoalState) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var result []string
	
	// Scan active goals
	activeDir := filepath.Join(m.dir, "goals", "active")
	entries, err := os.ReadDir(activeDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		goalID := entry.Name()[:len(entry.Name())-len(".state.jsonl")]
		if !isValidGoalID(goalID) {
			continue
		}
		
		goalState, err := m.getStateUnsafe(goalID)
		if err != nil {
			continue
		}
		if goalState == state {
			result = append(result, goalID)
		}
	}
	
	return result, nil
}

// StuckGoals returns goals that have been in a non-terminal state for longer than maxAge
func (m *StateManager) StuckGoals(maxAge time.Duration) ([]StuckGoal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var result []StuckGoal
	cutoff := time.Now().UTC().Add(-maxAge)
	
	// States that can be "stuck"
	stuckStates := []GoalState{StateBranching, StatePushing, StateMerging}
	
	activeDir := filepath.Join(m.dir, "goals", "active")
	entries, err := os.ReadDir(activeDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		goalID := entry.Name()[:len(entry.Name())-len(".state.jsonl")]
		if !isValidGoalID(goalID) {
			continue
		}
		
		events, err := m.readEvents(goalID)
		if err != nil || len(events) == 0 {
			continue
		}
		
		lastEvent := events[len(events)-1]
		
		// Check if in a stuck-able state and older than cutoff
		for _, stuckState := range stuckStates {
			if lastEvent.State == stuckState && lastEvent.Timestamp.Before(cutoff) {
				result = append(result, StuckGoal{
					GoalID:    goalID,
					State:     lastEvent.State,
					Since:     lastEvent.Timestamp,
					Duration:  time.Since(lastEvent.Timestamp),
				})
				break
			}
		}
	}
	
	return result, nil
}

// StuckGoal represents a goal that appears to be stuck
type StuckGoal struct {
	GoalID   string        `json:"goal_id"`
	State    GoalState     `json:"state"`
	Since    time.Time     `json:"since"`
	Duration time.Duration `json:"duration"`
}

// InvalidTransitionError is returned when an invalid state transition is attempted
type InvalidTransitionError struct {
	GoalID string
	From   GoalState
	To     GoalState
}

func (e *InvalidTransitionError) Error() string {
	allowed := validTransitions[e.From]
	return fmt.Sprintf("invalid state transition for goal %s: %s → %s (allowed: %v)", 
		e.GoalID, e.From, e.To, allowed)
}

// Internal helpers

func (m *StateManager) readEvents(goalID string) ([]StateEvent, error) {
	path := m.stateFilePath(goalID)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	var events []StateEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var event StateEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // Skip malformed lines
		}
		events = append(events, event)
	}
	
	return events, scanner.Err()
}

func (m *StateManager) appendEvent(goalID string, event StateEvent) error {
	path, err := m.stateFilePathForWrite(goalID)
	if err != nil {
		return err
	}
	
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	
	_, err = file.WriteString(string(data) + "\n")
	return err
}

// getStateUnsafe reads state without locking (caller must hold lock)
func (m *StateManager) getStateUnsafe(goalID string) (GoalState, error) {
	events, err := m.readEvents(goalID)
	if err != nil {
		if os.IsNotExist(err) {
			return StateWorking, nil
		}
		return "", err
	}
	if len(events) == 0 {
		return StateWorking, nil
	}
	return events[len(events)-1].State, nil
}

// isValidGoalID checks if a string looks like a goal ID
func isValidGoalID(s string) bool {
	// Hash IDs: 7 hex chars (e.g., "f3a8b2c")
	if len(s) == 7 {
		for _, c := range s {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return false
			}
		}
		return true
	}
	// Legacy numeric IDs
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0 && len(s) <= 4
}

// MoveStateFile moves a goal's state file to a new location (e.g., when archiving)
func (m *StateManager) MoveStateFile(goalID, fromDir, toDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check folder structure first, then flat
	fromFolderPath := filepath.Join(m.dir, "goals", fromDir, goalID, goalID+".state.jsonl")
	fromFlatPath := filepath.Join(m.dir, "goals", fromDir, goalID+".state.jsonl")
	
	var fromPath string
	if _, err := os.Stat(fromFolderPath); err == nil {
		fromPath = fromFolderPath
	} else if _, err := os.Stat(fromFlatPath); err == nil {
		fromPath = fromFlatPath
	} else {
		return nil // Nothing to move
	}
	
	// Determine target path (prefer folder structure if goal folder exists)
	toFolderDir := filepath.Join(m.dir, "goals", toDir, goalID)
	var toPath string
	if _, err := os.Stat(toFolderDir); err == nil {
		toPath = filepath.Join(toFolderDir, goalID+".state.jsonl")
	} else {
		toPath = filepath.Join(m.dir, "goals", toDir, goalID+".state.jsonl")
	}
	
	return os.Rename(fromPath, toPath)
}

// SyncGoalFile updates the goal markdown file's Status field to match the current state
// This keeps the human-readable markdown in sync with the machine-readable state
func (m *StateManager) SyncGoalFile(goalID string) error {
	state, err := m.GetState(goalID)
	if err != nil {
		return fmt.Errorf("getting state: %w", err)
	}
	
	return m.syncGoalFileInternal(goalID, state)
}

// syncGoalFileInternal updates goal file status without needing to read state (for use within locked sections)
func (m *StateManager) syncGoalFileInternal(goalID string, state GoalState) error {
	// Find the goal file
	goalPath, err := m.findGoalFile(goalID)
	if err != nil {
		return err
	}
	
	return m.updateGoalFileStatus(goalPath, state.ToHumanStatus())
}

// findGoalFile locates the goal markdown file in active, iced, or history
func (m *StateManager) findGoalFile(goalID string) (string, error) {
	locations := []string{
		filepath.Join(m.dir, "goals", "active", goalID+".md"),
		filepath.Join(m.dir, "goals", "iced", goalID+".md"),
		filepath.Join(m.dir, "goals", "history", goalID+".md"),
	}
	
	for _, path := range locations {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	
	return "", fmt.Errorf("goal file not found for %s", goalID)
}

// updateGoalFileStatus updates the **Status**: line in a goal markdown file
func (m *StateManager) updateGoalFileStatus(goalPath, newStatus string) error {
	content, err := os.ReadFile(goalPath)
	if err != nil {
		return fmt.Errorf("reading goal file: %w", err)
	}
	
	text := string(content)
	
	// Pattern matches **Status**: <anything>
	// This handles various formats like:
	// - **Status**: Active
	// - **Status**: Iced
	// - **Status**:Active (no space)
	statusRe := regexp.MustCompile(`(?m)^\*\*Status\*\*:\s*.*$`)
	
	if !statusRe.MatchString(text) {
		// No status line found, nothing to update
		return nil
	}
	
	// Replace with new status
	newText := statusRe.ReplaceAllString(text, fmt.Sprintf("**Status**: %s", newStatus))
	
	// Only write if changed
	if newText == text {
		return nil
	}
	
	return os.WriteFile(goalPath, []byte(newText), 0644)
}
