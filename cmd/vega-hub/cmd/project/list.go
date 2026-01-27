package project

import (
	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/operations"
	"github.com/spf13/cobra"
)

// ListResult contains the result of listing projects
type ListResult struct {
	Projects []ProjectInfo `json:"projects"`
	Count    int           `json:"count"`
}

// ProjectInfo contains project information for listing
type ProjectInfo struct {
	Name            string `json:"name"`
	BaseBranch      string `json:"base_branch"`
	Workspace       string `json:"workspace"`
	Upstream        string `json:"upstream"`
	WorkspaceStatus string `json:"workspace_status"`
	WorkspaceError  string `json:"workspace_error,omitempty"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects",
	Long: `List all registered projects in the vega-missile system.

Examples:
  vega-hub project list
  vega-hub project list --json`,
	Run: runList,
}

func init() {
	ProjectCmd.AddCommand(listCmd)
}

func runList(c *cobra.Command, args []string) {
	// Get vega-missile directory
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	// List projects
	projects, err := operations.ListProjects(vegaDir)
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "list_failed",
			"Failed to list projects",
			map[string]string{"error": err.Error()},
			nil)
	}

	// Convert to ProjectInfo
	infos := make([]ProjectInfo, 0, len(projects))
	for _, p := range projects {
		infos = append(infos, ProjectInfo{
			Name:            p.Name,
			BaseBranch:      p.BaseBranch,
			Workspace:       p.Workspace,
			Upstream:        p.Upstream,
			WorkspaceStatus: p.WorkspaceStatus,
			WorkspaceError:  p.WorkspaceError,
		})
	}

	result := ListResult{
		Projects: infos,
		Count:    len(infos),
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "project_list",
		Message: "",
		Data:    result,
	})

	// Human-readable output
	if !cli.JSONOutput {
		if len(infos) == 0 {
			cli.Info("No projects found")
			return
		}

		cli.Info("Projects (%d):", len(infos))
		for _, p := range infos {
			status := "✓"
			if p.WorkspaceStatus == "missing" {
				status = "⚠"
			} else if p.WorkspaceStatus == "error" {
				status = "✗"
			}
			cli.Info("  %s %s (branch: %s)", status, p.Name, p.BaseBranch)
		}
	}
}
