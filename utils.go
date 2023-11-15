package quickgraph

import (
	"fmt"
	"sort"
)

// keys is a generic function that takes a map and returns a slice of its keys.
// The keys are not sorted and their order is not guaranteed to be consistent.
func keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}

// sortedKeys is a function that takes a map with string keys and returns a slice of
// its keys sorted in ascending order. The function uses the sort.Strings function
// from the sort package to sort the keys.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

// toStringSlice is a function that takes a slice of items that implement the fmt.Stringer
// interface and returns a slice of their string representations. The function uses the
// String method of the fmt.Stringer interface to get the string representation of each item.
func toStringSlice[T fmt.Stringer](items []T) []string {
	result := make([]string, len(items))
	for i, item := range items {
		result[i] = item.String()
	}
	return result
}
