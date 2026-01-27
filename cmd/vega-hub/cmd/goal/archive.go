package goal

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/hub"
	"github.com/spf13/cobra"
)

var (
	archiveOlderThan string
	archiveDryRun    bool
)

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Archive completed goals older than specified duration",
	Long: `Move completed goals from history/ to history/archive/ if they are older than the specified duration.

Duration format: <number><unit>
  - d: days (e.g., 30d = 30 days)
  - w: weeks (e.g., 4w = 4 weeks)
  - m: months (e.g., 6m = 6 months)

Examples:
  vega-hub goal archive --older-than 30d          # Archive goals completed > 30 days ago
  vega-hub goal archive --older-than 4w --dry-run # Preview what would be archived`,
	RunE: runArchive,
}

func init() {
	GoalCmd.AddCommand(archiveCmd)
	archiveCmd.Flags().StringVar(&archiveOlderThan, "older-than", "", "Archive goals older than this duration (e.g., 30d, 4w, 6m)")
	archiveCmd.Flags().BoolVar(&archiveDryRun, "dry-run", false, "Show what would be archived without making changes")
	archiveCmd.MarkFlagRequired("older-than")
}

func runArchive(cmd *cobra.Command, args []string) error {
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		return err
	}

	duration, err := parseDuration(archiveOlderThan)
	if err != nil {
		return fmt.Errorf("invalid duration: %v", err)
	}

	manager := hub.NewCleanupManager(vegaDir)
	result := manager.ArchiveCompletedGoals(duration, archiveDryRun)

	// JSON output
	if cli.JSONOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Human-readable output
	if archiveDryRun {
		fmt.Println("Dry run - no changes made")
		fmt.Println()
	}

	if len(result.ArchivedGoals) > 0 {
		if archiveDryRun {
			fmt.Println("Would archive:")
		} else {
			fmt.Println("✓ Archived goals:")
		}
		for _, g := range result.ArchivedGoals {
			fmt.Printf("  - %s\n", g)
		}
	} else {
		fmt.Printf("No completed goals older than %s found\n", archiveOlderThan)
	}

	if len(result.Errors) > 0 {
		fmt.Println()
		fmt.Println("Errors:")
		for _, e := range result.Errors {
			fmt.Printf("  ✗ %s\n", e)
		}
		os.Exit(1)
	}

	return nil
}

// parseDuration parses a duration string like "30d", "4w", "6m"
func parseDuration(s string) (time.Duration, error) {
	re := regexp.MustCompile(`^(\d+)([dwm])$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid format, use <number><unit> where unit is d/w/m (e.g., 30d, 4w, 6m)")
	}

	value, _ := strconv.Atoi(matches[1])
	unit := matches[2]

	switch unit {
	case "d":
		return time.Duration(value) * 24 * time.Hour, nil
	case "w":
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case "m":
		return time.Duration(value) * 30 * 24 * time.Hour, nil // Approximate month
	default:
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}
}
