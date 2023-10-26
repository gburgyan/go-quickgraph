package quickgraph

import (
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
