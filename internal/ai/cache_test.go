package ai

import (
	"fmt"
	"testing"
	"time"
)

func TestCacheBasicOperations(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	// Test Set and Get
	key := "test-key"
	value := []string{"suggestion1", "suggestion2"}

	cache.Set(key, value)

	retrieved, found := cache.Get(key)
	if !found {
		t.Error("Expected to find cached value")
	}

	if len(retrieved) != len(value) {
		t.Errorf("Expected %d suggestions, got %d", len(value), len(retrieved))
	}

	for i, v := range value {
		if retrieved[i] != v {
			t.Errorf("Expected %s, got %s", v, retrieved[i])
		}
	}
}

func TestCacheExpiration(t *testing.T) {
	cache := NewCache(100 * time.Millisecond)

	key := "test-key"
	value := []string{"suggestion"}

	cache.Set(key, value)

	// Should be found immediately
	_, found := cache.Get(key)
	if !found {
		t.Error("Expected to find cached value immediately")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be found after expiration
	_, found = cache.Get(key)
	if found {
		t.Error("Expected cached value to be expired")
	}
}

func TestCacheMiss(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	_, found := cache.Get("non-existent-key")
	if found {
		t.Error("Expected cache miss for non-existent key")
	}
}

func TestCacheClear(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	cache.Set("key1", []string{"value1"})
	cache.Set("key2", []string{"value2"})

	if cache.Size() != 2 {
		t.Errorf("Expected cache size 2, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cache.Size())
	}
}

func TestCacheCleanExpired(t *testing.T) {
	cache := NewCache(100 * time.Millisecond)

	// Add some entries
	cache.Set("key1", []string{"value1"})
	cache.Set("key2", []string{"value2"})
	cache.Set("key3", []string{"value3"})

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Add a fresh entry
	cache.Set("key4", []string{"value4"})

	// Clean expired entries
	removed := cache.CleanExpired()

	if removed != 3 {
		t.Errorf("Expected 3 expired entries, got %d", removed)
	}

	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1 after cleanup, got %d", cache.Size())
	}

	// Fresh entry should still be there
	_, found := cache.Get("key4")
	if !found {
		t.Error("Expected fresh entry to still be in cache")
	}
}

func TestGenerateKey(t *testing.T) {
	key1 := GenerateKey("prefix", "content1")
	key2 := GenerateKey("prefix", "content1")
	key3 := GenerateKey("prefix", "content2")

	// Same content should generate same key
	if key1 != key2 {
		t.Error("Expected same key for same content")
	}

	// Different content should generate different key
	if key1 == key3 {
		t.Error("Expected different key for different content")
	}
}

func TestCacheConcurrency(t *testing.T) {
	cache := NewCache(1 * time.Hour)

	// Test concurrent writes and reads
	done := make(chan bool)

	// Writers
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cache.Set(key, []string{fmt.Sprintf("value-%d", j)})
			}
			done <- true
		}(i)
	}

	// Readers
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				cache.Get(key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 20; i++ {
		<-done
	}

	// Cache should have entries
	if cache.Size() == 0 {
		t.Error("Expected cache to have entries after concurrent operations")
	}
}
