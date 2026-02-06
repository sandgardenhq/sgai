---
name: go-testing-coverage
description: Go testing patterns, coverage analysis, and best practices. When writing tests for Go code or analyzing test coverage
metadata:
    languages: go
---

# Go Testing & Coverage

Testing patterns and coverage analysis for Go projects.

## Reference URLs

For deeper information, fetch these URLs:
- https://go.dev/doc/build-cover - Coverage profiling in Go
- https://go.dev/wiki/TestComments - Go Test Comments

## Basic Testing

### Test File Structure

Tests live in `*_test.go` files alongside the code:

```
mypackage/
├── mycode.go
└── mycode_test.go
```

### Simple Test

```go
package mypackage

import "testing"

func TestFunctionName(t *testing.T) {
    // Arrange
    input := "test"
    want := "expected"

    // Act
    got := FunctionName(input)

    // Assert
    if got != want {
        t.Errorf("FunctionName(%q) = %q; want %q", input, got, want)
    }
}
```

### Error Message Format

Follow Go convention: actual != expected, message matches order.

```go
// GOOD
if got != want {
    t.Errorf("FunctionName(%q) = %d; want %d", input, got, want)
}

// BAD - reversed
if want != got {
    t.Errorf("expected %d, got %d", want, got)
}
```

## Table-Driven Tests

Preferred pattern for multiple test cases:

```go
func TestReverseRunes(t *testing.T) {
    tests := []struct {
        name string
        in   string
        want string
    }{
        {"simple", "Hello", "olleH"},
        {"unicode", "Hello, 世界", "界世 ,olleH"},
        {"empty", "", ""},
        {"single", "a", "a"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := ReverseRunes(tt.in)
            if got != tt.want {
                t.Errorf("ReverseRunes(%q) = %q; want %q", tt.in, got, tt.want)
            }
        })
    }
}
```

### Parallel Tests

```go
func TestReverseRunes(t *testing.T) {
    tests := []struct {
        name string
        in   string
        want string
    }{
        {"simple", "Hello", "olleH"},
        {"unicode", "Hello, 世界", "界世 ,olleH"},
    }

    for _, tt := range tests {
        tt := tt // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            got := ReverseRunes(tt.in)
            if got != tt.want {
                t.Errorf("ReverseRunes(%q) = %q; want %q", tt.in, got, tt.want)
            }
        })
    }
}
```

## Running Tests

### Basic Commands

```bash
# Run all tests
go test ./...

# Run tests in current directory
go test

# Run specific test
go test -run TestFunctionName

# Run with subtests
go test -run TestReverseRunes/simple

# Verbose output
go test -v ./...

# Short mode (skip long tests)
go test -short ./...
```

### Race Detection

```bash
# Run with race detector
go test -race ./...
```

### Benchmarks

```bash
# Run benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkFunctionName ./...

# With memory stats
go test -bench=. -benchmem ./...
```

## Coverage Analysis

### Basic Coverage

```bash
# Coverage summary
go test -cover ./...

# Coverage with percentage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Coverage HTML report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Coverage Modes

```bash
# Count mode - how many times each statement executed
go test -covermode=count -coverprofile=coverage.out ./...

# Set mode - binary: was statement executed (default)
go test -covermode=set -coverprofile=coverage.out ./...

# Atomic mode - like count but thread-safe
go test -covermode=atomic -coverprofile=coverage.out ./...
```

### Integration Test Coverage

```bash
# Build instrumented binary
go build -cover -o myapp ./cmd/myapp

# Set coverage output directory
GOCOVERDIR=./coverage ./myapp

# Merge coverage data
go tool covdata textfmt -i=./coverage -o coverage.out
```

## Test Helpers

### Setup and Teardown

```go
func TestMain(m *testing.M) {
    // Setup
    setup()

    // Run tests
    code := m.Run()

    // Teardown
    teardown()

    os.Exit(code)
}
```

### Helper Functions

```go
func TestSomething(t *testing.T) {
    helper := setupHelper(t)
    defer helper.cleanup()

    // Test code...
}

func setupHelper(t *testing.T) *TestHelper {
    t.Helper() // Mark as helper for better error reporting

    // Setup code...
    return &TestHelper{}
}
```

### Temporary Files

```go
func TestWithTempFile(t *testing.T) {
    // Create temp directory (cleaned up automatically)
    dir := t.TempDir()

    // Create temp file
    f, err := os.CreateTemp(dir, "test-*.txt")
    if err != nil {
        t.Fatal(err)
    }
    defer f.Close()

    // Test code...
}
```

## Benchmarking

### Basic Benchmark

```go
func BenchmarkReverseRunes(b *testing.B) {
    input := "Hello, World!"

    for i := 0; i < b.N; i++ {
        ReverseRunes(input)
    }
}
```

### Benchmark with Setup

```go
func BenchmarkReverseRunes(b *testing.B) {
    input := "Hello, World!"

    b.ResetTimer() // Reset after setup

    for i := 0; i < b.N; i++ {
        ReverseRunes(input)
    }
}
```

### Sub-benchmarks

```go
func BenchmarkReverseRunes(b *testing.B) {
    cases := []struct {
        name string
        in   string
    }{
        {"short", "Hello"},
        {"medium", "Hello, World! How are you today?"},
        {"long", strings.Repeat("Hello, World!", 100)},
    }

    for _, tc := range cases {
        b.Run(tc.name, func(b *testing.B) {
            for i := 0; i < b.N; i++ {
                ReverseRunes(tc.in)
            }
        })
    }
}
```

## Test Patterns

### Testing Errors

```go
func TestFunctionError(t *testing.T) {
    _, err := FunctionThatMightFail(invalidInput)

    if err == nil {
        t.Fatal("expected error, got nil")
    }

    // Check specific error type
    var myErr *MyError
    if !errors.As(err, &myErr) {
        t.Errorf("expected MyError, got %T", err)
    }
}
```

### Testing HTTP Handlers

```go
func TestHandler(t *testing.T) {
    req := httptest.NewRequest("GET", "/path", nil)
    w := httptest.NewRecorder()

    handler(w, req)

    resp := w.Result()
    if resp.StatusCode != http.StatusOK {
        t.Errorf("status = %d; want %d", resp.StatusCode, http.StatusOK)
    }
}
```

### Testing with Context

```go
func TestWithContext(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    result, err := FunctionWithContext(ctx)
    if err != nil {
        t.Fatal(err)
    }

    // Assert result...
}
```

## External Test Package

Use `_test` suffix for black-box testing:

```go
// mypackage_test.go
package mypackage_test

import (
    "testing"
    "github.com/user/project/mypackage"
)

func TestPublicAPI(t *testing.T) {
    // Can only access exported symbols
    result := mypackage.PublicFunction()
    // ...
}
```

## Quick Reference

### Commands

| Command | Purpose |
|---------|---------|
| `go test` | Run tests |
| `go test -v` | Verbose output |
| `go test -race` | Race detection |
| `go test -cover` | Coverage summary |
| `go test -coverprofile=c.out` | Coverage profile |
| `go tool cover -html=c.out` | HTML report |
| `go test -bench=.` | Run benchmarks |

### Coverage Targets

- **80%+** for critical paths
- **60%+** for utilities
- Focus on meaningful coverage, not just numbers

## Modern Testing Patterns (Go 1.21+)

**Documentation:** https://go.dev/blog/testing-time

### Using slices in Tests

The `slices` package simplifies test assertions:

```go
func TestSort(t *testing.T) {
    got := MySort([]int{3, 1, 4, 1, 5})
    want := []int{1, 1, 3, 4, 5}

    // GOOD - use slices.Equal
    if !slices.Equal(got, want) {
        t.Errorf("MySort() = %v; want %v", got, want)
    }

    // Also useful: slices.Contains for membership tests
    if !slices.Contains(got, 3) {
        t.Error("expected result to contain 3")
    }
}
```

### Generic Test Helpers

Write type-safe test helpers using generics:

```go
// Generic assertion helper
func assertEqual[T comparable](t *testing.T, got, want T) {
    t.Helper()
    if got != want {
        t.Errorf("got %v; want %v", got, want)
    }
}

// Generic slice assertion
func assertSliceEqual[T comparable](t *testing.T, got, want []T) {
    t.Helper()
    if !slices.Equal(got, want) {
        t.Errorf("got %v; want %v", got, want)
    }
}
```

### testing/slogtest

For testing `log/slog` handlers:

```go
import "testing/slogtest"

func TestHandler(t *testing.T) {
    var buf bytes.Buffer
    h := slog.NewJSONHandler(&buf, nil)

    // slogtest verifies handler behavior
    results := func() []map[string]any {
        // Parse log output and return records
    }

    if err := slogtest.TestHandler(h, results); err != nil {
        t.Error(err)
    }
}
```

**References:**
- https://go.dev/blog/testing-time
- https://pkg.go.dev/slices
- https://pkg.go.dev/testing/slogtest

## Common Mistakes

- **Testing implementation** - Test behavior, not internals
- **Unhelpful errors** - Include inputs, expected, actual
- **No edge cases** - Empty, nil, boundary values
- **Flaky tests** - Avoid time-dependent tests (use clock injection)
- **Too many mocks** - Test real code when possible
- **Coverage obsession** - Quality > quantity
- **Manual slice comparison** - Use `slices.Equal` instead
