package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/lasmarois/vega-hub/internal/hub"
)

// RegisterRoutes sets up all API routes
func RegisterRoutes(mux *http.ServeMux, h *hub.Hub) {
	mux.HandleFunc("/api/ask", corsMiddleware(handleAsk(h)))
	mux.HandleFunc("/api/answer/", corsMiddleware(handleAnswer(h)))
	mux.HandleFunc("/api/questions", corsMiddleware(handleQuestions(h)))
	mux.HandleFunc("/api/events", handleSSE(h))
	mux.HandleFunc("/api/health", handleHealth())
}

// AskRequest is the request body for POST /api/ask
type AskRequest struct {
	GoalID    int          `json:"goal_id"`
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
