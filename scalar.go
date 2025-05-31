package quickgraph

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/alecthomas/participle/v2/lexer"
)

// ScalarDefinition defines a custom scalar type with its serialization and parsing functions.
//
// Custom scalars allow you to map Go types to GraphQL scalar types with custom serialization
// logic. This is useful for types like time.Time, custom ID types, or complex data structures
// that should be represented as scalars in GraphQL.
//
// Example usage:
//
//	type UserID string
//
//	graphy.RegisterScalar(ctx, ScalarDefinition{
//	    Name:   "UserID",
//	    GoType: reflect.TypeOf(UserID("")),
//	    Serialize: func(value interface{}) (interface{}, error) {
//	        if uid, ok := value.(UserID); ok {
//	            return string(uid), nil
//	        }
//	        return nil, fmt.Errorf("expected UserID, got %T", value)
//	    },
//	    ParseValue: func(value interface{}) (interface{}, error) {
//	        if str, ok := value.(string); ok {
//	            return UserID(str), nil
//	        }
//	        return nil, fmt.Errorf("expected string, got %T", value)
//	    },
//	})
type ScalarDefinition struct {
	// Name is the GraphQL scalar type name that will appear in the schema
	Name string

	// GoType is the Go type that this scalar represents
	GoType reflect.Type

	// Description is an optional description for the scalar type
	Description string

	// Serialize converts a Go value to a JSON-serializable value for output
	// The input value will be of the GoType, and the output should be a basic JSON type
	// (string, int, float64, bool, nil)
	Serialize func(value interface{}) (interface{}, error)

	// ParseValue converts a JSON value from variables to a Go value
	// The input value comes from GraphQL variables and should be converted to GoType
	ParseValue func(value interface{}) (interface{}, error)

	// ParseLiteral converts a GraphQL literal value to a Go value
	// This is called when the value appears directly in the query (not from variables)
	// If not provided, ParseValue will be used as a fallback
	ParseLiteral func(value interface{}) (interface{}, error)
}

// scalarRegistry holds all registered custom scalars
type scalarRegistry struct {
	mu     sync.RWMutex
	byName map[string]*ScalarDefinition
	byType map[reflect.Type]*ScalarDefinition
}

// RegisterScalar registers a custom scalar type.
//
// Once registered, any Go type matching the scalar's GoType will be treated as this
// scalar in GraphQL schemas and during request processing.
//
// Parameters:
//   - ctx: Context for the operation
//   - definition: The scalar definition containing type mapping and conversion functions
//
// Example:
//
//	// Register a DateTime scalar for time.Time
//	graphy.RegisterScalar(ctx, ScalarDefinition{
//	    Name:   "DateTime",
//	    GoType: reflect.TypeOf(time.Time{}),
//	    Serialize: func(value interface{}) (interface{}, error) {
//	        if t, ok := value.(time.Time); ok {
//	            return t.Format(time.RFC3339), nil
//	        }
//	        return nil, fmt.Errorf("expected time.Time, got %T", value)
//	    },
//	    ParseValue: func(value interface{}) (interface{}, error) {
//	        if str, ok := value.(string); ok {
//	            return time.Parse(time.RFC3339, str)
//	        }
//	        return nil, fmt.Errorf("expected string, got %T", value)
//	    },
//	})
func (g *Graphy) RegisterScalar(ctx context.Context, definition ScalarDefinition) error {
	g.structureLock.Lock()
	defer g.structureLock.Unlock()

	if err := validateScalarDefinition(definition); err != nil {
		return fmt.Errorf("invalid scalar definition: %w", err)
	}

	g.ensureScalarRegistry()

	g.scalars.mu.Lock()
	defer g.scalars.mu.Unlock()

	// Check for name conflicts
	if existing, exists := g.scalars.byName[definition.Name]; exists {
		return fmt.Errorf("scalar with name %q already registered for type %v", definition.Name, existing.GoType)
	}

	// Check for type conflicts
	if existing, exists := g.scalars.byType[definition.GoType]; exists {
		return fmt.Errorf("scalar for type %v already registered with name %q", definition.GoType, existing.Name)
	}

	// If ParseLiteral is not provided, use ParseValue as fallback
	if definition.ParseLiteral == nil {
		definition.ParseLiteral = definition.ParseValue
	}

	// Store the definition
	g.scalars.byName[definition.Name] = &definition
	g.scalars.byType[definition.GoType] = &definition

	// Clear schema buffer and type lookup cache to force regeneration
	g.schemaBuffer = nil

	// Clear any existing type lookup for this type to force re-evaluation
	g.typeMutex.Lock()
	if g.typeLookups != nil {
		delete(g.typeLookups, definition.GoType)
		// Also check for pointer type
		delete(g.typeLookups, reflect.PtrTo(definition.GoType))
	}
	g.typeMutex.Unlock()

	return nil
}

// GetScalarByName returns the scalar definition for the given GraphQL scalar name
func (g *Graphy) GetScalarByName(name string) (*ScalarDefinition, bool) {
	if g.scalars == nil {
		return nil, false
	}

	g.scalars.mu.RLock()
	defer g.scalars.mu.RUnlock()

	scalar, exists := g.scalars.byName[name]
	return scalar, exists
}

// GetScalarByType returns the scalar definition for the given Go type
func (g *Graphy) GetScalarByType(goType reflect.Type) (*ScalarDefinition, bool) {
	if g.scalars == nil {
		return nil, false
	}

	g.scalars.mu.RLock()
	defer g.scalars.mu.RUnlock()

	scalar, exists := g.scalars.byType[goType]
	return scalar, exists
}

// GetRegisteredScalars returns a map of all registered scalar names to their definitions
func (g *Graphy) GetRegisteredScalars() map[string]*ScalarDefinition {
	if g.scalars == nil {
		return make(map[string]*ScalarDefinition)
	}

	g.scalars.mu.RLock()
	defer g.scalars.mu.RUnlock()

	result := make(map[string]*ScalarDefinition, len(g.scalars.byName))
	for name, def := range g.scalars.byName {
		result[name] = def
	}
	return result
}

// serializeScalarValue serializes a value using the registered scalar's Serialize function
func (g *Graphy) serializeScalarValue(value interface{}, scalarType reflect.Type) (interface{}, error) {
	if scalar, exists := g.GetScalarByType(scalarType); exists {
		return scalar.Serialize(value)
	}
	return value, nil
}

// parseScalarValue parses a value using the registered scalar's ParseValue function
func (g *Graphy) parseScalarValue(value interface{}, targetType reflect.Type, pos lexer.Position) (interface{}, error) {
	if scalar, exists := g.GetScalarByType(targetType); exists {
		parsed, err := scalar.ParseValue(value)
		if err != nil {
			return nil, NewGraphError(fmt.Sprintf("failed to parse %s: %v", scalar.Name, err), pos)
		}
		return parsed, nil
	}
	return value, nil
}

// parseScalarLiteral parses a literal value using the registered scalar's ParseLiteral function
func (g *Graphy) parseScalarLiteral(value interface{}, targetType reflect.Type, pos lexer.Position) (interface{}, error) {
	if scalar, exists := g.GetScalarByType(targetType); exists {
		parsed, err := scalar.ParseLiteral(value)
		if err != nil {
			return nil, NewGraphError(fmt.Sprintf("failed to parse %s literal: %v", scalar.Name, err), pos)
		}
		return parsed, nil
	}
	return value, nil
}

// ensureScalarRegistry initializes the scalar registry if needed
func (g *Graphy) ensureScalarRegistry() {
	if g.scalars == nil {
		g.scalars = &scalarRegistry{
			byName: make(map[string]*ScalarDefinition),
			byType: make(map[reflect.Type]*ScalarDefinition),
		}
	}
}

// validateScalarDefinition validates that a scalar definition is complete and valid
func validateScalarDefinition(def ScalarDefinition) error {
	if def.Name == "" {
		return fmt.Errorf("scalar name cannot be empty")
	}

	if def.GoType == nil {
		return fmt.Errorf("scalar GoType cannot be nil")
	}

	if def.Serialize == nil {
		return fmt.Errorf("scalar Serialize function cannot be nil")
	}

	if def.ParseValue == nil {
		return fmt.Errorf("scalar ParseValue function cannot be nil")
	}

	// Validate that the scalar name follows GraphQL naming conventions
	if !isValidGraphQLName(def.Name) {
		return fmt.Errorf("scalar name %q is not a valid GraphQL name", def.Name)
	}

	return nil
}

// isValidGraphQLName checks if a name follows GraphQL naming conventions
func isValidGraphQLName(name string) bool {
	if len(name) == 0 {
		return false
	}

	// First character must be a letter or underscore
	first := name[0]
	if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_') {
		return false
	}

	// Remaining characters must be letters, digits, or underscores
	for i := 1; i < len(name); i++ {
		c := name[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}

// Built-in scalar definitions for common types

// RegisterDateTimeScalar registers a DateTime scalar for time.Time
func (g *Graphy) RegisterDateTimeScalar(ctx context.Context) error {
	return g.RegisterScalar(ctx, ScalarDefinition{
		Name:        "DateTime",
		GoType:      reflect.TypeOf(time.Time{}),
		Description: "RFC3339 formatted date-time string",
		Serialize: func(value interface{}) (interface{}, error) {
			if t, ok := value.(time.Time); ok {
				return t.Format(time.RFC3339), nil
			}
			if ptr, ok := value.(*time.Time); ok && ptr != nil {
				return ptr.Format(time.RFC3339), nil
			}
			return nil, fmt.Errorf("expected time.Time or *time.Time, got %T", value)
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			if str, ok := value.(string); ok {
				return time.Parse(time.RFC3339, str)
			}
			return nil, fmt.Errorf("expected string for DateTime, got %T", value)
		},
	})
}

// RegisterJSONScalar registers a JSON scalar for arbitrary JSON data
func (g *Graphy) RegisterJSONScalar(ctx context.Context) error {
	return g.RegisterScalar(ctx, ScalarDefinition{
		Name:        "JSON",
		GoType:      reflect.TypeOf(map[string]interface{}{}),
		Description: "Arbitrary JSON data",
		Serialize: func(value interface{}) (interface{}, error) {
			// JSON scalar passes through the value as-is since it's already JSON-compatible
			return value, nil
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			// For variables, the value is already parsed JSON
			return value, nil
		},
	})
}
