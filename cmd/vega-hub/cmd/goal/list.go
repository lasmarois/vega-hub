package goal

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/spf13/cobra"
)

var (
	listProject string
	listStatus  string
)

// ListResult contains the result of listing goals
type ListResult struct {
	Goals []goals.Goal `json:"goals"`
	Total int          `json:"total"`
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List goals",
	Long: `List all goals from the registry.

Filter by project or status:
  vega-hub goal list --project my-api
  vega-hub goal list --status active
  vega-hub goal list --status iced
  vega-hub goal list --status completed

Use --json for structured output.`,
	Run: runList,
}

func init() {
	GoalCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listProject, "project", "p", "", "Filter by project name")
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status (active, iced, completed)")
}

func runList(c *cobra.Command, args []string) {
	dir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	parser := goals.NewParser(dir)
	allGoals, err := parser.ParseRegistry()
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "parse_failed",
			fmt.Sprintf("Failed to parse registry: %v", err),
			map[string]string{"path": dir + "/goals/REGISTRY.md"},
			nil)
	}

	// Filter goals
	filtered := filterGoals(allGoals, listProject, listStatus)

	result := ListResult{
		Goals: filtered,
		Total: len(filtered),
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "goal_list",
		Message: fmt.Sprintf("Found %d goal(s)", len(filtered)),
		Data:    result,
	})

	// For human output, also print a table
	if !cli.JSONOutput {
		printGoalTable(filtered)
	}
}

func filterGoals(allGoals []goals.Goal, project, status string) []goals.Goal {
	if project == "" && status == "" {
		return allGoals
	}

	var filtered []goals.Goal
	for _, g := range allGoals {
		// Filter by status
		if status != "" && g.Status != status {
			continue
		}

		// Filter by project
		if project != "" {
			found := false
			for _, p := range g.Projects {
				if strings.EqualFold(p, project) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, g)
	}

	return filtered
}

func printGoalTable(goalList []goals.Goal) {
	if len(goalList) == 0 {
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "\nID\tTITLE\tPROJECT(S)\tSTATUS\tPHASE")
	fmt.Fprintln(w, "--\t-----\t----------\t------\t-----")

	for _, g := range goalList {
		projects := strings.Join(g.Projects, ", ")
		phase := g.Phase
		if phase == "" {
			phase = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			g.ID,
			truncate(g.Title, 40),
			truncate(projects, 20),
			g.Status,
			phase,
		)
	}
	w.Flush()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
