package hub

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtractGoalID(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/goals/active/abc1234.md", "abc1234"},
		{"/path/goals/active/4fd584d.md", "4fd584d"},
		{"/path/goals/iced/xyz9999.md", "xyz9999"},
		{"/path/goals/REGISTRY.md", ""},
		{"/path/goals/active/14.md", "14"},
		{"abc1234.md", "abc1234"},
	}

	for _, tt := range tests {
		result := extractGoalID(tt.path)
		if result != tt.expected {
			t.Errorf("extractGoalID(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestDetermineEventType(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/goals/REGISTRY.md", "registry_updated"},
		{"/path/goals/active/abc1234.md", "goal_updated"},
		{"/path/goals/iced/abc1234.md", "goal_iced"},
		{"/path/goals/history/abc1234.md", "goal_completed"},
		{"/path/other/file.md", "file_changed"},
	}

	for _, tt := range tests {
		result := determineEventType(tt.path)
		if result != tt.expected {
			t.Errorf("determineEventType(%q) = %q, want %q", tt.path, result, tt.expected)
		}
	}
}

func TestStartFileWatcher(t *testing.T) {
	dir := t.TempDir()

	// Create goals directory structure
	goalsDir := filepath.Join(dir, "goals")
	os.MkdirAll(filepath.Join(goalsDir, "active"), 0755)
	os.MkdirAll(filepath.Join(goalsDir, "iced"), 0755)
	os.MkdirAll(filepath.Join(goalsDir, "history"), 0755)

	// Create initial registry
	registryPath := filepath.Join(goalsDir, "REGISTRY.md")
	os.WriteFile(registryPath, []byte("# Registry\n"), 0644)

	h := New(dir)

	// Subscribe to events
	eventCh := h.Subscribe()
	defer h.Unsubscribe(eventCh)

	// Start watcher
	err := h.StartFileWatcher()
	if err != nil {
		t.Fatalf("StartFileWatcher failed: %v", err)
	}

	// Give watcher time to start
	time.Sleep(100 * time.Millisecond)

	// Modify a file
	goalFile := filepath.Join(goalsDir, "active", "test123.md")
	os.WriteFile(goalFile, []byte("# Goal\n"), 0644)

	// Wait for event (with timeout)
	select {
	case event := <-eventCh:
		if event.Type != "goal_updated" {
			t.Errorf("expected event type 'goal_updated', got '%s'", event.Type)
		}
		data, ok := event.Data.(map[string]interface{})
		if !ok {
			t.Fatal("expected event data to be map")
		}
		if data["goal_id"] != "test123" {
			t.Errorf("expected goal_id 'test123', got '%v'", data["goal_id"])
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for file change event")
	}
}

func TestStartFileWatcher_RegistryChange(t *testing.T) {
	dir := t.TempDir()

	// Create goals directory structure
	goalsDir := filepath.Join(dir, "goals")
	os.MkdirAll(goalsDir, 0755)

	// Create initial registry
	registryPath := filepath.Join(goalsDir, "REGISTRY.md")
	os.WriteFile(registryPath, []byte("# Registry\n"), 0644)

	h := New(dir)
	eventCh := h.Subscribe()
	defer h.Unsubscribe(eventCh)

	err := h.StartFileWatcher()
	if err != nil {
		t.Fatalf("StartFileWatcher failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Modify registry
	os.WriteFile(registryPath, []byte("# Registry Updated\n"), 0644)

	select {
	case event := <-eventCh:
		if event.Type != "registry_updated" {
			t.Errorf("expected event type 'registry_updated', got '%s'", event.Type)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for registry change event")
	}
}

func TestStartFileWatcher_Debounce(t *testing.T) {
	dir := t.TempDir()

	goalsDir := filepath.Join(dir, "goals")
	os.MkdirAll(filepath.Join(goalsDir, "active"), 0755)

	h := New(dir)
	eventCh := h.Subscribe()
	defer h.Unsubscribe(eventCh)

	err := h.StartFileWatcher()
	if err != nil {
		t.Fatalf("StartFileWatcher failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Write to same file rapidly
	goalFile := filepath.Join(goalsDir, "active", "debounce.md")
	for i := 0; i < 5; i++ {
		os.WriteFile(goalFile, []byte("# Update\n"), 0644)
		time.Sleep(10 * time.Millisecond)
	}

	// Should only receive one event due to debouncing
	eventCount := 0
	timeout := time.After(1 * time.Second)

loop:
	for {
		select {
		case <-eventCh:
			eventCount++
		case <-timeout:
			break loop
		}
	}

	// Due to debouncing, we should have fewer events than writes
	if eventCount > 2 {
		t.Errorf("expected <= 2 events due to debouncing, got %d", eventCount)
	}
}

func TestStartFileWatcher_IgnoresNonMarkdown(t *testing.T) {
	dir := t.TempDir()

	goalsDir := filepath.Join(dir, "goals")
	os.MkdirAll(filepath.Join(goalsDir, "active"), 0755)

	h := New(dir)
	eventCh := h.Subscribe()
	defer h.Unsubscribe(eventCh)

	err := h.StartFileWatcher()
	if err != nil {
		t.Fatalf("StartFileWatcher failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Write non-markdown file
	os.WriteFile(filepath.Join(goalsDir, "active", "test.txt"), []byte("text\n"), 0644)

	// Should not receive event
	select {
	case event := <-eventCh:
		t.Errorf("should not receive event for .txt file, got %s", event.Type)
	case <-time.After(500 * time.Millisecond):
		// Expected - no event
	}
}
