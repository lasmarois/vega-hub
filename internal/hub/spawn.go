package hub

import (
	"crypto/rand"
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
// It handles lifecycle tracking directly (register before start, stop on exit)
// because Claude's -p mode doesn't fire SessionStart/Stop hooks reliably.
func (h *Hub) SpawnExecutor(req SpawnRequest) SpawnResult {
	// Lock spawn to prevent concurrent spawns for same goal
	h.spawnMu.Lock()
	defer h.spawnMu.Unlock()

	// Check if an executor is already running for this goal
	h.mu.RLock()
	for _, e := range h.executors {
		if e.GoalID == req.GoalID {
			h.mu.RUnlock()
			return SpawnResult{
				Success: false,
				Message: fmt.Sprintf("Executor already running for Goal #%d (session: %s)", req.GoalID, e.SessionID),
			}
		}
	}
	h.mu.RUnlock()

	// Find the worktree for this goal
	worktree, err := h.findWorktree(req.GoalID)
	if err != nil {
		return SpawnResult{
			Success: false,
			Message: "Failed to find worktree: " + err.Error(),
		}
	}

	// Generate session ID for tracking
	sessionID := generateSessionID()

	// Build the prompt
	prompt := "Continue working on your assigned goal."
	if req.Context != "" {
		prompt = req.Context
	}

	// Build the command
	args := []string{
		"--allowedTools", "Read,Write,Edit,Bash,Skill,Glob,Grep,Task,AskUserQuestion",
		"--permission-mode", "dontAsk",
		"-p", prompt,
	}

	// Spawn Claude in the background
	// exec.Command inherits environment from vega-hub process,
	// so executor runs as the same user with same PATH/HOME/etc.
	cmd := exec.Command("claude", args...)
	cmd.Dir = worktree
	cmd.Env = os.Environ() // Explicitly inherit full environment

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

	// Register executor with vega-hub (don't rely on hooks)
	h.RegisterExecutor(req.GoalID, sessionID, worktree)

	// Monitor process and notify when done
	go func() {
		cmd.Wait()
		outFile.Close()
		// Notify vega-hub that executor stopped
		h.StopExecutor(req.GoalID, sessionID, "completed")
	}()

	return SpawnResult{
		Success:   true,
		Message:   fmt.Sprintf("Executor spawned for Goal #%d (PID: %d)", req.GoalID, cmd.Process.Pid),
		Worktree:  worktree,
		SessionID: sessionID,
	}
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
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
