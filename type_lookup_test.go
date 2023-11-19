package quickgraph

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseFieldLookup_JsonTag(t *testing.T) {
	field := reflect.StructField{
		Name: "TestField",
		Tag:  reflect.StructTag(`json:"test_json"`),
		Type: reflect.TypeOf(""),
	}
	g := Graphy{}
	result := g.baseFieldLookup(field, []int{0})

	assert.Equal(t, "test_json", result.name)
	assert.Equal(t, FieldTypeField, result.fieldType)
}

func TestBaseFieldLookup_GraphyTag(t *testing.T) {
	field := reflect.StructField{
		Name: "TestField",
		Tag:  reflect.StructTag(`graphy:"test_graphy"`),
		Type: reflect.TypeOf(""),
	}
	g := Graphy{}
	result := g.baseFieldLookup(field, []int{0})

	assert.Equal(t, "test_graphy", result.name)
	assert.Equal(t, FieldTypeField, result.fieldType)
}

func TestBaseFieldLookup_JsonTagIgnore(t *testing.T) {
	field := reflect.StructField{
		Name: "TestField",
		Tag:  reflect.StructTag(`json:"-"`),
		Type: reflect.TypeOf(""),
	}
	g := Graphy{}
	result := g.baseFieldLookup(field, []int{0})

	assert.Equal(t, "", result.name)
}

func TestBaseFieldLookup_GraphyTagDeprecated(t *testing.T) {
	field := reflect.StructField{
		Name: "TestField",
		Tag:  reflect.StructTag(`graphy:"name=test_graphy,deprecated=Deprecated for testing"`),
		Type: reflect.TypeOf(""),
	}
	g := Graphy{}
	result := g.baseFieldLookup(field, []int{0})

	assert.Equal(t, "test_graphy", result.name)
	assert.Equal(t, FieldTypeField, result.fieldType)
	assert.True(t, result.isDeprecated)
	assert.Equal(t, "Deprecated for testing", result.deprecatedReason)
}
