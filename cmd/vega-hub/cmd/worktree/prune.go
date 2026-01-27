package worktree

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/hub"
	"github.com/spf13/cobra"
)

var (
	pruneOrphans bool
	pruneForce   bool
	pruneRemove  bool
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Clean up stale and orphaned worktrees",
	Long: `Prune stale worktree references and optionally remove orphaned worktrees.

By default, runs 'git worktree prune' on all projects to clean up stale references.

With --orphans, also finds worktrees that exist on disk but aren't properly registered.
With --remove, removes orphaned worktrees (use with --force to skip safety checks).`,
	RunE: runPrune,
}

func init() {
	WorktreeCmd.AddCommand(pruneCmd)
	pruneCmd.Flags().BoolVar(&pruneOrphans, "orphans", false, "Find orphaned worktrees")
	pruneCmd.Flags().BoolVar(&pruneRemove, "remove", false, "Remove orphaned worktrees (requires --orphans)")
	pruneCmd.Flags().BoolVar(&pruneForce, "force", false, "Force removal even with uncommitted changes")
}

func runPrune(cmd *cobra.Command, args []string) error {
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		return err
	}

	manager := hub.NewCleanupManager(vegaDir)
	result := manager.PruneStaleWorktrees()

	// Find orphans if requested
	if pruneOrphans {
		orphanResult := manager.FindOrphanedWorktrees()
		result.OrphanedWorktrees = orphanResult.OrphanedWorktrees
		result.Errors = append(result.Errors, orphanResult.Errors...)

		// Remove orphans if requested
		if pruneRemove && len(result.OrphanedWorktrees) > 0 {
			var removed []string
			var remaining []hub.OrphanedWorktree

			for _, orphan := range result.OrphanedWorktrees {
				if err := manager.RemoveOrphanedWorktree(orphan.Path, pruneForce); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("Failed to remove %s: %v", orphan.Path, err))
					remaining = append(remaining, orphan)
				} else {
					removed = append(removed, orphan.GoalID)
				}
			}

			result.OrphanedWorktrees = remaining
			if len(removed) > 0 {
				result.PrunedWorktrees = append(result.PrunedWorktrees, removed...)
			}
		}
	}

	// JSON output
	if cli.JSONOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	// Human-readable output
	if len(result.PrunedWorktrees) > 0 {
		fmt.Println("✓ Pruned stale worktree references:")
		for _, p := range result.PrunedWorktrees {
			fmt.Printf("  - %s\n", p)
		}
	} else {
		fmt.Println("✓ No stale worktree references found")
	}

	if pruneOrphans {
		fmt.Println()
		if len(result.OrphanedWorktrees) > 0 {
			if pruneRemove {
				fmt.Println("⚠ Orphaned worktrees that could not be removed:")
			} else {
				fmt.Println("⚠ Orphaned worktrees found:")
			}
			for _, o := range result.OrphanedWorktrees {
				fmt.Printf("  - %s (%s)\n", o.Path, o.Reason)
			}
			if !pruneRemove {
				fmt.Println()
				fmt.Println("To remove: vega-hub worktree prune --orphans --remove")
			}
		} else {
			fmt.Println("✓ No orphaned worktrees found")
		}
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
