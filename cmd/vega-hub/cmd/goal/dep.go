package goal

import (
	"fmt"
	"strings"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/spf13/cobra"
)

var (
	depType string
)

// DepResult contains the result of dependency operations
type DepResult struct {
	Action    string   `json:"action"`
	GoalID    string   `json:"goal_id"`
	DependsOn string   `json:"depends_on,omitempty"`
	Type      string   `json:"type,omitempty"`
	Removed   string   `json:"removed,omitempty"`
}

// DepTreeResult contains dependency tree output
type DepTreeResult struct {
	GoalID string             `json:"goal_id"`
	Tree   *goals.DependencyNode `json:"tree"`
}

// DepInfoResult contains dependency information
type DepInfoResult struct {
	GoalID       string             `json:"goal_id"`
	Dependencies []goals.Dependency `json:"dependencies"`
	Dependents   []goals.Dependency `json:"dependents"`
}

var depCmd = &cobra.Command{
	Use:   "dep",
	Short: "Manage goal dependencies",
	Long: `Manage dependencies between goals.

Dependencies allow you to specify that one goal blocks another,
preventing work on blocked goals until their dependencies are complete.

Examples:
  vega-hub goal dep add abc123 def456           # abc123 is blocked by def456
  vega-hub goal dep add abc123 def456 --type related  # Related but not blocking
  vega-hub goal dep remove abc123 def456        # Remove dependency
  vega-hub goal dep tree abc123                 # Show dependency tree
  vega-hub goal dep show abc123                 # Show dependencies and dependents`,
}

var depAddCmd = &cobra.Command{
	Use:   "add <goal-id> <depends-on-id>",
	Short: "Add a dependency between goals",
	Long: `Add a dependency: <goal-id> depends on (is blocked by) <depends-on-id>.

The goal cannot be considered "ready" until all blocking dependencies are completed.

Types:
  blocks  - The dependency must be completed first (default)
  related - Related goals, no blocking relationship

Examples:
  vega-hub goal dep add abc123 def456                  # abc123 blocked by def456
  vega-hub goal dep add abc123 def456 --type blocks   # Same as above (explicit)
  vega-hub goal dep add abc123 def456 --type related  # Related but not blocking`,
	Args: cobra.ExactArgs(2),
	Run:  runDepAdd,
}

var depRemoveCmd = &cobra.Command{
	Use:   "remove <goal-id> <depends-on-id>",
	Short: "Remove a dependency between goals",
	Long: `Remove an existing dependency between two goals.

Examples:
  vega-hub goal dep remove abc123 def456`,
	Args: cobra.ExactArgs(2),
	Run:  runDepRemove,
}

var depTreeCmd = &cobra.Command{
	Use:   "tree <goal-id>",
	Short: "Show dependency tree for a goal",
	Long: `Display an ASCII tree showing all dependencies of a goal.

Examples:
  vega-hub goal dep tree abc123`,
	Args: cobra.ExactArgs(1),
	Run:  runDepTree,
}

var depShowCmd = &cobra.Command{
	Use:   "show <goal-id>",
	Short: "Show dependencies and dependents for a goal",
	Long: `Display all dependencies (goals this blocks) and dependents (goals blocked by this).

Examples:
  vega-hub goal dep show abc123`,
	Args: cobra.ExactArgs(1),
	Run:  runDepShow,
}

func init() {
	GoalCmd.AddCommand(depCmd)
	depCmd.AddCommand(depAddCmd)
	depCmd.AddCommand(depRemoveCmd)
	depCmd.AddCommand(depTreeCmd)
	depCmd.AddCommand(depShowCmd)

	depAddCmd.Flags().StringVarP(&depType, "type", "t", "blocks", "Dependency type: blocks (default) or related")
}

func runDepAdd(c *cobra.Command, args []string) {
	goalID := args[0]
	dependsOnID := args[1]

	dir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, nil)
	}

	dm := goals.NewDependencyManager(dir)

	// Validate and normalize type
	dt := goals.DependencyType(strings.ToLower(depType))
	if dt != goals.DependencyBlocks && dt != goals.DependencyRelated {
		cli.OutputError(cli.ExitValidationError, "invalid_type",
			fmt.Sprintf("Invalid dependency type: %s (must be 'blocks' or 'related')", depType),
			nil, nil)
	}

	if err := dm.AddDependency(goalID, dependsOnID, dt); err != nil {
		if strings.Contains(err.Error(), "circular") {
			cli.OutputError(cli.ExitValidationError, "circular_dependency", err.Error(),
				map[string]string{"goal": goalID, "depends_on": dependsOnID}, nil)
		} else if strings.Contains(err.Error(), "not found") {
			cli.OutputError(cli.ExitNotFound, "goal_not_found", err.Error(), nil, nil)
		} else if strings.Contains(err.Error(), "itself") {
			cli.OutputError(cli.ExitValidationError, "self_dependency", err.Error(), nil, nil)
		} else {
			cli.OutputError(cli.ExitInternalError, "add_failed", err.Error(), nil, nil)
		}
	}

	result := DepResult{
		Action:    "add",
		GoalID:    goalID,
		DependsOn: dependsOnID,
		Type:      string(dt),
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "dep_add",
		Message: fmt.Sprintf("Added dependency: %s depends on %s (%s)", goalID, dependsOnID, dt),
		Data:    result,
	})

	if !cli.JSONOutput {
		fmt.Printf("\n  %s → %s (%s)\n", goalID, dependsOnID, dt)
		if dt == goals.DependencyBlocks {
			fmt.Printf("  %s is now blocked until %s is completed\n", goalID, dependsOnID)
		}
	}
}

func runDepRemove(c *cobra.Command, args []string) {
	goalID := args[0]
	dependsOnID := args[1]

	dir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, nil)
	}

	dm := goals.NewDependencyManager(dir)

	if err := dm.RemoveDependency(goalID, dependsOnID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			cli.OutputError(cli.ExitNotFound, "dependency_not_found", err.Error(),
				map[string]string{"goal": goalID, "depends_on": dependsOnID}, nil)
		} else {
			cli.OutputError(cli.ExitInternalError, "remove_failed", err.Error(), nil, nil)
		}
	}

	result := DepResult{
		Action:  "remove",
		GoalID:  goalID,
		Removed: dependsOnID,
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "dep_remove",
		Message: fmt.Sprintf("Removed dependency: %s no longer depends on %s", goalID, dependsOnID),
		Data:    result,
	})
}

func runDepTree(c *cobra.Command, args []string) {
	goalID := args[0]

	dir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, nil)
	}

	dm := goals.NewDependencyManager(dir)

	tree, err := dm.GetDependencyTree(goalID)
	if err != nil {
		cli.OutputError(cli.ExitNotFound, "goal_not_found", err.Error(), nil, nil)
	}

	result := DepTreeResult{
		GoalID: goalID,
		Tree:   tree,
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "dep_tree",
		Message: fmt.Sprintf("Dependency tree for %s", goalID),
		Data:    result,
	})

	if !cli.JSONOutput {
		fmt.Println()
		printTree(tree, "", true)
	}
}

func printTree(node *goals.DependencyNode, prefix string, isLast bool) {
	// Choose connector
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	if prefix == "" {
		connector = ""
	}

	// Status indicator
	statusIcon := "○"
	switch node.Status {
	case "active":
		statusIcon = "◉"
	case "completed":
		statusIcon = "✓"
	case "iced":
		statusIcon = "❄"
	case "error":
		statusIcon = "✗"
	}

	// Type indicator (for non-root nodes)
	typeIndicator := ""
	if node.Type == goals.DependencyBlocks {
		typeIndicator = " [blocks]"
	} else if node.Type == goals.DependencyRelated {
		typeIndicator = " [related]"
	}

	// Print this node
	title := node.Title
	if len(title) > 40 {
		title = title[:37] + "..."
	}
	fmt.Printf("%s%s%s %s: %s%s\n", prefix, connector, statusIcon, node.GoalID, title, typeIndicator)

	// Prepare prefix for children
	childPrefix := prefix
	if prefix != "" {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	// Print children
	for i, child := range node.Children {
		isLastChild := i == len(node.Children)-1
		printTree(child, childPrefix, isLastChild)
	}
}

func runDepShow(c *cobra.Command, args []string) {
	goalID := args[0]

	dir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, nil)
	}

	dm := goals.NewDependencyManager(dir)

	info, err := dm.GetDependencies(goalID)
	if err != nil {
		cli.OutputError(cli.ExitNotFound, "goal_not_found", err.Error(), nil, nil)
	}

	result := DepInfoResult{
		GoalID:       goalID,
		Dependencies: info.Dependencies,
		Dependents:   info.Dependents,
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "dep_show",
		Message: fmt.Sprintf("Dependencies for %s", goalID),
		Data:    result,
	})

	if !cli.JSONOutput {
		fmt.Printf("\nGoal: %s\n", goalID)
		
		fmt.Printf("\nDepends on (blocked by):\n")
		if len(info.Dependencies) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, dep := range info.Dependencies {
				fmt.Printf("  → %s [%s]\n", dep.GoalID, dep.Type)
			}
		}

		fmt.Printf("\nDependents (blocks):\n")
		if len(info.Dependents) == 0 {
			fmt.Println("  (none)")
		} else {
			for _, dep := range info.Dependents {
				fmt.Printf("  ← %s [%s]\n", dep.GoalID, dep.Type)
			}
		}

		// Show blocked status
		isBlocked := false
		for _, dep := range info.Dependencies {
			if dep.Type == goals.DependencyBlocks {
				isBlocked = true
				break
			}
		}
		if isBlocked {
			fmt.Printf("\nStatus: BLOCKED (waiting on dependencies)\n")
		} else {
			fmt.Printf("\nStatus: READY (no blocking dependencies)\n")
		}
	}
}
