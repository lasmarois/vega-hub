package goals

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestDir creates a temporary directory with test registry and goal files
func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create directory structure
	os.MkdirAll(filepath.Join(dir, "goals", "active"), 0755)
	os.MkdirAll(filepath.Join(dir, "goals", "iced"), 0755)
	os.MkdirAll(filepath.Join(dir, "goals", "history"), 0755)

	return dir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

func TestParseRegistry_ActiveGoals(t *testing.T) {
	dir := setupTestDir(t)

	registry := `# Goal Registry

## Active Goals

| ID | Title | Project(s) | Status | Phase |
|----|-------|------------|--------|-------|
| abc1234 | Test goal one | project-a | Active | 1/3 |
| def5678 | Test goal two | project-b, project-c | Active | 2/4 |
| 14 | Legacy numeric goal | vega-missile | Active | 1/? |
| | | | | |

## Completed Goals

| ID | Title | Project(s) | Completed |
|----|-------|------------|-----------|
| xyz9999 | Old goal | project-a | 2026-01-01 |
`
	writeFile(t, filepath.Join(dir, "goals", "REGISTRY.md"), registry)

	parser := NewParser(dir)
	goals, err := parser.ParseRegistry()
	if err != nil {
		t.Fatalf("ParseRegistry failed: %v", err)
	}

	// Should have 4 goals total (3 active + 1 completed)
	if len(goals) != 4 {
		t.Errorf("expected 4 goals, got %d", len(goals))
	}

	// Check first active goal
	var found bool
	for _, g := range goals {
		if g.ID == "abc1234" {
			found = true
			if g.Title != "Test goal one" {
				t.Errorf("expected title 'Test goal one', got '%s'", g.Title)
			}
			if g.Status != "active" {
				t.Errorf("expected status 'active', got '%s'", g.Status)
			}
			if g.Phase != "1/3" {
				t.Errorf("expected phase '1/3', got '%s'", g.Phase)
			}
			if len(g.Projects) != 1 || g.Projects[0] != "project-a" {
				t.Errorf("expected projects [project-a], got %v", g.Projects)
			}
		}
	}
	if !found {
		t.Error("goal abc1234 not found")
	}

	// Check multi-project goal
	for _, g := range goals {
		if g.ID == "def5678" {
			if len(g.Projects) != 2 {
				t.Errorf("expected 2 projects, got %d", len(g.Projects))
			}
		}
	}

	// Check numeric ID support (legacy)
	for _, g := range goals {
		if g.ID == "14" {
			if g.Title != "Legacy numeric goal" {
				t.Errorf("expected legacy goal title, got '%s'", g.Title)
			}
		}
	}
}

func TestParseRegistry_IcedGoals(t *testing.T) {
	dir := setupTestDir(t)

	registry := `# Goal Registry

## Active Goals

| ID | Title | Project(s) | Status | Phase |
|----|-------|------------|--------|-------|
| | | | | |

## Iced Goals

| ID | Title | Project(s) | Reason |
|----|-------|------------|--------|
| ice1234 | Paused goal | project-x | Blocked on dependency |
| | | | |
`
	writeFile(t, filepath.Join(dir, "goals", "REGISTRY.md"), registry)

	parser := NewParser(dir)
	goals, err := parser.ParseRegistry()
	if err != nil {
		t.Fatalf("ParseRegistry failed: %v", err)
	}

	if len(goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(goals))
	}

	g := goals[0]
	if g.ID != "ice1234" {
		t.Errorf("expected ID 'ice1234', got '%s'", g.ID)
	}
	if g.Status != "iced" {
		t.Errorf("expected status 'iced', got '%s'", g.Status)
	}
	if g.Reason != "Blocked on dependency" {
		t.Errorf("expected reason 'Blocked on dependency', got '%s'", g.Reason)
	}
}

func TestParseRegistry_EmptyFile(t *testing.T) {
	dir := setupTestDir(t)
	writeFile(t, filepath.Join(dir, "goals", "REGISTRY.md"), "# Goal Registry\n")

	parser := NewParser(dir)
	goals, err := parser.ParseRegistry()
	if err != nil {
		t.Fatalf("ParseRegistry failed: %v", err)
	}

	if len(goals) != 0 {
		t.Errorf("expected 0 goals, got %d", len(goals))
	}
}

func TestParseRegistry_FileNotFound(t *testing.T) {
	dir := setupTestDir(t)
	// Don't create the registry file

	parser := NewParser(dir)
	_, err := parser.ParseRegistry()
	if err == nil {
		t.Error("expected error for missing registry file")
	}
}

func TestParseGoalDetail_BasicGoal(t *testing.T) {
	dir := setupTestDir(t)

	goalContent := `# Goal #abc1234: Test Goal Title

## Overview

This is a test goal for unit testing the parser.

## Project(s)

- **test-project**: Main project

## Phases

### Phase 1: Setup
- [x] Create directory structure
- [x] Initialize config
- [ ] Write documentation
- **Status:** in_progress

### Phase 2: Implementation
- [ ] Implement feature A
- [ ] Implement feature B
- **Status:** pending

## Acceptance Criteria

- [ ] All tests pass
- [ ] Documentation complete

## Status

**Current Phase**: 1
**Status**: Active
**Assigned To**: Executor

## Notes

- This is note one
- This is note two
`
	writeFile(t, filepath.Join(dir, "goals", "active", "abc1234.md"), goalContent)

	parser := NewParser(dir)
	detail, err := parser.ParseGoalDetail("abc1234")
	if err != nil {
		t.Fatalf("ParseGoalDetail failed: %v", err)
	}

	if detail.ID != "abc1234" {
		t.Errorf("expected ID 'abc1234', got '%s'", detail.ID)
	}
	if detail.Title != "Test Goal Title" {
		t.Errorf("expected title 'Test Goal Title', got '%s'", detail.Title)
	}
	if detail.Status != "active" {
		t.Errorf("expected status 'active', got '%s'", detail.Status)
	}

	// Check phases
	if len(detail.Phases) != 2 {
		t.Fatalf("expected 2 phases, got %d", len(detail.Phases))
	}

	phase1 := detail.Phases[0]
	if phase1.Number != 1 {
		t.Errorf("expected phase number 1, got %d", phase1.Number)
	}
	if phase1.Title != "Setup" {
		t.Errorf("expected phase title 'Setup', got '%s'", phase1.Title)
	}
	if len(phase1.Tasks) != 3 {
		t.Errorf("expected 3 tasks in phase 1, got %d", len(phase1.Tasks))
	}
	if phase1.Status != "in_progress" {
		t.Errorf("expected phase status 'in_progress', got '%s'", phase1.Status)
	}

	// Check task completion
	if !phase1.Tasks[0].Completed {
		t.Error("expected first task to be completed")
	}
	if phase1.Tasks[2].Completed {
		t.Error("expected third task to be incomplete")
	}

	// Check phase 2 is pending
	if detail.Phases[1].Status != "pending" {
		t.Errorf("expected phase 2 status 'pending', got '%s'", detail.Phases[1].Status)
	}

	// Check acceptance criteria
	if len(detail.Acceptance) != 2 {
		t.Errorf("expected 2 acceptance criteria, got %d", len(detail.Acceptance))
	}

	// Check notes
	if len(detail.Notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(detail.Notes))
	}

	// Check projects
	if len(detail.Projects) != 1 || detail.Projects[0] != "test-project" {
		t.Errorf("expected projects [test-project], got %v", detail.Projects)
	}
}

func TestParseGoalDetail_CompletedPhase(t *testing.T) {
	dir := setupTestDir(t)

	goalContent := `# Goal #abc1234: Completed Phase Test

## Phases

### Phase 1: Done
- [x] Task one
- [x] Task two
- **Status:** complete
`
	writeFile(t, filepath.Join(dir, "goals", "active", "abc1234.md"), goalContent)

	parser := NewParser(dir)
	detail, err := parser.ParseGoalDetail("abc1234")
	if err != nil {
		t.Fatalf("ParseGoalDetail failed: %v", err)
	}

	if len(detail.Phases) != 1 {
		t.Fatalf("expected 1 phase, got %d", len(detail.Phases))
	}

	if detail.Phases[0].Status != "complete" {
		t.Errorf("expected phase status 'complete', got '%s'", detail.Phases[0].Status)
	}
}

func TestParseGoalDetail_IcedGoal(t *testing.T) {
	dir := setupTestDir(t)

	goalContent := `# Goal #ice1234: Iced Goal

## Overview

This goal is on ice.
`
	writeFile(t, filepath.Join(dir, "goals", "iced", "ice1234.md"), goalContent)

	parser := NewParser(dir)
	detail, err := parser.ParseGoalDetail("ice1234")
	if err != nil {
		t.Fatalf("ParseGoalDetail failed: %v", err)
	}

	if detail.Status != "iced" {
		t.Errorf("expected status 'iced', got '%s'", detail.Status)
	}
}

func TestParseGoalDetail_NotFound(t *testing.T) {
	dir := setupTestDir(t)

	parser := NewParser(dir)
	_, err := parser.ParseGoalDetail("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent goal")
	}
}

func TestParseProjects(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"project-a", []string{"project-a"}},
		{"project-a, project-b", []string{"project-a", "project-b"}},
		{"  project-a  ,  project-b  ", []string{"project-a", "project-b"}},
		{"", []string{}},
		{"  ", []string{}},
	}

	for _, tt := range tests {
		result := parseProjects(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("parseProjects(%q): expected %v, got %v", tt.input, tt.expected, result)
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("parseProjects(%q)[%d]: expected %q, got %q", tt.input, i, tt.expected[i], result[i])
			}
		}
	}
}
