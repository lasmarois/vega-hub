package goals

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestDir creates a temporary test directory with necessary structure
func setupHierarchyTestDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "hierarchy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create goals directories
	os.MkdirAll(filepath.Join(dir, "goals", "active"), 0755)
	os.MkdirAll(filepath.Join(dir, "goals", "iced"), 0755)
	os.MkdirAll(filepath.Join(dir, "goals", "history"), 0755)

	return dir
}

// createTestGoal creates a minimal goal file for testing
func createTestGoal(t *testing.T, dir, goalID, status string) {
	goalContent := `# Goal ` + goalID + `: Test Goal

## Overview
Test goal for hierarchy testing.

## Status
**Status**: ` + status + `
`
	goalFile := filepath.Join(dir, "goals", status, goalID+".md")
	// Handle status mapping
	if status == "active" || status == "Active" {
		goalFile = filepath.Join(dir, "goals", "active", goalID+".md")
	} else if status == "iced" || status == "Iced" {
		goalFile = filepath.Join(dir, "goals", "iced", goalID+".md")
	} else if status == "completed" || status == "Completed" {
		goalFile = filepath.Join(dir, "goals", "history", goalID+".md")
	}
	
	if err := os.WriteFile(goalFile, []byte(goalContent), 0644); err != nil {
		t.Fatalf("Failed to create test goal: %v", err)
	}
}

func TestHierarchyDepth(t *testing.T) {
	dir := setupHierarchyTestDir(t)
	defer os.RemoveAll(dir)

	hm := NewHierarchyManager(dir)

	tests := []struct {
		goalID string
		depth  int
	}{
		{"abc123", 0},        // Root goal
		{"abc123.1", 1},      // First level child
		{"abc123.2", 1},      // First level child
		{"abc123.1.1", 2},    // Second level
		{"abc123.1.2", 2},    // Second level
		{"abc123.1.1.1", 3},  // Third level (max)
	}

	for _, tt := range tests {
		t.Run(tt.goalID, func(t *testing.T) {
			depth := hm.GetHierarchyDepth(tt.goalID)
			if depth != tt.depth {
				t.Errorf("GetHierarchyDepth(%q) = %d, want %d", tt.goalID, depth, tt.depth)
			}
		})
	}
}

func TestGenerateChildID(t *testing.T) {
	dir := setupHierarchyTestDir(t)
	defer os.RemoveAll(dir)

	hm := NewHierarchyManager(dir)

	// Create a parent goal
	parentID := "abc1234"
	createTestGoal(t, dir, parentID, "active")

	// Generate first child
	childID1, err := hm.GenerateChildID(parentID)
	if err != nil {
		t.Fatalf("GenerateChildID failed: %v", err)
	}
	if childID1 != parentID+".1" {
		t.Errorf("First child ID = %q, want %q", childID1, parentID+".1")
	}

	// Generate second child
	childID2, err := hm.GenerateChildID(parentID)
	if err != nil {
		t.Fatalf("GenerateChildID failed: %v", err)
	}
	if childID2 != parentID+".2" {
		t.Errorf("Second child ID = %q, want %q", childID2, parentID+".2")
	}

	// Generate third child
	childID3, err := hm.GenerateChildID(parentID)
	if err != nil {
		t.Fatalf("GenerateChildID failed: %v", err)
	}
	if childID3 != parentID+".3" {
		t.Errorf("Third child ID = %q, want %q", childID3, parentID+".3")
	}
}

func TestNestedHierarchy(t *testing.T) {
	dir := setupHierarchyTestDir(t)
	defer os.RemoveAll(dir)

	hm := NewHierarchyManager(dir)

	// Create root goal
	rootID := "abc1234"
	createTestGoal(t, dir, rootID, "active")

	// Create first level child
	child1ID, err := hm.GenerateChildID(rootID)
	if err != nil {
		t.Fatalf("GenerateChildID for child1 failed: %v", err)
	}
	createTestGoal(t, dir, child1ID, "active")

	// Create second level child (grandchild)
	grandchildID, err := hm.GenerateChildID(child1ID)
	if err != nil {
		t.Fatalf("GenerateChildID for grandchild failed: %v", err)
	}
	createTestGoal(t, dir, grandchildID, "active")

	// Verify IDs
	if child1ID != rootID+".1" {
		t.Errorf("Child1 ID = %q, want %q", child1ID, rootID+".1")
	}
	if grandchildID != rootID+".1.1" {
		t.Errorf("Grandchild ID = %q, want %q", grandchildID, rootID+".1.1")
	}

	// Create another child at first level
	child2ID, err := hm.GenerateChildID(rootID)
	if err != nil {
		t.Fatalf("GenerateChildID for child2 failed: %v", err)
	}
	if child2ID != rootID+".2" {
		t.Errorf("Child2 ID = %q, want %q", child2ID, rootID+".2")
	}
}

func TestMaxDepthLimit(t *testing.T) {
	dir := setupHierarchyTestDir(t)
	defer os.RemoveAll(dir)

	hm := NewHierarchyManager(dir)

	// Create chain to max depth
	currentID := "abc1234"
	createTestGoal(t, dir, currentID, "active")

	for depth := 0; depth < MaxHierarchyDepth; depth++ {
		childID, err := hm.GenerateChildID(currentID)
		if err != nil {
			t.Fatalf("GenerateChildID at depth %d failed: %v", depth, err)
		}
		createTestGoal(t, dir, childID, "active")
		currentID = childID
	}

	// Try to create one more (should fail)
	_, err := hm.GenerateChildID(currentID)
	if err == nil {
		t.Error("Expected error when exceeding max depth, got nil")
	}
}

func TestParentChildRelationship(t *testing.T) {
	dir := setupHierarchyTestDir(t)
	defer os.RemoveAll(dir)

	hm := NewHierarchyManager(dir)

	// Create parent
	parentID := "abc1234"
	createTestGoal(t, dir, parentID, "active")

	// Create child
	childID, _ := hm.GenerateChildID(parentID)
	createTestGoal(t, dir, childID, "active")

	// Set up hierarchy
	if err := hm.CreateChildGoal(childID, parentID); err != nil {
		t.Fatalf("CreateChildGoal failed: %v", err)
	}

	// Verify parent relationship
	gotParent, err := hm.GetParentID(childID)
	if err != nil {
		t.Fatalf("GetParentID failed: %v", err)
	}
	if gotParent != parentID {
		t.Errorf("GetParentID(%q) = %q, want %q", childID, gotParent, parentID)
	}

	// Verify children
	children, err := hm.GetChildren(parentID)
	if err != nil {
		t.Fatalf("GetChildren failed: %v", err)
	}
	if len(children) != 1 || children[0] != childID {
		t.Errorf("GetChildren(%q) = %v, want [%q]", parentID, children, childID)
	}
}

func TestValidateParentForChildCreation(t *testing.T) {
	dir := setupHierarchyTestDir(t)
	defer os.RemoveAll(dir)

	hm := NewHierarchyManager(dir)

	// Test: non-existent parent
	err := hm.ValidateParentForChildCreation("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent parent")
	}

	// Test: completed parent
	completedID := "comp123"
	createTestGoal(t, dir, completedID, "completed")
	err = hm.ValidateParentForChildCreation(completedID)
	if err == nil {
		t.Error("Expected error for completed parent")
	}

	// Test: active parent (should succeed)
	activeID := "active1"
	createTestGoal(t, dir, activeID, "active")
	err = hm.ValidateParentForChildCreation(activeID)
	if err != nil {
		t.Errorf("ValidateParentForChildCreation for active goal failed: %v", err)
	}

	// Test: iced parent (should succeed - iced goals can have children)
	icedID := "iced123"
	createTestGoal(t, dir, icedID, "iced")
	err = hm.ValidateParentForChildCreation(icedID)
	if err != nil {
		t.Errorf("ValidateParentForChildCreation for iced goal failed: %v", err)
	}
}

func TestCompareHierarchicalIDs(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int // -1 for a<b, 0 for equal, 1 for a>b
	}{
		{"abc1234", "abc1234", 0},
		{"abc1234", "bcd5678", -1},
		{"abc1234.1", "abc1234.2", -1},
		{"abc1234.2", "abc1234.1", 1},
		{"abc1234.1", "abc1234.10", -1},  // Numeric comparison
		{"abc1234.10", "abc1234.2", 1},   // 10 > 2
		{"abc1234", "abc1234.1", -1},      // Parent before child
		{"abc1234.1.1", "abc1234.1.2", -1},
		{"abc1234.1.2", "abc1234.2.1", -1}, // First index wins
	}

	for _, tt := range tests {
		t.Run(tt.a+" vs "+tt.b, func(t *testing.T) {
			result := compareHierarchicalIDs(tt.a, tt.b)
			if tt.expected < 0 && result >= 0 {
				t.Errorf("compareHierarchicalIDs(%q, %q) = %d, expected < 0", tt.a, tt.b, result)
			} else if tt.expected > 0 && result <= 0 {
				t.Errorf("compareHierarchicalIDs(%q, %q) = %d, expected > 0", tt.a, tt.b, result)
			} else if tt.expected == 0 && result != 0 {
				t.Errorf("compareHierarchicalIDs(%q, %q) = %d, expected 0", tt.a, tt.b, result)
			}
		})
	}
}

func TestIsHierarchicalID(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		{"abc1234", false},
		{"abc1234.1", true},
		{"abc1234.1.2", true},
		{"abc1234.1.2.3", true},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			result := IsHierarchicalID(tt.id)
			if result != tt.expected {
				t.Errorf("IsHierarchicalID(%q) = %v, want %v", tt.id, result, tt.expected)
			}
		})
	}
}

func TestGetRootID(t *testing.T) {
	tests := []struct {
		id       string
		expected string
	}{
		{"abc1234", "abc1234"},
		{"abc1234.1", "abc1234"},
		{"abc1234.1.2", "abc1234"},
		{"abc1234.1.2.3", "abc1234"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			result := GetRootID(tt.id)
			if result != tt.expected {
				t.Errorf("GetRootID(%q) = %q, want %q", tt.id, result, tt.expected)
			}
		})
	}
}

func TestGetParentIDFromHierarchical(t *testing.T) {
	tests := []struct {
		id       string
		expected string
	}{
		{"abc1234", ""},
		{"abc1234.1", "abc1234"},
		{"abc1234.1.2", "abc1234.1"},
		{"abc1234.1.2.3", "abc1234.1.2"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			result := GetParentIDFromHierarchical(tt.id)
			if result != tt.expected {
				t.Errorf("GetParentIDFromHierarchical(%q) = %q, want %q", tt.id, result, tt.expected)
			}
		})
	}
}

func TestParseHierarchicalID(t *testing.T) {
	tests := []struct {
		id         string
		baseID     string
		indicesLen int
	}{
		{"abc1234", "abc1234", 0},
		{"abc1234.1", "abc1234", 1},
		{"abc1234.1.2", "abc1234", 2},
		{"abc1234.1.2.3", "abc1234", 3},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			baseID, indices := ParseHierarchicalID(tt.id)
			if baseID != tt.baseID {
				t.Errorf("ParseHierarchicalID(%q) baseID = %q, want %q", tt.id, baseID, tt.baseID)
			}
			if len(indices) != tt.indicesLen {
				t.Errorf("ParseHierarchicalID(%q) indices len = %d, want %d", tt.id, len(indices), tt.indicesLen)
			}
		})
	}
}

func TestValidateHierarchicalID(t *testing.T) {
	tests := []struct {
		id      string
		wantErr bool
	}{
		{"abc1234", false},
		{"abc1234.1", false},
		{"abc1234.1.2", false},
		{"abc1234.1.2.3", false},
		{"abc1234.1.2.3.4", true},   // Too deep
		{"ABCDEFG", true},            // Not lowercase hex
		{"abc123", true},             // Too short
		{"abc12345", true},           // Too long
		{"abc1234.x", true},          // Non-numeric index
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			err := ValidateHierarchicalID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHierarchicalID(%q) error = %v, wantErr %v", tt.id, err, tt.wantErr)
			}
		})
	}
}

func TestTreeRendering(t *testing.T) {
	dir := setupHierarchyTestDir(t)
	defer os.RemoveAll(dir)

	hm := NewHierarchyManager(dir)

	// Create a simple tree:
	// abc1234 (root)
	// ├── abc1234.1
	// │   └── abc1234.1.1
	// └── abc1234.2

	rootID := "abc1234"
	createTestGoal(t, dir, rootID, "active")

	child1ID, _ := hm.GenerateChildID(rootID)
	createTestGoal(t, dir, child1ID, "active")
	hm.CreateChildGoal(child1ID, rootID)

	grandchildID, _ := hm.GenerateChildID(child1ID)
	createTestGoal(t, dir, grandchildID, "active")
	hm.CreateChildGoal(grandchildID, child1ID)

	child2ID, _ := hm.GenerateChildID(rootID)
	createTestGoal(t, dir, child2ID, "active")
	hm.CreateChildGoal(child2ID, rootID)

	// Create REGISTRY.md with goals
	registryContent := `# Goal Registry

## Active Goals

| ID | Title | Project(s) | Status | Phase |
|----|-------|------------|--------|-------|
| ` + rootID + ` | Root Goal | test | Active | 1/? |
| ` + child1ID + ` | Child 1 | test | Active | 1/? |
| ` + grandchildID + ` | Grandchild | test | Active | 1/? |
| ` + child2ID + ` | Child 2 | test | Active | 1/? |

## Iced Goals

| ID | Title | Project(s) | Reason |
|----|-------|------------|--------|

## Completed Goals

| ID | Title | Project(s) | Completed |
|----|-------|------------|-----------|
`
	registryPath := filepath.Join(dir, "goals", "REGISTRY.md")
	if err := os.WriteFile(registryPath, []byte(registryContent), 0644); err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Build and render tree
	output, err := hm.RenderTree("", "")
	if err != nil {
		t.Fatalf("RenderTree failed: %v", err)
	}

	// Just verify it produces output
	if output == "" {
		t.Error("RenderTree returned empty output")
	}

	// Verify structure contains all nodes
	if !containsSubstring(output, rootID) {
		t.Errorf("Tree output missing root: %s", rootID)
	}
	if !containsSubstring(output, child1ID) {
		t.Errorf("Tree output missing child1: %s", child1ID)
	}
	if !containsSubstring(output, grandchildID) {
		t.Errorf("Tree output missing grandchild: %s", grandchildID)
	}
	if !containsSubstring(output, child2ID) {
		t.Errorf("Tree output missing child2: %s", child2ID)
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstringHelper(s, sub))
}

func containsSubstringHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
