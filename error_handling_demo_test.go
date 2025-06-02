package quickgraph

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestErrorHandlingApproach demonstrates the new error handling approach
func TestErrorHandlingApproach(t *testing.T) {
	panicFunc := func(ctx context.Context) (string, error) {
		panic("sensitive internal error")
	}

	var errorHandlerCalled bool
	var loggedError error
	var loggedDetails map[string]interface{}

	// Create error handler to capture what gets logged
	errorHandler := ErrorHandlerFunc(func(ctx context.Context, category ErrorCategory, err error, details map[string]interface{}) {
		errorHandlerCalled = true
		loggedError = err
		loggedDetails = details
	})

	t.Run("single error logging in production mode", func(t *testing.T) {
		// Reset
		errorHandlerCalled = false
		loggedError = nil
		loggedDetails = nil

		g := &Graphy{ProductionMode: true}
		g.SetErrorHandler(errorHandler)
		g.RegisterQuery(context.Background(), "testPanic", panicFunc)

		result, err := g.ProcessRequest(context.Background(), `{testPanic}`, "{}")

		// Should have an error
		assert.Error(t, err)

		// Error handler should be called exactly once (at the top level)
		assert.True(t, errorHandlerCalled, "Error handler should be called")
		assert.NotNil(t, loggedError, "Error should be logged")

		// Logged error should contain full development details
		var gErr GraphError
		if errors.As(loggedError, &gErr) {
			assert.Contains(t, gErr.Message, "testPanic panicked", "Logged error should contain function name")
			assert.Contains(t, gErr.Message, "sensitive internal error", "Logged error should contain panic details")
		}

		// Logged details should contain sensitive information for debugging
		assert.Contains(t, loggedDetails, "stack", "Should log stack trace")
		assert.Contains(t, loggedDetails, "function_name", "Should log function name")
		assert.Contains(t, loggedDetails, "panic_value", "Should log panic value")

		// Client response should be sanitized
		assert.NotContains(t, result, "testPanic panicked", "Client response should not contain panic message")
		assert.NotContains(t, result, "sensitive internal error", "Client response should not contain panic details")
		assert.Contains(t, result, "Internal server error", "Client response should contain generic message")
		assert.Contains(t, result, `"path":["testPanic"]`, "Client response should still contain path for debugging")
	})

	t.Run("single error logging in development mode", func(t *testing.T) {
		// Reset
		errorHandlerCalled = false
		loggedError = nil
		loggedDetails = nil

		g := &Graphy{ProductionMode: false}
		g.SetErrorHandler(errorHandler)
		g.RegisterQuery(context.Background(), "testPanic", panicFunc)

		result, err := g.ProcessRequest(context.Background(), `{testPanic}`, "{}")

		// Should have an error
		assert.Error(t, err)

		// Error handler should be called exactly once (at the top level)
		assert.True(t, errorHandlerCalled, "Error handler should be called")
		assert.NotNil(t, loggedError, "Error should be logged")

		// Logged error should contain full development details (same as production logging)
		var gErr GraphError
		if errors.As(loggedError, &gErr) {
			assert.Contains(t, gErr.Message, "testPanic panicked", "Logged error should contain function name")
			assert.Contains(t, gErr.Message, "sensitive internal error", "Logged error should contain panic details")
		}

		// Logged details should contain sensitive information for debugging
		assert.Contains(t, loggedDetails, "stack", "Should log stack trace")
		assert.Contains(t, loggedDetails, "function_name", "Should log function name")
		assert.Contains(t, loggedDetails, "panic_value", "Should log panic value")

		// Client response should contain full details in development mode
		assert.Contains(t, result, "testPanic panicked", "Client response should contain function name")
		assert.Contains(t, result, "sensitive internal error", "Client response should contain panic details")
		assert.Contains(t, result, "stack", "Client response should contain stack trace")
	})
}
