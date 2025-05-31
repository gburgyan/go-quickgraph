package quickgraph

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for scalar function testing
type TestUserID string
type TestMoney int64
type TestEmail string

func TestSerializeScalarValue(t *testing.T) {
	ctx := context.Background()
	graphy := &Graphy{}

	// Register a custom scalar
	err := graphy.RegisterScalar(ctx, ScalarDefinition{
		Name:   "TestUserID",
		GoType: reflect.TypeOf(TestUserID("")),
		Serialize: func(value interface{}) (interface{}, error) {
			if uid, ok := value.(TestUserID); ok {
				return "uid_" + string(uid), nil
			}
			return nil, fmt.Errorf("expected TestUserID, got %T", value)
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			if str, ok := value.(string); ok {
				return TestUserID(str), nil
			}
			return nil, fmt.Errorf("expected string, got %T", value)
		},
	})
	require.NoError(t, err)

	tests := []struct {
		name           string
		value          interface{}
		scalarType     reflect.Type
		expectedResult interface{}
		expectError    bool
	}{
		{
			name:           "serialize registered scalar",
			value:          TestUserID("123"),
			scalarType:     reflect.TypeOf(TestUserID("")),
			expectedResult: "uid_123",
			expectError:    false,
		},
		{
			name:           "serialize non-scalar type returns value as-is",
			value:          "regular string",
			scalarType:     reflect.TypeOf(""),
			expectedResult: "regular string",
			expectError:    false,
		},
		{
			name:           "serialize int for non-scalar type",
			value:          42,
			scalarType:     reflect.TypeOf(42),
			expectedResult: 42,
			expectError:    false,
		},
		{
			name:        "serialize wrong type for registered scalar",
			value:       "not a TestUserID",
			scalarType:  reflect.TypeOf(TestUserID("")),
			expectError: true,
		},
		{
			name:        "serialize nil value",
			value:       nil,
			scalarType:  reflect.TypeOf(TestUserID("")),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := graphy.serializeScalarValue(tt.value, tt.scalarType)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestSerializeScalarValue_MultipleScalars(t *testing.T) {
	ctx := context.Background()
	graphy := &Graphy{}

	// Register multiple scalars
	err := graphy.RegisterScalar(ctx, ScalarDefinition{
		Name:   "TestUserID",
		GoType: reflect.TypeOf(TestUserID("")),
		Serialize: func(value interface{}) (interface{}, error) {
			if uid, ok := value.(TestUserID); ok {
				return "user:" + string(uid), nil
			}
			return nil, fmt.Errorf("expected TestUserID, got %T", value)
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			return TestUserID(""), nil
		},
	})
	require.NoError(t, err)

	err = graphy.RegisterScalar(ctx, ScalarDefinition{
		Name:   "TestMoney",
		GoType: reflect.TypeOf(TestMoney(0)),
		Serialize: func(value interface{}) (interface{}, error) {
			if money, ok := value.(TestMoney); ok {
				return fmt.Sprintf("$%.2f", float64(money)/100), nil
			}
			return nil, fmt.Errorf("expected TestMoney, got %T", value)
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			return TestMoney(0), nil
		},
	})
	require.NoError(t, err)

	// Test serialization of different scalar types
	result, err := graphy.serializeScalarValue(TestUserID("123"), reflect.TypeOf(TestUserID("")))
	assert.NoError(t, err)
	assert.Equal(t, "user:123", result)

	result, err = graphy.serializeScalarValue(TestMoney(2550), reflect.TypeOf(TestMoney(0)))
	assert.NoError(t, err)
	assert.Equal(t, "$25.50", result)
}

func TestParseScalarValue(t *testing.T) {
	ctx := context.Background()
	graphy := &Graphy{}

	// Register a custom scalar with validation
	err := graphy.RegisterScalar(ctx, ScalarDefinition{
		Name:   "TestEmail",
		GoType: reflect.TypeOf(TestEmail("")),
		Serialize: func(value interface{}) (interface{}, error) {
			if email, ok := value.(TestEmail); ok {
				return string(email), nil
			}
			return nil, fmt.Errorf("expected TestEmail, got %T", value)
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			if str, ok := value.(string); ok {
				if len(str) > 0 && str[0] == '@' {
					return nil, fmt.Errorf("email cannot start with @")
				}
				return TestEmail(str), nil
			}
			return nil, fmt.Errorf("expected string, got %T", value)
		},
	})
	require.NoError(t, err)

	pos := lexer.Position{Line: 1, Column: 10}

	tests := []struct {
		name           string
		value          interface{}
		targetType     reflect.Type
		expectedResult interface{}
		expectError    bool
		errorContains  string
	}{
		{
			name:           "parse valid email",
			value:          "user@example.com",
			targetType:     reflect.TypeOf(TestEmail("")),
			expectedResult: TestEmail("user@example.com"),
			expectError:    false,
		},
		{
			name:          "parse invalid email",
			value:         "@invalid.com",
			targetType:    reflect.TypeOf(TestEmail("")),
			expectError:   true,
			errorContains: "failed to parse TestEmail: email cannot start with @",
		},
		{
			name:          "parse wrong type for scalar",
			value:         42,
			targetType:    reflect.TypeOf(TestEmail("")),
			expectError:   true,
			errorContains: "failed to parse TestEmail: expected string, got int",
		},
		{
			name:           "parse non-scalar type returns value as-is",
			value:          "regular string",
			targetType:     reflect.TypeOf(""),
			expectedResult: "regular string",
			expectError:    false,
		},
		{
			name:           "parse int for non-scalar type",
			value:          42,
			targetType:     reflect.TypeOf(42),
			expectedResult: 42,
			expectError:    false,
		},
		{
			name:           "parse nil value for non-scalar",
			value:          nil,
			targetType:     reflect.TypeOf(""),
			expectedResult: nil,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := graphy.parseScalarValue(tt.value, tt.targetType, pos)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				// Check that error includes position information
				if graphErr, ok := err.(*GraphError); ok && len(graphErr.Locations) > 0 {
					assert.Equal(t, pos.Line, graphErr.Locations[0].Line)
					assert.Equal(t, pos.Column, graphErr.Locations[0].Column)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestParseScalarValue_BuiltInScalars(t *testing.T) {
	ctx := context.Background()
	graphy := &Graphy{}

	// Register built-in DateTime scalar
	err := graphy.RegisterDateTimeScalar(ctx)
	require.NoError(t, err)

	pos := lexer.Position{Line: 2, Column: 5}

	tests := []struct {
		name           string
		value          interface{}
		expectedResult interface{}
		expectError    bool
	}{
		{
			name:           "parse valid RFC3339 date",
			value:          "2023-12-25T15:30:45Z",
			expectedResult: time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC),
			expectError:    false,
		},
		{
			name:        "parse invalid date format",
			value:       "not-a-date",
			expectError: true,
		},
		{
			name:        "parse wrong type for DateTime",
			value:       12345,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := graphy.parseScalarValue(tt.value, reflect.TypeOf(time.Time{}), pos)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to parse DateTime")
			} else {
				assert.NoError(t, err)
				if expectedTime, ok := tt.expectedResult.(time.Time); ok {
					resultTime, ok := result.(time.Time)
					require.True(t, ok)
					assert.True(t, expectedTime.Equal(resultTime))
				} else {
					assert.Equal(t, tt.expectedResult, result)
				}
			}
		})
	}
}

func TestParseScalarLiteral(t *testing.T) {
	ctx := context.Background()
	graphy := &Graphy{}

	// Register a scalar with both ParseValue and ParseLiteral
	err := graphy.RegisterScalar(ctx, ScalarDefinition{
		Name:   "TestUserID",
		GoType: reflect.TypeOf(TestUserID("")),
		Serialize: func(value interface{}) (interface{}, error) {
			if uid, ok := value.(TestUserID); ok {
				return string(uid), nil
			}
			return nil, fmt.Errorf("expected TestUserID, got %T", value)
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			if str, ok := value.(string); ok {
				return TestUserID("var_" + str), nil // Different behavior for variables
			}
			return nil, fmt.Errorf("expected string, got %T", value)
		},
		ParseLiteral: func(value interface{}) (interface{}, error) {
			if str, ok := value.(string); ok {
				return TestUserID("lit_" + str), nil // Different behavior for literals
			}
			return nil, fmt.Errorf("expected string, got %T", value)
		},
	})
	require.NoError(t, err)

	// Register a scalar without ParseLiteral (should fall back to ParseValue)
	err = graphy.RegisterScalar(ctx, ScalarDefinition{
		Name:   "TestEmail",
		GoType: reflect.TypeOf(TestEmail("")),
		Serialize: func(value interface{}) (interface{}, error) {
			return string(value.(TestEmail)), nil
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			if str, ok := value.(string); ok {
				return TestEmail("parsed_" + str), nil
			}
			return nil, fmt.Errorf("expected string, got %T", value)
		},
		// ParseLiteral not provided - should use ParseValue as fallback
	})
	require.NoError(t, err)

	pos := lexer.Position{Line: 3, Column: 15}

	tests := []struct {
		name           string
		value          interface{}
		targetType     reflect.Type
		expectedResult interface{}
		expectError    bool
		errorContains  string
	}{
		{
			name:           "parse literal with custom ParseLiteral",
			value:          "123",
			targetType:     reflect.TypeOf(TestUserID("")),
			expectedResult: TestUserID("lit_123"),
			expectError:    false,
		},
		{
			name:           "parse literal with fallback to ParseValue",
			value:          "test@example.com",
			targetType:     reflect.TypeOf(TestEmail("")),
			expectedResult: TestEmail("parsed_test@example.com"),
			expectError:    false,
		},
		{
			name:          "parse literal with error",
			value:         42,
			targetType:    reflect.TypeOf(TestUserID("")),
			expectError:   true,
			errorContains: "failed to parse TestUserID literal: expected string, got int",
		},
		{
			name:           "parse non-scalar type returns value as-is",
			value:          "regular string",
			targetType:     reflect.TypeOf(""),
			expectedResult: "regular string",
			expectError:    false,
		},
		{
			name:           "parse complex type for non-scalar",
			value:          map[string]interface{}{"key": "value"},
			targetType:     reflect.TypeOf(map[string]interface{}{}),
			expectedResult: map[string]interface{}{"key": "value"},
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := graphy.parseScalarLiteral(tt.value, tt.targetType, pos)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				// Check that error includes position information
				if graphErr, ok := err.(*GraphError); ok && len(graphErr.Locations) > 0 {
					assert.Equal(t, pos.Line, graphErr.Locations[0].Line)
					assert.Equal(t, pos.Column, graphErr.Locations[0].Column)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestParseScalarLiteral_DifferenceFromParseValue(t *testing.T) {
	ctx := context.Background()
	graphy := &Graphy{}

	// Register a scalar where ParseLiteral and ParseValue behave differently
	err := graphy.RegisterScalar(ctx, ScalarDefinition{
		Name:   "TestMoney",
		GoType: reflect.TypeOf(TestMoney(0)),
		Serialize: func(value interface{}) (interface{}, error) {
			if money, ok := value.(TestMoney); ok {
				return int64(money), nil
			}
			return nil, fmt.Errorf("expected TestMoney, got %T", value)
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			// ParseValue expects cents as integer from variables
			if i, ok := value.(int64); ok {
				return TestMoney(i), nil
			}
			if i, ok := value.(int); ok {
				return TestMoney(i), nil
			}
			return nil, fmt.Errorf("expected int for variable, got %T", value)
		},
		ParseLiteral: func(value interface{}) (interface{}, error) {
			// ParseLiteral expects dollars as string from literals
			if _, ok := value.(string); ok {
				// Simple parsing: assume string is like "25.50"
				return TestMoney(2550), nil // Hardcoded for test
			}
			return nil, fmt.Errorf("expected string for literal, got %T", value)
		},
	})
	require.NoError(t, err)

	pos := lexer.Position{Line: 1, Column: 1}

	// Test ParseValue (for variables)
	result, err := graphy.parseScalarValue(2550, reflect.TypeOf(TestMoney(0)), pos)
	assert.NoError(t, err)
	assert.Equal(t, TestMoney(2550), result)

	// Test ParseLiteral (for literals)
	result, err = graphy.parseScalarLiteral("25.50", reflect.TypeOf(TestMoney(0)), pos)
	assert.NoError(t, err)
	assert.Equal(t, TestMoney(2550), result)

	// Test that they reject each other's expected input types
	_, err = graphy.parseScalarValue("25.50", reflect.TypeOf(TestMoney(0)), pos)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected int for variable")

	_, err = graphy.parseScalarLiteral(2550, reflect.TypeOf(TestMoney(0)), pos)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected string for literal")
}

func TestScalarFunctions_WithNoScalarsRegistered(t *testing.T) {
	graphy := &Graphy{}
	pos := lexer.Position{Line: 1, Column: 1}

	// Test all functions with no scalars registered
	result, err := graphy.serializeScalarValue("test", reflect.TypeOf(""))
	assert.NoError(t, err)
	assert.Equal(t, "test", result)

	result, err = graphy.parseScalarValue(42, reflect.TypeOf(42), pos)
	assert.NoError(t, err)
	assert.Equal(t, 42, result)

	result, err = graphy.parseScalarLiteral(true, reflect.TypeOf(true), pos)
	assert.NoError(t, err)
	assert.Equal(t, true, result)
}

func TestScalarFunctions_EdgeCases(t *testing.T) {
	ctx := context.Background()
	graphy := &Graphy{}
	pos := lexer.Position{Line: 5, Column: 20}

	// Register a scalar that can fail in various ways
	err := graphy.RegisterScalar(ctx, ScalarDefinition{
		Name:   "FailingScalar",
		GoType: reflect.TypeOf(TestUserID("")),
		Serialize: func(value interface{}) (interface{}, error) {
			if str, ok := value.(TestUserID); ok && string(str) == "serialize_error" {
				return nil, fmt.Errorf("serialize failed for test")
			}
			return string(value.(TestUserID)), nil
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			if str, ok := value.(string); ok && str == "parse_error" {
				return nil, fmt.Errorf("parse value failed for test")
			}
			return TestUserID(value.(string)), nil
		},
		ParseLiteral: func(value interface{}) (interface{}, error) {
			if str, ok := value.(string); ok {
				if str == "literal_error" {
					return nil, fmt.Errorf("parse literal failed for test")
				}
				return TestUserID(str), nil
			}
			return nil, fmt.Errorf("expected string, got %T", value)
		},
	})
	require.NoError(t, err)

	// Test serialization error
	_, err = graphy.serializeScalarValue(TestUserID("serialize_error"), reflect.TypeOf(TestUserID("")))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "serialize failed for test")

	// Test parse value error with proper error formatting
	_, err = graphy.parseScalarValue("parse_error", reflect.TypeOf(TestUserID("")), pos)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse FailingScalar: parse value failed for test")
	if graphErr, ok := err.(*GraphError); ok && len(graphErr.Locations) > 0 {
		assert.Equal(t, pos.Line, graphErr.Locations[0].Line)
		assert.Equal(t, pos.Column, graphErr.Locations[0].Column)
	}

	// Test parse literal error with proper error formatting
	_, err = graphy.parseScalarLiteral("literal_error", reflect.TypeOf(TestUserID("")), pos)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse FailingScalar literal: parse literal failed for test")
	if graphErr, ok := err.(*GraphError); ok && len(graphErr.Locations) > 0 {
		assert.Equal(t, pos.Line, graphErr.Locations[0].Line)
		assert.Equal(t, pos.Column, graphErr.Locations[0].Column)
	}
}
