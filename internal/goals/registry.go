package goals

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RegistryEntry represents one goal in the JSONL registry
type RegistryEntry struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Projects    []string `json:"projects"`
	Status      string   `json:"status"` // active, iced, completed
	Phase       string   `json:"phase"`
	ParentID    string   `json:"parent_id,omitempty"`
	BlockedBy   []string `json:"blocked_by,omitempty"`
	Reason      string   `json:"reason,omitempty"` // for iced goals
	CompletedAt string   `json:"completed_at,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// Registry handles JSONL registry operations
type Registry struct {
	path string
}

// ErrNotFound is returned when a goal is not found
var ErrNotFound = errors.New("goal not found")

// NewRegistry creates a new Registry instance
// path = vegaDir/goals/registry.jsonl
func NewRegistry(vegaDir string) *Registry {
	return &Registry{
		path: filepath.Join(vegaDir, "goals", "registry.jsonl"),
	}
}

// Load reads all entries from the JSONL file
// Reads line by line, skips empty lines
func (r *Registry) Load() ([]RegistryEntry, error) {
	file, err := os.Open(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []RegistryEntry{}, nil
		}
		return nil, fmt.Errorf("failed to open registry: %w", err)
	}
	defer file.Close()

	var entries []RegistryEntry
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		var entry RegistryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("failed to parse line %d: %w", lineNum, err)
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read registry: %w", err)
	}

	return entries, nil
}

// Save writes all entries to the JSONL file atomically
// Uses temp file + rename for atomic operation
func (r *Registry) Save(entries []RegistryEntry) error {
	// Ensure directory exists
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create temp file in same directory for atomic rename
	tmpFile, err := os.CreateTemp(dir, "registry-*.jsonl.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on error
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	// Write all entries
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to marshal entry %s: %w", entry.ID, err)
		}
		if _, err := tmpFile.Write(data); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to write entry %s: %w", entry.ID, err)
		}
		if _, err := tmpFile.WriteString("\n"); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	// Sync and close before rename
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, r.path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	success = true
	return nil
}

// Add appends a single entry to the file (fast path for new goals)
// Uses O_APPEND|O_CREATE|O_WRONLY for efficient append
func (r *Registry) Add(entry RegistryEntry) error {
	// Ensure directory exists
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.OpenFile(r.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open registry for append: %w", err)
	}
	defer file.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	// Write JSON line with newline
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// Update finds an entry by ID, applies the update function, and saves
func (r *Registry) Update(id string, fn func(*RegistryEntry)) error {
	entries, err := r.Load()
	if err != nil {
		return err
	}

	found := false
	for i := range entries {
		if entries[i].ID == id {
			fn(&entries[i])
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	return r.Save(entries)
}

// Get retrieves a single entry by ID
func (r *Registry) Get(id string) (*RegistryEntry, error) {
	entries, err := r.Load()
	if err != nil {
		return nil, err
	}

	for i := range entries {
		if entries[i].ID == id {
			return &entries[i], nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
}

// Delete removes an entry by ID
func (r *Registry) Delete(id string) error {
	entries, err := r.Load()
	if err != nil {
		return err
	}

	filtered := make([]RegistryEntry, 0, len(entries))
	found := false
	for _, entry := range entries {
		if entry.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, entry)
	}

	if !found {
		return fmt.Errorf("%w: %s", ErrNotFound, id)
	}

	return r.Save(filtered)
}

// List returns all entries matching the filter function
// If filter is nil, returns all entries
func (r *Registry) List(filter func(RegistryEntry) bool) ([]RegistryEntry, error) {
	entries, err := r.Load()
	if err != nil {
		return nil, err
	}

	if filter == nil {
		return entries, nil
	}

	var result []RegistryEntry
	for _, entry := range entries {
		if filter(entry) {
			result = append(result, entry)
		}
	}

	return result, nil
}
