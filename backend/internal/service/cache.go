package service

import (
	"sync"
	"time"
)

type memoryItem struct {
	value      interface{}
	expiration int64
}

// MemoryCache provides a fast, thread-safe L1 in-memory cache
type MemoryCache struct {
	items sync.Map
}

func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{}
	// Periodically prune expired items to prevent memory leak
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now().UnixNano()
			mc.items.Range(func(key, value interface{}) bool {
				item, ok := value.(memoryItem)
				if ok && item.expiration > 0 && now > item.expiration {
					mc.items.Delete(key)
				}
				return true
			})
		}
	}()
	return mc
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	val, ok := c.items.Load(key)
	if !ok {
		return nil, false
	}
	item := val.(memoryItem)
	if item.expiration > 0 && time.Now().UnixNano() > item.expiration {
		c.items.Delete(key)
		return nil, false
	}
	return item.value, true
}

func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}
	c.items.Store(key, memoryItem{
		value:      value,
		expiration: exp,
	})
}

func (c *MemoryCache) Delete(key string) {
	c.items.Delete(key)
}
