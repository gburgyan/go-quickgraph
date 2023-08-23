package quickgraph

import (
	"context"
	"reflect"
)

type DirectiveInput struct {
	Name string
	Type reflect.Type
}

type DirectiveHandler interface {
	InputTypes() []DirectiveInput
	HandlePreFetch(ctx context.Context, request *Request, directive *Directive) (bool, error)
}
