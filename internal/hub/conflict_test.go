package hub

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestConflictChecker_IsConflicted(t *testing.T) {
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

	checker := NewConflictChecker(tmpDir)

	t.Run("no conflict", func(t *testing.T) {
		if checker.IsConflicted() {
			t.Error("Expected no conflict")
		}
	})
}

func TestConflictChecker_DetectConflicts(t *testing.T) {
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

	checker := NewConflictChecker(tmpDir)

	t.Run("no conflict", func(t *testing.T) {
		details, err := checker.DetectConflicts()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if details != nil {
			t.Error("Expected no conflict details")
		}
	})
}

func TestConflictChecker_MarkResolved(t *testing.T) {
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

	// Create a new untracked file
	newFile := filepath.Join(tmpDir, "new.txt")
	os.WriteFile(newFile, []byte("new content"), 0644)

	checker := NewConflictChecker(tmpDir)

	err := checker.MarkResolved()
	if err != nil {
		t.Errorf("Failed to mark resolved: %v", err)
	}

	// Check if file is staged
	cmd := exec.Command("git", "-C", tmpDir, "diff", "--cached", "--name-only")
	output, _ := cmd.Output()
	if string(output) == "" {
		t.Error("Expected file to be staged")
	}
}

func TestParseConflictingFiles(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name: "content conflict",
			input: `Auto-merging file.txt
CONFLICT (content): Merge conflict in file.txt
Automatic merge failed; fix conflicts and then commit the result.`,
			expected: []string{"file.txt"},
		},
		{
			name: "multiple conflicts",
			input: `CONFLICT (content): Merge conflict in src/main.go
CONFLICT (content): Merge conflict in src/config.yaml`,
			expected: []string{"src/main.go", "src/config.yaml"},
		},
		{
			name:     "no conflicts",
			input:    "Already up to date.",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseConflictingFiles(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d files, got %d", len(tt.expected), len(result))
			}
			for i, file := range tt.expected {
				if i < len(result) && result[i] != file {
					t.Errorf("Expected file '%s', got '%s'", file, result[i])
				}
			}
		})
	}
}
