package hub

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGoalFolder(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create goals/active directory
	goalsDir := filepath.Join(tmpDir, "goals", "active")
	if err := os.MkdirAll(goalsDir, 0755); err != nil {
		t.Fatal(err)
	}

	h := &Hub{dir: tmpDir}

	t.Run("finds existing folder structure", func(t *testing.T) {
		// Create goal folder
		goalID := "abc123"
		goalFolder := filepath.Join(goalsDir, goalID)
		if err := os.MkdirAll(goalFolder, 0755); err != nil {
			t.Fatal(err)
		}

		found, err := h.findGoalFolder(goalID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if found != goalFolder {
			t.Errorf("expected %s, got %s", goalFolder, found)
		}
	})

	t.Run("creates folder for flat file goal", func(t *testing.T) {
		// Create flat goal file
		goalID := "flat123"
		flatFile := filepath.Join(goalsDir, goalID+".md")
		if err := os.WriteFile(flatFile, []byte("# Test Goal"), 0644); err != nil {
			t.Fatal(err)
		}

		found, err := h.findGoalFolder(goalID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		expectedFolder := filepath.Join(goalsDir, goalID)
		if found != expectedFolder {
			t.Errorf("expected %s, got %s", expectedFolder, found)
		}

		// Verify folder was created
		if info, err := os.Stat(found); err != nil || !info.IsDir() {
			t.Errorf("folder was not created: %v", err)
		}
	})

	t.Run("returns error for non-existent goal", func(t *testing.T) {
		_, err := h.findGoalFolder("nonexistent")
		if err == nil {
			t.Error("expected error for non-existent goal")
		}
	})
}

func TestFindWorktreeForProject(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create workspaces directory
	workspacesDir := filepath.Join(tmpDir, "workspaces")
	if err := os.MkdirAll(workspacesDir, 0755); err != nil {
		t.Fatal(err)
	}

	h := &Hub{dir: tmpDir}

	t.Run("finds worktree for goal in project", func(t *testing.T) {
		// Create project with worktree
		project := "my-api"
		goalID := "abc123"
		worktreeName := "goal-" + goalID + "-some-title"
		worktreePath := filepath.Join(workspacesDir, project, worktreeName)
		if err := os.MkdirAll(worktreePath, 0755); err != nil {
			t.Fatal(err)
		}

		found, err := h.findWorktreeForProject(goalID, project)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if found != worktreePath {
			t.Errorf("expected %s, got %s", worktreePath, found)
		}
	})

	t.Run("returns error for non-existent project", func(t *testing.T) {
		_, err := h.findWorktreeForProject("abc123", "nonexistent")
		if err == nil {
			t.Error("expected error for non-existent project")
		}
	})

	t.Run("returns error for goal not in project", func(t *testing.T) {
		// Create empty project
		project := "empty-project"
		if err := os.MkdirAll(filepath.Join(workspacesDir, project), 0755); err != nil {
			t.Fatal(err)
		}

		_, err := h.findWorktreeForProject("xyz789", project)
		if err == nil {
			t.Error("expected error when goal worktree not found")
		}
	})
}

func TestSpawnRequestValidation(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	h := &Hub{dir: tmpDir}

	t.Run("meta and project are mutually exclusive", func(t *testing.T) {
		result := h.SpawnExecutor(SpawnRequest{
			GoalID:  "test123",
			Meta:    true,
			Project: "some-project",
		})

		if result.Success {
			t.Error("expected failure when both meta and project are set")
		}
		if result.Message != "--meta and --project are mutually exclusive" {
			t.Errorf("unexpected message: %s", result.Message)
		}
	})
}

func TestValidModes(t *testing.T) {
	validModes := []string{"plan", "implement", "review", "test", "security", "quick"}
	invalidModes := []string{"", "invalid", "PLAN", "Plan"}

	for _, mode := range validModes {
		if !ValidModes[mode] {
			t.Errorf("expected %q to be valid", mode)
		}
	}

	for _, mode := range invalidModes {
		if ValidModes[mode] {
			t.Errorf("expected %q to be invalid", mode)
		}
	}
}

func TestValidExecutorTypes(t *testing.T) {
	validTypes := []string{"meta", "project", "manager"}
	invalidTypes := []string{"", "invalid", "META", "Project"}

	for _, typ := range validTypes {
		if !ValidExecutorTypes[typ] {
			t.Errorf("expected %q to be valid", typ)
		}
	}

	for _, typ := range invalidTypes {
		if ValidExecutorTypes[typ] {
			t.Errorf("expected %q to be invalid", typ)
		}
	}
}
