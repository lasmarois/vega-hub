package operations

import (
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

// TestConcurrentGoalCreation tests that two concurrent goal creates both succeed
func TestConcurrentGoalCreation(t *testing.T) {
	// Create a temporary vega directory structure
	tmpDir, err := os.MkdirTemp("", "vega-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create necessary directories
	dirs := []string{
		filepath.Join(tmpDir, "goals", "active"),
		filepath.Join(tmpDir, "goals", "history"),
		filepath.Join(tmpDir, "goals", "iced"),
		filepath.Join(tmpDir, "projects"),
		filepath.Join(tmpDir, "workspaces", "test-project", "worktree-base"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", d, err)
		}
	}

	// Create REGISTRY.md with Active Goals table
	registryContent := `# Goal Registry

## Active Goals

| ID | Title | Project(s) | Status | Phase |
|----|-------|------------|--------|-------|

## Iced Goals

| ID | Title | Project(s) | Reason |
|----|-------|------------|--------|

## Completed Goals

| ID | Title | Project(s) | Completed |
|----|-------|------------|-----------|
`
	registryPath := filepath.Join(tmpDir, "goals", "REGISTRY.md")
	if err := os.WriteFile(registryPath, []byte(registryContent), 0644); err != nil {
		t.Fatalf("Failed to create registry: %v", err)
	}

	// Create project config
	projectContent := `# Project: test-project

## Overview
Test project

## Location
**Base Branch**: main

## Active Goals
`
	projectPath := filepath.Join(tmpDir, "projects", "test-project.md")
	if err := os.WriteFile(projectPath, []byte(projectContent), 0644); err != nil {
		t.Fatalf("Failed to create project config: %v", err)
	}

	// Initialize git repo in worktree-base
	worktreeBase := filepath.Join(tmpDir, "workspaces", "test-project", "worktree-base")
	runGit := func(args ...string) error {
		cmd := append([]string{"-C", worktreeBase}, args...)
		return runCommand("git", cmd...)
	}
	if err := runGit("init"); err != nil {
		t.Fatalf("Failed to init git: %v", err)
	}
	if err := runGit("config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("Failed to config git: %v", err)
	}
	if err := runGit("config", "user.name", "Test"); err != nil {
		t.Fatalf("Failed to config git: %v", err)
	}
	// Create initial commit
	readmePath := filepath.Join(worktreeBase, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test"), 0644); err != nil {
		t.Fatalf("Failed to create README: %v", err)
	}
	if err := runGit("add", "."); err != nil {
		t.Fatalf("Failed to git add: %v", err)
	}
	if err := runGit("commit", "-m", "Initial commit"); err != nil {
		t.Fatalf("Failed to git commit: %v", err)
	}

	// Create two goals concurrently
	var wg sync.WaitGroup
	var results [2]*Result
	var createResults [2]*CreateResult
	var mu sync.Mutex

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			result, createResult := CreateGoal(CreateOptions{
				Title:      "Test Goal " + string(rune('A'+idx)),
				Project:    "test-project",
				NoWorktree: true, // Skip worktree to simplify test
				VegaDir:    tmpDir,
			})
			mu.Lock()
			results[idx] = result
			createResults[idx] = createResult
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Both should succeed
	for i, result := range results {
		if result == nil {
			t.Errorf("Goal %d: result is nil", i)
			continue
		}
		if !result.Success {
			errMsg := "unknown"
			if result.Error != nil {
				errMsg = result.Error.Message
			}
			t.Errorf("Goal %d: expected success, got error: %s", i, errMsg)
		}
	}

	// Verify both goals are in the registry
	content, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("Failed to read registry: %v", err)
	}

	registryStr := string(content)
	for i, cr := range createResults {
		if cr == nil {
			continue
		}
		if !containsGoalID(registryStr, cr.GoalID) {
			t.Errorf("Goal %d (ID: %s) not found in registry", i, cr.GoalID)
		}
	}

	t.Logf("Registry content:\n%s", registryStr)
}

func containsGoalID(content, goalID string) bool {
	return len(goalID) > 0 && len(content) > 0 && 
		(len(goalID) <= len(content)) && 
		(indexString(content, goalID) >= 0)
}

func indexString(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}
