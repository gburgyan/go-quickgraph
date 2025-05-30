package quickgraph

// TypeDiscoverable is an optional interface that types can implement to enable
// runtime type discovery. This is useful for GraphQL interfaces where a base
// type pointer needs to be resolved to its actual derived type.
type TypeDiscoverable interface {
	ActualType() interface{}
}

// Discover attempts to discover the actual type of a value that implements TypeDiscoverable.
// It returns the discovered type and true if successful, or nil and false if the value
// doesn't implement TypeDiscoverable or if type assertion fails.
func Discover[T any](value interface{}) (T, bool) {
	var zero T

	// Check if the value implements TypeDiscoverable
	if discoverable, ok := value.(TypeDiscoverable); ok {
		// Get the actual type
		if actual := discoverable.ActualType(); actual != nil {
			// Try to assert to the requested type
			if typed, ok := actual.(T); ok {
				return typed, true
			}
		}
	}

	// Try direct type assertion as fallback
	if typed, ok := value.(T); ok {
		return typed, true
	}

	return zero, false
}

// DiscoverType is a non-generic version that returns the actual type as interface{}
func DiscoverType(value interface{}) (interface{}, bool) {
	if discoverable, ok := value.(TypeDiscoverable); ok {
		if actual := discoverable.ActualType(); actual != nil {
			return actual, true
		}
	}
	return value, true // Return the original value if not discoverable
}
