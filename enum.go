package quickgraph

import "reflect"

var enumUnmarshalerType = reflect.TypeOf((*EnumUnmarshaler)(nil)).Elem()
var stringEnumValuesType = reflect.TypeOf((*StringEnumValues)(nil)).Elem()

type EnumUnmarshaler interface {
	UnmarshalString(input string) (interface{}, error)
}

type StringEnumValues interface {
	EnumValues() []string
}
