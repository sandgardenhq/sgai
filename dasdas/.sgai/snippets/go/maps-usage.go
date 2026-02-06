---
name: Maps Package Usage
description: Modern Go maps package operations (Go 1.21+); When cloning, comparing, or iterating over maps - prefer over manual loops
---

/* Demonstrates the maps package for common map operations.
 * Requires Go 1.21+. For Go 1.23+, also shows iterator-based functions.
 * Documentation: https://pkg.go.dev/maps
 */
package main

import (
	"fmt"
	"maps"
)

func main() {
	// --- Cloning ---
	// Prefer maps.Clone over manual copy loop
	original := map[string]int{
		"alice": 30,
		"bob":   25,
		"carol": 35,
	}

	// GOOD - use maps.Clone
	clone := maps.Clone(original)
	fmt.Println("Clone:", clone)

	// BAD - manual copy (don't do this anymore)
	// manualClone := make(map[string]int, len(original))
	// for k, v := range original { manualClone[k] = v }

	// --- Copying ---
	// Copy all key-value pairs from src to dst
	dst := map[string]int{"david": 40}
	maps.Copy(dst, original) // Copies original into dst
	fmt.Println("After Copy:", dst)

	// --- Comparison ---
	// Prefer maps.Equal over manual comparison
	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"a": 1, "b": 2}
	m3 := map[string]int{"a": 1, "b": 3}

	fmt.Printf("m1 == m2: %v\n", maps.Equal(m1, m2)) // true
	fmt.Printf("m1 == m3: %v\n", maps.Equal(m1, m3)) // false

	// EqualFunc for custom comparison
	type User struct{ Name string }
	users1 := map[int]User{1: {"Alice"}, 2: {"Bob"}}
	users2 := map[int]User{1: {"Alice"}, 2: {"Bob"}}
	equal := maps.EqualFunc(users1, users2, func(a, b User) bool {
		return a.Name == b.Name
	})
	fmt.Printf("users equal: %v\n", equal)

	// --- Delete by Predicate ---
	// DeleteFunc removes entries where the function returns true
	scores := map[string]int{
		"alice": 85,
		"bob":   65,
		"carol": 90,
		"david": 55,
	}
	// Remove all entries with score < 70
	maps.DeleteFunc(scores, func(k string, v int) bool {
		return v < 70
	})
	fmt.Println("After DeleteFunc:", scores) // Only alice and carol remain

	// --- Keys and Values (Go 1.23+) ---
	// These return iterators that work with range
	ages := map[string]int{
		"alice": 30,
		"bob":   25,
	}

	// Iterate over keys only
	fmt.Print("Keys: ")
	for k := range maps.Keys(ages) {
		fmt.Print(k, " ")
	}
	fmt.Println()

	// Iterate over values only
	fmt.Print("Values: ")
	for v := range maps.Values(ages) {
		fmt.Print(v, " ")
	}
	fmt.Println()

	// Collect keys into a slice (Go 1.23+ with slices.Collect)
	// import "slices"
	// keys := slices.Collect(maps.Keys(ages))

	// --- Practical Patterns ---

	// Pattern: Merge multiple maps
	defaults := map[string]string{"env": "dev", "log": "info"}
	overrides := map[string]string{"log": "debug"}
	config := maps.Clone(defaults)
	maps.Copy(config, overrides) // overrides win
	fmt.Println("Merged config:", config)

	// Pattern: Deep clone for maps of slices (maps.Clone is shallow!)
	nested := map[string][]int{"a": {1, 2, 3}}
	shallowClone := maps.Clone(nested)
	shallowClone["a"][0] = 999                                       // Modifies original too!
	fmt.Println("Original after shallow clone modify:", nested["a"]) // [999 2 3]

	// For deep clone, you need custom logic:
	deepClone := make(map[string][]int, len(nested))
	for k, v := range nested {
		deepClone[k] = append([]int{}, v...) // Clone the slice too
	}

	// Pattern: Create lookup map from slice
	type User2 struct {
		ID   int
		Name string
	}
	users := []User2{{1, "Alice"}, {2, "Bob"}}
	userByID := make(map[int]User2, len(users))
	for _, u := range users {
		userByID[u.ID] = u
	}
	fmt.Println("User lookup:", userByID[1])
}
