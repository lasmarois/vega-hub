package hub

import (
	"fmt"
	"os/exec"
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

	// Channels for SSE broadcasting
	subscribers map[chan Event]bool
	subMu       sync.RWMutex

	mdWriter *markdown.Writer
}

// Executor represents an active executor session
type Executor struct {
	SessionID string    `json:"session_id"`
	GoalID    int       `json:"goal_id"`
	CWD       string    `json:"cwd"`
	StartedAt time.Time `json:"started_at"`
}

// Question represents a pending question from an executor
type Question struct {
	ID        string    `json:"id"`
	GoalID    int       `json:"goal_id"`
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
func (h *Hub) RegisterExecutor(goalID int, sessionID, cwd string) string {
	h.mu.Lock()
	h.executors[sessionID] = &Executor{
		SessionID: sessionID,
		GoalID:    goalID,
		CWD:       cwd,
		StartedAt: time.Now(),
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
		},
	})

	// Build context for the executor
	context := h.buildExecutorContext(goalID, sessionID, cwd)
	return context
}

// StopExecutor marks an executor session as stopped
func (h *Hub) StopExecutor(goalID int, sessionID, reason string) {
	h.mu.Lock()
	delete(h.executors, sessionID)
	h.mu.Unlock()

	// Write to markdown
	if err := h.mdWriter.WriteExecutorEvent(goalID, sessionID, "Stopped", reason); err != nil {
		// Log error but don't fail
	}

	// Broadcast executor stopped event
	h.broadcast(Event{
		Type: "executor_stopped",
		Data: map[string]interface{}{
			"session_id": sessionID,
			"goal_id":    goalID,
			"reason":     reason,
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
func (h *Hub) buildExecutorContext(goalID int, sessionID, cwd string) string {
	return "[EXECUTOR SESSION START]\n" +
		"Working on Goal #" + itoa(goalID) + "\n" +
		"Directory: " + cwd + "\n" +
		"vega-hub: connected\n\n" +
		"IMPORTANT REMINDERS:\n" +
		"1. Load 'planning-with-files' skill if not already loaded\n" +
		"2. Planning files go at worktree root: task_plan.md, findings.md, progress.md\n" +
		"3. You can use AskUserQuestion to ask the human questions directly (via vega-hub)\n" +
		"4. Before completing, you MUST:\n" +
		"   - Archive planning files to docs/planning/history/goal-" + itoa(goalID) + "/\n" +
		"   - Commit the archive\n" +
		"   - Report to manager for approval\n" +
		"5. Commit messages must include 'Goal: #" + itoa(goalID) + "'"
}

// sendDesktopNotification sends a desktop notification (Linux/macOS)
func (h *Hub) sendDesktopNotification(goalID int, reason string) {
	title := "Executor Stopped"
	message := "Goal #" + itoa(goalID)
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
