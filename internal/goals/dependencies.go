package goals

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DependencyType represents the type of dependency relationship
type DependencyType string

const (
	// DependencyBlocks means the dependency must be completed before this goal can start
	DependencyBlocks DependencyType = "blocks"
	// DependencyRelated means the goals are related but don't block each other
	DependencyRelated DependencyType = "related"
)

// Dependency represents a single dependency relationship
type Dependency struct {
	GoalID string         `json:"goal_id"`
	Type   DependencyType `json:"type"`
}

// DependencyInfo contains all dependency information for a goal
type DependencyInfo struct {
	// Dependencies: goals that this goal depends on (i.e., goals that block this one)
	Dependencies []Dependency `json:"dependencies"`
	// Dependents: goals that depend on this goal (i.e., goals blocked by this one)
	Dependents []Dependency `json:"dependents"`
}

// GoalMetadata stores metadata for a goal, including dependencies and hierarchy
type GoalMetadata struct {
	Dependencies   []Dependency `json:"dependencies,omitempty"`
	ParentID       string       `json:"parent_id,omitempty"`        // Parent goal ID for hierarchical goals
	NextChildIndex int          `json:"next_child_index,omitempty"` // Next index for child goals
}

// DependencyManager handles goal dependency operations
type DependencyManager struct {
	dir string // vega-missile directory
	mu  sync.RWMutex
}

// NewDependencyManager creates a new DependencyManager
func NewDependencyManager(dir string) *DependencyManager {
	return &DependencyManager{dir: dir}
}

// metadataFilePath returns the path to a goal's metadata file
func (m *DependencyManager) metadataFilePath(goalID string) (string, error) {
	// Check active, then iced, then history
	locations := []string{
		filepath.Join(m.dir, "goals", "active"),
		filepath.Join(m.dir, "goals", "iced"),
		filepath.Join(m.dir, "goals", "history"),
	}

	for _, loc := range locations {
		goalFile := filepath.Join(loc, goalID+".md")
		if _, err := os.Stat(goalFile); err == nil {
			return filepath.Join(loc, goalID+".metadata.json"), nil
		}
	}

	// Default to active for new goals
	return filepath.Join(m.dir, "goals", "active", goalID+".metadata.json"), nil
}

// readMetadata reads the metadata file for a goal
func (m *DependencyManager) readMetadata(goalID string) (*GoalMetadata, error) {
	path, err := m.metadataFilePath(goalID)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &GoalMetadata{}, nil // Empty metadata
	}
	if err != nil {
		return nil, err
	}

	var meta GoalMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing metadata: %w", err)
	}

	return &meta, nil
}

// writeMetadata writes the metadata file for a goal
func (m *DependencyManager) writeMetadata(goalID string, meta *GoalMetadata) error {
	path, err := m.metadataFilePath(goalID)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// goalExists checks if a goal exists in any status
func (m *DependencyManager) goalExists(goalID string) bool {
	locations := []string{
		filepath.Join(m.dir, "goals", "active", goalID+".md"),
		filepath.Join(m.dir, "goals", "iced", goalID+".md"),
		filepath.Join(m.dir, "goals", "history", goalID+".md"),
	}

	for _, path := range locations {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// detectCircularDependency checks if adding a dependency would create a cycle
func (m *DependencyManager) detectCircularDependency(goalID, dependsOnID string, visited map[string]bool) (bool, []string) {
	if visited[dependsOnID] {
		return goalID == dependsOnID, nil
	}

	visited[dependsOnID] = true

	// Check dependencies of dependsOnID
	meta, err := m.readMetadata(dependsOnID)
	if err != nil {
		return false, nil
	}

	for _, dep := range meta.Dependencies {
		if dep.GoalID == goalID {
			// Found a cycle: dependsOnID (directly or transitively) depends on goalID
			return true, []string{dependsOnID, goalID}
		}
		if isCycle, path := m.detectCircularDependency(goalID, dep.GoalID, visited); isCycle {
			return true, append([]string{dependsOnID}, path...)
		}
	}

	return false, nil
}

// AddDependency adds a dependency relationship between goals
func (m *DependencyManager) AddDependency(goalID, dependsOnID string, depType DependencyType) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate dependency type
	if depType != DependencyBlocks && depType != DependencyRelated {
		return fmt.Errorf("invalid dependency type: %s (must be 'blocks' or 'related')", depType)
	}

	// Validate goals exist
	if !m.goalExists(goalID) {
		return fmt.Errorf("goal not found: %s", goalID)
	}
	if !m.goalExists(dependsOnID) {
		return fmt.Errorf("dependency goal not found: %s", dependsOnID)
	}

	// Prevent self-dependency
	if goalID == dependsOnID {
		return fmt.Errorf("goal cannot depend on itself")
	}

	// Check for circular dependencies (only for blocking deps)
	if depType == DependencyBlocks {
		visited := make(map[string]bool)
		if isCycle, path := m.detectCircularDependency(goalID, dependsOnID, visited); isCycle {
			return fmt.Errorf("circular dependency detected: %v", path)
		}
	}

	// Read current metadata
	meta, err := m.readMetadata(goalID)
	if err != nil {
		return fmt.Errorf("reading metadata: %w", err)
	}

	// Check if dependency already exists
	for _, dep := range meta.Dependencies {
		if dep.GoalID == dependsOnID {
			if dep.Type == depType {
				return nil // Already exists with same type
			}
			// Update type
			dep.Type = depType
			return m.writeMetadata(goalID, meta)
		}
	}

	// Add new dependency
	meta.Dependencies = append(meta.Dependencies, Dependency{
		GoalID: dependsOnID,
		Type:   depType,
	})

	return m.writeMetadata(goalID, meta)
}

// RemoveDependency removes a dependency relationship between goals
func (m *DependencyManager) RemoveDependency(goalID, dependsOnID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	meta, err := m.readMetadata(goalID)
	if err != nil {
		return fmt.Errorf("reading metadata: %w", err)
	}

	// Find and remove the dependency
	found := false
	newDeps := make([]Dependency, 0, len(meta.Dependencies))
	for _, dep := range meta.Dependencies {
		if dep.GoalID == dependsOnID {
			found = true
			continue
		}
		newDeps = append(newDeps, dep)
	}

	if !found {
		return fmt.Errorf("dependency not found: %s -> %s", goalID, dependsOnID)
	}

	meta.Dependencies = newDeps
	return m.writeMetadata(goalID, meta)
}

// GetDependencies returns all dependency information for a goal
func (m *DependencyManager) GetDependencies(goalID string) (*DependencyInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !m.goalExists(goalID) {
		return nil, fmt.Errorf("goal not found: %s", goalID)
	}

	info := &DependencyInfo{
		Dependencies: []Dependency{},
		Dependents:   []Dependency{},
	}

	// Get direct dependencies
	meta, err := m.readMetadata(goalID)
	if err != nil {
		return nil, err
	}
	info.Dependencies = meta.Dependencies
	if info.Dependencies == nil {
		info.Dependencies = []Dependency{}
	}

	// Find dependents by scanning all goals
	info.Dependents, err = m.findDependents(goalID)
	if err != nil {
		return nil, err
	}

	return info, nil
}

// findDependents finds all goals that depend on the given goal
func (m *DependencyManager) findDependents(goalID string) ([]Dependency, error) {
	var dependents []Dependency

	// Scan all goal directories
	dirs := []string{
		filepath.Join(m.dir, "goals", "active"),
		filepath.Join(m.dir, "goals", "iced"),
		filepath.Join(m.dir, "goals", "history"),
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			if filepath.Ext(entry.Name()) != ".json" {
				continue
			}
			if entry.Name() == goalID+".metadata.json" {
				continue // Skip self
			}

			// Extract goal ID from filename (e.g., "abc123.metadata.json" -> "abc123")
			name := entry.Name()
			if len(name) < 15 { // ".metadata.json" = 14 chars + at least 1 char for ID
				continue
			}
			otherID := name[:len(name)-14] // Remove ".metadata.json"

			// Read metadata and check if it depends on goalID
			metaPath := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(metaPath)
			if err != nil {
				continue
			}

			var meta GoalMetadata
			if err := json.Unmarshal(data, &meta); err != nil {
				continue
			}

			for _, dep := range meta.Dependencies {
				if dep.GoalID == goalID {
					dependents = append(dependents, Dependency{
						GoalID: otherID,
						Type:   dep.Type,
					})
					break
				}
			}
		}
	}

	return dependents, nil
}

// GetBlockers returns all open goals that block the given goal
func (m *DependencyManager) GetBlockers(goalID string) ([]Goal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	meta, err := m.readMetadata(goalID)
	if err != nil {
		return nil, err
	}

	var blockers []Goal
	parser := NewParser(m.dir)

	for _, dep := range meta.Dependencies {
		if dep.Type != DependencyBlocks {
			continue
		}

		// Check if the blocking goal is still open (active or iced, not completed)
		detail, err := parser.ParseGoalDetail(dep.GoalID)
		if err != nil {
			continue // Goal might not exist anymore
		}

		// A goal is a blocker if it's not completed
		if detail.Status != "completed" {
			blockers = append(blockers, detail.Goal)
		}
	}

	return blockers, nil
}

// IsBlocked returns true if the goal has any open blocking dependencies
func (m *DependencyManager) IsBlocked(goalID string) bool {
	blockers, err := m.GetBlockers(goalID)
	if err != nil {
		return false
	}
	return len(blockers) > 0
}

// GetBlockerIDs returns IDs of goals blocking this goal (for API responses)
func (m *DependencyManager) GetBlockerIDs(goalID string) []string {
	blockers, err := m.GetBlockers(goalID)
	if err != nil {
		return nil
	}

	ids := make([]string, len(blockers))
	for i, b := range blockers {
		ids[i] = b.ID
	}
	return ids
}

// GetReadyGoals returns all goals that are ready to work on (no open blockers)
func (m *DependencyManager) GetReadyGoals(projectFilter string) ([]Goal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	parser := NewParser(m.dir)
	allGoals, err := parser.ParseRegistry()
	if err != nil {
		return nil, err
	}

	var ready []Goal
	sm := NewStateManager(m.dir)

	for _, g := range allGoals {
		// Filter by project if specified
		if projectFilter != "" {
			found := false
			for _, p := range g.Projects {
				if p == projectFilter {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Only include active goals (not completed or iced)
		if g.Status != "active" {
			continue
		}

		// Check state - only "pending" or "working" are considered ready
		state, err := sm.GetState(g.ID)
		if err != nil {
			continue
		}
		if state != StatePending && state != StateWorking {
			continue
		}

		// Check if blocked
		if m.IsBlocked(g.ID) {
			continue
		}

		ready = append(ready, g)
	}

	return ready, nil
}

// GoalWithBlockers extends Goal with blocker information
type GoalWithBlockers struct {
	Goal
	BlockedBy []string `json:"blocked_by,omitempty"`
	IsBlocked bool     `json:"is_blocked"`
}

// GetGoalsWithBlockerInfo returns all goals with their blocker information
func (m *DependencyManager) GetGoalsWithBlockerInfo(projectFilter string) ([]GoalWithBlockers, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	parser := NewParser(m.dir)
	allGoals, err := parser.ParseRegistry()
	if err != nil {
		return nil, err
	}

	var result []GoalWithBlockers

	for _, g := range allGoals {
		// Filter by project if specified
		if projectFilter != "" {
			found := false
			for _, p := range g.Projects {
				if p == projectFilter {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		blockerIDs := m.GetBlockerIDs(g.ID)
		result = append(result, GoalWithBlockers{
			Goal:      g,
			BlockedBy: blockerIDs,
			IsBlocked: len(blockerIDs) > 0,
		})
	}

	return result, nil
}

// BuildDependencyTree builds a tree structure showing all dependencies
type DependencyNode struct {
	GoalID   string            `json:"goal_id"`
	Title    string            `json:"title"`
	Status   string            `json:"status"`
	Type     DependencyType    `json:"type,omitempty"`
	Children []*DependencyNode `json:"children,omitempty"`
}

// GetDependencyTree builds a dependency tree for visualization
func (m *DependencyManager) GetDependencyTree(goalID string) (*DependencyNode, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.buildTree(goalID, DependencyBlocks, make(map[string]bool))
}

func (m *DependencyManager) buildTree(goalID string, depType DependencyType, visited map[string]bool) (*DependencyNode, error) {
	if visited[goalID] {
		return &DependencyNode{
			GoalID: goalID,
			Title:  "(circular reference)",
			Status: "error",
		}, nil
	}
	visited[goalID] = true

	parser := NewParser(m.dir)
	detail, err := parser.ParseGoalDetail(goalID)
	if err != nil {
		return nil, err
	}

	node := &DependencyNode{
		GoalID: goalID,
		Title:  detail.Title,
		Status: detail.Status,
		Type:   depType,
	}

	// Get dependencies
	meta, err := m.readMetadata(goalID)
	if err != nil {
		return node, nil
	}

	for _, dep := range meta.Dependencies {
		child, err := m.buildTree(dep.GoalID, dep.Type, visited)
		if err != nil {
			// Include placeholder for missing goals
			child = &DependencyNode{
				GoalID: dep.GoalID,
				Title:  "(not found)",
				Status: "error",
				Type:   dep.Type,
			}
		}
		node.Children = append(node.Children, child)
	}

	return node, nil
}

// MoveMetadataFile moves a goal's metadata file when the goal moves (e.g., archiving)
func (m *DependencyManager) MoveMetadataFile(goalID, fromDir, toDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fromPath := filepath.Join(m.dir, "goals", fromDir, goalID+".metadata.json")
	toPath := filepath.Join(m.dir, "goals", toDir, goalID+".metadata.json")

	// Only move if source exists
	if _, err := os.Stat(fromPath); os.IsNotExist(err) {
		return nil // Nothing to move
	}

	return os.Rename(fromPath, toPath)
}
