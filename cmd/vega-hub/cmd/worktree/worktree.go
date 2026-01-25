package worktree

import (
	"github.com/spf13/cobra"
)

// WorktreeCmd is the parent command for worktree operations
var WorktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Manage git worktrees for goals",
	Long:  `Commands for creating, removing, and inspecting git worktrees associated with goals.`,
}

func init() {
	// Subcommands are added in their respective files
}
