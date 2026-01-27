---
name: Testing Time-Dependent Code
description: Patterns for testing time-dependent code with dependency injection
when_to_use: When writing tests for code that uses time.Now() or time-based logic
---

/* Demonstrates patterns for testing time-dependent code.
 * Based on: https://go.dev/blog/testing-time
 * 
 * Key principle: Inject time dependencies instead of calling time.Now() directly.
 */
package main

import (
	"fmt"
	"time"
)

// --- Clock Interface Pattern ---

// Clock abstracts time operations for testability
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	Until(t time.Time) time.Duration
}

// RealClock uses actual system time (for production)
type RealClock struct{}

func (RealClock) Now() time.Time                         { return time.Now() }
func (RealClock) Since(t time.Time) time.Duration        { return time.Since(t) }
func (RealClock) Until(t time.Time) time.Duration        { return time.Until(t) }

// FakeClock provides controllable time (for tests)
type FakeClock struct {
	current time.Time
}

func NewFakeClock(t time.Time) *FakeClock {
	return &FakeClock{current: t}
}

func (c *FakeClock) Now() time.Time                  { return c.current }
func (c *FakeClock) Since(t time.Time) time.Duration { return c.current.Sub(t) }
func (c *FakeClock) Until(t time.Time) time.Duration { return t.Sub(c.current) }

// Advance moves the fake clock forward
func (c *FakeClock) Advance(d time.Duration) {
	c.current = c.current.Add(d)
}

// Set changes the fake clock to a specific time
func (c *FakeClock) Set(t time.Time) {
	c.current = t
}

// --- Service Using Clock ---

// Token represents an authentication token with expiration
type Token struct {
	Value     string
	ExpiresAt time.Time
}

// TokenService manages tokens with testable time handling
type TokenService struct {
	clock    Clock
	lifetime time.Duration
}

// NewTokenService creates a service with real time (production default)
func NewTokenService(lifetime time.Duration) *TokenService {
	return &TokenService{
		clock:    RealClock{},
		lifetime: lifetime,
	}
}

// WithClock allows injecting a custom clock (for tests)
func (s *TokenService) WithClock(c Clock) *TokenService {
	s.clock = c
	return s
}

// CreateToken generates a new token
func (s *TokenService) CreateToken(value string) Token {
	return Token{
		Value:     value,
		ExpiresAt: s.clock.Now().Add(s.lifetime),
	}
}

// IsExpired checks if a token has expired
func (s *TokenService) IsExpired(t Token) bool {
	return s.clock.Now().After(t.ExpiresAt)
}

// TimeUntilExpiry returns how long until the token expires
func (s *TokenService) TimeUntilExpiry(t Token) time.Duration {
	return s.clock.Until(t.ExpiresAt)
}

// --- Alternative: Function Injection ---

// For simpler cases, inject a time function directly

type Cache struct {
	nowFn   func() time.Time
	entries map[string]cacheEntry
}

type cacheEntry struct {
	value     string
	expiresAt time.Time
}

func NewCache() *Cache {
	return &Cache{
		nowFn:   time.Now, // Default to real time
		entries: make(map[string]cacheEntry),
	}
}

// WithNowFunc allows injecting a custom time function (for tests)
func (c *Cache) WithNowFunc(fn func() time.Time) *Cache {
	c.nowFn = fn
	return c
}

func (c *Cache) Set(key, value string, ttl time.Duration) {
	c.entries[key] = cacheEntry{
		value:     value,
		expiresAt: c.nowFn().Add(ttl),
	}
}

func (c *Cache) Get(key string) (string, bool) {
	entry, ok := c.entries[key]
	if !ok {
		return "", false
	}
	if c.nowFn().After(entry.expiresAt) {
		delete(c.entries, key)
		return "", false
	}
	return entry.value, true
}

// --- Example Tests ---

func ExampleTokenServiceTest() {
	// Create a fake clock starting at a known time
	startTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	fakeClock := NewFakeClock(startTime)

	// Create service with injected clock
	svc := NewTokenService(1 * time.Hour).WithClock(fakeClock)

	// Create a token
	token := svc.CreateToken("abc123")
	fmt.Printf("Token created, expires at: %v\n", token.ExpiresAt)

	// Token should not be expired initially
	fmt.Printf("Expired (t=0): %v\n", svc.IsExpired(token)) // false

	// Advance time by 30 minutes
	fakeClock.Advance(30 * time.Minute)
	fmt.Printf("Expired (t=30m): %v\n", svc.IsExpired(token)) // false
	fmt.Printf("Time until expiry: %v\n", svc.TimeUntilExpiry(token)) // 30m

	// Advance time past expiration
	fakeClock.Advance(45 * time.Minute)
	fmt.Printf("Expired (t=75m): %v\n", svc.IsExpired(token)) // true
}

func ExampleCacheTest() {
	// Track current fake time
	currentTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Create cache with injected time function
	cache := NewCache().WithNowFunc(func() time.Time {
		return currentTime
	})

	// Set entry with 5 minute TTL
	cache.Set("key", "value", 5*time.Minute)

	// Entry should exist
	if v, ok := cache.Get("key"); ok {
		fmt.Println("Found:", v)
	}

	// Advance time past TTL
	currentTime = currentTime.Add(6 * time.Minute)

	// Entry should be expired
	if _, ok := cache.Get("key"); !ok {
		fmt.Println("Entry expired")
	}
}

func main() {
	fmt.Println("=== Token Service Test ===")
	ExampleTokenServiceTest()

	fmt.Println("\n=== Cache Test ===")
	ExampleCacheTest()
}
