package quickgraph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Types for interface test - defined outside function
type testAnimal interface {
	IsAnimal()
}

type testDog struct {
	Name  string `json:"name" graphy:"description=The dog's name"`
	Breed string `json:"breed" graphy:"description=Dog breed"`
}

func (testDog) IsAnimal() {}

func (testDog) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "A loyal canine companion",
	}
}

type testCat struct {
	Name  string `json:"name" graphy:"description=The cat's name"`
	Color string `json:"color" graphy:"description=Fur color"`
}

func (testCat) IsAnimal() {}

func (testCat) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "An independent feline friend",
	}
}

// Test that descriptions work with complex type hierarchies
func TestSchemaDescriptionWithInterfaces(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Register types
	g.RegisterTypes(ctx, testDog{}, testCat{})

	// Register a query that returns an interface
	desc := "Get all animals in the shelter"
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:        "getAnimals",
		Function:    func() []testAnimal { return []testAnimal{testDog{}, testCat{}} },
		Description: &desc,
		Mode:        ModeQuery,
	})

	schema := g.SchemaDefinition(ctx)

	// Check interface and implementation descriptions
	assert.Contains(t, schema, `"""A loyal canine companion"""`)
	assert.Contains(t, schema, `"""An independent feline friend"""`)
	assert.Contains(t, schema, `"""Get all animals in the shelter"""`)
	assert.Contains(t, schema, `"""The dog's name"""`)
	assert.Contains(t, schema, `"""Dog breed"""`)
	assert.Contains(t, schema, `"""The cat's name"""`)
	assert.Contains(t, schema, `"""Fur color"""`)
}

// Types for union test
type testSuccessResult struct {
	Message string `json:"message" graphy:"description=Success message"`
}

func (testSuccessResult) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Represents a successful operation",
	}
}

type testErrorResult struct {
	Code    int    `json:"code" graphy:"description=Error code"`
	Message string `json:"message" graphy:"description=Error message"`
}

func (testErrorResult) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Represents a failed operation",
	}
}

type testOperationResultUnion struct {
	Success *testSuccessResult
	Error   *testErrorResult
}

func (testOperationResultUnion) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Result of an operation that can succeed or fail",
	}
}

// Test that descriptions work with union types
func TestSchemaDescriptionWithUnions(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	desc := "Perform a risky operation"
	g.RegisterFunction(ctx, FunctionDefinition{
		Name: "performOperation",
		Function: func() testOperationResultUnion {
			return testOperationResultUnion{Success: &testSuccessResult{}}
		},
		Description: &desc,
		Mode:        ModeQuery,
	})

	schema := g.SchemaDefinition(ctx)

	// Check union and member descriptions
	assert.Contains(t, schema, `"""Result of an operation that can succeed or fail"""`)
	assert.Contains(t, schema, `"""Represents a successful operation"""`)
	assert.Contains(t, schema, `"""Represents a failed operation"""`)
	assert.Contains(t, schema, `"""Perform a risky operation"""`)
}

// Types for nested test
type testAddress struct {
	Street  string `json:"street" graphy:"description=Street address"`
	City    string `json:"city" graphy:"description=City name"`
	Country string `json:"country" graphy:"description=Country code"`
}

func (testAddress) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Physical address information",
	}
}

type testCompany struct {
	Name    string      `json:"name" graphy:"description=Company name"`
	Address testAddress `json:"address" graphy:"description=Company headquarters"`
}

func (testCompany) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "A business entity",
	}
}

// Test that descriptions work with nested types
func TestSchemaDescriptionWithNestedTypes(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	desc := "Get company information"
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:        "getCompany",
		Function:    func() testCompany { return testCompany{} },
		Description: &desc,
		Mode:        ModeQuery,
	})

	schema := g.SchemaDefinition(ctx)

	// Check nested type descriptions
	assert.Contains(t, schema, `"""A business entity"""`)
	assert.Contains(t, schema, `"""Physical address information"""`)
	assert.Contains(t, schema, `"""Company headquarters"""`)
	assert.Contains(t, schema, `"""Street address"""`)
}

// Types for escaping test
type testSpecialType struct {
	Field1 string `json:"field1" graphy:"description=Contains \"quotes\" and backslash \\"`
	Field2 string `json:"field2" graphy:"description=Line 1\nLine 2\nLine 3"`
	Field3 string `json:"field3" graphy:"description=Has triple quotes \"\"\" inside"`
}

func (testSpecialType) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Type with \"special\" characters\nand multiple lines",
	}
}

// Test escaping in descriptions
func TestSchemaDescriptionEscaping(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	g.RegisterQuery(ctx, "getSpecial", func() testSpecialType { return testSpecialType{} })

	schema := g.SchemaDefinition(ctx)

	// Check that special characters are properly handled
	assert.Contains(t, schema, `"""
Type with "special" characters
and multiple lines
"""`)
	assert.Contains(t, schema, `"""Contains "quotes" and backslash \"""`)
	assert.Contains(t, schema, `"""
	Line 1
	Line 2
	Line 3
	"""`)
}

// Types for enum test
type testPriority string

func (testPriority) EnumValues() []EnumValue {
	return []EnumValue{
		{
			Name:        "LOW",
			Description: "Low priority task",
		},
		{
			Name:        "MEDIUM",
			Description: "Medium priority task",
		},
		{
			Name:              "HIGH",
			Description:       "High priority task - use sparingly",
			IsDeprecated:      true,
			DeprecationReason: "Use URGENT for high priority items",
		},
		{
			Name:        "URGENT",
			Description: "Urgent task requiring immediate attention",
		},
	}
}

func (testPriority) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Task priority levels",
	}
}

// Test enum value descriptions with all fields
func TestEnumValueDescriptionsComplete(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	g.RegisterQuery(ctx, "tasks", func() []testPriority { return nil })

	schema := g.SchemaDefinition(ctx)

	// Check enum descriptions
	assert.Contains(t, schema, `"""Task priority levels"""
enum testPriority {`)
	assert.Contains(t, schema, `"""Low priority task"""
	LOW`)
	assert.Contains(t, schema, `"""High priority task - use sparingly"""
	HIGH @deprecated(reason: "Use URGENT for high priority items")`)
	assert.Contains(t, schema, `"""Urgent task requiring immediate attention"""
	URGENT`)
}

// Test that empty descriptions are not included
func TestEmptyDescriptionsOmitted(t *testing.T) {
	type MinimalType struct {
		Field1 string `json:"field1"`
		Field2 string `json:"field2" graphy:"description="`
	}

	g := Graphy{}
	ctx := context.Background()

	// Register with empty description
	emptyDesc := ""
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:        "minimal",
		Function:    func() MinimalType { return MinimalType{} },
		Description: &emptyDesc,
		Mode:        ModeQuery,
	})

	schema := g.SchemaDefinition(ctx)

	// Should not have empty description blocks
	assert.NotContains(t, schema, `""""""`)

	// Type should exist without description
	assert.Contains(t, schema, "type MinimalType {")
	assert.Contains(t, schema, "field1: String!")

	// Function should exist without description
	assert.Contains(t, schema, "minimal: MinimalType")
	assert.NotContains(t, schema, `"""
	minimal: MinimalType`)
}

// Types for input test
type testCreateUserInput struct {
	Name  string `json:"name" graphy:"description=User's full name"`
	Email string `json:"email" graphy:"description=User's email address"`
}

func (testCreateUserInput) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Input for creating a new user",
	}
}

// Test input type descriptions
func TestInputTypeDescriptions(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	desc := "Create a new user account"
	g.RegisterFunction(ctx, FunctionDefinition{
		Name: "createUser",
		Function: func(input testCreateUserInput) string {
			return "created"
		},
		Description:    &desc,
		Mode:           ModeMutation,
		ParameterNames: []string{"input"},
	})

	schema := g.SchemaDefinition(ctx)

	// Check input type descriptions
	assert.Contains(t, schema, `"""Input for creating a new user"""
input testCreateUserInput`)
	assert.Contains(t, schema, `"""User's full name"""
	name: String!`)
	assert.Contains(t, schema, `"""User's email address"""
	email: String!`)
	assert.Contains(t, schema, `"""Create a new user account"""
	createUser(input: testCreateUserInput!): String!`)
}

// Types for explicit registration test
type testExplicitType struct {
	Field string `json:"field" graphy:"description=A field with description"`
}

func (testExplicitType) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "An explicitly registered type",
	}
}

// Test that descriptions work correctly when types are registered explicitly
func TestExplicitlyRegisteredTypeDescriptions(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Explicitly register the type
	g.RegisterTypes(ctx, testExplicitType{})

	// Register a query that doesn't directly return this type
	g.RegisterQuery(ctx, "dummy", func() string { return "dummy" })

	schema := g.SchemaDefinition(ctx)

	// The explicitly registered type should appear with its description
	assert.Contains(t, schema, `"""An explicitly registered type"""
type testExplicitType`)
	assert.Contains(t, schema, `"""A field with description"""
	field: String!`)
}

// Types for subscription test
type testEvent struct {
	Type    string `json:"type" graphy:"description=Event type"`
	Payload string `json:"payload" graphy:"description=Event payload"`
}

func (testEvent) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "A system event",
	}
}

// Test subscription descriptions
func TestSubscriptionDescriptions(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	desc := "Subscribe to system events"
	g.RegisterFunction(ctx, FunctionDefinition{
		Name: "systemEvents",
		Function: func(ctx context.Context, filter string) (<-chan testEvent, error) {
			ch := make(chan testEvent)
			close(ch)
			return ch, nil
		},
		Description:    &desc,
		Mode:           ModeSubscription,
		ParameterNames: []string{"filter"},
	})

	schema := g.SchemaDefinition(ctx)

	// Check subscription description
	assert.Contains(t, schema, `"""Subscribe to system events"""
	systemEvents(filter: String!): testEvent!`)
	assert.Contains(t, schema, `"""A system event"""`)
	assert.Contains(t, schema, `"""Event type"""`)
	assert.Contains(t, schema, `"""Event payload"""`)
}
