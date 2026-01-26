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
	spawnPrompt string
	spawnMode   string
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
	GoalID    string `json:"goal_id"`
	SessionID string `json:"session_id"`
	Worktree  string `json:"worktree"`
	Message   string `json:"message"`
	User      string `json:"user,omitempty"` // Username who spawned this executor
}

// SpawnRequest is the request body for the spawn API
type SpawnRequest struct {
	Context string `json:"context,omitempty"`
	User    string `json:"user,omitempty"` // Username spawning this executor
	Mode    string `json:"mode,omitempty"` // Executor mode: plan, implement, review, test, security, quick
}

// SpawnResponse is the response from the spawn API
type SpawnResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Worktree  string `json:"worktree,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	User      string `json:"user,omitempty"`
	Error     string `json:"error,omitempty"`
}

var spawnCmd = &cobra.Command{
	Use:   "spawn <goal-id>",
	Short: "Spawn a new executor for a goal",
	Long: `Spawn a Claude Code executor to work on a goal.

Examples:
  vega-hub executor spawn f3a8b2c
  vega-hub executor spawn f3a8b2c --prompt "Focus on Phase 2"
  vega-hub executor spawn f3a8b2c --mode plan
  vega-hub executor spawn f3a8b2c --mode implement

Available modes:
  plan      - Create implementation plan (task_plan.md, findings.md)
  implement - Write code and tests (default behavior)
  review    - Code review and feedback
  test      - Write and run tests
  security  - Security audit
  quick     - Answer questions (no planning files)

The executor will:
  1. Start in the goal's worktree directory
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
		cli.OutputError(cli.ExitStateError, "spawn_failed",
			spawnResp.Message,
			map[string]string{
				"goal_id": goalID,
				"error":   spawnResp.Error,
			},
			[]cli.ErrorOption{
				{Action: "check", Description: "Verify goal exists and has a worktree"},
			})
	}

	// Success output
	result := SpawnResult{
		GoalID:    goalID,
		SessionID: spawnResp.SessionID,
		Worktree:  spawnResp.Worktree,
		Message:   spawnResp.Message,
		User:      spawnResp.User,
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "executor_spawn",
		Message: fmt.Sprintf("Spawned executor for goal %s", goalID),
		Data:    result,
		NextSteps: []string{
			"Monitor via: vega-hub executor list",
			"Stop with: vega-hub executor stop " + goalID,
		},
	})

	// Human-readable summary
	if !cli.JSONOutput {
		fmt.Printf("\n  Session: %s\n", spawnResp.SessionID)
		fmt.Printf("  Worktree: %s\n", spawnResp.Worktree)
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
