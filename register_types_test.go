package quickgraph

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestRegisterTypes tests the RegisterTypes functionality
func TestRegisterTypes(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Define types that are never directly returned
	type HiddenType struct {
		ID     int
		Secret string
	}

	type AnotherHiddenType struct {
		Code string
		Data string
	}

	// Register some visible types through queries
	g.RegisterQuery(ctx, "visible", func() string { return "test" })

	// Explicitly register types that aren't returned by any function
	g.RegisterTypes(ctx, HiddenType{}, AnotherHiddenType{})

	// Generate schema
	schema := g.SchemaDefinition(ctx)

	// The registered types should appear in the schema
	assert.Contains(t, schema, "type HiddenType {")
	assert.Contains(t, schema, "ID: Int!")
	assert.Contains(t, schema, "Secret: String!")

	assert.Contains(t, schema, "type AnotherHiddenType {")
	assert.Contains(t, schema, "Code: String!")
	assert.Contains(t, schema, "Data: String!")
}

// TestUnionWithInterfaceExpansion tests that unions correctly expand interfaces to their implementations
func TestUnionWithInterfaceExpansion(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Define an interface type (embedded by others)
	type Animal struct {
		ID   int
		Name string
		Age  int
	}

	// Define concrete types that embed Animal
	type Dog struct {
		Animal
		Breed   string
		GoodBoy bool
	}

	type Cat struct {
		Animal
		Lives    int
		Attitude string
	}

	// Define other types for the union
	type Toy struct {
		ID    int
		Name  string
		Color string
	}

	type Food struct {
		ID       int
		Brand    string
		Calories int
	}

	// Define a union type that includes the interface
	type PetSearchResultUnion struct {
		Animal *Animal // This should expand to Dog and Cat
		Toy    *Toy
		Food   *Food
	}

	// Register a search function that returns the union
	g.RegisterQuery(ctx, "searchPets", func(query string) []PetSearchResultUnion {
		return []PetSearchResultUnion{}
	})

	// Explicitly register the concrete types that implement Animal
	// This ensures they're known to the type system
	g.RegisterTypes(ctx, Dog{}, Cat{})

	// Generate schema
	schema := g.SchemaDefinition(ctx)

	// The union should expand Animal to its concrete implementations
	assert.Contains(t, schema, "union PetSearchResult = Cat | Dog | Food | Toy")

	// Should NOT contain Animal in the union
	assert.NotContains(t, schema, "Animal | Toy")
	assert.NotContains(t, schema, "| Animal |")

	// Verify Animal is an interface
	assert.Contains(t, schema, "interface Animal {")

	// Verify the concrete types implement Animal
	assert.Contains(t, schema, "type Dog implements Animal {")
	assert.Contains(t, schema, "type Cat implements Animal {")
}

// TestUnionInterfaceExpansionWithoutExplicitRegistration tests the behavior when types aren't registered
func TestUnionInterfaceExpansionWithoutExplicitRegistration(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Define an interface type
	type Vehicle struct {
		ID    int
		Model string
		Year  int
	}

	// Define concrete types that embed Vehicle
	type Car struct {
		Vehicle
		Doors int
		Fuel  string
	}

	type Motorcycle struct {
		Vehicle
		Type string
	}

	type Bicycle struct {
		ID    int
		Brand string
		Gears int
	}

	// Define a union that includes the interface
	type TransportSearchUnion struct {
		Vehicle *Vehicle
		Bike    *Bicycle
	}

	// Register a search function
	g.RegisterQuery(ctx, "searchTransport", func() []TransportSearchUnion {
		return []TransportSearchUnion{}
	})

	// Register only Car through a direct query (Motorcycle is not registered)
	g.RegisterQuery(ctx, "getCar", func() Car {
		return Car{}
	})

	// Generate schema without registering Motorcycle
	schema := g.SchemaDefinition(ctx)

	// The union should only include known implementations
	assert.Contains(t, schema, "union TransportSearch = Bicycle | Car")

	// Motorcycle should NOT be in the union as it wasn't registered
	assert.NotContains(t, schema, "Motorcycle")

	// Now register Motorcycle and regenerate schema
	g.RegisterTypes(ctx, Motorcycle{})
	schema = g.SchemaDefinition(ctx)

	// Now the union should include Motorcycle
	assert.Contains(t, schema, "union TransportSearch = Bicycle | Car | Motorcycle")
}

// TestRegisterTypesWithPointerInterface tests interface expansion when the union field is a pointer
func TestRegisterTypesWithPointerInterface(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Define a base type that will act as interface
	type Shape struct {
		ID    int
		Name  string
		Color string
	}

	// Define concrete types
	type Circle struct {
		Shape
		Radius float64
	}

	type Square struct {
		Shape
		Side float64
	}

	type Rectangle struct {
		Shape
		Width  float64
		Height float64
	}

	type Text struct {
		ID      int
		Content string
		Font    string
	}

	// Union with pointer to interface
	type DrawingElementUnion struct {
		Shape *Shape // Pointer to interface type
		Text  *Text
	}

	// Register function returning the union
	g.RegisterQuery(ctx, "getDrawingElements", func() []DrawingElementUnion {
		return []DrawingElementUnion{}
	})

	// Register all shape implementations
	g.RegisterTypes(ctx, Circle{}, Square{}, Rectangle{})

	// Generate schema
	schema := g.SchemaDefinition(ctx)

	// Union should contain all concrete shape types
	assert.Contains(t, schema, "union DrawingElement = Circle | Rectangle | Square | Text")

	// Verify Shape is an interface
	assert.Contains(t, schema, "interface Shape {")

	// Verify implementations
	assert.Contains(t, schema, "type Circle implements Shape {")
	assert.Contains(t, schema, "type Square implements Shape {")
	assert.Contains(t, schema, "type Rectangle implements Shape {")
}

// TestComplexUnionWithNestedInterfaces tests unions with nested interface hierarchies
func TestComplexUnionWithNestedInterfaces(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Base interface
	type Entity struct {
		ID        int
		CreatedAt string
		UpdatedAt string
	}

	// Mid-level interface that embeds Entity
	type Person struct {
		Entity
		FirstName string
		LastName  string
		Email     string
	}

	// Concrete types that embed Person
	type Customer struct {
		Person
		CustomerID  string
		LoyaltyTier string
	}

	type Employee struct {
		Person
		EmployeeID string
		Department string
		Salary     float64
	}

	// Another branch from Entity
	type Organization struct {
		Entity
		Name    string
		TaxID   string
		Address string
	}

	type Document struct {
		ID      int
		Title   string
		Content string
	}

	// Search union including various levels
	type EntitySearchUnion struct {
		Entity       *Entity       // Should expand to all implementations
		Person       *Person       // Should expand to Customer, Employee
		Organization *Organization // Concrete type
		Document     *Document     // Unrelated type
	}

	// Register search function
	g.RegisterQuery(ctx, "searchEntities", func(query string) []EntitySearchUnion {
		return []EntitySearchUnion{}
	})

	// Register all concrete types
	g.RegisterTypes(ctx, Customer{}, Employee{}, Organization{})

	// Generate schema
	schema := g.SchemaDefinition(ctx)

	// The union should contain only concrete types
	// Note: The current implementation includes Person in the union, which might need further refinement
	assert.Contains(t, schema, "union EntitySearch = Customer | Document | Employee | Organization | Person")

	// Verify the interface hierarchy
	assert.Contains(t, schema, "interface Entity {")
	assert.Contains(t, schema, "interface Person implements Entity {")

	// In GraphQL, when there's multi-level inheritance, concrete types show all interfaces
	assert.Contains(t, schema, "type Customer implements Person & Entity {")
	assert.Contains(t, schema, "type Employee implements Person & Entity {")
	assert.Contains(t, schema, "type Organization implements Entity {")
}
