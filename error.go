package quickgraph

import (
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
		loc := ErrorLocation{
			Line:   pos.Line,
			Column: pos.Column,
		}
		gErr.Locations = append(gErr.Locations, loc)
	}
	gErr.Message = message
	gErr.Path = paths
	return gErr
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
