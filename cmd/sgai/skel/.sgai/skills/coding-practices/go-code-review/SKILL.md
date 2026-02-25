---
name: go-code-review
description: Go code review checklist based on official Go style guides. When reviewing Go code for style, idioms, and best practices
metadata:
    languages: go
---

# Go Code Review

Comprehensive code review checklist based on official Go style guides.

## Reference URLs

For deeper information, fetch these URLs:
- https://go.dev/wiki/CodeReviewComments - Go Code Review Comments
- https://go.dev/wiki/TestComments - Go Test Comments
- https://google.github.io/styleguide/go/decisions - Google Go Style Guide

## Formatting

### Gofmt

- [ ] Code formatted with `gofmt` or `goimports`
- [ ] Tabs for indentation (not spaces)

Run: `gofmt -w .` or `goimports -w .`

### Line Length

- No rigid limit, but avoid uncomfortably long lines
- Break lines by semantics, not arbitrary length
- Long lines often indicate need for shorter names or refactoring

## Naming

### Package Names

- [ ] Short, concise, lowercase
- [ ] No underscores or mixedCaps
- [ ] Avoid: `util`, `common`, `misc`, `api`, `types`, `interfaces`
- [ ] Name doesn't stutter: `chubby.File` not `chubby.ChubbyFile`

### Variable Names

- [ ] Short for limited scope: `c`, `i`, `r`
- [ ] Longer for broader scope
- [ ] Basic rule: further from declaration = more descriptive

### Receiver Names

- [ ] 1-2 letter abbreviation of type: `c` for Client
- [ ] Consistent across all methods
- [ ] Never `me`, `this`, `self`

### Initialisms

- [ ] Consistent case: `URL` or `url`, never `Url`
- [ ] `ServeHTTP` not `ServeHttp`
- [ ] `xmlHTTPRequest` or `XMLHTTPRequest`
- [ ] `appID` not `appId`

### Getters/Setters

- [ ] `Owner()` not `GetOwner()`
- [ ] `SetOwner()` is fine

### Interface Names

- [ ] Single-method interfaces: `-er` suffix (`Reader`, `Writer`)
- [ ] Multi-method interfaces: descriptive noun

## Comments

### Comment Sentences

- [ ] Full sentences starting with name being described
- [ ] End with period
- [ ] Doc comments on all exported names

```go
// Request represents a request to run a command.
type Request struct { ... }

// Encode writes the JSON encoding of req to w.
func Encode(w io.Writer, req *Request) { ... }
```

### Package Comments

- [ ] Adjacent to `package` clause (no blank line)
- [ ] Full sentences

```go
// Package math provides basic constants and mathematical functions.
package math
```

## Error Handling

### Handle Errors

- [ ] All errors checked (no `_, err := Foo()` then ignored)
- [ ] Errors returned or handled, not swallowed

### Error Strings

- [ ] Lowercase (unless proper noun/acronym)
- [ ] No ending punctuation
- [ ] `fmt.Errorf("something bad")` not `"Something bad."`

### Error Flow

- [ ] Normal path at minimal indentation
- [ ] Error handling indented

```go
// GOOD
if err != nil {
    return err
}
// normal code

// BAD
if err != nil {
    // error
} else {
    // normal
}
```

### Don't Panic

- [ ] Use error returns for normal error handling
- [ ] Panic only for truly unrecoverable situations
- [ ] Library code almost never panics

## Context

- [ ] First parameter when used: `func F(ctx context.Context, ...)`
- [ ] Not stored in structs
- [ ] Passed through call chain
- [ ] Use `context.Background()` only with good reason

## Concurrency

### Goroutine Lifetimes

- [ ] Clear when/whether goroutines exit
- [ ] No goroutine leaks (blocked on unreachable channels)
- [ ] Documented exit conditions for non-obvious cases

### Synchronous Functions

- [ ] Prefer sync over async
- [ ] Let callers add concurrency

### Race Detection

- [ ] Tested with `go test -race`

## Interfaces

### Interface Location

- [ ] Defined at point of use (consumer), not implementation
- [ ] No premature interfaces

```go
// GOOD - consumer defines interface
package consumer
type Thinger interface { Thing() bool }

// BAD - producer defines interface
package producer
type Thinger interface { Thing() bool }
func NewThinger() Thinger { ... }
```

### Return Types

- [ ] Return concrete types, not interfaces
- [ ] Let consumers define interfaces as needed

## Data Types

### Empty Slices

- [ ] Prefer `var t []string` (nil slice)
- [ ] Use `t := []string{}` only when needed (JSON encoding)

### Copying

- [ ] Don't copy values with pointer methods
- [ ] Be careful copying structs with slices (aliasing)

### Pass Values

- [ ] Don't pass pointers just to save bytes
- [ ] `*string` and `*io.Reader` usually wrong
- [ ] Pointers for large structs or mutation

### Crypto Rand

- [ ] `crypto/rand` for keys, not `math/rand`
- [ ] `crypto/rand.Text()` for random text

## Imports

### Import Organization

- [ ] Standard library first, blank line, then others
- [ ] Grouped with blank lines

```go
import (
    "fmt"
    "os"

    "github.com/foo/bar"
)
```

### Import Naming

- [ ] Avoid renaming imports unless collision
- [ ] Rename most local/project-specific import on collision

### Import Blank

- [ ] `import _ "pkg"` only in main package or tests

### Import Dot

- [ ] Only for external test packages with circular deps
- [ ] Never in regular code

## Testing

### Test Failures

- [ ] Helpful error messages
- [ ] Show what was wrong, inputs, actual vs expected
- [ ] Order: actual != expected, message matches

```go
if got != tt.want {
    t.Errorf("Foo(%q) = %d; want %d", tt.in, got, tt.want)
}
```

### Test Names

- [ ] `TestFunctionName` format
- [ ] Subtests with clear names

### Table-Driven Tests

- [ ] Use for multiple test cases
- [ ] Clear field names in test struct

## Named Results

- [ ] Avoid when repetitive in godoc
- [ ] Use when clarifying multiple same-type returns
- [ ] Naked returns only in very short functions

```go
// Avoid
func (n *Node) Parent1() (node *Node) {}

// Better
func (n *Node) Parent1() *Node {}

// OK when clarifying
func (f *Foo) Location() (lat, long float64, err error)
```

## In-Band Errors

- [ ] Return additional value for validity (error or bool)
- [ ] Don't use -1, "", nil as error indicators

```go
// GOOD
func Lookup(key string) (value string, ok bool)

// BAD
func Lookup(key string) string  // returns "" on not found
```

## Switch Statements

### Direct Evaluation

- [ ] Use switch with expressions for cleaner multi-way conditionals
- [ ] Prefer switch over long if-else chains
- [ ] Switch evaluates cases top-to-bottom, first match wins
- [ ] No need for break statements (they're automatic)

```go
// GOOD - switch with direct evaluation
switch status := getStatus(); status {
case StatusPending:
    return "pending"
case StatusActive:
    return "active"
case StatusDone:
    return "done"
default:
    return "unknown"
}

// BAD - long if-else chain
if status := getStatus(); status == StatusPending {
    return "pending"
} else if status == StatusActive {
    return "active"
} else if status == StatusDone {
    return "done"
} else {
    return "unknown"
}
```

### Type Switches

- [ ] Use type switches for interface type assertions
- [ ] Use comma-ok idiom for safe type assertion

```go
// Type switch
var i interface{} = "hello"
switch v := i.(type) {
case string:
    fmt.Println("string:", v)
case int:
    fmt.Println("int:", v)
default:
    fmt.Println("unknown type")
}
```

### Fallthrough

- [ ] Avoid fallthrough unless intentional
- [ ] Use comma-separated cases when multiple values need same handling

```go
// GOOD - comma-separated cases
switch kind {
case "admin", "owner", "superadmin":
    grantFullAccess()
}
```

## Modern Go Idioms (Go 1.21+)

### Standard Library Preferences

- [ ] Uses `slices.Sort` instead of `sort.Slice`
- [ ] Uses `slices.Sort` instead of `sort.Ints`/`sort.Float64s`/`sort.Strings`
- [ ] Uses `slices.Contains` instead of manual search loops
- [ ] Uses `slices.Equal` instead of manual slice comparison
- [ ] Uses `maps.Clone` instead of manual map copy loops
- [ ] Uses `maps.Equal` instead of manual map comparison
- [ ] Uses `maps.Keys`/`maps.Values` (Go 1.23+) for iteration

```go
// GOOD - modern idioms
slices.Sort(items)
if slices.Contains(items, target) { ... }
clone := maps.Clone(original)

// BAD - unnecessary manual implementation
sort.Slice(items, func(i, j int) bool { return items[i] < items[j] })
sort.Ints(items)
for _, item := range items {
    if item == target { found = true; break }
}
clone := make(map[K]V, len(original))
for k, v := range original { clone[k] = v }
```

### Generics

- [ ] Generic functions used for type-safe utilities
- [ ] Not over-generalized (use specific types when appropriate)
- [ ] `cmp.Ordered` constraint used for comparable types
- [ ] `any` constraint only when truly type-agnostic

### Time-Dependent Code

- [ ] Uses dependency injection for `time.Now()` calls
- [ ] Clock interface pattern for testable time handling
- [ ] No direct `time.Now()` in business logic

```go
// GOOD - testable time handling
type Clock interface { Now() time.Time }
type Service struct { clock Clock }

// BAD - untestable
func (s *Service) IsExpired() bool {
    return time.Now().After(s.expiresAt)
}
```

### Iterators (Go 1.23+)

- [ ] Custom iterators use `iter.Seq` or `iter.Seq2`
- [ ] Range over functions for custom iteration patterns
- [ ] Iterators return early when yield returns false

### WaitGroup.Go (Go 1.24+)

- [ ] Uses `sync.WaitGroup.Go` for fire-and-forget goroutines
- [ ] `wg.Go(fn)` is equivalent to `go func() { wg.Wait(); fn() }()` but simpler
- [ ] Preferred over `go func() { defer wg.Done(); fn() }()` for simpler error handling

```go
// GOOD - Go 1.24+ WaitGroup.Go pattern
var wg sync.WaitGroup
for _, task := range tasks {
    wg.Go(func() {
        process(task)  // err handled inside
    })
}
wg.Wait()

// ACCEPTABLE - traditional pattern still works
var wg sync.WaitGroup
for _, task := range tasks {
    wg.Add(1)
    go func(t Task) {
        defer wg.Done()
        process(t)
    }(task)
}
wg.Wait()
```

## Quick Checklist

For fast reviews, check these critical items:

1. **gofmt** - Code is formatted
2. **Errors handled** - No ignored errors
3. **Doc comments** - Exports documented
4. **Naming** - Short, idiomatic names
5. **Tests** - Exist and helpful failures
6. **No panics** - Error returns used
7. **Context** - First param when used
8. **Interfaces** - At consumer, not producer
9. **Modern idioms** - Uses slices/maps packages where applicable
10. **Switch statements** - Use switch for multi-way conditionals

## Review Output Format

```markdown
## Code Review: [scope]

### Critical Issues
[Must fix before merge]

### Important Issues
[Should fix]

### Minor Issues
[Nice to have]

### Strengths
[What's done well]

### Verdict: [PASS/NEEDS WORK]
```

## Go References for Idomatic Code

- https://pkg.go.dev/sync#WaitGroup.Go
- https://pkg.go.dev/slices
- https://pkg.go.dev/maps
- https://pkg.go.dev/iter
- https://go.dev/doc/tutorial/generics
- https://go.dev/blog/testing-time
