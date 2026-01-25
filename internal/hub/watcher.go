package hub

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// StartFileWatcher starts watching the goals directory for changes
// and broadcasts SSE events when files are modified
func (h *Hub) StartFileWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Watch goals directories
	goalsDir := filepath.Join(h.dir, "goals")
	dirsToWatch := []string{
		goalsDir,
		filepath.Join(goalsDir, "active"),
		filepath.Join(goalsDir, "iced"),
		filepath.Join(goalsDir, "history"),
	}

	for _, dir := range dirsToWatch {
		if _, err := os.Stat(dir); err == nil {
			if err := watcher.Add(dir); err != nil {
				log.Printf("[WATCHER] Failed to watch %s: %v", dir, err)
			} else {
				log.Printf("[WATCHER] Watching %s", dir)
			}
		}
	}

	// Also watch REGISTRY.md specifically
	registryPath := filepath.Join(goalsDir, "REGISTRY.md")
	if _, err := os.Stat(registryPath); err == nil {
		if err := watcher.Add(registryPath); err != nil {
			log.Printf("[WATCHER] Failed to watch registry: %v", err)
		}
	}

	// Debounce map to avoid multiple events for same file
	lastEvent := make(map[string]time.Time)
	debounceWindow := 500 * time.Millisecond

	go func() {
		defer watcher.Close()

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// Only process write/create events on .md files
				if !isRelevantEvent(event) {
					continue
				}

				// Debounce - skip if we just saw this file
				now := time.Now()
				if last, exists := lastEvent[event.Name]; exists {
					if now.Sub(last) < debounceWindow {
						continue
					}
				}
				lastEvent[event.Name] = now

				// Determine what changed
				goalID := extractGoalID(event.Name)
				eventType := determineEventType(event.Name)

				log.Printf("[WATCHER] File changed: %s (goal: %s, type: %s)", event.Name, goalID, eventType)

				// Broadcast the change
				h.broadcast(Event{
					Type: eventType,
					Data: map[string]interface{}{
						"file":    event.Name,
						"goal_id": goalID,
						"action":  event.Op.String(),
					},
				})

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("[WATCHER] Error: %v", err)
			}
		}
	}()

	log.Printf("[WATCHER] File watcher started for %s", goalsDir)
	return nil
}

// isRelevantEvent checks if the event is for a markdown file we care about
func isRelevantEvent(event fsnotify.Event) bool {
	// Only care about write and create operations
	if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
		return false
	}

	// Only care about .md files
	if !strings.HasSuffix(event.Name, ".md") {
		return false
	}

	return true
}

// extractGoalID extracts the goal ID from a file path
// e.g., "/path/goals/active/abc1234.md" -> "abc1234"
func extractGoalID(path string) string {
	base := filepath.Base(path)

	// Handle REGISTRY.md
	if base == "REGISTRY.md" {
		return ""
	}

	// Remove .md extension
	if strings.HasSuffix(base, ".md") {
		return strings.TrimSuffix(base, ".md")
	}

	return ""
}

// determineEventType determines the SSE event type based on the file path
func determineEventType(path string) string {
	if strings.Contains(path, "REGISTRY.md") {
		return "registry_updated"
	}

	if strings.Contains(path, filepath.Join("goals", "active")) {
		return "goal_updated"
	}

	if strings.Contains(path, filepath.Join("goals", "iced")) {
		return "goal_iced"
	}

	if strings.Contains(path, filepath.Join("goals", "history")) {
		return "goal_completed"
	}

	return "file_changed"
}
