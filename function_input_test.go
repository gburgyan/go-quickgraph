package quickgraph

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type MyEnum string

const (
	EnumVal1 MyEnum = "EnumVal1"
	EnumVal2 MyEnum = "EnumVal2"
	EnumVal3 MyEnum = "EnumVal3"
)

func (e MyEnum) String() string {
	return string(e)
}

func (e *MyEnum) UnmarshalString(input string) (interface{}, error) {
	switch input {
	case "EnumVal1":
		return EnumVal1, nil
	case "EnumVal2":
		return EnumVal2, nil
	case "EnumVal3":
		return EnumVal3, nil
	default:
		return nil, fmt.Errorf("invalid enum value %s", input)
	}
}

func Test_parseIdentifierIntoValue(t *testing.T) {
	var x MyEnum
	v := reflect.ValueOf(&x)

	// Test a known identifier.
	err := parseIdentifierIntoValue("EnumVal2", v)
	assert.Equal(t, EnumVal2, x, "The enum value should have been set to EnumVal2")
	assert.NoError(t, err)

	// Test an unknown identifier.
	err = parseIdentifierIntoValue("Unknown", v)
	assert.Error(t, err)
}
