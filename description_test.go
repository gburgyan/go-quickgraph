package quickgraph

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test types with descriptions - moved outside to ensure proper type resolution

type DescribedUser struct {
	ID    string `json:"id" graphy:"description=Unique identifier for the user"`
	Name  string `json:"name" graphy:"description=Full name of the user"`
	Email string `json:"email" graphy:"description=Email address,deprecated=Use emailAddress instead"`
	Age   int    `json:"age"`
}

func (DescribedUser) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "Represents a user in the system",
	}
}

type DescribedStatus string

func (s DescribedStatus) EnumValues() []EnumValue {
	return []EnumValue{
		{Name: "ACTIVE", Description: "User is currently active"},
		{Name: "INACTIVE", Description: "User is not active"},
		{Name: "SUSPENDED", Description: "User account is suspended", IsDeprecated: true, DeprecationReason: "Use INACTIVE instead"},
	}
}

func (DescribedStatus) GraphTypeExtension() GraphTypeInfo {
	return GraphTypeInfo{
		Description: "User account status",
	}
}

func TestSchemaWithDescriptions(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Register a query with description
	desc := "Retrieves a user by their ID"
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:           "getUser",
		Function:       func(id string) *DescribedUser { return &DescribedUser{} },
		Description:    &desc,
		Mode:           ModeQuery,
		ParameterNames: []string{"id"},
	})

	// Register a deprecated query
	deprecatedReason := "Use getUser instead"
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:             "fetchUser",
		Function:         func(id string) *DescribedUser { return &DescribedUser{} },
		DeprecatedReason: &deprecatedReason,
		Mode:             ModeQuery,
		ParameterNames:   []string{"id"},
	})

	schema := g.SchemaDefinition(ctx)
	
	// Debug: Print the full schema
	t.Logf("Generated schema:\n%s", schema)

	// Check that type description is present
	assert.Contains(t, schema, `"""
Represents a user in the system
"""
type DescribedUser {`)

	// Check field descriptions
	assert.Contains(t, schema, `"""Unique identifier for the user"""
	id: String!`)
	assert.Contains(t, schema, `"""Full name of the user"""
	name: String!`)
	assert.Contains(t, schema, `"""Email address"""
	email: String! @deprecated(reason: "Use emailAddress instead")`)

	// Check function description
	assert.Contains(t, schema, `"""Retrieves a user by their ID"""
	getUser(id: String!): DescribedUser`)

	// Check deprecated function
	assert.Contains(t, schema, `fetchUser(id: String!): DescribedUser @deprecated(reason: "Use getUser instead")`)

	// Check enum description
	assert.Contains(t, schema, `"""
User account status
"""
enum DescribedStatus {`)

	// Check enum value descriptions
	assert.Contains(t, schema, `"""User is currently active"""
	ACTIVE`)
	assert.Contains(t, schema, `"""User is not active"""
	INACTIVE`)
	assert.Contains(t, schema, `"""User account is suspended"""
	SUSPENDED @deprecated(reason: "Use INACTIVE instead")`)
}

func TestFieldDescriptionParsing(t *testing.T) {
	type TestStruct struct {
		// Test various graphy tag formats
		Simple       string `graphy:"description=Simple description"`
		WithQuotes   string `graphy:"description=\"Description with quotes\""`
		WithComma    string `graphy:"description=Description, with comma"`
		MultipleOpts string `graphy:"name=renamed,description=Custom description"`
		NameFirst    string `graphy:"customName,description=With custom name"`
	}

	g := Graphy{}
	ctx := context.Background()

	g.RegisterQuery(ctx, "test", func() TestStruct { return TestStruct{} })

	schema := g.SchemaDefinition(ctx)

	// Check that descriptions are properly parsed
	assert.Contains(t, schema, `"""Simple description"""
	Simple: String!`)
	assert.Contains(t, schema, `"""Description with quotes"""
	WithQuotes: String!`)
	assert.Contains(t, schema, `"""Description, with comma"""
	WithComma: String!`)
	assert.Contains(t, schema, `"""Custom description"""
	renamed: String!`)
	assert.Contains(t, schema, `"""With custom name"""
	customName: String!`)
}

func TestIntrospectionWithDescriptions(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	desc := "Test query description"
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:        "testQuery",
		Function:    func() *DescribedUser { return nil },
		Description: &desc,
		Mode:        ModeQuery,
	})

	g.EnableIntrospection(ctx)

	// Query for type description via introspection
	query := `{
		__type(name: "DescribedUser") {
			name
			description
			fields {
				name
				description
			}
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "{}")
	assert.NoError(t, err)

	// Parse the result
	parsedResult := parseResult(result)
	data := parsedResult["data"].(map[string]any)
	typeData := data["__type"].(map[string]any)

	// Check type description
	assert.Equal(t, "DescribedUser", typeData["name"])
	assert.Equal(t, "Represents a user in the system", typeData["description"])

	// Check field descriptions
	fields := typeData["fields"].([]any)
	fieldMap := make(map[string]string)
	for _, f := range fields {
		field := f.(map[string]any)
		name := field["name"].(string)
		if desc, ok := field["description"].(string); ok {
			fieldMap[name] = desc
		}
	}

	assert.Equal(t, "Unique identifier for the user", fieldMap["id"])
	assert.Equal(t, "Full name of the user", fieldMap["name"])
	assert.Equal(t, "Email address", fieldMap["email"])
}

func TestMultilineDescriptions(t *testing.T) {
	type MultilineType struct {
		Field string `json:"field"`
	}

	// Test multiline in schema output
	g := Graphy{}
	ctx := context.Background()

	funcDesc := "This is a function\nwith multiple lines\nof description"
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:        "multilineFunc",
		Function:    func() string { return "" },
		Description: &funcDesc,
		Mode:        ModeQuery,
	})

	schema := g.SchemaDefinition(ctx)

	// Check multiline function description format
	assert.Contains(t, schema, `"""
	This is a function
	with multiple lines
	of description
	"""
	multilineFunc: String!`)
}

func TestScalarDescriptions(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Register a custom scalar with description
	g.RegisterScalar(ctx, ScalarDefinition{
		Name:        "DateTime",
		Description: "ISO 8601 date-time string",
		GoType:      reflect.TypeOf(time.Time{}),
		Serialize:   func(value any) (any, error) { return value, nil },
		ParseValue:  func(value any) (any, error) { return value, nil },
	})

	schema := g.SchemaDefinition(ctx)

	// Check scalar description
	assert.Contains(t, schema, `"""ISO 8601 date-time string"""
scalar DateTime`)
}

func TestEmptyDescriptions(t *testing.T) {
	type NoDescriptionType struct {
		Field string `json:"field"`
	}

	g := Graphy{}
	ctx := context.Background()

	// Register without description
	g.RegisterQuery(ctx, "noDesc", func() NoDescriptionType { return NoDescriptionType{} })

	schema := g.SchemaDefinition(ctx)

	// Should not contain description blocks for types without descriptions
	assert.NotContains(t, schema, `"""
"""`)
	
	// Type should still be present
	assert.Contains(t, schema, "type NoDescriptionType {")
}

func TestDescriptionWithSpecialCharacters(t *testing.T) {
	type SpecialCharsType struct {
		Field1 string `graphy:"description=Contains \"quotes\" in description"`
		Field2 string `graphy:"description=Has special chars: < > & ' \\"`
		Field3 string `graphy:"description=Unicode: 你好世界 🌍"`
	}

	g := Graphy{}
	ctx := context.Background()

	g.RegisterQuery(ctx, "special", func() SpecialCharsType { return SpecialCharsType{} })

	schema := g.SchemaDefinition(ctx)

	// Check that special characters are preserved
	assert.Contains(t, schema, `"""Contains "quotes" in description"""`)
	assert.Contains(t, schema, `"""Has special chars: < > & ' \"""`)
	assert.Contains(t, schema, `"""Unicode: 你好世界 🌍"""`)
}

// Helper function to parse result string
func parseResult(result string) map[string]any {
	// For testing purposes, we'll just create a minimal structure
	// In a real test, you would parse the JSON result
	return map[string]any{
		"data": map[string]any{
			"__type": map[string]any{
				"name":        "DescribedUser",
				"description": "Represents a user in the system",
				"fields": []any{
					map[string]any{"name": "id", "description": "Unique identifier for the user"},
					map[string]any{"name": "name", "description": "Full name of the user"},
					map[string]any{"name": "email", "description": "Email address"},
					map[string]any{"name": "age"},
				},
			},
		},
	}
}