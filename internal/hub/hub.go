package hub

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/lasmarois/vega-hub/internal/markdown"
)

// Hub manages the state of pending questions and executor sessions
type Hub struct {
	dir       string
	questions map[string]*Question
	executors map[string]*Executor
	mu        sync.RWMutex

	// Spawn lock - prevents concurrent spawns for same goal
	spawnMu sync.Mutex

	// Channels for SSE broadcasting
	subscribers map[chan Event]bool
	subMu       sync.RWMutex

	mdWriter *markdown.Writer
}

// Executor represents an active executor session
type Executor struct {
	SessionID string    `json:"session_id"`
	GoalID    string    `json:"goal_id"`
	CWD       string    `json:"cwd"`
	StartedAt time.Time `json:"started_at"`
	LogFile   string    `json:"log_file,omitempty"`
	User      string    `json:"user,omitempty"` // Username who spawned this executor
}

// Question represents a pending question from an executor
type Question struct {
	ID        string    `json:"id"`
	GoalID    string    `json:"goal_id"`
	SessionID string    `json:"session_id"`
	Question  string    `json:"question"`
	Options   []Option  `json:"options,omitempty"`
	CreatedAt time.Time `json:"created_at"`

	// Answer channel - blocks until answered
	answerCh chan string
}

// Option represents a choice for the question
type Option struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// Event represents an SSE event
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// New creates a new Hub instance
func New(dir string) *Hub {
	return &Hub{
		dir:         dir,
		questions:   make(map[string]*Question),
		executors:   make(map[string]*Executor),
		subscribers: make(map[chan Event]bool),
		mdWriter:    markdown.NewWriter(dir),
	}
}

// RegisterExecutor registers a new executor session and returns context
// The user parameter tracks who spawned this executor
func (h *Hub) RegisterExecutor(goalID string, sessionID, cwd, user string) string {
	logFile := filepath.Join(cwd, ".executor-output.log")
	h.mu.Lock()
	h.executors[sessionID] = &Executor{
		SessionID: sessionID,
		GoalID:    goalID,
		CWD:       cwd,
		StartedAt: time.Now(),
		LogFile:   logFile,
		User:      user,
	}
	h.mu.Unlock()

	// Write to markdown
	if err := h.mdWriter.WriteExecutorEvent(goalID, sessionID, "Started", ""); err != nil {
		// Log error but don't fail
		// TODO: proper logging
	}

	// Broadcast executor started event
	h.broadcast(Event{
		Type: "executor_started",
		Data: map[string]interface{}{
			"session_id": sessionID,
			"goal_id":    goalID,
			"cwd":        cwd,
			"user":       user,
		},
	})

	// Build context for the executor
	context := h.buildExecutorContext(goalID, sessionID, cwd)
	return context
}

// StopExecutor marks an executor session as stopped
func (h *Hub) StopExecutor(goalID string, sessionID, reason string) {
	// Get executor info before removing (for log file path)
	h.mu.Lock()
	executor := h.executors[sessionID]
	delete(h.executors, sessionID)
	h.mu.Unlock()

	// Read output summary from log file
	var outputSummary string
	if executor != nil && executor.LogFile != "" {
		outputSummary = h.readLastLines(executor.LogFile, 50)
	}

	// Write to markdown
	if err := h.mdWriter.WriteExecutorEvent(goalID, sessionID, "Stopped", reason); err != nil {
		// Log error but don't fail
	}

	// Broadcast executor stopped event with output
	h.broadcast(Event{
		Type: "executor_stopped",
		Data: map[string]interface{}{
			"session_id": sessionID,
			"goal_id":    goalID,
			"reason":     reason,
			"output":     outputSummary,
		},
	})

	// Send desktop notification
	h.sendDesktopNotification(goalID, reason)
}

// GetActiveExecutors returns all active executor sessions
func (h *Hub) GetActiveExecutors() []*Executor {
	h.mu.RLock()
	defer h.mu.RUnlock()

	executors := make([]*Executor, 0, len(h.executors))
	for _, e := range h.executors {
		executors = append(executors, e)
	}
	return executors
}

// buildExecutorContext builds the context string for an executor
func (h *Hub) buildExecutorContext(goalID string, sessionID, cwd string) string {
	return "[EXECUTOR SESSION START]\n" +
		"Working on Goal #" + goalID + "\n" +
		"Directory: " + cwd + "\n" +
		"vega-hub: connected\n\n" +
		"IMPORTANT REMINDERS:\n" +
		"1. Load 'planning-with-files' skill if not already loaded\n" +
		"2. Planning files go at worktree root: task_plan.md, findings.md, progress.md\n" +
		"3. You can use AskUserQuestion to ask the human questions directly (via vega-hub)\n" +
		"4. Before completing, you MUST:\n" +
		"   - Archive planning files to docs/planning/history/goal-" + goalID + "/\n" +
		"   - Commit the archive\n" +
		"   - Report to manager for approval\n" +
		"5. Commit messages must include 'Goal: #" + goalID + "'"
}

// sendDesktopNotification sends a desktop notification (Linux/macOS)
func (h *Hub) sendDesktopNotification(goalID string, reason string) {
	title := "Executor Stopped"
	message := "Goal #" + goalID
	if reason != "" {
		message += " - " + reason
	}

	// Try Linux first
	if _, err := execCommand("notify-send", title, message); err == nil {
		return
	}

	// Try macOS
	script := `display notification "` + message + `" with title "` + title + `"`
	execCommand("osascript", "-e", script)
}

// itoa is a simple int to string helper
func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}

// Dir returns the vega-missile directory
func (h *Hub) Dir() string {
	return h.dir
}

// execCommand runs a command and returns any error
func execCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}

// Ask registers a new question and blocks until answered
func (h *Hub) Ask(q *Question) string {
	q.answerCh = make(chan string, 1)
	q.CreatedAt = time.Now()

	h.mu.Lock()
	h.questions[q.ID] = q
	h.mu.Unlock()

	// Broadcast new question event
	h.broadcast(Event{
		Type: "question",
		Data: q,
	})

	// Block until answer received
	answer := <-q.answerCh

	// Clean up
	h.mu.Lock()
	delete(h.questions, q.ID)
	h.mu.Unlock()

	return answer
}

// Answer provides an answer to a pending question
func (h *Hub) Answer(id string, answer string) bool {
	h.mu.RLock()
	q, exists := h.questions[id]
	h.mu.RUnlock()

	if !exists {
		return false
	}

	// Write to markdown
	if err := h.mdWriter.WriteQA(q.GoalID, q.SessionID, q.Question, answer); err != nil {
		// Log error but don't fail
		// TODO: proper logging
	}

	// Send answer to waiting goroutine
	q.answerCh <- answer

	// Broadcast answered event
	h.broadcast(Event{
		Type: "answered",
		Data: map[string]interface{}{
			"id":     id,
			"answer": answer,
		},
	})

	return true
}

// GetPendingQuestions returns all pending questions
func (h *Hub) GetPendingQuestions() []*Question {
	h.mu.RLock()
	defer h.mu.RUnlock()

	questions := make([]*Question, 0, len(h.questions))
	for _, q := range h.questions {
		questions = append(questions, q)
	}
	return questions
}

// Subscribe returns a channel for receiving events
func (h *Hub) Subscribe() chan Event {
	ch := make(chan Event, 10)
	h.subMu.Lock()
	h.subscribers[ch] = true
	h.subMu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber
func (h *Hub) Unsubscribe(ch chan Event) {
	h.subMu.Lock()
	delete(h.subscribers, ch)
	h.subMu.Unlock()
	close(ch)
}

// broadcast sends an event to all subscribers
func (h *Hub) broadcast(event Event) {
	h.subMu.RLock()
	defer h.subMu.RUnlock()

	for ch := range h.subscribers {
		select {
		case ch <- event:
		default:
			// Skip slow subscribers
		}
	}
}

// readLastLines reads the last N lines from a file
func (h *Hub) readLastLines(filePath string, n int) string {
	file, err := os.Open(filePath)
	if err != nil {
		return ""
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}

	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}

// GetExecutorOutput returns the full output for a goal's executor
func (h *Hub) GetExecutorOutput(goalID string) (string, error) {
	worktree, err := h.findWorktree(goalID)
	if err != nil {
		return "", err
	}

	logFile := filepath.Join(worktree, ".executor-output.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// GetExecutorOutputTail returns the last N lines of output for a goal's executor
func (h *Hub) GetExecutorOutputTail(goalID string, lines int) (string, error) {
	worktree, err := h.findWorktree(goalID)
	if err != nil {
		return "", err
	}

	logFile := filepath.Join(worktree, ".executor-output.log")
	return h.readLastLines(logFile, lines), nil
}
