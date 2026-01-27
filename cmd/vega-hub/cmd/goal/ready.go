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
	readyProject string
)

// ReadyResult contains the result of listing ready goals
type ReadyResult struct {
	Goals []goals.Goal `json:"goals"`
	Total int          `json:"total"`
}

var readyCmd = &cobra.Command{
	Use:   "ready",
	Short: "List goals ready to work on",
	Long: `List all goals that are ready to work on (not blocked by dependencies).

A goal is "ready" if:
  - Status is active (not completed or iced)
  - State is pending or working
  - Has no open blocking dependencies

Filter by project:
  vega-hub goal ready --project my-api

Use --json for structured output.`,
	Run: runReady,
}

func init() {
	GoalCmd.AddCommand(readyCmd)
	readyCmd.Flags().StringVarP(&readyProject, "project", "p", "", "Filter by project name")
}

func runReady(c *cobra.Command, args []string) {
	dir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, nil)
	}

	dm := goals.NewDependencyManager(dir)

	readyGoals, err := dm.GetReadyGoals(readyProject)
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "get_ready_failed",
			fmt.Sprintf("Failed to get ready goals: %v", err), nil, nil)
	}

	result := ReadyResult{
		Goals: readyGoals,
		Total: len(readyGoals),
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "goal_ready",
		Message: fmt.Sprintf("Found %d ready goal(s)", len(readyGoals)),
		Data:    result,
	})

	if !cli.JSONOutput {
		printReadyGoals(readyGoals, dir)
	}
}

func printReadyGoals(goalList []goals.Goal, dir string) {
	if len(goalList) == 0 {
		fmt.Println("\nNo ready goals found.")
		fmt.Println("All active goals are either blocked by dependencies or in a non-working state.")
		return
	}

	dm := goals.NewDependencyManager(dir)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "\nID\tTITLE\tPROJECT(S)\tBLOCKS")
	fmt.Fprintln(w, "--\t-----\t----------\t------")

	for _, g := range goalList {
		projects := strings.Join(g.Projects, ", ")
		
		// Get dependents (goals this one blocks)
		info, _ := dm.GetDependencies(g.ID)
		blocksCount := 0
		if info != nil {
			for _, dep := range info.Dependents {
				if dep.Type == goals.DependencyBlocks {
					blocksCount++
				}
			}
		}
		
		blocksStr := "-"
		if blocksCount > 0 {
			blocksStr = fmt.Sprintf("%d goal(s)", blocksCount)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			g.ID,
			truncate(g.Title, 40),
			truncate(projects, 20),
			blocksStr,
		)
	}
	w.Flush()

	fmt.Printf("\nThese goals have no blocking dependencies and are ready to work on.\n")
}
