package goals

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Goal represents a goal from the registry
type Goal struct {
	ID       string   `json:"id"` // Can be numeric ("10"), hash ("4fd584d"), or hierarchical ("4fd584d.1")
	Title    string   `json:"title"`
	Projects []string `json:"projects"`
	Status   string   `json:"status"` // "active", "iced", "completed"
	Phase    string   `json:"phase"`  // e.g., "1/4" or "?"
	Reason   string   `json:"reason,omitempty"` // For iced goals
	ParentID string   `json:"parent_id,omitempty"` // Parent goal ID for hierarchical goals
	Children []string `json:"children,omitempty"` // Child goal IDs (populated dynamically)
}

// Parser handles parsing of goal registry and detail files
type Parser struct {
	dir string
}

// NewParser creates a parser for the given vega-missile directory
func NewParser(dir string) *Parser {
	return &Parser{dir: dir}
}

// Dir returns the vega-missile directory path
func (p *Parser) Dir() string {
	return p.dir
}

// ParseRegistry reads and parses the REGISTRY.md file
func (p *Parser) ParseRegistry() ([]Goal, error) {
	registryPath := filepath.Join(p.dir, "goals", "REGISTRY.md")
	file, err := os.Open(registryPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var goals []Goal
	scanner := bufio.NewScanner(file)

	// Track which section we're in
	section := ""
	inTable := false

	for scanner.Scan() {
		line := scanner.Text()

		// Detect section headers
		if strings.HasPrefix(line, "## Active Goals") {
			section = "active"
			inTable = false
			continue
		} else if strings.HasPrefix(line, "## Iced Goals") {
			section = "iced"
			inTable = false
			continue
		} else if strings.HasPrefix(line, "## Completed Goals") {
			section = "completed"
			inTable = false
			continue
		} else if strings.HasPrefix(line, "## ") {
			// Other sections we don't care about
			section = ""
			inTable = false
			continue
		}

		// Skip table headers (lines with |---)
		if strings.Contains(line, "|---") {
			inTable = true
			continue
		}

		// Skip non-table lines
		if !strings.HasPrefix(line, "|") || !strings.HasSuffix(strings.TrimSpace(line), "|") {
			continue
		}

		// Parse table row
		if !inTable {
			// First row after section header is the header row, skip it
			inTable = true
			continue
		}

		// Split by | and clean up
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}

		// Remove first and last empty elements from split
		parts = parts[1 : len(parts)-1]
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}

		// Parse based on section
		switch section {
		case "active":
			if len(parts) >= 5 && parts[0] != "" {
				goal := parseActiveGoal(parts)
				if goal != nil {
					goals = append(goals, *goal)
				}
			}
		case "iced":
			if len(parts) >= 4 && parts[0] != "" {
				goal := parseIcedGoal(parts)
				if goal != nil {
					goals = append(goals, *goal)
				}
			}
		case "completed":
			if len(parts) >= 4 && parts[0] != "" {
				goal := parseCompletedGoal(parts)
				if goal != nil {
					goals = append(goals, *goal)
				}
			}
		}
	}

	return goals, scanner.Err()
}

// parseActiveGoal parses an active goal row: | ID | Title | Project(s) | Status | Phase |
func parseActiveGoal(parts []string) *Goal {
	if len(parts) < 5 {
		return nil
	}

	id := strings.TrimSpace(parts[0])
	if id == "" {
		return nil
	}

	return &Goal{
		ID:       id,
		Title:    parts[1],
		Projects: parseProjects(parts[2]),
		Status:   "active",
		Phase:    parts[4],
	}
}

// parseIcedGoal parses an iced goal row: | ID | Title | Project(s) | Reason |
func parseIcedGoal(parts []string) *Goal {
	if len(parts) < 4 {
		return nil
	}

	id := strings.TrimSpace(parts[0])
	if id == "" {
		return nil
	}

	return &Goal{
		ID:       id,
		Title:    parts[1],
		Projects: parseProjects(parts[2]),
		Status:   "iced",
		Reason:   parts[3],
	}
}

// parseCompletedGoal parses a completed goal row: | ID | Title | Project(s) | Completed |
func parseCompletedGoal(parts []string) *Goal {
	if len(parts) < 4 {
		return nil
	}

	id := strings.TrimSpace(parts[0])
	if id == "" {
		return nil
	}

	return &Goal{
		ID:       id,
		Title:    parts[1],
		Projects: parseProjects(parts[2]),
		Status:   "completed",
	}
}

// parseProjects splits a comma-separated project list
func parseProjects(s string) []string {
	projects := strings.Split(s, ",")
	result := make([]string, 0, len(projects))
	for _, p := range projects {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// WorktreeInfo contains worktree metadata stored in goal file
type WorktreeInfo struct {
	Branch     string `json:"branch"`
	Project    string `json:"project"`
	Path       string `json:"path"`
	BaseBranch string `json:"base_branch"`
	Created    string `json:"created"`
}

// GoalDetail contains detailed information about a specific goal
type GoalDetail struct {
	Goal
	Overview   string        `json:"overview,omitempty"`
	Phases     []PhaseDetail `json:"phases,omitempty"`
	Acceptance []string      `json:"acceptance,omitempty"`
	Notes      []string      `json:"notes,omitempty"`
	Worktree   *WorktreeInfo `json:"worktree,omitempty"`
}

// PhaseDetail describes a phase within a goal
type PhaseDetail struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	Tasks  []Task   `json:"tasks"`
	Status string   `json:"status"` // "pending", "in_progress", "complete"
}

// Task represents a single task item
type Task struct {
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

// findGoalFile locates the goal markdown file, checking both flat and folder structures
// Returns (path, status) where status is "active", "iced", or "completed"
func (p *Parser) findGoalFile(id string) (string, string) {
	dirs := []struct {
		name   string
		status string
	}{
		{"active", "active"},
		{"iced", "iced"},
		{"history", "completed"},
	}

	for _, dir := range dirs {
		// Flat structure: goals/<dir>/<id>.md
		flatPath := filepath.Join(p.dir, "goals", dir.name, id+".md")
		if _, err := os.Stat(flatPath); err == nil {
			return flatPath, dir.status
		}

		// Folder structure: goals/<dir>/<id>/<id>.md
		folderPath := filepath.Join(p.dir, "goals", dir.name, id, id+".md")
		if _, err := os.Stat(folderPath); err == nil {
			return folderPath, dir.status
		}
	}

	return "", ""
}

// ParseGoalDetail reads and parses a specific goal file
func (p *Parser) ParseGoalDetail(id string) (*GoalDetail, error) {
	goalPath, goalStatus := p.findGoalFile(id)
	if goalPath == "" {
		return nil, os.ErrNotExist
	}

	file, err := os.Open(goalPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	detail := &GoalDetail{
		Goal: Goal{ID: id, Status: goalStatus},
	}

	scanner := bufio.NewScanner(file)
	section := ""
	var currentPhase *PhaseDetail
	phaseNum := 0

	// Regex patterns
	titleRe := regexp.MustCompile(`^# Goal #?([0-9a-f]+): (.+)$`)
	phaseRe := regexp.MustCompile(`^### Phase (\d+): (.+)$`)
	taskRe := regexp.MustCompile(`^- \[([ x])\] (.+)$`)

	var overviewLines []string
	var noteLines []string
	var acceptanceLines []string

	for scanner.Scan() {
		line := scanner.Text()

		// Parse title
		if matches := titleRe.FindStringSubmatch(line); matches != nil {
			detail.Title = matches[2]
			continue
		}

		// Detect sections
		if strings.HasPrefix(line, "## Overview") {
			section = "overview"
			continue
		} else if strings.HasPrefix(line, "## Phases") {
			section = "phases"
			continue
		} else if strings.HasPrefix(line, "## Acceptance Criteria") {
			section = "acceptance"
			continue
		} else if strings.HasPrefix(line, "## Notes") {
			section = "notes"
			continue
		} else if strings.HasPrefix(line, "## Status") {
			section = "status"
			continue
		} else if strings.HasPrefix(line, "## Project") {
			section = "project"
			continue
		} else if strings.HasPrefix(line, "## Worktree") {
			section = "worktree"
			continue
		} else if strings.HasPrefix(line, "## ") {
			section = ""
			continue
		}

		// Parse phase headers within Phases section
		if section == "phases" {
			if matches := phaseRe.FindStringSubmatch(line); matches != nil {
				// Save previous phase if exists
				if currentPhase != nil {
					detail.Phases = append(detail.Phases, *currentPhase)
				}
				phaseNum, _ = strconv.Atoi(matches[1])
				currentPhase = &PhaseDetail{
					Number: phaseNum,
					Title:  matches[2],
					Status: "pending",
				}
				continue
			}

			// Parse tasks
			if currentPhase != nil {
				if matches := taskRe.FindStringSubmatch(line); matches != nil {
					completed := matches[1] == "x"
					currentPhase.Tasks = append(currentPhase.Tasks, Task{
						Description: matches[2],
						Completed:   completed,
					})
				}
			}
		}

		// Collect overview lines
		if section == "overview" && strings.TrimSpace(line) != "" {
			overviewLines = append(overviewLines, line)
		}

		// Parse acceptance criteria
		if section == "acceptance" {
			if matches := taskRe.FindStringSubmatch(line); matches != nil {
				acceptanceLines = append(acceptanceLines, matches[2])
			}
		}

		// Parse notes
		if section == "notes" && strings.HasPrefix(line, "- ") {
			noteLines = append(noteLines, strings.TrimPrefix(line, "- "))
		}

		// Parse status section for current phase
		if section == "status" && strings.Contains(line, "Current Phase") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				detail.Phase = strings.TrimSpace(parts[1])
			}
		}

		// Parse project
		if section == "project" && strings.HasPrefix(line, "- **") {
			// Extract project name from "- **project-name**: description"
			re := regexp.MustCompile(`^\- \*\*([^*]+)\*\*`)
			if matches := re.FindStringSubmatch(line); matches != nil {
				detail.Projects = append(detail.Projects, matches[1])
			}
		}

		// Parse worktree metadata
		if section == "worktree" && strings.HasPrefix(line, "- **") {
			if detail.Worktree == nil {
				detail.Worktree = &WorktreeInfo{}
			}
			// Parse: - **Key**: value
			re := regexp.MustCompile(`^\- \*\*([^*]+)\*\*:\s*(.+)$`)
			if matches := re.FindStringSubmatch(line); matches != nil {
				key := strings.ToLower(matches[1])
				value := strings.TrimSpace(matches[2])
				switch key {
				case "branch":
					detail.Worktree.Branch = value
				case "project":
					detail.Worktree.Project = value
				case "path":
					detail.Worktree.Path = value
				case "base branch":
					detail.Worktree.BaseBranch = value
				case "created":
					detail.Worktree.Created = value
				}
			}
		}
	}

	// Don't forget the last phase
	if currentPhase != nil {
		detail.Phases = append(detail.Phases, *currentPhase)
	}

	// Set phase statuses based on tasks
	for i := range detail.Phases {
		phase := &detail.Phases[i]
		allComplete := true
		anyStarted := false
		for _, task := range phase.Tasks {
			if !task.Completed {
				allComplete = false
			} else {
				anyStarted = true
			}
		}
		if allComplete && len(phase.Tasks) > 0 {
			phase.Status = "complete"
		} else if anyStarted {
			phase.Status = "in_progress"
		}
	}

	detail.Overview = strings.Join(overviewLines, "\n")
	detail.Acceptance = acceptanceLines
	detail.Notes = noteLines

	return detail, scanner.Err()
}

// Project represents a managed project from projects/<name>.md
type Project struct {
	Name            string `json:"name"`
	Workspace       string `json:"workspace"`
	BaseBranch      string `json:"base_branch"`
	Upstream        string `json:"upstream"`         // Git remote URL or local path
	GitRemote       string `json:"git_remote"`       // Resolved git remote URL (from upstream or repo)
	WorkspaceStatus string `json:"workspace_status"` // "ready", "missing", "error"
	WorkspaceError  string `json:"workspace_error,omitempty"`
}

// ParseProject reads and parses a project configuration file
// Returns project details including git remote for credential validation
func (p *Parser) ParseProject(name string) (*Project, error) {
	projectPath := filepath.Join(p.dir, "projects", name+".md")
	file, err := os.Open(projectPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	project := &Project{Name: name}
	scanner := bufio.NewScanner(file)

	// Regex patterns for project config
	// Matches: **Workspace**: `workspaces/name/...` or - **Workspace**: `...`
	workspaceRe := regexp.MustCompile(`(?:\*\*)?Workspace(?:\*\*)?:?\s*` + "`" + `?([^` + "`" + `]+)` + "`?")
	// Matches: **Base Branch**: `master` or Base Branch: master
	baseBranchRe := regexp.MustCompile(`(?i)(?:\*\*)?Base Branch(?:\*\*)?:?\s*` + "`?" + `([a-zA-Z0-9_/-]+)` + "`?")
	// Matches: **Upstream**: `https://github.com/...` or Upstream: /local/path
	upstreamRe := regexp.MustCompile(`(?:\*\*)?Upstream(?:\*\*)?:?\s*` + "`?" + `([^` + "`" + `\s]+)` + "`?")

	for scanner.Scan() {
		line := scanner.Text()

		if matches := workspaceRe.FindStringSubmatch(line); matches != nil {
			project.Workspace = strings.TrimSpace(matches[1])
		}
		if matches := baseBranchRe.FindStringSubmatch(line); matches != nil {
			project.BaseBranch = strings.TrimSpace(matches[1])
		}
		if matches := upstreamRe.FindStringSubmatch(line); matches != nil {
			project.Upstream = strings.TrimSpace(matches[1])
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Resolve GitRemote from Upstream
	// If Upstream is a local path, try to get the actual git remote from the repo
	if project.Upstream != "" {
		if strings.HasPrefix(project.Upstream, "/") || strings.HasPrefix(project.Upstream, "~") {
			// Local path - try to get remote from the workspace
			workspacePath := filepath.Join(p.dir, "workspaces", name, "worktree-base")
			if remote, err := getGitRemote(workspacePath); err == nil {
				project.GitRemote = remote
			} else {
				// Fall back to trying the upstream path directly
				if remote, err := getGitRemote(project.Upstream); err == nil {
					project.GitRemote = remote
				}
			}
		} else {
			// Already a git URL
			project.GitRemote = project.Upstream
		}
	} else {
		// No upstream - try to get remote from workspace
		workspacePath := filepath.Join(p.dir, "workspaces", name, "worktree-base")
		if remote, err := getGitRemote(workspacePath); err == nil {
			project.GitRemote = remote
		}
	}

	// Check workspace status
	project.WorkspaceStatus, project.WorkspaceError = checkWorkspaceStatus(p.dir, name)

	return project, nil
}

// checkWorkspaceStatus checks if a project's workspace is properly set up
func checkWorkspaceStatus(vegaDir, projectName string) (status, errorMsg string) {
	worktreeBase := filepath.Join(vegaDir, "workspaces", projectName, "worktree-base")

	// Check if worktree-base directory exists
	info, err := os.Stat(worktreeBase)
	if os.IsNotExist(err) {
		return "missing", "Workspace not set up: workspaces/" + projectName + "/worktree-base/ does not exist"
	}
	if err != nil {
		return "error", "Cannot access workspace: " + err.Error()
	}
	if !info.IsDir() {
		return "error", "Workspace path exists but is not a directory"
	}

	// Check if it's a valid git repository
	gitDir := filepath.Join(worktreeBase, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return "error", "Workspace exists but is not a git repository"
	}

	return "ready", ""
}

// ParseProject is a standalone function to parse a project config
func ParseProject(vegaDir, name string) (*Project, error) {
	p := NewParser(vegaDir)
	return p.ParseProject(name)
}

// ParseProjects returns all projects with their details
func ParseProjects(vegaDir string) ([]Project, error) {
	p := NewParser(vegaDir)
	names, err := p.ListProjects()
	if err != nil {
		return nil, err
	}

	var projects []Project
	for _, name := range names {
		proj, err := p.ParseProject(name)
		if err != nil {
			continue // Skip projects that fail to parse
		}
		projects = append(projects, *proj)
	}
	return projects, nil
}

// getGitRemote gets the origin remote URL from a git repository
func getGitRemote(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// ListProjects returns all project names from projects/index.md
func (p *Parser) ListProjects() ([]string, error) {
	indexPath := filepath.Join(p.dir, "projects", "index.md")
	file, err := os.Open(indexPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var projects []string
	scanner := bufio.NewScanner(file)

	// Look for markdown links: [project-name](project-name.md)
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\.md\)`)

	for scanner.Scan() {
		line := scanner.Text()
		if matches := linkRe.FindStringSubmatch(line); matches != nil {
			// matches[1] is the display name, matches[2] is the filename
			projects = append(projects, matches[2])
		}
	}

	return projects, scanner.Err()
}
