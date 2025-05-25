package quickgraph

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_keys(t *testing.T) {
	m := map[string]int{
		"foo": 1,
		"bar": 2,
	}
	keys := keys(m)
	assert.Lenf(t, keys, 2, "keys should have 2 elements")
	assert.Contains(t, keys, "foo", "keys should contain foo")
	assert.Contains(t, keys, "bar", "keys should contain bar")
}

type testStringer struct {
	value string
}

func (t *testStringer) String() string {
	return t.value
}

func Test_toStringSlice(t *testing.T) {
	items := []fmt.Stringer{
		&testStringer{value: "foo"},
		&testStringer{value: "bar"},
	}
	result := toStringSlice(items)
	assert.Lenf(t, result, 2, "result should have 2 elements")
	assert.Equal(t, "foo", result[0])
	assert.Equal(t, "bar", result[1])
}

func Test_sortedKeys_ReturnsSortedKeys(t *testing.T) {
	m := map[string]int{
		"banana": 1,
		"apple":  2,
		"cherry": 3,
	}
	expected := []string{"apple", "banana", "cherry"}

	result := sortedKeys(m)

	assert.Equal(t, expected, result)
}

func Test_sortedKeys_WhenMapIsEmpty_ReturnsEmptySlice(t *testing.T) {
	m := map[string]int{}

	result := sortedKeys(m)

	assert.Empty(t, result)
}

func Test_sortedKeys_WhenMapHasOneElement_ReturnsSliceWithOneElement(t *testing.T) {
	m := map[string]int{
		"apple": 1,
	}
	expected := []string{"apple"}

	result := sortedKeys(m)

	assert.Equal(t, expected, result)
}

func Test_parseVariableName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantError bool
		errorMsg  string
	}{
		{
			name:     "valid simple variable",
			input:    "$foo",
			wantName: "foo",
		},
		{
			name:     "valid with underscore",
			input:    "$_foo",
			wantName: "_foo",
		},
		{
			name:     "valid with numbers",
			input:    "$foo123",
			wantName: "foo123",
		},
		{
			name:     "valid complex",
			input:    "$_foo_Bar_123",
			wantName: "_foo_Bar_123",
		},
		{
			name:      "empty string",
			input:     "",
			wantError: true,
			errorMsg:  `invalid variable reference: "" (too short)`,
		},
		{
			name:      "just dollar sign",
			input:     "$",
			wantError: true,
			errorMsg:  `invalid variable reference: "$" (too short)`,
		},
		{
			name:      "no dollar sign",
			input:     "foo",
			wantError: true,
			errorMsg:  `invalid variable reference: "foo" (must start with '$')`,
		},
		{
			name:      "starts with number",
			input:     "$123foo",
			wantError: true,
			errorMsg:  `invalid variable name: "123foo" (must match /[_A-Za-z][_0-9A-Za-z]*/)`,
		},
		{
			name:      "contains hyphen",
			input:     "$foo-bar",
			wantError: true,
			errorMsg:  `invalid variable name: "foo-bar" (must match /[_A-Za-z][_0-9A-Za-z]*/)`,
		},
		{
			name:      "contains space",
			input:     "$foo bar",
			wantError: true,
			errorMsg:  `invalid variable name: "foo bar" (must match /[_A-Za-z][_0-9A-Za-z]*/)`,
		},
		{
			name:      "special characters",
			input:     "$foo!@#",
			wantError: true,
			errorMsg:  `invalid variable name: "foo!@#" (must match /[_A-Za-z][_0-9A-Za-z]*/)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, err := parseVariableName(tt.input)

			if tt.wantError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantName, gotName)
			}
		})
	}
}
