package filelock

import (
	"sync"
	"testing"
	"time"
)

func TestLockUnlock(t *testing.T) {
	lm := NewLockManager()
	path := "/test/file.txt"

	// Lock should succeed
	if err := lm.Lock(path); err != nil {
		t.Errorf("Lock failed: %v", err)
	}

	// Unlock should succeed
	if err := lm.Unlock(path); err != nil {
		t.Errorf("Unlock failed: %v", err)
	}
}

func TestTryLock(t *testing.T) {
	lm := NewLockManager()
	path := "/test/file.txt"

	// First TryLock should succeed
	if !lm.TryLock(path) {
		t.Error("First TryLock should succeed")
	}

	// Second TryLock should fail (already locked)
	if lm.TryLock(path) {
		t.Error("Second TryLock should fail")
	}

	// Unlock
	lm.Unlock(path)

	// TryLock should succeed again
	if !lm.TryLock(path) {
		t.Error("TryLock after unlock should succeed")
	}

	lm.Unlock(path)
}

func TestIsLocked(t *testing.T) {
	lm := NewLockManager()
	path := "/test/file.txt"

	// Should not be locked initially
	if lm.IsLocked(path) {
		t.Error("File should not be locked initially")
	}

	// Lock the file
	lm.Lock(path)

	// Should be locked now
	if !lm.IsLocked(path) {
		t.Error("File should be locked")
	}

	// Unlock
	lm.Unlock(path)

	// Should not be locked anymore
	if lm.IsLocked(path) {
		t.Error("File should not be locked after unlock")
	}
}

func TestWithLock(t *testing.T) {
	lm := NewLockManager()
	path := "/test/file.txt"

	executed := false
	err := lm.WithLock(path, func() error {
		executed = true
		return nil
	})

	if err != nil {
		t.Errorf("WithLock failed: %v", err)
	}

	if !executed {
		t.Error("Function was not executed")
	}

	// File should be unlocked after WithLock
	if lm.IsLocked(path) {
		t.Error("File should be unlocked after WithLock")
	}
}

func TestConcurrentLocks(t *testing.T) {
	lm := NewLockManager()
	path := "/test/file.txt"

	var counter int
	var wg sync.WaitGroup

	// Start 10 goroutines that increment counter
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := lm.WithLock(path, func() error {
				// Simulate some work
				temp := counter
				time.Sleep(1 * time.Millisecond)
				counter = temp + 1
				return nil
			})

			if err != nil {
				t.Errorf("WithLock failed: %v", err)
			}
		}()
	}

	wg.Wait()

	// Counter should be exactly 10 (no race conditions)
	if counter != 10 {
		t.Errorf("Expected counter to be 10, got %d", counter)
	}
}

func TestMultipleFiles(t *testing.T) {
	lm := NewLockManager()

	// Lock multiple files
	paths := []string{"/file1.txt", "/file2.txt", "/file3.txt"}

	for _, path := range paths {
		if err := lm.Lock(path); err != nil {
			t.Errorf("Lock failed for %s: %v", path, err)
		}
	}

	// All should be locked
	for _, path := range paths {
		if !lm.IsLocked(path) {
			t.Errorf("File %s should be locked", path)
		}
	}

	// Unlock all
	for _, path := range paths {
		if err := lm.Unlock(path); err != nil {
			t.Errorf("Unlock failed for %s: %v", path, err)
		}
	}

	// None should be locked
	for _, path := range paths {
		if lm.IsLocked(path) {
			t.Errorf("File %s should not be locked", path)
		}
	}
}

func TestCleanupStale(t *testing.T) {
	lm := NewLockManager()

	// Create some locks
	paths := []string{"/file1.txt", "/file2.txt", "/file3.txt"}
	for _, path := range paths {
		lm.Lock(path)
		lm.Unlock(path)
	}

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Cleanup stale locks (older than 50ms)
	removed := lm.CleanupStale(50 * time.Millisecond)

	if removed != 3 {
		t.Errorf("Expected 3 stale locks to be removed, got %d", removed)
	}

	if lm.Size() != 0 {
		t.Errorf("Expected 0 locks after cleanup, got %d", lm.Size())
	}
}

func TestSize(t *testing.T) {
	lm := NewLockManager()

	if lm.Size() != 0 {
		t.Errorf("Expected size 0, got %d", lm.Size())
	}

	// Add some locks
	lm.Lock("/file1.txt")
	lm.Lock("/file2.txt")

	if lm.Size() != 2 {
		t.Errorf("Expected size 2, got %d", lm.Size())
	}
}
