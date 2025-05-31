package quickgraph

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helpers and mock data
func createMockTypeLookup(rootType reflect.Type, fundamental bool) *typeLookup {
	return &typeLookup{
		typ:                 rootType,
		rootType:            rootType,
		fundamental:         fundamental,
		fields:              make(map[string]fieldLookup),
		fieldsLowercase:     make(map[string]fieldLookup),
		implements:          make(map[string]*typeLookup),
		implementsLowercase: make(map[string]*typeLookup),
		union:               make(map[string]*typeLookup),
		unionLowercase:      make(map[string]*typeLookup),
	}
}

func createMockGraphy() *Graphy {
	g := &Graphy{}
	g.ensureInitialized()
	g.schemaBuffer = &schemaTypes{
		outputTypeNameLookup: make(typeNameMapping),
		inputTypeNameLookup:  make(typeNameMapping),
		enumTypeNameLookup:   make(typeNameMapping),
	}
	return g
}

func createMockSchema() *__Schema {
	return &__Schema{
		typeLookupByName: make(map[string]*__Type),
	}
}

// Test resolveIntrospectionTypeName
func TestResolveIntrospectionTypeName(t *testing.T) {
	g := createMockGraphy()

	t.Run("enum type", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf((*StringEnumValues)(nil)).Elem(), false)
		tl.rootType = stringEnumValuesType
		g.schemaBuffer.enumTypeNameLookup[tl] = "MyEnum"

		result := g.resolveIntrospectionTypeName(tl, TypeOutput)
		assert.Equal(t, "MyEnum", result)
	})

	t.Run("fundamental type with output lookup", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf(""), true)
		g.schemaBuffer.outputTypeNameLookup[tl] = "String"

		result := g.resolveIntrospectionTypeName(tl, TypeOutput)
		assert.Equal(t, "String", result)
	})

	t.Run("fundamental type without output lookup", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf(""), true)
		// No entry in outputTypeNameLookup, should use introspectionScalarName

		result := g.resolveIntrospectionTypeName(tl, TypeOutput)
		assert.Equal(t, "String", result)
	})

	t.Run("output type", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		g.schemaBuffer.outputTypeNameLookup[tl] = "MyObject"

		result := g.resolveIntrospectionTypeName(tl, TypeOutput)
		assert.Equal(t, "MyObject", result)
	})

	t.Run("input type", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		g.schemaBuffer.inputTypeNameLookup[tl] = "MyInput"

		result := g.resolveIntrospectionTypeName(tl, TypeInput)
		assert.Equal(t, "MyInput", result)
	})
}

// Test handleInterfaceConcreteTypeSplit
func TestHandleInterfaceConcreteTypeSplit(t *testing.T) {
	g := createMockGraphy()
	is := createMockSchema()

	t.Run("creates interface and returns concrete type", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		tl.implementedBy = []*typeLookup{
			createMockTypeLookup(reflect.TypeOf(struct{}{}), false),
		}

		result := g.handleInterfaceConcreteTypeSplit(is, tl, TypeOutput, "Employee")

		// Should create interface with "I" prefix
		interfaceType, exists := is.typeLookupByName["IEmployee"]
		assert.True(t, exists)
		assert.Equal(t, IntrospectionKindInterface, interfaceType.Kind)
		assert.Equal(t, "IEmployee", *interfaceType.Name)

		// Should return concrete type
		assert.Equal(t, "Employee", *result.Name)
	})

	t.Run("returns existing concrete type", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		existing := &__Type{Name: stringPtr("Employee")}
		is.typeLookupByName["Employee"] = existing

		result := g.handleInterfaceConcreteTypeSplit(is, tl, TypeOutput, "Employee")

		assert.Equal(t, existing, result)
	})
}

// Test createEnumIntrospectionType
func TestCreateEnumIntrospectionType(t *testing.T) {
	g := createMockGraphy()

	// Use the existing enumWithDescription type from introspection_test.go
	enumType := reflect.TypeOf(enumWithDescription(""))
	tl := createMockTypeLookup(enumType, false)
	result := &__Type{}

	g.createEnumIntrospectionType(tl, result)

	assert.Equal(t, IntrospectionKindEnum, result.Kind)
	assert.Len(t, result.enumValuesRaw, 3)

	// The enumWithDescription should have ENUM1, ENUM-HALF, ENUM2
	enumNames := []string{}
	for _, ev := range result.enumValuesRaw {
		enumNames = append(enumNames, ev.Name)
	}
	assert.Contains(t, enumNames, "ENUM1")
	assert.Contains(t, enumNames, "ENUM-HALF")
	assert.Contains(t, enumNames, "ENUM2")
}

// Test createUnionIntrospectionType
func TestCreateUnionIntrospectionType(t *testing.T) {
	t.Run("sets union kind", func(t *testing.T) {
		g := createMockGraphy()
		g.typeLookups = make(map[reflect.Type]*typeLookup)
		is := createMockSchema()

		// Create a simple union without complex recursive calls
		unionTl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		unionTl.union = map[string]*typeLookup{} // Empty union for simplicity

		result := &__Type{}
		g.createUnionIntrospectionType(is, unionTl, TypeOutput, result)

		assert.Equal(t, IntrospectionKindUnion, result.Kind)
	})
}

// Test createObjectIntrospectionType
func TestCreateObjectIntrospectionType(t *testing.T) {
	g := createMockGraphy()
	is := createMockSchema()

	t.Run("simple object type", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		result := &__Type{Name: stringPtr("TestObject")}

		g.createObjectIntrospectionType(is, tl, TypeOutput, result)

		assert.Equal(t, IntrospectionKindObject, result.Kind)
	})

	t.Run("object with embedded interfaces", func(t *testing.T) {
		// Create embedded interface
		embeddedTl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		embeddedTl.name = "EmbeddedInterface"
		g.schemaBuffer.outputTypeNameLookup[embeddedTl] = "EmbeddedInterface"

		// Create object that implements the interface
		tl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		tl.implements = map[string]*typeLookup{
			"EmbeddedInterface": embeddedTl,
		}

		result := &__Type{Name: stringPtr("TestObject")}

		g.createObjectIntrospectionType(is, tl, TypeOutput, result)

		assert.Equal(t, IntrospectionKindObject, result.Kind)
		// Should create the missing interface and add relationship
		assert.Len(t, result.Interfaces, 1)
	})
}

// Test resolveEmbeddedInterfaceName
func TestResolveEmbeddedInterfaceName(t *testing.T) {
	g := createMockGraphy()

	t.Run("enum interface", func(t *testing.T) {
		tl := createMockTypeLookup(stringEnumValuesType, false)
		g.schemaBuffer.enumTypeNameLookup[tl] = "MyEnum"

		result := g.resolveEmbeddedInterfaceName(tl, TypeOutput)
		assert.Equal(t, "MyEnum", result)
	})

	t.Run("output interface with implementations", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		tl.implementedBy = []*typeLookup{createMockTypeLookup(reflect.TypeOf(struct{}{}), false)}
		g.schemaBuffer.outputTypeNameLookup[tl] = "MyInterface"

		result := g.resolveEmbeddedInterfaceName(tl, TypeOutput)
		assert.Equal(t, "IMyInterface", result) // Should add "I" prefix
	})

	t.Run("interface only type", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		tl.implementedBy = []*typeLookup{createMockTypeLookup(reflect.TypeOf(struct{}{}), false)}
		tl.interfaceOnly = true
		g.schemaBuffer.outputTypeNameLookup[tl] = "MyInterface"

		result := g.resolveEmbeddedInterfaceName(tl, TypeOutput)
		assert.Equal(t, "MyInterface", result) // Should NOT add "I" prefix
	})
}

// Test getIntrospectionBaseType main orchestration
func TestGetIntrospectionBaseType_Orchestration(t *testing.T) {
	g := createMockGraphy()
	is := createMockSchema()

	t.Run("delegates to interface/concrete split", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		tl.implementedBy = []*typeLookup{createMockTypeLookup(reflect.TypeOf(struct{}{}), false)}
		g.schemaBuffer.outputTypeNameLookup[tl] = "Employee"

		result := g.getIntrospectionBaseType(is, tl, TypeOutput)

		// Should have created interface and returned concrete type
		_, interfaceExists := is.typeLookupByName["IEmployee"]
		assert.True(t, interfaceExists)
		assert.Equal(t, "Employee", *result.Name)
	})

	t.Run("returns cached type", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		g.schemaBuffer.outputTypeNameLookup[tl] = "CachedType"

		// Pre-populate cache
		cached := &__Type{Name: stringPtr("CachedType")}
		is.typeLookupByName["CachedType"] = cached

		result := g.getIntrospectionBaseType(is, tl, TypeOutput)

		assert.Equal(t, cached, result)
	})

	t.Run("creates union type", func(t *testing.T) {
		memberType := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		memberType.name = "Member"

		tl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		tl.union = map[string]*typeLookup{"Member": memberType}
		g.schemaBuffer.outputTypeNameLookup[tl] = "SearchResult"

		result := g.getIntrospectionBaseType(is, tl, TypeOutput)

		assert.Equal(t, IntrospectionKindUnion, result.Kind)
		assert.Equal(t, "SearchResult", *result.Name)
	})

	t.Run("creates scalar type", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf(""), true)
		g.schemaBuffer.outputTypeNameLookup[tl] = "String"

		result := g.getIntrospectionBaseType(is, tl, TypeOutput)

		assert.Equal(t, IntrospectionKindScalar, result.Kind)
		assert.Equal(t, "String", *result.Name)
	})

	t.Run("creates input object type", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		g.schemaBuffer.inputTypeNameLookup[tl] = "MyInput"

		result := g.getIntrospectionBaseType(is, tl, TypeInput)

		assert.Equal(t, IntrospectionKindInputObject, result.Kind)
		assert.Equal(t, "MyInput", *result.Name)
	})

	t.Run("creates object type", func(t *testing.T) {
		tl := createMockTypeLookup(reflect.TypeOf(struct{}{}), false)
		g.schemaBuffer.outputTypeNameLookup[tl] = "MyObject"

		result := g.getIntrospectionBaseType(is, tl, TypeOutput)

		assert.Equal(t, IntrospectionKindObject, result.Kind)
		assert.Equal(t, "MyObject", *result.Name)
	})
}

// Integration test to ensure refactored functions work together
func TestIntrospectionRefactoring_Integration(t *testing.T) {
	// This test ensures our refactored functions still work with the existing integration tests
	g := Graphy{}
	ctx := context.Background()

	// Use the same setup as the existing interface test
	g.RegisterFunction(ctx, FunctionDefinition{
		Name: "sample",
		Function: func(input string) any {
			return Droid{
				Character: Character{
					Name: input,
				},
				PrimaryFunction: "droiding",
			}
		},
		Mode:              ModeQuery,
		ParameterNames:    []string{"input"},
		ReturnAnyOverride: []any{Character{}},
	})
	g.RegisterTypes(ctx, Droid{}, Character{})
	g.EnableIntrospection(ctx)

	// Execute a simple introspection query
	query := `{
		__schema {
			types {
				name
				kind
			}
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	require.NoError(t, err)
	assert.Contains(t, result, "Character")
	assert.Contains(t, result, "Droid")
	assert.Contains(t, result, "ICharacter")
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
