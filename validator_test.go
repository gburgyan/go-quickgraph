package quickgraph

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// Test types that implement the Validator interface
type ValidatedUserInput struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func (u ValidatedUserInput) Validate() error {
	if u.Name == "" {
		return errors.New("name is required")
	}
	if len(u.Name) < 2 {
		return errors.New("name must be at least 2 characters")
	}
	if u.Email == "" {
		return errors.New("email is required")
	}
	if !strings.Contains(u.Email, "@") {
		return errors.New("invalid email format")
	}
	if u.Age < 13 || u.Age > 120 {
		return errors.New("age must be between 13 and 120")
	}
	return nil
}

// Test type that implements ValidatorWithContext
type AuthorizedUserInput struct {
	UserID string `json:"userId"`
	Role   string `json:"role"`
	Action string `json:"action"`
}

func (a AuthorizedUserInput) ValidateWithContext(ctx context.Context) error {
	// Check if user is authenticated
	currentUser, ok := ctx.Value("currentUser").(string)
	if !ok || currentUser == "" {
		return errors.New("authentication required")
	}
	
	// Check if user has permission for the action
	if a.Action == "delete" && a.Role != "admin" {
		return errors.New("only admins can delete")
	}
	
	// Check if user can modify their own data
	if a.UserID != currentUser && a.Role != "admin" {
		return errors.New("can only modify your own data")
	}
	
	return nil
}

// Test functions that use validated inputs
func CreateValidatedUser(ctx context.Context, input ValidatedUserInput) (string, error) {
	// The validation happens automatically before this function is called
	return fmt.Sprintf("User %s created with email %s", input.Name, input.Email), nil
}

func UpdateUserRole(ctx context.Context, input AuthorizedUserInput) (string, error) {
	// The validation happens automatically before this function is called
	return fmt.Sprintf("User %s role updated to %s", input.UserID, input.Role), nil
}

func TestValidatorInterface(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}
	g.RegisterQuery(ctx, "createValidatedUser", CreateValidatedUser, "input")
	
	// Test valid input
	query := `{
		createValidatedUser(input: {
			name: "Alice"
			email: "alice@example.com"
			age: 25
		})
	}`
	
	result, err := g.ProcessRequest(ctx, query, "")
	if err != nil {
		t.Errorf("Expected no error for valid input, got: %v", err)
	}
	if !strings.Contains(result, "User Alice created") {
		t.Errorf("Expected success message, got: %s", result)
	}
	
	// Test validation error - missing name
	query = `{
		createValidatedUser(input: {
			name: ""
			email: "alice@example.com"
			age: 25
		})
	}`
	
	result, err = g.ProcessRequest(ctx, query, "")
	// Validation errors are returned as errors from ProcessRequest
	if err == nil {
		t.Errorf("Expected validation error for empty name")
	} else if !strings.Contains(err.Error(), "name is required") {
		t.Errorf("Expected 'name is required' error, got: %v", err)
	}
	
	// Test validation error - invalid email
	query = `{
		createValidatedUser(input: {
			name: "Alice"
			email: "not-an-email"
			age: 25
		})
	}`
	
	result, err = g.ProcessRequest(ctx, query, "")
	if err == nil {
		t.Errorf("Expected validation error for invalid email")
	} else if !strings.Contains(err.Error(), "invalid email format") {
		t.Errorf("Expected 'invalid email format' error, got: %v", err)
	}
	
	// Test validation error - age out of range
	query = `{
		createValidatedUser(input: {
			name: "Alice"
			email: "alice@example.com"
			age: 150
		})
	}`
	
	result, err = g.ProcessRequest(ctx, query, "")
	if err == nil {
		t.Errorf("Expected validation error for age out of range")
	} else if !strings.Contains(err.Error(), "age must be between 13 and 120") {
		t.Errorf("Expected age validation error, got: %v", err)
	}
}

func TestValidatorWithContextInterface(t *testing.T) {
	g := Graphy{}
	
	// Create context with authenticated user
	ctx := context.WithValue(context.Background(), "currentUser", "user123")
	g.RegisterMutation(ctx, "updateUserRole", UpdateUserRole, "input")
	
	// Test valid input - user updating their own role
	query := `mutation {
		updateUserRole(input: {
			userId: "user123"
			role: "editor"
			action: "update"
		})
	}`
	
	result, err := g.ProcessRequest(ctx, query, "")
	if err != nil {
		t.Errorf("Expected no error for valid input, got: %v", err)
	}
	if !strings.Contains(result, "role updated to editor") {
		t.Errorf("Expected success message, got: %s", result)
	}
	
	// Test validation error - no authentication
	ctxNoAuth := context.Background()
	result, err = g.ProcessRequest(ctxNoAuth, query, "")
	if err == nil {
		t.Errorf("Expected authentication error")
	} else if !strings.Contains(err.Error(), "authentication required") {
		t.Errorf("Expected 'authentication required' error, got: %v", err)
	}
	
	// Test validation error - trying to modify another user
	query = `mutation {
		updateUserRole(input: {
			userId: "otheruser"
			role: "editor"
			action: "update"
		})
	}`
	
	result, err = g.ProcessRequest(ctx, query, "")
	if err == nil {
		t.Errorf("Expected permission error")
	} else if !strings.Contains(err.Error(), "can only modify your own data") {
		t.Errorf("Expected permission error, got: %v", err)
	}
	
	// Test validation error - non-admin trying to delete
	query = `mutation {
		updateUserRole(input: {
			userId: "user123"
			role: "editor"
			action: "delete"
		})
	}`
	
	result, err = g.ProcessRequest(ctx, query, "")
	if err == nil {
		t.Errorf("Expected permission error for delete")
	} else if !strings.Contains(err.Error(), "only admins can delete") {
		t.Errorf("Expected 'only admins can delete' error, got: %v", err)
	}
	
	// Test admin can modify other users
	ctxAdmin := context.WithValue(context.Background(), "currentUser", "admin123")
	query = `mutation {
		updateUserRole(input: {
			userId: "otheruser"
			role: "admin"
			action: "update"
		})
	}`
	
	result, err = g.ProcessRequest(ctxAdmin, query, "")
	if err != nil {
		t.Errorf("Expected no error for admin action, got: %v", err)
	}
	if !strings.Contains(result, "role updated to admin") {
		t.Errorf("Expected success message for admin, got: %s", result)
	}
}

// Test with pointer types
type ProductInput struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func (p *ProductInput) Validate() error {
	if p.Name == "" {
		return errors.New("product name is required")
	}
	if p.Price <= 0 {
		return fmt.Errorf("price must be greater than 0, got %.2f", p.Price)
	}
	return nil
}

func CreateProduct(ctx context.Context, input *ProductInput) (string, error) {
	// Debug: The validation should have already happened before we get here
	return fmt.Sprintf("Product %s created with price %.2f", input.Name, input.Price), nil
}

func TestValidatorWithPointer(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}
	g.RegisterMutation(ctx, "createProduct", CreateProduct)
	
	// Test valid input
	query := `mutation {
		createProduct(name: "Laptop", price: 999.99)
	}`
	
	result, err := g.ProcessRequest(ctx, query, "")
	if err != nil {
		t.Errorf("Expected no error for valid input, got: %v", err)
	}
	if !strings.Contains(result, "Product Laptop created") {
		t.Errorf("Expected success message, got: %s", result)
	}
	
	// Test validation error - empty name
	query = `mutation {
		createProduct(name: "", price: 100)
	}`
	
	result, err = g.ProcessRequest(ctx, query, "")
	if err == nil {
		t.Errorf("Expected validation error for empty name")
	} else if !strings.Contains(err.Error(), "product name is required") {
		t.Errorf("Expected name validation error, got: %v", err)
	}
}