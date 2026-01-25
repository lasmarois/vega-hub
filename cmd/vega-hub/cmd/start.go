package cmd

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/spf13/cobra"
)

// StartResult contains the result of a start operation
type StartResult struct {
	Status string `json:"status"` // "started", "already_running"
	Port   int    `json:"port"`
	PID    int    `json:"pid"`
	URL    string `json:"url"`
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start vega-hub in daemon mode",
	Long: `Start vega-hub as a background daemon.

If vega-hub is already running, returns the existing instance's status.
Automatically finds an available port in the range 8080-8089.

Creates the following files in the vega-missile directory:
  .vega-hub.pid   - Process ID
  .vega-hub.port  - Port number

Example:
  vega-hub start --dir /path/to/vega-missile
  vega-hub start  # auto-detect directory`,
	Run: runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) {
	dir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	// Check if already running
	if result, running := checkRunning(dir); running {
		cli.OutputSuccess("already_running", "vega-hub is already running", result)
		return
	}

	// Find available port
	port, err := findAvailablePort(8080, 8089)
	if err != nil {
		cli.OutputError(cli.ExitStateError, "no_port_available",
			"Could not find available port in range 8080-8089",
			map[string]string{"range": "8080-8089"},
			nil)
	}

	// Start daemon process
	pid, err := startDaemon(dir, port)
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "start_failed", err.Error(), nil, nil)
	}

	// Wait for health check
	if err := waitForHealthy(port, 10*time.Second); err != nil {
		cli.OutputError(cli.ExitInternalError, "health_check_failed",
			"Daemon started but health check failed",
			map[string]string{"port": strconv.Itoa(port), "pid": strconv.Itoa(pid)},
			nil)
	}

	// Write pid and port files
	writePidFile(dir, pid)
	writePortFile(dir, port)

	result := StartResult{
		Status: "started",
		Port:   port,
		PID:    pid,
		URL:    fmt.Sprintf("http://localhost:%d", port),
	}

	cli.OutputSuccess("started", fmt.Sprintf("vega-hub started on port %d", port), result)
}

func checkRunning(dir string) (*StartResult, bool) {
	portFile := filepath.Join(dir, ".vega-hub.port")
	pidFile := filepath.Join(dir, ".vega-hub.pid")

	// Read port file
	portData, err := os.ReadFile(portFile)
	if err != nil {
		return nil, false
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(portData)))
	if err != nil {
		return nil, false
	}

	// Check health endpoint
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/health", port))
	if err != nil {
		// Not responding, clean up stale files
		os.Remove(portFile)
		os.Remove(pidFile)
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false
	}

	// Read PID
	pid := 0
	if pidData, err := os.ReadFile(pidFile); err == nil {
		pid, _ = strconv.Atoi(strings.TrimSpace(string(pidData)))
	}

	return &StartResult{
		Status: "already_running",
		Port:   port,
		PID:    pid,
		URL:    fmt.Sprintf("http://localhost:%d", port),
	}, true
}

func findAvailablePort(start, end int) (int, error) {
	for port := start; port <= end; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port in range %d-%d", start, end)
}

func startDaemon(dir string, port int) (int, error) {
	// Get path to current executable
	exe, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("could not find executable: %w", err)
	}

	// Start as daemon using serve command
	cmd := exec.Command(exe, "serve", "--port", strconv.Itoa(port), "--dir", dir)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create new session (detach from terminal)
	}

	// Redirect output to log file
	logFile := filepath.Join(dir, ".vega-hub.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return 0, fmt.Errorf("could not open log file: %w", err)
	}
	cmd.Stdout = f
	cmd.Stderr = f

	if err := cmd.Start(); err != nil {
		f.Close()
		return 0, fmt.Errorf("could not start daemon: %w", err)
	}

	// Don't wait for process - it's a daemon
	go func() {
		cmd.Wait()
		f.Close()
	}()

	return cmd.Process.Pid, nil
}

func waitForHealthy(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("http://localhost:%d/api/health", port)

	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("health check timed out after %v", timeout)
}

func writePidFile(dir string, pid int) error {
	return os.WriteFile(
		filepath.Join(dir, ".vega-hub.pid"),
		[]byte(strconv.Itoa(pid)+"\n"),
		0644,
	)
}

func writePortFile(dir string, port int) error {
	return os.WriteFile(
		filepath.Join(dir, ".vega-hub.port"),
		[]byte(strconv.Itoa(port)+"\n"),
		0644,
	)
}
