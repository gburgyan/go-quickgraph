package quickgraph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
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
