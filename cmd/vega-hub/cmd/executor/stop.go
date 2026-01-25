package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/spf13/cobra"
)

var (
	stopReason string
)

// StopResult contains the result of stopping an executor
type StopResult struct {
	GoalID    string `json:"goal_id"`
	SessionID string `json:"session_id,omitempty"`
	Reason    string `json:"reason"`
	Message   string `json:"message"`
}

// StopRequest is the request body for the stop API
type StopRequest struct {
	GoalID    string `json:"goal_id"`
	SessionID string `json:"session_id,omitempty"`
	Reason    string `json:"reason"`
}

// StopResponse is the response from the stop API
type StopResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

var stopCmd = &cobra.Command{
	Use:   "stop <goal-id>",
	Short: "Stop an executor",
	Long: `Stop (record as stopped) an executor for a goal.

Examples:
  vega-hub executor stop f3a8b2c
  vega-hub executor stop f3a8b2c --reason "User requested"

NOTE: This records the executor as stopped in vega-hub but does not
forcefully terminate the Claude process. The process will stop naturally
when it completes or when interrupted.

Use 'vega-hub executor list' to see active executors.`,
	Args: cobra.ExactArgs(1),
	Run:  runStop,
}

func init() {
	ExecutorCmd.AddCommand(stopCmd)
	stopCmd.Flags().StringVarP(&stopReason, "reason", "r", "cli_stop", "Reason for stopping")
}

func runStop(c *cobra.Command, args []string) {
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

	// First, find the session ID for this goal
	sessionID, err := findSessionForGoal(port, goalID)
	if err != nil {
		cli.OutputError(cli.ExitNotFound, "executor_not_found",
			fmt.Sprintf("No active executor found for goal %s", goalID),
			map[string]string{
				"goal_id": goalID,
				"error":   err.Error(),
			},
			[]cli.ErrorOption{
				{Action: "list", Description: "Run: vega-hub executor list"},
			})
	}

	// Build request
	reqBody := StopRequest{
		GoalID:    goalID,
		SessionID: sessionID,
		Reason:    stopReason,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "json_error",
			"Failed to encode request",
			map[string]string{"error": err.Error()},
			nil)
	}

	// Call stop API
	url := fmt.Sprintf("http://localhost:%d/api/executor/stop", port)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		cli.OutputError(cli.ExitStateError, "api_error",
			"Failed to connect to vega-hub",
			map[string]string{
				"url":   url,
				"error": err.Error(),
			},
			nil)
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
	var stopResp StopResponse
	if err := json.Unmarshal(body, &stopResp); err != nil {
		cli.OutputError(cli.ExitInternalError, "parse_error",
			"Failed to parse response",
			map[string]string{
				"error":    err.Error(),
				"response": string(body),
			},
			nil)
	}

	// Handle error response
	if !stopResp.Success {
		cli.OutputError(cli.ExitStateError, "stop_failed",
			stopResp.Message,
			map[string]string{
				"goal_id": goalID,
				"error":   stopResp.Error,
			},
			nil)
	}

	// Success output
	result := StopResult{
		GoalID:    goalID,
		SessionID: sessionID,
		Reason:    stopReason,
		Message:   stopResp.Message,
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "executor_stop",
		Message: fmt.Sprintf("Stopped executor for goal %s", goalID),
		Data:    result,
	})
}

// findSessionForGoal finds the session ID for a given goal
func findSessionForGoal(port int, goalID string) (string, error) {
	url := fmt.Sprintf("http://localhost:%d/api/executors", port)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var executors []APIExecutor
	if err := json.Unmarshal(body, &executors); err != nil {
		return "", err
	}

	for _, e := range executors {
		if e.GoalID == goalID {
			return e.SessionID, nil
		}
	}

	return "", fmt.Errorf("no executor for goal %s", goalID)
}
