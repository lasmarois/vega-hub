package hub

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GoalStatus represents the status parsed from planning files
type GoalStatus struct {
	CurrentPhase  string        `json:"current_phase"`
	RecentActions []string      `json:"recent_actions"`
	ProgressLog   string        `json:"progress_log"`
	TaskPlan      string        `json:"task_plan"`
	Findings      string        `json:"findings"`
	HasWorktree   bool          `json:"has_worktree"`
	WorktreePath  string        `json:"worktree_path,omitempty"`
	PhaseProgress []PhaseStatus `json:"phase_progress,omitempty"`
}

// PhaseStatus represents the status of a single phase
type PhaseStatus struct {
	Number     int    `json:"number"`
	Title      string `json:"title"`
	Status     string `json:"status"` // "pending", "in_progress", "complete"
	TasksTotal int    `json:"tasks_total"`
	TasksDone  int    `json:"tasks_done"`
}

// GetGoalStatus reads planning files and returns the current status
func (h *Hub) GetGoalStatus(goalID string) (*GoalStatus, error) {
	status := &GoalStatus{
		HasWorktree: false,
	}

	// Find worktree
	worktree, err := h.findWorktree(goalID)
	if err != nil {
		// No worktree - return minimal status
		return status, nil
	}

	status.HasWorktree = true
	status.WorktreePath = worktree

	// Read task_plan.md
	taskPlanPath := filepath.Join(worktree, "task_plan.md")
	if content, err := readFileHead(taskPlanPath, 100); err == nil {
		status.TaskPlan = content
		status.CurrentPhase = parseCurrentPhase(content)
		status.PhaseProgress = parsePhaseProgress(content)
	}

	// Read progress.md
	progressPath := filepath.Join(worktree, "progress.md")
	if content, err := readFileHead(progressPath, 50); err == nil {
		status.ProgressLog = content
		status.RecentActions = parseRecentActions(content)
	}

	// Read findings.md
	findingsPath := filepath.Join(worktree, "findings.md")
	if content, err := readFileHead(findingsPath, 50); err == nil {
		status.Findings = content
	}

	return status, nil
}

// readFileHead reads the first N lines of a file
func readFileHead(path string, maxLines int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for i := 0; i < maxLines && scanner.Scan(); i++ {
		lines = append(lines, scanner.Text())
	}

	return strings.Join(lines, "\n"), scanner.Err()
}

// parseCurrentPhase extracts the current phase from task_plan.md
func parseCurrentPhase(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "## Current Phase") {
			// Next non-empty line should have the phase
			continue
		}
		if strings.HasPrefix(line, "Phase ") {
			return strings.TrimSpace(line)
		}
		// Look for inline format: "## Current Phase\nPhase X - Description"
		if strings.Contains(line, "Current Phase") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	// Try regex for "Phase X - ..." pattern
	re := regexp.MustCompile(`(?m)^## Current Phase\s*\n+(.+)$`)
	if matches := re.FindStringSubmatch(content); matches != nil {
		return strings.TrimSpace(matches[1])
	}

	return "Unknown"
}

// parseRecentActions extracts recent actions from progress.md
func parseRecentActions(content string) []string {
	var actions []string
	lines := strings.Split(content, "\n")

	inActionsSection := false
	for _, line := range lines {
		// Look for "Actions taken:" section
		if strings.Contains(line, "Actions taken:") {
			inActionsSection = true
			continue
		}

		// End section on new header or section marker
		if inActionsSection {
			if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "- **") {
				if len(actions) > 0 {
					break // Stop after first actions section
				}
				inActionsSection = false
				continue
			}

			// Collect action items (indented with -)
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "- ") {
				action := strings.TrimPrefix(trimmed, "- ")
				actions = append(actions, action)
			}
		}
	}

	// Limit to last 5 actions
	if len(actions) > 5 {
		actions = actions[len(actions)-5:]
	}

	return actions
}

// parsePhaseProgress extracts phase status from task_plan.md
func parsePhaseProgress(content string) []PhaseStatus {
	var phases []PhaseStatus

	lines := strings.Split(content, "\n")
	phaseRe := regexp.MustCompile(`^### Phase (\d+)[:\s]+(.+)$`)
	taskRe := regexp.MustCompile(`^- \[([ x])\] (.+)$`)
	statusRe := regexp.MustCompile(`^\*\*Status:\*\*\s*(\w+)`)

	var currentPhase *PhaseStatus

	for _, line := range lines {
		// Match phase header
		if matches := phaseRe.FindStringSubmatch(line); matches != nil {
			// Save previous phase
			if currentPhase != nil {
				phases = append(phases, *currentPhase)
			}

			var num int
			fmt.Sscanf(matches[1], "%d", &num)
			currentPhase = &PhaseStatus{
				Number: num,
				Title:  strings.TrimSpace(matches[2]),
				Status: "pending",
			}
			continue
		}

		if currentPhase != nil {
			// Match task checkbox
			if matches := taskRe.FindStringSubmatch(line); matches != nil {
				currentPhase.TasksTotal++
				if matches[1] == "x" {
					currentPhase.TasksDone++
				}
			}

			// Match explicit status
			if matches := statusRe.FindStringSubmatch(line); matches != nil {
				currentPhase.Status = strings.ToLower(matches[1])
			}
		}
	}

	// Save last phase
	if currentPhase != nil {
		phases = append(phases, *currentPhase)
	}

	// Update status based on task completion if not explicitly set
	for i := range phases {
		if phases[i].Status == "pending" && phases[i].TasksTotal > 0 {
			if phases[i].TasksDone == phases[i].TasksTotal {
				phases[i].Status = "complete"
			} else if phases[i].TasksDone > 0 {
				phases[i].Status = "in_progress"
			}
		}
	}

	return phases
}
