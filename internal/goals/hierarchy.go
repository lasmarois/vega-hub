package goals

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// MaxHierarchyDepth is the maximum allowed nesting depth for goals
// Depth 0: root goal (e.g., goal-abc123)
// Depth 1: first child (e.g., goal-abc123.1)
// Depth 2: grandchild (e.g., goal-abc123.1.1)
// Depth 3: great-grandchild (e.g., goal-abc123.1.1.1) - MAX
const MaxHierarchyDepth = 3

// HierarchyManager handles parent-child relationships between goals
type HierarchyManager struct {
	dir string // vega-missile directory
	mu  sync.RWMutex
}

// NewHierarchyManager creates a new HierarchyManager
func NewHierarchyManager(dir string) *HierarchyManager {
	return &HierarchyManager{dir: dir}
}

// HierarchyMetadata stores hierarchy-specific metadata for a goal
type HierarchyMetadata struct {
	ParentID      string `json:"parent_id,omitempty"`       // Parent goal ID (empty for root goals)
	NextChildIndex int   `json:"next_child_index,omitempty"` // Next index for child goals (starts at 1)
}

// GoalWithHierarchy extends Goal with hierarchy information
type GoalWithHierarchy struct {
	Goal
	ParentID  string   `json:"parent_id,omitempty"`
	Children  []string `json:"children,omitempty"` // Child goal IDs
	Depth     int      `json:"depth"`               // Hierarchy depth (0 = root)
	IsBlocked bool     `json:"is_blocked,omitempty"`
}

// hierarchyMetadataPath returns the path to a goal's hierarchy metadata file
func (m *HierarchyManager) hierarchyMetadataPath(goalID string) (string, error) {
	// Find goal location
	locations := []string{
		filepath.Join(m.dir, "goals", "active"),
		filepath.Join(m.dir, "goals", "iced"),
		filepath.Join(m.dir, "goals", "history"),
	}

	for _, loc := range locations {
		goalFile := filepath.Join(loc, goalID+".md")
		if _, err := os.Stat(goalFile); err == nil {
			return filepath.Join(loc, goalID+".hierarchy.json"), nil
		}
	}

	// Default to active for new goals
	return filepath.Join(m.dir, "goals", "active", goalID+".hierarchy.json"), nil
}

// readHierarchyMetadata reads the hierarchy metadata for a goal
func (m *HierarchyManager) readHierarchyMetadata(goalID string) (*HierarchyMetadata, error) {
	path, err := m.hierarchyMetadataPath(goalID)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &HierarchyMetadata{}, nil
	}
	if err != nil {
		return nil, err
	}

	var meta HierarchyMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing hierarchy metadata: %w", err)
	}

	return &meta, nil
}

// writeHierarchyMetadata writes the hierarchy metadata for a goal
func (m *HierarchyManager) writeHierarchyMetadata(goalID string, meta *HierarchyMetadata) error {
	path, err := m.hierarchyMetadataPath(goalID)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetParentID returns the parent ID for a goal (empty string for root goals)
func (m *HierarchyManager) GetParentID(goalID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	meta, err := m.readHierarchyMetadata(goalID)
	if err != nil {
		return "", err
	}
	return meta.ParentID, nil
}

// SetParentID sets the parent ID for a goal
func (m *HierarchyManager) SetParentID(goalID, parentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	meta, err := m.readHierarchyMetadata(goalID)
	if err != nil {
		return err
	}

	meta.ParentID = parentID
	return m.writeHierarchyMetadata(goalID, meta)
}

// GetChildren returns all direct child goal IDs for a parent goal
func (m *HierarchyManager) GetChildren(parentID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var children []string

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
			if !strings.HasSuffix(entry.Name(), ".hierarchy.json") {
				continue
			}

			// Extract goal ID
			goalID := strings.TrimSuffix(entry.Name(), ".hierarchy.json")

			// Read hierarchy metadata
			metaPath := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(metaPath)
			if err != nil {
				continue
			}

			var meta HierarchyMetadata
			if err := json.Unmarshal(data, &meta); err != nil {
				continue
			}

			if meta.ParentID == parentID {
				children = append(children, goalID)
			}
		}
	}

	// Sort children by their hierarchical index
	sort.Slice(children, func(i, j int) bool {
		return compareHierarchicalIDs(children[i], children[j]) < 0
	})

	return children, nil
}

// GetHierarchyDepth returns the depth of a goal in the hierarchy (0 for root)
func (m *HierarchyManager) GetHierarchyDepth(goalID string) int {
	// Count dots in the goal ID
	// goal-abc123 -> 0
	// goal-abc123.1 -> 1
	// goal-abc123.1.2 -> 2
	parts := strings.Split(goalID, ".")
	return len(parts) - 1
}

// GenerateChildID generates the next hierarchical child ID for a parent
func (m *HierarchyManager) GenerateChildID(parentID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check depth limit
	parentDepth := m.GetHierarchyDepth(parentID)
	if parentDepth >= MaxHierarchyDepth {
		return "", fmt.Errorf("maximum hierarchy depth (%d) reached for goal %s", MaxHierarchyDepth, parentID)
	}

	// Get and increment the next child index
	meta, err := m.readHierarchyMetadata(parentID)
	if err != nil {
		return "", err
	}

	// Initialize index if needed (first child is .1)
	if meta.NextChildIndex == 0 {
		meta.NextChildIndex = 1
	}

	childIndex := meta.NextChildIndex
	meta.NextChildIndex++

	// Save updated metadata
	if err := m.writeHierarchyMetadata(parentID, meta); err != nil {
		return "", err
	}

	// Generate child ID
	childID := fmt.Sprintf("%s.%d", parentID, childIndex)
	return childID, nil
}

// ValidateParentForChildCreation validates that a parent goal can have children
func (m *HierarchyManager) ValidateParentForChildCreation(parentID string) error {
	// Check if parent exists
	parser := NewParser(m.dir)
	detail, err := parser.ParseGoalDetail(parentID)
	if err != nil {
		return fmt.Errorf("parent goal not found: %s", parentID)
	}

	// Check if parent is completed (can't add children to completed goals)
	if detail.Status == "completed" {
		return fmt.Errorf("cannot create child goal: parent goal %s is completed", parentID)
	}

	// Check depth limit
	depth := m.GetHierarchyDepth(parentID)
	if depth >= MaxHierarchyDepth {
		return fmt.Errorf("cannot create child: maximum hierarchy depth (%d) reached", MaxHierarchyDepth)
	}

	return nil
}

// CreateChildGoal creates the hierarchy relationship for a child goal
// Should be called after the goal file is created
func (m *HierarchyManager) CreateChildGoal(childID, parentID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Set parent ID on child
	childMeta := &HierarchyMetadata{
		ParentID: parentID,
	}
	if err := m.writeHierarchyMetadata(childID, childMeta); err != nil {
		return err
	}

	// Optionally create blocking dependency (child blocked if parent blocked)
	dm := NewDependencyManager(m.dir)
	// Note: We add "related" dependency, not "blocks"
	// The parent doesn't block the child - they can work in parallel
	// But we track the relationship for tree display
	// If you want parent to block child, change to DependencyBlocks

	// Actually, per spec: "child blocked if parent blocked" - meaning if parent has blockers,
	// child inherits them. This is more about blocker propagation than direct dependency.
	// For now, we'll track the relationship but not create automatic blocking.
	_ = dm // Unused for now

	return nil
}

// GetGoalWithHierarchy returns a goal with its hierarchy information
func (m *HierarchyManager) GetGoalWithHierarchy(goalID string) (*GoalWithHierarchy, error) {
	parser := NewParser(m.dir)
	detail, err := parser.ParseGoalDetail(goalID)
	if err != nil {
		return nil, err
	}

	parentID, _ := m.GetParentID(goalID)
	children, _ := m.GetChildren(goalID)

	// Check if blocked
	dm := NewDependencyManager(m.dir)
	isBlocked := dm.IsBlocked(goalID)

	return &GoalWithHierarchy{
		Goal:      detail.Goal,
		ParentID:  parentID,
		Children:  children,
		Depth:     m.GetHierarchyDepth(goalID),
		IsBlocked: isBlocked,
	}, nil
}

// GetAllGoalsWithHierarchy returns all goals with hierarchy information
func (m *HierarchyManager) GetAllGoalsWithHierarchy() ([]GoalWithHierarchy, error) {
	parser := NewParser(m.dir)
	allGoals, err := parser.ParseRegistry()
	if err != nil {
		return nil, err
	}

	var result []GoalWithHierarchy
	dm := NewDependencyManager(m.dir)

	for _, g := range allGoals {
		parentID, _ := m.GetParentID(g.ID)
		children, _ := m.GetChildren(g.ID)
		isBlocked := dm.IsBlocked(g.ID)

		result = append(result, GoalWithHierarchy{
			Goal:      g,
			ParentID:  parentID,
			Children:  children,
			Depth:     m.GetHierarchyDepth(g.ID),
			IsBlocked: isBlocked,
		})
	}

	return result, nil
}

// GetRootGoals returns all goals that have no parent
func (m *HierarchyManager) GetRootGoals() ([]GoalWithHierarchy, error) {
	allGoals, err := m.GetAllGoalsWithHierarchy()
	if err != nil {
		return nil, err
	}

	var roots []GoalWithHierarchy
	for _, g := range allGoals {
		if g.ParentID == "" {
			roots = append(roots, g)
		}
	}

	return roots, nil
}

// BuildGoalTree builds a tree structure starting from root goals
type GoalTreeNode struct {
	Goal     GoalWithHierarchy `json:"goal"`
	Children []*GoalTreeNode   `json:"children,omitempty"`
}

// BuildTree builds the complete goal tree
func (m *HierarchyManager) BuildTree() ([]*GoalTreeNode, error) {
	allGoals, err := m.GetAllGoalsWithHierarchy()
	if err != nil {
		return nil, err
	}

	// Build a map for quick lookup
	goalMap := make(map[string]*GoalWithHierarchy)
	for i := range allGoals {
		goalMap[allGoals[i].ID] = &allGoals[i]
	}

	// Build nodes map
	nodeMap := make(map[string]*GoalTreeNode)
	for id, g := range goalMap {
		nodeMap[id] = &GoalTreeNode{Goal: *g}
	}

	// Connect parents to children
	for id, node := range nodeMap {
		if parentID := node.Goal.ParentID; parentID != "" {
			if parentNode, ok := nodeMap[parentID]; ok {
				parentNode.Children = append(parentNode.Children, node)
			}
		}
		// Also add children from the Children list
		for _, childID := range node.Goal.Children {
			if childNode, ok := nodeMap[childID]; ok {
				// Check if already added
				found := false
				for _, c := range node.Children {
					if c.Goal.ID == childID {
						found = true
						break
					}
				}
				if !found {
					node.Children = append(node.Children, childNode)
				}
			}
		}
		_ = id
	}

	// Sort children by hierarchical ID
	for _, node := range nodeMap {
		sort.Slice(node.Children, func(i, j int) bool {
			return compareHierarchicalIDs(node.Children[i].Goal.ID, node.Children[j].Goal.ID) < 0
		})
	}

	// Collect root nodes (no parent)
	var roots []*GoalTreeNode
	for _, node := range nodeMap {
		if node.Goal.ParentID == "" {
			roots = append(roots, node)
		}
	}

	// Sort roots by ID
	sort.Slice(roots, func(i, j int) bool {
		return roots[i].Goal.ID < roots[j].Goal.ID
	})

	return roots, nil
}

// RenderTree renders the goal tree as ASCII art
func (m *HierarchyManager) RenderTree(projectFilter, statusFilter string) (string, error) {
	roots, err := m.BuildTree()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for i, root := range roots {
		// Apply filters
		if !nodeMatchesFilter(root, projectFilter, statusFilter) {
			continue
		}
		renderTreeNode(&sb, root, "", i == len(roots)-1, projectFilter, statusFilter)
	}

	return sb.String(), nil
}

// nodeMatchesFilter checks if a node or any of its descendants match the filter
func nodeMatchesFilter(node *GoalTreeNode, projectFilter, statusFilter string) bool {
	// Check status filter
	if statusFilter != "" && node.Goal.Status != statusFilter {
		// Check children
		for _, child := range node.Children {
			if nodeMatchesFilter(child, projectFilter, statusFilter) {
				return true
			}
		}
		return false
	}

	// Check project filter
	if projectFilter != "" {
		found := false
		for _, p := range node.Goal.Projects {
			if strings.EqualFold(p, projectFilter) {
				found = true
				break
			}
		}
		if !found {
			// Check children
			for _, child := range node.Children {
				if nodeMatchesFilter(child, projectFilter, statusFilter) {
					return true
				}
			}
			return false
		}
	}

	return true
}

func renderTreeNode(sb *strings.Builder, node *GoalTreeNode, prefix string, isLast bool, projectFilter, statusFilter string) {
	// Status indicator
	statusIcon := "○"
	switch node.Goal.Status {
	case "active":
		if node.Goal.IsBlocked {
			statusIcon = "⊘" // Blocked
		} else {
			statusIcon = "◉" // Active
		}
	case "completed":
		statusIcon = "✓"
	case "iced":
		statusIcon = "❄"
	}

	// Choose connector
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	if prefix == "" {
		connector = ""
	}

	// Truncate title
	title := node.Goal.Title
	if len(title) > 40 {
		title = title[:37] + "..."
	}

	// Format: goal-id  Title (status)
	sb.WriteString(fmt.Sprintf("%s%s%s %s  %s\n", prefix, connector, statusIcon, node.Goal.ID, title))

	// Prepare prefix for children
	childPrefix := prefix
	if prefix != "" {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	// Render children
	filteredChildren := filterChildren(node.Children, projectFilter, statusFilter)
	for i, child := range filteredChildren {
		isLastChild := i == len(filteredChildren)-1
		renderTreeNode(sb, child, childPrefix, isLastChild, projectFilter, statusFilter)
	}
}

func filterChildren(children []*GoalTreeNode, projectFilter, statusFilter string) []*GoalTreeNode {
	if projectFilter == "" && statusFilter == "" {
		return children
	}

	var filtered []*GoalTreeNode
	for _, child := range children {
		if nodeMatchesFilter(child, projectFilter, statusFilter) {
			filtered = append(filtered, child)
		}
	}
	return filtered
}

// compareHierarchicalIDs compares two hierarchical goal IDs
// Returns negative if a < b, positive if a > b, 0 if equal
func compareHierarchicalIDs(a, b string) int {
	// Split by dots
	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	// Compare base IDs first
	if partsA[0] != partsB[0] {
		if partsA[0] < partsB[0] {
			return -1
		}
		return 1
	}

	// Compare numeric suffixes
	minLen := len(partsA)
	if len(partsB) < minLen {
		minLen = len(partsB)
	}

	for i := 1; i < minLen; i++ {
		numA, errA := strconv.Atoi(partsA[i])
		numB, errB := strconv.Atoi(partsB[i])

		if errA != nil || errB != nil {
			// Fall back to string comparison
			if partsA[i] < partsB[i] {
				return -1
			}
			if partsA[i] > partsB[i] {
				return 1
			}
			continue
		}

		if numA != numB {
			return numA - numB
		}
	}

	// If all compared parts are equal, shorter one comes first
	return len(partsA) - len(partsB)
}

// IsHierarchicalID checks if a goal ID is hierarchical (has parent indicators)
func IsHierarchicalID(goalID string) bool {
	return strings.Contains(goalID, ".")
}

// GetRootID extracts the root goal ID from a hierarchical ID
// e.g., "goal-abc123.1.2" -> "goal-abc123"
func GetRootID(goalID string) string {
	parts := strings.Split(goalID, ".")
	return parts[0]
}

// GetParentIDFromHierarchical extracts the parent ID from a hierarchical ID
// e.g., "goal-abc123.1.2" -> "goal-abc123.1"
// Returns empty string for root IDs
func GetParentIDFromHierarchical(goalID string) string {
	lastDot := strings.LastIndex(goalID, ".")
	if lastDot == -1 {
		return ""
	}
	return goalID[:lastDot]
}

// MoveHierarchyFile moves a goal's hierarchy metadata file when archiving
func (m *HierarchyManager) MoveHierarchyFile(goalID, fromDir, toDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fromPath := filepath.Join(m.dir, "goals", fromDir, goalID+".hierarchy.json")
	toPath := filepath.Join(m.dir, "goals", toDir, goalID+".hierarchy.json")

	// Only move if source exists
	if _, err := os.Stat(fromPath); os.IsNotExist(err) {
		return nil
	}

	return os.Rename(fromPath, toPath)
}

// ParseHierarchicalID parses a hierarchical goal ID into its components
// e.g., "goal-abc123.1.2" -> ("goal-abc123", []int{1, 2})
func ParseHierarchicalID(goalID string) (baseID string, indices []int) {
	parts := strings.Split(goalID, ".")
	baseID = parts[0]

	for i := 1; i < len(parts); i++ {
		if idx, err := strconv.Atoi(parts[i]); err == nil {
			indices = append(indices, idx)
		}
	}

	return baseID, indices
}

// ValidateHierarchicalID validates that a hierarchical ID is well-formed
func ValidateHierarchicalID(goalID string) error {
	// Must start with a valid base ID (7 hex chars)
	parts := strings.Split(goalID, ".")

	// Validate base ID format (7 lowercase hex chars)
	basePattern := regexp.MustCompile(`^[a-f0-9]{7}$`)
	if !basePattern.MatchString(parts[0]) {
		return fmt.Errorf("invalid base goal ID format: %s", parts[0])
	}

	// Validate numeric indices
	for i := 1; i < len(parts); i++ {
		if _, err := strconv.Atoi(parts[i]); err != nil {
			return fmt.Errorf("invalid hierarchy index: %s", parts[i])
		}
	}

	// Check depth
	if len(parts)-1 > MaxHierarchyDepth {
		return fmt.Errorf("hierarchy depth %d exceeds maximum %d", len(parts)-1, MaxHierarchyDepth)
	}

	return nil
}
