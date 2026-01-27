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
	SignalGoalPhases CompletionSignalType = "goal_phases"
	SignalAcceptance CompletionSignalType = "acceptance"
	SignalCommit     CompletionSignalType = "commit"
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

	// Check 1: Goal file phases (## Phases section with checkboxes)
	c.checkGoalPhases(detail, status)

	// Check 2: Acceptance criteria in goal file (## Acceptance Criteria section)
	c.checkAcceptanceCriteria(goalID, status)

	// Check 3: Commit messages
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
// Reads from ## Phases section with - [x] and - [ ] checkboxes
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
// Reads from ## Acceptance Criteria section with - [x] and - [ ] checkboxes
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
	// Weights adjusted since we now only read from goal file (phases + acceptance)
	signalWeight := map[CompletionSignalType]float64{
		SignalGoalPhases: 0.50,
		SignalAcceptance: 0.40,
		SignalCommit:     0.10,
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
// Checks both flat structure (goals/active/<id>.md) and folder structure (goals/active/<id>/<id>.md)
func (c *CompletionChecker) findGoalFile(goalID string) string {
	// Check all goal directories with both flat and folder structures
	dirs := []string{"active", "iced", "history"}

	for _, dir := range dirs {
		// Flat structure: goals/<dir>/<id>.md
		flatPath := filepath.Join(c.baseDir, "goals", dir, goalID+".md")
		if fileExists(flatPath) {
			return flatPath
		}

		// Folder structure: goals/<dir>/<id>/<id>.md
		folderPath := filepath.Join(c.baseDir, "goals", dir, goalID, goalID+".md")
		if fileExists(folderPath) {
			return folderPath
		}
	}

	return ""
}

// Legacy functions for backwards compatibility

// IsGoalComplete checks if a goal is complete by parsing its goal file
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
