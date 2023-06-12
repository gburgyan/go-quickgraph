package quickgraph

import (
	"context"
	"fmt"
	"reflect"
)

type GraphFunction struct {
	name        string
	function    any
	nameMapping map[string]FunctionNameMapping
	returnType  reflect.Type
}

type FunctionNameMapping struct {
	name       string
	paramIndex int
	paramType  reflect.Type
	required   bool
}

func NewGraphFunction(name string, graphFunc any, names ...string) GraphFunction {
	mft := reflect.TypeOf(graphFunc)
	if mft.Kind() != reflect.Func {
		panic("graphFunc must be a func")
	}

	returnType := validateFunctionReturnTypes(mft)

	// Check the parameters of the graphFunc. The count of names must match the count of parameters.
	// Note that context.Context is not a parameter that is counted.
	nameMapping := map[string]FunctionNameMapping{}
	for i := 0; i < mft.NumIn(); i++ {
		funcParam := mft.In(i)
		if funcParam.ConvertibleTo(contextType) {
			continue
		}

		if len(nameMapping) >= len(names) {
			panic("too few names provided")
		}

		name := names[len(nameMapping)]

		required := true
		switch funcParam.Kind() {
		case reflect.Ptr:
			required = false

		case reflect.Map:
			panic("map parameters are not supported")
		}

		mapping := FunctionNameMapping{
			name:       name,
			paramIndex: i,
			paramType:  funcParam,
			required:   required,
		}

		nameMapping[name] = mapping
	}
	if len(nameMapping) != len(names) {
		panic("the count of names must match the count of parameters")
	}

	gf := GraphFunction{
		name:        name,
		function:    graphFunc,
		nameMapping: nameMapping,
		returnType:  returnType,
	}
	return gf
}

func validateFunctionReturnTypes(mft reflect.Type) reflect.Type {
	// Validate that the mutatorFunc has a single non-error return value and an optional error.
	if mft.NumOut() == 0 {
		panic("mutatorFunc must have at least one return value")
	}
	if mft.NumOut() > 2 {
		panic("mutatorFunc must have at most two return values")
	}

	errorCount := 0
	nonErrorCount := 0
	var returnType reflect.Type
	for i := 0; i < mft.NumOut(); i++ {
		if mft.Out(i).ConvertibleTo(errorType) {
			errorCount++
		} else {
			nonErrorCount++
			returnType = mft.Out(i)
		}
	}
	if errorCount > 1 {
		panic("mutatorFunc may have at most one error return value")
	}
	if nonErrorCount == 0 {
		panic("mutatorFunc must have at least one non-error return value")
	}
	return returnType
}

func (f *GraphFunction) Call(ctx context.Context, req *Request, command Command) (any, error) {

	paramValues, err := f.getCallParameters(ctx, req, command)
	if err != nil {
		return nil, err
	}

	gfv := reflect.ValueOf(f.function)
	callResults := gfv.Call(paramValues)
	if len(callResults) == 0 {
		return nil, nil
	}

	// TODO: Tighten this up to deal with the return types better.
	for _, callResult := range callResults {
		if callResult.CanConvert(errorType) {
			return nil, fmt.Errorf("error calling function: %v", callResult.Convert(errorType).Interface().(error))
		}
	}

	// Process the results
	return f.processCallResults(command, callResults)
}
