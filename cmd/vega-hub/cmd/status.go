package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// StatusResult contains the status of vega-hub
type StatusResult struct {
	Running bool   `json:"running"`
	Port    int    `json:"port,omitempty"`
	PID     int    `json:"pid,omitempty"`
	URL     string `json:"url,omitempty"`
	Uptime  string `json:"uptime,omitempty"`
	Dir     string `json:"dir"`
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if vega-hub is running",
	Long: `Check the status of the vega-hub daemon.

Reports whether vega-hub is running, and if so, provides:
  - Port number
  - Process ID
  - URL
  - Directory being managed

Example:
  vega-hub status --dir /path/to/vega-missile
  vega-hub status  # auto-detect directory`,
	Run: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) {
	dir, err := GetVegaDir()
	if err != nil {
		OutputError(ExitValidationError, "no_directory", err.Error(), nil, []ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	result := StatusResult{
		Dir: dir,
	}

	portFile := filepath.Join(dir, ".vega-hub.port")
	pidFile := filepath.Join(dir, ".vega-hub.pid")

	// Read port file
	portData, err := os.ReadFile(portFile)
	if err != nil {
		result.Running = false
		OutputSuccess("status", "vega-hub is not running", result)
		return
	}

	port, err := strconv.Atoi(strings.TrimSpace(string(portData)))
	if err != nil {
		result.Running = false
		OutputSuccess("status", "vega-hub is not running (invalid port file)", result)
		return
	}

	// Check health endpoint
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/health", port))
	if err != nil {
		// Not responding, clean up stale files
		os.Remove(portFile)
		os.Remove(pidFile)
		result.Running = false
		OutputSuccess("status", "vega-hub is not running (stale files cleaned)", result)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		result.Running = false
		OutputSuccess("status", "vega-hub is not responding correctly", result)
		return
	}

	// Read PID
	pid := 0
	if pidData, err := os.ReadFile(pidFile); err == nil {
		pid, _ = strconv.Atoi(strings.TrimSpace(string(pidData)))
	}

	// Calculate uptime from pid file modification time
	uptime := ""
	if pidInfo, err := os.Stat(pidFile); err == nil {
		duration := time.Since(pidInfo.ModTime())
		uptime = formatDuration(duration)
	}

	result.Running = true
	result.Port = port
	result.PID = pid
	result.URL = fmt.Sprintf("http://localhost:%d", port)
	result.Uptime = uptime

	OutputSuccess("status", fmt.Sprintf("vega-hub is running on port %d", port), result)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	return fmt.Sprintf("%dd%dh", days, hours)
}
