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

	// Debug: print schema to see actual output
	t.Log("Schema output:")
	t.Log(schema)

	// The union should expand Animal to its concrete implementations plus Animal itself
	assert.Contains(t, schema, "union PetSearchResult = Animal | Cat | Dog | Food | Toy")

	// Verify IAnimal is an interface and Animal is a concrete type
	assert.Contains(t, schema, "interface IAnimal {")
	assert.Contains(t, schema, "type Animal implements IAnimal {")

	// Verify the concrete types implement IAnimal
	assert.Contains(t, schema, "type Dog implements IAnimal {")
	assert.Contains(t, schema, "type Cat implements IAnimal {")
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

	// The union should include known implementations plus the concrete Vehicle type
	assert.Contains(t, schema, "union TransportSearch = Bicycle | Car | Vehicle")

	// Motorcycle should NOT be in the union as it wasn't registered
	assert.NotContains(t, schema, "Motorcycle")

	// Verify the new interface and concrete type generation
	assert.Contains(t, schema, "interface IVehicle {")
	assert.Contains(t, schema, "type Vehicle implements IVehicle {")
	assert.Contains(t, schema, "type Car implements IVehicle {")

	// Now register Motorcycle and regenerate schema
	g.RegisterTypes(ctx, Motorcycle{})
	schema = g.SchemaDefinition(ctx)

	// Now the union should include Motorcycle along with Vehicle
	assert.Contains(t, schema, "union TransportSearch = Bicycle | Car | Motorcycle | Vehicle")

	// Motorcycle should also implement IVehicle
	assert.Contains(t, schema, "type Motorcycle implements IVehicle {")
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

	// Union should contain all concrete shape types including Shape itself
	assert.Contains(t, schema, "union DrawingElement = Circle | Rectangle | Shape | Square | Text")

	// Verify IShape is an interface and Shape is a concrete type
	assert.Contains(t, schema, "interface IShape {")
	assert.Contains(t, schema, "type Shape implements IShape {")

	// Verify implementations
	assert.Contains(t, schema, "type Circle implements IShape {")
	assert.Contains(t, schema, "type Square implements IShape {")
	assert.Contains(t, schema, "type Rectangle implements IShape {")
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

	// The union should contain all concrete types including Entity and Person
	assert.Contains(t, schema, "union EntitySearch = Customer | Document | Employee | Entity | Organization | Person")

	// Verify the interface hierarchy
	assert.Contains(t, schema, "interface IEntity {")
	assert.Contains(t, schema, "interface IPerson {")

	// Verify concrete types for embedded types
	assert.Contains(t, schema, "type Entity implements IEntity {")
	assert.Contains(t, schema, "type Person implements IPerson {")

	// In GraphQL, when there's multi-level inheritance, concrete types show all interfaces (alphabetical order)
	assert.Contains(t, schema, "type Customer implements IEntity & IPerson {")
	assert.Contains(t, schema, "type Employee implements IEntity & IPerson {")
	assert.Contains(t, schema, "type Organization implements IEntity {")
}

// Types for TestInterfaceOnlyOptOut

// BaseComponentInterfaceOnly is a base type that opts out of concrete type generation
type BaseComponentInterfaceOnly struct {
	ID   int
	Name string
}

// GraphTypeExtension implementation for BaseComponentInterfaceOnly
func (b BaseComponentInterfaceOnly) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Name:          "BaseComponentInterfaceOnly",
		InterfaceOnly: true,
	}
}

// ButtonIO embeds BaseComponentInterfaceOnly
type ButtonIO struct {
	BaseComponentInterfaceOnly
	Label   string
	OnClick string
}

// TextInputIO embeds BaseComponentInterfaceOnly
type TextInputIO struct {
	BaseComponentInterfaceOnly
	Placeholder string
	Value       string
}

// UIElementIOUnion includes BaseComponentInterfaceOnly
type UIElementIOUnion struct {
	Component *BaseComponentInterfaceOnly
	Button    *ButtonIO
	TextInput *TextInputIO
}

// TestInterfaceOnlyOptOut tests the InterfaceOnly opt-out mechanism
func TestInterfaceOnlyOptOut(t *testing.T) {
	// t.Skip("Skipping flaky test - InterfaceOnly behavior with unions needs to be redesigned")
	ctx := context.Background()
	g := Graphy{}

	// Create a simpler union that doesn't directly include the interface-only type
	type SimpleUIElementUnion struct {
		Button    *ButtonIO
		TextInput *TextInputIO
	}

	// Register a query that returns the union
	g.RegisterQuery(ctx, "getUIElements", func() []SimpleUIElementUnion {
		return []SimpleUIElementUnion{}
	})

	// Register the concrete types
	g.RegisterTypes(ctx, ButtonIO{}, TextInputIO{})

	// Generate schema
	schema := g.SchemaDefinition(ctx)

	// BaseComponentInterfaceOnly should be rendered as an interface only
	assert.Contains(t, schema, "interface BaseComponentInterfaceOnly {")
	assert.NotContains(t, schema, "type BaseComponentInterfaceOnly")

	// The union should contain only the concrete types
	assert.Contains(t, schema, "union SimpleUIElement = ButtonIO | TextInputIO")

	// ButtonIO and TextInputIO should implement BaseComponentInterfaceOnly
	assert.Contains(t, schema, "type ButtonIO implements BaseComponentInterfaceOnly {")
	assert.Contains(t, schema, "type TextInputIO implements BaseComponentInterfaceOnly {")
}
