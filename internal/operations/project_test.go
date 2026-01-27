package operations

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAddProjectFromPath(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "vega-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create vega-missile structure
	vegaDir := filepath.Join(tempDir, "vega-missile")
	projectsDir := filepath.Join(vegaDir, "projects")
	workspacesDir := filepath.Join(vegaDir, "workspaces")
	os.MkdirAll(projectsDir, 0755)
	os.MkdirAll(workspacesDir, 0755)

	// Create projects/index.md
	indexContent := `# Managed Projects

| Project | Path | Active Goals | Description |
|---------|------|--------------|-------------|
`
	os.WriteFile(filepath.Join(projectsDir, "index.md"), []byte(indexContent), 0644)

	// Create a test git repository
	repoDir := filepath.Join(tempDir, "test-repo")
	os.MkdirAll(repoDir, 0755)
	exec.Command("git", "-C", repoDir, "init").Run()
	exec.Command("git", "-C", repoDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", repoDir, "config", "user.name", "Test").Run()
	os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Test"), 0644)
	exec.Command("git", "-C", repoDir, "add", ".").Run()
	exec.Command("git", "-C", repoDir, "commit", "-m", "Initial").Run()

	// Test adding a project
	result, data := AddProjectFromPath(AddProjectOptions{
		Name:       "test-project",
		Path:       repoDir,
		BaseBranch: "main",
		VegaDir:    vegaDir,
	})

	if !result.Success {
		t.Errorf("AddProjectFromPath failed: %v", result.Error)
	}

	if data == nil {
		t.Fatal("AddProjectFromPath returned nil data")
	}

	if data.Name != "test-project" {
		t.Errorf("Expected name 'test-project', got '%s'", data.Name)
	}

	// Verify config file was created
	configFile := filepath.Join(projectsDir, "test-project.md")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Verify workspace symlink was created
	worktreeBase := filepath.Join(workspacesDir, "test-project", "worktree-base")
	info, err := os.Lstat(worktreeBase)
	if err != nil {
		t.Errorf("Workspace symlink not created: %v", err)
	} else if info.Mode()&os.ModeSymlink == 0 {
		t.Error("Workspace is not a symlink")
	}

	// Test duplicate project name
	result, _ = AddProjectFromPath(AddProjectOptions{
		Name:       "test-project",
		Path:       repoDir,
		BaseBranch: "main",
		VegaDir:    vegaDir,
	})

	if result.Success {
		t.Error("Expected duplicate project to fail")
	}
	if result.Error == nil || result.Error.Code != "project_exists" {
		t.Error("Expected 'project_exists' error code")
	}
}

func TestRemoveProject(t *testing.T) {
	// Create a temp directory for testing
	tempDir, err := os.MkdirTemp("", "vega-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create vega-missile structure
	vegaDir := filepath.Join(tempDir, "vega-missile")
	projectsDir := filepath.Join(vegaDir, "projects")
	workspacesDir := filepath.Join(vegaDir, "workspaces")
	goalsDir := filepath.Join(vegaDir, "goals")
	os.MkdirAll(projectsDir, 0755)
	os.MkdirAll(workspacesDir, 0755)
	os.MkdirAll(goalsDir, 0755)

	// Create project config
	projectConfig := `# Project: test-project

## Location

**Workspace**: ` + "`workspaces/test-project/worktree-base/`" + `
**Base Branch**: ` + "`main`" + `
`
	os.WriteFile(filepath.Join(projectsDir, "test-project.md"), []byte(projectConfig), 0644)

	// Create index.md with project entry
	indexContent := `# Managed Projects

| Project | Path | Active Goals | Description |
|---------|------|--------------|-------------|
| [test-project](test-project.md) | ` + "`workspaces/test-project/worktree-base/`" + ` | - | Test |
`
	os.WriteFile(filepath.Join(projectsDir, "index.md"), []byte(indexContent), 0644)

	// Create workspace with symlink
	os.MkdirAll(filepath.Join(workspacesDir, "test-project"), 0755)
	os.Symlink("/tmp/fake-repo", filepath.Join(workspacesDir, "test-project", "worktree-base"))

	// Create REGISTRY.md
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
	os.WriteFile(filepath.Join(goalsDir, "REGISTRY.md"), []byte(registryContent), 0644)

	// Test removing the project
	result, data := RemoveProject(RemoveProjectOptions{
		Name:    "test-project",
		Force:   false,
		VegaDir: vegaDir,
	})

	if !result.Success {
		t.Errorf("RemoveProject failed: %v", result.Error)
	}

	if data == nil {
		t.Fatal("RemoveProject returned nil data")
	}

	if !data.ConfigRemoved {
		t.Error("Config file was not removed")
	}

	if !data.IndexUpdated {
		t.Error("Index was not updated")
	}

	// Verify config file was deleted
	configFile := filepath.Join(projectsDir, "test-project.md")
	if _, err := os.Stat(configFile); !os.IsNotExist(err) {
		t.Error("Config file still exists")
	}

	// Test removing non-existent project
	result, _ = RemoveProject(RemoveProjectOptions{
		Name:    "nonexistent",
		Force:   false,
		VegaDir: vegaDir,
	})

	if result.Success {
		t.Error("Expected removing non-existent project to fail")
	}
}

func TestIsValidProjectName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"my-project", true},
		{"my_project", true},
		{"MyProject123", true},
		{"a", true},
		{"my project", false},
		{"my.project", false},
		{"my/project", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidProjectName(tt.name); got != tt.valid {
				t.Errorf("isValidProjectName(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}
