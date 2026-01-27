package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/hub"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check system health",
	Long: `Run health checks on the vega-hub system.

Checks:
  - Worktree bases (clean and fetchable)
  - Stuck goals (in intermediate states >1h)
  - Disk space (warns <5GB, fails <1GB)
  - Git credentials (remote access)
  - Orphaned worktrees
  - Stale locks

Returns exit code 0 if healthy, 1 if degraded, 2 if unhealthy.`,
	RunE: runHealth,
}

func init() {
	rootCmd.AddCommand(healthCmd)
}

func runHealth(cmd *cobra.Command, args []string) error {
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		return err
	}

	checker := hub.NewHealthChecker(vegaDir)
	result := checker.RunAllChecks()

	// JSON output
	if cli.JSONOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Human-readable output
	statusIcon := "✓"
	if result.Status == hub.HealthDegraded {
		statusIcon = "⚠"
	} else if result.Status == hub.HealthUnhealthy {
		statusIcon = "✗"
	}

	fmt.Printf("%s System status: %s\n\n", statusIcon, result.Status)

	// Print checks
	checkOrder := []string{
		"worktree_bases",
		"stuck_goals",
		"disk_space",
		"git_credentials",
		"orphaned_worktrees",
		"active_locks",
	}

	checkNames := map[string]string{
		"worktree_bases":     "Worktree bases",
		"stuck_goals":        "Stuck goals",
		"disk_space":         "Disk space",
		"git_credentials":    "Git credentials",
		"orphaned_worktrees": "Orphaned worktrees",
		"active_locks":       "Active locks",
	}

	for _, key := range checkOrder {
		check, exists := result.Checks[key]
		if !exists {
			continue
		}

		icon := "✓"
		if check.Status == hub.HealthDegraded {
			icon = "⚠"
		} else if check.Status == hub.HealthUnhealthy {
			icon = "✗"
		}

		fmt.Printf("  %s %s: %s\n", icon, checkNames[key], check.Message)
	}

	// Print issues if any
	if len(result.Issues) > 0 {
		fmt.Println()
		fmt.Println("Issues:")
		for _, issue := range result.Issues {
			severityIcon := "ℹ"
			if issue.Severity == "warning" {
				severityIcon = "⚠"
			} else if issue.Severity == "error" || issue.Severity == "critical" {
				severityIcon = "✗"
			}
			fmt.Printf("  %s [%s] %s\n", severityIcon, issue.Check, issue.Message)
		}
	}

	// Exit code based on status
	if result.Status == hub.HealthUnhealthy {
		os.Exit(2)
	} else if result.Status == hub.HealthDegraded {
		os.Exit(1)
	}

	return nil
}
