package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lasmarois/vega-hub/internal/credentials"
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
	// Session history routes
	mux.HandleFunc("/api/history/", corsMiddleware(handleHistoryRoutes(h)))
	// User identity and credentials routes
	mux.HandleFunc("/api/user", corsMiddleware(handleGetUser()))
	mux.HandleFunc("/api/user/", corsMiddleware(handleUserRoutes(p)))
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
	GoalID          string `json:"goal_id"`
	SessionID       string `json:"session_id"`         // vega-hub session ID
	ClaudeSessionID string `json:"claude_session_id"`  // Claude Code's session ID (from hooks)
	TranscriptPath  string `json:"transcript_path"`    // Path to Claude's conversation JSONL
	Reason          string `json:"reason,omitempty"`
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

		// Stop the executor with Claude session info
		h.StopExecutorWithClaudeInfo(hub.StopExecutorRequest{
			GoalID:          req.GoalID,
			SessionID:       req.SessionID,
			ClaudeSessionID: req.ClaudeSessionID,
			TranscriptPath:  req.TranscriptPath,
			Reason:          req.Reason,
		})

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
	ExecutorStatus   string `json:"executor_status"`            // "running", "waiting", "stopped", "none"
	PendingQuestions int    `json:"pending_questions"`
	ActiveExecutors  int    `json:"active_executors"`
	WorkspaceStatus  string `json:"workspace_status,omitempty"` // "ready", "missing", "error" (from project)
	WorkspaceError   string `json:"workspace_error,omitempty"`  // Error message if workspace not ready
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
	Project        string `json:"project"`
	Reason         string `json:"reason"`
	RemoveWorktree bool   `json:"remove_worktree,omitempty"` // If true, remove worktree (default: keep)
	Force          bool   `json:"force,omitempty"`           // If true, ignore uncommitted changes
}

// CleanupGoalRequest is the request body for POST /api/goals/:id/cleanup
type CleanupGoalRequest struct {
	Project string `json:"project"`
}

// ResumeGoalRequest is the request body for POST /api/goals/:id/resume
type ResumeGoalRequest struct {
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

		// Cache project workspace status
		projectStatus := make(map[string]struct {
			status string
			error  string
		})

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

			// Get workspace status from first project
			if len(g.Projects) > 0 {
				projectName := g.Projects[0]
				if cached, ok := projectStatus[projectName]; ok {
					summary.WorkspaceStatus = cached.status
					summary.WorkspaceError = cached.error
				} else {
					// Parse project to get workspace status
					if proj, err := p.ParseProject(projectName); err == nil {
						summary.WorkspaceStatus = proj.WorkspaceStatus
						summary.WorkspaceError = proj.WorkspaceError
						projectStatus[projectName] = struct {
							status string
							error  string
						}{proj.WorkspaceStatus, proj.WorkspaceError}
					} else {
						summary.WorkspaceStatus = "error"
						summary.WorkspaceError = "Project config not found"
						projectStatus[projectName] = struct {
							status string
							error  string
						}{"error", "Project config not found"}
					}
				}
			}

			summaries = append(summaries, summary)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(summaries)
	}
}

// BranchInfo contains git branch information for a goal's worktree
type BranchInfo struct {
	Branch           string `json:"branch"`
	BaseBranch       string `json:"base_branch"`
	Ahead            int    `json:"ahead"`
	Behind           int    `json:"behind"`
	UncommittedFiles int    `json:"uncommitted_files"`
	LastCommit       string `json:"last_commit,omitempty"`
	LastCommitMsg    string `json:"last_commit_message,omitempty"`
	WorktreePath     string `json:"worktree_path,omitempty"`
}

// GoalDetailResponse combines goal detail with Q&A history
type GoalDetailResponse struct {
	*goals.GoalDetail
	ExecutorStatus   string          `json:"executor_status"`
	PendingQuestions []*hub.Question `json:"pending_questions"`
	ActiveExecutors  []*hub.Executor `json:"active_executors"`
	WorkspaceStatus  string          `json:"workspace_status,omitempty"` // "ready", "missing", "error"
	WorkspaceError   string          `json:"workspace_error,omitempty"`  // Error message if not ready
	BranchInfo       *BranchInfo     `json:"branch_info,omitempty"`      // Git branch info for worktree
}

// SpawnRequest is the request body for POST /api/goals/:id/spawn
type SpawnRequest struct {
	Context string `json:"context,omitempty"`
	User    string `json:"user,omitempty"` // Username spawning this executor
	Mode    string `json:"mode,omitempty"` // Executor mode: plan, implement, review, test, security, quick
}

// CreateMRRequest is the request body for POST /api/goals/:id/create-mr
type CreateMRRequest struct {
	Title        string `json:"title"`
	Description  string `json:"description,omitempty"`
	TargetBranch string `json:"target_branch,omitempty"` // Defaults to base branch
	Draft        bool   `json:"draft,omitempty"`
}

// CreateMRResponse is the response for POST /api/goals/:id/create-mr
type CreateMRResponse struct {
	Success  bool   `json:"success"`
	MRURL    string `json:"mr_url,omitempty"`
	MRNumber int    `json:"mr_number,omitempty"`
	Service  string `json:"service,omitempty"` // "github" or "gitlab"
	Error    string `json:"error,omitempty"`
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

		// Handle nested paths like "messages/pending"
		actionParts := strings.SplitN(action, "/", 2)
		baseAction := actionParts[0]

		switch baseAction {
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
		case "resume":
			handleGoalResume(h, id)(w, r)
		case "sessions":
			handleGoalSessions(h, id)(w, r)
		case "history":
			handleGoalHistoryEntries(h, id)(w, r)
		case "chat":
			handleGoalChat(h, id)(w, r)
		case "messages":
			// Check for nested path like "messages/pending"
			if len(actionParts) > 1 && actionParts[1] == "pending" {
				handleGetPendingMessages(h, id)(w, r)
			} else {
				handleGoalMessages(h, id)(w, r)
			}
		case "create-mr":
			handleCreateMR(h, p, id)(w, r)
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

		// Get workspace status from first project
		if len(detail.Projects) > 0 {
			if proj, err := p.ParseProject(detail.Projects[0]); err == nil {
				response.WorkspaceStatus = proj.WorkspaceStatus
				response.WorkspaceError = proj.WorkspaceError
			} else {
				response.WorkspaceStatus = "error"
				response.WorkspaceError = "Project config not found"
			}
		}

		// Get branch info for active goals with worktrees
		if detail.Status == "active" && len(detail.Projects) > 0 {
			response.BranchInfo = getBranchInfo(p.Dir(), id, detail.Projects)
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

		// Validate mode if specified
		mode := req.Mode
		if mode != "" && !hub.ValidModes[mode] {
			http.Error(w, fmt.Sprintf("Invalid mode: %s. Valid modes: plan, implement, review, test, security, quick", mode), http.StatusBadRequest)
			return
		}

		log.Printf("[SPAWN] Processing spawn for Goal #%s, context: %q, user: %q, mode: %q", goalID, req.Context, user, mode)

		result := h.SpawnExecutor(hub.SpawnRequest{
			GoalID:  goalID,
			Context: req.Context,
			User:    user,
			Mode:    mode,
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

		log.Printf("[ICE] Icing goal %s in project %s (reason=%q, remove_worktree=%v, force=%v)", goalID, req.Project, req.Reason, req.RemoveWorktree, req.Force)

		result, data := operations.IceGoal(operations.IceOptions{
			GoalID:         goalID,
			Project:        req.Project,
			Reason:         req.Reason,
			RemoveWorktree: req.RemoveWorktree,
			Force:          req.Force,
			VegaDir:        h.Dir(),
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

// handleGoalResume handles POST /api/goals/:id/resume - resumes an iced goal
func handleGoalResume(h *hub.Hub, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		log.Printf("[RESUME] Received resume request for goal %s from %s", goalID, r.RemoteAddr)

		var req ResumeGoalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Project == "" {
			http.Error(w, "Project is required", http.StatusBadRequest)
			return
		}

		log.Printf("[RESUME] Resuming goal %s in project %s", goalID, req.Project)

		result, data := operations.ResumeGoal(operations.ResumeOptions{
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

		log.Printf("[RESUME] Goal %s resumed successfully", goalID)

		// Emit SSE event for goal resumed
		h.EmitEvent("goal_resumed", map[string]interface{}{
			"goal_id":          goalID,
			"title":            data.Title,
			"project":          data.Project,
			"worktree_created": data.WorktreeCreated,
		})

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    data,
		})
	}
}

// handleCreateMR handles POST /api/goals/:id/create-mr - creates a merge request
func handleCreateMR(h *hub.Hub, p *goals.Parser, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		log.Printf("[CREATE-MR] Received request for goal %s from %s", goalID, r.RemoteAddr)

		var req CreateMRRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(CreateMRResponse{
				Success: false,
				Error:   "Invalid JSON: " + err.Error(),
			})
			return
		}

		if req.Title == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(CreateMRResponse{
				Success: false,
				Error:   "Title is required",
			})
			return
		}

		// Get goal detail to find project
		detail, err := p.ParseGoalDetail(goalID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(CreateMRResponse{
				Success: false,
				Error:   "Goal not found: " + err.Error(),
			})
			return
		}

		if len(detail.Projects) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(CreateMRResponse{
				Success: false,
				Error:   "Goal has no associated projects",
			})
			return
		}

		// Find worktree for this goal
		project := detail.Projects[0]
		worktreePath, _ := findWorktreeForGoal(p.Dir(), goalID, detail.Projects)
		if worktreePath == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(CreateMRResponse{
				Success: false,
				Error:   "No worktree found for this goal",
			})
			return
		}

		// Get project config to determine git service
		proj, err := p.ParseProject(project)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(CreateMRResponse{
				Success: false,
				Error:   "Project config not found: " + err.Error(),
			})
			return
		}

		// Determine target branch
		targetBranch := req.TargetBranch
		if targetBranch == "" {
			targetBranch = proj.BaseBranch
			if targetBranch == "" {
				targetBranch = "main"
			}
		}

		// Detect git service from remote URL
		service := detectGitService(proj.GitRemote)
		log.Printf("[CREATE-MR] Detected service: %s for remote: %s", service, proj.GitRemote)

		// Create MR/PR
		var mrURL string
		var mrNumber int

		switch service {
		case "github":
			mrURL, mrNumber, err = createGitHubPR(worktreePath, req.Title, req.Description, targetBranch, req.Draft)
		case "gitlab":
			mrURL, mrNumber, err = createGitLabMR(worktreePath, req.Title, req.Description, targetBranch, req.Draft)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(CreateMRResponse{
				Success: false,
				Error:   "Unknown git service. Remote URL must contain github.com or gitlab",
			})
			return
		}

		if err != nil {
			log.Printf("[CREATE-MR] Failed to create MR: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(CreateMRResponse{
				Success: false,
				Service: service,
				Error:   err.Error(),
			})
			return
		}

		log.Printf("[CREATE-MR] Created %s MR #%d: %s", service, mrNumber, mrURL)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CreateMRResponse{
			Success:  true,
			MRURL:    mrURL,
			MRNumber: mrNumber,
			Service:  service,
		})
	}
}

// detectGitService determines if a remote URL is GitHub or GitLab
func detectGitService(remoteURL string) string {
	lower := strings.ToLower(remoteURL)
	if strings.Contains(lower, "github.com") {
		return "github"
	}
	if strings.Contains(lower, "gitlab") {
		return "gitlab"
	}
	return "unknown"
}

// createGitHubPR creates a pull request using gh CLI
func createGitHubPR(repoPath, title, description, targetBranch string, draft bool) (string, int, error) {
	args := []string{"pr", "create", "--title", title, "--base", targetBranch}

	if description != "" {
		args = append(args, "--body", description)
	} else {
		args = append(args, "--body", "")
	}

	if draft {
		args = append(args, "--draft")
	}

	cmd := exec.Command("gh", args...)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, fmt.Errorf("gh pr create failed: %s", strings.TrimSpace(string(output)))
	}

	// Parse URL from output (last line is the PR URL)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	prURL := lines[len(lines)-1]

	// Extract PR number from URL (e.g., https://github.com/user/repo/pull/123)
	prNumber := 0
	if idx := strings.LastIndex(prURL, "/"); idx != -1 {
		fmt.Sscanf(prURL[idx+1:], "%d", &prNumber)
	}

	return prURL, prNumber, nil
}

// createGitLabMR creates a merge request using glab CLI
func createGitLabMR(repoPath, title, description, targetBranch string, draft bool) (string, int, error) {
	args := []string{"mr", "create", "--title", title, "--target-branch", targetBranch, "--yes"}

	if description != "" {
		args = append(args, "--description", description)
	}

	if draft {
		args = append(args, "--draft")
	}

	cmd := exec.Command("glab", args...)
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, fmt.Errorf("glab mr create failed: %s", strings.TrimSpace(string(output)))
	}

	// Parse MR URL and number from output
	// glab outputs: "Creating merge request for branch into main in user/repo\n!123 https://gitlab.com/..."
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	mrURL := ""
	mrNumber := 0

	for _, line := range lines {
		// Look for URL
		if strings.Contains(line, "https://") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "https://") {
					mrURL = part
				}
				if strings.HasPrefix(part, "!") {
					fmt.Sscanf(part[1:], "%d", &mrNumber)
				}
			}
		}
	}

	if mrURL == "" {
		// If we can't parse, return the full output
		return strings.TrimSpace(string(output)), mrNumber, nil
	}

	return mrURL, mrNumber, nil
}

// ProjectSummary is a simplified project info for the API
type ProjectSummary struct {
	Name            string `json:"name"`
	BaseBranch      string `json:"base_branch"`
	Workspace       string `json:"workspace,omitempty"`
	Upstream        string `json:"upstream,omitempty"`
	WorkspaceStatus string `json:"workspace_status"`          // "ready", "missing", "error"
	WorkspaceError  string `json:"workspace_error,omitempty"` // Error message if not ready
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
				Name:            p.Name,
				BaseBranch:      p.BaseBranch,
				Workspace:       p.Workspace,
				Upstream:        p.Upstream,
				WorkspaceStatus: p.WorkspaceStatus,
				WorkspaceError:  p.WorkspaceError,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(summaries)
	}
}

// handleGoalSessions handles GET /api/goals/:id/sessions - returns session history for a goal
func handleGoalSessions(h *hub.Hub, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		sessions, err := h.GetGoalSessions(goalID)
		if err != nil {
			http.Error(w, "Failed to get sessions: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if sessions == nil {
			sessions = []*hub.ExecutorSession{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
	}
}

// handleGoalHistoryEntries handles GET /api/goals/:id/history - returns detailed history for a goal
func handleGoalHistoryEntries(h *hub.Hub, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check for limit parameter
		limit := 0
		if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
			if l, err := strconv.Atoi(limitParam); err == nil {
				limit = l
			}
		}

		entries, err := h.GetGoalHistory(goalID, limit)
		if err != nil {
			http.Error(w, "Failed to get history: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if entries == nil {
			entries = []hub.HistoryEntry{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	}
}

// handleHistoryRoutes handles /api/history/* routes
func handleHistoryRoutes(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse path: /api/history/:goal_id or /api/history/:goal_id/:session_id
		path := strings.TrimPrefix(r.URL.Path, "/api/history/")
		parts := strings.SplitN(path, "/", 2)

		if len(parts) == 0 || parts[0] == "" {
			http.Error(w, "Missing goal ID", http.StatusBadRequest)
			return
		}

		goalID := parts[0]

		if len(parts) == 1 {
			// GET /api/history/:goal_id - returns all history for a goal
			handleGoalHistoryEntries(h, goalID)(w, r)
			return
		}

		sessionID := parts[1]
		// GET /api/history/:goal_id/:session_id - returns history for a specific session
		handleSessionHistory(h, goalID, sessionID)(w, r)
	}
}

// ChatMessage represents a message in the chat thread
// It transforms HistoryEntry to a format optimized for the chat UI
type ChatMessage struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"` // "session_start", "session_stop", "question", "answer", "user_message", "activity"
	Timestamp    string                 `json:"timestamp"`
	SessionID    string                 `json:"session_id"`
	GoalID       string                 `json:"goal_id"`
	Content      string                 `json:"content,omitempty"`       // question/answer/user_message text
	Answer       string                 `json:"answer,omitempty"`        // for question messages with answer
	ActivityType string                 `json:"activity_type,omitempty"` // for activity messages
	Data         map[string]interface{} `json:"data,omitempty"`          // activity details (expandable)
	Pending      bool                   `json:"pending,omitempty"`       // true for unanswered questions
	Options      []hub.Option           `json:"options,omitempty"`       // for questions with predefined choices
	User         string                 `json:"user,omitempty"`          // who sent (executor user, answering user)
	StopReason   string                 `json:"stop_reason,omitempty"`   // for session_stop
}

// handleGoalChat handles GET /api/goals/:id/chat - returns chat history as ChatMessage[]
// Query params:
//   - session: filter to specific session ID
//   - limit: max number of messages (default: 100)
func handleGoalChat(h *hub.Hub, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Parse query params
		sessionFilter := r.URL.Query().Get("session")
		limit := 100
		if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
			if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
				limit = l
			}
		}

		// Get history entries
		var entries []hub.HistoryEntry
		var err error
		if sessionFilter != "" {
			entries, err = h.GetSessionHistory(goalID, sessionFilter)
		} else {
			entries, err = h.GetGoalHistory(goalID, 0) // Get all, we'll limit after merging pending
		}
		if err != nil {
			http.Error(w, "Failed to get chat history: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Convert history entries to chat messages
		messages := make([]ChatMessage, 0, len(entries))
		for i, entry := range entries {
			msg := ChatMessage{
				ID:        fmt.Sprintf("hist-%d", i),
				Type:      entry.Type,
				Timestamp: entry.Timestamp.Format(time.RFC3339),
				SessionID: entry.SessionID,
				GoalID:    entry.GoalID,
				User:      entry.User,
			}

			switch entry.Type {
			case "session_start":
				msg.Content = fmt.Sprintf("Executor started in %s", entry.CWD)
			case "session_stop":
				msg.StopReason = entry.StopReason
				if entry.StopReason != "" {
					msg.Content = fmt.Sprintf("Executor stopped: %s", entry.StopReason)
				} else {
					msg.Content = "Executor stopped"
				}
				// Extract output from Data if present
				if entry.Data != nil {
					if dataMap, ok := entry.Data.(map[string]interface{}); ok {
						if output, ok := dataMap["output"].(string); ok && output != "" {
							msg.Data = map[string]interface{}{"output": output}
						}
					}
				}
			case "question":
				msg.Content = entry.Question
				msg.Answer = entry.Answer
				msg.Pending = entry.Answer == "" // Pending if no answer recorded
			case "user_message", "user_message_delivered":
				// Extract content and user from Data field
				if entry.Data != nil {
					if dataMap, ok := entry.Data.(map[string]interface{}); ok {
						msg.Data = dataMap
						if content, ok := dataMap["content"].(string); ok {
							msg.Content = content
						}
						if user, ok := dataMap["user"].(string); ok {
							msg.User = user
						}
						if pending, ok := dataMap["pending"].(bool); ok {
							msg.Pending = pending
						}
					}
				}
			case "activity":
				msg.ActivityType = entry.Type
				if entry.Data != nil {
					if dataMap, ok := entry.Data.(map[string]interface{}); ok {
						msg.Data = dataMap
					}
				}
			default:
				msg.Content = entry.Question // Fallback
			}

			messages = append(messages, msg)
		}

		// Add pending questions that aren't in history yet
		pendingQuestions := h.GetPendingQuestions()
		for _, q := range pendingQuestions {
			if q.GoalID == goalID && (sessionFilter == "" || q.SessionID == sessionFilter) {
				// Check if this question is already in messages (by matching question text and session)
				alreadyExists := false
				for _, m := range messages {
					if m.Type == "question" && m.Content == q.Question && m.SessionID == q.SessionID {
						alreadyExists = true
						break
					}
				}
				if !alreadyExists {
					msg := ChatMessage{
						ID:        "pending-" + q.ID,
						Type:      "question",
						Timestamp: q.CreatedAt.Format(time.RFC3339),
						SessionID: q.SessionID,
						GoalID:    q.GoalID,
						Content:   q.Question,
						Options:   q.Options,
						Pending:   true,
					}
					messages = append(messages, msg)
				}
			}
		}

		// Apply limit (take last N messages)
		if len(messages) > limit {
			messages = messages[len(messages)-limit:]
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	}
}

// handleSessionHistory handles GET /api/history/:goal_id/:session_id - returns history for a specific session
func handleSessionHistory(h *hub.Hub, goalID, sessionID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		entries, err := h.GetSessionHistory(goalID, sessionID)
		if err != nil {
			http.Error(w, "Failed to get session history: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if entries == nil {
			entries = []hub.HistoryEntry{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	}
}

// SendMessageRequest is the request body for POST /api/goals/:id/messages
type SendMessageRequest struct {
	Content string `json:"content"`
	User    string `json:"user,omitempty"`
}

// PendingMessagesResponse is the response for GET /api/goals/:id/messages/pending
type PendingMessagesResponse struct {
	HasMessages bool              `json:"has_messages"`
	Messages    []*hub.UserMessage `json:"messages"`
	// For Stop hook decision
	Decision string `json:"decision,omitempty"` // "block" or "allow"
	Reason   string `json:"reason,omitempty"`   // context to inject
}

// handleGoalMessages handles /api/goals/:id/messages
// POST - send a message to the executor
// GET - check pending message count
func handleGoalMessages(h *hub.Hub, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handleSendMessage(h, goalID)(w, r)
		case http.MethodGet:
			// GET /api/goals/:id/messages - return pending count
			handleCheckPendingMessages(h, goalID)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleSendMessage handles POST /api/goals/:id/messages - user sends message to executor
func handleSendMessage(h *hub.Hub, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if req.Content == "" {
			http.Error(w, "Content is required", http.StatusBadRequest)
			return
		}

		// Get user from header or request
		user := r.Header.Get("X-Vega-User")
		if user == "" {
			user = req.User
		}

		msg := h.SendUserMessage(goalID, req.Content, user)

		log.Printf("[MESSAGE] User message sent to goal %s: %q (user: %s)", goalID, req.Content, user)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      true,
			"message": msg,
		})
	}
}

// handleCheckPendingMessages handles GET /api/goals/:id/messages - check pending message count
func handleCheckPendingMessages(h *hub.Hub, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hasPending := h.HasPendingUserMessages(goalID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"has_pending": hasPending,
		})
	}
}

// handleGetPendingMessages handles GET /api/goals/:id/messages/pending
// Called by Stop hook to retrieve and clear pending messages
// Returns decision for the Stop hook
func handleGetPendingMessages(h *hub.Hub, goalID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get and clear pending messages
		messages := h.GetPendingUserMessages(goalID)

		response := PendingMessagesResponse{
			HasMessages: len(messages) > 0,
			Messages:    messages,
		}

		if len(messages) > 0 {
			// Build context string from all messages
			var contextParts []string
			for _, msg := range messages {
				if msg.User != "" {
					contextParts = append(contextParts, fmt.Sprintf("[User %s]: %s", msg.User, msg.Content))
				} else {
					contextParts = append(contextParts, fmt.Sprintf("[User]: %s", msg.Content))
				}
			}
			context := strings.Join(contextParts, "\n")

			response.Decision = "block"
			response.Reason = fmt.Sprintf("You have new messages from the user. Please read and address them before stopping:\n\n%s", context)

			log.Printf("[MESSAGE] Stop hook blocked for goal %s: %d pending messages", goalID, len(messages))
		} else {
			response.Decision = "allow"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// handleGetUser handles GET /api/user - returns current OS user info
func handleGetUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		user, err := credentials.GetCurrentUser()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"code":    "user_detection_failed",
					"message": err.Error(),
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

// handleUserRoutes handles /api/user/* routes
func handleUserRoutes(p *goals.Parser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse path: /api/user/credentials/:project
		path := strings.TrimPrefix(r.URL.Path, "/api/user/")
		parts := strings.Split(path, "/")

		if len(parts) < 2 || parts[0] != "credentials" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		project := parts[1]
		handleGetCredentials(p, project)(w, r)
	}
}

// handleGetCredentials handles GET /api/user/credentials/:project
func handleGetCredentials(p *goals.Parser, project string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get current user
		user, err := credentials.GetCurrentUser()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"code":    "user_detection_failed",
					"message": err.Error(),
				},
			})
			return
		}

		// Parse project to get git remote
		proj, err := p.ParseProject(project)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"code":    "project_not_found",
					"message": fmt.Sprintf("Project '%s' not found: %v", project, err),
				},
			})
			return
		}

		if proj.GitRemote == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"code":    "no_git_remote",
					"message": fmt.Sprintf("Project '%s' has no git remote configured", project),
				},
			})
			return
		}

		// Parse git service from remote URL
		service, err := credentials.ParseGitService(proj.GitRemote)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]string{
					"code":    "invalid_git_remote",
					"message": err.Error(),
				},
			})
			return
		}

		// Validate credentials
		result := credentials.ValidateCredentials(user, service)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}
}

// getBranchInfo returns git branch information for a goal's worktree
func getBranchInfo(vegaDir, goalID string, projects []string) *BranchInfo {
	if len(projects) == 0 {
		return nil
	}

	// Find worktree for this goal
	worktreePath, project := findWorktreeForGoal(vegaDir, goalID, projects)
	if worktreePath == "" {
		return nil
	}

	info := &BranchInfo{
		WorktreePath: worktreePath,
	}

	// Get current branch
	info.Branch = getCurrentBranch(worktreePath)

	// Get base branch from project config
	projectConfigPath := filepath.Join(vegaDir, "projects", project+".md")
	if content, err := os.ReadFile(projectConfigPath); err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			if strings.HasPrefix(line, "Base Branch:") {
				info.BaseBranch = strings.TrimSpace(strings.TrimPrefix(line, "Base Branch:"))
				break
			}
		}
	}
	if info.BaseBranch == "" {
		info.BaseBranch = "main"
	}

	// Get ahead/behind counts
	info.Ahead, info.Behind = getAheadBehind(worktreePath, info.BaseBranch)

	// Count uncommitted files
	info.UncommittedFiles = countUncommittedFiles(worktreePath)

	// Get last commit
	info.LastCommit, info.LastCommitMsg = getLastCommit(worktreePath)

	return info
}

// findWorktreeForGoal searches for a worktree matching the goal ID
func findWorktreeForGoal(vegaDir, goalID string, projects []string) (string, string) {
	workspacesDir := filepath.Join(vegaDir, "workspaces")
	goalPrefix := fmt.Sprintf("goal-%s-", goalID)

	// Check each project
	for _, project := range projects {
		projectPath := filepath.Join(workspacesDir, project)
		entries, err := os.ReadDir(projectPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() && strings.HasPrefix(entry.Name(), goalPrefix) {
				return filepath.Join(projectPath, entry.Name()), project
			}
		}
	}

	return "", ""
}

// getCurrentBranch returns the current branch name
func getCurrentBranch(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getAheadBehind returns ahead and behind counts relative to base branch
func getAheadBehind(repoPath, baseBranch string) (int, int) {
	// Try with origin/ prefix first
	cmd := exec.Command("git", "-C", repoPath, "rev-list", "--left-right", "--count", baseBranch+"...HEAD")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 2 {
		return 0, 0
	}

	behind := 0
	ahead := 0
	fmt.Sscanf(parts[0], "%d", &behind)
	fmt.Sscanf(parts[1], "%d", &ahead)

	return ahead, behind
}

// countUncommittedFiles returns the number of uncommitted files
func countUncommittedFiles(repoPath string) int {
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}

// getLastCommit returns the last commit hash and message
func getLastCommit(repoPath string) (string, string) {
	cmd := exec.Command("git", "-C", repoPath, "log", "-1", "--format=%H|%s")
	output, err := cmd.Output()
	if err != nil {
		return "", ""
	}

	parts := strings.SplitN(strings.TrimSpace(string(output)), "|", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}
