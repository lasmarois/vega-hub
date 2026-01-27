package hub

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// PreflightCheck represents a single pre-flight check result
type PreflightCheck struct {
	Passed     bool   `json:"passed"`
	Error      string `json:"error,omitempty"`
	Behind     int    `json:"behind,omitempty"`
	AvailMB    int64  `json:"available_mb,omitempty"`
	RequiredMB int64  `json:"required_mb,omitempty"`
	Existing   string `json:"existing,omitempty"`
}

// PreflightResult contains the results of all pre-flight checks
type PreflightResult struct {
	Ready          bool                       `json:"ready"`
	Checks         map[string]PreflightCheck  `json:"checks"`
	BlockingIssues []string                   `json:"blocking_issues"`
	FixCommands    []string                   `json:"fix_commands"`
}

// PreflightChecker validates environment before goal creation
type PreflightChecker struct {
	worktreeBase string
	baseBranch   string
	branchName   string
	minDiskMB    int64
}

// NewPreflightChecker creates a new pre-flight checker
func NewPreflightChecker(worktreeBase, baseBranch, branchName string) *PreflightChecker {
	return &PreflightChecker{
		worktreeBase: worktreeBase,
		baseBranch:   baseBranch,
		branchName:   branchName,
		minDiskMB:    1024, // 1GB default
	}
}

// SetMinDiskMB sets the minimum required disk space in MB
func (p *PreflightChecker) SetMinDiskMB(mb int64) {
	p.minDiskMB = mb
}

// RunAll executes all pre-flight checks and returns a consolidated result
func (p *PreflightChecker) RunAll() *PreflightResult {
	result := &PreflightResult{
		Ready:          true,
		Checks:         make(map[string]PreflightCheck),
		BlockingIssues: []string{},
		FixCommands:    []string{},
	}

	// Check worktree cleanliness
	cleanCheck := p.CheckWorktreeClean()
	result.Checks["worktree_clean"] = cleanCheck
	if !cleanCheck.Passed {
		result.Ready = false
		result.BlockingIssues = append(result.BlockingIssues, "worktree_clean")
		result.FixCommands = append(result.FixCommands,
			fmt.Sprintf("cd %s && git stash", p.worktreeBase),
			fmt.Sprintf("cd %s && git checkout -- .", p.worktreeBase),
		)
	}

	// Check worktree sync with remote
	syncCheck := p.CheckWorktreeSync()
	result.Checks["worktree_synced"] = syncCheck
	if !syncCheck.Passed {
		result.Ready = false
		result.BlockingIssues = append(result.BlockingIssues, "worktree_synced")
		result.FixCommands = append(result.FixCommands,
			fmt.Sprintf("cd %s && git pull origin %s", p.worktreeBase, p.baseBranch),
		)
	}

	// Check disk space
	diskCheck := p.CheckDiskSpace()
	result.Checks["disk_space"] = diskCheck
	if !diskCheck.Passed {
		result.Ready = false
		result.BlockingIssues = append(result.BlockingIssues, "disk_space")
		result.FixCommands = append(result.FixCommands,
			"Free up disk space (at least 1GB required)",
		)
	}

	// Check git credentials
	credCheck := p.CheckCredentials()
	result.Checks["credentials"] = credCheck
	if !credCheck.Passed {
		result.Ready = false
		result.BlockingIssues = append(result.BlockingIssues, "credentials")
		result.FixCommands = append(result.FixCommands,
			"Configure git credentials: gh auth login OR glab auth login",
		)
	}

	// Check branch availability
	if p.branchName != "" {
		branchCheck := p.CheckBranchAvailable()
		result.Checks["branch_available"] = branchCheck
		if !branchCheck.Passed {
			result.Ready = false
			result.BlockingIssues = append(result.BlockingIssues, "branch_available")
			result.FixCommands = append(result.FixCommands,
				fmt.Sprintf("git branch -d %s (delete existing branch)", p.branchName),
			)
		}
	}

	// Check for in-progress operations (rebase/merge)
	opCheck := p.CheckNoInProgressOps()
	result.Checks["no_in_progress_ops"] = opCheck
	if !opCheck.Passed {
		result.Ready = false
		result.BlockingIssues = append(result.BlockingIssues, "no_in_progress_ops")
		result.FixCommands = append(result.FixCommands,
			fmt.Sprintf("cd %s && git rebase --abort OR git merge --abort", p.worktreeBase),
		)
	}

	return result
}

// CheckWorktreeClean verifies no uncommitted changes exist
func (p *PreflightChecker) CheckWorktreeClean() PreflightCheck {
	cmd := exec.Command("git", "-C", p.worktreeBase, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return PreflightCheck{
			Passed: false,
			Error:  fmt.Sprintf("Failed to check git status: %v", err),
		}
	}

	if len(strings.TrimSpace(string(output))) > 0 {
		return PreflightCheck{
			Passed: false,
			Error:  fmt.Sprintf("Uncommitted changes: %s", strings.TrimSpace(string(output))),
		}
	}

	return PreflightCheck{Passed: true}
}

// CheckWorktreeSync verifies worktree is up-to-date with remote
func (p *PreflightChecker) CheckWorktreeSync() PreflightCheck {
	// First fetch
	fetchCmd := exec.Command("git", "-C", p.worktreeBase, "fetch", "origin")
	if err := fetchCmd.Run(); err != nil {
		return PreflightCheck{
			Passed: false,
			Error:  fmt.Sprintf("Failed to fetch from origin: %v", err),
		}
	}

	// Check how many commits behind
	revListCmd := exec.Command("git", "-C", p.worktreeBase, "rev-list",
		fmt.Sprintf("HEAD..origin/%s", p.baseBranch), "--count")
	output, err := revListCmd.Output()
	if err != nil {
		return PreflightCheck{
			Passed: false,
			Error:  fmt.Sprintf("Failed to check commits behind: %v", err),
		}
	}

	behind, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return PreflightCheck{
			Passed: false,
			Error:  fmt.Sprintf("Failed to parse commit count: %v", err),
		}
	}

	if behind > 0 {
		return PreflightCheck{
			Passed: false,
			Behind: behind,
			Error:  fmt.Sprintf("Behind origin/%s by %d commits", p.baseBranch, behind),
		}
	}

	return PreflightCheck{Passed: true, Behind: 0}
}

// CheckDiskSpace verifies sufficient disk space is available
func (p *PreflightChecker) CheckDiskSpace() PreflightCheck {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(p.worktreeBase, &stat); err != nil {
		return PreflightCheck{
			Passed: false,
			Error:  fmt.Sprintf("Failed to check disk space: %v", err),
		}
	}

	// Available space in MB
	availMB := int64(stat.Bavail) * int64(stat.Bsize) / (1024 * 1024)

	if availMB < p.minDiskMB {
		return PreflightCheck{
			Passed:     false,
			AvailMB:    availMB,
			RequiredMB: p.minDiskMB,
			Error:      fmt.Sprintf("Insufficient disk space: %dMB available, %dMB required", availMB, p.minDiskMB),
		}
	}

	return PreflightCheck{
		Passed:     true,
		AvailMB:    availMB,
		RequiredMB: p.minDiskMB,
	}
}

// CheckCredentials verifies push access to remote
func (p *PreflightChecker) CheckCredentials() PreflightCheck {
	cmd := exec.Command("git", "-C", p.worktreeBase, "ls-remote", "origin")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return PreflightCheck{
			Passed: false,
			Error:  fmt.Sprintf("Cannot access remote: %s", strings.TrimSpace(string(output))),
		}
	}

	return PreflightCheck{Passed: true}
}

// CheckBranchAvailable verifies the branch doesn't already exist
func (p *PreflightChecker) CheckBranchAvailable() PreflightCheck {
	if p.branchName == "" {
		return PreflightCheck{Passed: true}
	}

	// Check local branch
	localCmd := exec.Command("git", "-C", p.worktreeBase, "rev-parse", "--verify", p.branchName)
	if err := localCmd.Run(); err == nil {
		return PreflightCheck{
			Passed:   false,
			Existing: "local",
			Error:    fmt.Sprintf("Branch '%s' already exists locally", p.branchName),
		}
	}

	// Check remote branch
	remoteCmd := exec.Command("git", "-C", p.worktreeBase, "rev-parse", "--verify",
		fmt.Sprintf("origin/%s", p.branchName))
	if err := remoteCmd.Run(); err == nil {
		return PreflightCheck{
			Passed:   false,
			Existing: "remote",
			Error:    fmt.Sprintf("Branch '%s' already exists on remote", p.branchName),
		}
	}

	return PreflightCheck{Passed: true}
}

// CheckNoInProgressOps verifies no rebase/merge is in progress
func (p *PreflightChecker) CheckNoInProgressOps() PreflightCheck {
	gitDir := filepath.Join(p.worktreeBase, ".git")

	// Check for rebase in progress
	rebaseMerge := filepath.Join(gitDir, "rebase-merge")
	if _, err := os.Stat(rebaseMerge); err == nil {
		return PreflightCheck{
			Passed: false,
			Error:  "Rebase in progress",
		}
	}

	rebaseApply := filepath.Join(gitDir, "rebase-apply")
	if _, err := os.Stat(rebaseApply); err == nil {
		return PreflightCheck{
			Passed: false,
			Error:  "Rebase in progress",
		}
	}

	// Check for merge in progress
	mergeHead := filepath.Join(gitDir, "MERGE_HEAD")
	if _, err := os.Stat(mergeHead); err == nil {
		return PreflightCheck{
			Passed: false,
			Error:  "Merge in progress",
		}
	}

	return PreflightCheck{Passed: true}
}

// RunPreflightForProject runs preflight checks for a project
func RunPreflightForProject(vegaDir, projectName string) (*PreflightResult, error) {
	// Load project config
	projectPath := filepath.Join(vegaDir, "projects", projectName+".md")
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("project not found: %s", projectName)
	}

	// Get worktree base path
	workspacesDir := filepath.Join(vegaDir, "workspaces", projectName)
	worktreeBase := filepath.Join(workspacesDir, "worktree-base")

	if _, err := os.Stat(worktreeBase); os.IsNotExist(err) {
		return nil, fmt.Errorf("worktree-base not found: %s (run 'vega-hub project setup %s' first)", worktreeBase, projectName)
	}

	// Get base branch from project config
	baseBranch, err := getProjectBaseBranch(projectPath)
	if err != nil {
		baseBranch = "main" // fallback
	}

	checker := NewPreflightChecker(worktreeBase, baseBranch, "")
	return checker.RunAll(), nil
}

// getProjectBaseBranch parses the base branch from project markdown
func getProjectBaseBranch(projectPath string) (string, error) {
	data, err := os.ReadFile(projectPath)
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Base Branch:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
		// Also check for "Default Branch:" format
		if strings.HasPrefix(line, "Default Branch:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}

	return "", fmt.Errorf("base branch not found in project config")
}
