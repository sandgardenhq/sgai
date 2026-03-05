package main

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTTLCacheGetSet(t *testing.T) {
	cache := newTTLCache[string, string](1 * time.Minute)

	cache.set("key1", "value1")
	value, ok := cache.get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", value)

	value, ok = cache.get("nonexistent")
	assert.False(t, ok)
	assert.Equal(t, "", value)
}

func TestTTLCacheExpiration(t *testing.T) {
	cache := newTTLCache[string, string](50 * time.Millisecond)

	cache.set("key1", "value1")

	value, ok := cache.get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", value)

	time.Sleep(60 * time.Millisecond)

	value, ok = cache.get("key1")
	assert.False(t, ok)
	assert.Equal(t, "", value)
}

func TestTTLCacheDelete(t *testing.T) {
	cache := newTTLCache[string, string](1 * time.Minute)

	cache.set("key1", "value1")
	cache.delete("key1")

	value, ok := cache.get("key1")
	assert.False(t, ok)
	assert.Equal(t, "", value)
}

func TestTTLCacheDeleteFunc(t *testing.T) {
	cache := newTTLCache[string, string](1 * time.Minute)

	cache.set("key1", "value1")
	cache.set("key2", "value2")
	cache.set("other", "value3")

	cache.deleteFunc(func(k string) bool {
		return k == "key1" || k == "key2"
	})

	_, ok1 := cache.get("key1")
	_, ok2 := cache.get("key2")
	_, okOther := cache.get("other")

	assert.False(t, ok1)
	assert.False(t, ok2)
	assert.True(t, okOther)
}

func TestTTLCacheConcurrent(_ *testing.T) {
	cache := newTTLCache[int, int](1 * time.Minute)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func(n int) {
			defer wg.Done()
			cache.set(n, n*2)
		}(i)
		go func(n int) {
			defer wg.Done()
			cache.get(n)
		}(i)
		go func(n int) {
			defer wg.Done()
			if n%2 == 0 {
				cache.delete(n)
			}
		}(i)
	}

	wg.Wait()
}
