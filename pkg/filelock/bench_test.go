package filelock

import (
	"testing"
)

func BenchmarkLockUnlock(b *testing.B) {
	lm := NewLockManager()
	path := "/test/file.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lm.Lock(path)
		lm.Unlock(path)
	}
}

func BenchmarkTryLock(b *testing.B) {
	lm := NewLockManager()
	path := "/test/file.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if lm.TryLock(path) {
			lm.Unlock(path)
		}
	}
}

func BenchmarkWithLock(b *testing.B) {
	lm := NewLockManager()
	path := "/test/file.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lm.WithLock(path, func() error {
			return nil
		})
	}
}

func BenchmarkConcurrentLocks(b *testing.B) {
	lm := NewLockManager()
	path := "/test/file.txt"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lm.Lock(path)
			lm.Unlock(path)
		}
	})
}
