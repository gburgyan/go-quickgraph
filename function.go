package quickgraph

import (
	"context"
	"fmt"
	"reflect"
)

type GraphFunctionMode int

const (
	NamedParamsStruct GraphFunctionMode = iota
	NamedParamsInline
	AnonymousParamsInline
)

type GraphFunction struct {
	name        string
	mode        GraphFunctionMode
	function    reflect.Value
	nameMapping map[string]FunctionNameMapping
	returnType  reflect.Type
	method      bool
}

type FunctionNameMapping struct {
	name              string
	paramIndex        int // Todo: make this into a slice of param indexes for anonymous params
	paramType         reflect.Type
	required          bool
	anonymousArgument bool
}

// NewGraphFunctionWithNames creates a new graph function given a name, function,
// and an optional list of parameter names. The function provided must be of the
// type func and the names provided should match the count of parameters. It
// panics if the function is not a func type, the number of names doesn't match
// parameters, or if unsupported parameters like map are provided.
func NewGraphFunctionWithNames(name string, graphFunc reflect.Value, names ...string) GraphFunction {
	mft := graphFunc.Type()
	if mft.Kind() != reflect.Func {
		panic("graphFunc must be a func")
	}

	if !isValidGraphFunction(graphFunc, false) {
		panic("not valid graph function")
	}

	returnType, err := validateFunctionReturnTypes(mft)
	if err != nil {
		panic(fmt.Sprintf("invalid return type: %s", err.Error()))
	}

	// Check the parameters of the graphFunc. The count of names must match the count
	// of parameters. Note that context.Context is not a parameter that is counted.
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
		mode:        NamedParamsInline,
	}
	return gf
}

func isValidGraphFunction(graphFunc reflect.Value, method bool) bool {
	// A valid graph function must be a func type. It's inputs must be zero or more
	// serializable types. If it's a method, the first parameter must be a pointer to
	// a struct for the receiver. It may, optionally, take a context.Context
	// parameter. It must return a serializable type. It may also return an error.

	// Check the function type.
	mft := graphFunc.Type()
	fmt.Printf("Mft type: %s\n", mft.Kind())
	if mft.Kind() != reflect.Func {
		return false
	}

	// Check the parameters of the graphFunc. The first parameter may be a
	// context.Context. If it is, it is ignored for the purposes of the graph
	// function.
	for i := 0; i < mft.NumIn(); i++ {
		funcParam := mft.In(i)
		if funcParam.ConvertibleTo(contextType) {
			continue
		}

		if i == 0 && method {
			// The first parameter must be a pointer to a struct.
			// TODO: Validation
		} else {

			switch funcParam.Kind() {
			case reflect.Ptr:
				return true

			case reflect.Map:
				return false
			}
		}
	}

	// Check the return types of the graphFunc. It must return a serializable
	// type. It may also return an error.
	returnType, err := validateFunctionReturnTypes(mft)
	if err != nil {
		return false
	}
	if returnType == nil {
		return false
	}

	return true
}

func NewGraphFunction(name string, graphFunc any, method bool) GraphFunction {
	// This form of the graph function needs to be able to figure out the params
	// only from the function signature. This is tricky because Go doesn't have
	// named parameters. To get around this, we can operate in two modes:
	// 1) The function takes a single struct as a parameter. In this case, we
	//    can use reflection to get the names of the struct fields and use those
	//    as the parameter names.
	// 2) The function takes multiple parameters. In this case, we can use the
	//    types of the parameters and create anonymous arguments. This will make
	//    schema generation a bit janky, but it will work. This will cause the
	//    call to the function to ignore the names of the parameters and just
	//    pass them in order.
	//
	// In the case that there is a single parameter, we can decide which of these
	// schemes to use by checking is the parameter is a struct. If it is, and it
	// is a pointer to a struct, we will use the first option. Otherwise, we'll
	// use the second option.
	//
	// In either case, there can be an optional context.Context parameter as the
	// first parameter. This will be ignored for the purposes of the graph
	// function.

	var funcTyp reflect.Type
	var funcVal reflect.Value

	if rVal, ok := graphFunc.(reflect.Value); ok {
		funcVal = rVal
		funcTyp = funcVal.Type()
	} else {
		funcVal = reflect.ValueOf(graphFunc)
		funcTyp = funcVal.Type()
	}

	if funcTyp.Kind() != reflect.Func {
		panic("graphFunc must be a function")
	}

	startParam := 0
	if method {
		startParam = 1
	}
	// Gather the parameter types, ignoring the context.Context if it is
	// present.
	inputTypes := []reflect.Type{}
	for i := startParam; i < funcTyp.NumIn(); i++ {
		in := funcTyp.In(i)
		if in.ConvertibleTo(contextType) {
			// Skip this parameter if it is a context.Context.
			continue
		}
		inputTypes = append(inputTypes, in)
	}

	if len(inputTypes) == 0 {
		// This is fine -- this case is used primarily in result generation. If a field's
		// output is expensive to get, it can be hidden behind a function to ensure it's
		// only invoked if it is asked for.
		return newAnonymousGraphFunction(name, funcVal, inputTypes, method)
	} else if len(inputTypes) > 1 {
		// We are in the case where there are multiple parameters. We will use the
		// types of the parameters to create anonymous arguments.
		// Invoke option 2
		return newAnonymousGraphFunction(name, funcVal, inputTypes, method)
	} else {
		// A single parameter. We will use the name of the parameter if it is a
		// struct, otherwise we will use an anonymous argument.
		paramType := inputTypes[0]
		if paramType.Kind() == reflect.Struct {
			// Invoke option 1
			return newStructGraphFunction(name, funcVal, paramType, method)
		}
		return newAnonymousGraphFunction(name, funcVal, inputTypes, method)
	}
}

func newAnonymousGraphFunction(name string, graphFunc reflect.Value, types []reflect.Type, method bool) GraphFunction {
	// We are in the case where there are multiple parameters. We will use the
	// types of the parameters to create anonymous arguments. We won't have any named
	// parameters as we don't have any names to use.

	gf := GraphFunction{
		name:     name,
		mode:     AnonymousParamsInline,
		function: graphFunc,
		method:   method,
	}

	mft := graphFunc.Type()
	returnType, err := validateFunctionReturnTypes(mft)
	if err != nil {
		panic(err)
	}
	gf.returnType = returnType

	// Iterate over the parameters and create the anonymous arguments.
	anonymousArgs := []FunctionNameMapping{}
	for i, paramType := range types {
		mapping := FunctionNameMapping{
			name:              fmt.Sprintf("arg%d", i),
			paramIndex:        i,
			paramType:         paramType,
			anonymousArgument: true,
		}

		// If the field is a pointer, it is optional.
		if paramType.Kind() == reflect.Ptr {
			mapping.required = false
		} else {
			mapping.required = true
		}

		anonymousArgs = append(anonymousArgs, mapping)
	}

	return gf
}

func newStructGraphFunction(name string, graphFunc reflect.Value, paramType reflect.Type, method bool) GraphFunction {
	// We are in the case where there is a single struct parameter. We will use
	// the names of the struct fields as the parameter names.

	gf := GraphFunction{
		name:     name,
		mode:     NamedParamsStruct,
		function: graphFunc,
		method:   method,
	}

	mft := graphFunc.Type()
	returnType, err := validateFunctionReturnTypes(mft)
	if err != nil {
		panic(err)
	}
	gf.returnType = returnType

	// The parameter type must be a pointer to a struct. We will panic if it is
	// not.
	if paramType.Kind() != reflect.Struct {
		panic("paramType must a struct")
	}

	// Iterate over the fields of the struct and create the name mapping.
	nameMapping := map[string]FunctionNameMapping{}

	for i := 0; i < paramType.NumField(); i++ {
		field := paramType.Field(i)
		if field.Anonymous {
			// Todo: support anonymous fields
			panic("anonymous fields are not supported")
		}

		name := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			name = jsonTag
		}

		mapping := FunctionNameMapping{
			name:              name,
			paramIndex:        i,
			paramType:         field.Type,
			anonymousArgument: false,
		}

		// If the field is a pointer, it is optional.
		if field.Type.Kind() == reflect.Ptr {
			mapping.required = false
		} else {
			mapping.required = true
		}

		nameMapping[name] = mapping
	}

	gf.nameMapping = nameMapping

	return gf
}

// validateFunctionReturnTypes validates the return types of the function passed. It requires the function
// to have at least one non-error return value and at most one error return value. The function should have
// between one and two return values.
func validateFunctionReturnTypes(mft reflect.Type) (reflect.Type, error) {
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
		return nil, fmt.Errorf("mutatorFunc may have at most one error return value")
	}
	if nonErrorCount == 0 {
		return nil, fmt.Errorf("mutatorFunc must have at least one non-error return value")
	}
	return returnType, nil
}

// Call executes the graph function with a given context, request and command. It first prepares the
// parameters for the function call, then invokes the function and processes the results. If the function
// returns an error, it returns a formatted error. If the function returns no results, it returns nil.
func (f *GraphFunction) Call(ctx context.Context, req *Request, params *ParameterList, methodTarget reflect.Value) (reflect.Value, error) {

	paramValues, err := f.getCallParameters(ctx, req, params, methodTarget)
	if err != nil {
		return reflect.Value{}, err
	}

	gfv := f.function
	callResults := gfv.Call(paramValues)
	if len(callResults) == 0 {
		return reflect.Value{}, nil
	}

	var resultValue reflect.Value
	// TODO: Tighten this up to deal with the return types better.
	for _, callResult := range callResults {
		if callResult.CanConvert(errorType) {
			return reflect.Value{}, fmt.Errorf("error calling function: %v", callResult.Convert(errorType).Interface().(error))
		} else {
			resultValue = callResult
		}
	}

	return resultValue, nil
}

func (f *GraphFunction) GenerateResult(ctx context.Context, req *Request, obj reflect.Value, filter *ResultFilter) (any, error) {
	// Process the results
	return f.processCallOutput(ctx, req, filter, obj)
}
