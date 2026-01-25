package hub

import (
	"os"
	"sync"
	"testing"
	"time"
)

func setupTestHub(t *testing.T) *Hub {
	t.Helper()
	dir := t.TempDir()
	return New(dir)
}

func TestNew(t *testing.T) {
	h := setupTestHub(t)

	if h.dir == "" {
		t.Error("expected dir to be set")
	}
	if h.questions == nil {
		t.Error("expected questions map to be initialized")
	}
	if h.executors == nil {
		t.Error("expected executors map to be initialized")
	}
	if h.subscribers == nil {
		t.Error("expected subscribers map to be initialized")
	}
}

func TestRegisterExecutor(t *testing.T) {
	h := setupTestHub(t)

	goalID := "abc1234"
	sessionID := "session-001"
	cwd := "/path/to/worktree"

	context := h.RegisterExecutor(goalID, sessionID, cwd, "testuser")

	// Check context contains expected info
	if context == "" {
		t.Error("expected non-empty context")
	}

	// Check executor was registered
	executors := h.GetActiveExecutors()
	if len(executors) != 1 {
		t.Fatalf("expected 1 executor, got %d", len(executors))
	}

	e := executors[0]
	if e.SessionID != sessionID {
		t.Errorf("expected session ID %s, got %s", sessionID, e.SessionID)
	}
	if e.GoalID != goalID {
		t.Errorf("expected goal ID %s, got %s", goalID, e.GoalID)
	}
	if e.CWD != cwd {
		t.Errorf("expected CWD %s, got %s", cwd, e.CWD)
	}
}

func TestStopExecutor(t *testing.T) {
	h := setupTestHub(t)

	goalID := "abc1234"
	sessionID := "session-001"
	cwd := "/path/to/worktree"

	h.RegisterExecutor(goalID, sessionID, cwd, "testuser")

	// Verify registered
	if len(h.GetActiveExecutors()) != 1 {
		t.Fatal("executor not registered")
	}

	// Stop the executor
	h.StopExecutor(goalID, sessionID, "completed")

	// Verify removed
	if len(h.GetActiveExecutors()) != 0 {
		t.Error("executor should be removed after stop")
	}
}

func TestAskAndAnswer(t *testing.T) {
	h := setupTestHub(t)

	q := &Question{
		ID:        "q-001",
		GoalID:    "abc1234",
		SessionID: "session-001",
		Question:  "What color?",
		Options: []Option{
			{Label: "Red", Description: "The color red"},
			{Label: "Blue", Description: "The color blue"},
		},
	}

	// Start ask in goroutine (it blocks)
	var answer string
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		answer = h.Ask(q)
	}()

	// Wait a bit for the question to be registered
	time.Sleep(50 * time.Millisecond)

	// Check question is pending
	pending := h.GetPendingQuestions()
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending question, got %d", len(pending))
	}
	if pending[0].ID != "q-001" {
		t.Errorf("expected question ID 'q-001', got '%s'", pending[0].ID)
	}

	// Answer the question
	ok := h.Answer("q-001", "Blue")
	if !ok {
		t.Error("Answer should return true")
	}

	// Wait for Ask to complete
	wg.Wait()

	if answer != "Blue" {
		t.Errorf("expected answer 'Blue', got '%s'", answer)
	}

	// Question should be removed
	if len(h.GetPendingQuestions()) != 0 {
		t.Error("question should be removed after answer")
	}
}

func TestAnswer_NonexistentQuestion(t *testing.T) {
	h := setupTestHub(t)

	ok := h.Answer("nonexistent", "test")
	if ok {
		t.Error("Answer should return false for nonexistent question")
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	h := setupTestHub(t)

	// Subscribe
	ch := h.Subscribe()
	if ch == nil {
		t.Fatal("Subscribe should return a channel")
	}

	// Check subscriber count
	h.subMu.RLock()
	count := len(h.subscribers)
	h.subMu.RUnlock()
	if count != 1 {
		t.Errorf("expected 1 subscriber, got %d", count)
	}

	// Unsubscribe
	h.Unsubscribe(ch)

	h.subMu.RLock()
	count = len(h.subscribers)
	h.subMu.RUnlock()
	if count != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", count)
	}
}

func TestBroadcast(t *testing.T) {
	h := setupTestHub(t)

	ch := h.Subscribe()

	// Broadcast an event
	event := Event{
		Type: "test_event",
		Data: map[string]string{"key": "value"},
	}
	h.broadcast(event)

	// Should receive the event
	select {
	case received := <-ch:
		if received.Type != "test_event" {
			t.Errorf("expected event type 'test_event', got '%s'", received.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected to receive broadcast event")
	}

	h.Unsubscribe(ch)
}

func TestBroadcast_MultipleSubscribers(t *testing.T) {
	h := setupTestHub(t)

	ch1 := h.Subscribe()
	ch2 := h.Subscribe()

	event := Event{Type: "multi_test", Data: nil}
	h.broadcast(event)

	// Both should receive
	for i, ch := range []chan Event{ch1, ch2} {
		select {
		case received := <-ch:
			if received.Type != "multi_test" {
				t.Errorf("subscriber %d: wrong event type", i)
			}
		case <-time.After(100 * time.Millisecond):
			t.Errorf("subscriber %d: did not receive event", i)
		}
	}

	h.Unsubscribe(ch1)
	h.Unsubscribe(ch2)
}

func TestConcurrentExecutorRegistration(t *testing.T) {
	h := setupTestHub(t)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			goalID := "goal-" + itoa(i)
			sessionID := "session-" + itoa(i)
			h.RegisterExecutor(goalID, sessionID, "/path/"+itoa(i), "testuser")
		}(i)
	}
	wg.Wait()

	executors := h.GetActiveExecutors()
	if len(executors) != 10 {
		t.Errorf("expected 10 executors, got %d", len(executors))
	}
}

func TestConcurrentQuestions(t *testing.T) {
	h := setupTestHub(t)

	var wg sync.WaitGroup
	answers := make([]string, 5)

	// Start multiple questions
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			q := &Question{
				ID:        "q-" + itoa(i),
				GoalID:    "goal-" + itoa(i),
				SessionID: "session-" + itoa(i),
				Question:  "Question " + itoa(i),
			}
			answers[i] = h.Ask(q)
		}(i)
	}

	// Wait for questions to be registered
	time.Sleep(50 * time.Millisecond)

	// Answer all questions
	for i := 0; i < 5; i++ {
		h.Answer("q-"+itoa(i), "Answer "+itoa(i))
	}

	wg.Wait()

	// Verify all answers
	for i := 0; i < 5; i++ {
		expected := "Answer " + itoa(i)
		if answers[i] != expected {
			t.Errorf("answer[%d]: expected '%s', got '%s'", i, expected, answers[i])
		}
	}
}

func TestReadLastLines(t *testing.T) {
	h := setupTestHub(t)

	// Create a test file
	content := "line 1\nline 2\nline 3\nline 4\nline 5\n"
	filePath := h.dir + "/test.log"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Read last 3 lines
	result := h.readLastLines(filePath, 3)
	expected := "line 3\nline 4\nline 5"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	// Read more lines than exist
	result = h.readLastLines(filePath, 10)
	expected = "line 1\nline 2\nline 3\nline 4\nline 5"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestReadLastLines_FileNotFound(t *testing.T) {
	h := setupTestHub(t)
	result := h.readLastLines("/nonexistent/file.log", 10)
	if result != "" {
		t.Errorf("expected empty string for nonexistent file, got %q", result)
	}
}
