package quickgraph

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test types for nested validation
type valAddressInput struct {
	Street  string `graphy:"street"`
	City    string `graphy:"city"`
	ZipCode string `graphy:"zipCode"`
}

func (a valAddressInput) Validate() error {
	if a.Street == "" {
		return errors.New("street is required")
	}
	if a.City == "" {
		return errors.New("city is required")
	}
	if a.ZipCode == "" {
		return errors.New("zip code is required")
	}
	// Validate zip code format (5 digits)
	if len(a.ZipCode) != 5 {
		return errors.New("zip code must be 5 digits")
	}
	for _, ch := range a.ZipCode {
		if ch < '0' || ch > '9' {
			return errors.New("zip code must contain only digits")
		}
	}
	return nil
}

type valPersonInput struct {
	Name    string          `graphy:"name"`
	Age     int             `graphy:"age"`
	Address valAddressInput `graphy:"address"`
}

func (p valPersonInput) Validate() error {
	if p.Name == "" {
		return errors.New("name is required")
	}
	if p.Age < 0 || p.Age > 150 {
		return errors.New("age must be between 0 and 150")
	}
	// Nested validation happens automatically
	return nil
}

// Test validation with nested input types
func TestValidationWithNestedTypes(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterMutation(ctx, "createPerson", func(ctx context.Context, input valPersonInput) (string, error) {
		return fmt.Sprintf("Person %s created at %s, %s", input.Name, input.Address.Street, input.Address.City), nil
	}, "input")

	// Test valid nested input
	query := `mutation {
		createPerson(input: {
			name: "John Doe"
			age: 30
			address: {
				street: "123 Main St"
				city: "New York"
				zipCode: "10001"
			}
		})
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, "Person John Doe created")

	// Test validation error in nested type
	query = `mutation {
		createPerson(input: {
			name: "Jane Doe"
			age: 25
			address: {
				street: "456 Oak Ave"
				city: "Boston"
				zipCode: "ABCDE"
			}
		})
	}`

	result, err = g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "zip code must contain only digits")

	// Test missing field in nested type
	query = `mutation {
		createPerson(input: {
			name: "Bob Smith"
			age: 40
			address: {
				street: ""
				city: "Chicago"
				zipCode: "60601"
			}
		})
	}`

	result, err = g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "street is required")
}

// Test types for parameter modes
type valRangeInput struct {
	Min int `graphy:"min"`
	Max int `graphy:"max"`
}

func (r valRangeInput) Validate() error {
	if r.Min > r.Max {
		return errors.New("min must be less than or equal to max")
	}
	if r.Min < 0 {
		return errors.New("min must be non-negative")
	}
	if r.Max > 1000 {
		return errors.New("max must be at most 1000")
	}
	return nil
}

type valValidatedInt struct {
	Value int `graphy:"value"`
}

func (v valValidatedInt) Validate() error {
	if v.Value < 1 || v.Value > 10 {
		return errors.New("value must be between 1 and 10")
	}
	return nil
}

// Test validation with different parameter modes
func TestValidationWithParameterModes(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Test with struct-based parameter mode
	g.RegisterQuery(ctx, "getNumbersInRange", func(ctx context.Context, range_ valRangeInput) []int {
		result := []int{}
		for i := range_.Min; i <= range_.Max; i++ {
			result = append(result, i)
		}
		return result
	}, "range")

	// Test with anonymous parameter mode using ValidatedInput wrapper
	g.RegisterQuery(ctx, "multiplyByTwo", func(ctx context.Context, num valValidatedInt) int {
		return num.Value * 2
	}, "num")

	// Test struct-based with valid input
	query := `{
		getNumbersInRange(range: { min: 5, max: 8 })
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, "[5,6,7,8]")

	// Test struct-based with validation error
	query = `{
		getNumbersInRange(range: { min: 10, max: 5 })
	}`

	result, err = g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "min must be less than or equal to max")

	// Test anonymous mode with valid input
	query = `{
		multiplyByTwo(num: { value: 5 })
	}`

	result, err = g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, "10")

	// Test anonymous mode with validation error
	query = `{
		multiplyByTwo(num: { value: 15 })
	}`

	result, err = g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "value must be between 1 and 10")
}

// Test types for error propagation
type valTestInput struct {
	Field1 string `graphy:"field1"`
	Field2 int    `graphy:"field2"`
}

func (t valTestInput) Validate() error {
	if t.Field1 == "error1" {
		return errors.New("field1 validation failed")
	}
	if t.Field2 == 42 {
		return errors.New("field2 cannot be 42")
	}
	return nil
}

// Test validation error propagation in GraphQL responses
func TestValidationErrorPropagation(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterMutation(ctx, "testValidation", func(ctx context.Context, input valTestInput) (string, error) {
		return "success", nil
	}, "input")

	// Test that validation errors are properly formatted in GraphQL response
	query := `mutation {
		testValidation(input: { field1: "error1", field2: 10 })
	}`

	_, err := g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "field1 validation failed")

	// Verify error doesn't contain implementation details
	assert.NotContains(t, err.Error(), "valTestInput")
	assert.NotContains(t, err.Error(), "Validate()")
}

// Test types for optional fields
type valOptionalFieldsInput struct {
	RequiredField string  `graphy:"requiredField"`
	OptionalField *string `graphy:"optionalField"`
	OptionalInt   *int    `graphy:"optionalInt"`
}

func (o valOptionalFieldsInput) Validate() error {
	if o.RequiredField == "" {
		return errors.New("requiredField is required")
	}
	// Validate optional fields only if present
	if o.OptionalField != nil && *o.OptionalField == "" {
		return errors.New("optionalField cannot be empty if provided")
	}
	if o.OptionalInt != nil && *o.OptionalInt < 0 {
		return errors.New("optionalInt must be non-negative if provided")
	}
	return nil
}

// Test validation with optional fields (pointers)
func TestValidationWithOptionalFields(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterQuery(ctx, "processOptionalFields", func(ctx context.Context, input valOptionalFieldsInput) string {
		result := "Required: " + input.RequiredField
		if input.OptionalField != nil {
			result += ", Optional: " + *input.OptionalField
		}
		if input.OptionalInt != nil {
			result += fmt.Sprintf(", Int: %d", *input.OptionalInt)
		}
		return result
	}, "input")

	// Test with only required field
	query := `{
		processOptionalFields(input: { requiredField: "test" })
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, "Required: test")

	// Test with all fields
	query = `{
		processOptionalFields(input: { 
			requiredField: "test",
			optionalField: "optional",
			optionalInt: 5
		})
	}`

	result, err = g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, "Required: test, Optional: optional, Int: 5")

	// Test validation error on optional field
	query = `{
		processOptionalFields(input: { 
			requiredField: "test",
			optionalField: ""
		})
	}`

	result, err = g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "optionalField cannot be empty if provided")

	// Test validation error on missing required field
	query = `{
		processOptionalFields(input: { 
			requiredField: "",
			optionalField: "test"
		})
	}`

	result, err = g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requiredField is required")
}

// Test types for arrays
type valItemInput struct {
	Name     string `graphy:"name"`
	Quantity int    `graphy:"quantity"`
}

func (i valItemInput) Validate() error {
	if i.Name == "" {
		return errors.New("item name is required")
	}
	if i.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	if i.Quantity > 100 {
		return errors.New("quantity cannot exceed 100")
	}
	return nil
}

type valOrderInput struct {
	CustomerName string         `graphy:"customerName"`
	Items        []valItemInput `graphy:"items"`
}

func (o valOrderInput) Validate() error {
	if o.CustomerName == "" {
		return errors.New("customer name is required")
	}
	if len(o.Items) == 0 {
		return errors.New("order must contain at least one item")
	}
	if len(o.Items) > 10 {
		return errors.New("order cannot contain more than 10 items")
	}
	// Check for duplicate items
	itemNames := make(map[string]bool)
	for _, item := range o.Items {
		if itemNames[item.Name] {
			return fmt.Errorf("duplicate item: %s", item.Name)
		}
		itemNames[item.Name] = true
	}
	return nil
}

// Test validation with arrays and complex types
func TestValidationWithArrays(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterMutation(ctx, "createOrder", func(ctx context.Context, input valOrderInput) (string, error) {
		total := 0
		for _, item := range input.Items {
			total += item.Quantity
		}
		return fmt.Sprintf("Order created for %s with %d total items", input.CustomerName, total), nil
	}, "input")

	// Test valid order
	query := `mutation {
		createOrder(input: {
			customerName: "Alice"
			items: [
				{ name: "Widget A", quantity: 5 },
				{ name: "Widget B", quantity: 3 }
			]
		})
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, "Order created for Alice with 8 total items")

	// Test validation error in array item
	query = `mutation {
		createOrder(input: {
			customerName: "Bob"
			items: [
				{ name: "Widget A", quantity: 5 },
				{ name: "", quantity: 3 }
			]
		})
	}`

	result, err = g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "item name is required")

	// Test validation error for duplicate items
	query = `mutation {
		createOrder(input: {
			customerName: "Charlie"
			items: [
				{ name: "Widget A", quantity: 5 },
				{ name: "Widget A", quantity: 3 }
			]
		})
	}`

	result, err = g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate item: Widget A")

	// Test validation error for empty items array
	query = `mutation {
		createOrder(input: {
			customerName: "David"
			items: []
		})
	}`

	result, err = g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order must contain at least one item")
}

// Custom validation error that includes field information
type valValidationError struct {
	Field   string
	Message string
}

func (v valValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", v.Field, v.Message)
}

type valComplexInput struct {
	Username string `graphy:"username"`
	Email    string `graphy:"email"`
	Age      int    `graphy:"age"`
}

func (c valComplexInput) Validate() error {
	// Username validation
	if c.Username == "" {
		return valValidationError{Field: "username", Message: "username is required"}
	}
	if len(c.Username) < 3 {
		return valValidationError{Field: "username", Message: "username must be at least 3 characters"}
	}
	if strings.Contains(c.Username, " ") {
		return valValidationError{Field: "username", Message: "username cannot contain spaces"}
	}

	// Email validation
	if c.Email == "" {
		return valValidationError{Field: "email", Message: "email is required"}
	}
	if !strings.Contains(c.Email, "@") || !strings.Contains(c.Email, ".") {
		return valValidationError{Field: "email", Message: "invalid email format"}
	}

	// Age validation
	if c.Age < 13 {
		return valValidationError{Field: "age", Message: "minimum age is 13"}
	}
	if c.Age > 120 {
		return valValidationError{Field: "age", Message: "age seems unrealistic"}
	}

	return nil
}

// Test validation with custom error types
func TestValidationWithCustomErrors(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterMutation(ctx, "registerUser", func(ctx context.Context, input valComplexInput) (string, error) {
		return fmt.Sprintf("User %s registered with email %s", input.Username, input.Email), nil
	}, "input")

	// Test validation with custom error for username
	query := `mutation {
		registerUser(input: {
			username: "ab"
			email: "test@example.com"
			age: 25
		})
	}`

	_, err := g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed for field 'username'")
	assert.Contains(t, err.Error(), "username must be at least 3 characters")

	// Test validation with custom error for email
	query = `mutation {
		registerUser(input: {
			username: "testuser"
			email: "invalid-email"
			age: 25
		})
	}`

	_, err = g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation failed for field 'email'")
	assert.Contains(t, err.Error(), "invalid email format")
}

type valTrackedInput struct {
	Value string `graphy:"value"`
}

func (t valTrackedInput) Validate() error {
	if t.Value == "invalid" {
		return errors.New("invalid value")
	}
	return nil
}

// Test that validation happens before function execution
func TestValidationTiming(t *testing.T) {
	executionCount := 0

	ctx := context.Background()
	g := Graphy{}

	g.RegisterMutation(ctx, "trackedMutation", func(ctx context.Context, input valTrackedInput) (string, error) {
		executionCount++
		return fmt.Sprintf("Executed with value: %s (count: %d)", input.Value, executionCount), nil
	}, "input")

	// Execute with valid input
	query := `mutation {
		trackedMutation(input: { value: "valid" })
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, "Executed with value: valid (count: 1)")
	assert.Equal(t, 1, executionCount)

	// Execute with invalid input - should not increase execution count
	query = `mutation {
		trackedMutation(input: { value: "invalid" })
	}`

	_, err = g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value")
	assert.Equal(t, 1, executionCount) // Should still be 1, function not executed
}
