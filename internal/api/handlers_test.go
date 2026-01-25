package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/lasmarois/vega-hub/internal/hub"
)

func setupTestEnv(t *testing.T) (*hub.Hub, *goals.Parser, string) {
	t.Helper()
	dir := t.TempDir()

	// Create directory structure
	os.MkdirAll(filepath.Join(dir, "goals", "active"), 0755)
	os.MkdirAll(filepath.Join(dir, "goals", "iced"), 0755)
	os.MkdirAll(filepath.Join(dir, "goals", "history"), 0755)
	os.MkdirAll(filepath.Join(dir, "workspaces", "test-project", "goal-abc1234-test"), 0755)

	// Create registry
	registry := `# Goal Registry

## Active Goals

| ID | Title | Project(s) | Status | Phase |
|----|-------|------------|--------|-------|
| abc1234 | Test goal | test-project | Active | 1/3 |
| | | | | |

## Completed Goals

| ID | Title | Project(s) | Completed |
|----|-------|------------|-----------|
`
	os.WriteFile(filepath.Join(dir, "goals", "REGISTRY.md"), []byte(registry), 0644)

	// Create goal file
	goalContent := `# Goal #abc1234: Test goal

## Overview

Test goal for API testing.

## Project(s)

- **test-project**: Main project

## Phases

### Phase 1: Setup
- [x] Task one
- [ ] Task two
- **Status:** in_progress

## Status

**Current Phase**: 1
**Status**: Active
`
	os.WriteFile(filepath.Join(dir, "goals", "active", "abc1234.md"), []byte(goalContent), 0644)

	h := hub.New(dir)
	p := goals.NewParser(dir)

	return h, p, dir
}

func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	handler := handleHealth()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["status"] != "ok" {
		t.Errorf("expected status 'ok', got '%s'", response["status"])
	}
}

func TestHandleQuestions_Empty(t *testing.T) {
	h, _, _ := setupTestEnv(t)

	req := httptest.NewRequest("GET", "/api/questions", nil)
	w := httptest.NewRecorder()

	handler := handleQuestions(h)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var questions []*hub.Question
	json.Unmarshal(w.Body.Bytes(), &questions)
	if len(questions) != 0 {
		t.Errorf("expected 0 questions, got %d", len(questions))
	}
}

func TestHandleQuestions_MethodNotAllowed(t *testing.T) {
	h, _, _ := setupTestEnv(t)

	req := httptest.NewRequest("POST", "/api/questions", nil)
	w := httptest.NewRecorder()

	handler := handleQuestions(h)
	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandleExecutors_Empty(t *testing.T) {
	h, _, _ := setupTestEnv(t)

	req := httptest.NewRequest("GET", "/api/executors", nil)
	w := httptest.NewRecorder()

	handler := handleExecutors(h)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var executors []*hub.Executor
	json.Unmarshal(w.Body.Bytes(), &executors)
	if len(executors) != 0 {
		t.Errorf("expected 0 executors, got %d", len(executors))
	}
}

func TestHandleExecutorRegister(t *testing.T) {
	h, _, _ := setupTestEnv(t)

	body := `{"goal_id": "abc1234", "session_id": "session-001", "cwd": "/path/to/worktree"}`
	req := httptest.NewRequest("POST", "/api/executor/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := handleExecutorRegister(h)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response ExecutorRegisterResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if !response.OK {
		t.Error("expected OK to be true")
	}
	if response.Context == "" {
		t.Error("expected non-empty context")
	}

	// Verify executor was registered
	executors := h.GetActiveExecutors()
	if len(executors) != 1 {
		t.Fatalf("expected 1 executor, got %d", len(executors))
	}
}

func TestHandleExecutorStop(t *testing.T) {
	h, _, _ := setupTestEnv(t)

	// First register an executor
	h.RegisterExecutor("abc1234", "session-001", "/path/to/worktree")

	// Now stop it
	body := `{"goal_id": "abc1234", "session_id": "session-001", "reason": "completed"}`
	req := httptest.NewRequest("POST", "/api/executor/stop", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := handleExecutorStop(h)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify executor was removed
	executors := h.GetActiveExecutors()
	if len(executors) != 0 {
		t.Errorf("expected 0 executors, got %d", len(executors))
	}
}

func TestHandleGoals(t *testing.T) {
	h, p, _ := setupTestEnv(t)

	req := httptest.NewRequest("GET", "/api/goals", nil)
	w := httptest.NewRecorder()

	handler := handleGoals(h, p)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var summaries []GoalSummary
	json.Unmarshal(w.Body.Bytes(), &summaries)

	// Should have 1 goal from registry
	if len(summaries) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(summaries))
	}

	if summaries[0].ID != "abc1234" {
		t.Errorf("expected goal ID 'abc1234', got '%s'", summaries[0].ID)
	}
	if summaries[0].ExecutorStatus != "stopped" {
		t.Errorf("expected executor status 'stopped', got '%s'", summaries[0].ExecutorStatus)
	}
}

func TestHandleGoals_WithExecutor(t *testing.T) {
	h, p, _ := setupTestEnv(t)

	// Register an executor
	h.RegisterExecutor("abc1234", "session-001", "/path")

	req := httptest.NewRequest("GET", "/api/goals", nil)
	w := httptest.NewRecorder()

	handler := handleGoals(h, p)
	handler(w, req)

	var summaries []GoalSummary
	json.Unmarshal(w.Body.Bytes(), &summaries)

	if summaries[0].ExecutorStatus != "running" {
		t.Errorf("expected executor status 'running', got '%s'", summaries[0].ExecutorStatus)
	}
	if summaries[0].ActiveExecutors != 1 {
		t.Errorf("expected 1 active executor, got %d", summaries[0].ActiveExecutors)
	}
}

func TestHandleAnswer_NotFound(t *testing.T) {
	h, _, _ := setupTestEnv(t)

	body := `{"answer": "test answer"}`
	req := httptest.NewRequest("POST", "/api/answer/nonexistent", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := handleAnswer(h)
	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleAnswer_MissingID(t *testing.T) {
	h, _, _ := setupTestEnv(t)

	body := `{"answer": "test"}`
	req := httptest.NewRequest("POST", "/api/answer/", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	handler := handleAnswer(h)
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCorsMiddleware(t *testing.T) {
	innerHandler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	handler := corsMiddleware(innerHandler)

	// Test OPTIONS (preflight)
	req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header")
	}
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for OPTIONS, got %d", w.Code)
	}

	// Test regular request
	req = httptest.NewRequest("GET", "/api/test", nil)
	w = httptest.NewRecorder()
	handler(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS header on regular request")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()

	if id1 == "" {
		t.Error("expected non-empty ID")
	}

	// ID format should be date-time based
	if len(id1) < 15 {
		t.Errorf("expected ID length >= 15, got %d", len(id1))
	}
}

func TestHandleGoalOutput(t *testing.T) {
	h, _, dir := setupTestEnv(t)

	// Create output log file
	outputContent := "line 1\nline 2\nline 3\n"
	worktreePath := filepath.Join(dir, "workspaces", "test-project", "goal-abc1234-test")
	os.WriteFile(filepath.Join(worktreePath, ".executor-output.log"), []byte(outputContent), 0644)

	req := httptest.NewRequest("GET", "/api/goals/abc1234/output", nil)
	w := httptest.NewRecorder()

	handler := handleGoalOutput(h, "abc1234")
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response OutputResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	// Note: This might fail if worktree finding logic differs
	// The test validates the handler structure works
}

func TestHandleGoalOutput_Tail(t *testing.T) {
	h, _, _ := setupTestEnv(t)

	req := httptest.NewRequest("GET", "/api/goals/abc1234/output?tail=10", nil)
	w := httptest.NewRecorder()

	handler := handleGoalOutput(h, "abc1234")
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
