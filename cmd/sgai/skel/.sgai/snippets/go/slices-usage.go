---
name: Slices Package Usage
description: Modern Go slices package operations (Go 1.21+)
when_to_use: When sorting, searching, or manipulating slices - prefer over manual loops and sort.Slice
---

/* Demonstrates the slices package for common slice operations.
 * Requires Go 1.21+. For Go 1.23+, also shows iterator-based functions.
 * Documentation: https://pkg.go.dev/slices
 */
package main

import (
	"fmt"
	"slices"
)

func main() {
	// --- Sorting ---
	// Prefer slices.Sort over sort.Slice
	nums := []int{3, 1, 4, 1, 5, 9, 2, 6}
	slices.Sort(nums)
	fmt.Println("Sorted:", nums) // [1 1 2 3 4 5 6 9]

	// Sort with custom comparison
	type Person struct {
		Name string
		Age  int
	}
	people := []Person{{"Alice", 30}, {"Bob", 25}, {"Carol", 35}}
	slices.SortFunc(people, func(a, b Person) int {
		return a.Age - b.Age // Sort by age ascending
	})
	fmt.Println("Sorted by age:", people)

	// --- Searching ---
	// Prefer slices.Contains over manual loops
	fruits := []string{"apple", "banana", "cherry"}
	if slices.Contains(fruits, "banana") {
		fmt.Println("Found banana!")
	}

	// Find index of element
	idx := slices.Index(fruits, "cherry")
	fmt.Println("Cherry at index:", idx) // 2

	// Find with custom predicate
	idx = slices.IndexFunc(people, func(p Person) bool {
		return p.Age > 28
	})
	fmt.Println("First person over 28 at index:", idx)

	// Binary search (slice must be sorted)
	sortedNums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	pos, found := slices.BinarySearch(sortedNums, 5)
	fmt.Printf("BinarySearch(5): pos=%d, found=%v\n", pos, found)

	// --- Comparison ---
	// Prefer slices.Equal over manual comparison
	a := []int{1, 2, 3}
	b := []int{1, 2, 3}
	c := []int{1, 2, 4}
	fmt.Printf("a == b: %v\n", slices.Equal(a, b)) // true
	fmt.Printf("a == c: %v\n", slices.Equal(a, c)) // false

	// --- Deduplication ---
	// Compact removes consecutive duplicates (sort first for full dedup)
	dupes := []int{1, 2, 2, 3, 3, 3, 4}
	deduped := slices.Compact(dupes)
	fmt.Println("Compacted:", deduped) // [1 2 3 4]

	// --- Cloning ---
	original := []string{"a", "b", "c"}
	clone := slices.Clone(original)
	fmt.Println("Clone:", clone)

	// --- Min/Max ---
	numbers := []int{5, 2, 8, 1, 9}
	fmt.Println("Min:", slices.Min(numbers)) // 1
	fmt.Println("Max:", slices.Max(numbers)) // 9

	// --- Reverse ---
	toReverse := []int{1, 2, 3, 4, 5}
	slices.Reverse(toReverse)
	fmt.Println("Reversed:", toReverse) // [5 4 3 2 1]

	// --- Insert/Delete ---
	s := []int{1, 2, 5, 6}
	s = slices.Insert(s, 2, 3, 4) // Insert 3, 4 at index 2
	fmt.Println("After insert:", s)

	s = slices.Delete(s, 1, 3) // Delete elements [1:3)
	fmt.Println("After delete:", s)

	// --- Grow/Clip ---
	// Grow increases capacity without changing length
	grown := slices.Grow([]int{1, 2, 3}, 10)
	fmt.Printf("Grown: len=%d, cap=%d\n", len(grown), cap(grown))

	// Clip reduces capacity to length (releases unused memory)
	clipped := slices.Clip(grown)
	fmt.Printf("Clipped: len=%d, cap=%d\n", len(clipped), cap(clipped))
}
