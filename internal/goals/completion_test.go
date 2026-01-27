package goals

import (
	"os"
	"path/filepath"
	"testing"
)

// setupCompletionTestDir creates a temp directory with task plan structure
func setupCompletionTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create planning directories
	os.MkdirAll(filepath.Join(dir, "docs", "planning", "history"), 0755)
	os.MkdirAll(filepath.Join(dir, "docs", "planning"), 0755)

	return dir
}

func writeTaskPlan(t *testing.T, dir, goalID, content string) {
	t.Helper()
	planDir := filepath.Join(dir, "docs", "planning", "history", "goal-"+goalID)
	os.MkdirAll(planDir, 0755)
	path := filepath.Join(planDir, "task_plan.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write task plan: %v", err)
	}
}

func TestIsGoalComplete_AllPhasesComplete(t *testing.T) {
	dir := setupCompletionTestDir(t)

	taskPlan := `# Task Plan: Test Goal

## Phase 1: Setup [complete]

### Tasks
- [x] Initialize project
- [x] Configure settings

## Phase 2: Implementation [complete]

### Tasks
- [x] Implement feature A
- [x] Implement feature B
- [x] Write tests
`
	writeTaskPlan(t, dir, "abc1234", taskPlan)

	status, err := IsGoalComplete("abc1234", dir)
	if err != nil {
		t.Fatalf("IsGoalComplete failed: %v", err)
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
}

func TestIsGoalComplete_PartiallyComplete(t *testing.T) {
	dir := setupCompletionTestDir(t)

	taskPlan := `# Task Plan: Test Goal

## Phase 1: Setup [complete]

### Tasks
- [x] Initialize project
- [x] Configure settings

## Phase 2: Implementation [in_progress]

### Tasks
- [x] Implement feature A
- [ ] Implement feature B
- [ ] Write tests

## Phase 3: Polish

### Tasks
- [ ] Add documentation
- [ ] Performance tuning
`
	writeTaskPlan(t, dir, "def5678", taskPlan)

	status, err := IsGoalComplete("def5678", dir)
	if err != nil {
		t.Fatalf("IsGoalComplete failed: %v", err)
	}

	if status.Complete {
		t.Error("expected goal to be incomplete")
	}
	if status.TotalPhases != 3 {
		t.Errorf("expected 3 total phases, got %d", status.TotalPhases)
	}
	if status.CompletedPhases != 1 {
		t.Errorf("expected 1 completed phase, got %d", status.CompletedPhases)
	}
	if len(status.MissingTasks) != 4 {
		t.Errorf("expected 4 missing tasks, got %d: %v", len(status.MissingTasks), status.MissingTasks)
	}

	// Check that missing tasks include phase context
	foundPhase2Task := false
	foundPhase3Task := false
	for _, task := range status.MissingTasks {
		if task == "Implementation: Implement feature B" {
			foundPhase2Task = true
		}
		if task == "Polish: Add documentation" {
			foundPhase3Task = true
		}
	}
	if !foundPhase2Task {
		t.Error("expected to find 'Implementation: Implement feature B' in missing tasks")
	}
	if !foundPhase3Task {
		t.Error("expected to find 'Polish: Add documentation' in missing tasks")
	}
}

func TestIsGoalComplete_NoPhases(t *testing.T) {
	dir := setupCompletionTestDir(t)

	taskPlan := `# Task Plan: Test Goal

## Overview

Just some overview text without phases.

## Notes

Some notes here.
`
	writeTaskPlan(t, dir, "nophase", taskPlan)

	status, err := IsGoalComplete("nophase", dir)
	if err != nil {
		t.Fatalf("IsGoalComplete failed: %v", err)
	}

	if status.Complete {
		t.Error("expected goal with no phases to be incomplete")
	}
	if status.TotalPhases != 0 {
		t.Errorf("expected 0 phases, got %d", status.TotalPhases)
	}
}

func TestIsGoalComplete_NotFound(t *testing.T) {
	dir := setupCompletionTestDir(t)

	_, err := IsGoalComplete("nonexistent", dir)
	if err == nil {
		t.Error("expected error for nonexistent task plan")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist, got %v", err)
	}
}

func TestIsGoalComplete_HashStyleHeaders(t *testing.T) {
	dir := setupCompletionTestDir(t)

	// Test with ### Phase headers (3 hashes)
	taskPlan := `# Task Plan

## Phases

### Phase 1: Foundation
- [x] Task one
- [x] Task two

### Phase 2: Building
- [x] Task three
- [ ] Task four
`
	writeTaskPlan(t, dir, "hashtest", taskPlan)

	status, err := IsGoalComplete("hashtest", dir)
	if err != nil {
		t.Fatalf("IsGoalComplete failed: %v", err)
	}

	if status.TotalPhases != 2 {
		t.Errorf("expected 2 phases, got %d", status.TotalPhases)
	}
	if status.CompletedPhases != 1 {
		t.Errorf("expected 1 completed phase, got %d", status.CompletedPhases)
	}
}

func TestIsGoalComplete_MixedCaseCheckboxes(t *testing.T) {
	dir := setupCompletionTestDir(t)

	taskPlan := `# Task Plan

## Phase 1: Test
- [X] Uppercase X checkbox
- [x] Lowercase x checkbox
- [ ] Incomplete task
`
	writeTaskPlan(t, dir, "mixedcase", taskPlan)

	status, err := IsGoalComplete("mixedcase", dir)
	if err != nil {
		t.Fatalf("IsGoalComplete failed: %v", err)
	}

	if status.Complete {
		t.Error("expected incomplete due to one incomplete task")
	}
	if len(status.MissingTasks) != 1 {
		t.Errorf("expected 1 missing task, got %d", len(status.MissingTasks))
	}
}

func TestIsGoalComplete_EmptyPhases(t *testing.T) {
	dir := setupCompletionTestDir(t)

	// Phase with no tasks should not count as complete
	taskPlan := `# Task Plan

## Phase 1: Empty Phase

Just some text, no tasks.

## Phase 2: Has Tasks
- [x] One task
`
	writeTaskPlan(t, dir, "emptyphase", taskPlan)

	status, err := IsGoalComplete("emptyphase", dir)
	if err != nil {
		t.Fatalf("IsGoalComplete failed: %v", err)
	}

	// Phase 1 has no tasks, so it shouldn't count as completed
	// Phase 2 has all tasks complete
	if status.TotalPhases != 2 {
		t.Errorf("expected 2 total phases, got %d", status.TotalPhases)
	}
	if status.CompletedPhases != 1 {
		t.Errorf("expected 1 completed phase (only phase with tasks), got %d", status.CompletedPhases)
	}
}

func TestCompletionChecker_CheckGoal(t *testing.T) {
	dir := setupCompletionTestDir(t)

	taskPlan := `# Task Plan

## Phase 1: Test [complete]
- [x] Task one
`
	writeTaskPlan(t, dir, "checker1", taskPlan)

	checker := NewCompletionChecker(dir)
	status, err := checker.CheckGoal("checker1")
	if err != nil {
		t.Fatalf("CheckGoal failed: %v", err)
	}

	if !status.Complete {
		t.Error("expected goal to be complete")
	}
}

func TestCompletionChecker_IsComplete(t *testing.T) {
	dir := setupCompletionTestDir(t)

	taskPlan := `# Task Plan

## Phase 1: Test
- [x] Task one
`
	writeTaskPlan(t, dir, "checker2", taskPlan)

	checker := NewCompletionChecker(dir)
	complete, err := checker.IsComplete("checker2")
	if err != nil {
		t.Fatalf("IsComplete failed: %v", err)
	}

	if !complete {
		t.Error("expected goal to be complete")
	}
}

func TestFindTaskPlan_ActiveLocation(t *testing.T) {
	dir := setupCompletionTestDir(t)

	// Create task plan in active planning location
	activeDir := filepath.Join(dir, "docs", "planning", "goal-active1")
	os.MkdirAll(activeDir, 0755)
	taskPlanPath := filepath.Join(activeDir, "task_plan.md")
	os.WriteFile(taskPlanPath, []byte("## Phase 1: Test\n- [x] Done"), 0644)

	path, err := findTaskPlan("active1", dir)
	if err != nil {
		t.Fatalf("findTaskPlan failed: %v", err)
	}

	if path != taskPlanPath {
		t.Errorf("expected path %s, got %s", taskPlanPath, path)
	}
}

func TestParseTaskPlanCompletion_RealWorldFormat(t *testing.T) {
	dir := setupCompletionTestDir(t)

	// Test with format from actual task_plan.md files
	taskPlan := `# Task Plan: Mobile-first UI Redesign

**Goal:** #97ade68
**Project:** vega-hub
**Started:** 2026-01-24

---

## Phase 1: Foundation [complete]

### Tasks
- [x] Install shadcn components
- [x] Create Layout component
- [x] Set up React Router

### Acceptance Criteria
- [x] App has bottom nav on mobile
- [x] Build compiles successfully

---

## Phase 2: Core Views [complete]

### Tasks
- [x] Home dashboard
- [x] Goals list with filters
- [x] GoalSheet.tsx

---

## Phase 3: Projects & History [complete]

### Tasks
- [x] Projects view with ProjectCard
- [x] ProjectSheet.tsx with tabs
- [x] History view with search
- [ ] New API endpoints (backend) - deferred

---

## Phase 4: Polish [complete]

### Tasks
- [x] Command palette (Cmd+K)
- [x] Toast notifications via SSE
- [x] Loading states (Skeleton)
`
	writeTaskPlan(t, dir, "97ade68", taskPlan)

	status, err := IsGoalComplete("97ade68", dir)
	if err != nil {
		t.Fatalf("IsGoalComplete failed: %v", err)
	}

	if status.TotalPhases != 4 {
		t.Errorf("expected 4 phases, got %d", status.TotalPhases)
	}

	// Phase 3 has one incomplete task despite [complete] marker
	// The checkbox parsing should take precedence
	if status.Complete {
		t.Error("expected incomplete due to deferred API endpoints task")
	}

	if len(status.MissingTasks) != 1 {
		t.Errorf("expected 1 missing task, got %d: %v", len(status.MissingTasks), status.MissingTasks)
	}
}
