package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/lasmarois/vega-hub/internal/hub"
)

// RegisterRoutes sets up all API routes
func RegisterRoutes(mux *http.ServeMux, h *hub.Hub, p *goals.Parser) {
	mux.HandleFunc("/api/ask", corsMiddleware(handleAsk(h)))
	mux.HandleFunc("/api/answer/", corsMiddleware(handleAnswer(h)))
	mux.HandleFunc("/api/questions", corsMiddleware(handleQuestions(h)))
	mux.HandleFunc("/api/executors", corsMiddleware(handleExecutors(h)))
	mux.HandleFunc("/api/executor/register", corsMiddleware(handleExecutorRegister(h)))
	mux.HandleFunc("/api/executor/stop", corsMiddleware(handleExecutorStop(h)))
	mux.HandleFunc("/api/events", handleSSE(h))
	mux.HandleFunc("/api/health", handleHealth())
	mux.HandleFunc("/api/goals", corsMiddleware(handleGoals(h, p)))
	mux.HandleFunc("/api/goals/", corsMiddleware(handleGoalRoutes(h, p)))
}

// AskRequest is the request body for POST /api/ask
type AskRequest struct {
	GoalID    string       `json:"goal_id"`
	SessionID string       `json:"session_id"`
	Question  string       `json:"question"`
	Options   []hub.Option `json:"options,omitempty"`
}

// AskResponse is the response for POST /api/ask
type AskResponse struct {
	Answer string `json:"answer"`
}

// AnswerRequest is the request body for POST /api/answer/:id
type AnswerRequest struct {
	Answer string `json:"answer"`
}

// handleAsk handles POST /api/ask - blocks until question is answered
func handleAsk(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req AskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Generate question ID
		id := generateID()

		q := &hub.Question{
			ID:        id,
			GoalID:    req.GoalID,
			SessionID: req.SessionID,
			Question:  req.Question,
			Options:   req.Options,
		}

		// This blocks until answer is received
		answer := h.Ask(q)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AskResponse{Answer: answer})
	}
}

// handleAnswer handles POST /api/answer/:id
func handleAnswer(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract ID from path
		id := strings.TrimPrefix(r.URL.Path, "/api/answer/")
		if id == "" {
			http.Error(w, "Missing question ID", http.StatusBadRequest)
			return
		}

		var req AnswerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if !h.Answer(id, req.Answer) {
			http.Error(w, "Question not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

// handleQuestions handles GET /api/questions
func handleQuestions(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		questions := h.GetPendingQuestions()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(questions)
	}
}

// handleHealth handles GET /api/health
func handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

// ExecutorRegisterRequest is the request body for POST /api/executor/register
type ExecutorRegisterRequest struct {
	GoalID    string `json:"goal_id"`
	SessionID string `json:"session_id"`
	CWD       string `json:"cwd"`
}

// ExecutorRegisterResponse is the response for POST /api/executor/register
type ExecutorRegisterResponse struct {
	OK      bool   `json:"ok"`
	Context string `json:"context"`
}

// ExecutorStopRequest is the request body for POST /api/executor/stop
type ExecutorStopRequest struct {
	GoalID    string `json:"goal_id"`
	SessionID string `json:"session_id"`
	Reason    string `json:"reason,omitempty"`
}

// handleExecutorRegister handles POST /api/executor/register
func handleExecutorRegister(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ExecutorRegisterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Register the executor and get context
		// Note: This is the legacy hook-based registration path, user is unknown
		context := h.RegisterExecutor(req.GoalID, req.SessionID, req.CWD, "")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ExecutorRegisterResponse{
			OK:      true,
			Context: context,
		})
	}
}

// handleExecutorStop handles POST /api/executor/stop
func handleExecutorStop(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ExecutorStopRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Stop the executor
		h.StopExecutor(req.GoalID, req.SessionID, req.Reason)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}
}

// handleExecutors handles GET /api/executors
func handleExecutors(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		executors := h.GetActiveExecutors()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(executors)
	}
}

// corsMiddleware adds CORS headers for development
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// generateID creates a simple unique ID
func generateID() string {
	// Simple timestamp-based ID for MVP
	return strings.ReplaceAll(
		strings.ReplaceAll(
			time.Now().Format("20060102-150405.000"),
			".", "-"),
		":", "")
}

// GoalSummary combines registry goal with runtime status
type GoalSummary struct {
	goals.Goal
	ExecutorStatus   string `json:"executor_status"` // "running", "waiting", "stopped", "none"
	PendingQuestions int    `json:"pending_questions"`
	ActiveExecutors  int    `json:"active_executors"`
}

// handleGoals handles GET /api/goals - lists all goals with runtime status
func handleGoals(h *hub.Hub, p *goals.Parser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse registry
		registryGoals, err := p.ParseRegistry()
		if err != nil {
			http.Error(w, "Failed to parse registry: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Get runtime state
		executors := h.GetActiveExecutors()
		questions := h.GetPendingQuestions()

		// Build executor and question maps by goal ID
		executorsByGoal := make(map[string]int)
		for _, e := range executors {
			executorsByGoal[e.GoalID]++
		}

		questionsByGoal := make(map[string]int)
		for _, q := range questions {
			questionsByGoal[q.GoalID]++
		}

		// Build summaries
		summaries := make([]GoalSummary, 0, len(registryGoals))
		for _, g := range registryGoals {
			summary := GoalSummary{
				Goal:             g,
				PendingQuestions: questionsByGoal[g.ID],
				ActiveExecutors:  executorsByGoal[g.ID],
			}

			// Determine executor status
			if questionsByGoal[g.ID] > 0 {
				summary.ExecutorStatus = "waiting"
			} else if executorsByGoal[g.ID] > 0 {
				summary.ExecutorStatus = "running"
			} else if g.Status == "active" {
				summary.ExecutorStatus = "stopped"
			} else {
				summary.ExecutorStatus = "none"
			}

			summaries = append(summaries, summary)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(summaries)
	}
}

// GoalDetailResponse combines goal detail with Q&A history
type GoalDetailResponse struct {
	*goals.GoalDetail
	ExecutorStatus   string          `json:"executor_status"`
	PendingQuestions []*hub.Question `json:"pending_questions"`
	ActiveExecutors  []*hub.Executor `json:"active_executors"`
}

// SpawnRequest is the request body for POST /api/goals/:id/spawn
type SpawnRequest struct {
	Context string `json:"context,omitempty"`
	User    string `json:"user,omitempty"` // Username spawning this executor
}

// handleGoalRoutes routes /api/goals/:id/* requests
func handleGoalRoutes(h *hub.Hub, p *goals.Parser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse path: /api/goals/:id or /api/goals/:id/action
		path := strings.TrimPrefix(r.URL.Path, "/api/goals/")
		parts := strings.SplitN(path, "/", 2)

		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "Missing goal ID", http.StatusBadRequest)
			return
		}

		id := parts[0] // Goal ID can be numeric ("10") or hash ("4fd584d")

		// Route to appropriate handler
		if len(parts) == 1 {
			// GET /api/goals/:id
			handleGoalDetail(h, p, id)(w, r)
			return
		}

		action := parts[1]
		switch action {
		case "spawn":
			handleGoalSpawn(h, id)(w, r)
		case "status":
			handleGoalStatus(h, id)(w, r)
		case "output":
			handleGoalOutput(h, id)(w, r)
		default:
			http.Error(w, "Unknown action: "+action, http.StatusNotFound)
		}
	}
}

// handleGoalDetail handles GET /api/goals/:id - returns goal detail with Q&A
func handleGoalDetail(h *hub.Hub, p *goals.Parser, id string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse goal detail
		detail, err := p.ParseGoalDetail(id)
		if err != nil {
			http.Error(w, "Goal not found: "+err.Error(), http.StatusNotFound)
			return
		}

		// Get runtime state for this goal
		allExecutors := h.GetActiveExecutors()
		allQuestions := h.GetPendingQuestions()

		// Filter to this goal
		var goalExecutors []*hub.Executor
		for _, e := range allExecutors {
			if e.GoalID == id {
				goalExecutors = append(goalExecutors, e)
			}
		}

		var goalQuestions []*hub.Question
		for _, q := range allQuestions {
			if q.GoalID == id {
				goalQuestions = append(goalQuestions, q)
			}
		}

		// Determine status
		status := "none"
		if len(goalQuestions) > 0 {
			status = "waiting"
		} else if len(goalExecutors) > 0 {
			status = "running"
		} else if detail.Status == "active" {
			status = "stopped"
		}

		response := GoalDetailResponse{
			GoalDetail:       detail,
			ExecutorStatus:   status,
			PendingQuestions: goalQuestions,
			ActiveExecutors:  goalExecutors,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// handleGoalSpawn handles POST /api/goals/:id/spawn - spawns an executor
func handleGoalSpawn(h *hub.Hub, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[SPAWN] Received spawn request for Goal #%s from %s", goalID, r.RemoteAddr)

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req SpawnRequest
		if r.Body != nil && r.ContentLength > 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
		}

		// Get user from X-Vega-User header, or from request body, or auto-detect
		user := r.Header.Get("X-Vega-User")
		if user == "" {
			user = req.User
		}

		log.Printf("[SPAWN] Processing spawn for Goal #%s, context: %q, user: %q", goalID, req.Context, user)

		result := h.SpawnExecutor(hub.SpawnRequest{
			GoalID:  goalID,
			Context: req.Context,
			User:    user,
		})

		log.Printf("[SPAWN] Result for Goal #%s: success=%v, message=%s", goalID, result.Success, result.Message)

		w.Header().Set("Content-Type", "application/json")
		if !result.Success {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(result)
	}
}

// handleGoalStatus handles GET /api/goals/:id/status - returns planning file status
func handleGoalStatus(h *hub.Hub, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		status, err := h.GetGoalStatus(goalID)
		if err != nil {
			http.Error(w, "Failed to get status: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	}
}

// OutputResponse is the response for GET /api/goals/:id/output
type OutputResponse struct {
	Output    string `json:"output"`
	Available bool   `json:"available"`
}

// handleGoalOutput handles GET /api/goals/:id/output - returns executor output
func handleGoalOutput(h *hub.Hub, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check for tail parameter (last N lines)
		tailParam := r.URL.Query().Get("tail")
		var output string
		var err error

		if tailParam != "" {
			lines, parseErr := strconv.Atoi(tailParam)
			if parseErr != nil {
				lines = 50 // default
			}
			output, err = h.GetExecutorOutputTail(goalID, lines)
		} else {
			output, err = h.GetExecutorOutput(goalID)
		}

		response := OutputResponse{
			Output:    output,
			Available: err == nil && output != "",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
