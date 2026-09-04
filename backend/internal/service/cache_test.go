package service

import (
	"testing"
	"time"
)

func TestMemoryCache_SetAndGet(t *testing.T) {
	cache := NewMemoryCache()

	cache.Set("test-key", "test-value", 1*time.Second)

	val, ok := cache.Get("test-key")
	if !ok {
		t.Fatalf("expected key to exist in cache")
	}
	if val.(string) != "test-value" {
		t.Fatalf("expected 'test-value', got '%v'", val)
	}

	// Test missing key
	_, ok = cache.Get("non-existent")
	if ok {
		t.Fatalf("expected missing key to return false")
	}
}

func TestMemoryCache_Expiration(t *testing.T) {
	cache := NewMemoryCache()

	cache.Set("expire-key", "val", 10*time.Millisecond)
	time.Sleep(25 * time.Millisecond)

	_, ok := cache.Get("expire-key")
	if ok {
		t.Fatalf("expected key to be expired")
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache()

	cache.Set("key-to-delete", 123, 10*time.Second)
	cache.Delete("key-to-delete")

	_, ok := cache.Get("key-to-delete")
	if ok {
		t.Fatalf("expected deleted key to return false")
	}
}
