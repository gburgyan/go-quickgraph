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

// Test types with circular references
type CircularA struct {
	Name string
	B    *CircularB
}

type CircularB struct {
	Name string
	C    *CircularC
}

type CircularC struct {
	Name string
	A    *CircularA // Completes the circle: A -> B -> C -> A
}

// Test self-referencing type
type SelfRef struct {
	Name  string
	Child *SelfRef
}

// Test mutual references
type MutualA struct {
	Name string
	B    *MutualB
}

type MutualB struct {
	Name string
	A    *MutualA
}

func TestTypeLookup_CircularReferences(t *testing.T) {
	g := Graphy{}

	// Test that typeLookup doesn't cause stack overflow with circular types
	typeA := reflect.TypeOf((*CircularA)(nil)).Elem()
	lookupA := g.typeLookup(typeA)

	assert.NotNil(t, lookupA)
	assert.Equal(t, "CircularA", lookupA.name)
	assert.Contains(t, lookupA.fields, "Name")
	assert.Contains(t, lookupA.fields, "B")

	// The type should be cached
	lookupA2 := g.typeLookup(typeA)
	assert.Equal(t, lookupA, lookupA2)
}

func TestTypeLookup_SelfReference(t *testing.T) {
	g := Graphy{}

	// Test self-referencing type
	typeSelf := reflect.TypeOf((*SelfRef)(nil)).Elem()
	lookupSelf := g.typeLookup(typeSelf)

	assert.NotNil(t, lookupSelf)
	assert.Equal(t, "SelfRef", lookupSelf.name)
	assert.Contains(t, lookupSelf.fields, "Name")
	assert.Contains(t, lookupSelf.fields, "Child")
}

func TestTypeLookup_MutualReferences(t *testing.T) {
	g := Graphy{}

	// Test mutually referencing types
	typeA := reflect.TypeOf((*MutualA)(nil)).Elem()
	typeB := reflect.TypeOf((*MutualB)(nil)).Elem()

	lookupA := g.typeLookup(typeA)
	lookupB := g.typeLookup(typeB)

	assert.NotNil(t, lookupA)
	assert.NotNil(t, lookupB)
	assert.Equal(t, "MutualA", lookupA.name)
	assert.Equal(t, "MutualB", lookupB.name)

	// Both should have their fields properly populated
	assert.Contains(t, lookupA.fields, "Name")
	assert.Contains(t, lookupA.fields, "B")
	assert.Contains(t, lookupB.fields, "Name")
	assert.Contains(t, lookupB.fields, "A")
}

// Test with deeper circular reference chain
type DeepCircularA struct {
	Name string
	B    *DeepCircularB
}

type DeepCircularB struct {
	Name string
	C    *DeepCircularC
}

type DeepCircularC struct {
	Name string
	D    *DeepCircularD
}

type DeepCircularD struct {
	Name string
	E    *DeepCircularE
}

type DeepCircularE struct {
	Name string
	A    *DeepCircularA // Back to A
}

func TestTypeLookup_DeepCircularReferences(t *testing.T) {
	g := Graphy{}

	// Test deeper circular reference chain
	typeA := reflect.TypeOf((*DeepCircularA)(nil)).Elem()
	lookupA := g.typeLookup(typeA)

	assert.NotNil(t, lookupA)
	assert.Equal(t, "DeepCircularA", lookupA.name)

	// Verify the entire chain was processed without stack overflow
	assert.Contains(t, lookupA.fields, "B")
}
