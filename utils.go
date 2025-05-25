package quickgraph

import (
	"fmt"
	"regexp"
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

// validVariableNameRegex defines the pattern for valid GraphQL variable names
// According to GraphQL spec, names must match /[_A-Za-z][_0-9A-Za-z]*/
var validVariableNameRegex = regexp.MustCompile(`^[_A-Za-z][_0-9A-Za-z]*$`)

// parseVariableName extracts and validates a variable name from a GraphQL variable reference.
// Variable references should start with '$' followed by a valid name.
// Returns the variable name without the '$' prefix and an error if invalid.
func parseVariableName(varRef string) (string, error) {
	if len(varRef) < 2 {
		return "", fmt.Errorf("invalid variable reference: %q (too short)", varRef)
	}

	if varRef[0] != '$' {
		return "", fmt.Errorf("invalid variable reference: %q (must start with '$')", varRef)
	}

	varName := varRef[1:]

	// Validate the variable name according to GraphQL naming rules
	if !validVariableNameRegex.MatchString(varName) {
		return "", fmt.Errorf("invalid variable name: %q (must match /[_A-Za-z][_0-9A-Za-z]*/)", varName)
	}

	return varName, nil
}
