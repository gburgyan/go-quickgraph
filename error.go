package quickgraph

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alecthomas/participle/v2/lexer"
	"strings"
)

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
type GraphError struct {
	Message    string            `json:"message"`
	Locations  []ErrorLocation   `json:"locations,omitempty"`
	Path       []string          `json:"path,omitempty"`
	Extensions map[string]string `json:"extensions,omitempty"`
	InnerError error             `json:"-"`
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
		for i, l := range e.Locations {
			if i > 0 {
				s.WriteString(", ")
			}
			s.WriteString(l.String())
		}
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
	// There is no really good way to have the inner error get exposed, even
	// though it has good information. We need a custom marshaller to do this.

	// We need to create a new type that has all of the fields of GraphError
	// except for InnerError. We can then marshal that type.
	type graphErrorNoInnerError struct {
		Message    string            `json:"message"`
		Locations  []ErrorLocation   `json:"locations,omitempty"`
		Path       []string          `json:"path,omitempty"`
		Extensions map[string]string `json:"extensions,omitempty"`
	}

	// Create a new instance of the new type and copy the fields over.
	var gErr graphErrorNoInnerError
	gErr.Message = e.Message
	gErr.Locations = e.Locations
	gErr.Path = e.Path
	gErr.Extensions = e.Extensions

	// If there is an inner error, append that to the message.
	if e.InnerError != nil {
		gErr.Message = fmt.Sprintf("%s: %s", gErr.Message, e.InnerError.Error())
	}

	// Marshal the new type.
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

func formatError(err error) string {
	// If the error is a GraphError, make this into a graph-style error JSON. Otherwise, return "".
	var ge GraphError
	if errors.As(err, &ge) {
		resultMap := map[string]any{
			"errors": []any{
				ge,
			},
		}
		resultJson, _ := json.Marshal(resultMap)
		return string(resultJson)
	}
	return ""
}

func (e UnknownCommandError) Unwrap() error {
	return e.GraphError
}
