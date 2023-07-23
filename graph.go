package quickgraph

import (
	"context"
	"reflect"
	"strings"
	"sync"
)

type Graphy struct {
	processors  map[string]GraphFunction
	typeLookups map[reflect.Type]*TypeLookup
	anyTypes    []*TypeLookup
	m           sync.Mutex
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()
var stringType = reflect.TypeOf((*string)(nil)).Elem()
var anyType = reflect.TypeOf((*any)(nil)).Elem()

func (g *Graphy) RegisterProcessorWithParamNames(ctx context.Context, name string, f any, names ...string) {
	g.ensureInitialized()
	gf := g.newGraphFunctionWithNames(name, reflect.ValueOf(f), names...)
	g.processors[name] = gf
}

func (g *Graphy) RegisterProcessor(ctx context.Context, name string, f any) {
	g.ensureInitialized()
	gf := g.newGraphFunction(name, f, false)
	g.processors[name] = gf
}

func (g *Graphy) RegisterAnyType(ctx context.Context, types ...any) {
	for _, t := range types {
		tl := g.typeLookup(reflect.TypeOf(t))
		g.anyTypes = append(g.anyTypes, tl)
	}
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

func (g *Graphy) typeLookup(typ reflect.Type) *TypeLookup {
	g.m.Lock()

	if g.typeLookups == nil {
		g.typeLookups = map[reflect.Type]*TypeLookup{}
	}

	if tl, ok := g.typeLookups[typ]; ok {
		g.m.Unlock()
		return tl
	}

	if typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() == reflect.Struct {
		g.m.Unlock()
		tl := g.makeTypeFieldLookup(typ)
		g.m.Lock()
		g.typeLookups[typ] = tl
		g.m.Unlock()
		return tl
	}
	if typ == anyType {
		result := &TypeLookup{
			fields:              make(map[string]FieldLookup),
			fieldsLowercase:     map[string]FieldLookup{},
			implements:          map[string]bool{},
			implementsLowercase: map[string]bool{},
			union:               map[string]*TypeLookup{},
			unionLowercase:      map[string]*TypeLookup{},
		}
		for _, at := range g.anyTypes {
			result.union[at.name] = at
			result.unionLowercase[strings.ToLower(at.name)] = at
		}
		g.typeLookups[typ] = result
		g.m.Unlock()
		return result
	}
	g.typeLookups[typ] = nil
	g.m.Unlock()
	return nil
}

// TODO: Generate the schema from the registered processors.
