package quickgraph

import (
	"fmt"
	"strings"
)

type GraphError struct {
	Message    string            `json:"message"`
	Locations  []ErrorLocation   `json:"locations,omitempty"`
	Path       []string          `json:"path,omitempty"`
	Extensions map[string]string `json:"extensions,omitempty"`
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

func (e ErrorLocation) String() string {
	return fmt.Sprintf("%d:%d", e.Line, e.Column)
}
