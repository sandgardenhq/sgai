package main

import (
	"sync"
	"time"
)

type ttlCacheEntry[V any] struct {
	value     V
	expiresAt time.Time
}

type ttlCache[K comparable, V any] struct {
	mu      sync.Mutex
	entries map[K]ttlCacheEntry[V]
	ttl     time.Duration
}

func newTTLCache[K comparable, V any](ttl time.Duration) *ttlCache[K, V] {
	return &ttlCache[K, V]{
		entries: make(map[K]ttlCacheEntry[V]),
		ttl:     ttl,
	}
}

func (c *ttlCache[K, V]) get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[key]
	if !ok {
		var zero V
		return zero, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(c.entries, key)
		var zero V
		return zero, false
	}
	return entry.value, true
}

func (c *ttlCache[K, V]) set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = ttlCacheEntry[V]{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *ttlCache[K, V]) delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

func (c *ttlCache[K, V]) deleteFunc(fn func(K) bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.entries {
		if fn(k) {
			delete(c.entries, k)
		}
	}
}
