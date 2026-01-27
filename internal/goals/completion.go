package goals

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CompletionStatus represents the completion state of a goal
type CompletionStatus struct {
	Complete        bool     `json:"complete"`
	CompletedPhases int      `json:"completed_phases"`
	TotalPhases     int      `json:"total_phases"`
	MissingTasks    []string `json:"missing_tasks"`
}

// IsGoalComplete checks if a goal is complete by parsing its task_plan.md
// It looks for the task plan in the goal's planning directory
func IsGoalComplete(goalID string, baseDir string) (*CompletionStatus, error) {
	taskPlanPath, err := findTaskPlan(goalID, baseDir)
	if err != nil {
		return nil, err
	}

	return parseTaskPlanCompletion(taskPlanPath)
}

// findTaskPlan locates the task_plan.md file for a goal
// Checks both active and history locations
func findTaskPlan(goalID string, baseDir string) (string, error) {
	// Try common locations for task plans
	locations := []string{
		// Active planning location
		filepath.Join(baseDir, "docs", "planning", "goal-"+goalID, "task_plan.md"),
		// History location
		filepath.Join(baseDir, "docs", "planning", "history", "goal-"+goalID, "task_plan.md"),
		// Alternative: directly in goal folder
		filepath.Join(baseDir, "goals", "active", goalID, "task_plan.md"),
		filepath.Join(baseDir, "goals", "history", goalID, "task_plan.md"),
	}

	for _, path := range locations {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", os.ErrNotExist
}

// parseTaskPlanCompletion parses a task_plan.md file and extracts completion status
func parseTaskPlanCompletion(path string) (*CompletionStatus, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	status := &CompletionStatus{
		MissingTasks: []string{},
	}

	scanner := bufio.NewScanner(file)

	// Regex patterns
	// Phase header: ## Phase N: Title [status] or ### Phase N: Title
	phaseHeaderRe := regexp.MustCompile(`^#{2,3}\s+Phase\s+\d+:\s*(.+?)(?:\s+\[(complete|in_progress|pending)\])?$`)
	// Task checkbox: - [x] or - [ ]
	taskRe := regexp.MustCompile(`^-\s+\[([ xX])\]\s+(.+)$`)

	var currentPhase string
	var phaseComplete bool
	var inPhase bool
	phaseHasTasks := false

	for scanner.Scan() {
		line := scanner.Text()

		// Check for phase headers
		if matches := phaseHeaderRe.FindStringSubmatch(line); matches != nil {
			// Save previous phase status
			if inPhase && phaseHasTasks && phaseComplete {
				status.CompletedPhases++
			}

			// Start new phase
			currentPhase = strings.TrimSpace(matches[1])
			status.TotalPhases++
			phaseComplete = true // Assume complete until we find an incomplete task
			phaseHasTasks = false
			inPhase = true

			// Check for explicit [complete] marker
			if len(matches) > 2 && matches[2] == "complete" {
				// Phase explicitly marked as complete
				continue
			}
			continue
		}

		// Check for task checkboxes
		if matches := taskRe.FindStringSubmatch(line); matches != nil {
			phaseHasTasks = true
			checkbox := strings.ToLower(matches[1])
			taskDesc := strings.TrimSpace(matches[2])

			if checkbox != "x" {
				// Incomplete task
				phaseComplete = false
				// Add to missing tasks with phase context
				if currentPhase != "" {
					status.MissingTasks = append(status.MissingTasks, currentPhase+": "+taskDesc)
				} else {
					status.MissingTasks = append(status.MissingTasks, taskDesc)
				}
			}
		}
	}

	// Don't forget the last phase
	if inPhase && phaseHasTasks && phaseComplete {
		status.CompletedPhases++
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Goal is complete if all phases are complete and there are phases
	status.Complete = status.TotalPhases > 0 && status.CompletedPhases == status.TotalPhases

	return status, nil
}

// CompletionChecker provides methods for checking goal completion
type CompletionChecker struct {
	baseDir string
}

// NewCompletionChecker creates a new CompletionChecker for the given base directory
func NewCompletionChecker(baseDir string) *CompletionChecker {
	return &CompletionChecker{baseDir: baseDir}
}

// CheckGoal returns the completion status for a specific goal
func (c *CompletionChecker) CheckGoal(goalID string) (*CompletionStatus, error) {
	return IsGoalComplete(goalID, c.baseDir)
}

// IsComplete is a convenience method that returns just the completion boolean
func (c *CompletionChecker) IsComplete(goalID string) (bool, error) {
	status, err := IsGoalComplete(goalID, c.baseDir)
	if err != nil {
		return false, err
	}
	return status.Complete, nil
}
