package project

import (
	"fmt"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/operations"
	"github.com/spf13/cobra"
)

var (
	removeForce bool
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a project",
	Long: `Remove a project from vega-missile management.

This will:
  1. Remove the project config file (projects/<name>.md)
  2. Remove the project from projects/index.md
  3. Optionally remove the workspace directory (with --force)

Safety checks:
  - Warns if project has active goals
  - Requires --force if goals exist

Examples:
  vega-hub project remove my-api
  vega-hub project remove my-api --force`,
	Args: cobra.ExactArgs(1),
	Run:  runRemove,
}

func init() {
	ProjectCmd.AddCommand(removeCmd)
	removeCmd.Flags().BoolVarP(&removeForce, "force", "f", false, "Force removal even if goals exist")
}

func runRemove(c *cobra.Command, args []string) {
	name := args[0]

	// Get vega-missile directory
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	// Remove project
	result, data := operations.RemoveProject(operations.RemoveProjectOptions{
		Name:    name,
		Force:   removeForce,
		VegaDir: vegaDir,
	})

	if !result.Success {
		// Check if it's the goals exist error
		if result.Error != nil && result.Error.Code == "has_active_goals" {
			cli.OutputError(cli.ExitValidationError, result.Error.Code,
				result.Error.Message,
				result.Error.Details,
				[]cli.ErrorOption{
					{Flag: "--force", Description: "Force removal anyway"},
				})
		}
		cli.OutputError(cli.ExitInternalError, result.Error.Code,
			result.Error.Message,
			result.Error.Details,
			nil)
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "project_remove",
		Message: fmt.Sprintf("Removed project: %s", name),
		Data:    data,
	})

	// Human-readable summary
	if !cli.JSONOutput {
		cli.Info("Removed project: %s", name)
		if data.ConfigRemoved {
			cli.Info("  ✓ Removed config file")
		}
		if data.IndexUpdated {
			cli.Info("  ✓ Updated project index")
		}
		if data.WorkspaceRemoved {
			cli.Info("  ✓ Removed workspace directory")
		}
		if data.GoalsWarning != "" {
			cli.Warn("  ⚠ %s", data.GoalsWarning)
		}
	}
}
