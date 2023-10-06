package quickgraph

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alecthomas/participle/v2/lexer"
	"strings"
)

type GraphError struct {
	Message    string            `json:"message"`
	Locations  []ErrorLocation   `json:"locations,omitempty"`
	Path       []string          `json:"path,omitempty"`
	Extensions map[string]string `json:"extensions,omitempty"`
	InnerError error             `json:"-"`
}

type ErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

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

	return s.String()
}

func NewGraphError(message string, pos lexer.Position, paths ...string) error {
	var gErr GraphError
	if pos.Offset > 0 {
		loc := lexerPositionError(pos)
		gErr.Locations = append(gErr.Locations, loc)
	}
	gErr.Message = message
	gErr.Path = paths
	return gErr
}

func lexerPositionError(pos lexer.Position) ErrorLocation {
	return ErrorLocation{
		Line:   pos.Line,
		Column: pos.Column,
	}
}

func AugmentGraphError(err error, message string, pos lexer.Position, paths ...string) error {
	var gErr GraphError

	// We should never have a regular error wrapping a GraphError. If that ever happens
	// the extra error information is lost as we're only using the wrapped GraphError.
	ok := errors.As(err, &gErr)
	if !ok {
		gErr = GraphError{
			Message:    err.Error(),
			InnerError: err,
		}
	}

	// If the message isn't set, set it.
	if gErr.Message != "" {
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
