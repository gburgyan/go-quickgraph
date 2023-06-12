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

// NewGraphFunction creates a new graph function given a name, function, and an optional list of parameter names.
// The function provided must be of the type func and the names provided should match the count of parameters.
// It panics if the function is not a func type, the number of names doesn't match parameters, or if unsupported
// parameters like map are provided.
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

// validateFunctionReturnTypes validates the return types of the function passed. It requires the function
// to have at least one non-error return value and at most one error return value. The function should have
// between one and two return values.
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

// Call executes the graph function with a given context, request and command. It first prepares the
// parameters for the function call, then invokes the function and processes the results. If the function
// returns an error, it returns a formatted error. If the function returns no results, it returns nil.
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
