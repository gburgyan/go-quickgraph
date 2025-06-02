package quickgraph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductionModeErrorSanitization(t *testing.T) {
	tests := []struct {
		name                string
		productionMode      bool
		errorMessage        string
		productionMessage   string
		hasInnerError       bool
		innerError          string
		extensions          map[string]string
		sensitiveExtensions map[string]string
		expectSanitized     bool
	}{
		{
			name:           "development mode shows all details",
			productionMode: false,
			errorMessage:   "function testFunction panicked: runtime error",
			hasInnerError:  true,
			innerError:     "nil pointer dereference",
			extensions: map[string]string{
				"line": "5",
			},
			sensitiveExtensions: map[string]string{
				"stack":         "goroutine 1 [running]:\nmain.go:123",
				"function_name": "testFunction",
			},
			expectSanitized: false,
		},
		{
			name:              "production mode sanitizes sensitive info",
			productionMode:    true,
			errorMessage:      "function testFunction panicked: runtime error",
			productionMessage: "Internal server error",
			hasInnerError:     true,
			innerError:        "nil pointer dereference",
			extensions: map[string]string{
				"line": "5",
			},
			sensitiveExtensions: map[string]string{
				"stack":         "goroutine 1 [running]:\nmain.go:123",
				"function_name": "testFunction",
			},
			expectSanitized: true,
		},
		{
			name:              "production mode keeps safe extensions",
			productionMode:    true,
			errorMessage:      "Invalid query syntax",
			productionMessage: "Invalid query syntax", // Safe message
			extensions: map[string]string{
				"line":       "5",
				"column":     "10",
				"safe_field": "safe_value",
			},
			sensitiveExtensions: map[string]string{
				"stack": "sensitive stack trace",
			},
			expectSanitized: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gErr GraphError
			gErr.Message = tt.errorMessage
			gErr.ProductionMessage = tt.productionMessage
			if tt.hasInnerError {
				gErr.InnerError = fmt.Errorf(tt.innerError)
			}
			gErr.Extensions = tt.extensions
			gErr.SensitiveExtensions = tt.sensitiveExtensions

			var jsonBytes []byte
			var err error
			if tt.productionMode {
				jsonBytes, err = gErr.MarshalJSONProduction()
			} else {
				jsonBytes, err = gErr.MarshalJSON()
			}
			require.NoError(t, err)

			var result map[string]interface{}
			err = json.Unmarshal(jsonBytes, &result)
			require.NoError(t, err)

			if tt.expectSanitized {
				// In production mode, sensitive information should be filtered out
				if extensions, exists := result["extensions"]; exists {
					extMap := extensions.(map[string]interface{})
					assert.NotContains(t, extMap, "stack", "stack trace should be filtered out")
					assert.NotContains(t, extMap, "function_name", "function name should be filtered out")
					assert.NotContains(t, extMap, "panic_value", "panic value should be filtered out")

					// Safe extensions should remain
					if tt.extensions["safe_field"] != "" {
						assert.Contains(t, extMap, "safe_field")
					}
				}

				// Should use production message
				message := result["message"].(string)
				if tt.productionMessage != "" {
					assert.Equal(t, tt.productionMessage, message, "should use production message")
				}
			} else {
				// In development mode, all information should be preserved
				if (tt.extensions != nil && len(tt.extensions) > 0) || (tt.sensitiveExtensions != nil && len(tt.sensitiveExtensions) > 0) {
					assert.Contains(t, result, "extensions", "extensions should be present in dev mode")
					extMap := result["extensions"].(map[string]interface{})
					// Both regular and sensitive extensions should be present in dev mode
					for key := range tt.extensions {
						assert.Contains(t, extMap, key, fmt.Sprintf("extension %s should be present in dev mode", key))
					}
					for key := range tt.sensitiveExtensions {
						assert.Contains(t, extMap, key, fmt.Sprintf("sensitive extension %s should be present in dev mode", key))
					}
				}

				// Inner error details should be appended to message
				if tt.hasInnerError {
					message := result["message"].(string)
					assert.Contains(t, message, tt.innerError, "inner error should be exposed in dev mode")
				}
			}
		})
	}
}

func TestContainsSensitiveInfo(t *testing.T) {
	tests := []struct {
		message   string
		sensitive bool
	}{
		{"Invalid query syntax", false},
		{"Field not found", false},
		{"function testFunc panicked: runtime error", true},
		{"panic: nil pointer dereference", true},
		{"runtime error: index out of range", true},
		{"goroutine 1 [running]", true},
		{"main.go:123 error occurred", true},
		{"reflect.Value.Call panic", true},
		{"Query validation failed", false},
		{"Authentication required", false},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			result := containsSensitiveInfo(tt.message)
			assert.Equal(t, tt.sensitive, result, "Sensitivity detection should match expected result")
		})
	}
}

func TestNewGraphErrorProduction(t *testing.T) {
	pos := lexer.Position{Line: 5, Column: 10, Offset: 50}

	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "safe message preserved",
			message:  "Invalid query syntax",
			expected: "Invalid query syntax",
		},
		{
			name:     "sensitive message sanitized",
			message:  "function testFunc panicked: runtime error",
			expected: "An error occurred while processing the request",
		},
		{
			name:     "empty message gets generic",
			message:  "",
			expected: "An error occurred while processing the request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gErr := NewGraphErrorProduction(tt.message, pos, "test", "path")
			assert.Equal(t, tt.expected, gErr.Message)
			assert.Equal(t, []string{"test", "path"}, gErr.Path)
			assert.Equal(t, 5, gErr.Locations[0].Line)
			assert.Equal(t, 10, gErr.Locations[0].Column)
		})
	}
}

func TestFormatErrorWithMode(t *testing.T) {
	// Create test errors
	basicErr := fmt.Errorf("basic error")
	graphErr := GraphError{
		Message:           "function test panicked",
		ProductionMessage: "Internal server error",
		InnerError:        fmt.Errorf("nil pointer dereference"),
		Extensions: map[string]string{
			"line": "5",
		},
		SensitiveExtensions: map[string]string{
			"stack":         "stack trace data",
			"function_name": "testFunction",
		},
	}

	errors := []error{basicErr, graphErr}

	t.Run("development mode", func(t *testing.T) {
		result := formatErrorWithMode(false, errors...)

		var response map[string]interface{}
		err := json.Unmarshal([]byte(result), &response)
		require.NoError(t, err)

		errorsArray := response["errors"].([]interface{})
		assert.Len(t, errorsArray, 2)

		// Check that sensitive information is present in dev mode
		errorStr := result
		assert.Contains(t, errorStr, "nil pointer dereference")
		assert.Contains(t, errorStr, "stack")
		assert.Contains(t, errorStr, "function_name")
	})

	t.Run("production mode", func(t *testing.T) {
		result := formatErrorWithMode(true, errors...)

		var response map[string]interface{}
		err := json.Unmarshal([]byte(result), &response)
		require.NoError(t, err)

		errorsArray := response["errors"].([]interface{})
		assert.Len(t, errorsArray, 2)

		// Check that sensitive information is sanitized in production mode
		errorStr := result
		assert.NotContains(t, errorStr, "nil pointer dereference")
		assert.NotContains(t, errorStr, "stack trace data")
		assert.NotContains(t, errorStr, "testFunction")
		assert.Contains(t, errorStr, "Internal server error")
	})
}

func TestGraphyProductionModeIntegration(t *testing.T) {
	// Test function that panics
	panicFunc := func(ctx context.Context) (string, error) {
		panic("test panic message")
	}

	t.Run("development mode shows panic details", func(t *testing.T) {
		g := &Graphy{ProductionMode: false}
		g.RegisterQuery(context.Background(), "testPanic", panicFunc)

		result, err := g.ProcessRequest(context.Background(), `{testPanic}`, "{}")

		// Should have an error
		assert.Error(t, err)

		// Result should contain detailed panic information
		assert.Contains(t, result, "testPanic panicked")
		assert.Contains(t, result, "test panic message")

		// Parse JSON to check extensions
		var response map[string]interface{}
		jsonErr := json.Unmarshal([]byte(result), &response)
		require.NoError(t, jsonErr)

		errorsArray := response["errors"].([]interface{})
		require.Len(t, errorsArray, 1)

		errorObj := errorsArray[0].(map[string]interface{})
		if extensions, exists := errorObj["extensions"]; exists {
			extMap := extensions.(map[string]interface{})
			assert.Contains(t, extMap, "stack")
			assert.Contains(t, extMap, "function_name")
			assert.Contains(t, extMap, "panic_value")
		}
	})

	t.Run("production mode sanitizes panic details", func(t *testing.T) {
		g := &Graphy{ProductionMode: true}
		g.RegisterQuery(context.Background(), "testPanic", panicFunc)

		result, err := g.ProcessRequest(context.Background(), `{testPanic}`, "{}")

		// Should have an error
		assert.Error(t, err)

		// Result should contain only generic error information
		assert.NotContains(t, result, "testPanic panicked")
		assert.NotContains(t, result, "test panic message")
		assert.Contains(t, result, "Internal server error")

		// Parse JSON to check extensions are sanitized
		var response map[string]interface{}
		jsonErr := json.Unmarshal([]byte(result), &response)
		require.NoError(t, jsonErr)

		errorsArray := response["errors"].([]interface{})
		require.Len(t, errorsArray, 1)

		errorObj := errorsArray[0].(map[string]interface{})
		if extensions, exists := errorObj["extensions"]; exists {
			extMap := extensions.(map[string]interface{})
			assert.NotContains(t, extMap, "stack")
			assert.NotContains(t, extMap, "function_name")
			assert.NotContains(t, extMap, "panic_value")
		}
	})
}

func TestProductionModeFormatError(t *testing.T) {
	// Test backward compatibility - formatError should default to development mode
	graphErr := GraphError{
		Message:    "function test panicked",
		InnerError: fmt.Errorf("nil pointer dereference"),
		Extensions: map[string]string{
			"stack": "stack trace data",
		},
	}

	result := formatError(graphErr)

	// Should contain development mode details (backward compatibility)
	assert.Contains(t, result, "nil pointer dereference")
	assert.Contains(t, result, "stack")
}

func TestEmptyExtensionsHandling(t *testing.T) {
	// Test that empty extensions map is properly handled in production mode
	gErr := GraphError{
		Message:           "test error",
		ProductionMessage: "Generic error",
		SensitiveExtensions: map[string]string{
			"stack":         "sensitive data",
			"function_name": "testFunc",
		},
		// No regular extensions - only sensitive ones
	}

	jsonBytes, err := gErr.MarshalJSONProduction()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(jsonBytes, &result)
	require.NoError(t, err)

	// Extensions should be nil/absent when all extensions are sensitive and filtered out
	_, hasExtensions := result["extensions"]
	assert.False(t, hasExtensions, "Extensions should be removed when all keys are sensitive")
}

func TestErrorHandlerReceivesPanicDetails(t *testing.T) {
	// Test that error handler receives full panic details in both production and development modes
	panicFunc := func(ctx context.Context) (string, error) {
		panic("test panic for error handler")
	}

	tests := []struct {
		name           string
		productionMode bool
	}{
		{"development mode", false},
		{"production mode", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedError error
			var capturedDetails map[string]interface{}
			var capturedCategory ErrorCategory

			// Custom error handler to capture what gets logged
			errorHandler := ErrorHandlerFunc(func(ctx context.Context, category ErrorCategory, err error, details map[string]interface{}) {
				capturedError = err
				capturedDetails = details
				capturedCategory = category
			})

			g := &Graphy{
				ProductionMode: tt.productionMode,
			}
			g.SetErrorHandler(errorHandler)
			g.RegisterQuery(context.Background(), "testPanic", panicFunc)

			// Execute the request that will panic
			result, err := g.ProcessRequest(context.Background(), `{testPanic}`, "{}")

			// Should have an error in both modes
			assert.Error(t, err)

			// Verify error handler was called with full details
			assert.NotNil(t, capturedError, "Error handler should receive the panic error")
			assert.Equal(t, ErrorCategoryExecution, capturedCategory, "Should be categorized as execution error")

			// Verify error handler gets full panic details regardless of production mode
			assert.Contains(t, capturedDetails, "function_name", "Error handler should receive function name")
			assert.Contains(t, capturedDetails, "panic_value", "Error handler should receive panic value")
			assert.Contains(t, capturedDetails, "stack", "Error handler should receive stack trace")
			assert.Equal(t, "testPanic", capturedDetails["function_name"], "Function name should be correct")
			assert.Equal(t, "test panic for error handler", capturedDetails["panic_value"], "Panic value should be correct")
			assert.Contains(t, capturedDetails["stack"], "goroutine", "Stack trace should contain goroutine info")

			// Verify the captured error contains full details
			var gErr GraphError
			errors.As(capturedError, &gErr)
			assert.Contains(t, gErr.Message, "testPanic panicked", "Error message should contain function name")
			assert.Contains(t, gErr.Message, "test panic for error handler", "Error message should contain panic details")

			// Verify client response differs based on mode
			if tt.productionMode {
				// Production mode: client should get sanitized response
				assert.NotContains(t, result, "testPanic panicked", "Production response should not contain panic message")
				assert.NotContains(t, result, "test panic for error handler", "Production response should not contain panic details")
				assert.Contains(t, result, "Internal server error", "Production response should contain generic message")
				assert.Contains(t, result, `"path":["testPanic"]`, "Production response should still contain path for debugging")
			} else {
				// Development mode: client should get detailed response
				assert.Contains(t, result, "testPanic panicked", "Development response should contain function name")
				assert.Contains(t, result, "test panic for error handler", "Development response should contain panic details")
			}
		})
	}
}
