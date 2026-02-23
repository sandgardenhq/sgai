package main

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSingleflightBasicDedup(t *testing.T) {
	var sf singleflight[string, int]
	var callCount atomic.Int32

	const goroutines = 10
	var wg sync.WaitGroup

	gate := make(chan struct{})
	results := make([]int, goroutines)
	errs := make([]error, goroutines)

	for i := range goroutines {
		wg.Go(func() {
			<-gate
			results[i], errs[i] = sf.do("key", func() (int, error) {
				callCount.Add(1)
				time.Sleep(50 * time.Millisecond)
				return 42, nil
			})
		})
	}

	close(gate)
	wg.Wait()

	if got := callCount.Load(); got != 1 {
		t.Errorf("expected fn called once, got %d", got)
	}
	for i := range goroutines {
		if errs[i] != nil {
			t.Errorf("goroutine %d: unexpected error: %v", i, errs[i])
		}
		if results[i] != 42 {
			t.Errorf("goroutine %d: got %d, want 42", i, results[i])
		}
	}
}

func TestSingleflightErrorShared(t *testing.T) {
	var sf singleflight[string, string]
	errExpected := errors.New("test error")
	var callCount atomic.Int32

	const goroutines = 5
	var wg sync.WaitGroup

	gate := make(chan struct{})
	errs := make([]error, goroutines)

	for i := range goroutines {
		wg.Go(func() {
			<-gate
			_, errs[i] = sf.do("key", func() (string, error) {
				callCount.Add(1)
				time.Sleep(50 * time.Millisecond)
				return "", errExpected
			})
		})
	}

	close(gate)
	wg.Wait()

	if got := callCount.Load(); got != 1 {
		t.Errorf("expected fn called once, got %d", got)
	}
	for i := range goroutines {
		if !errors.Is(errs[i], errExpected) {
			t.Errorf("goroutine %d: got error %v, want %v", i, errs[i], errExpected)
		}
	}
}

func TestSingleflightDifferentKeys(t *testing.T) {
	var sf singleflight[string, string]
	var callCount atomic.Int32

	var wg sync.WaitGroup

	gate := make(chan struct{})

	wg.Go(func() {
		<-gate
		_, _ = sf.do("key1", func() (string, error) {
			callCount.Add(1)
			time.Sleep(50 * time.Millisecond)
			return "a", nil
		})
	})

	wg.Go(func() {
		<-gate
		_, _ = sf.do("key2", func() (string, error) {
			callCount.Add(1)
			time.Sleep(50 * time.Millisecond)
			return "b", nil
		})
	})

	close(gate)
	wg.Wait()

	if got := callCount.Load(); got != 2 {
		t.Errorf("expected fn called twice (different keys), got %d", got)
	}
}

func TestSingleflightSequentialCallsNotCached(t *testing.T) {
	var sf singleflight[string, int]
	var callCount atomic.Int32

	for range 3 {
		val, err := sf.do("key", func() (int, error) {
			callCount.Add(1)
			return 99, nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != 99 {
			t.Fatalf("got %d, want 99", val)
		}
	}

	if got := callCount.Load(); got != 3 {
		t.Errorf("sequential calls should each execute fn, got %d calls", got)
	}
}

func TestSingleflightZeroValue(t *testing.T) {
	var sf singleflight[string, int]
	val, err := sf.do("key", func() (int, error) {
		return 0, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 0 {
		t.Errorf("got %d, want 0", val)
	}
}
