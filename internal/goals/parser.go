package goals

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Goal represents a goal from the registry
type Goal struct {
	ID       int      `json:"id"`
	Title    string   `json:"title"`
	Projects []string `json:"projects"`
	Status   string   `json:"status"` // "active", "iced", "completed"
	Phase    string   `json:"phase"`  // e.g., "1/4" or "?"
	Reason   string   `json:"reason,omitempty"` // For iced goals
}

// Parser handles parsing of goal registry and detail files
type Parser struct {
	dir string
}

// NewParser creates a parser for the given vega-missile directory
func NewParser(dir string) *Parser {
	return &Parser{dir: dir}
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

	id, err := strconv.Atoi(parts[0])
	if err != nil {
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

	id, err := strconv.Atoi(parts[0])
	if err != nil {
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

	id, err := strconv.Atoi(parts[0])
	if err != nil {
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

// GoalDetail contains detailed information about a specific goal
type GoalDetail struct {
	Goal
	Overview   string        `json:"overview,omitempty"`
	Phases     []PhaseDetail `json:"phases,omitempty"`
	Acceptance []string      `json:"acceptance,omitempty"`
	Notes      []string      `json:"notes,omitempty"`
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

// ParseGoalDetail reads and parses a specific goal file
func (p *Parser) ParseGoalDetail(id int) (*GoalDetail, error) {
	goalPath := filepath.Join(p.dir, "goals", "active", strconv.Itoa(id)+".md")
	goalStatus := "active"

	// Try active first, then iced, then completed
	if _, err := os.Stat(goalPath); os.IsNotExist(err) {
		goalPath = filepath.Join(p.dir, "goals", "iced", strconv.Itoa(id)+".md")
		goalStatus = "iced"
		if _, err := os.Stat(goalPath); os.IsNotExist(err) {
			goalPath = filepath.Join(p.dir, "goals", "history", strconv.Itoa(id)+".md")
			goalStatus = "completed"
		}
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
	titleRe := regexp.MustCompile(`^# Goal #(\d+): (.+)$`)
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
