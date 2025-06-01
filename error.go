package quickgraph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alecthomas/participle/v2/lexer"
	"strings"
)

// ErrorCategory represents different types of errors that can occur
type ErrorCategory string

const (
	// ErrorCategoryValidation - GraphQL validation errors (invalid queries, missing variables, etc.)
	ErrorCategoryValidation ErrorCategory = "validation"
	// ErrorCategoryExecution - Runtime errors during GraphQL execution
	ErrorCategoryExecution ErrorCategory = "execution"
	// ErrorCategoryWebSocket - WebSocket connection and protocol errors
	ErrorCategoryWebSocket ErrorCategory = "websocket"
	// ErrorCategoryHTTP - HTTP protocol errors (malformed requests, etc.)
	ErrorCategoryHTTP ErrorCategory = "http"
	// ErrorCategoryInternal - Internal library errors
	ErrorCategoryInternal ErrorCategory = "internal"
)

// ErrorHandler is an interface for handling different types of errors
// Applications can implement this to customize error logging and handling
type ErrorHandler interface {
	// HandleError is called when an error occurs during GraphQL processing
	// ctx: the request context (may be nil for some errors)
	// category: the type of error that occurred
	// err: the error (will be a GraphError if possible)
	// details: additional context about the error (e.g., "subscription_id", "request_method")
	HandleError(ctx context.Context, category ErrorCategory, err error, details map[string]interface{})
}

// ErrorHandlerFunc is a function adapter for ErrorHandler
type ErrorHandlerFunc func(ctx context.Context, category ErrorCategory, err error, details map[string]interface{})

func (f ErrorHandlerFunc) HandleError(ctx context.Context, category ErrorCategory, err error, details map[string]interface{}) {
	f(ctx, category, err, details)
}

// GraphError represents an error that occurs within a graph structure.
// It provides a structured way to express errors with added context,
// such as their location in the source and any associated path.
//
// Fields:
// - Message: The main error message.
// - Locations: A slice of ErrorLocation structs that detail where in the source the error occurred.
// - Path: Represents the path in the graph where the error occurred.
// - Extensions: A map containing additional error information not part of the standard fields.
// - InnerError: An underlying error that might have caused this GraphError. It is not serialized to JSON.
// - ProductionMessage: Sanitized message for production use (if different from Message).
// - SensitiveExtensions: Extensions that should be filtered in production mode.
type GraphError struct {
	Message             string            `json:"message"`
	Locations           []ErrorLocation   `json:"locations,omitempty"`
	Path                []string          `json:"path,omitempty"`
	Extensions          map[string]string `json:"extensions,omitempty"`
	InnerError          error             `json:"-"`
	ProductionMessage   string            `json:"-"` // Sanitized message for production
	SensitiveExtensions map[string]string `json:"-"` // Extensions to filter in production
}

// ErrorLocation provides details about where in the source a particular error occurred.
//
// Fields:
// - Line: The line number of the error.
// - Column: The column number of the error.
type ErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// UnknownCommandError is a specific type of GraphError that
// occurs when an unknown command is encountered.
// It embeds the GraphError and adds a Commands field which contains
// a list of commands that were unrecognized.
type UnknownCommandError struct {
	GraphError
	Commands []string
}

// Implement the error interface
func (e GraphError) Error() string {
	// Return the message as well as the path (if it exists) as well as the error locations (if they exist).
	s := strings.Builder{}
	s.WriteString(e.Message)
	if len(e.Path) > 0 {
		s.WriteString(" (path: ")
		s.WriteString(strings.Join(e.Path, "/"))
		s.WriteString(")")
	}
	if len(e.Locations) > 0 {
		s.WriteString(" [")
		s.WriteString(strings.Join(toStringSlice(e.Locations), ", "))
		s.WriteString("]")
	}

	if e.InnerError != nil {
		s.WriteString(": ")
		s.WriteString(e.InnerError.Error())
	}

	return s.String()
}

// NewGraphError creates a new GraphError with the provided message, position, and path. It
// uses the position to create an ErrorLocation structure and adds it to the Locations field.
func NewGraphError(message string, pos lexer.Position, paths ...string) GraphError {
	var gErr GraphError
	if pos.Offset > 0 {
		loc := lexerPositionError(pos)
		gErr.Locations = append(gErr.Locations, loc)
	}
	gErr.Message = message
	gErr.Path = paths
	return gErr
}

// NewGraphErrorProduction creates a sanitized GraphError for production use
func NewGraphErrorProduction(message string, pos lexer.Position, paths ...string) GraphError {
	// In production mode, provide a generic error message to avoid information disclosure
	genericMessage := "An error occurred while processing the request"

	// For validation errors, keep the original message as it's usually safe
	if message != "" && !containsSensitiveInfo(message) {
		genericMessage = message
	}

	var gErr GraphError
	if pos.Offset > 0 {
		loc := lexerPositionError(pos)
		gErr.Locations = append(gErr.Locations, loc)
	}
	gErr.Message = genericMessage
	gErr.Path = paths
	return gErr
}

// NewGraphErrorWithProduction creates a GraphError with both development and production messages
func NewGraphErrorWithProduction(devMessage, prodMessage string, pos lexer.Position, paths ...string) GraphError {
	var gErr GraphError
	if pos.Offset > 0 {
		loc := lexerPositionError(pos)
		gErr.Locations = append(gErr.Locations, loc)
	}
	gErr.Message = devMessage
	gErr.ProductionMessage = prodMessage
	gErr.Path = paths
	return gErr
}

// NewPanicError creates a GraphError for function panics with both dev and production messages
func NewPanicError(functionName string, panicValue interface{}, pos lexer.Position, paths ...string) GraphError {
	devMessage := fmt.Sprintf("function %s panicked: %v", functionName, panicValue)
	prodMessage := "Internal server error"

	gErr := NewGraphErrorWithProduction(devMessage, prodMessage, pos, paths...)
	gErr.InnerError = fmt.Errorf("panic: %v", panicValue)

	// Initialize sensitive extensions
	if gErr.SensitiveExtensions == nil {
		gErr.SensitiveExtensions = make(map[string]string)
	}

	return gErr
}

// AddSensitiveExtension adds a key-value pair to the SensitiveExtensions field
// These extensions will be filtered out in production mode
func (e *GraphError) AddSensitiveExtension(key string, value string) {
	if e.SensitiveExtensions == nil {
		e.SensitiveExtensions = make(map[string]string)
	}
	e.SensitiveExtensions[key] = value
}

// containsSensitiveInfo checks if a message contains potentially sensitive information
func containsSensitiveInfo(message string) bool {
	// Check for common patterns that might leak sensitive information
	sensitivePatterns := []string{
		"panic:", "runtime error:", "stack trace:", "function", "reflect.",
		"goroutine", "main.go", ".go:", "nil pointer", "index out of range",
	}

	messageLower := strings.ToLower(message)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(messageLower, pattern) {
			return true
		}
	}
	return false
}

// lexerPositionError takes a lexer.Position and returns an ErrorLocation that is the equivalent.
func lexerPositionError(pos lexer.Position) ErrorLocation {
	return ErrorLocation{
		Line:   pos.Line,
		Column: pos.Column,
	}
}

// AugmentGraphError wraps the provided error with additional context, creating or augmenting
// a GraphError structure. This function serves to enrich errors with Graph-specific details
// such as file position, path, and a custom message.
//
// The function primarily handles two cases:
//  1. If the passed error is already a GraphError, it augments the existing GraphError with the
//     provided context without losing any existing data.
//  2. If the passed error is not a GraphError, it wraps it within a new GraphError, setting
//     the provided context.
//
// Parameters:
//   - err: The error to be wrapped or augmented. It can be a regular error or a GraphError.
//   - message: A custom message to be set on the GraphError. If a GraphError is passed and it
//     already contains a message, this parameter is ignored.
//   - pos: A lexer.Position structure indicating where in the source the error occurred. If a
//     GraphError is passed and it already contains location data, this parameter is ignored.
//   - paths: A variadic slice of strings indicating the path in the graph where the error occurred.
//     These are prepended to any existing paths in a GraphError.
//
// Returns:
// - A GraphError containing the augmented or wrapped error details.
func AugmentGraphError(err error, message string, pos lexer.Position, paths ...string) error {
	var gErr GraphError

	// We should never have a regular error wrapping a GraphError. If that ever happens
	// the extra error information is lost as we're only using the wrapped GraphError.
	ok := errors.As(err, &gErr)
	if !ok {
		gErr = GraphError{
			Message:    message,
			InnerError: err,
		}
	}

	// If the message isn't set, set it.
	if gErr.Message == "" {
		gErr.Message = message
	}

	// Prepend the path to the existing path.
	if len(paths) > 0 {
		gErr.Path = append(paths, gErr.Path...)
	}

	if pos.Offset > 0 && len(gErr.Locations) == 0 {
		loc := ErrorLocation{
			Line:   pos.Line,
			Column: pos.Column,
		}
		gErr.Locations = append(gErr.Locations, loc)
	}

	return gErr
}

func (e GraphError) Unwrap() error {
	return e.InnerError
}

func (e ErrorLocation) String() string {
	return fmt.Sprintf("%d:%d", e.Line, e.Column)
}

// getLineAndColumnFromOffset takes a string and an offset and returns the line and column
// corresponding to that offset.
func getLineAndColumnFromOffset(input string, offset int) (line int, column int) {
	line = 1
	column = 1
	for i := 0; i < offset && i < len(input); i++ {
		if input[i] == '\n' {
			line++
			column = 0
		} else {
			column++
		}
	}
	return
}

// transformJsonError takes an error from the json package and transforms it into a GraphError.
func transformJsonError(input string, err error) error {
	if err == nil {
		return nil
	}

	// Deal with errors from the Json unmarshalling:
	// * json.UnmarshalTypeError
	// * json.SyntaxError
	// Each of these has an Offset field that we can use to get the line and column.
	// We can then use that to create a GraphError.

	var uterr *json.UnmarshalTypeError
	if errors.As(err, &uterr) {
		line, column := getLineAndColumnFromOffset(input, int(uterr.Offset))
		return GraphError{
			Message:    err.Error(),
			Locations:  []ErrorLocation{{Line: line, Column: column}},
			InnerError: err,
		}
	}

	var serr *json.SyntaxError
	if errors.As(err, &serr) {
		line, column := getLineAndColumnFromOffset(input, int(serr.Offset))
		return GraphError{
			Message:    err.Error(),
			Locations:  []ErrorLocation{{Line: line, Column: column}},
			InnerError: err,
		}
	}

	// Otherwise, return a degenerate GraphError.
	return GraphError{
		Message:    err.Error(),
		InnerError: err,
	}
}

// MarshalJSON implements the json.Marshaler interface for GraphError.
// This allows us to format the error in the way that GraphQL expects.
func (e GraphError) MarshalJSON() ([]byte, error) {
	return e.marshalJSONWithMode(false) // Default to development mode for backward compatibility
}

// MarshalJSONProduction marshals the error for production use, sanitizing sensitive information
func (e GraphError) MarshalJSONProduction() ([]byte, error) {
	return e.marshalJSONWithMode(true)
}

// marshalJSONWithMode handles JSON marshaling with production mode control
func (e GraphError) marshalJSONWithMode(productionMode bool) ([]byte, error) {
	// We need to create a new type that has all of the fields of GraphError
	// except for InnerError and our internal fields
	type graphErrorForSerialization struct {
		Message    string            `json:"message"`
		Locations  []ErrorLocation   `json:"locations,omitempty"`
		Path       []string          `json:"path,omitempty"`
		Extensions map[string]string `json:"extensions,omitempty"`
	}

	// Create a new instance and copy the fields over
	var gErr graphErrorForSerialization
	gErr.Locations = e.Locations
	gErr.Path = e.Path

	// Choose message based on production mode
	if productionMode && e.ProductionMessage != "" {
		gErr.Message = e.ProductionMessage
	} else {
		gErr.Message = e.Message
		// In development mode, append inner error details to the message
		if e.InnerError != nil {
			gErr.Message = fmt.Sprintf("%s: %s", gErr.Message, e.InnerError.Error())
		}
	}

	// Handle extensions based on production mode
	if productionMode {
		// In production mode, filter out sensitive extensions
		gErr.Extensions = make(map[string]string)
		for k, v := range e.Extensions {
			// Skip extensions that are marked as sensitive
			if _, isSensitive := e.SensitiveExtensions[k]; isSensitive {
				continue
			}
			gErr.Extensions[k] = v
		}
		// Remove empty extensions map
		if len(gErr.Extensions) == 0 {
			gErr.Extensions = nil
		}
	} else {
		// In development mode, include all extensions (both regular and sensitive)
		gErr.Extensions = make(map[string]string)
		for k, v := range e.Extensions {
			gErr.Extensions[k] = v
		}
		for k, v := range e.SensitiveExtensions {
			gErr.Extensions[k] = v
		}
		// Remove empty extensions map
		if len(gErr.Extensions) == 0 {
			gErr.Extensions = nil
		}
	}

	// Marshal the new type
	return json.Marshal(gErr)
}

// AddExtension adds a key-value pair to the Extensions field of a GraphError.
// Extensions in a GraphError provide a way to include additional error
// information that is not part of the standard error fields.
//
// If the Extensions map is nil, it will be initialized before adding the key-value pair.
//
// Parameters:
// - key: The key for the extension. It represents the name or identifier for the additional data.
// - value: The value associated with the key. It provides extra context or data for the error.
func (e *GraphError) AddExtension(key string, value string) {
	if e.Extensions == nil {
		e.Extensions = map[string]string{}
	}
	e.Extensions[key] = value
}

func formatError(errs ...error) string {
	return formatErrorWithMode(false, errs...) // Default to development mode for backward compatibility
}

func formatErrorWithMode(productionMode bool, errs ...error) string {
	var resultErrors []json.RawMessage
	for _, err := range errs {
		var ge GraphError
		if !errors.As(err, &ge) {
			ge = GraphError{
				Message:    err.Error(),
				InnerError: err,
			}
		}

		var errJson []byte
		if productionMode {
			errJson, _ = ge.MarshalJSONProduction()
		} else {
			errJson, _ = ge.MarshalJSON()
		}
		resultErrors = append(resultErrors, errJson)
	}
	resultMap := map[string]any{
		"errors": resultErrors,
	}
	resultJson, _ := json.Marshal(resultMap)
	return string(resultJson)
}

func (e UnknownCommandError) Unwrap() error {
	return e.GraphError
}
