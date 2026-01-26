package hub

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// SessionHistory stores historical executor sessions for a goal
type SessionHistory struct {
	mu       sync.RWMutex
	dir      string                       // vega-missile directory
	sessions map[string][]*ExecutorSession // goal_id -> sessions
}

// ExecutorSession represents a completed or active executor session
type ExecutorSession struct {
	SessionID       string     `json:"session_id"`        // vega-hub generated ID
	ClaudeSessionID string     `json:"claude_session_id"` // Claude Code's session ID
	TranscriptPath  string     `json:"transcript_path"`   // Path to Claude's conversation JSONL
	GoalID          string     `json:"goal_id"`
	CWD             string     `json:"cwd"`
	User            string     `json:"user,omitempty"`
	StartedAt       time.Time  `json:"started_at"`
	StoppedAt       *time.Time `json:"stopped_at,omitempty"`
	StopReason      string     `json:"stop_reason,omitempty"`
	Activities      []Activity `json:"activities,omitempty"` // In-memory only, not persisted per-session
}

// Activity represents an executor activity event
type Activity struct {
	Timestamp time.Time   `json:"timestamp"`
	Type      string      `json:"type"` // "started", "stopped", "question", "answer", "tool_use", etc.
	Data      interface{} `json:"data,omitempty"`
}

// HistoryEntry is a single line in the history JSONL file
type HistoryEntry struct {
	Timestamp       time.Time   `json:"timestamp"`
	GoalID          string      `json:"goal_id"`
	SessionID       string      `json:"session_id"`
	ClaudeSessionID string      `json:"claude_session_id,omitempty"`
	TranscriptPath  string      `json:"transcript_path,omitempty"`
	Type            string      `json:"type"` // "session_start", "session_stop", "question", "answer", "activity"
	User            string      `json:"user,omitempty"`
	CWD             string      `json:"cwd,omitempty"`
	StopReason      string      `json:"stop_reason,omitempty"`
	Question        string      `json:"question,omitempty"`
	Answer          string      `json:"answer,omitempty"`
	Data            interface{} `json:"data,omitempty"`
}

// NewSessionHistory creates a new session history manager
func NewSessionHistory(dir string) *SessionHistory {
	return &SessionHistory{
		dir:      dir,
		sessions: make(map[string][]*ExecutorSession),
	}
}

// historyDir returns the directory for history files
func (h *SessionHistory) historyDir() string {
	return filepath.Join(h.dir, ".vega-hub-history")
}

// historyFile returns the history file path for a goal
func (h *SessionHistory) historyFile(goalID string) string {
	return filepath.Join(h.historyDir(), fmt.Sprintf("goal-%s.jsonl", goalID))
}

// ensureHistoryDir creates the history directory if it doesn't exist
func (h *SessionHistory) ensureHistoryDir() error {
	dir := h.historyDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create history dir: %w", err)
	}
	return nil
}

// appendEntry appends an entry to the goal's history file
func (h *SessionHistory) appendEntry(entry HistoryEntry) error {
	if err := h.ensureHistoryDir(); err != nil {
		return err
	}

	file, err := os.OpenFile(h.historyFile(entry.GoalID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}

	return nil
}

// RecordSessionStart records a new session starting
func (h *SessionHistory) RecordSessionStart(goalID, sessionID, cwd, user string) error {
	entry := HistoryEntry{
		Timestamp: time.Now(),
		GoalID:    goalID,
		SessionID: sessionID,
		Type:      "session_start",
		User:      user,
		CWD:       cwd,
	}

	// Add to in-memory cache
	h.mu.Lock()
	session := &ExecutorSession{
		SessionID: sessionID,
		GoalID:    goalID,
		CWD:       cwd,
		User:      user,
		StartedAt: entry.Timestamp,
	}
	h.sessions[goalID] = append(h.sessions[goalID], session)
	h.mu.Unlock()

	return h.appendEntry(entry)
}

// RecordSessionStop records a session stopping
func (h *SessionHistory) RecordSessionStop(goalID, sessionID, claudeSessionID, transcriptPath, reason, output string) error {
	now := time.Now()
	entry := HistoryEntry{
		Timestamp:       now,
		GoalID:          goalID,
		SessionID:       sessionID,
		ClaudeSessionID: claudeSessionID,
		TranscriptPath:  transcriptPath,
		Type:            "session_stop",
		StopReason:      reason,
	}
	// Store output in Data field if present
	if output != "" {
		entry.Data = map[string]interface{}{
			"output": output,
		}
	}

	// Update in-memory cache
	h.mu.Lock()
	for _, s := range h.sessions[goalID] {
		if s.SessionID == sessionID {
			s.StoppedAt = &now
			s.StopReason = reason
			s.ClaudeSessionID = claudeSessionID
			s.TranscriptPath = transcriptPath
			break
		}
	}
	h.mu.Unlock()

	return h.appendEntry(entry)
}

// RecordQuestion records a Q&A exchange
func (h *SessionHistory) RecordQuestion(goalID, sessionID, question, answer string) error {
	entry := HistoryEntry{
		Timestamp: time.Now(),
		GoalID:    goalID,
		SessionID: sessionID,
		Type:      "question",
		Question:  question,
		Answer:    answer,
	}
	return h.appendEntry(entry)
}

// RecordActivity records a generic activity
func (h *SessionHistory) RecordActivity(goalID, sessionID, activityType string, data interface{}) error {
	entry := HistoryEntry{
		Timestamp: time.Now(),
		GoalID:    goalID,
		SessionID: sessionID,
		Type:      activityType,
		Data:      data,
	}
	return h.appendEntry(entry)
}

// UpdateClaudeSession updates Claude's session info for an active session
func (h *SessionHistory) UpdateClaudeSession(goalID, sessionID, claudeSessionID, transcriptPath string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, s := range h.sessions[goalID] {
		if s.SessionID == sessionID {
			s.ClaudeSessionID = claudeSessionID
			s.TranscriptPath = transcriptPath
			break
		}
	}
}

// GetGoalSessions returns all sessions for a goal (loads from file if not in memory)
func (h *SessionHistory) GetGoalSessions(goalID string) ([]*ExecutorSession, error) {
	// Check in-memory cache first
	h.mu.RLock()
	if sessions, ok := h.sessions[goalID]; ok && len(sessions) > 0 {
		h.mu.RUnlock()
		return sessions, nil
	}
	h.mu.RUnlock()

	// Load from file
	return h.loadGoalHistory(goalID)
}

// loadGoalHistory loads session history from file
func (h *SessionHistory) loadGoalHistory(goalID string) ([]*ExecutorSession, error) {
	file, err := os.Open(h.historyFile(goalID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()

	sessionMap := make(map[string]*ExecutorSession)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var entry HistoryEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue // Skip malformed lines
		}

		switch entry.Type {
		case "session_start":
			sessionMap[entry.SessionID] = &ExecutorSession{
				SessionID: entry.SessionID,
				GoalID:    entry.GoalID,
				CWD:       entry.CWD,
				User:      entry.User,
				StartedAt: entry.Timestamp,
			}
		case "session_stop":
			if s, ok := sessionMap[entry.SessionID]; ok {
				s.StoppedAt = &entry.Timestamp
				s.StopReason = entry.StopReason
				s.ClaudeSessionID = entry.ClaudeSessionID
				s.TranscriptPath = entry.TranscriptPath
			}
		}
	}

	// Convert map to slice and sort by start time
	sessions := make([]*ExecutorSession, 0, len(sessionMap))
	for _, s := range sessionMap {
		sessions = append(sessions, s)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartedAt.Before(sessions[j].StartedAt)
	})

	// Update cache
	h.mu.Lock()
	h.sessions[goalID] = sessions
	h.mu.Unlock()

	return sessions, nil
}

// GetGoalHistory returns all history entries for a goal (for detailed view)
func (h *SessionHistory) GetGoalHistory(goalID string, limit int) ([]HistoryEntry, error) {
	file, err := os.Open(h.historyFile(goalID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var entry HistoryEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}

	// Return last N entries if limit specified
	if limit > 0 && len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	return entries, nil
}

// GetSessionHistory returns history for a specific session
func (h *SessionHistory) GetSessionHistory(goalID, sessionID string) ([]HistoryEntry, error) {
	file, err := os.Open(h.historyFile(goalID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		var entry HistoryEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if entry.SessionID == sessionID {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}
