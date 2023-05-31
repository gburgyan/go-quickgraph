package quickgraph

import (
	"context"
	"reflect"
)

type Graphy struct {
	processors map[string]GraphFunction
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()

func (g *Graphy) RegisterProcessorWithParamNames(ctx context.Context, name string, mutatorFunc any, names ...string) {
	gf := NewGraphFunction(name, mutatorFunc, names...)
	if g.processors == nil {
		g.processors = map[string]GraphFunction{}
	}
	g.processors[name] = gf
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