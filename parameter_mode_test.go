package quickgraph

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

// Test types for parameter mode testing
type UserInput struct {
	ID     string  `graphy:"id"`
	Name   *string `graphy:"name"`
	Active *bool   `graphy:"active"`
}

type User struct {
	ID     string `graphy:"id"`
	Name   string `graphy:"name"`
	Active bool   `graphy:"active"`
}

// Test functions with different signatures
func getUserStruct(ctx context.Context, input UserInput) *User {
	name := "Default"
	if input.Name != nil {
		name = *input.Name
	}
	active := true
	if input.Active != nil {
		active = *input.Active
	}
	return &User{
		ID:     input.ID,
		Name:   name,
		Active: active,
	}
}

func getUserMultiParam(ctx context.Context, id string, name *string, active *bool) *User {
	nameVal := "Default"
	if name != nil {
		nameVal = *name
	}
	activeVal := true
	if active != nil {
		activeVal = *active
	}
	return &User{
		ID:     id,
		Name:   nameVal,
		Active: activeVal,
	}
}

func getUserPositional(ctx context.Context, id string, includeInactive bool) *User {
	return &User{
		ID:     id,
		Name:   "User " + id,
		Active: !includeInactive,
	}
}

func TestParameterMode_StructParams(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Should work with struct parameter
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:          "getUser",
		Function:      getUserStruct,
		Mode:          ModeQuery,
		ParameterMode: StructParams,
	})

	query := `{
		getUser(id: "123", name: "John") {
			id
			name
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, `"id":"123"`)
	assert.Contains(t, result, `"name":"John"`)
}

func TestParameterMode_StructParams_ValidationError(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Should panic with non-struct parameter
	assert.Panics(t, func() {
		g.RegisterFunction(ctx, FunctionDefinition{
			Name:          "getUser",
			Function:      getUserMultiParam,
			Mode:          ModeQuery,
			ParameterMode: StructParams,
		})
	})
}

func TestParameterMode_NamedParams(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Should work with named parameters
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:           "getUser",
		Function:       getUserMultiParam,
		Mode:           ModeQuery,
		ParameterMode:  NamedParams,
		ParameterNames: []string{"id", "name", "active"},
	})

	query := `{
		getUser(id: "456", active: false) {
			id
			name
			active
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, `"id":"456"`)
	assert.Contains(t, result, `"name":"Default"`)
	assert.Contains(t, result, `"active":false`)
}

func TestParameterMode_NamedParams_ValidationError(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Should panic without parameter names
	assert.Panics(t, func() {
		g.RegisterFunction(ctx, FunctionDefinition{
			Name:          "getUser",
			Function:      getUserMultiParam,
			Mode:          ModeQuery,
			ParameterMode: NamedParams,
			// No ParameterNames provided
		})
	})

	// Should panic with wrong number of parameter names
	assert.Panics(t, func() {
		g.RegisterFunction(ctx, FunctionDefinition{
			Name:           "getUser",
			Function:       getUserMultiParam,
			Mode:           ModeQuery,
			ParameterMode:  NamedParams,
			ParameterNames: []string{"id", "name"}, // Missing "active"
		})
	})
}

func TestParameterMode_PositionalParams(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Should work with positional parameters
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:          "getUser",
		Function:      getUserPositional,
		Mode:          ModeQuery,
		ParameterMode: PositionalParams,
	})

	// Parameters are matched by position, not name
	query := `{
		getUser(arg1: "789", arg2: true) {
			id
			name
			active
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, `"id":"789"`)
	assert.Contains(t, result, `"name":"User 789"`)
	assert.Contains(t, result, `"active":false`) // includeInactive=true means active=false
}

func TestParameterMode_PositionalParams_ValidationError(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Should panic with parameter names
	assert.Panics(t, func() {
		g.RegisterFunction(ctx, FunctionDefinition{
			Name:           "getUser",
			Function:       getUserPositional,
			Mode:           ModeQuery,
			ParameterMode:  PositionalParams,
			ParameterNames: []string{"id", "includeInactive"}, // Not allowed!
		})
	})
}

func TestParameterMode_AutoDetect_Struct(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// AutoDetect should detect struct parameter
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:          "getUser",
		Function:      getUserStruct,
		Mode:          ModeQuery,
		ParameterMode: AutoDetect, // Or just omit it
	})

	query := `{
		getUser(id: "auto1", name: "Auto") {
			id
			name
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, `"id":"auto1"`)
	assert.Contains(t, result, `"name":"Auto"`)
}

func TestParameterMode_AutoDetect_MultipleParams(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// AutoDetect should use positional for multiple params without names
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:     "getUser",
		Function: getUserMultiParam,
		Mode:     ModeQuery,
		// ParameterMode defaults to AutoDetect
	})

	// Without parameter names, these are positional (arg1, arg2, arg3)
	query := `{
		getUser(arg1: "auto2", arg3: true) {
			id
			active
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, `"id":"auto2"`)
	assert.Contains(t, result, `"active":true`)
}

func TestParameterMode_AutoDetect_WithNames(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// AutoDetect with parameter names should use named params
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:           "getUser",
		Function:       getUserMultiParam,
		Mode:           ModeQuery,
		ParameterNames: []string{"id", "name", "active"},
		// ParameterMode defaults to AutoDetect
	})

	query := `{
		getUser(id: "auto3", active: false) {
			id
			active
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, `"id":"auto3"`)
	assert.Contains(t, result, `"active":false`)
}

func TestParameterMode_BackwardCompatibility(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Old style registration should still work
	g.RegisterQuery(ctx, "getUserOld", getUserStruct)

	query := `{
		getUserOld(id: "compat", name: "Compatible") {
			id
			name
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, `"id":"compat"`)
	assert.Contains(t, result, `"name":"Compatible"`)
}
