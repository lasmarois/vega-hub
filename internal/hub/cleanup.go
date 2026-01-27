package hub

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// OrphanedWorktree represents a worktree that exists on disk but isn't properly registered
type OrphanedWorktree struct {
	Path       string    `json:"path"`
	GoalID     string    `json:"goal_id,omitempty"`
	Project    string    `json:"project"`
	Reason     string    `json:"reason"`
	ModifiedAt time.Time `json:"modified_at"`
}

// CleanupResult contains the results of a cleanup operation
type CleanupResult struct {
	PrunedWorktrees   []string          `json:"pruned_worktrees,omitempty"`
	OrphanedWorktrees []OrphanedWorktree `json:"orphaned_worktrees,omitempty"`
	ArchivedGoals     []string          `json:"archived_goals,omitempty"`
	Errors            []string          `json:"errors,omitempty"`
}

// CleanupManager handles cleanup operations for worktrees and goals
type CleanupManager struct {
	vegaDir string
}

// NewCleanupManager creates a new cleanup manager
func NewCleanupManager(vegaDir string) *CleanupManager {
	return &CleanupManager{vegaDir: vegaDir}
}

// PruneStaleWorktrees runs git worktree prune on all project worktree-bases
func (c *CleanupManager) PruneStaleWorktrees() *CleanupResult {
	result := &CleanupResult{}

	workspacesDir := filepath.Join(c.vegaDir, "workspaces")
	entries, err := os.ReadDir(workspacesDir)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read workspaces: %v", err))
		return result
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectName := entry.Name()
		worktreeBase := filepath.Join(workspacesDir, projectName, "worktree-base")

		if _, err := os.Stat(worktreeBase); os.IsNotExist(err) {
			continue
		}

		// Run git worktree prune
		cmd := exec.Command("git", "-C", worktreeBase, "worktree", "prune")
		output, err := cmd.CombinedOutput()
		if err != nil {
			result.Errors = append(result.Errors,
				fmt.Sprintf("Failed to prune %s: %v - %s", projectName, err, string(output)))
		} else {
			result.PrunedWorktrees = append(result.PrunedWorktrees, projectName)
		}
	}

	return result
}

// FindOrphanedWorktrees finds worktrees that exist on disk but aren't registered with git
func (c *CleanupManager) FindOrphanedWorktrees() *CleanupResult {
	result := &CleanupResult{}

	workspacesDir := filepath.Join(c.vegaDir, "workspaces")
	entries, err := os.ReadDir(workspacesDir)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read workspaces: %v", err))
		return result
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectName := entry.Name()
		projectWorkspaceDir := filepath.Join(workspacesDir, projectName)
		worktreeBase := filepath.Join(projectWorkspaceDir, "worktree-base")

		if _, err := os.Stat(worktreeBase); os.IsNotExist(err) {
			continue
		}

		// Get registered worktrees from git
		registeredPaths := c.getRegisteredWorktrees(worktreeBase)

		// Find goal-* directories in workspace
		goalDirs, err := filepath.Glob(filepath.Join(projectWorkspaceDir, "goal-*"))
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to glob %s: %v", projectName, err))
			continue
		}

		for _, goalDir := range goalDirs {
			// Check if this worktree is registered
			if !registeredPaths[goalDir] {
				// Extract goal ID from path
				goalID := extractGoalIDFromPath(goalDir)

				// Get modification time
				info, _ := os.Stat(goalDir)
				modTime := time.Time{}
				if info != nil {
					modTime = info.ModTime()
				}

				// Determine reason
				reason := "Not registered with git worktree"
				if !c.hasGoalFile(goalID) {
					reason = "No corresponding goal file"
				}

				result.OrphanedWorktrees = append(result.OrphanedWorktrees, OrphanedWorktree{
					Path:       goalDir,
					GoalID:     goalID,
					Project:    projectName,
					Reason:     reason,
					ModifiedAt: modTime,
				})
			}
		}
	}

	return result
}

// RemoveOrphanedWorktree removes an orphaned worktree directory
func (c *CleanupManager) RemoveOrphanedWorktree(path string, force bool) error {
	// Safety check: must be under workspaces directory
	workspacesDir := filepath.Join(c.vegaDir, "workspaces")
	if !strings.HasPrefix(path, workspacesDir) {
		return fmt.Errorf("path is not under workspaces directory: %s", path)
	}

	// Safety check: must be a goal-* directory
	baseName := filepath.Base(path)
	if !strings.HasPrefix(baseName, "goal-") {
		return fmt.Errorf("path is not a goal directory: %s", path)
	}

	// Check if directory exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", path)
	}

	// If not forcing, check for uncommitted changes
	if !force {
		cmd := exec.Command("git", "-C", path, "status", "--porcelain")
		output, err := cmd.Output()
		if err == nil && len(strings.TrimSpace(string(output))) > 0 {
			return fmt.Errorf("worktree has uncommitted changes (use --force to override)")
		}
	}

	// Remove the directory
	return os.RemoveAll(path)
}

// ArchiveCompletedGoals archives goals that have been completed for longer than the specified duration
func (c *CleanupManager) ArchiveCompletedGoals(olderThan time.Duration, dryRun bool) *CleanupResult {
	result := &CleanupResult{}

	historyDir := filepath.Join(c.vegaDir, "goals", "history")
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return result // No history directory, nothing to archive
		}
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to read history: %v", err))
		return result
	}

	cutoff := time.Now().Add(-olderThan)

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		goalFile := filepath.Join(historyDir, entry.Name())
		info, err := os.Stat(goalFile)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			goalID := strings.TrimSuffix(entry.Name(), ".md")

			if dryRun {
				result.ArchivedGoals = append(result.ArchivedGoals, goalID+" (dry-run)")
			} else {
				// Archive: move to archive subdirectory
				archiveDir := filepath.Join(historyDir, "archive")
				if err := os.MkdirAll(archiveDir, 0755); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to create archive dir: %v", err))
					continue
				}

				archivePath := filepath.Join(archiveDir, entry.Name())
				if err := os.Rename(goalFile, archivePath); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to archive %s: %v", goalID, err))
				} else {
					result.ArchivedGoals = append(result.ArchivedGoals, goalID)
				}

				// Also archive state file if exists
				stateFile := filepath.Join(historyDir, goalID+".state.jsonl")
				if _, err := os.Stat(stateFile); err == nil {
					os.Rename(stateFile, filepath.Join(archiveDir, goalID+".state.jsonl"))
				}
			}
		}
	}

	return result
}

// CleanupGoal performs cleanup for a specific goal (remove worktree, optionally delete branch)
func (c *CleanupManager) CleanupGoal(goalID, project string, deleteBranch, force bool) error {
	workspacesDir := filepath.Join(c.vegaDir, "workspaces", project)
	worktreeBase := filepath.Join(workspacesDir, "worktree-base")

	// Find the worktree for this goal
	worktreePath := ""
	entries, err := os.ReadDir(workspacesDir)
	if err != nil {
		return fmt.Errorf("failed to read workspaces: %v", err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), fmt.Sprintf("goal-%s-", goalID)) {
			worktreePath = filepath.Join(workspacesDir, entry.Name())
			break
		}
	}

	if worktreePath == "" {
		return fmt.Errorf("worktree not found for goal %s", goalID)
	}

	// Check for uncommitted changes
	if !force {
		cmd := exec.Command("git", "-C", worktreePath, "status", "--porcelain")
		output, err := cmd.Output()
		if err == nil && len(strings.TrimSpace(string(output))) > 0 {
			return fmt.Errorf("worktree has uncommitted changes (use --force to override)")
		}
	}

	// Get branch name before removing worktree
	branchName := ""
	cmd := exec.Command("git", "-C", worktreePath, "branch", "--show-current")
	if output, err := cmd.Output(); err == nil {
		branchName = strings.TrimSpace(string(output))
	}

	// Remove worktree via git
	cmd = exec.Command("git", "-C", worktreeBase, "worktree", "remove", worktreePath)
	if force {
		cmd = exec.Command("git", "-C", worktreeBase, "worktree", "remove", "--force", worktreePath)
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try removing directory directly as fallback
		if rmErr := os.RemoveAll(worktreePath); rmErr != nil {
			return fmt.Errorf("failed to remove worktree: %v - %s", err, string(output))
		}
	}

	// Delete branch if requested
	if deleteBranch && branchName != "" {
		// Delete local branch
		cmd = exec.Command("git", "-C", worktreeBase, "branch", "-D", branchName)
		cmd.Run() // Ignore errors

		// Delete remote branch (best effort)
		cmd = exec.Command("git", "-C", worktreeBase, "push", "origin", "--delete", branchName)
		cmd.Run() // Ignore errors
	}

	return nil
}

// RunStartupCleanup performs cleanup tasks on startup
func (c *CleanupManager) RunStartupCleanup() *CleanupResult {
	result := &CleanupResult{}

	// 1. Prune stale worktree references
	pruneResult := c.PruneStaleWorktrees()
	result.PrunedWorktrees = pruneResult.PrunedWorktrees
	result.Errors = append(result.Errors, pruneResult.Errors...)

	// 2. Find orphaned worktrees (report only, don't auto-remove)
	orphanResult := c.FindOrphanedWorktrees()
	result.OrphanedWorktrees = orphanResult.OrphanedWorktrees
	result.Errors = append(result.Errors, orphanResult.Errors...)

	return result
}

// getRegisteredWorktrees returns a set of paths that are registered with git worktree
func (c *CleanupManager) getRegisteredWorktrees(worktreeBase string) map[string]bool {
	registered := make(map[string]bool)

	cmd := exec.Command("git", "-C", worktreeBase, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return registered
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			registered[path] = true
		}
	}

	return registered
}

// hasGoalFile checks if a goal file exists for the given ID
func (c *CleanupManager) hasGoalFile(goalID string) bool {
	if goalID == "" {
		return false
	}

	// Check active goals
	activePath := filepath.Join(c.vegaDir, "goals", "active", goalID+".md")
	if _, err := os.Stat(activePath); err == nil {
		return true
	}

	// Check iced goals
	icedPath := filepath.Join(c.vegaDir, "goals", "iced", goalID+".md")
	if _, err := os.Stat(icedPath); err == nil {
		return true
	}

	// Check history
	historyPath := filepath.Join(c.vegaDir, "goals", "history", goalID+".md")
	if _, err := os.Stat(historyPath); err == nil {
		return true
	}

	return false
}

// extractGoalIDFromPath extracts the goal ID from a worktree path like "goal-abc1234-some-title"
func extractGoalIDFromPath(path string) string {
	baseName := filepath.Base(path)
	if !strings.HasPrefix(baseName, "goal-") {
		return ""
	}

	// Format: goal-<7-char-id>-<slug>
	parts := strings.SplitN(baseName, "-", 3)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
