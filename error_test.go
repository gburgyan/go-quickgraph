package quickgraph

import (
	"context"
	"errors"
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
			column: 1,
		},
		{
			name:   "Multiple Lines With Offset On Second Line",
			input:  "Hello,\nWorld!",
			offset: 8,
			line:   2,
			column: 2,
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

	assert.Empty(t, resultAny)
	var uce UnknownCommandError
	errors.As(err, &uce)
	assert.Contains(t, uce.Commands, "hero")
}
