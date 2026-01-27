package hub

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ConflictDetails contains information about a merge conflict
type ConflictDetails struct {
	SourceBranch     string    `json:"source_branch"`
	TargetBranch     string    `json:"target_branch"`
	ConflictingFiles []string  `json:"conflicting_files"`
	MergeBase        string    `json:"merge_base,omitempty"`
	DetectedAt       time.Time `json:"detected_at"`
}

// ConflictChecker handles merge conflict detection and resolution
type ConflictChecker struct {
	worktreePath string
}

// NewConflictChecker creates a new conflict checker
func NewConflictChecker(worktreePath string) *ConflictChecker {
	return &ConflictChecker{worktreePath: worktreePath}
}

// DetectConflicts checks if there are any merge conflicts in the worktree
func (c *ConflictChecker) DetectConflicts() (*ConflictDetails, error) {
	// Check git status for conflict markers
	cmd := exec.Command("git", "-C", c.worktreePath, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to check git status: %v", err)
	}

	var conflictingFiles []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if len(line) < 3 {
			continue
		}
		// UU = both modified (conflict), AA = both added, DD = both deleted
		status := line[:2]
		if status == "UU" || status == "AA" || status == "DD" ||
			status == "AU" || status == "UA" || status == "DU" || status == "UD" {
			file := strings.TrimSpace(line[3:])
			conflictingFiles = append(conflictingFiles, file)
		}
	}

	if len(conflictingFiles) == 0 {
		return nil, nil // No conflicts
	}

	// Get current branch
	branchCmd := exec.Command("git", "-C", c.worktreePath, "branch", "--show-current")
	branchOutput, _ := branchCmd.Output()
	sourceBranch := strings.TrimSpace(string(branchOutput))

	// Get merge base
	mergeBaseCmd := exec.Command("git", "-C", c.worktreePath, "merge-base", "HEAD", "MERGE_HEAD")
	mergeBaseOutput, _ := mergeBaseCmd.Output()
	mergeBase := strings.TrimSpace(string(mergeBaseOutput))

	return &ConflictDetails{
		SourceBranch:     sourceBranch,
		ConflictingFiles: conflictingFiles,
		MergeBase:        mergeBase,
		DetectedAt:       time.Now(),
	}, nil
}

// CheckMergeConflict attempts a dry-run merge to detect potential conflicts
func (c *ConflictChecker) CheckMergeConflict(targetBranch string) (*ConflictDetails, error) {
	// First, fetch to ensure we have latest
	exec.Command("git", "-C", c.worktreePath, "fetch", "origin").Run()

	// Get current branch
	branchCmd := exec.Command("git", "-C", c.worktreePath, "branch", "--show-current")
	branchOutput, _ := branchCmd.Output()
	sourceBranch := strings.TrimSpace(string(branchOutput))

	// Try merge with --no-commit --no-ff to detect conflicts
	mergeCmd := exec.Command("git", "-C", c.worktreePath, "merge", "--no-commit", "--no-ff",
		fmt.Sprintf("origin/%s", targetBranch))
	mergeOutput, err := mergeCmd.CombinedOutput()

	if err != nil {
		// Abort the merge attempt
		exec.Command("git", "-C", c.worktreePath, "merge", "--abort").Run()

		// Check if it was a conflict
		if strings.Contains(string(mergeOutput), "CONFLICT") ||
			strings.Contains(string(mergeOutput), "Automatic merge failed") {

			// Parse conflicting files from output
			conflictingFiles := parseConflictingFiles(string(mergeOutput))

			return &ConflictDetails{
				SourceBranch:     sourceBranch,
				TargetBranch:     targetBranch,
				ConflictingFiles: conflictingFiles,
				DetectedAt:       time.Now(),
			}, nil
		}

		return nil, fmt.Errorf("merge check failed: %s", string(mergeOutput))
	}

	// No conflicts - abort the merge (we were just checking)
	exec.Command("git", "-C", c.worktreePath, "merge", "--abort").Run()
	return nil, nil
}

// IsConflicted returns true if the worktree is currently in a conflicted state
func (c *ConflictChecker) IsConflicted() bool {
	cmd := exec.Command("git", "-C", c.worktreePath, "ls-files", "--unmerged")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// AbortMerge aborts an in-progress merge
func (c *ConflictChecker) AbortMerge() error {
	cmd := exec.Command("git", "-C", c.worktreePath, "merge", "--abort")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to abort merge: %s", string(output))
	}
	return nil
}

// MarkResolved marks a file as resolved (after manual conflict resolution)
func (c *ConflictChecker) MarkResolved(files ...string) error {
	if len(files) == 0 {
		// Mark all as resolved
		cmd := exec.Command("git", "-C", c.worktreePath, "add", "-A")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to mark files as resolved: %s", string(output))
		}
		return nil
	}

	// Mark specific files
	args := append([]string{"-C", c.worktreePath, "add"}, files...)
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to mark files as resolved: %s", string(output))
	}
	return nil
}

// ContinueMerge continues a merge after conflicts have been resolved
func (c *ConflictChecker) ContinueMerge(commitMessage string) error {
	// Check if there are still unresolved conflicts
	if c.IsConflicted() {
		return fmt.Errorf("there are still unresolved conflicts")
	}

	// Commit the merge
	cmd := exec.Command("git", "-C", c.worktreePath, "commit", "-m", commitMessage)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to complete merge: %s", string(output))
	}
	return nil
}

// parseConflictingFiles extracts file names from merge conflict output
func parseConflictingFiles(output string) []string {
	seen := make(map[string]bool)
	var files []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Look for "CONFLICT (content): Merge conflict in <file>"
		if strings.Contains(line, "CONFLICT") && strings.Contains(line, "Merge conflict in") {
			parts := strings.Split(line, "Merge conflict in ")
			if len(parts) >= 2 {
				file := strings.TrimSpace(parts[1])
				if !seen[file] {
					seen[file] = true
					files = append(files, file)
				}
			}
		}
	}
	return files
}
