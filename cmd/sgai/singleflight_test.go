package main

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSingleflightDo(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		fn       func() (string, error)
		expected string
		wantErr  bool
	}{
		{
			name:     "success",
			key:      "test-key",
			fn:       func() (string, error) { return "result", nil },
			expected: "result",
			wantErr:  false,
		},
		{
			name:    "error",
			key:     "test-key",
			fn:      func() (string, error) { return "", errors.New("test error") },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sf singleflight[string, string]
			result, err := sf.do(tt.key, tt.fn)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSingleflightDeduplication(t *testing.T) {
	var sf singleflight[string, int]
	var callCount int
	var mu sync.Mutex
	shouldFail := false

	fn := func() (int, error) {
		mu.Lock()
		callCount++
		fail := shouldFail
		mu.Unlock()
		if fail {
			return 0, errors.New("intentional failure")
		}
		time.Sleep(100 * time.Millisecond)
		return 42, nil
	}

	var wg sync.WaitGroup
	results := make(chan int, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := sf.do("same-key", fn)
			require.NoError(t, err)
			results <- result
		}()
	}

	wg.Wait()
	close(results)

	for result := range results {
		assert.Equal(t, 42, result)
	}

	mu.Lock()
	count := callCount
	mu.Unlock()

	assert.Equal(t, 1, count, "function should only be called once")

	mu.Lock()
	shouldFail = true
	mu.Unlock()
	_, err := sf.do("fail-key", fn)
	require.Error(t, err)
}

func TestSingleflightDifferentKeys(t *testing.T) {
	var sf singleflight[string, int]
	var callCount int
	var mu sync.Mutex

	fn := func() (int, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		return 1, nil
	}

	result1, err := sf.do("key1", fn)
	require.NoError(t, err)
	assert.Equal(t, 1, result1)

	result2, err := sf.do("key2", fn)
	require.NoError(t, err)
	assert.Equal(t, 1, result2)

	mu.Lock()
	count := callCount
	mu.Unlock()

	assert.Equal(t, 2, count, "function should be called twice for different keys")
}
