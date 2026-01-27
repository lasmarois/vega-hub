package hub

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupManager_PruneStaleWorktrees(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace structure
	workspacesDir := filepath.Join(tmpDir, "workspaces", "test-project")
	worktreeBase := filepath.Join(workspacesDir, "worktree-base")
	os.MkdirAll(worktreeBase, 0755)

	// Initialize git repo
	exec.Command("git", "-C", worktreeBase, "init").Run()
	exec.Command("git", "-C", worktreeBase, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", worktreeBase, "config", "user.name", "Test User").Run()

	// Create initial commit
	testFile := filepath.Join(worktreeBase, "test.txt")
	os.WriteFile(testFile, []byte("initial"), 0644)
	exec.Command("git", "-C", worktreeBase, "add", ".").Run()
	exec.Command("git", "-C", worktreeBase, "commit", "-m", "initial").Run()

	manager := NewCleanupManager(tmpDir)
	result := manager.PruneStaleWorktrees()

	// Should succeed with no errors
	if len(result.Errors) > 0 {
		t.Errorf("Expected no errors, got: %v", result.Errors)
	}

	// Should report project was pruned
	if len(result.PrunedWorktrees) != 1 {
		t.Errorf("Expected 1 pruned worktree, got %d", len(result.PrunedWorktrees))
	}
}

func TestCleanupManager_FindOrphanedWorktrees(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace structure
	workspacesDir := filepath.Join(tmpDir, "workspaces", "test-project")
	worktreeBase := filepath.Join(workspacesDir, "worktree-base")
	os.MkdirAll(worktreeBase, 0755)

	// Initialize git repo
	exec.Command("git", "-C", worktreeBase, "init").Run()
	exec.Command("git", "-C", worktreeBase, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", worktreeBase, "config", "user.name", "Test User").Run()

	// Create initial commit
	testFile := filepath.Join(worktreeBase, "test.txt")
	os.WriteFile(testFile, []byte("initial"), 0644)
	exec.Command("git", "-C", worktreeBase, "add", ".").Run()
	exec.Command("git", "-C", worktreeBase, "commit", "-m", "initial").Run()

	// Create an orphaned worktree directory (not registered with git)
	orphanDir := filepath.Join(workspacesDir, "goal-abc1234-test-orphan")
	os.MkdirAll(orphanDir, 0755)

	manager := NewCleanupManager(tmpDir)
	result := manager.FindOrphanedWorktrees()

	// Should find the orphan
	if len(result.OrphanedWorktrees) != 1 {
		t.Errorf("Expected 1 orphaned worktree, got %d", len(result.OrphanedWorktrees))
	}

	if len(result.OrphanedWorktrees) > 0 {
		orphan := result.OrphanedWorktrees[0]
		if orphan.GoalID != "abc1234" {
			t.Errorf("Expected goal ID 'abc1234', got '%s'", orphan.GoalID)
		}
		if orphan.Project != "test-project" {
			t.Errorf("Expected project 'test-project', got '%s'", orphan.Project)
		}
	}
}

func TestCleanupManager_RemoveOrphanedWorktree(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace structure
	workspacesDir := filepath.Join(tmpDir, "workspaces", "test-project")
	os.MkdirAll(workspacesDir, 0755)

	manager := NewCleanupManager(tmpDir)

	t.Run("removes orphan", func(t *testing.T) {
		orphanDir := filepath.Join(workspacesDir, "goal-abc1234-test")
		os.MkdirAll(orphanDir, 0755)

		err := manager.RemoveOrphanedWorktree(orphanDir, false)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if _, err := os.Stat(orphanDir); !os.IsNotExist(err) {
			t.Error("Expected directory to be removed")
		}
	})

	t.Run("rejects non-goal path", func(t *testing.T) {
		invalidDir := filepath.Join(workspacesDir, "not-a-goal")
		os.MkdirAll(invalidDir, 0755)

		err := manager.RemoveOrphanedWorktree(invalidDir, false)
		if err == nil {
			t.Error("Expected error for non-goal path")
		}
	})

	t.Run("rejects path outside workspaces", func(t *testing.T) {
		outsideDir := filepath.Join(tmpDir, "goal-abc1234-outside")
		os.MkdirAll(outsideDir, 0755)

		err := manager.RemoveOrphanedWorktree(outsideDir, false)
		if err == nil {
			t.Error("Expected error for path outside workspaces")
		}
	})
}

func TestCleanupManager_ArchiveCompletedGoals(t *testing.T) {
	tmpDir := t.TempDir()

	// Create history directory with some goals
	historyDir := filepath.Join(tmpDir, "goals", "history")
	os.MkdirAll(historyDir, 0755)

	// Create an old goal file
	oldGoal := filepath.Join(historyDir, "old1234.md")
	os.WriteFile(oldGoal, []byte("# Old Goal"), 0644)
	// Set modification time to 60 days ago
	oldTime := time.Now().Add(-60 * 24 * time.Hour)
	os.Chtimes(oldGoal, oldTime, oldTime)

	// Create a recent goal file
	recentGoal := filepath.Join(historyDir, "new5678.md")
	os.WriteFile(recentGoal, []byte("# Recent Goal"), 0644)

	manager := NewCleanupManager(tmpDir)

	t.Run("dry run", func(t *testing.T) {
		result := manager.ArchiveCompletedGoals(30*24*time.Hour, true) // 30 days

		if len(result.ArchivedGoals) != 1 {
			t.Errorf("Expected 1 goal to be archived, got %d", len(result.ArchivedGoals))
		}

		// File should still exist (dry run)
		if _, err := os.Stat(oldGoal); os.IsNotExist(err) {
			t.Error("Expected file to still exist in dry run")
		}
	})

	t.Run("actual archive", func(t *testing.T) {
		result := manager.ArchiveCompletedGoals(30*24*time.Hour, false) // 30 days

		if len(result.ArchivedGoals) != 1 {
			t.Errorf("Expected 1 goal to be archived, got %d", len(result.ArchivedGoals))
		}

		// Old file should be moved to archive
		archivedPath := filepath.Join(historyDir, "archive", "old1234.md")
		if _, err := os.Stat(archivedPath); os.IsNotExist(err) {
			t.Error("Expected file to be moved to archive")
		}

		// Recent file should still be in history
		if _, err := os.Stat(recentGoal); os.IsNotExist(err) {
			t.Error("Expected recent file to remain in history")
		}
	})
}

func TestExtractGoalIDFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/to/goal-abc1234-some-title", "abc1234"},
		{"/path/to/goal-xyz9999-another", "xyz9999"},
		{"goal-1234567-test", "1234567"},
		{"/path/to/not-a-goal", ""},
		{"/path/to/worktree-base", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := extractGoalIDFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestCleanupManager_RunStartupCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal workspace structure
	workspacesDir := filepath.Join(tmpDir, "workspaces", "test-project")
	worktreeBase := filepath.Join(workspacesDir, "worktree-base")
	os.MkdirAll(worktreeBase, 0755)

	// Initialize git repo
	exec.Command("git", "-C", worktreeBase, "init").Run()
	exec.Command("git", "-C", worktreeBase, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", worktreeBase, "config", "user.name", "Test User").Run()
	testFile := filepath.Join(worktreeBase, "test.txt")
	os.WriteFile(testFile, []byte("initial"), 0644)
	exec.Command("git", "-C", worktreeBase, "add", ".").Run()
	exec.Command("git", "-C", worktreeBase, "commit", "-m", "initial").Run()

	// Create an orphan
	orphanDir := filepath.Join(workspacesDir, "goal-abc1234-orphan")
	os.MkdirAll(orphanDir, 0755)

	manager := NewCleanupManager(tmpDir)
	result := manager.RunStartupCleanup()

	// Should prune stale references
	if len(result.PrunedWorktrees) == 0 {
		t.Error("Expected some pruning to occur")
	}

	// Should find the orphan
	if len(result.OrphanedWorktrees) != 1 {
		t.Errorf("Expected 1 orphan, got %d", len(result.OrphanedWorktrees))
	}
}
