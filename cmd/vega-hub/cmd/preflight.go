package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/hub"
	"github.com/spf13/cobra"
)

var preflightCmd = &cobra.Command{
	Use:   "preflight <project>",
	Short: "Run pre-flight checks before goal creation",
	Long: `Validate environment before creating a goal.

Checks:
  - Worktree-base is clean (no uncommitted changes)
  - Worktree-base is synced with remote
  - Sufficient disk space (1GB minimum)
  - Git credentials are valid
  - No rebase/merge in progress

Returns exit code 0 if all checks pass, 1 if any fail.`,
	Args: cobra.ExactArgs(1),
	RunE: runPreflight,
}

func init() {
	rootCmd.AddCommand(preflightCmd)
}

func runPreflight(cmd *cobra.Command, args []string) error {
	projectName := args[0]

	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		return err
	}

	result, err := hub.RunPreflightForProject(vegaDir, projectName)
	if err != nil {
		return err
	}

	if cli.JSONOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Human-readable output
	if result.Ready {
		fmt.Println("✓ All pre-flight checks passed")
		fmt.Println()
		printCheckResults(result)
		return nil
	}

	fmt.Println("✗ Pre-flight checks failed")
	fmt.Println()
	printCheckResults(result)

	if len(result.FixCommands) > 0 {
		fmt.Println()
		fmt.Println("To fix:")
		for _, cmd := range result.FixCommands {
			fmt.Printf("  $ %s\n", cmd)
		}
	}

	os.Exit(1)
	return nil
}

func printCheckResults(result *hub.PreflightResult) {
	checkOrder := []string{
		"worktree_clean",
		"worktree_synced",
		"disk_space",
		"credentials",
		"branch_available",
		"no_in_progress_ops",
	}

	checkNames := map[string]string{
		"worktree_clean":     "Worktree clean",
		"worktree_synced":    "Worktree synced",
		"disk_space":         "Disk space",
		"credentials":        "Git credentials",
		"branch_available":   "Branch available",
		"no_in_progress_ops": "No ops in progress",
	}

	for _, key := range checkOrder {
		check, exists := result.Checks[key]
		if !exists {
			continue
		}

		name := checkNames[key]
		if check.Passed {
			fmt.Printf("  ✓ %s", name)
			// Add extra info for some checks
			if key == "disk_space" && check.AvailMB > 0 {
				fmt.Printf(" (%d MB available)", check.AvailMB)
			}
			if key == "worktree_synced" {
				fmt.Print(" (up to date)")
			}
			fmt.Println()
		} else {
			fmt.Printf("  ✗ %s: %s\n", name, check.Error)
		}
	}
}
