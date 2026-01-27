package goals

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestDependencies creates a test vega-missile directory with goals
func setupTestDependencies(t *testing.T) (string, func()) {
	t.Helper()

	// Create temp directory
	dir, err := os.MkdirTemp("", "vega-deps-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Create directory structure
	dirs := []string{
		filepath.Join(dir, "goals", "active"),
		filepath.Join(dir, "goals", "iced"),
		filepath.Join(dir, "goals", "history"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", d, err)
		}
	}

	// Create test goals
	goals := map[string]string{
		"goal-a": "# Goal #goal-a: Build Foundation\n\n**Status**: Active\n",
		"goal-b": "# Goal #goal-b: Add Features\n\n**Status**: Active\n",
		"goal-c": "# Goal #goal-c: Polish UI\n\n**Status**: Active\n",
		"goal-d": "# Goal #goal-d: Completed Task\n\n**Status**: Completed\n",
	}

	for id, content := range goals {
		var goalDir string
		if id == "goal-d" {
			goalDir = filepath.Join(dir, "goals", "history")
		} else {
			goalDir = filepath.Join(dir, "goals", "active")
		}
		if err := os.WriteFile(filepath.Join(goalDir, id+".md"), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create goal file %s: %v", id, err)
		}
	}

	// Create a basic REGISTRY.md for the parser
	registry := `# Goal Registry

## Active Goals

| ID | Title | Project(s) | Status | Phase |
|---|---|---|---|---|
| goal-a | Build Foundation | test-proj | Active | 1/2 |
| goal-b | Add Features | test-proj | Active | 1/2 |
| goal-c | Polish UI | test-proj | Active | 1/2 |

## Iced Goals

| ID | Title | Project(s) | Reason |
|---|---|---|---|

## Completed Goals

| ID | Title | Project(s) | Completed |
|---|---|---|---|
| goal-d | Completed Task | test-proj | 2024-01-15 |
`
	if err := os.WriteFile(filepath.Join(dir, "goals", "REGISTRY.md"), []byte(registry), 0644); err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(dir)
	}

	return dir, cleanup
}

func TestAddDependency(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	// Test adding a valid dependency
	err := dm.AddDependency("goal-b", "goal-a", DependencyBlocks)
	if err != nil {
		t.Errorf("AddDependency failed: %v", err)
	}

	// Verify dependency was added
	info, err := dm.GetDependencies("goal-b")
	if err != nil {
		t.Errorf("GetDependencies failed: %v", err)
	}
	if len(info.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency, got %d", len(info.Dependencies))
	}
	if info.Dependencies[0].GoalID != "goal-a" {
		t.Errorf("Expected dependency on goal-a, got %s", info.Dependencies[0].GoalID)
	}
	if info.Dependencies[0].Type != DependencyBlocks {
		t.Errorf("Expected type 'blocks', got %s", info.Dependencies[0].Type)
	}
}

func TestAddDependencyIdempotent(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	// Add same dependency twice
	dm.AddDependency("goal-b", "goal-a", DependencyBlocks)
	err := dm.AddDependency("goal-b", "goal-a", DependencyBlocks)
	if err != nil {
		t.Errorf("Adding same dependency twice should not error: %v", err)
	}

	// Verify only one dependency exists
	info, _ := dm.GetDependencies("goal-b")
	if len(info.Dependencies) != 1 {
		t.Errorf("Expected 1 dependency after duplicate add, got %d", len(info.Dependencies))
	}
}

func TestSelfDependency(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	err := dm.AddDependency("goal-a", "goal-a", DependencyBlocks)
	if err == nil {
		t.Error("Self-dependency should be rejected")
	}
}

func TestCircularDependency(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	// Create A -> B
	dm.AddDependency("goal-a", "goal-b", DependencyBlocks)

	// Try to create B -> A (would create cycle)
	err := dm.AddDependency("goal-b", "goal-a", DependencyBlocks)
	if err == nil {
		t.Error("Circular dependency should be rejected")
	}
}

func TestCircularDependencyTransitive(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	// Create A -> B -> C
	dm.AddDependency("goal-a", "goal-b", DependencyBlocks)
	dm.AddDependency("goal-b", "goal-c", DependencyBlocks)

	// Try to create C -> A (would create cycle through transitivity)
	err := dm.AddDependency("goal-c", "goal-a", DependencyBlocks)
	if err == nil {
		t.Error("Transitive circular dependency should be rejected")
	}
}

func TestRelatedDependencyNoCircularCheck(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	// Create bidirectional related relationships (allowed)
	dm.AddDependency("goal-a", "goal-b", DependencyRelated)
	err := dm.AddDependency("goal-b", "goal-a", DependencyRelated)
	if err != nil {
		t.Error("Related dependencies should allow bidirectional relationships")
	}
}

func TestRemoveDependency(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	// Add then remove
	dm.AddDependency("goal-b", "goal-a", DependencyBlocks)
	err := dm.RemoveDependency("goal-b", "goal-a")
	if err != nil {
		t.Errorf("RemoveDependency failed: %v", err)
	}

	// Verify removal
	info, _ := dm.GetDependencies("goal-b")
	if len(info.Dependencies) != 0 {
		t.Errorf("Expected 0 dependencies after removal, got %d", len(info.Dependencies))
	}
}

func TestRemoveNonexistentDependency(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	err := dm.RemoveDependency("goal-b", "goal-a")
	if err == nil {
		t.Error("Removing nonexistent dependency should error")
	}
}

func TestGetDependents(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	// Create B -> A and C -> A
	dm.AddDependency("goal-b", "goal-a", DependencyBlocks)
	dm.AddDependency("goal-c", "goal-a", DependencyBlocks)

	// Get dependents of A (goals that depend on A)
	info, err := dm.GetDependencies("goal-a")
	if err != nil {
		t.Errorf("GetDependencies failed: %v", err)
	}
	if len(info.Dependents) != 2 {
		t.Errorf("Expected 2 dependents, got %d", len(info.Dependents))
	}
}

func TestGetBlockers(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	// B depends on A (blocks) and D (completed)
	dm.AddDependency("goal-b", "goal-a", DependencyBlocks)
	dm.AddDependency("goal-b", "goal-d", DependencyBlocks)

	blockers, err := dm.GetBlockers("goal-b")
	if err != nil {
		t.Errorf("GetBlockers failed: %v", err)
	}

	// Only goal-a should block (goal-d is completed)
	if len(blockers) != 1 {
		t.Errorf("Expected 1 blocker (completed goals don't block), got %d", len(blockers))
	}
	if len(blockers) > 0 && blockers[0].ID != "goal-a" {
		t.Errorf("Expected blocker goal-a, got %s", blockers[0].ID)
	}
}

func TestIsBlocked(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	// Initially not blocked
	if dm.IsBlocked("goal-b") {
		t.Error("goal-b should not be blocked initially")
	}

	// Add blocking dependency
	dm.AddDependency("goal-b", "goal-a", DependencyBlocks)

	// Now should be blocked
	if !dm.IsBlocked("goal-b") {
		t.Error("goal-b should be blocked after adding dependency")
	}

	// Related dependency should not block
	dm.RemoveDependency("goal-b", "goal-a")
	dm.AddDependency("goal-b", "goal-a", DependencyRelated)

	if dm.IsBlocked("goal-b") {
		t.Error("goal-b should not be blocked by related dependency")
	}
}

func TestGetReadyGoals(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	// Create state files for goals to make them "working"
	sm := NewStateManager(dir)
	sm.ForceState("goal-a", StateWorking, "test")
	sm.ForceState("goal-b", StateWorking, "test")
	sm.ForceState("goal-c", StateWorking, "test")

	// Initially all active goals should be ready
	ready, err := dm.GetReadyGoals("")
	if err != nil {
		t.Errorf("GetReadyGoals failed: %v", err)
	}
	if len(ready) != 3 {
		t.Errorf("Expected 3 ready goals initially, got %d", len(ready))
	}

	// Block goal-b with goal-a
	dm.AddDependency("goal-b", "goal-a", DependencyBlocks)

	ready, _ = dm.GetReadyGoals("")
	if len(ready) != 2 {
		t.Errorf("Expected 2 ready goals after blocking goal-b, got %d", len(ready))
	}

	// Verify goal-b is not in ready list
	for _, g := range ready {
		if g.ID == "goal-b" {
			t.Error("goal-b should not be in ready list while blocked")
		}
	}
}

func TestGetDependencyTree(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	// Create chain: C -> B -> A
	dm.AddDependency("goal-b", "goal-a", DependencyBlocks)
	dm.AddDependency("goal-c", "goal-b", DependencyBlocks)

	tree, err := dm.GetDependencyTree("goal-c")
	if err != nil {
		t.Errorf("GetDependencyTree failed: %v", err)
	}

	if tree.GoalID != "goal-c" {
		t.Errorf("Expected root goal-c, got %s", tree.GoalID)
	}
	if len(tree.Children) != 1 {
		t.Errorf("Expected 1 child (goal-b), got %d", len(tree.Children))
	}
	if tree.Children[0].GoalID != "goal-b" {
		t.Errorf("Expected child goal-b, got %s", tree.Children[0].GoalID)
	}
	if len(tree.Children[0].Children) != 1 {
		t.Errorf("Expected goal-b to have 1 child (goal-a), got %d", len(tree.Children[0].Children))
	}
}

func TestNonexistentGoal(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	// Try to add dependency on nonexistent goal
	err := dm.AddDependency("goal-a", "nonexistent", DependencyBlocks)
	if err == nil {
		t.Error("Adding dependency on nonexistent goal should error")
	}

	err = dm.AddDependency("nonexistent", "goal-a", DependencyBlocks)
	if err == nil {
		t.Error("Adding dependency from nonexistent goal should error")
	}
}

func TestInvalidDependencyType(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	err := dm.AddDependency("goal-a", "goal-b", DependencyType("invalid"))
	if err == nil {
		t.Error("Invalid dependency type should be rejected")
	}
}

func TestGoalsWithBlockerInfo(t *testing.T) {
	dir, cleanup := setupTestDependencies(t)
	defer cleanup()

	dm := NewDependencyManager(dir)

	// Block goal-b with goal-a
	dm.AddDependency("goal-b", "goal-a", DependencyBlocks)

	goals, err := dm.GetGoalsWithBlockerInfo("")
	if err != nil {
		t.Errorf("GetGoalsWithBlockerInfo failed: %v", err)
	}

	// Find goal-b in results
	var goalB *GoalWithBlockers
	for i, g := range goals {
		if g.ID == "goal-b" {
			goalB = &goals[i]
			break
		}
	}

	if goalB == nil {
		t.Fatal("goal-b not found in results")
	}

	if !goalB.IsBlocked {
		t.Error("goal-b should be marked as blocked")
	}
	if len(goalB.BlockedBy) != 1 || goalB.BlockedBy[0] != "goal-a" {
		t.Errorf("goal-b should be blocked by goal-a, got %v", goalB.BlockedBy)
	}
}
