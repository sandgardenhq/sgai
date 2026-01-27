---
name: Go Iterators
description: Range-over-function iterators (Go 1.23+)
when_to_use: When creating custom iteration patterns using iter.Seq and iter.Seq2
---

/* Demonstrates Go 1.23+ range-over-function iterators.
 * Requires Go 1.23+.
 * Documentation: https://pkg.go.dev/iter
 * Release notes: https://go.dev/doc/go1.23#iterators
 */
package main

import (
	"fmt"
	"iter"
)

// --- Basic Iterator (iter.Seq) ---
// Returns single values per iteration

// Backward iterates over a slice in reverse order
func Backward[E any](s []E) iter.Seq[E] {
	return func(yield func(E) bool) {
		for i := len(s) - 1; i >= 0; i-- {
			if !yield(s[i]) {
				return // Stop if caller breaks early
			}
		}
	}
}

// Repeat yields a value n times
func Repeat[E any](value E, n int) iter.Seq[E] {
	return func(yield func(E) bool) {
		for i := 0; i < n; i++ {
			if !yield(value) {
				return
			}
		}
	}
}

// Filter yields only elements that pass the predicate
func Filter[E any](s []E, pred func(E) bool) iter.Seq[E] {
	return func(yield func(E) bool) {
		for _, v := range s {
			if pred(v) {
				if !yield(v) {
					return
				}
			}
		}
	}
}

// --- Key-Value Iterator (iter.Seq2) ---
// Returns two values per iteration (like map ranging)

// Enumerate yields index-value pairs
func Enumerate[E any](s []E) iter.Seq2[int, E] {
	return func(yield func(int, E) bool) {
		for i, v := range s {
			if !yield(i, v) {
				return
			}
		}
	}
}

// Zip combines two slices into key-value pairs
func Zip[K, V any](keys []K, values []V) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		minLen := len(keys)
		if len(values) < minLen {
			minLen = len(values)
		}
		for i := 0; i < minLen; i++ {
			if !yield(keys[i], values[i]) {
				return
			}
		}
	}
}

// MapEntries iterates over map with guaranteed order
func MapEntries[K comparable, V any](m map[K]V, keys []K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, k := range keys {
			if v, ok := m[k]; ok {
				if !yield(k, v) {
					return
				}
			}
		}
	}
}

// --- Transformation Iterators ---

// Map transforms each element using the given function
func Map[In, Out any](s []In, fn func(In) Out) iter.Seq[Out] {
	return func(yield func(Out) bool) {
		for _, v := range s {
			if !yield(fn(v)) {
				return
			}
		}
	}
}

// --- Using with slices.Collect (Go 1.23+) ---
// Collect materializes an iterator into a slice
// import "slices"
// result := slices.Collect(Filter(nums, func(n int) bool { return n > 5 }))

func main() {
	nums := []int{1, 2, 3, 4, 5}

	// Basic iteration
	fmt.Println("Backward:")
	for v := range Backward(nums) {
		fmt.Println(v) // 5, 4, 3, 2, 1
	}

	// Key-value iteration
	fmt.Println("\nEnumerate:")
	for i, v := range Enumerate(nums) {
		fmt.Printf("  index=%d, value=%d\n", i, v)
	}

	// Filtering
	fmt.Println("\nFiltered (> 2):")
	for v := range Filter(nums, func(n int) bool { return n > 2 }) {
		fmt.Println(v) // 3, 4, 5
	}

	// Transformation
	fmt.Println("\nDoubled:")
	for v := range Map(nums, func(n int) int { return n * 2 }) {
		fmt.Println(v) // 2, 4, 6, 8, 10
	}

	// Zipping two slices
	names := []string{"Alice", "Bob", "Carol"}
	ages := []int{30, 25, 35}
	fmt.Println("\nZipped:")
	for name, age := range Zip(names, ages) {
		fmt.Printf("  %s: %d\n", name, age)
	}

	// Early break works correctly
	fmt.Println("\nEarly break (first 2 backward):")
	count := 0
	for v := range Backward(nums) {
		fmt.Println(v)
		count++
		if count >= 2 {
			break // Iterator handles cleanup properly
		}
	}

	// Repeat
	fmt.Println("\nRepeat 'hello' 3 times:")
	for s := range Repeat("hello", 3) {
		fmt.Println(s)
	}
}
