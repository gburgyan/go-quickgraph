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
