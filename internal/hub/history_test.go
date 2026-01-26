package hub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionHistory_RecordSessionStart(t *testing.T) {
	dir := t.TempDir()
	h := NewSessionHistory(dir)

	err := h.RecordSessionStart("goal-123", "session-abc", "/path/to/worktree", "testuser")
	if err != nil {
		t.Fatalf("RecordSessionStart failed: %v", err)
	}

	// Check in-memory cache
	sessions, err := h.GetGoalSessions("goal-123")
	if err != nil {
		t.Fatalf("GetGoalSessions failed: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.SessionID != "session-abc" {
		t.Errorf("expected session_id 'session-abc', got '%s'", s.SessionID)
	}
	if s.GoalID != "goal-123" {
		t.Errorf("expected goal_id 'goal-123', got '%s'", s.GoalID)
	}
	if s.User != "testuser" {
		t.Errorf("expected user 'testuser', got '%s'", s.User)
	}
	if s.CWD != "/path/to/worktree" {
		t.Errorf("expected cwd '/path/to/worktree', got '%s'", s.CWD)
	}

	// Check file was created
	historyFile := filepath.Join(dir, ".vega-hub-history", "goal-goal-123.jsonl")
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		t.Errorf("history file not created at %s", historyFile)
	}
}

func TestSessionHistory_RecordSessionStop(t *testing.T) {
	dir := t.TempDir()
	h := NewSessionHistory(dir)

	// Start a session first
	err := h.RecordSessionStart("goal-123", "session-abc", "/path/to/worktree", "testuser")
	if err != nil {
		t.Fatalf("RecordSessionStart failed: %v", err)
	}

	// Stop the session with Claude info
	err = h.RecordSessionStop("goal-123", "session-abc", "claude-xyz", "/path/to/transcript.jsonl", "completed")
	if err != nil {
		t.Fatalf("RecordSessionStop failed: %v", err)
	}

	// Check in-memory cache updated
	sessions, err := h.GetGoalSessions("goal-123")
	if err != nil {
		t.Fatalf("GetGoalSessions failed: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.ClaudeSessionID != "claude-xyz" {
		t.Errorf("expected claude_session_id 'claude-xyz', got '%s'", s.ClaudeSessionID)
	}
	if s.TranscriptPath != "/path/to/transcript.jsonl" {
		t.Errorf("expected transcript_path '/path/to/transcript.jsonl', got '%s'", s.TranscriptPath)
	}
	if s.StopReason != "completed" {
		t.Errorf("expected stop_reason 'completed', got '%s'", s.StopReason)
	}
	if s.StoppedAt == nil {
		t.Error("expected StoppedAt to be set")
	}
}

func TestSessionHistory_RecordQuestion(t *testing.T) {
	dir := t.TempDir()
	h := NewSessionHistory(dir)

	err := h.RecordQuestion("goal-123", "session-abc", "What is the answer?", "42")
	if err != nil {
		t.Fatalf("RecordQuestion failed: %v", err)
	}

	// Check file contains the question
	entries, err := h.GetGoalHistory("goal-123", 0)
	if err != nil {
		t.Fatalf("GetGoalHistory failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Type != "question" {
		t.Errorf("expected type 'question', got '%s'", e.Type)
	}
	if e.Question != "What is the answer?" {
		t.Errorf("expected question 'What is the answer?', got '%s'", e.Question)
	}
	if e.Answer != "42" {
		t.Errorf("expected answer '42', got '%s'", e.Answer)
	}
}

func TestSessionHistory_GetGoalHistory_WithLimit(t *testing.T) {
	dir := t.TempDir()
	h := NewSessionHistory(dir)

	// Record multiple entries
	for i := 0; i < 10; i++ {
		err := h.RecordActivity("goal-123", "session-abc", "test_activity", map[string]int{"index": i})
		if err != nil {
			t.Fatalf("RecordActivity failed: %v", err)
		}
	}

	// Get with limit
	entries, err := h.GetGoalHistory("goal-123", 3)
	if err != nil {
		t.Fatalf("GetGoalHistory failed: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	// Should be the last 3 entries
	for i, e := range entries {
		data, ok := e.Data.(map[string]interface{})
		if !ok {
			t.Errorf("entry %d: expected map data, got %T", i, e.Data)
			continue
		}
		// JSON unmarshals numbers as float64
		idx := int(data["index"].(float64))
		expectedIdx := 7 + i // entries 7, 8, 9
		if idx != expectedIdx {
			t.Errorf("entry %d: expected index %d, got %d", i, expectedIdx, idx)
		}
	}
}

func TestSessionHistory_GetSessionHistory(t *testing.T) {
	dir := t.TempDir()
	h := NewSessionHistory(dir)

	// Record entries for two sessions
	h.RecordSessionStart("goal-123", "session-a", "/path/a", "user1")
	h.RecordSessionStart("goal-123", "session-b", "/path/b", "user2")
	h.RecordQuestion("goal-123", "session-a", "Q1", "A1")
	h.RecordQuestion("goal-123", "session-b", "Q2", "A2")
	h.RecordQuestion("goal-123", "session-a", "Q3", "A3")

	// Get session-a history only
	entries, err := h.GetSessionHistory("goal-123", "session-a")
	if err != nil {
		t.Fatalf("GetSessionHistory failed: %v", err)
	}

	if len(entries) != 3 { // start + 2 questions
		t.Errorf("expected 3 entries for session-a, got %d", len(entries))
	}

	for _, e := range entries {
		if e.SessionID != "session-a" {
			t.Errorf("expected session_id 'session-a', got '%s'", e.SessionID)
		}
	}
}

func TestSessionHistory_LoadFromFile(t *testing.T) {
	dir := t.TempDir()

	// Create history with first instance
	h1 := NewSessionHistory(dir)
	h1.RecordSessionStart("goal-123", "session-abc", "/path/to/worktree", "testuser")
	h1.RecordSessionStop("goal-123", "session-abc", "claude-xyz", "/transcript.jsonl", "completed")

	// Create new instance (simulates restart) and load from file
	h2 := NewSessionHistory(dir)

	// Clear in-memory cache to force file load
	h2.sessions = make(map[string][]*ExecutorSession)

	sessions, err := h2.GetGoalSessions("goal-123")
	if err != nil {
		t.Fatalf("GetGoalSessions failed: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session loaded from file, got %d", len(sessions))
	}

	s := sessions[0]
	if s.SessionID != "session-abc" {
		t.Errorf("expected session_id 'session-abc', got '%s'", s.SessionID)
	}
	if s.ClaudeSessionID != "claude-xyz" {
		t.Errorf("expected claude_session_id 'claude-xyz', got '%s'", s.ClaudeSessionID)
	}
	if s.StoppedAt == nil {
		t.Error("expected StoppedAt to be set after loading from file")
	}
}

func TestSessionHistory_NonExistentGoal(t *testing.T) {
	dir := t.TempDir()
	h := NewSessionHistory(dir)

	// Get sessions for non-existent goal
	sessions, err := h.GetGoalSessions("non-existent")
	if err != nil {
		t.Fatalf("GetGoalSessions failed: %v", err)
	}

	if sessions != nil && len(sessions) != 0 {
		t.Errorf("expected nil or empty sessions, got %d", len(sessions))
	}

	// Get history for non-existent goal
	entries, err := h.GetGoalHistory("non-existent", 0)
	if err != nil {
		t.Fatalf("GetGoalHistory failed: %v", err)
	}

	if entries != nil && len(entries) != 0 {
		t.Errorf("expected nil or empty entries, got %d", len(entries))
	}
}

func TestSessionHistory_UpdateClaudeSession(t *testing.T) {
	dir := t.TempDir()
	h := NewSessionHistory(dir)

	// Start a session
	h.RecordSessionStart("goal-123", "session-abc", "/path", "user")

	// Update Claude session info
	h.UpdateClaudeSession("goal-123", "session-abc", "claude-new", "/new/transcript.jsonl")

	sessions, _ := h.GetGoalSessions("goal-123")
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.ClaudeSessionID != "claude-new" {
		t.Errorf("expected claude_session_id 'claude-new', got '%s'", s.ClaudeSessionID)
	}
	if s.TranscriptPath != "/new/transcript.jsonl" {
		t.Errorf("expected transcript_path '/new/transcript.jsonl', got '%s'", s.TranscriptPath)
	}
}

func TestHistoryEntry_JSONFormat(t *testing.T) {
	entry := HistoryEntry{
		Timestamp:       time.Now(),
		GoalID:          "goal-123",
		SessionID:       "session-abc",
		ClaudeSessionID: "claude-xyz",
		TranscriptPath:  "/path/to/transcript.jsonl",
		Type:            "session_stop",
		StopReason:      "completed",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal entry: %v", err)
	}

	var decoded HistoryEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal entry: %v", err)
	}

	if decoded.GoalID != entry.GoalID {
		t.Errorf("expected goal_id '%s', got '%s'", entry.GoalID, decoded.GoalID)
	}
	if decoded.ClaudeSessionID != entry.ClaudeSessionID {
		t.Errorf("expected claude_session_id '%s', got '%s'", entry.ClaudeSessionID, decoded.ClaudeSessionID)
	}
}
