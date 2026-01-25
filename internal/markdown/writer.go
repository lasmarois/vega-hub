package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Writer handles writing Q&A to goal markdown files
type Writer struct {
	dir string
	mu  sync.Mutex
}

// NewWriter creates a new markdown writer
func NewWriter(dir string) *Writer {
	return &Writer{dir: dir}
}

// WriteQA appends a Q&A entry to the goal's markdown file
func (w *Writer) WriteQA(goalID string, sessionID, question, answer string) error {
	if w.dir == "" {
		return nil // No directory configured, skip
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Find the goal file
	goalFile := filepath.Join(w.dir, "goals", "active", goalID+".md")

	// Check if file exists
	if _, err := os.Stat(goalFile); os.IsNotExist(err) {
		return fmt.Errorf("goal file not found: %s", goalFile)
	}

	// Read existing content
	content, err := os.ReadFile(goalFile)
	if err != nil {
		return fmt.Errorf("failed to read goal file: %w", err)
	}

	// Format the Q&A entry
	timestamp := time.Now().Format("2006-01-02 15:04")
	entry := fmt.Sprintf(`
---

**%s** | Goal #%s | Executor session %s
**Q:** %s
**A:** %s
`, timestamp, goalID, sessionID, question, answer)

	// Check if "## Executor Questions" section exists
	contentStr := string(content)
	sectionHeader := "## Executor Questions"

	var newContent string
	if idx := findSection(contentStr, sectionHeader); idx != -1 {
		// Append to existing section
		newContent = contentStr[:idx+len(sectionHeader)] + entry + contentStr[idx+len(sectionHeader):]
	} else {
		// Add new section at the end
		newContent = contentStr + "\n" + sectionHeader + entry
	}

	// Write back
	if err := os.WriteFile(goalFile, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write goal file: %w", err)
	}

	return nil
}

// WriteExecutorEvent appends an executor lifecycle event to the goal's markdown file
func (w *Writer) WriteExecutorEvent(goalID string, sessionID, eventType, detail string) error {
	if w.dir == "" {
		return nil // No directory configured, skip
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Find the goal file
	goalFile := filepath.Join(w.dir, "goals", "active", goalID+".md")

	// Check if file exists
	if _, err := os.Stat(goalFile); os.IsNotExist(err) {
		return fmt.Errorf("goal file not found: %s", goalFile)
	}

	// Read existing content
	content, err := os.ReadFile(goalFile)
	if err != nil {
		return fmt.Errorf("failed to read goal file: %w", err)
	}

	// Format the event entry
	timestamp := time.Now().Format("2006-01-02 15:04")
	entry := fmt.Sprintf("\n**%s** | %s | Session %s", timestamp, eventType, sessionID)
	if detail != "" {
		entry += " | " + detail
	}

	// Check if "## Executor Activity" section exists
	contentStr := string(content)
	sectionHeader := "## Executor Activity"

	var newContent string
	if idx := findSection(contentStr, sectionHeader); idx != -1 {
		// Find end of header line
		endOfHeader := idx + len(sectionHeader)
		// Append after header
		newContent = contentStr[:endOfHeader] + entry + contentStr[endOfHeader:]
	} else {
		// Add new section at the end
		newContent = contentStr + "\n\n" + sectionHeader + entry
	}

	// Write back
	if err := os.WriteFile(goalFile, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write goal file: %w", err)
	}

	return nil
}

// findSection returns the index of a section header, or -1 if not found
func findSection(content, header string) int {
	for i := 0; i <= len(content)-len(header); i++ {
		if content[i:i+len(header)] == header {
			return i
		}
	}
	return -1
}
