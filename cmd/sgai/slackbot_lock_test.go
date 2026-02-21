package main

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestSessionLockMapBasic(_ *testing.T) {
	m := newSessionLockMap()

	m.acquire("session-1")
	m.release("session-1")
}

func TestSessionLockMapConcurrent(t *testing.T) {
	m := newSessionLockMap()
	var counter atomic.Int64
	var wg sync.WaitGroup

	for range 100 {
		wg.Go(func() {
			m.acquire("session-1")
			defer m.release("session-1")
			counter.Add(1)
		})
	}

	wg.Wait()

	if counter.Load() != 100 {
		t.Errorf("counter = %d, want 100", counter.Load())
	}
}

func TestSessionLockMapMultipleSessions(t *testing.T) {
	m := newSessionLockMap()
	var counter1, counter2 atomic.Int64
	var wg sync.WaitGroup

	for range 50 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			m.acquire("session-A")
			defer m.release("session-A")
			counter1.Add(1)
		}()
		go func() {
			defer wg.Done()
			m.acquire("session-B")
			defer m.release("session-B")
			counter2.Add(1)
		}()
	}

	wg.Wait()

	if counter1.Load() != 50 {
		t.Errorf("counter1 = %d, want 50", counter1.Load())
	}
	if counter2.Load() != 50 {
		t.Errorf("counter2 = %d, want 50", counter2.Load())
	}
}

func TestSessionLockMapSerializesAccess(t *testing.T) {
	m := newSessionLockMap()
	var inCriticalSection atomic.Int64
	var maxConcurrent atomic.Int64
	var wg sync.WaitGroup

	for range 50 {
		wg.Go(func() {
			m.acquire("session-1")
			defer m.release("session-1")

			current := inCriticalSection.Add(1)
			if current > maxConcurrent.Load() {
				maxConcurrent.Store(current)
			}
			inCriticalSection.Add(-1)
		})
	}

	wg.Wait()

	if maxConcurrent.Load() > 1 {
		t.Errorf("max concurrent in critical section = %d, want <= 1", maxConcurrent.Load())
	}
}
