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
	listTree    bool
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

Display as tree (shows parent-child relationships):
  vega-hub goal list --tree
  vega-hub goal list --tree --project my-api

Use --json for structured output.`,
	Run: runList,
}

func init() {
	GoalCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&listProject, "project", "p", "", "Filter by project name")
	listCmd.Flags().StringVarP(&listStatus, "status", "s", "", "Filter by status (active, iced, completed)")
	listCmd.Flags().BoolVarP(&listTree, "tree", "t", false, "Display goals as tree (shows hierarchy)")
}

func runList(c *cobra.Command, args []string) {
	dir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	// Handle tree display mode
	if listTree {
		runListTree(dir)
		return
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

// TreeResult contains the result of listing goals as a tree
type TreeResult struct {
	Tree  []*goals.GoalTreeNode `json:"tree"`
	Total int                   `json:"total"`
}

func runListTree(dir string) {
	hm := goals.NewHierarchyManager(dir)

	// Get tree
	tree, err := hm.BuildTree()
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "tree_failed",
			fmt.Sprintf("Failed to build goal tree: %v", err),
			nil, nil)
	}

	// Count total nodes
	total := countTreeNodes(tree)

	// For JSON output, return the tree structure
	if cli.JSONOutput {
		result := TreeResult{
			Tree:  tree,
			Total: total,
		}
		cli.Output(cli.Result{
			Success: true,
			Action:  "goal_list_tree",
			Message: fmt.Sprintf("Found %d goal(s)", total),
			Data:    result,
		})
		return
	}

	// For human output, render ASCII tree
	treeOutput, err := hm.RenderTree(listProject, listStatus)
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "render_failed",
			fmt.Sprintf("Failed to render goal tree: %v", err),
			nil, nil)
	}

	if treeOutput == "" {
		fmt.Println("\nNo goals found matching the criteria.")
		return
	}

	fmt.Printf("\nGoal Hierarchy (%d goals):\n\n", total)
	fmt.Println(treeOutput)

	// Legend
	fmt.Println("Legend: ◉ Active  ⊘ Blocked  ❄ Iced  ✓ Completed")
}

func countTreeNodes(nodes []*goals.GoalTreeNode) int {
	count := 0
	for _, node := range nodes {
		count++
		count += countTreeNodes(node.Children)
	}
	return count
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
