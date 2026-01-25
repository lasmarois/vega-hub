package project

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/spf13/cobra"
)

var (
	addBranch string
)

// AddResult contains the result of adding a project
type AddResult struct {
	Name        string `json:"name"`
	GitURL      string `json:"git_url"`
	Branch      string `json:"branch"`
	WorktreePath string `json:"worktree_path"`
	ConfigFile  string `json:"config_file"`
}

var addCmd = &cobra.Command{
	Use:   "add <name> <git-url>",
	Short: "Add a new project",
	Long: `Add a new project to vega-missile management.

Examples:
  vega-hub project add my-api https://github.com/user/my-api.git
  vega-hub project add my-api https://github.com/user/my-api.git --branch dev

This command will:
  1. Clone the repository into workspaces/<name>/worktree-base/
  2. Check out the specified branch (default: repo's default branch)
  3. Set up .claude/ structure with hooks
  4. Create projects/<name>.md config file
  5. Register the project in projects/index.md`,
	Args: cobra.ExactArgs(2),
	Run:  runAdd,
}

func init() {
	ProjectCmd.AddCommand(addCmd)
	addCmd.Flags().StringVarP(&addBranch, "branch", "b", "", "Branch to check out (default: repo's default)")
}

func runAdd(c *cobra.Command, args []string) {
	name := args[0]
	gitURL := args[1]

	// Validate project name (alphanumeric, dash, underscore)
	if !isValidProjectName(name) {
		cli.OutputError(cli.ExitValidationError, "invalid_name",
			"Invalid project name",
			map[string]string{
				"name":    name,
				"allowed": "alphanumeric, dash, underscore",
			},
			nil)
	}

	// Get vega-missile directory
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	// Check if project already exists
	projectBase := filepath.Join(vegaDir, "workspaces", name, "worktree-base")
	if _, err := os.Stat(projectBase); err == nil {
		cli.OutputError(cli.ExitConflict, "project_exists",
			fmt.Sprintf("Project '%s' already exists", name),
			map[string]string{
				"path": projectBase,
			},
			nil)
	}

	cli.Info("Adding project: %s", name)
	cli.Info("  Git URL: %s", gitURL)

	// Track created resources for rollback
	var rollback []func()
	doRollback := func() {
		for i := len(rollback) - 1; i >= 0; i-- {
			rollback[i]()
		}
	}

	// Step 1: Create workspaces directory and clone
	workspacesDir := filepath.Join(vegaDir, "workspaces", name)
	if err := os.MkdirAll(workspacesDir, 0755); err != nil {
		cli.OutputError(cli.ExitInternalError, "mkdir_failed",
			"Failed to create workspaces directory",
			map[string]string{"error": err.Error()},
			nil)
	}
	rollback = append(rollback, func() { os.RemoveAll(workspacesDir) })

	cli.Info("Cloning repository...")
	if err := gitClone(gitURL, projectBase); err != nil {
		doRollback()
		cli.OutputError(cli.ExitStateError, "clone_failed",
			"Failed to clone repository",
			map[string]string{
				"url":   gitURL,
				"error": err.Error(),
			},
			[]cli.ErrorOption{
				{Action: "verify", Description: "Check that the git URL is correct and accessible"},
			})
	}

	// Step 2: Checkout branch if specified
	branch := addBranch
	if branch != "" {
		cli.Info("Checking out branch: %s", branch)
		if err := gitCheckout(projectBase, branch); err != nil {
			doRollback()
			cli.OutputError(cli.ExitStateError, "checkout_failed",
				fmt.Sprintf("Failed to checkout branch '%s'", branch),
				map[string]string{"error": err.Error()},
				[]cli.ErrorOption{
					{Action: "list", Description: "Run: git -C <path> branch -a"},
				})
		}
	} else {
		// Get the current branch name
		branch, _ = getCurrentBranch(projectBase)
		if branch == "" {
			branch = "main" // fallback
		}
	}

	// Step 3: Set up .claude/ structure
	cli.Info("Setting up .claude/ structure...")
	templateDir := filepath.Join(vegaDir, "templates", "project-init", ".claude")
	destDir := filepath.Join(projectBase, ".claude")
	if err := copyDir(templateDir, destDir); err != nil {
		cli.Warn("Could not copy .claude template: %v", err)
	}

	// Create docs/planning directories
	planningDirs := []string{
		filepath.Join(projectBase, "docs", "planning", "history"),
		filepath.Join(projectBase, "docs", "planning", "iced"),
	}
	for _, dir := range planningDirs {
		os.MkdirAll(dir, 0755)
	}

	// Step 4: Create project config file
	cli.Info("Creating project config...")
	configFile := filepath.Join(vegaDir, "projects", name+".md")
	if err := createProjectConfig(configFile, name, gitURL, branch); err != nil {
		cli.Warn("Could not create project config: %v", err)
	}

	// Step 5: Update projects/index.md
	cli.Info("Updating project index...")
	indexFile := filepath.Join(vegaDir, "projects", "index.md")
	if err := addProjectToIndex(indexFile, name); err != nil {
		cli.Warn("Could not update project index: %v", err)
	}

	// Success output
	result := AddResult{
		Name:         name,
		GitURL:       gitURL,
		Branch:       branch,
		WorktreePath: projectBase,
		ConfigFile:   configFile,
	}

	cli.Output(cli.Result{
		Success: true,
		Action:  "project_add",
		Message: fmt.Sprintf("Added project: %s", name),
		Data:    result,
		NextSteps: []string{
			fmt.Sprintf("Edit project config: %s", configFile),
			fmt.Sprintf("Create a goal: vega-hub goal create \"<title>\" %s", name),
		},
	})

	// Human-readable summary
	if !cli.JSONOutput {
		fmt.Printf("\n  Workspace: %s\n", projectBase)
		fmt.Printf("  Branch: %s\n", branch)
		fmt.Printf("  Config: %s\n", configFile)
	}
}

// isValidProjectName checks if a project name is valid
func isValidProjectName(name string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	return re.MatchString(name)
}

// gitClone clones a repository
func gitClone(url, dest string) error {
	cmd := exec.Command("git", "clone", url, dest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

// gitCheckout checks out a branch
func gitCheckout(repoPath, branch string) error {
	cmd := exec.Command("git", "-C", repoPath, "checkout", branch)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

// getCurrentBranch gets the current branch name
func getCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// createProjectConfig creates a project config file from template
func createProjectConfig(path, name, gitURL, branch string) error {
	content := fmt.Sprintf(`# Project: %s

## Overview

_Brief description of the project_

## Configuration

- **Workspace**: %s
- **Base Branch**: %s
- **Upstream**: %s

## Active Goals

_None currently active_

## Completed Goals

| ID | Title | Completed |
|----|-------|-----------|
| | | |

## Project Context

### Key Directories
- %s - Source code

### Related Projects
- None

## Notes for Executors

When working on this project:
1. Load %s skill at session start
2. Planning files go at worktree root
3. Commit with %s reference
`, name,
		fmt.Sprintf("`workspaces/%s/worktree-base/`", name),
		fmt.Sprintf("`%s`", branch),
		fmt.Sprintf("`%s`", gitURL),
		"`src/`",
		"`planning-with-files`",
		"`Goal: <id>`")

	return os.WriteFile(path, []byte(content), 0644)
}

// addProjectToIndex adds a project to the index.md file
func addProjectToIndex(indexPath, name string) error {
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	added := false

	for i, line := range lines {
		newLines = append(newLines, line)

		// Find the table and add after the header separator
		if !added && strings.Contains(line, "|---") {
			// Check if this is in the projects table (has Project header)
			if i > 0 && strings.Contains(lines[i-1], "| Project |") {
				// Add new project row
				projectRow := fmt.Sprintf("| [%s](%s.md) | `workspaces/%s/worktree-base/` | - | _Add description_ |",
					name, name, name)
				newLines = append(newLines, projectRow)
				added = true
			}
		}
	}

	return os.WriteFile(indexPath, []byte(strings.Join(newLines, "\n")), 0644)
}

// copyDir recursively copies a directory (reused from goal/create.go)
func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file (reused from goal/create.go)
func copyFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, content, info.Mode())
}
