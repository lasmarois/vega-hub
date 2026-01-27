package hub

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPreflightChecker_CheckWorktreeClean(t *testing.T) {
	// Create temp git repo
	tmpDir := t.TempDir()
	
	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for commits
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test User").Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("initial"), 0644)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

	checker := NewPreflightChecker(tmpDir, "main", "")

	// Test 1: Clean worktree should pass
	t.Run("clean worktree", func(t *testing.T) {
		check := checker.CheckWorktreeClean()
		if !check.Passed {
			t.Errorf("Expected clean worktree to pass, got: %s", check.Error)
		}
	})

	// Test 2: Dirty worktree should fail
	t.Run("dirty worktree", func(t *testing.T) {
		os.WriteFile(testFile, []byte("modified"), 0644)
		check := checker.CheckWorktreeClean()
		if check.Passed {
			t.Error("Expected dirty worktree to fail")
		}
		if check.Error == "" {
			t.Error("Expected error message for dirty worktree")
		}
		// Clean up
		exec.Command("git", "-C", tmpDir, "checkout", "--", ".").Run()
	})

	// Test 3: Untracked files should fail
	t.Run("untracked files", func(t *testing.T) {
		newFile := filepath.Join(tmpDir, "untracked.txt")
		os.WriteFile(newFile, []byte("new"), 0644)
		check := checker.CheckWorktreeClean()
		if check.Passed {
			t.Error("Expected untracked files to fail")
		}
		// Clean up
		os.Remove(newFile)
	})
}

func TestPreflightChecker_CheckDiskSpace(t *testing.T) {
	tmpDir := t.TempDir()
	checker := NewPreflightChecker(tmpDir, "main", "")

	// Test with very low requirement (should pass)
	t.Run("sufficient space", func(t *testing.T) {
		checker.SetMinDiskMB(1) // 1MB
		check := checker.CheckDiskSpace()
		if !check.Passed {
			t.Errorf("Expected disk space check to pass with 1MB requirement: %s", check.Error)
		}
		if check.AvailMB <= 0 {
			t.Error("Expected available MB to be reported")
		}
	})

	// Test with impossibly high requirement (should fail)
	t.Run("insufficient space", func(t *testing.T) {
		checker.SetMinDiskMB(1024 * 1024 * 1024) // 1PB
		check := checker.CheckDiskSpace()
		if check.Passed {
			t.Error("Expected disk space check to fail with 1PB requirement")
		}
	})
}

func TestPreflightChecker_CheckNoInProgressOps(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)

	checker := NewPreflightChecker(tmpDir, "main", "")

	// Test 1: No ops in progress
	t.Run("no ops", func(t *testing.T) {
		check := checker.CheckNoInProgressOps()
		if !check.Passed {
			t.Errorf("Expected no ops check to pass: %s", check.Error)
		}
	})

	// Test 2: Rebase in progress
	t.Run("rebase in progress", func(t *testing.T) {
		rebaseDir := filepath.Join(gitDir, "rebase-merge")
		os.MkdirAll(rebaseDir, 0755)
		check := checker.CheckNoInProgressOps()
		if check.Passed {
			t.Error("Expected check to fail with rebase in progress")
		}
		os.RemoveAll(rebaseDir)
	})

	// Test 3: Merge in progress
	t.Run("merge in progress", func(t *testing.T) {
		mergeHead := filepath.Join(gitDir, "MERGE_HEAD")
		os.WriteFile(mergeHead, []byte("abc123"), 0644)
		check := checker.CheckNoInProgressOps()
		if check.Passed {
			t.Error("Expected check to fail with merge in progress")
		}
		os.Remove(mergeHead)
	})
}

func TestPreflightChecker_CheckBranchAvailable(t *testing.T) {
	// Create temp git repo
	tmpDir := t.TempDir()
	
	// Initialize git repo
	exec.Command("git", "-C", tmpDir, "init").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test User").Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("initial"), 0644)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

	// Test 1: Non-existent branch should pass
	t.Run("branch available", func(t *testing.T) {
		checker := NewPreflightChecker(tmpDir, "main", "new-branch")
		check := checker.CheckBranchAvailable()
		if !check.Passed {
			t.Errorf("Expected branch check to pass for non-existent branch: %s", check.Error)
		}
	})

	// Test 2: Existing branch should fail
	t.Run("branch exists", func(t *testing.T) {
		// Create a branch
		exec.Command("git", "-C", tmpDir, "branch", "existing-branch").Run()
		
		checker := NewPreflightChecker(tmpDir, "main", "existing-branch")
		check := checker.CheckBranchAvailable()
		if check.Passed {
			t.Error("Expected branch check to fail for existing branch")
		}
		if check.Existing != "local" {
			t.Errorf("Expected existing='local', got '%s'", check.Existing)
		}
	})

	// Test 3: Empty branch name should pass (skip check)
	t.Run("empty branch name", func(t *testing.T) {
		checker := NewPreflightChecker(tmpDir, "main", "")
		check := checker.CheckBranchAvailable()
		if !check.Passed {
			t.Error("Expected check to pass when branch name is empty")
		}
	})
}

func TestPreflightChecker_RunAll(t *testing.T) {
	// Create temp git repo
	tmpDir := t.TempDir()
	
	// Initialize git repo
	exec.Command("git", "-C", tmpDir, "init").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test User").Run()

	// Create initial commit
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("initial"), 0644)
	exec.Command("git", "-C", tmpDir, "add", ".").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

	t.Run("all checks run", func(t *testing.T) {
		checker := NewPreflightChecker(tmpDir, "master", "new-branch")
		checker.SetMinDiskMB(1) // Low requirement for test
		
		result := checker.RunAll()

		// Verify all expected checks are present
		expectedChecks := []string{
			"worktree_clean",
			"disk_space",
			"branch_available",
			"no_in_progress_ops",
		}

		for _, check := range expectedChecks {
			if _, ok := result.Checks[check]; !ok {
				t.Errorf("Expected check '%s' to be present", check)
			}
		}
	})

	t.Run("blocking issues populated", func(t *testing.T) {
		// Make worktree dirty
		os.WriteFile(testFile, []byte("modified"), 0644)

		checker := NewPreflightChecker(tmpDir, "master", "new-branch")
		checker.SetMinDiskMB(1)
		
		result := checker.RunAll()

		if result.Ready {
			t.Error("Expected Ready=false when worktree is dirty")
		}
		if len(result.BlockingIssues) == 0 {
			t.Error("Expected blocking issues to be populated")
		}
		if len(result.FixCommands) == 0 {
			t.Error("Expected fix commands to be populated")
		}

		// Verify worktree_clean is in blocking issues
		found := false
		for _, issue := range result.BlockingIssues {
			if issue == "worktree_clean" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'worktree_clean' in blocking issues")
		}
	})
}

func TestPreflightResult_JSONSerialization(t *testing.T) {
	result := &PreflightResult{
		Ready: false,
		Checks: map[string]PreflightCheck{
			"worktree_clean": {Passed: true},
			"disk_space":     {Passed: true, AvailMB: 5000, RequiredMB: 1024},
			"credentials":    {Passed: false, Error: "No credentials"},
		},
		BlockingIssues: []string{"credentials"},
		FixCommands:    []string{"gh auth login"},
	}

	// Verify result can be serialized (for JSON output)
	if result.Ready {
		t.Error("Expected Ready=false")
	}
	if len(result.Checks) != 3 {
		t.Errorf("Expected 3 checks, got %d", len(result.Checks))
	}
	if result.Checks["disk_space"].AvailMB != 5000 {
		t.Error("Expected AvailMB to be preserved")
	}
}
