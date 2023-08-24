package quickgraph

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

type TestUnion struct {
	A *string
	B *int
}

type NonUnion struct {
	A string
	B int
}

func TestDeferenceUnionType(t *testing.T) {
	// Test 1: Union type with only one field that is not nil.
	s := "test"
	testUnion := TestUnion{A: &s, B: nil}
	res, err := deferenceUnionType(testUnion)
	assert.NoError(t, err)
	assert.Equal(t, s, res)

	// Test 2: Union type with more than one field that is not nil.
	i := 1
	testUnion = TestUnion{A: &s, B: &i}
	_, err = deferenceUnionType(testUnion)
	assert.Error(t, err)
	assert.Equal(t, "more than one field in union type is not nil", err.Error())

	// Test 3: Union type with all fields that are nil.
	testUnion = TestUnion{A: nil, B: nil}
	_, err = deferenceUnionType(testUnion)
	assert.Error(t, err)
	assert.Equal(t, "no fields in union type are not nil", err.Error())

	// Test 4: Non-union type struct.
	nonUnion := NonUnion{A: "test", B: 1}
	_, err = deferenceUnionType(nonUnion)
	assert.Error(t, err)
	assert.Equal(t, "fields in union type must be pointers, maps, slices, or interfaces", err.Error())
}

func TestGraphFunction_Struct(t *testing.T) {
	type in struct {
		InString string
	}
	type res struct {
		OutString string
	}
	f := func(ctx context.Context, i in) res {
		return res{OutString: i.InString}
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterProcessor(ctx, "f", f)

	gf := g.newGraphFunction(FunctionDefinition{Name: "f", Function: f}, false)
	assert.Equal(t, "f", gf.name)
	assert.Equal(t, NamedParamsStruct, gf.paramType)

	gql := `
query {
  f(InString: "InputString") {
    OutString
  }
}`

	response, err := g.ProcessRequest(ctx, gql, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"f":{"OutString":"InputString"}}}`, response)
}

func TestGraphFunction_Anonymous(t *testing.T) {
	type res struct {
		OutString string
	}
	f := func(ctx context.Context, input string) res {
		return res{OutString: input}
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterProcessor(ctx, "f", f)

	gf := g.newGraphFunction(FunctionDefinition{Name: "f", Function: f}, false)
	assert.Equal(t, "f", gf.name)
	assert.Equal(t, AnonymousParamsInline, gf.paramType)

	gql := `
query {
  f(FooBar: "InputString") {
    OutString
  }
}`

	response, err := g.ProcessRequest(ctx, gql, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"f":{"OutString":"InputString"}}}`, response)
}

type resultWithFunc struct {
	OutString string
}

type funcResult struct {
	OutString string
}

func (r resultWithFunc) Func() string {
	return r.OutString
}

func (r *resultWithFunc) PFunc() string {
	return r.OutString
}

func (r resultWithFunc) FuncParam(s string) string {
	return s + " " + r.OutString
}

func TestGraphFunction_FuncReturn(t *testing.T) {
	type in struct {
		InString string
	}
	f := func(ctx context.Context, i in) resultWithFunc {
		return resultWithFunc{OutString: i.InString}
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterProcessor(ctx, "f", f)

	gf := g.newGraphFunction(FunctionDefinition{Name: "f", Function: f}, false)
	assert.Equal(t, "f", gf.name)
	assert.Equal(t, NamedParamsStruct, gf.paramType)

	gql := `
query {
  f(InString: "InputString") {
    Func
  }
}`

	response, err := g.ProcessRequest(ctx, gql, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"f":{"Func":"InputString"}}}`, response)
}

func TestGraphFunction_PointerFuncReturn(t *testing.T) {
	type in struct {
		InString string
	}
	f := func(ctx context.Context, i in) resultWithFunc {
		return resultWithFunc{OutString: i.InString}
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterProcessor(ctx, "f", f)

	gf := g.newGraphFunction(FunctionDefinition{Name: "f", Function: f}, false)
	assert.Equal(t, "f", gf.name)
	assert.Equal(t, NamedParamsStruct, gf.paramType)

	gql := `
query {
  f(InString: "InputString") {
    PFunc
  }
}`

	response, err := g.ProcessRequest(ctx, gql, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"f":{"PFunc":"InputString"}}}`, response)
}

func TestGraphFunction_FuncParamReturn(t *testing.T) {
	type in struct {
		InString string
	}
	f := func(ctx context.Context, i in) resultWithFunc {
		return resultWithFunc{OutString: i.InString}
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterProcessor(ctx, "f", f)

	gf := g.newGraphFunction(FunctionDefinition{Name: "f", Function: f}, false)
	assert.Equal(t, "f", gf.name)
	assert.Equal(t, NamedParamsStruct, gf.paramType)

	gql := `
query {
  f(InString: "InputString") {
    FuncParam(s: "Hello")
  }
}`

	response, err := g.ProcessRequest(ctx, gql, "")
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"f":{"FuncParam":"Hello InputString"}}}`, response)
}

func TestGraphFunction_FuncVariableParamReturn(t *testing.T) {
	type in struct {
		InString string
	}
	f := func(ctx context.Context, i in) resultWithFunc {
		return resultWithFunc{OutString: i.InString}
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterProcessor(ctx, "f", f)

	gf := g.newGraphFunction(FunctionDefinition{Name: "f", Function: f}, false)
	assert.Equal(t, "f", gf.name)
	assert.Equal(t, NamedParamsStruct, gf.paramType)

	gql := `
query {
  f(InString: "InputString") {
    FuncParam(s: $var)
  }
}`
	vars := `
{
  "var": "Hello"
}
`

	response, err := g.ProcessRequest(ctx, gql, vars)
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"f":{"FuncParam":"Hello InputString"}}}`, response)
}
