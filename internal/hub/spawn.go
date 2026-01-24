package hub

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SpawnRequest contains parameters for spawning an executor
type SpawnRequest struct {
	GoalID  int    `json:"goal_id"`
	Context string `json:"context,omitempty"` // Optional additional context/instructions
}

// SpawnResult contains the result of spawning an executor
type SpawnResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Worktree  string `json:"worktree,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// SpawnExecutor spawns a new Claude executor in the goal's worktree
func (h *Hub) SpawnExecutor(req SpawnRequest) SpawnResult {
	// Find the worktree for this goal
	worktree, err := h.findWorktree(req.GoalID)
	if err != nil {
		return SpawnResult{
			Success: false,
			Message: "Failed to find worktree: " + err.Error(),
		}
	}

	// Build the prompt
	prompt := "Continue working on your assigned goal."
	if req.Context != "" {
		prompt = req.Context
	}

	// Build the command
	args := []string{
		"--allowedTools", "Read,Write,Edit,Bash,Skill,Glob,Grep,Task",
		"--permission-mode", "dontAsk",
		"-p", prompt,
	}

	// Spawn Claude in the background using nohup
	cmd := exec.Command("claude", args...)
	cmd.Dir = worktree

	// Redirect output to log file
	logFile := filepath.Join(worktree, ".executor-output.log")
	outFile, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return SpawnResult{
			Success: false,
			Message: "Failed to create log file: " + err.Error(),
		}
	}

	cmd.Stdout = outFile
	cmd.Stderr = outFile

	// Start the process in the background
	if err := cmd.Start(); err != nil {
		outFile.Close()
		return SpawnResult{
			Success: false,
			Message: "Failed to start executor: " + err.Error(),
		}
	}

	// Don't wait - let it run in background
	go func() {
		cmd.Wait()
		outFile.Close()
	}()

	return SpawnResult{
		Success:  true,
		Message:  fmt.Sprintf("Executor spawned for Goal #%d (PID: %d)", req.GoalID, cmd.Process.Pid),
		Worktree: worktree,
	}
}

// findWorktree finds the worktree directory for a goal
func (h *Hub) findWorktree(goalID int) (string, error) {
	workspacesDir := filepath.Join(h.dir, "workspaces")

	// List all project directories
	projects, err := os.ReadDir(workspacesDir)
	if err != nil {
		return "", fmt.Errorf("failed to read workspaces: %w", err)
	}

	goalPrefix := fmt.Sprintf("goal-%d-", goalID)

	for _, project := range projects {
		if !project.IsDir() {
			continue
		}

		projectPath := filepath.Join(workspacesDir, project.Name())
		entries, err := os.ReadDir(projectPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), goalPrefix) {
				return filepath.Join(projectPath, entry.Name()), nil
			}
		}
	}

	return "", fmt.Errorf("no worktree found for goal %d", goalID)
}

// GetWorktreePath returns the worktree path for a goal, if it exists
func (h *Hub) GetWorktreePath(goalID int) (string, error) {
	return h.findWorktree(goalID)
}
