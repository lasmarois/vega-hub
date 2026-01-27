package hub

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/lasmarois/vega-hub/internal/goals"
)

// HealthStatus represents the overall health status
type HealthStatus string

const (
	HealthHealthy  HealthStatus = "healthy"
	HealthDegraded HealthStatus = "degraded"
	HealthUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a single health check result
type HealthCheck struct {
	Status  HealthStatus `json:"status"`
	Message string       `json:"message,omitempty"`
	Details interface{}  `json:"details,omitempty"`
}

// HealthIssue represents a health issue
type HealthIssue struct {
	Severity string `json:"severity"` // "warning", "error", "critical"
	Check    string `json:"check"`
	Message  string `json:"message"`
}

// HealthResult contains the complete health check results
type HealthResult struct {
	Status    HealthStatus           `json:"status"`
	Checks    map[string]HealthCheck `json:"checks"`
	Issues    []HealthIssue          `json:"issues,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// HealthChecker performs health checks on the vega-hub system
type HealthChecker struct {
	vegaDir      string
	stateManager *goals.StateManager
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(vegaDir string) *HealthChecker {
	return &HealthChecker{
		vegaDir:      vegaDir,
		stateManager: goals.NewStateManager(vegaDir),
	}
}

// RunAllChecks performs all health checks and returns the results
func (h *HealthChecker) RunAllChecks() *HealthResult {
	result := &HealthResult{
		Status:    HealthHealthy,
		Checks:    make(map[string]HealthCheck),
		Issues:    []HealthIssue{},
		Timestamp: time.Now(),
	}

	// Check 1: Worktree bases
	worktreeCheck := h.CheckWorktreeBases()
	result.Checks["worktree_bases"] = worktreeCheck
	if worktreeCheck.Status != HealthHealthy {
		result.Issues = append(result.Issues, HealthIssue{
			Severity: "warning",
			Check:    "worktree_bases",
			Message:  worktreeCheck.Message,
		})
	}

	// Check 2: Stuck goals
	stuckCheck := h.CheckStuckGoals()
	result.Checks["stuck_goals"] = stuckCheck
	if stuckCheck.Status != HealthHealthy {
		result.Issues = append(result.Issues, HealthIssue{
			Severity: "warning",
			Check:    "stuck_goals",
			Message:  stuckCheck.Message,
		})
	}

	// Check 3: Disk space
	diskCheck := h.CheckDiskSpace()
	result.Checks["disk_space"] = diskCheck
	if diskCheck.Status == HealthUnhealthy {
		result.Issues = append(result.Issues, HealthIssue{
			Severity: "critical",
			Check:    "disk_space",
			Message:  diskCheck.Message,
		})
	} else if diskCheck.Status == HealthDegraded {
		result.Issues = append(result.Issues, HealthIssue{
			Severity: "warning",
			Check:    "disk_space",
			Message:  diskCheck.Message,
		})
	}

	// Check 4: Git credentials
	credCheck := h.CheckGitCredentials()
	result.Checks["git_credentials"] = credCheck
	if credCheck.Status != HealthHealthy {
		result.Issues = append(result.Issues, HealthIssue{
			Severity: "error",
			Check:    "git_credentials",
			Message:  credCheck.Message,
		})
	}

	// Check 5: Orphaned worktrees
	orphanCheck := h.CheckOrphanedWorktrees()
	result.Checks["orphaned_worktrees"] = orphanCheck
	if orphanCheck.Status != HealthHealthy {
		result.Issues = append(result.Issues, HealthIssue{
			Severity: "warning",
			Check:    "orphaned_worktrees",
			Message:  orphanCheck.Message,
		})
	}

	// Check 6: Active locks
	lockCheck := h.CheckActiveLocks()
	result.Checks["active_locks"] = lockCheck
	if lockCheck.Status != HealthHealthy {
		result.Issues = append(result.Issues, HealthIssue{
			Severity: "warning",
			Check:    "active_locks",
			Message:  lockCheck.Message,
		})
	}

	// Determine overall status
	for _, check := range result.Checks {
		if check.Status == HealthUnhealthy {
			result.Status = HealthUnhealthy
			break
		}
		if check.Status == HealthDegraded && result.Status == HealthHealthy {
			result.Status = HealthDegraded
		}
	}

	return result
}

// CheckWorktreeBases verifies all worktree bases are clean and fetchable
func (h *HealthChecker) CheckWorktreeBases() HealthCheck {
	workspacesDir := filepath.Join(h.vegaDir, "workspaces")
	entries, err := os.ReadDir(workspacesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return HealthCheck{Status: HealthHealthy, Message: "No workspaces configured"}
		}
		return HealthCheck{Status: HealthDegraded, Message: fmt.Sprintf("Cannot read workspaces: %v", err)}
	}

	var issues []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectName := entry.Name()
		worktreeBase := filepath.Join(workspacesDir, projectName, "worktree-base")

		if _, err := os.Stat(worktreeBase); os.IsNotExist(err) {
			continue
		}

		// Check if clean
		cmd := exec.Command("git", "-C", worktreeBase, "status", "--porcelain")
		output, _ := cmd.Output()
		if len(strings.TrimSpace(string(output))) > 0 {
			issues = append(issues, fmt.Sprintf("%s: has uncommitted changes", projectName))
		}

		// Check if fetchable
		cmd = exec.Command("git", "-C", worktreeBase, "fetch", "--dry-run", "origin")
		if err := cmd.Run(); err != nil {
			issues = append(issues, fmt.Sprintf("%s: cannot fetch from origin", projectName))
		}
	}

	if len(issues) > 0 {
		return HealthCheck{
			Status:  HealthDegraded,
			Message: fmt.Sprintf("%d worktree base issues", len(issues)),
			Details: issues,
		}
	}

	return HealthCheck{Status: HealthHealthy, Message: "All worktree bases healthy"}
}

// CheckStuckGoals finds goals stuck in intermediate states
func (h *HealthChecker) CheckStuckGoals() HealthCheck {
	stuckGoals, err := h.stateManager.StuckGoals(1 * time.Hour)
	if err != nil {
		return HealthCheck{Status: HealthHealthy, Message: "No stuck goals (unable to check)"}
	}

	if len(stuckGoals) > 0 {
		return HealthCheck{
			Status:  HealthDegraded,
			Message: fmt.Sprintf("%d goals stuck in intermediate states", len(stuckGoals)),
			Details: stuckGoals,
		}
	}

	return HealthCheck{Status: HealthHealthy, Message: "No stuck goals"}
}

// CheckDiskSpace verifies sufficient disk space
func (h *HealthChecker) CheckDiskSpace() HealthCheck {
	checker := NewPreflightChecker(h.vegaDir, "", "")
	checker.SetMinDiskMB(1024) // 1GB minimum

	check := checker.CheckDiskSpace()
	if !check.Passed {
		return HealthCheck{
			Status:  HealthUnhealthy,
			Message: check.Error,
			Details: map[string]int64{"available_mb": check.AvailMB, "required_mb": check.RequiredMB},
		}
	}

	// Warn if less than 5GB
	if check.AvailMB < 5120 {
		return HealthCheck{
			Status:  HealthDegraded,
			Message: fmt.Sprintf("Low disk space: %d MB available", check.AvailMB),
			Details: map[string]int64{"available_mb": check.AvailMB},
		}
	}

	return HealthCheck{
		Status:  HealthHealthy,
		Message: fmt.Sprintf("%d MB available", check.AvailMB),
		Details: map[string]int64{"available_mb": check.AvailMB},
	}
}

// CheckGitCredentials verifies git credentials for all projects
func (h *HealthChecker) CheckGitCredentials() HealthCheck {
	workspacesDir := filepath.Join(h.vegaDir, "workspaces")
	entries, err := os.ReadDir(workspacesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return HealthCheck{Status: HealthHealthy, Message: "No workspaces configured"}
		}
		return HealthCheck{Status: HealthDegraded, Message: fmt.Sprintf("Cannot read workspaces: %v", err)}
	}

	var issues []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectName := entry.Name()
		worktreeBase := filepath.Join(workspacesDir, projectName, "worktree-base")

		if _, err := os.Stat(worktreeBase); os.IsNotExist(err) {
			continue
		}

		// Check if we can access the remote
		cmd := exec.Command("git", "-C", worktreeBase, "ls-remote", "--exit-code", "origin")
		if err := cmd.Run(); err != nil {
			issues = append(issues, fmt.Sprintf("%s: cannot access remote", projectName))
		}
	}

	if len(issues) > 0 {
		return HealthCheck{
			Status:  HealthDegraded,
			Message: fmt.Sprintf("Credential issues for %d projects", len(issues)),
			Details: issues,
		}
	}

	return HealthCheck{Status: HealthHealthy, Message: "Git credentials valid"}
}

// CheckOrphanedWorktrees finds orphaned worktrees
func (h *HealthChecker) CheckOrphanedWorktrees() HealthCheck {
	manager := NewCleanupManager(h.vegaDir)
	result := manager.FindOrphanedWorktrees()

	if len(result.OrphanedWorktrees) > 0 {
		return HealthCheck{
			Status:  HealthDegraded,
			Message: fmt.Sprintf("%d orphaned worktrees found", len(result.OrphanedWorktrees)),
			Details: result.OrphanedWorktrees,
		}
	}

	return HealthCheck{Status: HealthHealthy, Message: "No orphaned worktrees"}
}

// CheckActiveLocks checks for stale locks
func (h *HealthChecker) CheckActiveLocks() HealthCheck {
	manager := NewLockManager(h.vegaDir)
	locks, err := manager.ListLocks()
	if err != nil {
		return HealthCheck{Status: HealthHealthy, Message: "No locks directory"}
	}

	var staleLocks []string
	for _, lock := range locks {
		if lock.IsStale() {
			staleLocks = append(staleLocks, fmt.Sprintf("%s/%s", lock.LockType, lock.Resource))
		}
	}

	if len(staleLocks) > 0 {
		return HealthCheck{
			Status:  HealthDegraded,
			Message: fmt.Sprintf("%d stale locks found", len(staleLocks)),
			Details: staleLocks,
		}
	}

	if len(locks) > 0 {
		return HealthCheck{
			Status:  HealthHealthy,
			Message: fmt.Sprintf("%d active locks", len(locks)),
		}
	}

	return HealthCheck{Status: HealthHealthy, Message: "No active locks"}
}
