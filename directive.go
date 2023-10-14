package quickgraph

import (
	"context"
	"reflect"
)

type directiveInput struct {
	Name string
	Type reflect.Type
}

type directiveHandler interface {
	InputTypes() []directiveInput
	HandlePreFetch(ctx context.Context, request *request, directive *directive) (bool, error)
}
