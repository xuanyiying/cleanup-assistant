package ai

import (
	"testing"
	"time"
)

func BenchmarkCacheGet(b *testing.B) {
	cache := NewCache(1 * time.Hour)
	cache.Set("key", []string{"value"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("key")
	}
}

func BenchmarkCacheSet(b *testing.B) {
	cache := NewCache(1 * time.Hour)
	value := []string{"value"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("key", value)
	}
}

func BenchmarkCacheConcurrent(b *testing.B) {
	cache := NewCache(1 * time.Hour)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := "key" + string(rune(i%100))
			if i%2 == 0 {
				cache.Set(key, []string{"value"})
			} else {
				cache.Get(key)
			}
			i++
		}
	})
}

func BenchmarkGenerateKey(b *testing.B) {
	content := "test content for cache key generation"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateKey("prefix", content)
	}
}
