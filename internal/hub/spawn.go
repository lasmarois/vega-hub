package hub

import (
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

// ValidModes defines the allowed executor modes
var ValidModes = map[string]bool{
	"plan":      true,
	"implement": true,
	"review":    true,
	"test":      true,
	"security":  true,
	"quick":     true,
}

// ValidExecutorTypes defines the allowed executor types
var ValidExecutorTypes = map[string]bool{
	"meta":    true,
	"project": true,
	"manager": true,
}

// SpawnRequest contains parameters for spawning an executor
type SpawnRequest struct {
	GoalID  string `json:"goal_id"`
	Context string `json:"context,omitempty"` // Optional additional context/instructions
	User    string `json:"user,omitempty"`    // Username spawning this executor (auto-detected if empty)
	Mode    string `json:"mode,omitempty"`    // Executor mode: plan, implement, review, test, security, quick
	Meta    bool   `json:"meta,omitempty"`    // If true, spawn as meta-executor in goal folder
	Project string `json:"project,omitempty"` // Project name for project executor (required if not meta)
}

// SpawnResult contains the result of spawning an executor
type SpawnResult struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	Worktree     string `json:"worktree,omitempty"`      // For project executors
	GoalFolder   string `json:"goal_folder,omitempty"`   // For meta-executors
	SessionID    string `json:"session_id,omitempty"`
	User         string `json:"user,omitempty"`          // Username who spawned this executor
	ExecutorType string `json:"executor_type,omitempty"` // "meta" or "project"
}

// SpawnExecutor spawns a new Claude executor for a goal.
// It handles lifecycle tracking directly (register before start, stop on exit)
// because Claude's -p mode doesn't fire SessionStart/Stop hooks reliably.
//
// Two executor types are supported:
//   - meta: Spawns in goals/active/<goal-id>/ folder (orchestrates multi-project goals)
//   - project: Spawns in worktree workspaces/<project>/goal-<id>-<slug>/ (works on single project)
func (h *Hub) SpawnExecutor(req SpawnRequest) SpawnResult {
	// Validate mutually exclusive flags
	if req.Meta && req.Project != "" {
		return SpawnResult{
			Success: false,
			Message: "--meta and --project are mutually exclusive",
		}
	}

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
				Message: fmt.Sprintf("Executor already running for Goal #%s (session: %s)", req.GoalID, e.SessionID),
			}
		}
	}
	h.mu.RUnlock()

	// Determine executor type and working directory
	var workDir string
	var executorType string
	var err error

	if req.Meta {
		// Meta-executor: spawn in goal folder
		executorType = "meta"
		workDir, err = h.findGoalFolder(req.GoalID)
		if err != nil {
			return SpawnResult{
				Success: false,
				Message: "Failed to find goal folder: " + err.Error(),
			}
		}
	} else {
		// Project executor: spawn in worktree
		executorType = "project"
		if req.Project != "" {
			// Find worktree for specific project
			workDir, err = h.findWorktreeForProject(req.GoalID, req.Project)
		} else {
			// Legacy behavior: find any worktree for this goal
			workDir, err = h.findWorktree(req.GoalID)
		}
		if err != nil {
			return SpawnResult{
				Success: false,
				Message: "Failed to find worktree: " + err.Error(),
			}
		}
	}

	// Generate session ID for tracking
	sessionID := generateSessionID()

	// Detect or use provided user
	username := req.User
	if username == "" {
		if u, err := user.Current(); err == nil {
			username = u.Username
		}
	}

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
	cmd.Dir = workDir

	// Build environment: inherit current env + inject vega-hub vars
	// This allows executor hooks to communicate with vega-hub and know their role/mode
	env := os.Environ()
	if h.port > 0 {
		env = append(env, fmt.Sprintf("VEGA_HUB_PORT=%d", h.port))
	}
	// Inject executor type (meta/project) for hook role detection
	env = append(env, fmt.Sprintf("VEGA_EXECUTOR_TYPE=%s", executorType))
	// Inject goal ID
	env = append(env, fmt.Sprintf("VEGA_GOAL_ID=%s", req.GoalID))
	// Inject mode if specified (validated before spawn)
	if req.Mode != "" {
		env = append(env, fmt.Sprintf("VEGA_EXECUTOR_MODE=%s", req.Mode))
		// Also keep legacy VEGA_HUB_MODE for backwards compatibility
		env = append(env, fmt.Sprintf("VEGA_HUB_MODE=%s", req.Mode))
	}
	// Inject project for project executors
	if req.Project != "" {
		env = append(env, fmt.Sprintf("VEGA_PROJECT=%s", req.Project))
	}
	cmd.Env = env

	// Redirect output to log file
	logFile := filepath.Join(workDir, ".executor-output.log")
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
	h.RegisterExecutor(req.GoalID, sessionID, workDir, username)

	// Monitor process and notify when done
	go func() {
		cmd.Wait()
		outFile.Close()
		// Notify vega-hub that executor stopped
		h.StopExecutor(req.GoalID, sessionID, "completed")
	}()

	// Build result
	result := SpawnResult{
		Success:      true,
		Message:      fmt.Sprintf("%s executor spawned for Goal #%s (PID: %d)", executorType, req.GoalID, cmd.Process.Pid),
		SessionID:    sessionID,
		User:         username,
		ExecutorType: executorType,
	}

	if req.Meta {
		result.GoalFolder = workDir
	} else {
		result.Worktree = workDir
	}

	return result
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// findWorktree finds any worktree directory for a goal (legacy behavior)
func (h *Hub) findWorktree(goalID string) (string, error) {
	workspacesDir := filepath.Join(h.dir, "workspaces")

	// List all project directories
	projects, err := os.ReadDir(workspacesDir)
	if err != nil {
		return "", fmt.Errorf("failed to read workspaces: %w", err)
	}

	goalPrefix := fmt.Sprintf("goal-%s-", goalID)

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

	return "", fmt.Errorf("no worktree found for goal %s", goalID)
}

// findWorktreeForProject finds the worktree directory for a specific goal+project combination
func (h *Hub) findWorktreeForProject(goalID, project string) (string, error) {
	projectPath := filepath.Join(h.dir, "workspaces", project)

	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return "", fmt.Errorf("project %s not found in workspaces: %w", project, err)
	}

	goalPrefix := fmt.Sprintf("goal-%s-", goalID)

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), goalPrefix) {
			return filepath.Join(projectPath, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("no worktree found for goal %s in project %s", goalID, project)
}

// findGoalFolder finds the goal folder for a meta-executor
// Goal folders are in: goals/active/<goal-id>/
func (h *Hub) findGoalFolder(goalID string) (string, error) {
	// Check folder structure first: goals/active/<goal-id>/
	folderPath := filepath.Join(h.dir, "goals", "active", goalID)
	if info, err := os.Stat(folderPath); err == nil && info.IsDir() {
		return folderPath, nil
	}

	// Check if flat file exists (goals/active/<goal-id>.md) - create folder structure
	flatFilePath := filepath.Join(h.dir, "goals", "active", goalID+".md")
	if _, err := os.Stat(flatFilePath); err == nil {
		// Goal exists as flat file - need to create folder structure for meta-executor
		if err := os.MkdirAll(folderPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create goal folder: %w", err)
		}
		return folderPath, nil
	}

	return "", fmt.Errorf("goal %s not found in goals/active/", goalID)
}

// GetWorktreePath returns the worktree path for a goal, if it exists
func (h *Hub) GetWorktreePath(goalID string) (string, error) {
	return h.findWorktree(goalID)
}

// GetGoalFolderPath returns the goal folder path for a meta-executor
func (h *Hub) GetGoalFolderPath(goalID string) (string, error) {
	return h.findGoalFolder(goalID)
}
