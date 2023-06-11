package quickgraph

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type TestUnion struct {
	A *string
	B *int
}

type NonUnion struct {
	A string
	B int
}

func TestDeferenceUnionType(t *testing.T) {
	// Test 1: Union type with only one field that is not nil.
	s := "test"
	testUnion := TestUnion{A: &s, B: nil}
	res, err := deferenceUnionType(testUnion)
	assert.NoError(t, err)
	assert.Equal(t, s, res)

	// Test 2: Union type with more than one field that is not nil.
	i := 1
	testUnion = TestUnion{A: &s, B: &i}
	_, err = deferenceUnionType(testUnion)
	assert.Error(t, err)
	assert.Equal(t, "more than one field in union type is not nil", err.Error())

	// Test 3: Union type with all fields that are nil.
	testUnion = TestUnion{A: nil, B: nil}
	_, err = deferenceUnionType(testUnion)
	assert.Error(t, err)
	assert.Equal(t, "no fields in union type are not nil", err.Error())

	// Test 4: Non-union type struct.
	nonUnion := NonUnion{A: "test", B: 1}
	_, err = deferenceUnionType(nonUnion)
	assert.Error(t, err)
	assert.Equal(t, "fields in union type must be pointers, maps, slices, or interfaces", err.Error())
}
