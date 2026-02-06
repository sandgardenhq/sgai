---
name: Go Generics Patterns
description: Common generic patterns for type-safe reusable code (Go 1.18+); When writing reusable functions or data structures that work with multiple types
---

/* Demonstrates common Go generics patterns.
 * Requires Go 1.18+.
 * Documentation: https://go.dev/doc/tutorial/generics
 * Blog: https://go.dev/blog/generic-interfaces
 */
package main

import (
	"cmp"
	"fmt"
)

// --- Generic Functions ---

// Min returns the smaller of two ordered values
// Uses cmp.Ordered constraint for <, >, == support
func Min[T cmp.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Max returns the larger of two ordered values
func Max[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Clamp restricts a value to a range [min, max]
func Clamp[T cmp.Ordered](value, min, max T) T {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// --- Generic Data Structures ---

// Stack is a generic LIFO data structure
type Stack[T any] struct {
	items []T
}

func NewStack[T any]() *Stack[T] {
	return &Stack[T]{}
}

func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	item := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return item, true
}

func (s *Stack[T]) Peek() (T, bool) {
	if len(s.items) == 0 {
		var zero T
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

func (s *Stack[T]) Len() int {
	return len(s.items)
}

// Pair holds two values of different types
type Pair[K, V any] struct {
	Key   K
	Value V
}

// Result represents success or failure with a value
type Result[T any] struct {
	value T
	err   error
}

func Ok[T any](value T) Result[T] {
	return Result[T]{value: value}
}

func Err[T any](err error) Result[T] {
	return Result[T]{err: err}
}

func (r Result[T]) Unwrap() (T, error) {
	return r.value, r.err
}

func (r Result[T]) IsOk() bool {
	return r.err == nil
}

// --- Custom Type Constraints ---

// Number constraint for numeric types
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Sum adds all numbers in a slice
func Sum[T Number](nums []T) T {
	var sum T
	for _, n := range nums {
		sum += n
	}
	return sum
}

// Average calculates the mean of numbers
func Average[T Number](nums []T) float64 {
	if len(nums) == 0 {
		return 0
	}
	var sum T
	for _, n := range nums {
		sum += n
	}
	return float64(sum) / float64(len(nums))
}

// Stringer constraint for types that can be converted to string
type Stringer interface {
	String() string
}

// --- Generic Collection Functions ---

// Map applies a function to each element
func Map[T, U any](items []T, fn func(T) U) []U {
	result := make([]U, len(items))
	for i, item := range items {
		result[i] = fn(item)
	}
	return result
}

// Filter returns elements that pass the predicate
func Filter[T any](items []T, pred func(T) bool) []T {
	var result []T
	for _, item := range items {
		if pred(item) {
			result = append(result, item)
		}
	}
	return result
}

// Reduce combines elements into a single value
func Reduce[T, U any](items []T, initial U, fn func(U, T) U) U {
	result := initial
	for _, item := range items {
		result = fn(result, item)
	}
	return result
}

// GroupBy groups elements by a key function
func GroupBy[T any, K comparable](items []T, keyFn func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, item := range items {
		key := keyFn(item)
		result[key] = append(result[key], item)
	}
	return result
}

// --- Generic Pointer Helpers ---

// Ptr returns a pointer to the value (useful for optional fields)
func Ptr[T any](v T) *T {
	return &v
}

// ValueOr returns the value if non-nil, otherwise the default
func ValueOr[T any](ptr *T, defaultValue T) T {
	if ptr != nil {
		return *ptr
	}
	return defaultValue
}

// --- Generic Cache ---

type Cache[K comparable, V any] struct {
	data map[K]V
}

func NewCache[K comparable, V any]() *Cache[K, V] {
	return &Cache[K, V]{data: make(map[K]V)}
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
	v, ok := c.data[key]
	return v, ok
}

func (c *Cache[K, V]) Set(key K, value V) {
	c.data[key] = value
}

func (c *Cache[K, V]) GetOrSet(key K, fn func() V) V {
	if v, ok := c.data[key]; ok {
		return v
	}
	v := fn()
	c.data[key] = v
	return v
}

func main() {
	// Generic functions
	fmt.Println("Min(3, 5):", Min(3, 5))             // 3
	fmt.Println("Max(3.14, 2.71):", Max(3.14, 2.71)) // 3.14
	fmt.Println("Clamp(15, 0, 10):", Clamp(15, 0, 10)) // 10

	// Generic stack
	stack := NewStack[string]()
	stack.Push("first")
	stack.Push("second")
	if v, ok := stack.Pop(); ok {
		fmt.Println("Popped:", v) // "second"
	}

	// Collection functions
	nums := []int{1, 2, 3, 4, 5}
	doubled := Map(nums, func(n int) int { return n * 2 })
	fmt.Println("Doubled:", doubled) // [2, 4, 6, 8, 10]

	evens := Filter(nums, func(n int) bool { return n%2 == 0 })
	fmt.Println("Evens:", evens) // [2, 4]

	sum := Reduce(nums, 0, func(acc, n int) int { return acc + n })
	fmt.Println("Sum:", sum) // 15

	// Number constraint
	fmt.Println("Sum[]:", Sum(nums))       // 15
	fmt.Println("Average:", Average(nums)) // 3.0

	// Generic cache
	cache := NewCache[string, int]()
	cache.Set("answer", 42)
	if v, ok := cache.Get("answer"); ok {
		fmt.Println("Cached:", v) // 42
	}

	// Pointer helpers
	name := Ptr("Alice")
	fmt.Println("Name:", ValueOr(name, "Unknown")) // Alice
	fmt.Println("Nil:", ValueOr[string](nil, "Unknown")) // Unknown

	// GroupBy
	type Person struct {
		Name string
		City string
	}
	people := []Person{
		{"Alice", "NYC"},
		{"Bob", "LA"},
		{"Carol", "NYC"},
	}
	byCity := GroupBy(people, func(p Person) string { return p.City })
	fmt.Println("By city:", byCity)
}
