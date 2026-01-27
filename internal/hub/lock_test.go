package hub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLockManager_AcquireAndRelease(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager(dir)

	// Acquire lock
	lock, err := lm.AcquireWorktreeBase("test-project", "test-user")
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Verify lock file exists
	lockPath := lm.lockPath(LockWorktreeBase, "test-project")
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Fatal("Lock file was not created")
	}

	// Verify lock info
	info, err := lm.GetLockInfo(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock info: %v", err)
	}
	if info.PID != os.Getpid() {
		t.Errorf("Expected PID %d, got %d", os.Getpid(), info.PID)
	}
	if info.Owner != "test-user" {
		t.Errorf("Expected owner 'test-user', got '%s'", info.Owner)
	}
	if info.LockType != LockWorktreeBase {
		t.Errorf("Expected lock type '%s', got '%s'", LockWorktreeBase, info.LockType)
	}

	// Release lock
	if err := lock.Release(); err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}

	// Verify lock file is removed
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatal("Lock file was not removed after release")
	}
}

func TestLockManager_TryAcquire(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager(dir)

	// First lock should succeed
	lock1, err := lm.TryAcquire(LockBranch, "test-resource", "goal-1", "project-1", "user-1")
	if err != nil {
		t.Fatalf("First TryAcquire failed: %v", err)
	}
	defer lock1.Release()

	// Second lock should fail immediately
	_, err = lm.TryAcquire(LockBranch, "test-resource", "goal-1", "project-1", "user-2")
	if err == nil {
		t.Fatal("Second TryAcquire should have failed")
	}

	// Verify it's a LockError
	lockErr, ok := err.(*LockError)
	if !ok {
		t.Fatalf("Expected LockError, got %T", err)
	}
	if lockErr.Resource != "test-resource" {
		t.Errorf("Expected resource 'test-resource', got '%s'", lockErr.Resource)
	}
}

func TestLockManager_Timeout(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager(dir)

	// Acquire first lock
	lock1, err := lm.AcquireWorktreeBase("test-project", "user-1")
	if err != nil {
		t.Fatalf("Failed to acquire first lock: %v", err)
	}
	defer lock1.Release()

	// Try to acquire with short timeout - should fail
	start := time.Now()
	_, err = lm.Acquire(LockWorktreeBase, "test-project", "", "test-project", "user-2", 200*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	// Verify it's a LockTimeoutError
	timeoutErr, ok := err.(*LockTimeoutError)
	if !ok {
		t.Fatalf("Expected LockTimeoutError, got %T: %v", err, err)
	}
	if timeoutErr.HeldBy == nil {
		t.Error("Expected HeldBy to contain lock holder info")
	}

	// Verify we waited approximately the timeout duration
	if elapsed < 150*time.Millisecond || elapsed > 500*time.Millisecond {
		t.Errorf("Expected to wait ~200ms, waited %v", elapsed)
	}
}

func TestLockManager_StaleLock(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager(dir)

	// Create a stale lock file (with non-existent PID and old timestamp)
	lockPath := lm.lockPath(LockWorktreeBase, "test-project")
	if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
		t.Fatalf("Failed to create locks dir: %v", err)
	}

	staleLock := &LockInfo{
		PID:        99999999, // Very unlikely to be a real PID
		Hostname:   "stale-host",
		AcquiredAt: time.Now().Add(-time.Hour), // Old timestamp
		Resource:   "test-project",
		LockType:   LockWorktreeBase,
		Owner:      "stale-owner",
	}

	data, _ := json.Marshal(staleLock)
	if err := os.WriteFile(lockPath, data, 0644); err != nil {
		t.Fatalf("Failed to create stale lock: %v", err)
	}

	// Acquiring lock should succeed by stealing stale lock
	lock, err := lm.AcquireWorktreeBase("test-project", "new-user")
	if err != nil {
		t.Fatalf("Failed to acquire lock (should have stolen stale): %v", err)
	}
	defer lock.Release()

	// Verify new lock info
	info, _ := lm.GetLockInfo(lockPath)
	if info.Owner != "new-user" {
		t.Errorf("Expected owner 'new-user', got '%s'", info.Owner)
	}
	if info.PID != os.Getpid() {
		t.Errorf("Expected PID %d, got %d", os.Getpid(), info.PID)
	}
}

func TestLockManager_ConcurrentAcquisition(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager(dir)

	const numGoroutines = 10
	var wg sync.WaitGroup
	var successCount int32

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			lock, err := lm.TryAcquire(LockWorktreeBase, "shared", "", "shared", "")
			if err == nil {
				atomic.AddInt32(&successCount, 1)
				// Hold lock briefly
				time.Sleep(10 * time.Millisecond)
				lock.Release()
			}
		}(i)
	}

	wg.Wait()

	// Only one goroutine should have succeeded
	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful acquisition, got %d", successCount)
	}
}

func TestLockManager_ConcurrentWithWait(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager(dir)

	const numGoroutines = 5
	var wg sync.WaitGroup
	var counter int32
	var order []int32

	var orderMu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			lock, err := lm.Acquire(LockWorktreeBase, "shared", "", "shared", "", 5*time.Second)
			if err != nil {
				t.Errorf("Goroutine %d failed to acquire lock: %v", id, err)
				return
			}

			// Critical section
			val := atomic.AddInt32(&counter, 1)
			orderMu.Lock()
			order = append(order, val)
			orderMu.Unlock()

			time.Sleep(10 * time.Millisecond)
			lock.Release()
		}(i)
	}

	wg.Wait()

	// All goroutines should have succeeded
	if counter != numGoroutines {
		t.Errorf("Expected counter %d, got %d", numGoroutines, counter)
	}

	// Order should be sequential
	for i, val := range order {
		if val != int32(i+1) {
			t.Errorf("Expected order[%d] = %d, got %d", i, i+1, val)
		}
	}
}

func TestLockManager_ListLocks(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager(dir)

	// Create several locks
	lock1, _ := lm.AcquireWorktreeBase("project-1", "user-1")
	lock2, _ := lm.AcquireBranch("project-2", "goal-a", "user-2")
	lock3, _ := lm.AcquireMerge("project-3", "user-3")

	defer lock1.Release()
	defer lock2.Release()
	defer lock3.Release()

	locks, err := lm.ListLocks()
	if err != nil {
		t.Fatalf("Failed to list locks: %v", err)
	}

	if len(locks) != 3 {
		t.Errorf("Expected 3 locks, got %d", len(locks))
	}

	// Verify lock types
	types := make(map[LockType]bool)
	for _, l := range locks {
		types[l.LockType] = true
	}

	if !types[LockWorktreeBase] {
		t.Error("Missing worktree-base lock")
	}
	if !types[LockBranch] {
		t.Error("Missing branch lock")
	}
	if !types[LockMerge] {
		t.Error("Missing merge lock")
	}
}

func TestLockManager_ForceRelease(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager(dir)

	// Acquire lock
	lock, err := lm.AcquireWorktreeBase("test-project", "user-1")
	if err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	// Force release from outside
	if err := lm.ForceRelease(LockWorktreeBase, "test-project"); err != nil {
		t.Fatalf("Failed to force release: %v", err)
	}

	// Original lock should now be orphaned (release will fail silently)
	lock.Release() // Should not panic

	// New lock should succeed
	newLock, err := lm.TryAcquire(LockWorktreeBase, "test-project", "", "test-project", "user-2")
	if err != nil {
		t.Fatalf("Failed to acquire after force release: %v", err)
	}
	newLock.Release()
}

func TestLockManager_CleanStaleLocks(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager(dir)

	// Create locks directory
	if err := lm.ensureLocksDir(); err != nil {
		t.Fatalf("Failed to create locks dir: %v", err)
	}

	// Create a stale lock
	staleLock := &LockInfo{
		PID:        99999999,
		Hostname:   "stale-host",
		AcquiredAt: time.Now().Add(-time.Hour),
		Resource:   "stale-project",
		LockType:   LockWorktreeBase,
	}
	stalePath := lm.lockPath(LockWorktreeBase, "stale-project")
	data, _ := json.Marshal(staleLock)
	os.WriteFile(stalePath, data, 0644)

	// Create a fresh lock
	freshLock, _ := lm.AcquireWorktreeBase("fresh-project", "user")

	// Clean stale locks
	cleaned, err := lm.CleanStaleLocks()
	if err != nil {
		t.Fatalf("Failed to clean stale locks: %v", err)
	}

	if cleaned != 1 {
		t.Errorf("Expected 1 cleaned lock, got %d", cleaned)
	}

	// Fresh lock should still exist
	locks, _ := lm.ListLocks()
	if len(locks) != 1 {
		t.Errorf("Expected 1 remaining lock, got %d", len(locks))
	}
	if locks[0].Resource != "fresh-project" {
		t.Errorf("Wrong lock remained: %s", locks[0].Resource)
	}

	freshLock.Release()
}

func TestLockManager_WithLock(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager(dir)

	executed := false

	err := lm.WithWorktreeBaseLock("test-project", "user", func() error {
		executed = true

		// Verify lock is held during function execution
		locks, _ := lm.ListLocks()
		if len(locks) != 1 {
			t.Errorf("Expected lock to be held, found %d locks", len(locks))
		}

		return nil
	})

	if err != nil {
		t.Fatalf("WithLock failed: %v", err)
	}

	if !executed {
		t.Error("Function was not executed")
	}

	// Verify lock is released after function completes
	locks, _ := lm.ListLocks()
	if len(locks) != 0 {
		t.Errorf("Expected no locks after WithLock, found %d", len(locks))
	}
}

func TestLockManager_DifferentLockTypes(t *testing.T) {
	dir := t.TempDir()
	lm := NewLockManager(dir)

	// Different lock types for same resource should not conflict
	lock1, err := lm.AcquireWorktreeBase("project", "user")
	if err != nil {
		t.Fatalf("Failed to acquire worktree-base lock: %v", err)
	}

	lock2, err := lm.AcquireBranch("project", "goal-1", "user")
	if err != nil {
		t.Fatalf("Failed to acquire branch lock: %v", err)
	}

	lock3, err := lm.AcquireMerge("project", "user")
	if err != nil {
		t.Fatalf("Failed to acquire merge lock: %v", err)
	}

	defer lock1.Release()
	defer lock2.Release()
	defer lock3.Release()

	locks, _ := lm.ListLocks()
	if len(locks) != 3 {
		t.Errorf("Expected 3 locks, got %d", len(locks))
	}
}

func TestLockInfo_IsStale(t *testing.T) {
	tests := []struct {
		name     string
		info     *LockInfo
		expected bool
	}{
		{
			name: "fresh lock with running process",
			info: &LockInfo{
				PID:        os.Getpid(), // Current process
				AcquiredAt: time.Now(),
			},
			expected: false,
		},
		{
			name: "old lock with running process",
			info: &LockInfo{
				PID:        os.Getpid(),
				AcquiredAt: time.Now().Add(-time.Hour),
			},
			expected: true,
		},
		{
			name: "fresh lock with dead process",
			info: &LockInfo{
				PID:        99999999, // Very unlikely to be real
				AcquiredAt: time.Now(),
			},
			expected: true,
		},
		{
			name: "old lock with dead process",
			info: &LockInfo{
				PID:        99999999,
				AcquiredAt: time.Now().Add(-time.Hour),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.IsStale(); got != tt.expected {
				t.Errorf("IsStale() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestSanitizeResource(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with-dash", "with-dash"},
		{"with_underscore", "with_underscore"},
		{"with/slash", "with-slash"},
		{"with:colon", "with-colon"},
		{"with spaces", "with-spaces"},
		{"MixedCase123", "MixedCase123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := sanitizeResource(tt.input); got != tt.expected {
				t.Errorf("sanitizeResource(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}
