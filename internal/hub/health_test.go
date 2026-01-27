package hub

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHealthChecker_CheckDiskSpace(t *testing.T) {
	tmpDir := t.TempDir()
	checker := NewHealthChecker(tmpDir)

	check := checker.CheckDiskSpace()

	// Should be healthy (tests run on systems with disk space)
	if check.Status == HealthUnhealthy {
		t.Errorf("Expected healthy disk space, got: %s", check.Message)
	}
}

func TestHealthChecker_CheckStuckGoals(t *testing.T) {
	tmpDir := t.TempDir()

	// Create goals directory structure
	goalsDir := filepath.Join(tmpDir, "goals", "active")
	os.MkdirAll(goalsDir, 0755)

	checker := NewHealthChecker(tmpDir)
	check := checker.CheckStuckGoals()

	// Should be healthy with no goals
	if check.Status != HealthHealthy {
		t.Errorf("Expected healthy, got: %s - %s", check.Status, check.Message)
	}
}

func TestHealthChecker_CheckWorktreeBases(t *testing.T) {
	tmpDir := t.TempDir()

	// No workspaces = healthy
	t.Run("no workspaces", func(t *testing.T) {
		checker := NewHealthChecker(tmpDir)
		check := checker.CheckWorktreeBases()

		if check.Status != HealthHealthy {
			t.Errorf("Expected healthy with no workspaces, got: %s", check.Status)
		}
	})

	// Clean worktree base = healthy
	t.Run("clean worktree base", func(t *testing.T) {
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

		checker := NewHealthChecker(tmpDir)
		check := checker.CheckWorktreeBases()

		// May be degraded due to fetch issues (no remote), but not unhealthy
		if check.Status == HealthUnhealthy {
			t.Errorf("Expected healthy or degraded, got unhealthy: %s", check.Message)
		}
	})
}

func TestHealthChecker_CheckOrphanedWorktrees(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workspace with orphan
	workspacesDir := filepath.Join(tmpDir, "workspaces", "test-project")
	worktreeBase := filepath.Join(workspacesDir, "worktree-base")
	orphanDir := filepath.Join(workspacesDir, "goal-abc1234-orphan")
	os.MkdirAll(worktreeBase, 0755)
	os.MkdirAll(orphanDir, 0755)

	// Initialize git repo
	exec.Command("git", "-C", worktreeBase, "init").Run()
	exec.Command("git", "-C", worktreeBase, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", worktreeBase, "config", "user.name", "Test User").Run()
	testFile := filepath.Join(worktreeBase, "test.txt")
	os.WriteFile(testFile, []byte("initial"), 0644)
	exec.Command("git", "-C", worktreeBase, "add", ".").Run()
	exec.Command("git", "-C", worktreeBase, "commit", "-m", "initial").Run()

	checker := NewHealthChecker(tmpDir)
	check := checker.CheckOrphanedWorktrees()

	// Should be degraded due to orphan
	if check.Status != HealthDegraded {
		t.Errorf("Expected degraded due to orphan, got: %s - %s", check.Status, check.Message)
	}
}

func TestHealthChecker_CheckActiveLocks(t *testing.T) {
	tmpDir := t.TempDir()

	checker := NewHealthChecker(tmpDir)
	check := checker.CheckActiveLocks()

	// No locks = healthy
	if check.Status != HealthHealthy {
		t.Errorf("Expected healthy with no locks, got: %s", check.Status)
	}
}

func TestHealthChecker_RunAllChecks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create minimal structure
	os.MkdirAll(filepath.Join(tmpDir, "goals", "active"), 0755)

	checker := NewHealthChecker(tmpDir)
	result := checker.RunAllChecks()

	// Should have all expected checks
	expectedChecks := []string{
		"worktree_bases",
		"stuck_goals",
		"disk_space",
		"git_credentials",
		"orphaned_worktrees",
		"active_locks",
	}

	for _, check := range expectedChecks {
		if _, ok := result.Checks[check]; !ok {
			t.Errorf("Expected check '%s' to be present", check)
		}
	}

	// Timestamp should be set
	if result.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}
}

func TestHealthStatus(t *testing.T) {
	// Verify constants
	if HealthHealthy != "healthy" {
		t.Error("HealthHealthy should be 'healthy'")
	}
	if HealthDegraded != "degraded" {
		t.Error("HealthDegraded should be 'degraded'")
	}
	if HealthUnhealthy != "unhealthy" {
		t.Error("HealthUnhealthy should be 'unhealthy'")
	}
}
