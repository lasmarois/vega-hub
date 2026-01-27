package goals

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PlanningFilesManager handles storage and retrieval of project planning files
// Planning files are stored in: goals/active/<goal-id>/project-plans/<project>/
type PlanningFilesManager struct {
	baseDir string
}

// NewPlanningFilesManager creates a new PlanningFilesManager
func NewPlanningFilesManager(baseDir string) *PlanningFilesManager {
	return &PlanningFilesManager{baseDir: baseDir}
}

// getPlanningDir returns the planning files directory for a goal/project
func (m *PlanningFilesManager) getPlanningDir(goalID, project string) string {
	return filepath.Join(m.baseDir, "goals", "active", goalID, "project-plans", project)
}

// getGoalPlanningDir returns the root planning directory for a goal
func (m *PlanningFilesManager) getGoalPlanningDir(goalID string) string {
	return filepath.Join(m.baseDir, "goals", "active", goalID, "project-plans")
}

// ensureGoalDir ensures the goal directory exists (creates if needed)
func (m *PlanningFilesManager) ensureGoalDir(goalID string) error {
	goalDir := filepath.Join(m.baseDir, "goals", "active", goalID)
	return os.MkdirAll(goalDir, 0755)
}

// SavePlanningFile saves a planning file for a project
// Files are stored in: goals/active/<goal-id>/project-plans/<project>/<filename>
func (m *PlanningFilesManager) SavePlanningFile(goalID, project, filename, content string) error {
	if goalID == "" {
		return fmt.Errorf("goal ID is required")
	}
	if project == "" {
		return fmt.Errorf("project is required")
	}
	if filename == "" {
		return fmt.Errorf("filename is required")
	}

	// Validate filename (prevent path traversal)
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || filename == ".." {
		return fmt.Errorf("invalid filename: %s", filename)
	}

	// Ensure goal directory exists first
	if err := m.ensureGoalDir(goalID); err != nil {
		return fmt.Errorf("creating goal directory: %w", err)
	}

	// Create planning directory if needed
	planningDir := m.getPlanningDir(goalID, project)
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		return fmt.Errorf("creating planning directory: %w", err)
	}

	// Write the file
	filePath := filepath.Join(planningDir, filename)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing planning file: %w", err)
	}

	return nil
}

// GetPlanningFile retrieves a planning file's content
func (m *PlanningFilesManager) GetPlanningFile(goalID, project, filename string) (string, error) {
	if goalID == "" {
		return "", fmt.Errorf("goal ID is required")
	}
	if project == "" {
		return "", fmt.Errorf("project is required")
	}
	if filename == "" {
		return "", fmt.Errorf("filename is required")
	}

	// Validate filename (prevent path traversal)
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || filename == ".." {
		return "", fmt.Errorf("invalid filename: %s", filename)
	}

	filePath := filepath.Join(m.getPlanningDir(goalID, project), filename)
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("planning file not found: %s/%s", project, filename)
		}
		return "", fmt.Errorf("reading planning file: %w", err)
	}

	return string(content), nil
}

// ListPlanningFiles returns a map of project -> list of files for a goal
// Returns: {"project1": ["task_plan.md", "findings.md"], "project2": ["task_plan.md"]}
func (m *PlanningFilesManager) ListPlanningFiles(goalID string) (map[string][]string, error) {
	if goalID == "" {
		return nil, fmt.Errorf("goal ID is required")
	}

	result := make(map[string][]string)

	planningRoot := m.getGoalPlanningDir(goalID)
	
	// Check if planning directory exists
	if _, err := os.Stat(planningRoot); os.IsNotExist(err) {
		return result, nil // Return empty map if no planning files exist
	}

	// List project directories
	projectEntries, err := os.ReadDir(planningRoot)
	if err != nil {
		return nil, fmt.Errorf("reading planning directory: %w", err)
	}

	for _, projectEntry := range projectEntries {
		if !projectEntry.IsDir() {
			continue
		}

		projectName := projectEntry.Name()
		projectDir := filepath.Join(planningRoot, projectName)

		// List files in project directory
		fileEntries, err := os.ReadDir(projectDir)
		if err != nil {
			continue // Skip projects we can't read
		}

		var files []string
		for _, fileEntry := range fileEntries {
			if fileEntry.IsDir() {
				continue
			}
			files = append(files, fileEntry.Name())
		}

		// Sort files for consistent ordering
		sort.Strings(files)

		if len(files) > 0 {
			result[projectName] = files
		}
	}

	return result, nil
}

// GetAllPlanningFiles returns all planning file contents for a goal
// Returns: {"project1": {"task_plan.md": "content...", "findings.md": "..."}}
func (m *PlanningFilesManager) GetAllPlanningFiles(goalID string) (map[string]map[string]string, error) {
	if goalID == "" {
		return nil, fmt.Errorf("goal ID is required")
	}

	result := make(map[string]map[string]string)

	// Get the file listing first
	fileList, err := m.ListPlanningFiles(goalID)
	if err != nil {
		return nil, err
	}

	// Read content for each file
	for project, files := range fileList {
		result[project] = make(map[string]string)
		
		for _, filename := range files {
			content, err := m.GetPlanningFile(goalID, project, filename)
			if err != nil {
				// Include error message as content if we can't read the file
				result[project][filename] = fmt.Sprintf("Error reading file: %v", err)
				continue
			}
			result[project][filename] = content
		}
	}

	return result, nil
}

// DeletePlanningFile removes a planning file
func (m *PlanningFilesManager) DeletePlanningFile(goalID, project, filename string) error {
	if goalID == "" {
		return fmt.Errorf("goal ID is required")
	}
	if project == "" {
		return fmt.Errorf("project is required")
	}
	if filename == "" {
		return fmt.Errorf("filename is required")
	}

	// Validate filename (prevent path traversal)
	if strings.Contains(filename, "/") || strings.Contains(filename, "\\") || filename == ".." {
		return fmt.Errorf("invalid filename: %s", filename)
	}

	filePath := filepath.Join(m.getPlanningDir(goalID, project), filename)
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("planning file not found: %s/%s", project, filename)
		}
		return fmt.Errorf("deleting planning file: %w", err)
	}

	// Clean up empty directories
	m.cleanupEmptyDirs(goalID, project)

	return nil
}

// cleanupEmptyDirs removes empty project and project-plans directories
func (m *PlanningFilesManager) cleanupEmptyDirs(goalID, project string) {
	// Try to remove empty project directory
	projectDir := m.getPlanningDir(goalID, project)
	entries, err := os.ReadDir(projectDir)
	if err == nil && len(entries) == 0 {
		os.Remove(projectDir)
	}

	// Try to remove empty project-plans directory
	planningRoot := m.getGoalPlanningDir(goalID)
	entries, err = os.ReadDir(planningRoot)
	if err == nil && len(entries) == 0 {
		os.Remove(planningRoot)
	}
}
