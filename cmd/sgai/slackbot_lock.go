package main

import (
	"sync"
)

type sessionLockMap struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func newSessionLockMap() *sessionLockMap {
	return &sessionLockMap{
		locks: make(map[string]*sync.Mutex),
	}
}

func (m *sessionLockMap) acquire(sessionID string) {
	m.mu.Lock()
	lock, ok := m.locks[sessionID]
	if !ok {
		lock = &sync.Mutex{}
		m.locks[sessionID] = lock
	}
	m.mu.Unlock()
	lock.Lock()
}

func (m *sessionLockMap) release(sessionID string) {
	m.mu.Lock()
	lock, ok := m.locks[sessionID]
	m.mu.Unlock()
	if ok {
		lock.Unlock()
	}
}
