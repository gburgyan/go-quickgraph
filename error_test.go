package quickgraph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/alecthomas/participle/v2/lexer"
	"github.com/stretchr/testify/assert"
)

func Test_getLineAndColumnFromOffset(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		offset int
		line   int
		column int
	}{
		{
			name:   "Empty String",
			input:  "",
			offset: 0,
			line:   1,
			column: 1,
		},
		{
			name:   "Single Line Without Offset",
			input:  "Hello, World!",
			offset: 0,
			line:   1,
			column: 1,
		},
		{
			name:   "Single Line With Offset",
			input:  "Hello, World!",
			offset: 7,
			line:   1,
			column: 8,
		},
		{
			name:   "Multiple Lines Without Offset",
			input:  "Hello,\nWorld!",
			offset: 0,
			line:   1,
			column: 1,
		},
		{
			name:   "Multiple Lines With Offset On First Line",
			input:  "Hello,\nWorld!",
			offset: 5,
			line:   1,
			column: 6,
		},
		{
			name:   "Multiple Lines With Offset On NewLine",
			input:  "Hello,\nWorld!",
			offset: 7,
			line:   2,
			column: 0,
		},
		{
			name:   "Multiple Lines With Offset On Second Line",
			input:  "Hello,\nWorld!",
			offset: 8,
			line:   2,
			column: 1,
		},
		{
			name:   "Offset Larger Than Input Length",
			input:  "Hello, World!",
			offset: 100,
			line:   1,
			column: 14, // This will only go until the length of the string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, column := getLineAndColumnFromOffset(tt.input, tt.offset)
			assert.Equal(t, tt.line, line)
			assert.Equal(t, tt.column, column)
		})
	}
}

func Test_UnknownCommand(t *testing.T) {
	input := `
{
  hero {
    name
  }
}`

	ctx := context.Background()
	g := Graphy{}

	resultAny, err := g.ProcessRequest(ctx, input, "")

	assert.Equal(t, `{"errors":[{"message":"unknown command(s) in request: hero","locations":[{"line":3,"column":3}]}]}`, resultAny)
	var uce UnknownCommandError
	errors.As(err, &uce)
	assert.Contains(t, uce.Commands, "hero")
}

func Test_JsonError_NoError(t *testing.T) {
	err := transformJsonError("", nil)
	assert.NoError(t, err)
}

func Test_JsonError_UnmarshalTypeError(t *testing.T) {
	type testStruct struct {
		Val int
	}
	var target testStruct
	input := "{\"Val\"\n:\"42\"}"
	jsonErr := json.Unmarshal([]byte(input), &target)

	err := transformJsonError(input, jsonErr)
	var ge GraphError
	errors.As(err, &ge)

	assert.Equal(t, 2, ge.Locations[0].Line)
	assert.Equal(t, 5, ge.Locations[0].Column)
}

func Test_JsonError_SyntaxError(t *testing.T) {
	type testStruct struct {
		Val int
	}
	var target testStruct
	input := "{\"Val\"\n\"42\"}"
	jsonErr := json.Unmarshal([]byte(input), &target)

	err := transformJsonError(input, jsonErr)
	var ge GraphError
	errors.As(err, &ge)

	assert.Equal(t, 2, ge.Locations[0].Line)
	assert.Equal(t, 1, ge.Locations[0].Column)
}

func Test_JsonError_RandomError(t *testing.T) {
	rErr := fmt.Errorf("random error")
	err := transformJsonError("", rErr)
	var ge GraphError
	errors.As(err, &ge)
	assert.Equal(t, rErr, ge.InnerError)
	assert.Equal(t, rErr, ge.Unwrap())
}

func Test_formatError(t *testing.T) {
	err1 := fmt.Errorf("random error")
	err2 := GraphError{
		Message:   "graph error",
		Locations: []ErrorLocation{{Line: 1, Column: 1}},
	}
	msg := formatError(err1, err2)
	assert.Equal(t, `{"errors":[{"message":"random error: random error"},{"message":"graph error","locations":[{"line":1,"column":1}]}]}`, msg)
}

func TestErrorSanitizer_ContainsSensitiveInfo(t *testing.T) {
	tests := []struct {
		name            string
		message         string
		expectSensitive bool
		description     string
	}{
		// Test cases that should be flagged as sensitive
		{
			name:            "panic_error",
			message:         "panic: runtime error: invalid memory address",
			expectSensitive: true,
			description:     "Should detect panic messages",
		},
		{
			name:            "runtime_error",
			message:         "runtime error: index out of range [5] with length 3",
			expectSensitive: true,
			description:     "Should detect runtime errors",
		},
		{
			name:            "stack_trace",
			message:         "goroutine 1 [running]: main.go:123",
			expectSensitive: true,
			description:     "Should detect stack traces",
		},
		{
			name:            "memory_address",
			message:         "pointer 0x40e020 is invalid",
			expectSensitive: true,
			description:     "Should detect memory addresses",
		},
		{
			name:            "reflection_call",
			message:         "reflect.ValueOf failed on type string",
			expectSensitive: true,
			description:     "Should detect reflection internals",
		},
		{
			name:            "file_path_unix",
			message:         "failed to open /usr/local/src/app/main.go",
			expectSensitive: true,
			description:     "Should detect Unix file paths",
		},
		{
			name:            "file_path_windows",
			message:         "error in C:\\\\Program Files\\\\app\\\\main.go",
			expectSensitive: true,
			description:     "Should detect Windows file paths",
		},
		{
			name:            "function_signature",
			message:         "func(string, int) error returned nil",
			expectSensitive: true,
			description:     "Should detect function signatures",
		},
		{
			name:            "method_call",
			message:         "pkg.MethodName() failed with error",
			expectSensitive: true,
			description:     "Should detect method calls",
		},
		{
			name:            "nil_pointer",
			message:         "nil pointer dereference in handler",
			expectSensitive: true,
			description:     "Should detect nil pointer errors",
		},
		{
			name:            "slice_bounds",
			message:         "slice bounds out of range [10:5]",
			expectSensitive: true,
			description:     "Should detect slice bounds errors",
		},
		{
			name:            "interface_type",
			message:         "cannot convert interface {} to string",
			expectSensitive: true,
			description:     "Should detect interface{} references",
		},

		// Test cases that should NOT be flagged as sensitive (allowed patterns)
		{
			name:            "validation_error",
			message:         "validation failed: field is required",
			expectSensitive: false,
			description:     "Validation errors should be safe",
		},
		{
			name:            "invalid_field",
			message:         "invalid field name provided",
			expectSensitive: false,
			description:     "Field validation should be safe",
		},
		{
			name:            "missing_parameter",
			message:         "missing required parameter: userID",
			expectSensitive: false,
			description:     "Parameter errors should be safe",
		},
		{
			name:            "graphql_type_error",
			message:         "unknown type User in query",
			expectSensitive: false,
			description:     "GraphQL type errors should be safe",
		},
		{
			name:            "query_syntax_error",
			message:         "syntax error in query at line 5",
			expectSensitive: false,
			description:     "Query syntax errors should be safe",
		},
		{
			name:            "mutation_error",
			message:         "mutation createUser failed: invalid input",
			expectSensitive: false,
			description:     "Mutation errors should be safe",
		},
		{
			name:            "subscription_error",
			message:         "subscription not found: userUpdates",
			expectSensitive: false,
			description:     "Subscription errors should be safe",
		},
		{
			name:            "variable_error",
			message:         "variable $userId not defined",
			expectSensitive: false,
			description:     "Variable errors should be safe",
		},
		{
			name:            "argument_error",
			message:         "argument 'name' is required",
			expectSensitive: false,
			description:     "Argument errors should be safe",
		},
		{
			name:            "expected_type_error",
			message:         "expected String, got Int",
			expectSensitive: false,
			description:     "Type mismatch errors should be safe",
		},
		{
			name:            "empty_message",
			message:         "",
			expectSensitive: false,
			description:     "Empty messages should be safe",
		},
		{
			name:            "simple_message",
			message:         "user not found",
			expectSensitive: false,
			description:     "Simple user messages should be safe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsSensitiveInfo(tt.message)
			assert.Equal(t, tt.expectSensitive, result,
				"Test %s failed: %s. Message: %q", tt.name, tt.description, tt.message)
		})
	}
}

func TestErrorSanitizer_CustomSanitizer(t *testing.T) {
	// Create a custom sanitizer for testing
	customSanitizer := &ErrorSanitizer{
		RegexPatterns: []*regexp.Regexp{
			regexp.MustCompile(`\bsecret\b`),
			regexp.MustCompile(`\bpassword\b`),
		},
		StringPatterns: []string{
			"confidential",
		},
		CustomSanitizer: func(message string) bool {
			return message == "custom sensitive message"
		},
	}

	tests := []struct {
		name            string
		message         string
		expectSensitive bool
	}{
		{
			name:            "custom_regex_pattern",
			message:         "the secret key is invalid",
			expectSensitive: true,
		},
		{
			name:            "custom_string_pattern",
			message:         "this is confidential information",
			expectSensitive: true,
		},
		{
			name:            "custom_function",
			message:         "custom sensitive message",
			expectSensitive: true,
		},
		{
			name:            "safe_message",
			message:         "user not found",
			expectSensitive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := customSanitizer.ContainsSensitiveInfo(tt.message)
			assert.Equal(t, tt.expectSensitive, result,
				"Custom sanitizer test %s failed. Message: %q", tt.name, tt.message)
		})
	}
}

func TestNewGraphErrorProductionSanitization(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		expected    string
		description string
	}{
		{
			name:        "sensitive_panic_message",
			message:     "panic: runtime error in main.go:123",
			expected:    "An error occurred while processing the request",
			description: "Sensitive messages should be completely sanitized",
		},
		{
			name:        "safe_validation_message",
			message:     "validation failed: field is required",
			expected:    "validation failed: field is required",
			description: "Safe validation messages should be preserved",
		},
		{
			name:        "partially_sensitive_message",
			message:     "error at main.go:123 in validation",
			expected:    "An error occurred while processing the request",
			description: "Messages with any sensitive content should be replaced with generic message",
		},
		{
			name:        "empty_message",
			message:     "",
			expected:    "An error occurred while processing the request",
			description: "Empty messages should get generic message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewGraphErrorProduction(tt.message, lexer.Position{})
			assert.Equal(t, tt.expected, err.Message,
				"Production error test %s failed: %s", tt.name, tt.description)
		})
	}
}

func TestGraphError_MarshalJSONProduction(t *testing.T) {
	tests := []struct {
		name              string
		error             GraphError
		expectContains    []string
		expectNotContains []string
		description       string
	}{
		{
			name: "production_mode_sensitive_extensions_filtered",
			error: GraphError{
				Message: "An error occurred",
				Extensions: map[string]string{
					"safe_key": "safe_value",
				},
				SensitiveExtensions: map[string]string{
					"stack_trace":   "goroutine 1 [running]",
					"function_name": "pkg.ProcessRequest",
					"panic_value":   "runtime error",
				},
			},
			expectContains: []string{
				"\"safe_key\":\"safe_value\"",
				"\"message\":\"An error occurred\"",
			},
			expectNotContains: []string{
				"stack_trace",
				"function_name",
				"panic_value",
				"goroutine",
				"ProcessRequest",
			},
			description: "Production mode should filter sensitive extensions",
		},
		{
			name: "production_message_used",
			error: GraphError{
				Message:           "panic: runtime error in main.go:123",
				ProductionMessage: "Internal server error",
			},
			expectContains: []string{
				"\"message\":\"Internal server error\"",
			},
			expectNotContains: []string{
				"panic",
				"runtime error",
				"main.go",
			},
			description: "Production message should override development message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBytes, err := tt.error.MarshalJSONProduction()
			assert.NoError(t, err, "JSON marshaling should not fail")

			jsonStr := string(jsonBytes)

			for _, expected := range tt.expectContains {
				assert.Contains(t, jsonStr, expected,
					"Production JSON should contain %q. Full JSON: %s", expected, jsonStr)
			}

			for _, notExpected := range tt.expectNotContains {
				assert.NotContains(t, jsonStr, notExpected,
					"Production JSON should not contain %q. Full JSON: %s", notExpected, jsonStr)
			}
		})
	}
}

func TestSetErrorSanitizer(t *testing.T) {
	// Save original sanitizer
	originalSanitizer := defaultErrorSanitizer
	defer func() {
		// Restore original sanitizer
		SetErrorSanitizer(originalSanitizer)
	}()

	// Create a test sanitizer that considers "test" as sensitive
	testSanitizer := &ErrorSanitizer{
		StringPatterns: []string{"test"},
	}

	// Test message before custom sanitizer
	assert.False(t, containsSensitiveInfo("this is a test message"))

	// Set custom sanitizer
	SetErrorSanitizer(testSanitizer)

	// Test message after custom sanitizer
	assert.True(t, containsSensitiveInfo("this is a test message"))

	// Test that non-matching messages are still safe
	assert.False(t, containsSensitiveInfo("this is a safe message"))
}

func TestErrorSanitizer_EdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		message         string
		expectSensitive bool
		description     string
	}{
		{
			name:            "case_insensitive_panic",
			message:         "PANIC: something went wrong",
			expectSensitive: true,
			description:     "Should detect case-insensitive panic",
		},
		{
			name:            "case_insensitive_runtime",
			message:         "Runtime Error occurred",
			expectSensitive: true,
			description:     "Should detect case-insensitive runtime error",
		},
		{
			name:            "partial_word_match",
			message:         "this is not a panicking situation",
			expectSensitive: false,
			description:     "Word boundary patterns should not match partial words",
		},
		{
			name:            "multiple_addresses",
			message:         "addresses 0x40e020 and 0x50f130 are invalid",
			expectSensitive: true,
			description:     "Should detect multiple memory addresses",
		},
		{
			name:            "mixed_sensitive_and_safe",
			message:         "validation failed but panic occurred at main.go:123",
			expectSensitive: true,
			description:     "Messages with both safe and sensitive parts should be flagged",
		},
		{
			name:            "unicode_in_message",
			message:         "validation failed: ñame is required",
			expectSensitive: false,
			description:     "Unicode in safe messages should work correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsSensitiveInfo(tt.message)
			assert.Equal(t, tt.expectSensitive, result,
				"Edge case test %s failed: %s. Message: %q", tt.name, tt.description, tt.message)
		})
	}
}

func TestErrorSanitizer_Performance(t *testing.T) {
	// Test that the regex compilation is done once and reused
	sanitizer := &ErrorSanitizer{
		RegexPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\bpanic\b`),
			regexp.MustCompile(`(?i)\bruntime error\b`),
		},
	}

	// Run many iterations to ensure no performance issues
	testMessage := "this is a panic message with runtime error"
	for i := 0; i < 1000; i++ {
		result := sanitizer.ContainsSensitiveInfo(testMessage)
		assert.True(t, result, "Performance test iteration %d failed", i)
	}
}

// TestErrorSanitizer_RealWorldScenarios tests with actual error messages that might occur
func TestErrorSanitizer_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name            string
		message         string
		expectSensitive bool
		description     string
	}{
		{
			name:            "database_connection_error",
			message:         "failed to connect to database: connection refused",
			expectSensitive: false,
			description:     "Database connection errors should generally be safe",
		},
		{
			name:            "json_unmarshal_error",
			message:         "json: cannot unmarshal string into Go struct field User.Age of type int",
			expectSensitive: false,
			description:     "JSON unmarshaling errors should be safe when they don't reveal internals",
		},
		{
			name:            "reflection_error_internal",
			message:         "reflect: call of reflect.Value.Interface on zero Value",
			expectSensitive: true,
			description:     "Reflection errors should be considered sensitive",
		},
		{
			name:            "http_timeout",
			message:         "context deadline exceeded",
			expectSensitive: false,
			description:     "HTTP timeout errors should be safe",
		},
		{
			name:            "graphql_validation",
			message:         `Cannot query field "nonexistent" on type "User"`,
			expectSensitive: false,
			description:     "GraphQL validation errors should be safe",
		},
		{
			name:            "stack_overflow_like",
			message:         "runtime: goroutine stack exceeds 1000000000-byte limit",
			expectSensitive: true,
			description:     "Runtime stack errors should be sensitive",
		},
		{
			name:            "custom_business_error",
			message:         "user account is suspended",
			expectSensitive: false,
			description:     "Business logic errors should be safe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsSensitiveInfo(tt.message)
			assert.Equal(t, tt.expectSensitive, result,
				"Real world scenario %s failed: %s. Message: %q", tt.name, tt.description, tt.message)
		})
	}
}
