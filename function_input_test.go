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

func Test_parseIdentifierIntoValue_Enum(t *testing.T) {
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

func Test_parseIdentifierIntoValue_Bool(t *testing.T) {
	var x bool
	v := reflect.ValueOf(&x).Elem()

	// Test a known identifier.
	err := parseIdentifierIntoValue("true", v)
	assert.Equal(t, true, x)
	assert.NoError(t, err)

	err = parseIdentifierIntoValue("false", v)
	assert.Equal(t, false, x)
	assert.NoError(t, err)

	err = parseIdentifierIntoValue("random", v)
	assert.Error(t, err)
}

func Test_parseIdentifierIntoValue_BoolPtr(t *testing.T) {
	var x *bool
	v := reflect.ValueOf(&x).Elem()

	// Test a known identifier.
	err := parseIdentifierIntoValue("true", v)
	assert.Equal(t, true, *x)
	assert.NoError(t, err)

	err = parseIdentifierIntoValue("false", v)
	assert.Equal(t, false, *x)
	assert.NoError(t, err)
}

func Test_parseFloatIntoValue_Base(t *testing.T) {
	x := 42.23

	inVal := genericValue{
		Float: &x,
	}

	var outVal float64
	v := reflect.ValueOf(&outVal).Elem()

	req := &Request{}
	err := parseInputIntoValue(req, inVal, v)

	assert.NoError(t, err)
	assert.Equal(t, x, outVal)
}

func Test_parseFloatIntoValue_Ptr(t *testing.T) {
	x := 42.23

	inVal := genericValue{
		Float: &x,
	}

	var outVal *float64
	v := reflect.ValueOf(&outVal).Elem()

	req := &Request{}
	err := parseInputIntoValue(req, inVal, v)

	assert.NoError(t, err)
	assert.Equal(t, x, *outVal)
}

func Test_parseIntIntoValue_Base(t *testing.T) {
	var x int64 = 42

	inVal := genericValue{
		Int: &x,
	}

	var outVal int64
	v := reflect.ValueOf(&outVal).Elem()

	req := &Request{}
	err := parseInputIntoValue(req, inVal, v)

	assert.NoError(t, err)
	assert.Equal(t, x, outVal)
}

func Test_parseIntIntoValue_Ptr(t *testing.T) {
	var x int64 = 42

	inVal := genericValue{
		Int: &x,
	}

	var outVal *int64
	v := reflect.ValueOf(&outVal).Elem()

	req := &Request{}
	err := parseInputIntoValue(req, inVal, v)

	assert.NoError(t, err)
	assert.Equal(t, x, *outVal)
}
