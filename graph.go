package quickgraph

import (
	"context"
	"reflect"
	"sync"
)

type Graphy struct {
	processors  map[string]GraphFunction
	typeLookups map[reflect.Type]*TypeLookup
	m           sync.Mutex
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()
var stringType = reflect.TypeOf((*string)(nil)).Elem()

func (g *Graphy) RegisterProcessorWithParamNames(ctx context.Context, name string, mutatorFunc any, names ...string) {
	g.ensureInitialized()
	gf := NewGraphFunctionWithNames(name, reflect.ValueOf(mutatorFunc), names...)
	g.processors[name] = gf
}

func (g *Graphy) RegisterProcessor(ctx context.Context, name string, mutatorFunc any) {
	g.ensureInitialized()
	gf := NewGraphFunction(name, mutatorFunc, false)
	g.processors[name] = gf
}

func (g *Graphy) ensureInitialized() {
	if g.processors == nil {
		g.processors = map[string]GraphFunction{}
	}
}

func (g *Graphy) ProcessRequest(ctx context.Context, request string, variableJson string) (string, error) {
	var err error

	// TODO: Caching of the request stubs based on the request w/o the variables.
	rs, err := g.NewRequestStub(request)
	if err != nil {
		return "", err
	}

	newRequest, err := rs.NewRequest(variableJson)
	if err != nil {
		return "", err
	}

	return newRequest.Execute(ctx)
}

func (g *Graphy) TypeLookup(typ reflect.Type) *TypeLookup {
	g.m.Lock()
	defer g.m.Unlock()

	if g.typeLookups == nil {
		g.typeLookups = map[reflect.Type]*TypeLookup{}
	}

	if tl, ok := g.typeLookups[typ]; ok {
		return tl
	}

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() == reflect.Struct {
		tl := MakeTypeFieldLookup(typ)
		g.typeLookups[typ] = tl
		return tl
	}
	return nil
}

// TODO: Generate the schema from the registered processors.
