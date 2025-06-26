package quickgraph

import "context"

// Validator is an interface that types can implement to provide custom validation logic.
// The Validate method will be called automatically after the type is parsed from input.
type Validator interface {
	// Validate checks if the receiver value is valid.
	// It should return an error if validation fails, or nil if validation passes.
	Validate() error
}

// ValidatorWithContext is an interface that types can implement to provide context-aware validation.
// This is useful when validation requires access to request context (e.g., for authentication,
// database lookups, or other contextual information).
type ValidatorWithContext interface {
	// ValidateWithContext checks if the receiver value is valid within the given context.
	// It should return an error if validation fails, or nil if validation passes.
	ValidateWithContext(ctx context.Context) error
}