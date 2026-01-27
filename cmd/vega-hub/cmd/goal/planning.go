package goal

import (
	"fmt"
	"os"
	"sort"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/spf13/cobra"
)

var (
	planningFile    string
	planningContent string
)

// PlanningListResult is the result of listing planning files
type PlanningListResult struct {
	GoalID string              `json:"goal_id"`
	Files  map[string][]string `json:"files"`
	Count  int                 `json:"count"`
}

// PlanningGetResult is the result of getting a planning file
type PlanningGetResult struct {
	GoalID   string `json:"goal_id"`
	Project  string `json:"project"`
	Filename string `json:"filename"`
	Content  string `json:"content"`
}

// PlanningSetResult is the result of setting a planning file
type PlanningSetResult struct {
	GoalID   string `json:"goal_id"`
	Project  string `json:"project"`
	Filename string `json:"filename"`
	Success  bool   `json:"success"`
}

var planningCmd = &cobra.Command{
	Use:   "planning",
	Short: "Manage planning files for a goal",
	Long: `Manage planning files (task_plan.md, findings.md, progress.md) for a goal.

Planning files are stored per-project within a goal and are used by the
meta-executor to coordinate work across multiple project executors.

Examples:
  vega-hub goal planning list abc123
  vega-hub goal planning get abc123 my-api task_plan.md
  vega-hub goal planning set abc123 my-api task_plan.md --file plan.md
  vega-hub goal planning set abc123 my-api findings.md --content "# Findings"`,
}

var planningListCmd = &cobra.Command{
	Use:   "list <goal-id>",
	Short: "List all planning files for a goal",
	Long: `List all planning files for a goal, organized by project.

Example:
  vega-hub goal planning list abc123`,
	Args: cobra.ExactArgs(1),
	Run:  runPlanningList,
}

var planningGetCmd = &cobra.Command{
	Use:   "get <goal-id> <project> <filename>",
	Short: "Get a planning file's content",
	Long: `Get the content of a planning file.

Example:
  vega-hub goal planning get abc123 my-api task_plan.md`,
	Args: cobra.ExactArgs(3),
	Run:  runPlanningGet,
}

var planningSetCmd = &cobra.Command{
	Use:   "set <goal-id> <project> <filename>",
	Short: "Set a planning file's content",
	Long: `Set the content of a planning file.

Either --file or --content must be specified.

Examples:
  vega-hub goal planning set abc123 my-api task_plan.md --file plan.md
  vega-hub goal planning set abc123 my-api findings.md --content "# Findings"`,
	Args: cobra.ExactArgs(3),
	Run:  runPlanningSet,
}

func init() {
	GoalCmd.AddCommand(planningCmd)
	planningCmd.AddCommand(planningListCmd)
	planningCmd.AddCommand(planningGetCmd)
	planningCmd.AddCommand(planningSetCmd)

	planningSetCmd.Flags().StringVar(&planningFile, "file", "", "Path to file containing content")
	planningSetCmd.Flags().StringVar(&planningContent, "content", "", "Content to set directly")
}

func runPlanningList(c *cobra.Command, args []string) {
	goalID := args[0]

	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-hub directory explicitly"},
		})
	}

	mgr := goals.NewPlanningFilesManager(vegaDir)
	files, err := mgr.ListPlanningFiles(goalID)
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "list_failed",
			fmt.Sprintf("Failed to list planning files for goal '%s'", goalID),
			map[string]string{"goal_id": goalID, "error": err.Error()},
			nil)
	}

	// Count total files
	totalCount := 0
	for _, projectFiles := range files {
		totalCount += len(projectFiles)
	}

	result := PlanningListResult{
		GoalID: goalID,
		Files:  files,
		Count:  totalCount,
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "planning_list",
		Message: fmt.Sprintf("Found %d planning files across %d projects", totalCount, len(files)),
		Data:    result,
	})

	// Human-readable output
	if !cli.JSONOutput {
		if len(files) == 0 {
			fmt.Println("\n  No planning files found.")
			return
		}

		// Sort project names for consistent output
		projects := make([]string, 0, len(files))
		for project := range files {
			projects = append(projects, project)
		}
		sort.Strings(projects)

		fmt.Println("\n  Planning Files:")
		for _, project := range projects {
			fmt.Printf("\n  📁 %s/\n", project)
			for _, filename := range files[project] {
				fmt.Printf("     └─ %s\n", filename)
			}
		}
	}
}

func runPlanningGet(c *cobra.Command, args []string) {
	goalID := args[0]
	project := args[1]
	filename := args[2]

	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-hub directory explicitly"},
		})
	}

	mgr := goals.NewPlanningFilesManager(vegaDir)
	content, err := mgr.GetPlanningFile(goalID, project, filename)
	if err != nil {
		cli.OutputError(cli.ExitNotFound, "file_not_found",
			fmt.Sprintf("Planning file not found: %s/%s", project, filename),
			map[string]string{"goal_id": goalID, "project": project, "filename": filename, "error": err.Error()},
			nil)
	}

	result := PlanningGetResult{
		GoalID:   goalID,
		Project:  project,
		Filename: filename,
		Content:  content,
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "planning_get",
		Message: fmt.Sprintf("Retrieved %s/%s (%d bytes)", project, filename, len(content)),
		Data:    result,
	})

	// Human-readable output: print content directly
	if !cli.JSONOutput {
		fmt.Println() // Empty line before content
		fmt.Println(content)
	}
}

func runPlanningSet(c *cobra.Command, args []string) {
	goalID := args[0]
	project := args[1]
	filename := args[2]

	// Validate that exactly one of --file or --content is specified
	if planningFile == "" && planningContent == "" {
		cli.OutputError(cli.ExitValidationError, "no_content",
			"Either --file or --content must be specified",
			nil,
			[]cli.ErrorOption{
				{Flag: "file", Description: "Path to file containing content"},
				{Flag: "content", Description: "Content to set directly"},
			})
	}
	if planningFile != "" && planningContent != "" {
		cli.OutputError(cli.ExitValidationError, "ambiguous_content",
			"Cannot specify both --file and --content",
			nil,
			nil)
	}

	// Get content
	var content string
	if planningFile != "" {
		data, err := os.ReadFile(planningFile)
		if err != nil {
			cli.OutputError(cli.ExitValidationError, "file_read_failed",
				fmt.Sprintf("Failed to read file: %s", planningFile),
				map[string]string{"path": planningFile, "error": err.Error()},
				nil)
		}
		content = string(data)
	} else {
		content = planningContent
	}

	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-hub directory explicitly"},
		})
	}

	mgr := goals.NewPlanningFilesManager(vegaDir)
	if err := mgr.SavePlanningFile(goalID, project, filename, content); err != nil {
		cli.OutputError(cli.ExitInternalError, "save_failed",
			fmt.Sprintf("Failed to save planning file: %s/%s", project, filename),
			map[string]string{"goal_id": goalID, "project": project, "filename": filename, "error": err.Error()},
			nil)
	}

	result := PlanningSetResult{
		GoalID:   goalID,
		Project:  project,
		Filename: filename,
		Success:  true,
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "planning_set",
		Message: fmt.Sprintf("Saved %s/%s (%d bytes)", project, filename, len(content)),
		Data:    result,
	})

	// Human-readable output
	if !cli.JSONOutput {
		fmt.Printf("\n  ✓ Saved %s/%s (%d bytes)\n", project, filename, len(content))
	}
}
