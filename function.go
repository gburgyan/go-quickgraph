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
	for _, callResult := range callResults {
		kind := callResult.Kind()
		if (kind == reflect.Pointer) && !callResult.IsNil() {
			// If this is a pointer, dereference it.
			callResult = callResult.Elem()
			kind = callResult.Kind() // Update the kind
		}
		if kind == reflect.Slice {
			if !callResult.IsNil() {
				var retVal []any
				count := callResult.Len()
				for i := 0; i < count; i++ {
					a := callResult.Index(i).Interface()
					sr, err := processStruct(command.ResultFilter, a)
					if err != nil {
						return nil, err
					}
					retVal = append(retVal, sr)
				}
				return retVal, nil
			}

		} else if kind == reflect.Map {
			return nil, fmt.Errorf("return of map type not supported")
		} else if kind == reflect.Struct {
			sr, err := processStruct(command.ResultFilter, callResult.Interface())
			if err != nil {
				return nil, err
			}
			return sr, nil
		}
	}

	// TODO: Better error handling.

	return nil, nil
}
