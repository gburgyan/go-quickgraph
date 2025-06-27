package quickgraph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test types for schema generation - Department
type testDepartment struct {
	ID   int    `json:"id" graphy:"description=Unique department identifier"`
	Name string `json:"name" graphy:"description=Department name"`
}

func (testDepartment) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "An organizational department within the company",
	}
}

// Test types for schema generation - EmployeeStatus
type testEmployeeStatus string

func (testEmployeeStatus) EnumValues() []EnumValue {
	return []EnumValue{
		{Name: "ACTIVE", Description: "Employee is currently working"},
		{Name: "ON_LEAVE", Description: "Employee is on temporary leave"},
		{Name: "TERMINATED", Description: "Employee no longer works here", IsDeprecated: true, DeprecationReason: "Use INACTIVE status instead"},
		{Name: "INACTIVE", Description: "Employee is not currently active"},
	}
}

func (testEmployeeStatus) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Current employment status of an employee",
	}
}

// Test types for schema generation - Employee
type testEmployee struct {
	ID         int                `json:"id" graphy:"description=Unique employee identifier"`
	Name       string             `json:"name" graphy:"description=Full name of the employee"`
	Department testDepartment     `json:"department" graphy:"description=The department this employee belongs to"`
	Status     testEmployeeStatus `json:"status" graphy:"description=Current employment status"`
	Email      string             `json:"email" graphy:"description=Work email address,deprecated=Use workEmail instead"`
}

func (testEmployee) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Represents an employee in the company",
	}
}

// Test types for schema generation - CreateEmployeeInput
type testCreateEmployeeInput struct {
	Name         string `json:"name" graphy:"description=Full name of the new employee"`
	DepartmentID int    `json:"departmentId" graphy:"description=ID of the department to assign the employee to"`
	Email        string `json:"email" graphy:"description=Work email address for the employee"`
}

func (testCreateEmployeeInput) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Input for creating a new employee record",
	}
}

// Test that schema generation properly includes descriptions at all levels
func TestSchemaGenerationWithDescriptions(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Register functions with descriptions
	queryDesc := "Retrieves all employees in the system"
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:        "getAllEmployees",
		Function:    func() []testEmployee { return []testEmployee{} },
		Description: &queryDesc,
		Mode:        ModeQuery,
	})

	mutationDesc := "Creates a new employee record in the system"
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:           "createEmployee",
		Function:       func(input testCreateEmployeeInput) testEmployee { return testEmployee{} },
		Description:    &mutationDesc,
		Mode:           ModeMutation,
		ParameterNames: []string{"input"},
	})

	deprecatedDesc := "Legacy employee creation endpoint"
	deprecatedReason := "Use createEmployee instead"
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:             "addEmployee",
		Function:         func(name string) testEmployee { return testEmployee{} },
		Description:      &deprecatedDesc,
		DeprecatedReason: &deprecatedReason,
		Mode:             ModeMutation,
		ParameterNames:   []string{"name"},
	})

	// Generate schema
	schema := g.SchemaDefinition(ctx)

	// Verify Query descriptions
	assert.Contains(t, schema, `type Query {
	"""Retrieves all employees in the system"""
	getAllEmployees: [testEmployee!]!
}`)

	// Verify Mutation descriptions
	assert.Contains(t, schema, `"""Creates a new employee record in the system"""
	createEmployee(input: testCreateEmployeeInput!): testEmployee!`)
	assert.Contains(t, schema, `"""Legacy employee creation endpoint"""
	addEmployee(name: String!): testEmployee! @deprecated(reason: "Use createEmployee instead")`)

	// Verify type descriptions
	assert.Contains(t, schema, `"""Represents an employee in the company"""
type testEmployee {`)
	assert.Contains(t, schema, `"""An organizational department within the company"""
type testDepartment {`)

	// Verify field descriptions
	assert.Contains(t, schema, `"""Unique employee identifier"""
	id: Int!`)
	assert.Contains(t, schema, `"""Full name of the employee"""
	name: String!`)
	assert.Contains(t, schema, `"""The department this employee belongs to"""
	department: testDepartment!`)
	assert.Contains(t, schema, `"""Work email address"""
	email: String! @deprecated(reason: "Use workEmail instead")`)

	// Verify enum descriptions
	assert.Contains(t, schema, `"""Current employment status of an employee"""
enum testEmployeeStatus {`)
	assert.Contains(t, schema, `"""Employee is currently working"""
	ACTIVE`)
	assert.Contains(t, schema, `"""Employee no longer works here"""
	TERMINATED @deprecated(reason: "Use INACTIVE status instead")`)

	// Verify input type descriptions
	assert.Contains(t, schema, `"""Input for creating a new employee record"""
input testCreateEmployeeInput {`)
	assert.Contains(t, schema, `"""Full name of the new employee"""
	name: String!`)
}

// Test that types with empty or missing descriptions are handled properly
func TestSchemaGenerationWithoutDescriptions(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	type SimpleType struct {
		Field1 string `json:"field1"`
		Field2 int    `json:"field2" graphy:"description="` // Empty description
	}

	g.RegisterQuery(ctx, "getSimple", func() SimpleType { return SimpleType{} })

	schema := g.SchemaDefinition(ctx)

	// Type should exist without description block
	assert.Contains(t, schema, "type SimpleType {")
	assert.NotContains(t, schema, `"""
type SimpleType {`)

	// Fields should exist without descriptions
	assert.Contains(t, schema, "field1: String!")
	assert.Contains(t, schema, "field2: Int!")
	assert.NotContains(t, schema, `"""
	field1: String!`)
}

// Test types for complex hierarchy - interfaces
type sgAnimal interface {
	IsAnimal()
}

type sgDog struct {
	Name  string `json:"name" graphy:"description=The dog's name"`
	Breed string `json:"breed" graphy:"description=Dog breed"`
}

func (sgDog) IsAnimal() {}
func (sgDog) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Man's best friend",
	}
}

type sgCat struct {
	Name  string `json:"name" graphy:"description=The cat's name"`
	Lives int    `json:"lives" graphy:"description=Number of lives remaining"`
}

func (sgCat) IsAnimal() {}
func (sgCat) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Independent feline companion",
	}
}

// Test types for complex hierarchy - union
type sgSearchResultUnion struct {
	Dog *sgDog
	Cat *sgCat
}

func (sgSearchResultUnion) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Results from searching for pets",
	}
}

// Test schema generation with complex type hierarchies
func TestSchemaGenerationComplexHierarchy(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Register types
	g.RegisterTypes(ctx, sgDog{}, sgCat{})

	searchDesc := "Search for pets by name"
	g.RegisterFunction(ctx, FunctionDefinition{
		Name: "searchPets",
		Function: func(name string) []sgSearchResultUnion {
			return []sgSearchResultUnion{}
		},
		Description:    &searchDesc,
		Mode:           ModeQuery,
		ParameterNames: []string{"name"},
	})

	schema := g.SchemaDefinition(ctx)

	// Check that types have descriptions (no interface since it's not discovered)
	assert.Contains(t, schema, `"""Man's best friend"""
type sgDog {`)
	assert.Contains(t, schema, `"""Independent feline companion"""
type sgCat {`)

	// Check union has description
	assert.Contains(t, schema, `"""Results from searching for pets"""`)
	assert.Contains(t, schema, `union sgSearchResult = sgCat | sgDog`)

	// Check field descriptions in implementations
	assert.Contains(t, schema, `"""The dog's name"""
	name: String!`)
	assert.Contains(t, schema, `"""Dog breed"""
	breed: String!`)
}

// Test types for special characters
type sgSpecialType struct {
	Field1 string `json:"field1" graphy:"description=Contains \"quotes\" and backslash \\"`
	Field2 string `json:"field2" graphy:"description=Multi\nline\ndescription"`
	Field3 string `json:"field3" graphy:"description=Has 'single quotes' too"`
}

func (sgSpecialType) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Type with \"special\" characters\nin description",
	}
}

// Test that schema generation handles special characters in descriptions
func TestSchemaGenerationSpecialCharacters(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	g.RegisterQuery(ctx, "getSpecial", func() sgSpecialType { return sgSpecialType{} })

	schema := g.SchemaDefinition(ctx)

	// Check that special characters are properly escaped/formatted
	assert.Contains(t, schema, `"""
Type with "special" characters
in description
"""`)
	assert.Contains(t, schema, `"""Contains "quotes" and backslash \"""`)
}

// Test types for subscription
type sgEvent struct {
	Type    string `json:"type" graphy:"description=Event type identifier"`
	Message string `json:"message" graphy:"description=Event message content"`
}

func (sgEvent) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Real-time event notification",
	}
}

// Test subscription descriptions in schema
func TestSchemaGenerationSubscriptions(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	subDesc := "Subscribe to real-time system events"
	g.RegisterFunction(ctx, FunctionDefinition{
		Name: "systemEvents",
		Function: func(ctx context.Context) (<-chan sgEvent, error) {
			ch := make(chan sgEvent)
			close(ch)
			return ch, nil
		},
		Description: &subDesc,
		Mode:        ModeSubscription,
	})

	schema := g.SchemaDefinition(ctx)

	// Verify subscription section exists with description
	assert.Contains(t, schema, `type Subscription {
	"""Subscribe to real-time system events"""
	systemEvents: sgEvent!
}`)

	// Verify Event type has descriptions
	assert.Contains(t, schema, `"""Real-time event notification"""
type sgEvent {`)
	assert.Contains(t, schema, `"""Event type identifier"""
	type: String!`)
	assert.Contains(t, schema, `"""Event message content"""
	message: String!`)
}

// Test types for type name generation
type sgUserInput struct {
	Name string `json:"name"`
}

type sgProductInput struct {
	Name string `json:"name"`
}

type sgComplexType struct {
	Simple  string  `json:"simple"`
	Pointer *string `json:"pointer"`
}

// Test that fixed type names are rendered correctly
func TestSchemaGenerationFixedTypeNames(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Test various type patterns that previously caused issues
	g.RegisterMutation(ctx, "createUser", func(input sgUserInput) string { return "ok" }, "input")
	g.RegisterMutation(ctx, "createProduct", func(input *sgProductInput) string { return "ok" }, "input")

	g.RegisterQuery(ctx, "getComplex", func() sgComplexType { return sgComplexType{} })

	schema := g.SchemaDefinition(ctx)

	// Verify input types are properly named
	assert.Contains(t, schema, "input sgUserInput {")
	assert.Contains(t, schema, "input sgProductInput {")
	assert.NotContains(t, schema, "input sgProductInputInput") // Should not have double "Input"

	// Verify output types are properly named
	assert.Contains(t, schema, "type sgComplexType {")

	// Verify mutations use correct type names
	assert.Contains(t, schema, "createUser(input: sgUserInput!): String!")
	assert.Contains(t, schema, "createProduct(input: sgProductInput): String!") // Pointer type is optional
}
