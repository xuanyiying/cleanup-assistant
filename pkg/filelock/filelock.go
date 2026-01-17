// Package filelock provides thread-safe file locking to prevent concurrent operations on the same file.
//
// This package implements a lock manager that uses file paths as lock keys, ensuring that
// only one goroutine can operate on a specific file at a time.
//
// Example usage:
//
//	lm := filelock.NewLockManager()
//
//	// Manual locking
//	if err := lm.Lock("/path/to/file"); err != nil {
//	    return err
//	}
//	defer lm.Unlock("/path/to/file")
//	// ... perform file operation
//
//	// Automatic locking with function
//	err := lm.WithLock("/path/to/file", func() error {
//	    // ... perform file operation
//	    return nil
//	})
//
//	// Non-blocking lock attempt
//	if lm.TryLock("/path/to/file") {
//	    defer lm.Unlock("/path/to/file")
//	    // ... perform file operation
//	}
package filelock

import (
	"fmt"
	"sync"
	"time"
)

// LockManager manages file locks to prevent concurrent operations on the same file
type LockManager struct {
	locks map[string]*fileLock
	mu    sync.Mutex
}

// fileLock represents a lock on a specific file
type fileLock struct {
	path      string
	mu        sync.Mutex
	acquired  time.Time
	goroutine string
}

// NewLockManager creates a new file lock manager
func NewLockManager() *LockManager {
	return &LockManager{
		locks: make(map[string]*fileLock),
	}
}

// Lock acquires a lock on a file path
// Returns an error if the file is already locked
func (lm *LockManager) Lock(path string) error {
	lm.mu.Lock()
	
	// Get or create lock for this path
	lock, exists := lm.locks[path]
	if !exists {
		lock = &fileLock{
			path: path,
		}
		lm.locks[path] = lock
	}
	lm.mu.Unlock()

	// Try to acquire the file-specific lock
	lock.mu.Lock()
	lock.acquired = time.Now()
	
	return nil
}

// Unlock releases a lock on a file path
func (lm *LockManager) Unlock(path string) error {
	lm.mu.Lock()
	lock, exists := lm.locks[path]
	lm.mu.Unlock()

	if !exists {
		return fmt.Errorf("no lock found for path: %s", path)
	}

	lock.mu.Unlock()
	return nil
}

// TryLock attempts to acquire a lock without blocking
// Returns true if lock was acquired, false if already locked
func (lm *LockManager) TryLock(path string) bool {
	lm.mu.Lock()
	
	// Get or create lock for this path
	lock, exists := lm.locks[path]
	if !exists {
		lock = &fileLock{
			path: path,
		}
		lm.locks[path] = lock
	}
	lm.mu.Unlock()

	// Try to acquire the file-specific lock
	if lock.mu.TryLock() {
		lock.acquired = time.Now()
		return true
	}
	
	return false
}

// IsLocked checks if a file path is currently locked
func (lm *LockManager) IsLocked(path string) bool {
	lm.mu.Lock()
	lock, exists := lm.locks[path]
	lm.mu.Unlock()

	if !exists {
		return false
	}

	// Try to acquire and immediately release
	if lock.mu.TryLock() {
		lock.mu.Unlock()
		return false
	}

	return true
}

// WithLock executes a function while holding a lock on the file
func (lm *LockManager) WithLock(path string, fn func() error) error {
	if err := lm.Lock(path); err != nil {
		return err
	}
	defer lm.Unlock(path)

	return fn()
}

// CleanupStale removes locks that haven't been used recently
// This is a safety mechanism to prevent lock leaks
func (lm *LockManager) CleanupStale(maxAge time.Duration) int {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	removed := 0
	for path, lock := range lm.locks {
		// Try to acquire the lock
		if lock.mu.TryLock() {
			// Check if it's stale
			if time.Since(lock.acquired) > maxAge {
				lock.mu.Unlock()
				delete(lm.locks, path)
				removed++
			} else {
				lock.mu.Unlock()
			}
		}
	}

	return removed
}

// Size returns the number of locks currently managed
func (lm *LockManager) Size() int {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	return len(lm.locks)
}
