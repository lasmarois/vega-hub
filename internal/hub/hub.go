package hub

import (
	"sync"
	"time"

	"github.com/lasmarois/vega-hub/internal/markdown"
)

// Hub manages the state of pending questions and executor sessions
type Hub struct {
	dir       string
	questions map[string]*Question
	mu        sync.RWMutex

	// Channels for SSE broadcasting
	subscribers map[chan Event]bool
	subMu       sync.RWMutex

	mdWriter *markdown.Writer
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
		subscribers: make(map[chan Event]bool),
		mdWriter:    markdown.NewWriter(dir),
	}
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
