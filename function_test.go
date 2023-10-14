package quickgraph

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type TestUnion struct {
	A *string
	B *int
}

type NonUnion struct {
	A string
	B int
}

type StringResult struct {
	Out string
}

func DelayedFunc(ctx context.Context, sleepTime int64) StringResult {
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	return StringResult{Out: fmt.Sprintf("DelayedFunc: %v", sleepTime)}
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

func TestGraphFunction_ImplicitReturnUnion(t *testing.T) {
	type in struct {
		InString string
	}
	type resultA struct {
		OutStringA string
	}
	type resultB struct {
		OutStringB string
	}
	f := func(ctx context.Context, selector string) (*resultA, *resultB, error) {
		if selector == "A" {
			return &resultA{OutStringA: "A-Result"}, nil, nil
		}
		if selector == "B" {
			return nil, &resultB{OutStringB: "B-Result"}, nil
		}
		if selector == "AB" {
			return &resultA{OutStringA: "A-Result"}, &resultB{OutStringB: "B-Result"}, nil
		}
		if selector == "error" {
			return nil, nil, fmt.Errorf("error selector")
		}
		return nil, nil, nil
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterProcessor(ctx, "f", f)
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:            "CustomResultFunc",
		Function:        f,
		ReturnUnionName: "MyUnion",
	})

	gql := `
query f($arg: String!) {
  f(Arg: $arg) {
    __typename
	... on resultA {
		OutStringA
	}
	... on resultB {
		OutStringB
	}
  }
}`

	response, err := g.ProcessRequest(ctx, gql, `{"arg":"A"}`)
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"f":{"OutStringA":"A-Result","__typename":"resultA"}}}`, response)

	response, err = g.ProcessRequest(ctx, gql, `{"arg":"B"}`)
	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"f":{"OutStringB":"B-Result","__typename":"resultB"}}}`, response)

	response, err = g.ProcessRequest(ctx, gql, `{"arg":"AB"}`)
	assert.Error(t, err)
	assert.Equal(t, `{"data":{},"errors":[{"message":"function f returned multiple non-nil values","locations":[{"line":3,"column":5}],"path":["f"]}]}`, response)

	response, err = g.ProcessRequest(ctx, gql, `{"arg":"error"}`)
	assert.Error(t, err)
	assert.Equal(t, `{"data":{},"errors":[{"message":"function f returned error: error selector","locations":[{"line":3,"column":5}],"path":["f"]}]}`, response)

	response, err = g.ProcessRequest(ctx, gql, `{"arg":""}`)
	assert.Error(t, err)
	assert.Equal(t, `{"data":{},"errors":[{"message":"function f returned no non-nil values","locations":[{"line":3,"column":5}],"path":["f"]}]}`, response)

	expected := `type Query {
	CustomResultFunc(arg1: String!): MyUnion!
	f(arg1: String!): fResultUnion!
}

union MyUnion = resultA | resultB

union fResultUnion = resultA | resultB

type resultA {
	OutStringA: String!
}

type resultB {
	OutStringB: String!
}

`

	schema, err := g.SchemaDefinition(ctx)
	assert.NoError(t, err)
	assert.Equal(t, expected, schema)
}

func TestGraphFunction_ParallelQuery(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}
	g.RegisterProcessor(ctx, "delay", DelayedFunc)

	startTime := time.Now()
	gql := `
query {
  a: delay(sleepTime: 75) {
    Out
  }
  b: delay(sleepTime: 125) {
    Out
  }
}
`
	response, err := g.ProcessRequest(ctx, gql, "")
	endTime := time.Now()

	assert.NoError(t, err)
	assert.Equal(t, `{"data":{"a":{"Out":"DelayedFunc: 75"},"b":{"Out":"DelayedFunc: 125"}}}`, response)

	// The total time should be less than 200ms, since the queries are run in parallel.
	duration := endTime.Sub(startTime)
	assert.True(t, duration < 200*time.Millisecond)
}

func TestGraphFunction_ParallelQuery_Timeout(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	g := Graphy{}
	g.RegisterProcessor(ctx, "delay", DelayedFunc)

	startTime := time.Now()
	gql := `
query {
  a: delay(sleepTime: 50) {
    Out
  }
  b: delay(sleepTime: 125) {
    Out
  }
}
`
	response, err := g.ProcessRequest(ctx, gql, "")
	endTime := time.Now()

	assert.Error(t, err)
	assert.Equal(t, `{"data":{"a":{"Out":"DelayedFunc: 50"}},"errors":[{"message":"context timed out: context deadline exceeded"}]}`, response)

	// The total time should be less than 200ms, since the queries are run in parallel.
	duration := endTime.Sub(startTime)
	assert.True(t, duration < 150*time.Millisecond)
}

func TestGraphFunction_SerialQuery_Timeout(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 40*time.Millisecond)
	defer cancel()
	g := Graphy{}
	g.RegisterProcessor(ctx, "delay", DelayedFunc)

	startTime := time.Now()
	gql := `
mutation {
  a: delay(sleepTime: 50) {
    Out
  }
  b: delay(sleepTime: 125) {
    Out
  }
}
`
	response, err := g.ProcessRequest(ctx, gql, "")
	endTime := time.Now()

	assert.Error(t, err)
	assert.Equal(t, `{"data":{"a":{"Out":"DelayedFunc: 50"}},"errors":[{"message":"context timed out: context deadline exceeded"}]}`, response)

	// The total time should be less than 200ms, since the queries are run in parallel.
	duration := endTime.Sub(startTime)
	assert.True(t, duration < 100*time.Millisecond)
}

func TestGraphFunction_Invalid(t *testing.T) {
	type in struct {
		InString string
	}
	type res struct {
		OutString string
	}

	ctx := context.Background()
	g := Graphy{}
	assert.PanicsWithValue(t, "not valid graph function: function may have at most one non-pointer return value", func() {
		g.RegisterProcessor(ctx, "f", func() (episode, episode) { return "foo", "bar" })
	})

	assert.PanicsWithValue(t, "not valid graph function: function must have at least one non-error return value", func() {
		g.RegisterProcessor(ctx, "f", func() error { return nil })
	})

	assert.PanicsWithValue(t, "not valid graph function: function may have at most one error return value", func() {
		g.RegisterProcessor(ctx, "f", func() (episode, error, error) { return "foo", nil, nil })
	})

	assert.PanicsWithValue(t, "not valid graph function: function f is not a func: string", func() {
		g.RegisterProcessor(ctx, "f", "Not a function")
	})
}

func TestGraphFunction_IncorrectQueryMode(t *testing.T) {
	type in struct {
		InString string
	}
	f := func(ctx context.Context, i in) Character {
		return Character{}
	}

	ctx := context.Background()
	g := Graphy{}
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:     "f",
		Function: f,
		Mode:     ModeMutation,
	})

	gql := `
query {
  f(InString: "InputString") {
    Name
  }
}`
	response, err := g.ProcessRequest(ctx, gql, "")
	assert.Error(t, err)
	assert.Equal(t, `{"errors":[{"message":"mutation f used in query","locations":[{"line":3,"column":3}]}]}`, response)
}

func TestGraphFunction_BadVariableType(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 40*time.Millisecond)
	defer cancel()
	g := Graphy{}
	g.RegisterProcessor(ctx, "delay", DelayedFunc)

	gql := `
query f($time: int!) {
  delay(sleepTime: $time) {
    Out
  }
}
`
	response, err := g.ProcessRequest(ctx, gql, `{"time": "foo"}`)

	assert.Error(t, err)
	assert.Equal(t, `{"errors":[{"message":"error parsing variable time into type int64: json: cannot unmarshal string into Go value of type int64"}]}`, response)
}

func TestGraphFunction_BadDefaultVariableType(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 40*time.Millisecond)
	defer cancel()
	g := Graphy{}
	g.RegisterProcessor(ctx, "delay", DelayedFunc)

	gql := `
query f($time: int! = "foo") {
  delay(sleepTime: $time) {
    Out
  }
}
`
	response, err := g.ProcessRequest(ctx, gql, ``)

	assert.Error(t, err)
	assert.Equal(t, `{"errors":[{"message":"error parsing default variable time into type int64: panic: reflect: call of reflect.Value.SetString on int64 Value","locations":[{"line":2,"column":23}]}]}`, response)
}

func TestGraphFunction_MissingVariable(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 40*time.Millisecond)
	defer cancel()
	g := Graphy{}
	g.RegisterProcessor(ctx, "delay", DelayedFunc)

	gql := `
query f($time: int!) {
  delay(sleepTime: $time) {
    Out
  }
}
`
	response, err := g.ProcessRequest(ctx, gql, ``)

	assert.Error(t, err)
	assert.Equal(t, `{"errors":[{"message":"variable time not provided"}]}`, response)
}
