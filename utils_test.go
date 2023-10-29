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
