package executor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/spf13/cobra"
)

var (
	listGoalFilter string
)

// ExecutorInfo contains information about an executor
type ExecutorInfo struct {
	SessionID string `json:"session_id"`
	GoalID    string `json:"goal_id"`
	Worktree  string `json:"worktree"`
	StartedAt string `json:"started_at"`
	Uptime    string `json:"uptime"`
}

// ListResult contains the result of listing executors
type ListResult struct {
	Total     int            `json:"total"`
	Executors []ExecutorInfo `json:"executors"`
}

// APIExecutor is the executor format from the API
type APIExecutor struct {
	SessionID string    `json:"session_id"`
	GoalID    string    `json:"goal_id"`
	CWD       string    `json:"cwd"`
	StartedAt time.Time `json:"started_at"`
	LogFile   string    `json:"log_file"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active executors",
	Long: `List all currently active Claude Code executors.

Examples:
  vega-hub executor list
  vega-hub executor list --goal f3a8b2c
  vega-hub executor list --json

Shows:
  - Session ID
  - Goal ID
  - Worktree path
  - Start time
  - Uptime

NOTE: vega-hub must be running.`,
	Run: runList,
}

func init() {
	ExecutorCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listGoalFilter, "goal", "g", "", "Filter by goal ID")
}

func runList(c *cobra.Command, args []string) {
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

	// Call list API
	url := fmt.Sprintf("http://localhost:%d/api/executors", port)
	resp, err := http.Get(url)
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
	var apiExecutors []APIExecutor
	if err := json.Unmarshal(body, &apiExecutors); err != nil {
		cli.OutputError(cli.ExitInternalError, "parse_error",
			"Failed to parse response",
			map[string]string{
				"error":    err.Error(),
				"response": string(body),
			},
			nil)
	}

	// Convert and filter
	var executors []ExecutorInfo
	now := time.Now()
	for _, e := range apiExecutors {
		// Filter by goal if specified
		if listGoalFilter != "" && e.GoalID != listGoalFilter {
			continue
		}

		uptime := now.Sub(e.StartedAt)
		executors = append(executors, ExecutorInfo{
			SessionID: e.SessionID,
			GoalID:    e.GoalID,
			Worktree:  e.CWD,
			StartedAt: e.StartedAt.Format(time.RFC3339),
			Uptime:    formatDuration(uptime),
		})
	}

	// Output result
	result := ListResult{
		Total:     len(executors),
		Executors: executors,
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "executor_list",
		Message: fmt.Sprintf("Found %d active executor(s)", len(executors)),
		Data:    result,
	})

	// Human-readable table
	if !cli.JSONOutput {
		printExecutorTable(executors)
	}
}

func printExecutorTable(executors []ExecutorInfo) {
	if len(executors) == 0 {
		fmt.Println("\nNo active executors.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "\nGOAL\tSESSION\tWORKTREE\tUPTIME")
	fmt.Fprintln(w, "----\t-------\t--------\t------")

	for _, e := range executors {
		// Truncate worktree for display
		worktree := e.Worktree
		if len(worktree) > 40 {
			worktree = "..." + worktree[len(worktree)-37:]
		}
		// Truncate session ID
		session := e.SessionID
		if len(session) > 12 {
			session = session[:12] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			e.GoalID,
			session,
			worktree,
			e.Uptime,
		)
	}
	w.Flush()
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
