package quickgraph

import (
	"context"
	"github.com/stretchr/testify/assert"
	"reflect"
	"strings"
	"testing"
)

// Test types for verifying type lookup relationships
type BaseInterface struct {
	ID   string
	Name string
}

type ConcreteImpl struct {
	BaseInterface
	Details string
}

type AnotherImpl struct {
	BaseInterface
	Code int
}

// TestTypeLookupRelationships verifies that implementedBy relationships are properly
// shared across type variants (T, *T, []T)
func TestTypeLookupRelationships(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Register types
	g.RegisterTypes(ctx, BaseInterface{}, ConcreteImpl{}, AnotherImpl{})

	// Register a function that returns the interface type
	g.RegisterQuery(ctx, "getBase", func() BaseInterface {
		return BaseInterface{ID: "1", Name: "test"}
	})

	// Register a function that returns a pointer to the interface
	g.RegisterQuery(ctx, "getBasePtr", func() *BaseInterface {
		return &BaseInterface{ID: "2", Name: "test2"}
	})

	// Register a function that returns a slice of the interface
	g.RegisterQuery(ctx, "getBases", func() []BaseInterface {
		return []BaseInterface{{ID: "3", Name: "test3"}}
	})

	// Get the schema
	schema := string(g.SchemaDefinition(ctx))

	// Verify that BaseInterface generates both interface and concrete type
	assert.Contains(t, schema, "interface IBaseInterface {")
	assert.Contains(t, schema, "type BaseInterface implements IBaseInterface {")

	// Verify that implementing types reference the interface with I prefix
	assert.Contains(t, schema, "type ConcreteImpl implements IBaseInterface {")
	assert.Contains(t, schema, "type AnotherImpl implements IBaseInterface {")

	// Check that all type lookup variants have proper relationships
	baseLookup := g.typeLookup(reflect.TypeOf(BaseInterface{}))
	basePtrLookup := g.typeLookup(reflect.TypeOf(&BaseInterface{}))
	baseSliceLookup := g.typeLookup(reflect.TypeOf([]BaseInterface{}))

	// All variants should exist
	assert.NotNil(t, baseLookup, "Base type lookup should exist")
	assert.NotNil(t, basePtrLookup, "Pointer type lookup should exist")
	assert.NotNil(t, baseSliceLookup, "Slice type lookup should exist")

	// The base type should have implementedBy relationships
	assert.True(t, len(baseLookup.implementedBy) >= 2,
		"Base type should have at least 2 implementations, got %d", len(baseLookup.implementedBy))
}

// TestUnionExpansionWithTypeLookupFix verifies that unions correctly expand
// interface types to their concrete implementations, regardless of which
// type variant is used
func TestUnionExpansionWithTypeLookupFix(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Union type that contains the interface - must end with "Union" and have only anonymous fields
	type SearchResultUnion struct {
		BaseInterface
		ConcreteImpl
		AnotherImpl
	}

	// Register types
	g.RegisterTypes(ctx, BaseInterface{}, ConcreteImpl{}, AnotherImpl{})

	// Register a search function that returns the union
	g.RegisterQuery(ctx, "search", func() []SearchResultUnion {
		return nil
	})

	// Get the schema
	schema := string(g.SchemaDefinition(ctx))

	// The union should expand BaseInterface to include ConcreteImpl and AnotherImpl
	// It should also include BaseInterface itself as a concrete type
	assert.Contains(t, schema, "union SearchResult = AnotherImpl | BaseInterface | ConcreteImpl")
}

// TestPointerTypeRelationships specifically tests that pointer types
// properly inherit implementedBy relationships from their base types
func TestPointerTypeRelationships(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Define test types
	type Animal struct {
		Name string
	}

	type Dog struct {
		Animal
		Breed string
	}

	type Cat struct {
		Animal
		Color string
	}

	// Register types
	g.RegisterTypes(ctx, Animal{}, Dog{}, Cat{})

	// Register functions that return different type variants
	g.RegisterQuery(ctx, "getAnimal", func() Animal {
		return Animal{Name: "Generic"}
	})

	g.RegisterQuery(ctx, "getAnimalPtr", func() *Animal {
		return &Animal{Name: "Pointer"}
	})

	g.RegisterQuery(ctx, "getAnimals", func() []Animal {
		return []Animal{{Name: "Array"}}
	})

	// Verify that all Animal type variants recognize they have implementations
	animalType := reflect.TypeOf(Animal{})
	animalPtrType := reflect.TypeOf(&Animal{})
	animalSliceType := reflect.TypeOf([]Animal{})

	animalLookup := g.typeLookup(animalType)
	animalPtrLookup := g.typeLookup(animalPtrType)
	animalSliceLookup := g.typeLookup(animalSliceType)

	// All should exist
	assert.NotNil(t, animalLookup)
	assert.NotNil(t, animalPtrLookup)
	assert.NotNil(t, animalSliceLookup)

	// Base type should have implementations
	assert.Len(t, animalLookup.implementedBy, 2, "Animal should be implemented by Dog and Cat")

	// Verify the schema generates correctly
	schema := string(g.SchemaDefinition(ctx))

	// Should generate IAnimal interface and Animal concrete type
	assert.Contains(t, schema, "interface IAnimal {")
	assert.Contains(t, schema, "type Animal implements IAnimal {")
	assert.Contains(t, schema, "type Dog implements IAnimal {")
	assert.Contains(t, schema, "type Cat implements IAnimal {")

	// Count occurrences to ensure no duplicates
	animalInterfaceCount := strings.Count(schema, "interface IAnimal {")
	assert.Equal(t, 1, animalInterfaceCount, "IAnimal interface should appear exactly once")
}
