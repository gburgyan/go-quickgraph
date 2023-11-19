package quickgraph

import "reflect"

var enumUnmarshalerType = reflect.TypeOf((*EnumUnmarshaler)(nil)).Elem()
var stringEnumValuesType = reflect.TypeOf((*StringEnumValues)(nil)).Elem()

// EnumUnmarshaler provides an interface for types that can unmarshal
// a string representation into their enumerated type. This is useful
// for types that need to convert a string, typically from external sources
// like JSON or XML, into a specific enumerated type in Go.
//
// UnmarshalString should return the appropriate enumerated value for the
// given input string, or an error if the input is not valid for the enumeration.
type EnumUnmarshaler interface {
	UnmarshalString(input string) (interface{}, error)
}

// StringEnumValues provides an interface for types that can return
// a list of valid string representations for their enumeration.
// This can be useful in scenarios like validation or auto-generation
// of documentation where a list of valid enum values is required.
//
// EnumValues should return a slice of strings representing the valid values
// for the enumeration.
type StringEnumValues interface {
	EnumValues() []EnumValue
}

type EnumValue struct {
	Name              string
	Description       string
	IsDeprecated      bool
	DeprecationReason string
}
