package goals

import (
	"os"
	"path/filepath"
	"testing"
)

// setupCompletionTestDir creates a temp directory with full goal structure
func setupCompletionTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create directory structure matching vega-missile
	os.MkdirAll(filepath.Join(dir, "goals", "active"), 0755)
	os.MkdirAll(filepath.Join(dir, "goals", "iced"), 0755)
	os.MkdirAll(filepath.Join(dir, "goals", "history"), 0755)
	os.MkdirAll(filepath.Join(dir, "workspaces", "test-project"), 0755)

	return dir
}

func writeGoalFile(t *testing.T, dir, goalID, content string) {
	t.Helper()
	path := filepath.Join(dir, "goals", "active", goalID+".md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write goal file: %v", err)
	}
}

func writeGoalFileInFolder(t *testing.T, dir, goalID, content string) {
	t.Helper()
	goalDir := filepath.Join(dir, "goals", "active", goalID)
	os.MkdirAll(goalDir, 0755)
	path := filepath.Join(goalDir, goalID+".md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write goal file: %v", err)
	}
}

// ============================================================================
// Goal File Phases Tests
// ============================================================================

func TestCheckGoal_AllPhasesComplete(t *testing.T) {
	dir := setupCompletionTestDir(t)

	goalContent := `# Goal #abc1234: Test Goal

## Phases

### Phase 1: Setup
- [x] Initialize project
- [x] Configure settings

### Phase 2: Implementation
- [x] Implement feature A
- [x] Implement feature B

## Acceptance Criteria

- [x] All tests pass
- [x] Documentation complete
`
	writeGoalFile(t, dir, "abc1234", goalContent)

	checker := NewCompletionChecker(dir)
	status, err := checker.CheckGoal("abc1234")
	if err != nil {
		t.Fatalf("CheckGoal failed: %v", err)
	}

	if !status.Complete {
		t.Error("expected goal to be complete")
	}
	if status.TotalPhases != 2 {
		t.Errorf("expected 2 total phases, got %d", status.TotalPhases)
	}
	if status.CompletedPhases != 2 {
		t.Errorf("expected 2 completed phases, got %d", status.CompletedPhases)
	}
	if len(status.MissingTasks) != 0 {
		t.Errorf("expected no missing tasks, got %v", status.MissingTasks)
	}
	if status.Confidence < 0.5 {
		t.Errorf("expected confidence >= 0.5, got %f", status.Confidence)
	}

	// Should have signals for goal phases and acceptance
	hasGoalPhasesSignal := false
	hasAcceptanceSignal := false
	for _, s := range status.Signals {
		if s.Type == SignalGoalPhases {
			hasGoalPhasesSignal = true
		}
		if s.Type == SignalAcceptance {
			hasAcceptanceSignal = true
		}
	}
	if !hasGoalPhasesSignal {
		t.Error("expected SignalGoalPhases signal")
	}
	if !hasAcceptanceSignal {
		t.Error("expected SignalAcceptance signal")
	}
}

func TestCheckGoal_PartiallyComplete(t *testing.T) {
	dir := setupCompletionTestDir(t)

	goalContent := `# Goal #def5678: Test Goal

## Phases

### Phase 1: Setup
- [x] Initialize project
- [x] Configure settings

### Phase 2: Implementation
- [x] Implement feature A
- [ ] Implement feature B
- [ ] Write tests

## Acceptance Criteria

- [ ] All tests pass
- [x] Documentation complete
`
	writeGoalFile(t, dir, "def5678", goalContent)

	checker := NewCompletionChecker(dir)
	status, err := checker.CheckGoal("def5678")
	if err != nil {
		t.Fatalf("CheckGoal failed: %v", err)
	}

	if status.Complete {
		t.Error("expected goal to be incomplete")
	}
	if status.TotalPhases != 2 {
		t.Errorf("expected 2 total phases, got %d", status.TotalPhases)
	}
	if status.CompletedPhases != 1 {
		t.Errorf("expected 1 completed phase, got %d", status.CompletedPhases)
	}

	// Should have 3 missing tasks (2 from phase 2, 1 from acceptance)
	if len(status.MissingTasks) != 3 {
		t.Errorf("expected 3 missing tasks, got %d: %v", len(status.MissingTasks), status.MissingTasks)
	}

	// Check missing tasks include correct context
	foundPhase2Task := false
	foundAcceptanceTask := false
	for _, task := range status.MissingTasks {
		if task == "Phase 2: Implement feature B" {
			foundPhase2Task = true
		}
		if task == "Acceptance: All tests pass" {
			foundAcceptanceTask = true
		}
	}
	if !foundPhase2Task {
		t.Error("expected to find 'Phase 2: Implement feature B' in missing tasks")
	}
	if !foundAcceptanceTask {
		t.Error("expected to find 'Acceptance: All tests pass' in missing tasks")
	}
}

// ============================================================================
// Acceptance Criteria Tests
// ============================================================================

func TestCheckGoal_AcceptanceCriteriaComplete(t *testing.T) {
	dir := setupCompletionTestDir(t)

	goalContent := `# Goal #acc1234: Test Goal

## Phases

### Phase 1: Work
- [x] Do work

## Acceptance Criteria

- [x] Feature works correctly
- [x] Edge cases handled
- [x] Performance acceptable
`
	writeGoalFile(t, dir, "acc1234", goalContent)

	checker := NewCompletionChecker(dir)
	status, err := checker.CheckGoal("acc1234")
	if err != nil {
		t.Fatalf("CheckGoal failed: %v", err)
	}

	if !status.Complete {
		t.Error("expected goal to be complete")
	}

	// Check for acceptance signal
	hasAcceptanceSignal := false
	for _, s := range status.Signals {
		if s.Type == SignalAcceptance {
			hasAcceptanceSignal = true
			if s.Message != "All 3 acceptance criteria met" {
				t.Errorf("unexpected message: %s", s.Message)
			}
		}
	}
	if !hasAcceptanceSignal {
		t.Error("expected SignalAcceptance signal")
	}
}

func TestCheckGoal_AcceptanceCriteriaIncomplete(t *testing.T) {
	dir := setupCompletionTestDir(t)

	goalContent := `# Goal #acc5678: Test Goal

## Phases

### Phase 1: Work
- [x] Do work

## Acceptance Criteria

- [x] Feature works correctly
- [ ] Edge cases handled
- [x] Performance acceptable
`
	writeGoalFile(t, dir, "acc5678", goalContent)

	checker := NewCompletionChecker(dir)
	status, err := checker.CheckGoal("acc5678")
	if err != nil {
		t.Fatalf("CheckGoal failed: %v", err)
	}

	if status.Complete {
		t.Error("expected goal to be incomplete due to missing acceptance criteria")
	}

	// Should NOT have acceptance signal
	for _, s := range status.Signals {
		if s.Type == SignalAcceptance {
			t.Error("should not have SignalAcceptance when criteria incomplete")
		}
	}

	// Should have the incomplete criterion in missing
	found := false
	for _, task := range status.MissingTasks {
		if task == "Acceptance: Edge cases handled" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'Acceptance: Edge cases handled' in missing tasks")
	}
}

// ============================================================================
// Folder Structure Tests
// ============================================================================

func TestCheckGoal_GoalInFolder(t *testing.T) {
	dir := setupCompletionTestDir(t)

	goalContent := `# Goal #folder1: Test Goal

## Phases

### Phase 1: Work
- [x] Do work

## Acceptance Criteria

- [x] All done
`
	// Write goal in folder structure: goals/active/<id>/<id>.md
	writeGoalFileInFolder(t, dir, "folder1", goalContent)

	checker := NewCompletionChecker(dir)
	status, err := checker.CheckGoal("folder1")
	if err != nil {
		t.Fatalf("CheckGoal failed: %v", err)
	}

	if !status.Complete {
		t.Error("expected goal to be complete")
	}

	// Check for acceptance signal with correct source path
	hasAcceptanceSignal := false
	for _, s := range status.Signals {
		if s.Type == SignalAcceptance {
			hasAcceptanceSignal = true
			expectedPath := filepath.Join(dir, "goals", "active", "folder1", "folder1.md")
			if s.Source != expectedPath {
				t.Errorf("expected source %s, got %s", expectedPath, s.Source)
			}
		}
	}
	if !hasAcceptanceSignal {
		t.Error("expected SignalAcceptance signal")
	}
}

// ============================================================================
// Confidence Scoring Tests
// ============================================================================

func TestCheckGoal_ConfidenceCalculation(t *testing.T) {
	dir := setupCompletionTestDir(t)

	// Goal with all signals present
	goalContent := `# Goal #conf1234: Test Goal

## Phases

### Phase 1: Work
- [x] Do work

## Acceptance Criteria

- [x] Feature works
`
	writeGoalFile(t, dir, "conf1234", goalContent)

	checker := NewCompletionChecker(dir)
	status, err := checker.CheckGoal("conf1234")
	if err != nil {
		t.Fatalf("CheckGoal failed: %v", err)
	}

	// Should have high confidence with goal phases (0.5) + acceptance (0.4) = 0.9
	if status.Confidence < 0.8 {
		t.Errorf("expected confidence >= 0.8, got %f", status.Confidence)
	}
}

func TestCheckGoal_ConfidencePenaltyForMissing(t *testing.T) {
	dir := setupCompletionTestDir(t)

	goalContent := `# Goal #conf5678: Test Goal

## Phases

### Phase 1: Work
- [x] Task 1
- [ ] Task 2
- [ ] Task 3
- [ ] Task 4
- [ ] Task 5
`
	writeGoalFile(t, dir, "conf5678", goalContent)

	checker := NewCompletionChecker(dir)
	status, err := checker.CheckGoal("conf5678")
	if err != nil {
		t.Fatalf("CheckGoal failed: %v", err)
	}

	// Multiple missing tasks should reduce confidence
	if status.Confidence > 0.3 {
		t.Errorf("expected low confidence due to missing tasks, got %f", status.Confidence)
	}
}

// ============================================================================
// Legacy Function Tests (backwards compatibility)
// ============================================================================

func TestIsGoalComplete_LegacyFunction(t *testing.T) {
	dir := setupCompletionTestDir(t)

	goalContent := `# Goal #legacy1: Test Goal

## Phases

### Phase 1: Work
- [x] Do work

## Acceptance Criteria

- [x] Done
`
	writeGoalFile(t, dir, "legacy1", goalContent)

	status, err := IsGoalComplete("legacy1", dir)
	if err != nil {
		t.Fatalf("IsGoalComplete failed: %v", err)
	}

	if !status.Complete {
		t.Error("expected goal to be complete")
	}
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestCheckGoal_NoPhases(t *testing.T) {
	dir := setupCompletionTestDir(t)

	goalContent := `# Goal #nophase: Test Goal

## Overview

Just an overview, no phases.

## Notes

Some notes.
`
	writeGoalFile(t, dir, "nophase", goalContent)

	checker := NewCompletionChecker(dir)
	status, err := checker.CheckGoal("nophase")
	if err != nil {
		t.Fatalf("CheckGoal failed: %v", err)
	}

	if status.Complete {
		t.Error("expected goal with no phases to be incomplete")
	}
	if status.TotalPhases != 0 {
		t.Errorf("expected 0 phases, got %d", status.TotalPhases)
	}
}

func TestCheckGoal_NotFound(t *testing.T) {
	dir := setupCompletionTestDir(t)

	checker := NewCompletionChecker(dir)
	_, err := checker.CheckGoal("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent goal")
	}
}

func TestCheckGoal_MixedCaseCheckboxes(t *testing.T) {
	dir := setupCompletionTestDir(t)

	goalContent := `# Goal #mixed: Test Goal

## Phases

### Phase 1: Test
- [X] Uppercase X checkbox
- [x] Lowercase x checkbox

## Acceptance Criteria

- [X] Criterion with uppercase X
`
	writeGoalFile(t, dir, "mixed", goalContent)

	checker := NewCompletionChecker(dir)
	status, err := checker.CheckGoal("mixed")
	if err != nil {
		t.Fatalf("CheckGoal failed: %v", err)
	}

	if !status.Complete {
		t.Error("expected goal to be complete (both X and x should count)")
	}
}

// ============================================================================
// Signal Types Tests
// ============================================================================

func TestSignalTypes(t *testing.T) {
	tests := []struct {
		signal   CompletionSignalType
		expected string
	}{
		{SignalGoalPhases, "goal_phases"},
		{SignalAcceptance, "acceptance"},
		{SignalCommit, "commit"},
	}

	for _, tt := range tests {
		if string(tt.signal) != tt.expected {
			t.Errorf("expected signal type %s, got %s", tt.expected, tt.signal)
		}
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{9, "9"},
		{10, "10"},
		{42, "42"},
		{123, "123"},
	}

	for _, tt := range tests {
		result := itoa(tt.input)
		if result != tt.expected {
			t.Errorf("itoa(%d): expected %s, got %s", tt.input, tt.expected, result)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d): expected %q, got %q", tt.input, tt.maxLen, tt.expected, result)
		}
	}
}

// ============================================================================
// findGoalFile Tests
// ============================================================================

func TestFindGoalFile_FlatStructure(t *testing.T) {
	dir := setupCompletionTestDir(t)

	goalContent := `# Goal #flat1: Test`
	writeGoalFile(t, dir, "flat1", goalContent)

	checker := NewCompletionChecker(dir)
	path := checker.findGoalFile("flat1")

	expected := filepath.Join(dir, "goals", "active", "flat1.md")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestFindGoalFile_FolderStructure(t *testing.T) {
	dir := setupCompletionTestDir(t)

	goalContent := `# Goal #folder2: Test`
	writeGoalFileInFolder(t, dir, "folder2", goalContent)

	checker := NewCompletionChecker(dir)
	path := checker.findGoalFile("folder2")

	expected := filepath.Join(dir, "goals", "active", "folder2", "folder2.md")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestFindGoalFile_NotFound(t *testing.T) {
	dir := setupCompletionTestDir(t)

	checker := NewCompletionChecker(dir)
	path := checker.findGoalFile("nonexistent")

	if path != "" {
		t.Errorf("expected empty string for nonexistent goal, got %s", path)
	}
}
