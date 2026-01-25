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
	"github.com/lasmarois/vega-hub/internal/operations"
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
	mux.HandleFunc("/api/goals", corsMiddleware(handleGoalsRoot(h, p)))
	mux.HandleFunc("/api/goals/", corsMiddleware(handleGoalRoutes(h, p)))
	mux.HandleFunc("/api/projects", corsMiddleware(handleProjects(h)))
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

// CreateGoalRequest is the request body for POST /api/goals
type CreateGoalRequest struct {
	Title      string `json:"title"`
	Project    string `json:"project"`
	BaseBranch string `json:"base_branch,omitempty"`
}

// CompleteGoalRequest is the request body for POST /api/goals/:id/complete
type CompleteGoalRequest struct {
	Project string `json:"project"`
	NoMerge bool   `json:"no_merge,omitempty"`
	Force   bool   `json:"force,omitempty"`
}

// IceGoalRequest is the request body for POST /api/goals/:id/ice
type IceGoalRequest struct {
	Project string `json:"project"`
	Reason  string `json:"reason"`
	Force   bool   `json:"force,omitempty"`
}

// CleanupGoalRequest is the request body for POST /api/goals/:id/cleanup
type CleanupGoalRequest struct {
	Project string `json:"project"`
}

// handleGoalsRoot handles /api/goals - GET lists goals, POST creates a goal
func handleGoalsRoot(h *hub.Hub, p *goals.Parser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleGoals(h, p)(w, r)
		case http.MethodPost:
			handleCreateGoal(h)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
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
		case "complete":
			handleGoalComplete(h, id)(w, r)
		case "ice":
			handleGoalIce(h, id)(w, r)
		case "cleanup":
			handleGoalCleanup(h, id)(w, r)
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

// handleCreateGoal handles POST /api/goals - creates a new goal
func handleCreateGoal(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[CREATE] Received create goal request from %s", r.RemoteAddr)

		var req CreateGoalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Title == "" {
			http.Error(w, "Title is required", http.StatusBadRequest)
			return
		}
		if req.Project == "" {
			http.Error(w, "Project is required", http.StatusBadRequest)
			return
		}

		log.Printf("[CREATE] Creating goal: title=%q, project=%q, base_branch=%q", req.Title, req.Project, req.BaseBranch)

		result, data := operations.CreateGoal(operations.CreateOptions{
			Title:      req.Title,
			Project:    req.Project,
			BaseBranch: req.BaseBranch,
			VegaDir:    h.Dir(),
		})

		w.Header().Set("Content-Type", "application/json")
		if !result.Success {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(result)
			return
		}

		log.Printf("[CREATE] Goal created: id=%s, branch=%s", data.GoalID, data.GoalBranch)

		// Emit SSE event for goal created
		h.EmitEvent("goal_created", map[string]interface{}{
			"goal_id": data.GoalID,
			"title":   data.Title,
			"project": data.Project,
		})

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    data,
		})
	}
}

// handleGoalComplete handles POST /api/goals/:id/complete - completes a goal
func handleGoalComplete(h *hub.Hub, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		log.Printf("[COMPLETE] Received complete request for goal %s from %s", goalID, r.RemoteAddr)

		var req CompleteGoalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Project == "" {
			http.Error(w, "Project is required", http.StatusBadRequest)
			return
		}

		log.Printf("[COMPLETE] Completing goal %s in project %s (no_merge=%v, force=%v)", goalID, req.Project, req.NoMerge, req.Force)

		result, data := operations.CompleteGoal(operations.CompleteOptions{
			GoalID:  goalID,
			Project: req.Project,
			NoMerge: req.NoMerge,
			Force:   req.Force,
			VegaDir: h.Dir(),
		})

		w.Header().Set("Content-Type", "application/json")
		if !result.Success {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(result)
			return
		}

		log.Printf("[COMPLETE] Goal %s completed successfully", goalID)

		// Emit SSE event for goal completed
		h.EmitEvent("goal_completed", map[string]interface{}{
			"goal_id": goalID,
			"title":   data.Title,
			"project": data.Project,
			"merged":  data.Merged,
		})

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    data,
		})
	}
}

// handleGoalIce handles POST /api/goals/:id/ice - ices (pauses) a goal
func handleGoalIce(h *hub.Hub, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		log.Printf("[ICE] Received ice request for goal %s from %s", goalID, r.RemoteAddr)

		var req IceGoalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Project == "" {
			http.Error(w, "Project is required", http.StatusBadRequest)
			return
		}
		if req.Reason == "" {
			http.Error(w, "Reason is required", http.StatusBadRequest)
			return
		}

		log.Printf("[ICE] Icing goal %s in project %s (reason=%q, force=%v)", goalID, req.Project, req.Reason, req.Force)

		result, data := operations.IceGoal(operations.IceOptions{
			GoalID:  goalID,
			Project: req.Project,
			Reason:  req.Reason,
			Force:   req.Force,
			VegaDir: h.Dir(),
		})

		w.Header().Set("Content-Type", "application/json")
		if !result.Success {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(result)
			return
		}

		log.Printf("[ICE] Goal %s iced successfully", goalID)

		// Emit SSE event for goal iced
		h.EmitEvent("goal_iced", map[string]interface{}{
			"goal_id": goalID,
			"title":   data.Title,
			"project": data.Project,
			"reason":  data.Reason,
		})

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    data,
		})
	}
}

// handleGoalCleanup handles POST /api/goals/:id/cleanup - cleans up a completed goal's branch
func handleGoalCleanup(h *hub.Hub, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		log.Printf("[CLEANUP] Received cleanup request for goal %s from %s", goalID, r.RemoteAddr)

		var req CleanupGoalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Project == "" {
			http.Error(w, "Project is required", http.StatusBadRequest)
			return
		}

		log.Printf("[CLEANUP] Cleaning up goal %s in project %s", goalID, req.Project)

		result, data := operations.CleanupGoal(operations.CleanupOptions{
			GoalID:  goalID,
			Project: req.Project,
			VegaDir: h.Dir(),
		})

		w.Header().Set("Content-Type", "application/json")
		if !result.Success {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(result)
			return
		}

		log.Printf("[CLEANUP] Goal %s branch cleaned up successfully", goalID)

		// Emit SSE event for cleanup
		h.EmitEvent("goal_cleanup", map[string]interface{}{
			"goal_id": goalID,
			"project": data.Project,
			"branch":  data.Branch,
		})

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    data,
		})
	}
}

// ProjectSummary is a simplified project info for the API
type ProjectSummary struct {
	Name       string `json:"name"`
	BaseBranch string `json:"base_branch"`
	Workspace  string `json:"workspace,omitempty"`
	Upstream   string `json:"upstream,omitempty"`
}

// handleProjects handles GET /api/projects - lists all projects
func handleProjects(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		projects, err := operations.ListProjects(h.Dir())
		if err != nil {
			http.Error(w, "Failed to list projects: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert to summaries
		summaries := make([]ProjectSummary, 0, len(projects))
		for _, p := range projects {
			summaries = append(summaries, ProjectSummary{
				Name:       p.Name,
				BaseBranch: p.BaseBranch,
				Workspace:  p.Workspace,
				Upstream:   p.Upstream,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(summaries)
	}
}
