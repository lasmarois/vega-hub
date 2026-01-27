package hub

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// LockType represents the type of lock being acquired
type LockType string

const (
	// LockWorktreeBase protects worktree-base operations (pull, branch, merge)
	LockWorktreeBase LockType = "worktree-base"

	// LockBranch protects branch creation for a specific goal
	LockBranch LockType = "branch"

	// LockMerge protects merge operations
	LockMerge LockType = "merge"

	// LockGoalState protects goal state transitions
	LockGoalState LockType = "goal-state"

	// LockProject protects project-level operations
	LockProject LockType = "project"

	// LockRegistry protects REGISTRY.md updates
	LockRegistry LockType = "registry"
)

// Default lock settings
const (
	DefaultLockTimeout      = 10 * time.Minute
	WorktreeLockTimeout     = 30 * time.Second
	BranchLockTimeout       = 10 * time.Second
	MergeLockTimeout        = 60 * time.Second
	GoalStateLockTimeout    = 5 * time.Second
	RegistryLockTimeout     = 5 * time.Second
	StaleThreshold          = 15 * time.Minute // Consider lock stale after this duration
	LockAcquireRetryDelay   = 100 * time.Millisecond
)

// LockInfo contains metadata about a lock
type LockInfo struct {
	PID       int       `json:"pid"`
	Hostname  string    `json:"hostname"`
	AcquiredAt time.Time `json:"acquired_at"`
	Resource  string    `json:"resource"`
	LockType  LockType  `json:"lock_type"`
	GoalID    string    `json:"goal_id,omitempty"`
	Project   string    `json:"project,omitempty"`
	Owner     string    `json:"owner,omitempty"` // User or session that acquired the lock
}

// IsStale checks if the lock is stale based on timestamp and PID
func (l *LockInfo) IsStale() bool {
	// Check if timestamp is too old
	if time.Since(l.AcquiredAt) > StaleThreshold {
		return true
	}

	// Check if the process is still running
	if !isProcessRunning(l.PID) {
		return true
	}

	return false
}

// Lock represents a file-based lock
type Lock struct {
	path       string
	info       *LockInfo
	acquired   bool
}

// LockManager handles lock acquisition and release
type LockManager struct {
	dir string // vega-missile directory (locks stored in .locks/)
}

// NewLockManager creates a new LockManager
func NewLockManager(dir string) *LockManager {
	return &LockManager{dir: dir}
}

// locksDir returns the path to the locks directory
func (m *LockManager) locksDir() string {
	return filepath.Join(m.dir, ".locks")
}

// ensureLocksDir creates the locks directory if it doesn't exist
func (m *LockManager) ensureLocksDir() error {
	return os.MkdirAll(m.locksDir(), 0755)
}

// lockPath returns the path for a specific lock
func (m *LockManager) lockPath(lockType LockType, resource string) string {
	filename := fmt.Sprintf("%s-%s.lock", lockType, sanitizeResource(resource))
	return filepath.Join(m.locksDir(), filename)
}

// AcquireWorktreeBase acquires a lock for worktree-base operations
func (m *LockManager) AcquireWorktreeBase(project, owner string) (*Lock, error) {
	return m.acquire(LockWorktreeBase, project, "", project, owner, WorktreeLockTimeout)
}

// AcquireBranch acquires a lock for branch creation
func (m *LockManager) AcquireBranch(project, goalID, owner string) (*Lock, error) {
	resource := fmt.Sprintf("%s-%s", project, goalID)
	return m.acquire(LockBranch, resource, goalID, project, owner, BranchLockTimeout)
}

// AcquireMerge acquires a lock for merge operations
func (m *LockManager) AcquireMerge(project, owner string) (*Lock, error) {
	return m.acquire(LockMerge, project, "", project, owner, MergeLockTimeout)
}

// AcquireGoalState acquires a lock for goal state transitions
func (m *LockManager) AcquireGoalState(goalID, owner string) (*Lock, error) {
	return m.acquire(LockGoalState, goalID, goalID, "", owner, GoalStateLockTimeout)
}

// AcquireRegistry acquires a lock for REGISTRY.md updates
func (m *LockManager) AcquireRegistry(owner string) (*Lock, error) {
	return m.acquire(LockRegistry, "registry", "", "", owner, RegistryLockTimeout)
}

// Acquire acquires a lock with custom timeout
func (m *LockManager) Acquire(lockType LockType, resource, goalID, project, owner string, timeout time.Duration) (*Lock, error) {
	return m.acquire(lockType, resource, goalID, project, owner, timeout)
}

// acquire is the internal lock acquisition implementation
func (m *LockManager) acquire(lockType LockType, resource, goalID, project, owner string, timeout time.Duration) (*Lock, error) {
	if err := m.ensureLocksDir(); err != nil {
		return nil, fmt.Errorf("creating locks directory: %w", err)
	}

	lockPath := m.lockPath(lockType, resource)
	deadline := time.Now().Add(timeout)

	hostname, _ := os.Hostname()
	info := &LockInfo{
		PID:        os.Getpid(),
		Hostname:   hostname,
		AcquiredAt: time.Now(),
		Resource:   resource,
		LockType:   lockType,
		GoalID:     goalID,
		Project:    project,
		Owner:      owner,
	}

	lock := &Lock{
		path: lockPath,
		info: info,
	}

	for {
		// Check timeout
		if time.Now().After(deadline) {
			// Get info about who holds the lock for better error message
			existingLock, _ := m.GetLockInfo(lockPath)
			return nil, &LockTimeoutError{
				Resource:     resource,
				LockType:     lockType,
				Timeout:      timeout,
				HeldBy:       existingLock,
			}
		}

		// Try to acquire lock
		err := lock.tryAcquire()
		if err == nil {
			return lock, nil
		}

		// If lock exists, check if it's stale
		if os.IsExist(err) {
			existingLock, readErr := m.GetLockInfo(lockPath)
			if readErr == nil && existingLock.IsStale() {
				// Steal stale lock
				if stealErr := lock.steal(existingLock); stealErr == nil {
					return lock, nil
				}
			}
		}

		// Wait before retry
		time.Sleep(LockAcquireRetryDelay)
	}
}

// TryAcquire attempts to acquire a lock without waiting
func (m *LockManager) TryAcquire(lockType LockType, resource, goalID, project, owner string) (*Lock, error) {
	if err := m.ensureLocksDir(); err != nil {
		return nil, fmt.Errorf("creating locks directory: %w", err)
	}

	lockPath := m.lockPath(lockType, resource)
	
	hostname, _ := os.Hostname()
	info := &LockInfo{
		PID:        os.Getpid(),
		Hostname:   hostname,
		AcquiredAt: time.Now(),
		Resource:   resource,
		LockType:   lockType,
		GoalID:     goalID,
		Project:    project,
		Owner:      owner,
	}

	lock := &Lock{
		path: lockPath,
		info: info,
	}

	err := lock.tryAcquire()
	if err != nil {
		// Check if stale
		if os.IsExist(err) {
			existingLock, readErr := m.GetLockInfo(lockPath)
			if readErr == nil && existingLock.IsStale() {
				if stealErr := lock.steal(existingLock); stealErr == nil {
					return lock, nil
				}
			}
		}
		return nil, &LockError{Resource: resource, LockType: lockType, Err: err}
	}

	return lock, nil
}

// GetLockInfo reads lock information from a lock file
func (m *LockManager) GetLockInfo(lockPath string) (*LockInfo, error) {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return nil, err
	}

	var info LockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parsing lock info: %w", err)
	}

	return &info, nil
}

// ForceRelease forcefully releases a lock (escape hatch)
func (m *LockManager) ForceRelease(lockType LockType, resource string) error {
	lockPath := m.lockPath(lockType, resource)
	
	// Check if lock exists
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		return fmt.Errorf("lock not found: %s-%s", lockType, resource)
	}

	return os.Remove(lockPath)
}

// ListLocks returns all current locks
func (m *LockManager) ListLocks() ([]*LockInfo, error) {
	entries, err := os.ReadDir(m.locksDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var locks []*LockInfo
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".lock" {
			continue
		}

		lockPath := filepath.Join(m.locksDir(), entry.Name())
		info, err := m.GetLockInfo(lockPath)
		if err != nil {
			continue // Skip unreadable locks
		}
		locks = append(locks, info)
	}

	return locks, nil
}

// CleanStaleLocks removes all stale locks
func (m *LockManager) CleanStaleLocks() (int, error) {
	locks, err := m.ListLocks()
	if err != nil {
		return 0, err
	}

	cleaned := 0
	for _, lock := range locks {
		if lock.IsStale() {
			lockPath := m.lockPath(lock.LockType, lock.Resource)
			if err := os.Remove(lockPath); err == nil {
				cleaned++
			}
		}
	}

	return cleaned, nil
}

// Lock methods

// tryAcquire attempts to create the lock file atomically
func (l *Lock) tryAcquire() error {
	data, err := json.MarshalIndent(l.info, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling lock info: %w", err)
	}

	// Use O_EXCL for atomic creation (fails if file exists)
	file, err := os.OpenFile(l.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		// Clean up on write failure
		os.Remove(l.path)
		return fmt.Errorf("writing lock file: %w", err)
	}

	l.acquired = true
	return nil
}

// steal takes over a stale lock
func (l *Lock) steal(existingLock *LockInfo) error {
	// Remove the stale lock
	if err := os.Remove(l.path); err != nil {
		return fmt.Errorf("removing stale lock: %w", err)
	}

	// Try to acquire again
	return l.tryAcquire()
}

// Release releases the lock
func (l *Lock) Release() error {
	if !l.acquired {
		return nil
	}

	l.acquired = false
	return os.Remove(l.path)
}

// Info returns the lock's metadata
func (l *Lock) Info() *LockInfo {
	return l.info
}

// IsAcquired returns whether the lock is currently held
func (l *Lock) IsAcquired() bool {
	return l.acquired
}

// Error types

// LockError represents a general lock error
type LockError struct {
	Resource string
	LockType LockType
	Err      error
}

func (e *LockError) Error() string {
	return fmt.Sprintf("lock error for %s-%s: %v", e.LockType, e.Resource, e.Err)
}

func (e *LockError) Unwrap() error {
	return e.Err
}

// LockTimeoutError represents a timeout waiting for a lock
type LockTimeoutError struct {
	Resource string
	LockType LockType
	Timeout  time.Duration
	HeldBy   *LockInfo
}

func (e *LockTimeoutError) Error() string {
	msg := fmt.Sprintf("timeout acquiring %s lock for %s after %v", e.LockType, e.Resource, e.Timeout)
	if e.HeldBy != nil {
		msg += fmt.Sprintf(" (held by PID %d since %s)", e.HeldBy.PID, e.HeldBy.AcquiredAt.Format(time.RFC3339))
	}
	return msg
}

// Helper functions

// sanitizeResource converts a resource name to a safe filename
func sanitizeResource(resource string) string {
	// Replace path separators and other problematic chars
	result := make([]byte, 0, len(resource))
	for i := 0; i < len(resource); i++ {
		c := resource[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			result = append(result, c)
		} else {
			result = append(result, '-')
		}
	}
	return string(result)
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	// On Unix, sending signal 0 checks if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// WithLock is a helper that acquires a lock, runs a function, and releases the lock
func (m *LockManager) WithLock(lockType LockType, resource, goalID, project, owner string, timeout time.Duration, fn func() error) error {
	lock, err := m.Acquire(lockType, resource, goalID, project, owner, timeout)
	if err != nil {
		return err
	}
	defer lock.Release()

	return fn()
}

// WithWorktreeBaseLock is a convenience wrapper for worktree-base operations
func (m *LockManager) WithWorktreeBaseLock(project, owner string, fn func() error) error {
	lock, err := m.AcquireWorktreeBase(project, owner)
	if err != nil {
		return err
	}
	defer lock.Release()

	return fn()
}

// WithMergeLock is a convenience wrapper for merge operations
func (m *LockManager) WithMergeLock(project, owner string, fn func() error) error {
	lock, err := m.AcquireMerge(project, owner)
	if err != nil {
		return err
	}
	defer lock.Release()

	return fn()
}

// WithRegistryLock is a convenience wrapper for REGISTRY.md updates
func (m *LockManager) WithRegistryLock(owner string, fn func() error) error {
	lock, err := m.AcquireRegistry(owner)
	if err != nil {
		return err
	}
	defer lock.Release()

	return fn()
}
