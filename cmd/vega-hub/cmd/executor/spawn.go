package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/spf13/cobra"
)

var (
	spawnPrompt  string
	spawnMode    string
	spawnMeta    bool
	spawnProject string
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

// SpawnResult contains the result of spawning an executor
type SpawnResult struct {
	GoalID       string `json:"goal_id"`
	SessionID    string `json:"session_id"`
	Worktree     string `json:"worktree,omitempty"`     // For project executors
	GoalFolder   string `json:"goal_folder,omitempty"` // For meta-executors
	Message      string `json:"message"`
	User         string `json:"user,omitempty"`         // Username who spawned this executor
	ExecutorType string `json:"executor_type,omitempty"` // "meta" or "project"
}

// SpawnRequest is the request body for the spawn API
type SpawnRequest struct {
	Context string `json:"context,omitempty"`
	User    string `json:"user,omitempty"`    // Username spawning this executor
	Mode    string `json:"mode,omitempty"`    // Executor mode: plan, implement, review, test, security, quick
	Meta    bool   `json:"meta,omitempty"`    // If true, spawn as meta-executor in goal folder
	Project string `json:"project,omitempty"` // Project name for project executor
}

// SpawnResponse is the response from the spawn API
type SpawnResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	Worktree     string `json:"worktree,omitempty"`      // For project executors
	GoalFolder   string `json:"goal_folder,omitempty"`   // For meta-executors
	SessionID    string `json:"session_id,omitempty"`
	User         string `json:"user,omitempty"`
	ExecutorType string `json:"executor_type,omitempty"` // "meta" or "project"
	Error        string `json:"error,omitempty"`
}

var spawnCmd = &cobra.Command{
	Use:   "spawn <goal-id>",
	Short: "Spawn a new executor for a goal",
	Long: `Spawn a Claude Code executor to work on a goal.

Two executor types are supported:
  - meta: Spawns in goal folder (orchestrates multi-project goals)
  - project: Spawns in worktree (works on a single project)

Examples:
  # Spawn meta-executor for goal orchestration
  vega-hub executor spawn f3a8b2c --meta

  # Spawn project executor for specific project
  vega-hub executor spawn f3a8b2c --project my-api

  # Spawn with mode and prompt
  vega-hub executor spawn f3a8b2c --project my-api --mode plan
  vega-hub executor spawn f3a8b2c --meta --prompt "Orchestrate this multi-project goal"

Available modes:
  plan      - Create implementation plan (task_plan.md, findings.md)
  implement - Write code and tests (default behavior)
  review    - Code review and feedback
  test      - Write and run tests
  security  - Security audit
  quick     - Answer questions (no planning files)

Environment variables set for executor:
  VEGA_EXECUTOR_TYPE  - "meta" or "project"
  VEGA_EXECUTOR_MODE  - The mode (if --mode specified)
  VEGA_GOAL_ID        - The goal ID
  VEGA_PROJECT        - Project name (project executors only)
  VEGA_HUB_PORT       - Port for vega-hub communication

The executor will:
  1. Start in the goal folder (meta) or worktree (project)
  2. Load context from inherited .claude/ rules
  3. Receive mode-specific instructions (if --mode specified)
  4. Work autonomously on the goal
  5. Communicate via vega-hub for questions

NOTE: vega-hub must be running. Use 'vega-hub start' first.`,
	Args: cobra.ExactArgs(1),
	Run:  runSpawn,
}

func init() {
	ExecutorCmd.AddCommand(spawnCmd)
	spawnCmd.Flags().StringVarP(&spawnPrompt, "prompt", "p", "", "Custom prompt/context for the executor")
	spawnCmd.Flags().StringVarP(&spawnMode, "mode", "m", "", "Executor mode: plan, implement, review, test, security, quick")
	spawnCmd.Flags().BoolVar(&spawnMeta, "meta", false, "Spawn as meta-executor in goal folder (not worktree)")
	spawnCmd.Flags().StringVar(&spawnProject, "project", "", "Project name for project executor (required if not --meta)")
}

func runSpawn(c *cobra.Command, args []string) {
	goalID := args[0]

	// Get vega-missile directory
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	// Get vega-hub port
	port, err := getVegaHubPort(vegaDir)
	if err != nil {
		cli.OutputError(cli.ExitStateError, "vega_hub_not_running",
			"vega-hub is not running",
			map[string]string{"error": err.Error()},
			[]cli.ErrorOption{
				{Action: "start", Description: "Run: vega-hub start"},
			})
	}

	// Validate mutually exclusive flags
	if spawnMeta && spawnProject != "" {
		cli.OutputError(cli.ExitValidationError, "invalid_flags",
			"--meta and --project are mutually exclusive",
			nil,
			[]cli.ErrorOption{
				{Flag: "--meta", Description: "Spawn as meta-executor in goal folder"},
				{Flag: "--project", Description: "Spawn as project executor in worktree"},
			})
	}

	// Validate mode if specified
	if spawnMode != "" && !ValidModes[spawnMode] {
		validModesList := "plan, implement, review, test, security, quick"
		cli.OutputError(cli.ExitValidationError, "invalid_mode",
			fmt.Sprintf("Invalid mode: %s", spawnMode),
			map[string]string{"valid_modes": validModesList},
			nil)
	}

	// Detect current user
	var username string
	if u, err := user.Current(); err == nil {
		username = u.Username
	}

	// Build request
	reqBody := SpawnRequest{
		Context: spawnPrompt,
		User:    username,
		Mode:    spawnMode,
		Meta:    spawnMeta,
		Project: spawnProject,
	}
	if reqBody.Context == "" {
		reqBody.Context = "Continue working on your assigned goal."
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "json_error",
			"Failed to encode request",
			map[string]string{"error": err.Error()},
			nil)
	}

	// Call spawn API
	url := fmt.Sprintf("http://localhost:%d/api/goals/%s/spawn", port, goalID)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		cli.OutputError(cli.ExitStateError, "api_error",
			"Failed to connect to vega-hub",
			map[string]string{
				"url":   url,
				"error": err.Error(),
			},
			[]cli.ErrorOption{
				{Action: "check", Description: "Verify vega-hub is running"},
			})
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "read_error",
			"Failed to read response",
			map[string]string{"error": err.Error()},
			nil)
	}

	// Parse response
	var spawnResp SpawnResponse
	if err := json.Unmarshal(body, &spawnResp); err != nil {
		cli.OutputError(cli.ExitInternalError, "parse_error",
			"Failed to parse response",
			map[string]string{
				"error":    err.Error(),
				"response": string(body),
			},
			nil)
	}

	// Handle error response
	if !spawnResp.Success {
		hint := "Verify goal exists"
		if spawnMeta {
			hint += " in goals/active/"
		} else {
			hint += " and has a worktree"
		}
		cli.OutputError(cli.ExitStateError, "spawn_failed",
			spawnResp.Message,
			map[string]string{
				"goal_id": goalID,
				"error":   spawnResp.Error,
			},
			[]cli.ErrorOption{
				{Action: "check", Description: hint},
			})
	}

	// Success output
	result := SpawnResult{
		GoalID:       goalID,
		SessionID:    spawnResp.SessionID,
		Worktree:     spawnResp.Worktree,
		GoalFolder:   spawnResp.GoalFolder,
		Message:      spawnResp.Message,
		User:         spawnResp.User,
		ExecutorType: spawnResp.ExecutorType,
	}

	executorTypeLabel := "project"
	if spawnMeta {
		executorTypeLabel = "meta"
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "executor_spawn",
		Message: fmt.Sprintf("Spawned %s executor for goal %s", executorTypeLabel, goalID),
		Data:    result,
		NextSteps: []string{
			"Monitor via: vega-hub executor list",
			"Stop with: vega-hub executor stop " + goalID,
		},
	})

	// Human-readable summary
	if !cli.JSONOutput {
		fmt.Printf("\n  Session: %s\n", spawnResp.SessionID)
		fmt.Printf("  Type: %s\n", spawnResp.ExecutorType)
		if spawnResp.Worktree != "" {
			fmt.Printf("  Worktree: %s\n", spawnResp.Worktree)
		}
		if spawnResp.GoalFolder != "" {
			fmt.Printf("  Goal Folder: %s\n", spawnResp.GoalFolder)
		}
	}
}

// getVegaHubPort reads the port from .vega-hub.port file
func getVegaHubPort(vegaDir string) (int, error) {
	portFile := filepath.Join(vegaDir, ".vega-hub.port")
	data, err := os.ReadFile(portFile)
	if err != nil {
		return 0, fmt.Errorf("could not read port file: %w", err)
	}

	portStr := strings.TrimSpace(string(data))
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		return 0, fmt.Errorf("invalid port: %s", portStr)
	}

	return port, nil
}
