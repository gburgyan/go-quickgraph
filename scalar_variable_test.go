package quickgraph

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test custom scalar types for variable processing
type CustomMoney string
type CustomColor string

func TestScalarVariableProcessing(t *testing.T) {
	ctx := context.Background()
	graphy := &Graphy{}

	// Register custom scalars for testing variable processing
	err := graphy.RegisterScalar(ctx, ScalarDefinition{
		Name:   "CustomMoney",
		GoType: reflect.TypeOf(CustomMoney("")),
		Serialize: func(value interface{}) (interface{}, error) {
			if money, ok := value.(CustomMoney); ok {
				return string(money), nil
			}
			return nil, fmt.Errorf("expected CustomMoney, got %T", value)
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			if str, ok := value.(string); ok {
				return CustomMoney("PARSED:" + str), nil
			}
			return nil, fmt.Errorf("expected string, got %T", value)
		},
	})
	require.NoError(t, err)

	err = graphy.RegisterScalar(ctx, ScalarDefinition{
		Name:   "CustomColor",
		GoType: reflect.TypeOf(CustomColor("")),
		Serialize: func(value interface{}) (interface{}, error) {
			if color, ok := value.(CustomColor); ok {
				return string(color), nil
			}
			return nil, fmt.Errorf("expected CustomColor, got %T", value)
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			if str, ok := value.(string); ok {
				return CustomColor("COLOR:" + str), nil
			}
			return nil, fmt.Errorf("expected string, got %T", value)
		},
	})
	require.NoError(t, err)

	// Register a function that uses custom scalars
	graphy.RegisterMutation(ctx, "createTestProduct", func(name string, price CustomMoney, color CustomColor) string {
		return fmt.Sprintf("name:%s,price:%s,color:%s", name, string(price), string(color))
	}, "name", "price", "color")

	// Test with variables containing custom scalar values
	query := `mutation CreateTestProduct($name: String!, $price: CustomMoney!, $color: CustomColor!) {
		createTestProduct(name: $name, price: $price, color: $color)
	}`

	variables := `{
		"name": "Test Product",
		"price": "45.00 EUR",
		"color": "#FF0000"
	}`

	result, err := graphy.ProcessRequest(ctx, query, variables)
	require.NoError(t, err)

	// Verify that custom scalar ParseValue functions were called
	assert.Contains(t, result, `name:Test Product`)
	assert.Contains(t, result, `price:PARSED:45.00 EUR`)
	assert.Contains(t, result, `color:COLOR:#FF0000`)
}

func TestScalarVariableProcessing_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	graphy := &Graphy{}

	// Register a scalar that validates input
	err := graphy.RegisterScalar(ctx, ScalarDefinition{
		Name:   "ValidatedMoney",
		GoType: reflect.TypeOf(CustomMoney("")),
		Serialize: func(value interface{}) (interface{}, error) {
			return string(value.(CustomMoney)), nil
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			if str, ok := value.(string); ok {
				if str == "INVALID" {
					return nil, fmt.Errorf("invalid money format")
				}
				return CustomMoney(str), nil
			}
			return nil, fmt.Errorf("expected string, got %T", value)
		},
	})
	require.NoError(t, err)

	// Register a function that uses the validating scalar
	graphy.RegisterMutation(ctx, "createValidatedProduct", func(price CustomMoney) string {
		return string(price)
	}, "price")

	// Test with invalid scalar value
	query := `mutation CreateValidatedProduct($price: ValidatedMoney!) {
		createValidatedProduct(price: $price)
	}`

	variables := `{"price": "INVALID"}`

	_, err = graphy.ProcessRequest(ctx, query, variables)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid money format")
}

func TestScalarVariableProcessing_WithNonScalarTypes(t *testing.T) {
	ctx := context.Background()
	graphy := &Graphy{}

	// Register custom scalar
	err := graphy.RegisterScalar(ctx, ScalarDefinition{
		Name:   "TestString",
		GoType: reflect.TypeOf(CustomMoney("")),
		Serialize: func(value interface{}) (interface{}, error) {
			return string(value.(CustomMoney)), nil
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			if str, ok := value.(string); ok {
				return CustomMoney("SCALAR:" + str), nil
			}
			return nil, fmt.Errorf("expected string, got %T", value)
		},
	})
	require.NoError(t, err)

	// Register a function that mixes scalar and non-scalar types
	graphy.RegisterMutation(ctx, "mixedFunction", func(id int, scalarVal CustomMoney, flag bool) string {
		return fmt.Sprintf("id:%d,scalarVal:%s,flag:%t", id, string(scalarVal), flag)
	}, "id", "scalarVal", "flag")

	query := `mutation MixedFunction($id: Int!, $scalarVal: TestString!, $flag: Boolean!) {
		mixedFunction(id: $id, scalarVal: $scalarVal, flag: $flag)
	}`

	variables := `{
		"id": 42,
		"scalarVal": "test",
		"flag": true
	}`

	result, err := graphy.ProcessRequest(ctx, query, variables)
	require.NoError(t, err)

	// Verify that scalar processing worked for the scalar type
	assert.Contains(t, result, `id:42`)
	assert.Contains(t, result, `scalarVal:SCALAR:test`)
	assert.Contains(t, result, `flag:true`)
}

func TestScalarVariableProcessing_LiteralVsVariable(t *testing.T) {
	ctx := context.Background()
	graphy := &Graphy{}

	// Register a scalar that behaves differently for ParseValue vs ParseLiteral
	err := graphy.RegisterScalar(ctx, ScalarDefinition{
		Name:   "LiteralTestMoney",
		GoType: reflect.TypeOf(CustomMoney("")),
		Serialize: func(value interface{}) (interface{}, error) {
			return string(value.(CustomMoney)), nil
		},
		ParseValue: func(value interface{}) (interface{}, error) {
			if str, ok := value.(string); ok {
				return CustomMoney("VAR:" + str), nil // Variable prefix
			}
			return nil, fmt.Errorf("expected string, got %T", value)
		},
		ParseLiteral: func(value interface{}) (interface{}, error) {
			if str, ok := value.(string); ok {
				return CustomMoney("LIT:" + str), nil // Literal prefix
			}
			return nil, fmt.Errorf("expected string, got %T", value)
		},
	})
	require.NoError(t, err)

	// Register function
	graphy.RegisterMutation(ctx, "testLiteralVsVar", func(literalVal CustomMoney, variableVal CustomMoney) string {
		return fmt.Sprintf("literalVal:%s,variableVal:%s", string(literalVal), string(variableVal))
	}, "literalVal", "variableVal")

	// Test with one literal value and one variable value
	query := `mutation TestLiteralVsVar($variableVal: LiteralTestMoney!) {
		testLiteralVsVar(literalVal: "literal", variableVal: $variableVal)
	}`

	variables := `{"variableVal": "variable"}`

	result, err := graphy.ProcessRequest(ctx, query, variables)
	require.NoError(t, err)

	// Verify that literal and variable processing used different functions
	assert.Contains(t, result, `literalVal:LIT:literal`)   // Should use ParseLiteral
	assert.Contains(t, result, `variableVal:VAR:variable`) // Should use ParseValue
}
