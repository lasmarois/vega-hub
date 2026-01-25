package project

import (
	"github.com/spf13/cobra"
)

// ProjectCmd represents the project command
var ProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Long: `Manage projects in the vega-missile system.

Available subcommands:
  add       Add a new project
  list      List all projects`,
}

func init() {
	// Subcommands are added in their respective files
}
