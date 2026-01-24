package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lasmarois/vega-hub/internal/hub"
)

// handleSSE handles GET /api/events - Server-Sent Events stream
func handleSSE(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Get flusher for streaming
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		// Subscribe to events
		events := h.Subscribe()
		defer h.Unsubscribe(events)

		// Send initial connection event
		fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
		flusher.Flush()

		// Send current pending questions
		for _, q := range h.GetPendingQuestions() {
			data, _ := json.Marshal(q)
			fmt.Fprintf(w, "event: question\ndata: %s\n\n", data)
		}
		flusher.Flush()

		// Stream events
		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}
				data, err := json.Marshal(event.Data)
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
				flusher.Flush()

			case <-r.Context().Done():
				return
			}
		}
	}
}
