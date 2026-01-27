package goals

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlanningFilesManager_SaveAndGet(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "planning-files-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create goals/active directory structure
	activeDir := filepath.Join(tmpDir, "goals", "active")
	if err := os.MkdirAll(activeDir, 0755); err != nil {
		t.Fatalf("Failed to create active dir: %v", err)
	}

	mgr := NewPlanningFilesManager(tmpDir)

	// Test save
	goalID := "abc123"
	project := "my-api"
	filename := "task_plan.md"
	content := "# Task Plan\n\n- [x] Step 1\n- [ ] Step 2"

	err = mgr.SavePlanningFile(goalID, project, filename, content)
	if err != nil {
		t.Fatalf("SavePlanningFile failed: %v", err)
	}

	// Test get
	retrieved, err := mgr.GetPlanningFile(goalID, project, filename)
	if err != nil {
		t.Fatalf("GetPlanningFile failed: %v", err)
	}

	if retrieved != content {
		t.Errorf("Content mismatch:\n  got: %q\n  want: %q", retrieved, content)
	}

	// Verify file structure
	expectedPath := filepath.Join(activeDir, goalID, "project-plans", project, filename)
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Expected file at %s does not exist", expectedPath)
	}
}

func TestPlanningFilesManager_ListFiles(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "planning-files-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create goals/active directory structure
	activeDir := filepath.Join(tmpDir, "goals", "active")
	if err := os.MkdirAll(activeDir, 0755); err != nil {
		t.Fatalf("Failed to create active dir: %v", err)
	}

	mgr := NewPlanningFilesManager(tmpDir)
	goalID := "abc123"

	// Save multiple files across multiple projects
	files := map[string][]struct {
		filename string
		content  string
	}{
		"my-api": {
			{filename: "task_plan.md", content: "# API Plan"},
			{filename: "findings.md", content: "# API Findings"},
		},
		"my-frontend": {
			{filename: "task_plan.md", content: "# Frontend Plan"},
		},
	}

	for project, projectFiles := range files {
		for _, f := range projectFiles {
			if err := mgr.SavePlanningFile(goalID, project, f.filename, f.content); err != nil {
				t.Fatalf("SavePlanningFile failed for %s/%s: %v", project, f.filename, err)
			}
		}
	}

	// Test list
	list, err := mgr.ListPlanningFiles(goalID)
	if err != nil {
		t.Fatalf("ListPlanningFiles failed: %v", err)
	}

	// Verify my-api project
	if apiFiles, ok := list["my-api"]; !ok {
		t.Error("Expected my-api project in list")
	} else {
		if len(apiFiles) != 2 {
			t.Errorf("Expected 2 files in my-api, got %d", len(apiFiles))
		}
		// Check files are sorted
		if len(apiFiles) >= 2 && apiFiles[0] != "findings.md" {
			t.Errorf("Expected first file to be findings.md (sorted), got %s", apiFiles[0])
		}
	}

	// Verify my-frontend project
	if frontendFiles, ok := list["my-frontend"]; !ok {
		t.Error("Expected my-frontend project in list")
	} else {
		if len(frontendFiles) != 1 {
			t.Errorf("Expected 1 file in my-frontend, got %d", len(frontendFiles))
		}
	}
}

func TestPlanningFilesManager_GetAllFiles(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "planning-files-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create goals/active directory structure
	activeDir := filepath.Join(tmpDir, "goals", "active")
	if err := os.MkdirAll(activeDir, 0755); err != nil {
		t.Fatalf("Failed to create active dir: %v", err)
	}

	mgr := NewPlanningFilesManager(tmpDir)
	goalID := "abc123"

	// Save files
	if err := mgr.SavePlanningFile(goalID, "project-a", "plan.md", "# Plan A"); err != nil {
		t.Fatalf("SavePlanningFile failed: %v", err)
	}
	if err := mgr.SavePlanningFile(goalID, "project-b", "plan.md", "# Plan B"); err != nil {
		t.Fatalf("SavePlanningFile failed: %v", err)
	}

	// Test get all
	all, err := mgr.GetAllPlanningFiles(goalID)
	if err != nil {
		t.Fatalf("GetAllPlanningFiles failed: %v", err)
	}

	// Verify content
	if all["project-a"]["plan.md"] != "# Plan A" {
		t.Errorf("Wrong content for project-a/plan.md: %q", all["project-a"]["plan.md"])
	}
	if all["project-b"]["plan.md"] != "# Plan B" {
		t.Errorf("Wrong content for project-b/plan.md: %q", all["project-b"]["plan.md"])
	}
}

func TestPlanningFilesManager_EmptyGoal(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "planning-files-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create goals/active directory structure
	activeDir := filepath.Join(tmpDir, "goals", "active")
	if err := os.MkdirAll(activeDir, 0755); err != nil {
		t.Fatalf("Failed to create active dir: %v", err)
	}

	mgr := NewPlanningFilesManager(tmpDir)

	// List files for non-existent goal (should return empty, not error)
	list, err := mgr.ListPlanningFiles("nonexistent")
	if err != nil {
		t.Fatalf("ListPlanningFiles failed for non-existent goal: %v", err)
	}

	if len(list) != 0 {
		t.Errorf("Expected empty list for non-existent goal, got %d projects", len(list))
	}

	// GetAllPlanningFiles should also return empty
	all, err := mgr.GetAllPlanningFiles("nonexistent")
	if err != nil {
		t.Fatalf("GetAllPlanningFiles failed for non-existent goal: %v", err)
	}

	if len(all) != 0 {
		t.Errorf("Expected empty map for non-existent goal, got %d projects", len(all))
	}
}

func TestPlanningFilesManager_NotFound(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "planning-files-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewPlanningFilesManager(tmpDir)

	// Get non-existent file
	_, err = mgr.GetPlanningFile("goal1", "project1", "missing.md")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestPlanningFilesManager_PathTraversal(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "planning-files-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewPlanningFilesManager(tmpDir)

	// Test path traversal prevention
	invalidFilenames := []string{
		"../secret.txt",
		"subdir/file.txt",
		"..",
		"..\\passwd",
	}

	for _, filename := range invalidFilenames {
		err := mgr.SavePlanningFile("goal1", "project1", filename, "content")
		if err == nil {
			t.Errorf("Expected error for invalid filename %q, got nil", filename)
		}
	}
}

func TestPlanningFilesManager_Delete(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "planning-files-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create goals/active directory structure
	activeDir := filepath.Join(tmpDir, "goals", "active")
	if err := os.MkdirAll(activeDir, 0755); err != nil {
		t.Fatalf("Failed to create active dir: %v", err)
	}

	mgr := NewPlanningFilesManager(tmpDir)
	goalID := "abc123"
	project := "my-api"
	filename := "task_plan.md"

	// Save and then delete
	if err := mgr.SavePlanningFile(goalID, project, filename, "content"); err != nil {
		t.Fatalf("SavePlanningFile failed: %v", err)
	}

	// Verify file exists
	if _, err := mgr.GetPlanningFile(goalID, project, filename); err != nil {
		t.Fatalf("File should exist before delete: %v", err)
	}

	// Delete
	if err := mgr.DeletePlanningFile(goalID, project, filename); err != nil {
		t.Fatalf("DeletePlanningFile failed: %v", err)
	}

	// Verify file is gone
	if _, err := mgr.GetPlanningFile(goalID, project, filename); err == nil {
		t.Error("File should not exist after delete")
	}
}

func TestPlanningFilesManager_RequiredParams(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "planning-files-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewPlanningFilesManager(tmpDir)

	// Test empty goal ID
	if err := mgr.SavePlanningFile("", "project", "file.md", "content"); err == nil {
		t.Error("Expected error for empty goal ID")
	}

	// Test empty project
	if err := mgr.SavePlanningFile("goal1", "", "file.md", "content"); err == nil {
		t.Error("Expected error for empty project")
	}

	// Test empty filename
	if err := mgr.SavePlanningFile("goal1", "project", "", "content"); err == nil {
		t.Error("Expected error for empty filename")
	}
}
