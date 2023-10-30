package quickgraph

import (
	"context"
	"fmt"
	"github.com/alecthomas/participle/v2/lexer"
	"reflect"
	"runtime/debug"
	"strings"
)

type GraphFunctionParamType int

const (
	NamedParamsStruct GraphFunctionParamType = iota
	NamedParamsInline
	AnonymousParamsInline
)

type GraphFunctionMode int

const (
	ModeQuery GraphFunctionMode = iota
	ModeMutation
)

type FunctionDefinition struct {
	// Name is the name of the function.
	// This is used to map the function to the GraphQL query.
	Name string

	// Function is the function to call.
	Function any

	// ParameterNames is a list of parameter names to use for the function. Since reflection
	// doesn't provide parameter names, this is needed to map the parameters to the function.
	// This is used when the function can have anonymous parameters otherwise.
	ParameterNames []string

	// ReturnAnyOverride is a list of types that may be returned as `any` when returned from
	// the function. This is a function-specific override to the global `any` types that are
	// on the base Graphy instance.
	ReturnAnyOverride []any

	// Mode controls how the function is to be run. Mutators are functions that change the
	// state of the system. They will be called sequentially and in the order they are referred
	// to in the query. Regular queries are functions that do not change the state of the
	// system. They will be called in parallel.
	Mode GraphFunctionMode

	// ReturnEnumName is used to provide a custom name for implicit return unions. If this is
	// not set the default name is the name of the function followed by "ResultUnion".
	ReturnUnionName string
}

type graphFunction struct {
	// General information about the function.
	g        *Graphy
	name     string
	function reflect.Value
	method   bool

	// Input handling
	paramType    GraphFunctionParamType
	mode         GraphFunctionMode
	nameMapping  map[string]functionNameMapping
	indexMapping []functionNameMapping

	// Output handling
	baseReturnType *typeLookup
	rawReturnType  reflect.Type
}

type functionNameMapping struct {
	name              string
	paramIndex        int // Todo: make this into a slice of param indexes for anonymous params
	paramType         reflect.Type
	required          bool
	anonymousArgument bool
}

func (g *Graphy) validateGraphFunction(graphFunc reflect.Value, name string, method bool) error {
	// A valid graph function must be a func type. It's inputs must be zero or more
	// serializable types. If it's a method, the first parameter must be a pointer to
	// a struct for the receiver. It may, optionally, take a context.Context
	// parameter. It must return a serializable type. It may also return an error.

	// Check the function type.
	mft := graphFunc.Type()
	if mft.Kind() != reflect.Func {
		return fmt.Errorf("function %s is not a func: %v", name, mft)
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
			continue
		} else {
			logicalParamNumber := i
			if method {
				logicalParamNumber--
			}
			switch funcParam.Kind() {
			case reflect.Map:
				return fmt.Errorf("function %s has a parameter %d of type map, which is not supported", name, logicalParamNumber)

			case reflect.Interface:
				return fmt.Errorf("function %s has a parameter %d of type interface, which is not supported", name, logicalParamNumber)
			}
		}
	}

	// Check the return types of the graphFunc. It must return a serializable
	// type. It may also return an error.
	_, err := g.validateFunctionReturnTypes(mft, FunctionDefinition{Name: name})
	if err != nil {
		return err
	}

	return nil
}

func (g *Graphy) newGraphFunction(def FunctionDefinition, method bool) graphFunction {
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

	// Todo: This feels awkward. Is there a better way to do this? This is called from
	//  both registering functions directly, as well we from registering methods on the
	//  output types.
	if rVal, ok := def.Function.(reflect.Value); ok {
		funcVal = rVal
		funcTyp = funcVal.Type()
	} else {
		funcVal = reflect.ValueOf(def.Function)
		funcTyp = funcVal.Type()
	}

	err := g.validateGraphFunction(funcVal, def.Name, method)
	if err != nil {
		panic("not valid graph function: " + err.Error())
	}

	startParam := 0
	if method {
		startParam = 1
	}
	// Gather the parameter types, ignoring the context.Context if it is
	// present.
	var inputTypes []functionNameMapping

	for i := startParam; i < funcTyp.NumIn(); i++ {
		in := funcTyp.In(i)
		if in.ConvertibleTo(contextType) {
			// Skip this parameter if it is a context.Context.
			continue
		}
		fnm := functionNameMapping{
			paramIndex: i,
			paramType:  in,
		}
		inputTypes = append(inputTypes, fnm)
	}

	if len(inputTypes) == 0 {
		// This is fine -- this case is used primarily in result generation. If a field's
		// output is expensive to get, it can be hidden behind a function to ensure it's
		// only invoked if it is asked for.
		return g.newAnonymousGraphFunction(def, funcVal, inputTypes, method)
	} else if len(inputTypes) > 1 {
		// We are in the case where there are multiple parameters. We will use the
		// types of the parameters to create anonymous arguments.
		// Invoke option 2
		return g.newAnonymousGraphFunction(def, funcVal, inputTypes, method)
	} else {
		// A single parameter. We will use the name of the parameter if it is a
		// struct, otherwise we will use an anonymous argument.
		paramType := inputTypes[0].paramType
		if paramType.Kind() == reflect.Struct && len(def.ParameterNames) == 0 {
			// Invoke option 1
			return g.newStructGraphFunction(def, funcVal, paramType, method)
		}
		return g.newAnonymousGraphFunction(def, funcVal, inputTypes, method)
	}
}

func (g *Graphy) newAnonymousGraphFunction(def FunctionDefinition, graphFunc reflect.Value, inputs []functionNameMapping, method bool) graphFunction {
	// We are in the case where there are multiple parameters. We will use the
	// types of the parameters to create anonymous arguments. We won't have any named
	// parameters as we don't have any names to use.

	gf := graphFunction{
		g:           g,
		name:        def.Name,
		mode:        def.Mode,
		function:    graphFunc,
		method:      method,
		nameMapping: map[string]functionNameMapping{},
	}

	if len(def.ParameterNames) > 0 {
		gf.paramType = NamedParamsInline
	} else {
		gf.paramType = AnonymousParamsInline
	}

	mft := graphFunc.Type()
	returnType, err := g.validateFunctionReturnTypes(mft, def)
	if err != nil {
		panic(err)
	}
	if returnType.typ == anyType && len(def.ReturnAnyOverride) > 0 {
		gf.baseReturnType = g.convertAnySlice(def.ReturnAnyOverride)
		// We need special handling for the `any` type later.
		gf.rawReturnType = returnType.typ
	} else {
		gf.baseReturnType = returnType
		gf.rawReturnType = returnType.typ
	}

	hasNames := false
	if len(def.ParameterNames) > 0 {
		if len(def.ParameterNames) != len(inputs) {
			panic("parameter names count must match parameter count")
		}
		hasNames = true
	} else {
		gf.indexMapping = make([]functionNameMapping, len(inputs))
	}

	// Iterate over the parameters and create the anonymous arguments.
	for i, mapping := range inputs {
		mapping := mapping

		// If the field is a pointer, it is optional.
		if mapping.paramType.Kind() == reflect.Ptr {
			mapping.required = false
		} else {
			mapping.required = true
		}

		if hasNames {
			mapping.name = def.ParameterNames[i]
			mapping.anonymousArgument = false
			gf.nameMapping[def.ParameterNames[i]] = mapping
		} else {
			mapping.name = fmt.Sprintf("arg%d", mapping.paramIndex)
			mapping.anonymousArgument = true
			gf.nameMapping[mapping.name] = mapping
			gf.indexMapping[i] = mapping
		}
	}

	return gf
}

func (g *Graphy) newStructGraphFunction(def FunctionDefinition, graphFunc reflect.Value, paramType reflect.Type, method bool) graphFunction {
	// We are in the case where there is a single struct parameter. We will use
	// the names of the struct fields as the parameter names.

	gf := graphFunction{
		g:         g,
		name:      def.Name,
		paramType: NamedParamsStruct,
		mode:      def.Mode,
		function:  graphFunc,
		method:    method,
	}

	mft := graphFunc.Type()
	returnType, err := g.validateFunctionReturnTypes(mft, def)
	if err != nil {
		panic(err)
	}
	if returnType.typ == anyType && len(def.ReturnAnyOverride) > 0 {
		gf.baseReturnType = g.convertAnySlice(def.ReturnAnyOverride)
		gf.rawReturnType = returnType.typ
	} else {
		gf.baseReturnType = returnType
		gf.rawReturnType = returnType.typ
	}

	if paramType.Kind() != reflect.Struct {
		// We should never get here because the upstream code should have already
		// checked this and wouldn't have called this function if it wasn't a
		// struct.
		panic("paramType must a struct")
	}

	// Iterate over the fields of the struct and create the name mapping.
	nameMapping := map[string]functionNameMapping{}

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

		mapping := functionNameMapping{
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

func (g *Graphy) convertAnySlice(types []any) *typeLookup {
	result := &typeLookup{
		fields:              make(map[string]fieldLookup),
		fieldsLowercase:     map[string]fieldLookup{},
		implements:          map[string]*typeLookup{},
		implementsLowercase: map[string]*typeLookup{},
		union:               map[string]*typeLookup{},
		unionLowercase:      map[string]*typeLookup{},
	}
	for _, typ := range types {
		at := g.typeLookup(reflect.TypeOf(typ))
		result.union[at.name] = at
		result.unionLowercase[strings.ToLower(at.name)] = at
	}
	return result
}

// validateFunctionReturnTypes validates the return types of the function passed. It requires the function
// to have at least one non-error return value and at most one error return value. The function should have
// between one and two return values.
func (g *Graphy) validateFunctionReturnTypes(mft reflect.Type, definition FunctionDefinition) (*typeLookup, error) {
	errorCount := 0

	nonPointerCount := 0
	var returnTypes []reflect.Type

	for i := 0; i < mft.NumOut(); i++ {
		out := mft.Out(i)
		if out.ConvertibleTo(errorType) {
			errorCount++
		} else {
			returnTypes = append(returnTypes, out)
			if out.Kind() != reflect.Ptr {
				nonPointerCount++
			}
		}
	}

	if errorCount > 1 {
		return nil, fmt.Errorf("function may have at most one error return value")
	}
	if len(returnTypes) == 0 {
		return nil, fmt.Errorf("function must have at least one non-error return value")
	}
	if len(returnTypes) == 1 {
		// This is the simple case where we have a single return type.
		return g.typeLookup(returnTypes[0]), nil
	}
	if nonPointerCount > 1 {
		return nil, fmt.Errorf("function may have at most one non-pointer return value")
	}

	// If we have multiple return types, we're in the implicit union case.
	// We need to create a union type for the return types.
	var unionName string
	if definition.ReturnUnionName != "" {
		unionName = definition.ReturnUnionName
	} else {
		unionName = definition.Name + "ResultUnion"
	}
	result := &typeLookup{
		name:                unionName,
		fields:              make(map[string]fieldLookup),
		fieldsLowercase:     make(map[string]fieldLookup),
		implements:          make(map[string]*typeLookup),
		implementsLowercase: make(map[string]*typeLookup),
		union:               make(map[string]*typeLookup),
		unionLowercase:      make(map[string]*typeLookup),
	}
	for _, returnType := range returnTypes {
		tl := g.typeLookup(returnType)
		result.union[tl.name] = tl
		result.unionLowercase[strings.ToLower(tl.name)] = tl
	}
	return result, nil
}

// Call executes the graph function with a given context, request and command. It first prepares the
// parameters for the function call, then invokes the function and processes the results. If the function
// returns an error, it returns a formatted error. If the function returns no results, it returns nil.
func (f *graphFunction) Call(ctx context.Context, req *request, params *parameterList, methodTarget reflect.Value) (val reflect.Value, retErr error) {
	// Catch panics and return them as errors.
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			val = reflect.Value{}
			var pos lexer.Position
			if params != nil {
				pos = params.Pos
			} else {
				pos = lexer.Position{}
			}
			gErr := NewGraphError(fmt.Sprintf("function %s panicked: %v", f.name, r), pos)
			gErr.AddExtension("stack", stack)
			retErr = gErr
		}
	}()

	paramValues, err := f.getCallParameters(ctx, req, params, methodTarget)
	if err != nil {
		var pos lexer.Position
		if params != nil {
			pos = params.Pos
		} else {
			pos = lexer.Position{}
		}
		return reflect.Value{}, AugmentGraphError(err, fmt.Sprintf("error getting call parameters for function %s", f.name), pos)
	}

	gfv := f.function
	callResults := gfv.Call(paramValues)
	if len(callResults) == 0 {
		// We should never get here because all functions must return at least one value and an optional error.
		return reflect.Value{}, NewGraphError("function returned no values", params.Pos, f.name)
	}

	var resultValues []reflect.Value
	for _, callResult := range callResults {
		if callResult.CanConvert(errorType) {
			if !callResult.IsNil() {
				err := callResult.Convert(errorType).Interface().(error)
				return reflect.Value{}, AugmentGraphError(err, fmt.Sprintf("function %s returned error", f.name), params.Pos)
			}
		} else {
			resultValues = append(resultValues, callResult)
		}
	}

	if len(resultValues) == 1 {
		return resultValues[0], nil
	}

	// At this point, we are in the implicit union case. We need to return the single non-nil result
	// value from the results. If we have zero or more than one non-nil result value, that is an error.
	// Otherwise, return the single non-nil result value.
	var nonNilResult reflect.Value
	for _, resultValue := range resultValues {
		if !resultValue.IsNil() {
			if nonNilResult.IsValid() {
				return reflect.Value{}, NewGraphError(fmt.Sprintf("function %s returned multiple non-nil values", f.name), params.Pos)
			}
			nonNilResult = resultValue
		}
	}
	if !nonNilResult.IsValid() {
		return reflect.Value{}, NewGraphError(fmt.Sprintf("function %s returned no non-nil values", f.name), params.Pos)
	}
	return nonNilResult, nil
}

func (f *graphFunction) GenerateResult(ctx context.Context, req *request, obj reflect.Value, filter *resultFilter) (any, error) {
	// Process the results
	return f.processCallOutput(ctx, req, filter, obj)
}

func (f *graphFunction) receiverValueForFunction(target reflect.Value) reflect.Value {
	if !f.method {
		panic("receiverValueForFunction called on non-method")
	}

	receiverType := f.function.Type().In(0)

	// This is an odd case -- in all circumstances, the type of the receiver is a pointer.
	// The way we dereference the value that is passed in is always a struct. Even through
	// detailed testing, there seems to be no cases where anything other than the first
	// case is needed. However, it's plausible that there are cases where the other cases
	// are needed. Those cases have not been found. See the TestGraphFunction_MethodCall
	// test case for the full matrix of cases that are tested.
	if receiverType.Kind() == reflect.Ptr && target.Kind() != reflect.Ptr {
		// Make a new pointer to the target.
		ptrElem := reflect.New(target.Type())
		ptrElem.Elem().Set(target)
		return ptrElem
	} else if receiverType.Kind() == reflect.Struct && target.Kind() == reflect.Ptr {
		return target.Elem()
	} else if receiverType.Kind() == target.Kind() {
		return target
	}
	panic("receiverValueForFunction called with incompatible receiver type")
}
