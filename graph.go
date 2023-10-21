package quickgraph

import (
	"context"
	"reflect"
	"strings"
	"sync"
)

// Graphy is the main entry point for the go-quickgraph library. This holds all the
// registered functions and types and provides methods for executing requests.
// It is safe to use concurrently once it has been initialized -- there is no guarantee
// that the initialization is thread-safe.
//
// The zero value for Graphy is safe to use.
//
// The RequestCache is optional, but it can be used to cache the results of parsing
// requests. This can be useful if you are using the library in a server environment
// and want to cache the results of parsing requests to speed things up. Refer to
// GraphRequestCache for more information.
type Graphy struct {
	RequestCache GraphRequestCache

	processors  map[string]graphFunction
	typeLookups map[reflect.Type]*typeLookup
	anyTypes    []*typeLookup
	m           sync.Mutex
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()
var stringType = reflect.TypeOf((*string)(nil)).Elem()
var anyType = reflect.TypeOf((*any)(nil)).Elem()

// RegisterQuery registers a function as a query.
//
// The function must return a valid result value and may return an error. If the function
// returns multiple values, they must be pointers and the result will be an implicit
// union type.
//
// If the names are specified, they must match the non-context parameter count of the function.
// If the names are not specified, then the parameters are dealt with as either anonymous
// parameters or as a single parameter that is a struct. If the function has a single parameter
// that is a struct, then the names of the struct fields are used as the parameter names.
func (g *Graphy) RegisterQuery(ctx context.Context, name string, f any, names ...string) {
	g.ensureInitialized()
	gf := g.newGraphFunction(FunctionDefinition{
		Name:           name,
		Function:       f,
		ParameterNames: names,
		Mode:           ModeQuery,
	}, false)
	g.processors[name] = gf
}

// RegisterMutation registers a function as a mutator.
//
// The function must return a valid result value and may return an error. If the function
// returns multiple values, they must be pointers and the result will be an implicit
// union type.
//
// If the names are specified, they must match the non-context parameter count of the function.
// If the names are not specified, then the parameters are dealt with as either anonymous
// parameters or as a single parameter that is a struct. If the function has a single parameter
// that is a struct, then the names of the struct fields are used as the parameter names.
func (g *Graphy) RegisterMutation(ctx context.Context, name string, f any, names ...string) {
	g.ensureInitialized()
	gf := g.newGraphFunction(FunctionDefinition{
		Name:           name,
		Function:       f,
		ParameterNames: names,
		Mode:           ModeMutation,
	}, false)
	g.processors[name] = gf
}

// RegisterFunction is similar to both RegisterQuery and RegisterMutation, but it allows
// the caller to specify additional parameters that are less commonly used. See the
// FunctionDefinition documentation for more information.
func (g *Graphy) RegisterFunction(ctx context.Context, def FunctionDefinition) {
	g.ensureInitialized()
	gf := g.newGraphFunction(def, false)
	g.processors[def.Name] = gf
}

// RegisterAnyType registers a type that is potentially used as a return type for a function
// that returns `any`. This isn't critical to use all cases, but it can be needed for results
// that contain functions that can be called. Without this, those functions would not be
// found -- this it needed to infer the types of parameters in cases those are fulfilled with
// variables.
func (g *Graphy) RegisterAnyType(ctx context.Context, types ...any) {
	for _, t := range types {
		tl := g.typeLookup(reflect.TypeOf(t))
		g.anyTypes = append(g.anyTypes, tl)
	}
}

func (g *Graphy) ensureInitialized() {
	if g.processors == nil {
		g.processors = map[string]graphFunction{}
	}
}

func (g *Graphy) ProcessRequest(ctx context.Context, request string, variableJson string) (string, error) {
	rs, err := g.getRequestStub(ctx, request)
	if err != nil {
		return formatError(err), err
	}

	newRequest, err := rs.newRequest(variableJson)
	if err != nil {
		return formatError(err), err
	}

	return newRequest.execute(ctx)
}

func (g *Graphy) typeLookup(typ reflect.Type) *typeLookup {
	g.m.Lock()

	if g.typeLookups == nil {
		g.typeLookups = map[reflect.Type]*typeLookup{}
	}

	if tl, ok := g.typeLookups[typ]; ok {
		g.m.Unlock()
		return tl
	}

	result := &typeLookup{
		typ:                 typ,
		fields:              make(map[string]fieldLookup),
		fieldsLowercase:     make(map[string]fieldLookup),
		implements:          make(map[string]*typeLookup),
		implementsLowercase: make(map[string]*typeLookup),
		union:               make(map[string]*typeLookup),
		unionLowercase:      make(map[string]*typeLookup),
	}

	rootTyp := typ

	if rootTyp.Kind() == reflect.Ptr {
		rootTyp = rootTyp.Elem()
		result.isPointer = true
	}
	if rootTyp.Kind() == reflect.Slice {
		rootTyp = rootTyp.Elem()
		result.isSlice = true
		if rootTyp.Kind() == reflect.Ptr {
			rootTyp = rootTyp.Elem()
			result.isPointerSlice = true
		}
	}

	result.rootType = rootTyp

	result.name = rootTyp.Name()
	if rootTyp.Kind() == reflect.Struct {
		g.m.Unlock()
		g.processFieldLookup(rootTyp, nil, result)
		g.m.Lock()
		g.typeLookups[typ] = result
		g.m.Unlock()
		return result
	}
	if typ == anyType {
		for _, at := range g.anyTypes {
			result.union[at.name] = at
			result.unionLowercase[strings.ToLower(at.name)] = at
		}
		g.typeLookups[typ] = result
		g.m.Unlock()
		return result
	}
	// Fundamental types like floats and ints don't need these lookups because it doesn't make
	// sense in this context.
	result.fundamental = true
	g.typeLookups[typ] = result
	g.m.Unlock()
	return result
}

func (g *Graphy) getRequestStub(ctx context.Context, request string) (*RequestStub, error) {
	if g.RequestCache == nil {
		return g.newRequestStub(request)
	}

	stub, err := g.RequestCache.GetRequestStub(ctx, request)
	if stub != nil || err != nil {
		return stub, err
	}
	stub, err = g.newRequestStub(request)
	g.RequestCache.SetRequestStub(ctx, request, stub, err)
	if err != nil {
		return nil, err
	}
	return stub, nil
}
