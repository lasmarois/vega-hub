// Package operations provides shared goal/project operations for CLI and API.
package operations

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lasmarois/vega-hub/internal/goals"
)

// Result is the standard result format for operations
type Result struct {
	Success bool        `json:"success"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorInfo contains structured error information
type ErrorInfo struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// CompleteOptions contains options for completing a goal
type CompleteOptions struct {
	GoalID   string
	Project  string
	NoMerge  bool
	Force    bool
	VegaDir  string
}

// CompleteResult contains the result of completing a goal
type CompleteResult struct {
	GoalID          string `json:"goal_id"`
	Title           string `json:"title"`
	Project         string `json:"project"`
	Merged          bool   `json:"merged"`
	MergedTo        string `json:"merged_to,omitempty"`
	MergedFrom      string `json:"merged_from,omitempty"`
	WorktreeRemoved bool   `json:"worktree_removed"`
	BranchDeleted   bool   `json:"branch_deleted"`
	GoalArchived    bool   `json:"goal_archived"`
	HistoryFile     string `json:"history_file"`
}

// IceOptions contains options for icing a goal
type IceOptions struct {
	GoalID          string
	Project         string
	Reason          string
	RemoveWorktree  bool // If true, remove worktree (default: keep it)
	Force           bool // If true, ignore uncommitted changes when removing worktree
	VegaDir         string
}

// IceResult contains the result of icing a goal
type IceResult struct {
	GoalID            string `json:"goal_id"`
	Title             string `json:"title"`
	Project           string `json:"project"`
	Reason            string `json:"reason"`
	BranchPreserved   string `json:"branch_preserved"`
	WorktreeRemoved   bool   `json:"worktree_removed"`
	WorktreePreserved string `json:"worktree_preserved,omitempty"`
}

// ResumeOptions contains options for resuming an iced goal
type ResumeOptions struct {
	GoalID  string
	Project string
	VegaDir string
}

// ResumeResult contains the result of resuming an iced goal
type ResumeResult struct {
	GoalID          string `json:"goal_id"`
	Title           string `json:"title"`
	Project         string `json:"project"`
	WorktreeCreated bool   `json:"worktree_created"`
	WorktreePath    string `json:"worktree_path,omitempty"`
	WorktreeExisted bool   `json:"worktree_existed"`
}

// CleanupOptions contains options for cleaning up a goal's branch
type CleanupOptions struct {
	GoalID  string
	Project string
	VegaDir string
}

// CleanupResult contains the result of cleaning up a goal
type CleanupResult struct {
	GoalID        string `json:"goal_id"`
	Project       string `json:"project"`
	Branch        string `json:"branch"`
	BranchDeleted bool   `json:"branch_deleted"`
	BranchExisted bool   `json:"branch_existed"`
}

// CreateOptions contains options for creating a goal
type CreateOptions struct {
	Title      string
	Project    string
	BaseBranch string // Optional override
	NoWorktree bool
	VegaDir    string
}

// CreateResult contains the result of creating a goal
type CreateResult struct {
	GoalID       string `json:"goal_id"`
	Title        string `json:"title"`
	Project      string `json:"project"`
	BaseBranch   string `json:"base_branch"`
	GoalBranch   string `json:"goal_branch"`
	WorktreePath string `json:"worktree_path"`
	GoalFile     string `json:"goal_file"`
}

// CompleteGoal completes a goal (merge, cleanup, archive)
func CompleteGoal(opts CompleteOptions) (*Result, *CompleteResult) {
	// Validate goal exists and is active
	goalFile := filepath.Join(opts.VegaDir, "goals", "active", opts.GoalID+".md")
	if _, err := os.Stat(goalFile); os.IsNotExist(err) {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "goal_not_found",
				Message: fmt.Sprintf("Active goal '%s' not found", opts.GoalID),
				Details: map[string]string{"goal_id": opts.GoalID},
			},
		}, nil
	}

	// Get goal title
	goalTitle := getGoalTitle(goalFile, opts.GoalID)

	// Get project base folder
	projectBase := filepath.Join(opts.VegaDir, "workspaces", opts.Project, "worktree-base")
	if _, err := os.Stat(projectBase); os.IsNotExist(err) {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "project_not_found",
				Message: fmt.Sprintf("Project '%s' not found", opts.Project),
				Details: map[string]string{"project": opts.Project},
			},
		}, nil
	}

	// Get base branch
	baseBranch, err := getProjectBaseBranch(opts.VegaDir, opts.Project)
	if err != nil {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "no_base_branch",
				Message: fmt.Sprintf("Could not determine base branch for project '%s'", opts.Project),
				Details: map[string]string{"error": err.Error()},
			},
		}, nil
	}

	// Find worktree
	worktreeDir, err := findWorktreeDir(opts.VegaDir, opts.Project, opts.GoalID)
	if err != nil {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "worktree_not_found",
				Message: err.Error(),
				Details: map[string]string{"goal_id": opts.GoalID},
			},
		}, nil
	}

	// Get branch name
	branchName, err := getWorktreeBranch(worktreeDir)
	if err != nil {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "branch_detection_failed",
				Message: "Could not determine branch name from worktree",
				Details: map[string]string{"error": err.Error()},
			},
		}, nil
	}

	// Safety check
	if !opts.Force {
		if err := checkWorktreeClean(worktreeDir); err != nil {
			return &Result{
				Success: false,
				Error: &ErrorInfo{
					Code:    "uncommitted_changes",
					Message: "Worktree has uncommitted changes",
					Details: map[string]string{"worktree": worktreeDir, "details": err.Error()},
				},
			}, nil
		}
	}

	result := &CompleteResult{
		GoalID:  opts.GoalID,
		Title:   goalTitle,
		Project: opts.Project,
	}

	// Step 1: Merge branch (unless --no-merge)
	if !opts.NoMerge {
		if err := mergeBranch(projectBase, branchName, baseBranch, opts.GoalID, goalTitle); err != nil {
			return &Result{
				Success: false,
				Error: &ErrorInfo{
					Code:    "merge_failed",
					Message: "Merge failed",
					Details: map[string]string{"source": branchName, "target": baseBranch, "error": err.Error()},
				},
			}, nil
		}
		result.Merged = true
		result.MergedTo = baseBranch
		result.MergedFrom = branchName
	}

	// Step 2: Remove worktree
	removeWorktree(projectBase, worktreeDir)
	result.WorktreeRemoved = true

	// Step 3: Delete branch (unless --no-merge)
	if !opts.NoMerge {
		if err := deleteBranch(projectBase, branchName); err == nil {
			result.BranchDeleted = true
		}
	}

	// Step 4: Move goal file to history
	historyDir := filepath.Join(opts.VegaDir, "goals", "history")
	os.MkdirAll(historyDir, 0755)
	historyFile := filepath.Join(historyDir, opts.GoalID+".md")
	if err := os.Rename(goalFile, historyFile); err == nil {
		result.GoalArchived = true
		result.HistoryFile = historyFile
	}

	// Step 5: Update registry
	registryPath := filepath.Join(opts.VegaDir, "goals", "REGISTRY.md")
	completeGoalInRegistry(registryPath, opts.GoalID, goalTitle, opts.Project)

	// Step 6: Update project config
	projectConfig := filepath.Join(opts.VegaDir, "projects", opts.Project+".md")
	completeGoalInProjectConfig(projectConfig, opts.GoalID, goalTitle)

	return &Result{Success: true}, result
}

// IceGoal pauses a goal for later
func IceGoal(opts IceOptions) (*Result, *IceResult) {
	// Validate goal exists and is active
	goalFile := filepath.Join(opts.VegaDir, "goals", "active", opts.GoalID+".md")
	if _, err := os.Stat(goalFile); os.IsNotExist(err) {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "goal_not_found",
				Message: fmt.Sprintf("Active goal '%s' not found", opts.GoalID),
				Details: map[string]string{"goal_id": opts.GoalID},
			},
		}, nil
	}

	// Get goal title
	goalTitle := getGoalTitle(goalFile, opts.GoalID)

	// Get project base folder
	projectBase := filepath.Join(opts.VegaDir, "workspaces", opts.Project, "worktree-base")
	if _, err := os.Stat(projectBase); os.IsNotExist(err) {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "project_not_found",
				Message: fmt.Sprintf("Project '%s' not found", opts.Project),
				Details: map[string]string{"project": opts.Project},
			},
		}, nil
	}

	// Find worktree
	worktreeDir, err := findWorktreeDir(opts.VegaDir, opts.Project, opts.GoalID)
	if err != nil {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "worktree_not_found",
				Message: err.Error(),
				Details: map[string]string{"goal_id": opts.GoalID},
			},
		}, nil
	}

	// Get branch name
	branchName, _ := getWorktreeBranch(worktreeDir)

	// Safety check - only needed if removing worktree
	if opts.RemoveWorktree && !opts.Force {
		if err := checkWorktreeClean(worktreeDir); err != nil {
			return &Result{
				Success: false,
				Error: &ErrorInfo{
					Code:    "uncommitted_changes",
					Message: "Worktree has uncommitted changes",
					Details: map[string]string{"worktree": worktreeDir, "details": err.Error()},
				},
			}, nil
		}
	}

	result := &IceResult{
		GoalID:  opts.GoalID,
		Title:   goalTitle,
		Project: opts.Project,
		Reason:  opts.Reason,
	}

	// Step 1: Optionally remove worktree (branch always preserved)
	if opts.RemoveWorktree {
		removeWorktree(projectBase, worktreeDir)
		result.WorktreeRemoved = true
	} else {
		result.WorktreePreserved = worktreeDir
	}
	result.BranchPreserved = branchName

	// Step 2: Move goal file to iced
	icedDir := filepath.Join(opts.VegaDir, "goals", "iced")
	os.MkdirAll(icedDir, 0755)
	icedFile := filepath.Join(icedDir, opts.GoalID+".md")
	os.Rename(goalFile, icedFile)

	// Step 3: Update goal file with iced status
	updateGoalStatus(icedFile, "iced", opts.Reason)

	// Step 4: Update registry
	registryPath := filepath.Join(opts.VegaDir, "goals", "REGISTRY.md")
	iceGoalInRegistry(registryPath, opts.GoalID, opts.Reason)

	return &Result{Success: true}, result
}

// ResumeGoal resumes an iced goal
func ResumeGoal(opts ResumeOptions) (*Result, *ResumeResult) {
	// Validate goal exists and is iced
	icedFile := filepath.Join(opts.VegaDir, "goals", "iced", opts.GoalID+".md")
	if _, err := os.Stat(icedFile); os.IsNotExist(err) {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "goal_not_found",
				Message: fmt.Sprintf("Iced goal '%s' not found", opts.GoalID),
				Details: map[string]string{"goal_id": opts.GoalID},
			},
		}, nil
	}

	// Get goal title
	goalTitle := getGoalTitle(icedFile, opts.GoalID)

	// Get project info
	project, err := goals.ParseProject(opts.VegaDir, opts.Project)
	if err != nil {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "project_not_found",
				Message: fmt.Sprintf("Project '%s' not found", opts.Project),
				Details: map[string]string{"error": err.Error()},
			},
		}, nil
	}

	projectBase := filepath.Join(opts.VegaDir, "workspaces", opts.Project, "worktree-base")

	result := &ResumeResult{
		GoalID:  opts.GoalID,
		Title:   goalTitle,
		Project: opts.Project,
	}

	// Step 1: Move goal file back to active
	activeDir := filepath.Join(opts.VegaDir, "goals", "active")
	os.MkdirAll(activeDir, 0755)
	activeFile := filepath.Join(activeDir, opts.GoalID+".md")
	if err := os.Rename(icedFile, activeFile); err != nil {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "file_move_failed",
				Message: "Could not move goal file to active",
				Details: map[string]string{"error": err.Error()},
			},
		}, nil
	}

	// Step 2: Update registry
	registryPath := filepath.Join(opts.VegaDir, "goals", "REGISTRY.md")
	resumeGoalInRegistry(registryPath, opts.GoalID, goalTitle, opts.Project)

	// Step 3: Check if worktree exists, recreate if needed
	worktreeDir, err := findWorktreeDir(opts.VegaDir, opts.Project, opts.GoalID)
	if err != nil {
		// Worktree doesn't exist, need to recreate it
		// Find the branch
		branchName, err := findGoalBranch(projectBase, opts.GoalID)
		if err != nil {
			return &Result{
				Success: false,
				Error: &ErrorInfo{
					Code:    "branch_not_found",
					Message: fmt.Sprintf("No branch found for goal '%s'", opts.GoalID),
					Details: map[string]string{"error": err.Error()},
				},
			}, nil
		}

		// Recreate worktree from existing branch
		baseBranch := project.BaseBranch
		if baseBranch == "" {
			baseBranch = "main"
		}
		worktreePath := filepath.Join(opts.VegaDir, "workspaces", opts.Project, branchName)

		if err := recreateWorktree(projectBase, worktreePath, branchName); err != nil {
			return &Result{
				Success: false,
				Error: &ErrorInfo{
					Code:    "worktree_create_failed",
					Message: "Could not recreate worktree",
					Details: map[string]string{"error": err.Error()},
				},
			}, nil
		}

		result.WorktreeCreated = true
		result.WorktreePath = worktreePath

		// Copy hooks to worktree
		copyHooksToWorktree(opts.VegaDir, worktreePath)
	} else {
		// Worktree already exists
		result.WorktreeExisted = true
		result.WorktreePath = worktreeDir
	}

	return &Result{Success: true}, result
}

// CleanupGoal deletes a completed goal's branch
func CleanupGoal(opts CleanupOptions) (*Result, *CleanupResult) {
	// Check goal is in history
	historyFile := filepath.Join(opts.VegaDir, "goals", "history", opts.GoalID+".md")
	activeFile := filepath.Join(opts.VegaDir, "goals", "active", opts.GoalID+".md")
	icedFile := filepath.Join(opts.VegaDir, "goals", "iced", opts.GoalID+".md")

	if _, err := os.Stat(activeFile); err == nil {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "goal_still_active",
				Message: fmt.Sprintf("Goal '%s' is still active", opts.GoalID),
				Details: map[string]string{"goal_id": opts.GoalID, "status": "active"},
			},
		}, nil
	}

	if _, err := os.Stat(icedFile); err == nil {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "goal_is_iced",
				Message: fmt.Sprintf("Goal '%s' is iced (paused), not completed", opts.GoalID),
				Details: map[string]string{"goal_id": opts.GoalID, "status": "iced"},
			},
		}, nil
	}

	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "goal_not_found",
				Message: fmt.Sprintf("Goal '%s' not found in history", opts.GoalID),
				Details: map[string]string{"goal_id": opts.GoalID},
			},
		}, nil
	}

	// Get project base folder
	projectBase := filepath.Join(opts.VegaDir, "workspaces", opts.Project, "worktree-base")
	if _, err := os.Stat(projectBase); os.IsNotExist(err) {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "project_not_found",
				Message: fmt.Sprintf("Project '%s' not found", opts.Project),
				Details: map[string]string{"project": opts.Project},
			},
		}, nil
	}

	// Find branch
	branchName, err := findGoalBranch(projectBase, opts.GoalID)
	if err != nil {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "branch_not_found",
				Message: fmt.Sprintf("No branch found for goal '%s'", opts.GoalID),
				Details: map[string]string{"goal_id": opts.GoalID, "error": err.Error()},
			},
		}, nil
	}

	result := &CleanupResult{
		GoalID:        opts.GoalID,
		Project:       opts.Project,
		Branch:        branchName,
		BranchExisted: true,
	}

	// Delete branch
	if err := deleteBranch(projectBase, branchName); err != nil {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "branch_delete_failed",
				Message: "Could not delete branch",
				Details: map[string]string{"branch": branchName, "error": err.Error()},
			},
		}, nil
	}
	result.BranchDeleted = true

	return &Result{Success: true}, result
}

// CreateGoal creates a new goal with worktree
func CreateGoal(opts CreateOptions) (*Result, *CreateResult) {
	// Parse project config
	project, err := goals.ParseProject(opts.VegaDir, opts.Project)
	if err != nil {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "project_not_found",
				Message: fmt.Sprintf("Project '%s' not found", opts.Project),
				Details: map[string]string{"error": err.Error()},
			},
		}, nil
	}

	// Determine base branch
	baseBranch := opts.BaseBranch
	if baseBranch == "" {
		baseBranch = project.BaseBranch
	}
	if baseBranch == "" {
		baseBranch = "main" // fallback
	}

	// Generate goal ID
	goalID := generateGoalID()
	slug := slugify(opts.Title)
	branchName := fmt.Sprintf("goal-%s-%s", goalID, slug)

	// Create goal file
	goalFile := filepath.Join(opts.VegaDir, "goals", "active", goalID+".md")
	if err := createGoalFile(goalFile, goalID, opts.Title, opts.Project); err != nil {
		return &Result{
			Success: false,
			Error: &ErrorInfo{
				Code:    "file_create_failed",
				Message: "Could not create goal file",
				Details: map[string]string{"error": err.Error()},
			},
		}, nil
	}

	// Update registry
	registryPath := filepath.Join(opts.VegaDir, "goals", "REGISTRY.md")
	addGoalToRegistry(registryPath, goalID, opts.Title, opts.Project)

	result := &CreateResult{
		GoalID:     goalID,
		Title:      opts.Title,
		Project:    opts.Project,
		BaseBranch: baseBranch,
		GoalBranch: branchName,
		GoalFile:   goalFile,
	}

	// Create worktree (unless --no-worktree)
	if !opts.NoWorktree {
		projectBase := filepath.Join(opts.VegaDir, "workspaces", opts.Project, "worktree-base")
		worktreePath := filepath.Join(opts.VegaDir, "workspaces", opts.Project, branchName)

		if err := createWorktree(projectBase, worktreePath, branchName, baseBranch); err != nil {
			return &Result{
				Success: false,
				Error: &ErrorInfo{
					Code:    "worktree_create_failed",
					Message: "Could not create worktree",
					Details: map[string]string{"error": err.Error()},
				},
			}, nil
		}
		result.WorktreePath = worktreePath

		// Copy hooks to worktree
		copyHooksToWorktree(opts.VegaDir, worktreePath)

		// Write worktree metadata to goal file
		worktreeSection := fmt.Sprintf("\n## Worktree\n- **Branch**: %s\n- **Project**: %s\n- **Path**: workspaces/%s/%s\n- **Base Branch**: %s\n- **Created**: %s\n",
			branchName, opts.Project, opts.Project, branchName, baseBranch, time.Now().Format("2006-01-02"))

		// Read existing content and insert before Status section
		if content, err := os.ReadFile(goalFile); err == nil {
			contentStr := string(content)
			if idx := strings.Index(contentStr, "## Status"); idx != -1 {
				contentStr = contentStr[:idx] + worktreeSection + "\n" + contentStr[idx:]
			} else {
				contentStr += worktreeSection
			}
			os.WriteFile(goalFile, []byte(contentStr), 0644)
		}

		// Update project config
		projectConfig := filepath.Join(opts.VegaDir, "projects", opts.Project+".md")
		addGoalToProjectConfig(projectConfig, goalID, opts.Title)
	}

	return &Result{Success: true}, result
}

// ListProjects returns all registered projects
func ListProjects(vegaDir string) ([]goals.Project, error) {
	return goals.ParseProjects(vegaDir)
}

// Helper functions

func getGoalTitle(goalFile, goalID string) string {
	file, err := os.Open(goalFile)
	if err != nil {
		return "goal-" + goalID
	}
	defer file.Close()

	re := regexp.MustCompile(`^# Goal #?[a-f0-9]+: (.+)$`)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if matches := re.FindStringSubmatch(scanner.Text()); matches != nil {
			return matches[1]
		}
	}
	return "goal-" + goalID
}

func getProjectBaseBranch(vegaDir, project string) (string, error) {
	p, err := goals.ParseProject(vegaDir, project)
	if err != nil {
		return "", err
	}
	if p.BaseBranch == "" {
		return "main", nil
	}
	return p.BaseBranch, nil
}

func findWorktreeDir(vegaDir, project, goalID string) (string, error) {
	pattern := filepath.Join(vegaDir, "workspaces", project, fmt.Sprintf("goal-%s-*", goalID))
	matches, _ := filepath.Glob(pattern)

	var dirs []string
	for _, m := range matches {
		if info, err := os.Stat(m); err == nil && info.IsDir() {
			dirs = append(dirs, m)
		}
	}

	if len(dirs) == 0 {
		return "", fmt.Errorf("no worktree found matching pattern: goal-%s-*", goalID)
	}
	if len(dirs) > 1 {
		return "", fmt.Errorf("multiple worktrees found for goal %s", goalID)
	}
	return dirs[0], nil
}

func getWorktreeBranch(worktreeDir string) (string, error) {
	cmd := exec.Command("git", "-C", worktreeDir, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return filepath.Base(worktreeDir), nil
	}
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return filepath.Base(worktreeDir), nil
	}
	return branch, nil
}

func checkWorktreeClean(worktreeDir string) error {
	cmd := exec.Command("git", "-C", worktreeDir, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}
	if len(strings.TrimSpace(string(output))) > 0 {
		return fmt.Errorf("uncommitted changes:\n%s", string(output))
	}
	return nil
}

func mergeBranch(projectBase, sourceBranch, targetBranch, goalID, goalTitle string) error {
	cmd := exec.Command("git", "-C", projectBase, "checkout", targetBranch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("checkout %s: %s", targetBranch, string(output))
	}

	mergeMsg := fmt.Sprintf("Merge goal %s: %s", goalID, goalTitle)
	cmd = exec.Command("git", "-C", projectBase, "merge", sourceBranch, "-m", mergeMsg)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("merge: %s", string(output))
	}
	return nil
}

func removeWorktree(projectBase, worktreeDir string) {
	// Calculate relative path from projectBase to worktreeDir
	relPath, err := filepath.Rel(projectBase, worktreeDir)
	if err != nil {
		relPath = worktreeDir
	}
	cmd := exec.Command("git", "-C", projectBase, "worktree", "remove", relPath, "--force")
	if err := cmd.Run(); err != nil {
		os.RemoveAll(worktreeDir)
		exec.Command("git", "-C", projectBase, "worktree", "prune").Run()
	}
}

func deleteBranch(projectBase, branchName string) error {
	cmd := exec.Command("git", "-C", projectBase, "branch", "-d", branchName)
	if err := cmd.Run(); err != nil {
		cmd = exec.Command("git", "-C", projectBase, "branch", "-D", branchName)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("could not delete branch: %w", err)
		}
	}
	return nil
}

func findGoalBranch(projectBase, goalID string) (string, error) {
	cmd := exec.Command("git", "-C", projectBase, "branch", "--list", fmt.Sprintf("goal-%s-*", goalID))
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git branch list failed: %w", err)
	}

	branches := strings.TrimSpace(string(output))
	if branches == "" {
		return "", fmt.Errorf("no branch found matching pattern goal-%s-*", goalID)
	}

	lines := strings.Split(branches, "\n")
	for _, line := range lines {
		branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if branch != "" {
			return branch, nil
		}
	}
	return "", fmt.Errorf("no valid branch found")
}

func generateGoalID() string {
	// Use first 7 chars of a UUID-like string
	return fmt.Sprintf("%x", time.Now().UnixNano())[:7]
}

func slugify(title string) string {
	// Convert to lowercase, replace spaces with dashes, remove special chars
	slug := strings.ToLower(title)
	slug = regexp.MustCompile(`[^a-z0-9\s-]`).ReplaceAllString(slug, "")
	slug = regexp.MustCompile(`\s+`).ReplaceAllString(slug, "-")
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	// Truncate if too long
	if len(slug) > 40 {
		slug = slug[:40]
	}
	return slug
}

func createGoalFile(goalFile, goalID, title, project string) error {
	content := fmt.Sprintf(`# Goal %s: %s

## Overview

<Brief description of the goal>

## Project(s)

- **%s**: <what this project does for the goal>

## Phases

### Phase 1: <Phase Title>
- [ ] Task 1
- [ ] Task 2

### Phase 2: <Phase Title>
- [ ] Task 1
- [ ] Task 2

## Status

Current Phase: 1/?
`, goalID, title, project)

	return os.WriteFile(goalFile, []byte(content), 0644)
}

func createWorktree(projectBase, worktreePath, branchName, baseBranch string) error {
	// Calculate relative path from projectBase to worktreePath
	// projectBase is like /path/workspaces/project/worktree-base
	// worktreePath is like /path/workspaces/project/goal-xxx
	// So relative path should be ../goal-xxx
	relPath, err := filepath.Rel(projectBase, worktreePath)
	if err != nil {
		// Fall back to absolute path if relative fails
		relPath = worktreePath
	}
	cmd := exec.Command("git", "-C", projectBase, "worktree", "add", "-b", branchName, relPath, baseBranch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add: %s", string(output))
	}
	return nil
}

func copyHooksToWorktree(vegaDir, worktreePath string) {
	// Copy hooks from template
	templateHooks := filepath.Join(vegaDir, "templates", "project-init", ".claude", "hooks")
	destHooks := filepath.Join(worktreePath, ".claude", "hooks")

	os.MkdirAll(destHooks, 0755)

	// Copy each hook file
	files, _ := os.ReadDir(templateHooks)
	for _, f := range files {
		src := filepath.Join(templateHooks, f.Name())
		dst := filepath.Join(destHooks, f.Name())
		if content, err := os.ReadFile(src); err == nil {
			os.WriteFile(dst, content, 0755)
		}
	}

	// Also copy rules
	templateRules := filepath.Join(vegaDir, "templates", "project-init", ".claude", "rules")
	destRules := filepath.Join(worktreePath, ".claude", "rules")
	os.MkdirAll(destRules, 0755)

	// Recursively copy rules
	filepath.Walk(templateRules, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(templateRules, path)
		dst := filepath.Join(destRules, rel)
		os.MkdirAll(filepath.Dir(dst), 0755)
		if content, err := os.ReadFile(path); err == nil {
			os.WriteFile(dst, content, 0644)
		}
		return nil
	})

	// Copy settings.local.json
	templateSettings := filepath.Join(vegaDir, "templates", "project-init", ".claude", "settings.local.json")
	destSettings := filepath.Join(worktreePath, ".claude", "settings.local.json")
	if content, err := os.ReadFile(templateSettings); err == nil {
		os.WriteFile(destSettings, content, 0644)
	}
}

func addGoalToRegistry(registryPath, goalID, title, project string) error {
	content, err := os.ReadFile(registryPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	added := false

	// Add after Active Goals table header
	for i, line := range lines {
		newLines = append(newLines, line)

		if !added && strings.Contains(line, "| ID | Title | Project(s) | Status | Phase |") {
			// Next line is separator
			if i+1 < len(lines) && strings.Contains(lines[i+1], "|---") {
				newLines = append(newLines, lines[i+1])
				// Add new goal
				newRow := fmt.Sprintf("| %s | %s | %s | Active | 1/? |", goalID, title, project)
				newLines = append(newLines, newRow)
				added = true
				// Skip the separator in main loop
				lines = append(lines[:i+1], lines[i+2:]...)
			}
		}
	}

	return os.WriteFile(registryPath, []byte(strings.Join(newLines, "\n")), 0644)
}

func addGoalToProjectConfig(configPath, goalID, title string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string

	// Add to Active Goals section
	for _, line := range lines {
		newLines = append(newLines, line)
		if strings.Contains(line, "## Active Goals") {
			newLines = append(newLines, fmt.Sprintf("- %s: %s", goalID, title))
		}
	}

	return os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0644)
}

func completeGoalInRegistry(registryPath, goalID, goalTitle, project string) error {
	content, err := os.ReadFile(registryPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	today := time.Now().Format("2006-01-02")
	addedToCompleted := false

	// Remove from Active Goals
	activePattern := regexp.MustCompile(fmt.Sprintf(`^\| %s \|.*\| Active \|`, regexp.QuoteMeta(goalID)))
	for _, line := range lines {
		if activePattern.MatchString(line) {
			continue
		}
		newLines = append(newLines, line)
	}

	// Add to Completed Goals
	var finalLines []string
	for i, line := range newLines {
		finalLines = append(finalLines, line)

		if !addedToCompleted && strings.Contains(line, "| ID | Title | Project(s) | Completed |") {
			if i+1 < len(newLines) && strings.Contains(newLines[i+1], "|---") {
				finalLines = append(finalLines, newLines[i+1])
				completedRow := fmt.Sprintf("| %s | %s | %s | %s |", goalID, goalTitle, project, today)
				finalLines = append(finalLines, completedRow)
				addedToCompleted = true
				newLines = append(newLines[:i+1], newLines[i+2:]...)
			}
		}
	}

	if !addedToCompleted {
		finalLines = newLines
	}

	return os.WriteFile(registryPath, []byte(strings.Join(finalLines, "\n")), 0644)
}

func completeGoalInProjectConfig(configPath, goalID, goalTitle string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	today := time.Now().Format("2006-01-02")

	activePattern := regexp.MustCompile(fmt.Sprintf(`^- #?%s:`, regexp.QuoteMeta(goalID)))

	for i, line := range lines {
		if activePattern.MatchString(line) {
			continue
		}
		newLines = append(newLines, line)

		if strings.Contains(line, "| ID | Title | Completed |") {
			if i+1 < len(lines) && strings.Contains(lines[i+1], "|---") {
				newLines = append(newLines, lines[i+1])
				completedRow := fmt.Sprintf("| %s | %s | %s |", goalID, goalTitle, today)
				newLines = append(newLines, completedRow)
			}
		}
	}

	return os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0644)
}

func iceGoalInRegistry(registryPath, goalID, reason string) error {
	content, err := os.ReadFile(registryPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	var goalTitle, goalProject string
	addedToIced := false

	// First pass: remove from Active Goals and capture details
	activePattern := regexp.MustCompile(fmt.Sprintf(`^\| %s \|([^|]*)\|([^|]*)\| Active`, regexp.QuoteMeta(goalID)))
	for _, line := range lines {
		if matches := activePattern.FindStringSubmatch(line); matches != nil {
			goalTitle = strings.TrimSpace(matches[1])
			goalProject = strings.TrimSpace(matches[2])
			continue // Skip this line (remove from active)
		}
		newLines = append(newLines, line)
	}

	// Second pass: add to Iced Goals section
	var finalLines []string
	for i, line := range newLines {
		finalLines = append(finalLines, line)

		// Add after Iced Goals table header
		if !addedToIced && strings.Contains(line, "| ID | Title | Project(s) | Reason |") {
			// Check if next line is separator
			if i+1 < len(newLines) && strings.Contains(newLines[i+1], "|---") {
				finalLines = append(finalLines, newLines[i+1])
				// Add the iced goal row
				icedRow := fmt.Sprintf("| %s | %s | %s | %s |", goalID, goalTitle, goalProject, reason)
				finalLines = append(finalLines, icedRow)
				addedToIced = true
				// Skip the separator in the next iteration
				newLines = append(newLines[:i+1], newLines[i+2:]...)
			}
		}
	}

	if !addedToIced {
		finalLines = newLines
	}

	return os.WriteFile(registryPath, []byte(strings.Join(finalLines, "\n")), 0644)
}

func updateGoalStatus(goalFile, status, reason string) error {
	content, err := os.ReadFile(goalFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string

	for _, line := range lines {
		newLines = append(newLines, line)
	}

	// Add status section if not present
	newLines = append(newLines, "")
	newLines = append(newLines, fmt.Sprintf("## Status: %s", status))
	if reason != "" {
		newLines = append(newLines, fmt.Sprintf("Reason: %s", reason))
	}
	newLines = append(newLines, fmt.Sprintf("Updated: %s", time.Now().Format("2006-01-02")))

	return os.WriteFile(goalFile, []byte(strings.Join(newLines, "\n")), 0644)
}

func recreateWorktree(projectBase, worktreePath, branchName string) error {
	// Calculate relative path from projectBase to worktreePath
	relPath, err := filepath.Rel(projectBase, worktreePath)
	if err != nil {
		relPath = worktreePath
	}
	// Use existing branch (don't create new one)
	cmd := exec.Command("git", "-C", projectBase, "worktree", "add", relPath, branchName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree add: %s", string(output))
	}
	return nil
}

func resumeGoalInRegistry(registryPath, goalID, goalTitle, project string) error {
	content, err := os.ReadFile(registryPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	addedToActive := false

	// First pass: remove from Iced Goals section
	icedPattern := regexp.MustCompile(fmt.Sprintf(`^\| %s \|`, regexp.QuoteMeta(goalID)))
	inIcedSection := false
	for _, line := range lines {
		if strings.Contains(line, "## Iced Goals") {
			inIcedSection = true
		} else if strings.HasPrefix(line, "## ") {
			inIcedSection = false
		}

		// Skip the goal row if we're in the Iced section
		if inIcedSection && icedPattern.MatchString(line) {
			continue
		}
		newLines = append(newLines, line)
	}

	// Second pass: add to Active Goals section
	var finalLines []string
	for i, line := range newLines {
		finalLines = append(finalLines, line)

		// Add after Active Goals table header
		if !addedToActive && strings.Contains(line, "| ID | Title | Project(s) | Status | Phase |") {
			// Check if next line is separator
			if i+1 < len(newLines) && strings.Contains(newLines[i+1], "|---") {
				finalLines = append(finalLines, newLines[i+1])
				// Add the active goal row
				activeRow := fmt.Sprintf("| %s | %s | %s | Active | 1/? |", goalID, goalTitle, project)
				finalLines = append(finalLines, activeRow)
				addedToActive = true
				// Skip the separator in the next iteration
				newLines = append(newLines[:i+1], newLines[i+2:]...)
			}
		}
	}

	if !addedToActive {
		finalLines = newLines
	}

	return os.WriteFile(registryPath, []byte(strings.Join(finalLines, "\n")), 0644)
}
