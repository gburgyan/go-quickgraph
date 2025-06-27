package quickgraph

import (
	"context"
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

func Test_parseNothing_Error(t *testing.T) {
	var x int
	v := reflect.ValueOf(&x).Elem()

	req := &request{}
	err := parseInputIntoValue(context.Background(), req, genericValue{}, v)

	assert.EqualError(t, err, "no input found to parse into value")
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

	req := &request{}
	err := parseInputIntoValue(context.Background(), req, inVal, v)

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

	req := &request{}
	err := parseInputIntoValue(context.Background(), req, inVal, v)

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

	req := &request{}
	err := parseInputIntoValue(context.Background(), req, inVal, v)

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

	req := &request{}
	err := parseInputIntoValue(context.Background(), req, inVal, v)

	assert.NoError(t, err)
	assert.Equal(t, x, *outVal)
}

func Test_parseStringIntoValue_Base(t *testing.T) {
	x := "\"hello\""

	inVal := genericValue{
		String: &x,
	}

	var outVal string
	v := reflect.ValueOf(&outVal).Elem()

	req := &request{}
	err := parseInputIntoValue(context.Background(), req, inVal, v)

	assert.NoError(t, err)
	assert.Equal(t, "hello", outVal)
}

func Test_parseStringIntoValue_Ptr(t *testing.T) {
	x := "\"hello\""

	inVal := genericValue{
		String: &x,
	}

	var outVal *string
	v := reflect.ValueOf(&outVal).Elem()

	req := &request{}
	err := parseInputIntoValue(context.Background(), req, inVal, v)

	assert.NoError(t, err)
	assert.Equal(t, "hello", *outVal)
}

func Test_parseIdentifierIntoValue_Base(t *testing.T) {
	x := "hello"

	inVal := genericValue{
		Identifier: &x,
	}

	var outVal string
	v := reflect.ValueOf(&outVal).Elem()

	req := &request{}
	err := parseInputIntoValue(context.Background(), req, inVal, v)

	assert.NoError(t, err)
	assert.Equal(t, "hello", outVal)
}

func Test_parseIdentifierIntoValue_Ptr(t *testing.T) {
	x := "hello"

	inVal := genericValue{
		Identifier: &x,
	}

	var outVal *string
	v := reflect.ValueOf(&outVal).Elem()

	req := &request{}
	err := parseInputIntoValue(context.Background(), req, inVal, v)

	assert.NoError(t, err)
	assert.Equal(t, "hello", *outVal)
}

func Test_parseIdentifierIntoValue_BaseType(t *testing.T) {
	x := "hello"

	inVal := genericValue{
		Identifier: &x,
	}

	type myType string
	var outVal myType
	v := reflect.ValueOf(&outVal).Elem()

	req := &request{}
	err := parseInputIntoValue(context.Background(), req, inVal, v)

	assert.NoError(t, err)
	assert.Equal(t, myType("hello"), outVal)
}

func Test_parseIdentifierIntoValue_PtrType(t *testing.T) {
	x := "hello"

	inVal := genericValue{
		Identifier: &x,
	}

	type myType string
	var outVal *myType
	v := reflect.ValueOf(&outVal).Elem()

	req := &request{}
	err := parseInputIntoValue(context.Background(), req, inVal, v)

	assert.NoError(t, err)
	assert.Equal(t, myType("hello"), *outVal)
}

func Test_parseIntIntoValue_Overflow(t *testing.T) {
	tests := []struct {
		name      string
		value     int64
		target    interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name:      "int8 overflow positive",
			value:     200,
			target:    new(int8),
			wantError: true,
			errorMsg:  "value 200 overflows int8",
		},
		{
			name:      "int8 overflow negative",
			value:     -200,
			target:    new(int8),
			wantError: true,
			errorMsg:  "value -200 overflows int8",
		},
		{
			name:      "int8 valid",
			value:     100,
			target:    new(int8),
			wantError: false,
		},
		{
			name:      "int16 overflow",
			value:     40000,
			target:    new(int16),
			wantError: true,
			errorMsg:  "value 40000 overflows int16",
		},
		{
			name:      "int32 overflow",
			value:     3000000000,
			target:    new(int32),
			wantError: true,
			errorMsg:  "value 3000000000 overflows int32",
		},
		{
			name:      "uint8 negative",
			value:     -1,
			target:    new(uint8),
			wantError: true,
			errorMsg:  "value -1 overflows uint8",
		},
		{
			name:      "uint8 overflow",
			value:     300,
			target:    new(uint8),
			wantError: true,
			errorMsg:  "value 300 overflows uint8",
		},
		{
			name:      "uint8 valid",
			value:     200,
			target:    new(uint8),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.target).Elem()
			err := parseIntIntoValue(nil, tt.value, v)

			if tt.wantError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_parseIntIntoValue_Float(t *testing.T) {
	tests := []struct {
		name   string
		value  int64
		target interface{}
	}{
		{
			name:   "int to float32",
			value:  42,
			target: new(float32),
		},
		{
			name:   "int to float64",
			value:  100,
			target: new(float64),
		},
		{
			name:   "negative int to float32",
			value:  -42,
			target: new(float32),
		},
		{
			name:   "negative int to float64",
			value:  -100,
			target: new(float64),
		},
		{
			name:   "large int to float64",
			value:  1000000,
			target: new(float64),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.target).Elem()
			err := parseIntIntoValue(nil, tt.value, v)

			assert.NoError(t, err)

			// Check the value was set correctly
			switch tt.target.(type) {
			case *float32:
				assert.Equal(t, float32(tt.value), v.Interface().(float32))
			case *float64:
				assert.Equal(t, float64(tt.value), v.Interface().(float64))
			}
		})
	}
}

func Test_parseIntIntoValue_UnsignedTypes(t *testing.T) {
	tests := []struct {
		name      string
		value     int64
		target    interface{}
		wantError bool
		errorMsg  string
	}{
		// uint16 tests
		{
			name:      "uint16 negative",
			value:     -1,
			target:    new(uint16),
			wantError: true,
			errorMsg:  "value -1 overflows uint16",
		},
		{
			name:      "uint16 overflow",
			value:     70000,
			target:    new(uint16),
			wantError: true,
			errorMsg:  "value 70000 overflows uint16",
		},
		{
			name:      "uint16 valid",
			value:     30000,
			target:    new(uint16),
			wantError: false,
		},
		// uint32 tests
		{
			name:      "uint32 negative",
			value:     -1,
			target:    new(uint32),
			wantError: true,
			errorMsg:  "value -1 overflows uint32",
		},
		{
			name:      "uint32 overflow",
			value:     5000000000,
			target:    new(uint32),
			wantError: true,
			errorMsg:  "value 5000000000 overflows uint32",
		},
		{
			name:      "uint32 valid",
			value:     3000000000,
			target:    new(uint32),
			wantError: false,
		},
		// uint64 tests
		{
			name:      "uint64 negative",
			value:     -1,
			target:    new(uint64),
			wantError: true,
			errorMsg:  "value -1 cannot be negative for uint64",
		},
		{
			name:      "uint64 valid",
			value:     9223372036854775807, // max int64
			target:    new(uint64),
			wantError: false,
		},
		// uint tests
		{
			name:      "uint negative",
			value:     -1,
			target:    new(uint),
			wantError: true,
			errorMsg:  "value -1 overflows uint",
		},
		{
			name:      "uint valid",
			value:     1000000,
			target:    new(uint),
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.target).Elem()
			err := parseIntIntoValue(nil, tt.value, v)

			if tt.wantError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMsg, err.Error())
			} else {
				assert.NoError(t, err)
				// Verify the value was set correctly
				switch tt.target.(type) {
				case *uint16:
					assert.Equal(t, uint16(tt.value), v.Interface().(uint16))
				case *uint32:
					assert.Equal(t, uint32(tt.value), v.Interface().(uint32))
				case *uint64:
					assert.Equal(t, uint64(tt.value), v.Interface().(uint64))
				case *uint:
					assert.Equal(t, uint(tt.value), v.Interface().(uint))
				}
			}
		})
	}
}

func Test_parseIntIntoValue_RegularInt(t *testing.T) {
	tests := []struct {
		name   string
		value  int64
		target interface{}
	}{
		{
			name:   "regular int",
			value:  42,
			target: new(int),
		},
		{
			name:   "negative int",
			value:  -42,
			target: new(int),
		},
		{
			name:   "zero int",
			value:  0,
			target: new(int),
		},
		{
			name:   "large int",
			value:  1000000,
			target: new(int),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.target).Elem()
			err := parseIntIntoValue(nil, tt.value, v)

			assert.NoError(t, err)
			assert.Equal(t, int(tt.value), v.Interface().(int))
		})
	}
}

// Test for uint overflow on 32-bit systems
func Test_parseIntIntoValue_Uint32BitSystem(t *testing.T) {
	// This test simulates behavior on a 32-bit system where uint has size 4
	// We can't actually change the architecture, but we can test the logic
	// by checking if the value would overflow a 32-bit uint
	var target uint
	v := reflect.ValueOf(&target).Elem()

	// Value that would overflow a 32-bit uint
	value := int64(5000000000)

	// On a 64-bit system, this won't error, but the code checks for it
	err := parseIntIntoValue(nil, value, v)

	// On 64-bit systems, uint is 64-bit, so this should succeed
	if v.Type().Size() == 8 {
		assert.NoError(t, err)
		assert.Equal(t, uint(value), target)
	} else {
		// On 32-bit systems, this would error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "overflows uint")
	}
}

// Test struct for parseMapIntoValue tests
type TestMapStruct struct {
	Name     string  `graphy:"name"`
	Age      int     `graphy:"age"`
	Optional *bool   `graphy:"optional"`
	NoTag    *string // Field without JSON tag - pointer so it's optional
	Ignored  string  `graphy:"-"` // Field that should be ignored
}

func Test_parseMapIntoValue_NilPointerStruct(t *testing.T) {
	// Test the specific code path where targetValue is a nil pointer to a struct
	// This tests the lines:
	// if targetType.Kind() == reflect.Ptr {
	//     isNilPtr := targetValue.IsNil()
	//     targetType = targetType.Elem()
	//     if isNilPtr {
	//         targetValue.Set(reflect.New(targetType))
	//     }
	//     targetValue = targetValue.Elem()
	// }

	// Create a nil pointer to TestMapStruct
	var target *TestMapStruct
	v := reflect.ValueOf(&target).Elem()

	// Create input map
	nameVal := "\"John\""
	ageVal := int64(30)

	inValue := genericValue{
		Map: []namedValue{
			{
				Name: "name",
				Value: genericValue{
					String: &nameVal,
				},
			},
			{
				Name: "age",
				Value: genericValue{
					Int: &ageVal,
				},
			},
		},
	}

	req := &request{}
	err := parseMapIntoValue(context.Background(), req, inValue, v)

	assert.NoError(t, err)
	assert.NotNil(t, target)
	assert.Equal(t, "John", target.Name)
	assert.Equal(t, 30, target.Age)
}

func Test_parseMapIntoValue_NonNilPointerStruct(t *testing.T) {
	// Test with a non-nil pointer to ensure the other branch is covered
	target := &TestMapStruct{
		Name: "Initial",
		Age:  0,
	}
	v := reflect.ValueOf(&target).Elem()

	// Create input map
	nameVal := "\"Jane\""
	ageVal := int64(25)

	inValue := genericValue{
		Map: []namedValue{
			{
				Name: "name",
				Value: genericValue{
					String: &nameVal,
				},
			},
			{
				Name: "age",
				Value: genericValue{
					Int: &ageVal,
				},
			},
		},
	}

	req := &request{}
	err := parseMapIntoValue(context.Background(), req, inValue, v)

	assert.NoError(t, err)
	assert.NotNil(t, target)
	assert.Equal(t, "Jane", target.Name)
	assert.Equal(t, 25, target.Age)
}

func Test_parseMapIntoValue_FieldWithoutJsonTag(t *testing.T) {
	// Test that fields without JSON tags can be set by name
	var target TestMapStruct
	v := reflect.ValueOf(&target).Elem()

	// Create input map with all required fields plus NoTag field
	nameVal := "\"TestName\""
	ageVal := int64(40)
	noTagVal := "\"NoTagValue\""

	inValue := genericValue{
		Map: []namedValue{
			{
				Name: "name",
				Value: genericValue{
					String: &nameVal,
				},
			},
			{
				Name: "age",
				Value: genericValue{
					Int: &ageVal,
				},
			},
			{
				Name: "NoTag",
				Value: genericValue{
					String: &noTagVal,
				},
			},
		},
	}

	req := &request{}
	err := parseMapIntoValue(context.Background(), req, inValue, v)

	assert.NoError(t, err)
	assert.NotNil(t, target.NoTag)
	assert.Equal(t, "NoTagValue", *target.NoTag)
}

func Test_parseMapIntoValue_MissingRequiredField(t *testing.T) {
	// Test error when required field is missing
	var target TestMapStruct
	v := reflect.ValueOf(&target).Elem()

	// Create input map missing required fields
	inValue := genericValue{
		Map: []namedValue{},
	}

	req := &request{}
	err := parseMapIntoValue(context.Background(), req, inValue, v)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required fields")
}

func Test_parseMapIntoValue_UnknownField(t *testing.T) {
	// Test error when trying to set unknown field
	var target TestMapStruct
	v := reflect.ValueOf(&target).Elem()

	// Create input map with unknown field
	unknownVal := "\"value\""

	inValue := genericValue{
		Map: []namedValue{
			{
				Name: "unknownField",
				Value: genericValue{
					String: &unknownVal,
				},
			},
		},
	}

	req := &request{}
	err := parseMapIntoValue(context.Background(), req, inValue, v)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field unknownField not found in input struct")
}

func Test_parseInputIntoValue_StructWithMap(t *testing.T) {
	// Test parseInputIntoValue with a struct target and map input
	// This ensures the isStruct branch is covered
	var target TestMapStruct
	v := reflect.ValueOf(&target).Elem()

	nameVal := "\"Alice\""
	ageVal := int64(35)
	ignoredVal := "\"Ignored\""

	inValue := genericValue{
		Map: []namedValue{
			{
				Name: "name",
				Value: genericValue{
					String: &nameVal,
				},
			},
			{
				Name: "age",
				Value: genericValue{
					Int: &ageVal,
				},
			},
			{
				Name: "Ignored",
				Value: genericValue{
					String: &ignoredVal,
				},
			},
		},
	}

	req := &request{}
	err := parseInputIntoValue(context.Background(), req, inValue, v)

	assert.NoError(t, err)
	assert.Equal(t, "Alice", target.Name)
	assert.Equal(t, 35, target.Age)
}

func Test_parseMapIntoValue_IgnoredJsonTag(t *testing.T) {
	// Test that fields with graphy:"-" tag are properly ignored
	var target TestMapStruct
	v := reflect.ValueOf(&target).Elem()

	nameVal := "\"Bob\""
	ageVal := int64(45)

	inValue := genericValue{
		Map: []namedValue{
			{
				Name: "name",
				Value: genericValue{
					String: &nameVal,
				},
			},
			{
				Name: "age",
				Value: genericValue{
					Int: &ageVal,
				},
			},
		},
	}

	req := &request{}
	err := parseMapIntoValue(context.Background(), req, inValue, v)

	// Should succeed even though Ignored field is not provided
	assert.NoError(t, err)
	assert.Equal(t, "Bob", target.Name)
	assert.Equal(t, 45, target.Age)
	assert.Equal(t, "", target.Ignored) // Should remain empty
}
