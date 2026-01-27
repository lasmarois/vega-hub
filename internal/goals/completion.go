package goals

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// CompletionSignalType defines the type of completion signal detected
type CompletionSignalType string

const (
	SignalPlanningFile CompletionSignalType = "planning_file"
	SignalAcceptance   CompletionSignalType = "acceptance"
	SignalCommit       CompletionSignalType = "commit"
	SignalGoalPhases   CompletionSignalType = "goal_phases"
)

// CompletionSignal represents a single signal indicating goal completion
type CompletionSignal struct {
	Type    CompletionSignalType `json:"type"`
	Source  string               `json:"source"`  // File path or commit hash
	Message string               `json:"message"` // Human-readable description
}

// CompletionStatus represents the completion state of a goal
type CompletionStatus struct {
	Complete        bool               `json:"complete"`
	Signals         []CompletionSignal `json:"signals"`
	CompletedPhases int                `json:"completed_phases"`
	TotalPhases     int                `json:"total_phases"`
	MissingTasks    []string           `json:"missing_tasks"`
	Confidence      float64            `json:"confidence"` // 0.0-1.0
}

// CompletionChecker provides methods for checking goal completion
type CompletionChecker struct {
	baseDir string
}

// NewCompletionChecker creates a new CompletionChecker for the given base directory
func NewCompletionChecker(baseDir string) *CompletionChecker {
	return &CompletionChecker{baseDir: baseDir}
}

// CheckGoal returns the comprehensive completion status for a goal
func (c *CompletionChecker) CheckGoal(goalID string) (*CompletionStatus, error) {
	status := &CompletionStatus{
		Signals:      []CompletionSignal{},
		MissingTasks: []string{},
	}

	// Parse goal detail to get worktree info and phases
	parser := NewParser(c.baseDir)
	detail, err := parser.ParseGoalDetail(goalID)
	if err != nil {
		return nil, err
	}

	// Check 1: Goal file phases
	c.checkGoalPhases(detail, status)

	// Check 2: Acceptance criteria in goal file
	c.checkAcceptanceCriteria(goalID, status)

	// Check 3: Planning file (task_plan.md) in worktree
	c.checkPlanningFile(detail, status)

	// Check 4: Commit messages
	c.checkCommitMessages(detail, status)

	// Calculate overall completion and confidence
	c.calculateCompletion(status)

	return status, nil
}

// IsComplete is a convenience method that returns just the completion boolean
func (c *CompletionChecker) IsComplete(goalID string) (bool, error) {
	status, err := c.CheckGoal(goalID)
	if err != nil {
		return false, err
	}
	return status.Complete, nil
}

// checkGoalPhases checks if all phases in the goal file are complete
func (c *CompletionChecker) checkGoalPhases(detail *GoalDetail, status *CompletionStatus) {
	if len(detail.Phases) == 0 {
		return
	}

	status.TotalPhases = len(detail.Phases)
	allComplete := true

	for _, phase := range detail.Phases {
		if phase.Status == "complete" {
			status.CompletedPhases++
		} else {
			allComplete = false
			// Add incomplete tasks to missing
			for _, task := range phase.Tasks {
				if !task.Completed {
					status.MissingTasks = append(status.MissingTasks,
						"Phase "+itoa(phase.Number)+": "+task.Description)
				}
			}
		}
	}

	if allComplete && len(detail.Phases) > 0 {
		status.Signals = append(status.Signals, CompletionSignal{
			Type:    SignalGoalPhases,
			Source:  "goal file",
			Message: "All " + itoa(len(detail.Phases)) + " phases marked complete",
		})
	}
}

// checkAcceptanceCriteria parses acceptance criteria from goal file and checks completion
func (c *CompletionChecker) checkAcceptanceCriteria(goalID string, status *CompletionStatus) {
	goalPath := c.findGoalFile(goalID)
	if goalPath == "" {
		return
	}

	file, err := os.Open(goalPath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inAcceptance := false
	taskRe := regexp.MustCompile(`^- \[([ xX])\] (.+)$`)

	var completed []string
	var incomplete []string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "## Acceptance Criteria") {
			inAcceptance = true
			continue
		}
		if inAcceptance && strings.HasPrefix(line, "## ") {
			break
		}

		if inAcceptance {
			if matches := taskRe.FindStringSubmatch(line); matches != nil {
				if strings.ToLower(matches[1]) == "x" {
					completed = append(completed, matches[2])
				} else {
					incomplete = append(incomplete, matches[2])
				}
			}
		}
	}

	// Add incomplete criteria to missing
	for _, item := range incomplete {
		status.MissingTasks = append(status.MissingTasks, "Acceptance: "+item)
	}

	// If all criteria are complete, add signal
	if len(completed) > 0 && len(incomplete) == 0 {
		status.Signals = append(status.Signals, CompletionSignal{
			Type:    SignalAcceptance,
			Source:  goalPath,
			Message: "All " + itoa(len(completed)) + " acceptance criteria met",
		})
	}
}

// checkPlanningFile checks task_plan.md in the worktree for completion
func (c *CompletionChecker) checkPlanningFile(detail *GoalDetail, status *CompletionStatus) {
	// Try to find task_plan.md in worktree first
	var planPath string
	if detail.Worktree != nil && detail.Worktree.Path != "" {
		planPath = filepath.Join(c.baseDir, detail.Worktree.Path, "task_plan.md")
	}

	// Fallback to docs/planning locations if worktree path doesn't have it
	if planPath == "" || !fileExists(planPath) {
		planPath = c.findTaskPlan(detail.ID)
	}

	if planPath == "" || !fileExists(planPath) {
		return
	}

	planStatus, err := parseTaskPlanCompletion(planPath)
	if err != nil {
		return
	}

	// If planning file shows all complete, add signal
	if planStatus.Complete {
		status.Signals = append(status.Signals, CompletionSignal{
			Type:    SignalPlanningFile,
			Source:  planPath,
			Message: "All phases in task_plan.md complete (" + itoa(planStatus.CompletedPhases) + "/" + itoa(planStatus.TotalPhases) + ")",
		})
	}

	// Add missing tasks from planning file (avoid duplicates)
	for _, task := range planStatus.MissingTasks {
		if !containsTask(status.MissingTasks, task) {
			status.MissingTasks = append(status.MissingTasks, "Planning: "+task)
		}
	}
}

// checkCommitMessages looks for completion signals in recent commits
func (c *CompletionChecker) checkCommitMessages(detail *GoalDetail, status *CompletionStatus) {
	if detail.Worktree == nil || detail.Worktree.Path == "" {
		return
	}

	worktreePath := filepath.Join(c.baseDir, detail.Worktree.Path)
	if !fileExists(worktreePath) {
		return
	}

	// Get recent commits on this branch
	cmd := exec.Command("git", "-C", worktreePath, "log", "--oneline", "-20", "--format=%H %s")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	// Patterns indicating completion
	goalIDPattern := regexp.QuoteMeta(detail.ID)
	completionPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)complete[sd]?\s+(goal|phase|implementation)`),
		regexp.MustCompile(`(?i)finish(ed|es)?\s+(goal|phase|implementation)`),
		regexp.MustCompile(`(?i)done\s+with\s+(goal|phase)`),
		regexp.MustCompile(`(?i)goal\s+#?` + goalIDPattern + `\s+(complete|done|finished)`),
		regexp.MustCompile(`(?i)(complete|done|finished)\s+goal\s+#?` + goalIDPattern),
		regexp.MustCompile(`(?i)archive\s+planning\s+files`),
		regexp.MustCompile(`(?i)final\s+(commit|changes|implementation)`),
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		commitHash := parts[0]
		if len(commitHash) > 7 {
			commitHash = commitHash[:7]
		}
		message := parts[1]

		for _, re := range completionPatterns {
			if re.MatchString(message) {
				status.Signals = append(status.Signals, CompletionSignal{
					Type:    SignalCommit,
					Source:  commitHash,
					Message: "Commit indicates completion: " + truncate(message, 60),
				})
				return // Only report first matching commit
			}
		}
	}
}

// calculateCompletion determines overall completion and confidence
func (c *CompletionChecker) calculateCompletion(status *CompletionStatus) {
	// Signal weights for confidence calculation
	signalWeight := map[CompletionSignalType]float64{
		SignalGoalPhases:   0.40,
		SignalAcceptance:   0.30,
		SignalPlanningFile: 0.20,
		SignalCommit:       0.10,
	}

	confidence := 0.0
	hasStrongSignal := false

	for _, signal := range status.Signals {
		confidence += signalWeight[signal.Type]
		if signal.Type == SignalGoalPhases || signal.Type == SignalAcceptance {
			hasStrongSignal = true
		}
	}

	// Reduce confidence for each missing item (capped)
	missingPenalty := float64(len(status.MissingTasks)) * 0.05
	if missingPenalty > 0.5 {
		missingPenalty = 0.5
	}
	confidence -= missingPenalty

	// Clamp confidence to [0, 1]
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	status.Confidence = confidence

	// Conservative completion logic:
	// Complete if: no missing items AND (strong signal OR high confidence)
	status.Complete = len(status.MissingTasks) == 0 && (hasStrongSignal || confidence >= 0.5)
}

// findGoalFile locates the goal markdown file
func (c *CompletionChecker) findGoalFile(goalID string) string {
	locations := []string{
		filepath.Join(c.baseDir, "goals", "active", goalID+".md"),
		filepath.Join(c.baseDir, "goals", "iced", goalID+".md"),
		filepath.Join(c.baseDir, "goals", "history", goalID+".md"),
	}

	for _, path := range locations {
		if fileExists(path) {
			return path
		}
	}

	return ""
}

// findTaskPlan locates the task_plan.md file for a goal
func (c *CompletionChecker) findTaskPlan(goalID string) string {
	locations := []string{
		filepath.Join(c.baseDir, "docs", "planning", "goal-"+goalID, "task_plan.md"),
		filepath.Join(c.baseDir, "docs", "planning", "history", "goal-"+goalID, "task_plan.md"),
		filepath.Join(c.baseDir, "goals", "active", goalID, "task_plan.md"),
		filepath.Join(c.baseDir, "goals", "history", goalID, "task_plan.md"),
	}

	for _, path := range locations {
		if fileExists(path) {
			return path
		}
	}

	return ""
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
		Signals:      []CompletionSignal{},
	}

	scanner := bufio.NewScanner(file)

	// Regex patterns
	phaseHeaderRe := regexp.MustCompile(`^#{2,3}\s+Phase\s+\d+:\s*(.+?)(?:\s+\[(complete|in_progress|pending)\])?$`)
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
			phaseComplete = true
			phaseHasTasks = false
			inPhase = true
			continue
		}

		// Check for task checkboxes
		if matches := taskRe.FindStringSubmatch(line); matches != nil {
			phaseHasTasks = true
			checkbox := strings.ToLower(matches[1])
			taskDesc := strings.TrimSpace(matches[2])

			if checkbox != "x" {
				phaseComplete = false
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

// Legacy functions for backwards compatibility

// IsGoalComplete checks if a goal is complete by parsing its task_plan.md
// Deprecated: Use CompletionChecker.CheckGoal for comprehensive checking
func IsGoalComplete(goalID string, baseDir string) (*CompletionStatus, error) {
	checker := NewCompletionChecker(baseDir)
	return checker.CheckGoal(goalID)
}

// Helper functions

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func containsTask(tasks []string, task string) bool {
	for _, t := range tasks {
		if strings.Contains(t, task) {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
