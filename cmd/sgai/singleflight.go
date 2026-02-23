package main

import "sync"

type singleflightCall[V any] struct {
	wg  sync.WaitGroup
	val V
	err error
}

type singleflight[K ~string, V any] struct {
	mu    sync.Mutex
	calls map[K]*singleflightCall[V]
}

func (s *singleflight[K, V]) do(key K, fn func() (V, error)) (V, error) {
	s.mu.Lock()
	if s.calls == nil {
		s.calls = make(map[K]*singleflightCall[V])
	}
	if c, ok := s.calls[key]; ok {
		s.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := &singleflightCall[V]{}
	c.wg.Add(1)
	s.calls[key] = c
	s.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	s.mu.Lock()
	delete(s.calls, key)
	s.mu.Unlock()

	return c.val, c.err
}
