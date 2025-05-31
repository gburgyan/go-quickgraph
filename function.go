package quickgraph

import (
	"context"
	"fmt"
	"github.com/alecthomas/participle/v2/lexer"
	"reflect"
	"runtime/debug"
	"strings"
)

// ParameterMode defines how function parameters are handled
type ParameterMode int

const (
	// AutoDetect maintains backward compatibility by automatically detecting parameter mode
	AutoDetect ParameterMode = iota

	// StructParams explicitly uses a struct for parameters
	StructParams

	// NamedParams explicitly uses named inline parameters
	NamedParams

	// PositionalParams explicitly uses positional parameters (no names required)
	PositionalParams
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
	ModeSubscription
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

	// ParameterMode explicitly defines how parameters should be handled.
	// If not set (AutoDetect), the mode is inferred from the function signature.
	ParameterMode ParameterMode

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

	// Description is used to provide a description for the function. This will be used in the
	// schema.
	Description *string

	// DeprecatedReason is used to mark a function as deprecated. This will cause the function to
	// be marked as deprecated in the schema.
	DeprecatedReason *string
}

type graphFunction struct {
	// General information about the function.
	g        *Graphy
	name     string
	function reflect.Value
	method   bool

	// Input handling
	paramType     GraphFunctionParamType
	mode          GraphFunctionMode
	paramsByName  map[string]functionParamNameMapping
	paramsByIndex []functionParamNameMapping

	// Output handling
	baseReturnType *typeLookup
	rawReturnType  reflect.Type
	isSubscription bool
	channelType    reflect.Type // For subscriptions, the channel type returned by the function
}

type functionParamNameMapping struct {
	name              string
	paramIndex        int   // For struct params: index of the top-level field
	fieldPath         []int // For embedded fields: path to navigate to the actual field
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
				// Allow maps that are registered as custom scalars (e.g., JSON scalar)
				if _, exists := g.GetScalarByType(funcParam); !exists {
					return fmt.Errorf("function %s has a parameter %d of type map, which is not supported (unless registered as a custom scalar)", name, logicalParamNumber)
				}

			case reflect.Interface:
				// Allow interfaces that are registered as custom scalars
				if _, exists := g.GetScalarByType(funcParam); !exists {
					return fmt.Errorf("function %s has a parameter %d of type interface, which is not supported (unless registered as a custom scalar)", name, logicalParamNumber)
				}
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
	var inputTypes []functionParamNameMapping

	for i := startParam; i < funcTyp.NumIn(); i++ {
		in := funcTyp.In(i)
		if in.ConvertibleTo(contextType) {
			// Skip this parameter if it is a context.Context.
			continue
		}
		fnm := functionParamNameMapping{
			paramIndex: i,
			paramType:  in,
		}
		inputTypes = append(inputTypes, fnm)
	}

	// Check explicit parameter mode first
	switch def.ParameterMode {
	case StructParams:
		// Validate we have exactly one struct parameter
		if len(inputTypes) != 1 {
			panic(fmt.Sprintf("StructParams mode requires exactly one non-context parameter, got %d", len(inputTypes)))
		}
		if inputTypes[0].paramType.Kind() != reflect.Struct {
			panic(fmt.Sprintf("StructParams mode requires a struct parameter, got %s", inputTypes[0].paramType.Kind()))
		}
		return g.newStructGraphFunction(def, funcVal, inputTypes[0].paramType, method)

	case NamedParams:
		// Validate we have parameter names
		if len(def.ParameterNames) == 0 {
			panic("NamedParams mode requires ParameterNames to be set")
		}
		if len(def.ParameterNames) != len(inputTypes) {
			panic(fmt.Sprintf("NamedParams mode requires %d parameter names, got %d", len(inputTypes), len(def.ParameterNames)))
		}
		return g.newAnonymousGraphFunction(def, funcVal, inputTypes, method)

	case PositionalParams:
		// Validate no parameter names are set
		if len(def.ParameterNames) > 0 {
			panic("PositionalParams mode cannot be used with ParameterNames - parameters are positional only")
		}
		return g.newAnonymousGraphFunction(def, funcVal, inputTypes, method)

	case AutoDetect:
		// Fall through to original auto-detection logic
	default:
		panic(fmt.Sprintf("unknown ParameterMode: %d", def.ParameterMode))
	}

	// Original auto-detection logic for backward compatibility
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

func (g *Graphy) newAnonymousGraphFunction(def FunctionDefinition, graphFunc reflect.Value, inputs []functionParamNameMapping, method bool) graphFunction {
	// We are in the case where there are multiple parameters. We will use the
	// types of the parameters to create anonymous arguments. We won't have any named
	// parameters as we don't have any names to use.

	gf := graphFunction{
		g:              g,
		name:           def.Name,
		mode:           def.Mode,
		function:       graphFunc,
		method:         method,
		paramsByName:   map[string]functionParamNameMapping{},
		isSubscription: def.Mode == ModeSubscription,
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

	// For subscriptions, store the channel type
	if def.Mode == ModeSubscription && mft.NumOut() > 0 {
		for i := 0; i < mft.NumOut(); i++ {
			out := mft.Out(i)
			if out.Kind() == reflect.Chan {
				gf.channelType = out
				break
			}
		}
	}

	if returnType.typ == anyType && len(def.ReturnAnyOverride) > 0 {
		gf.baseReturnType = g.createImplicitTypeLookupUnion(unionNameGenerator(def), def.ReturnAnyOverride)
		// We need special handling for the `any` type later.
		gf.rawReturnType = returnType.typ
	} else {
		gf.baseReturnType = returnType
		gf.rawReturnType = returnType.typ
	}

	hasNames := false
	gf.paramsByIndex = make([]functionParamNameMapping, len(inputs))
	if len(def.ParameterNames) > 0 {
		if len(def.ParameterNames) != len(inputs) {
			panic("parameter names count must match parameter count")
		}
		hasNames = true
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
			gf.paramsByName[def.ParameterNames[i]] = mapping
			gf.paramsByIndex[i] = mapping
		} else {
			mapping.name = fmt.Sprintf("arg%d", mapping.paramIndex)
			mapping.anonymousArgument = true
			gf.paramsByName[mapping.name] = mapping
			gf.paramsByIndex[i] = mapping
		}
	}

	return gf
}

func (g *Graphy) newStructGraphFunction(def FunctionDefinition, graphFunc reflect.Value, paramType reflect.Type, method bool) graphFunction {
	// We are in the case where there is a single struct parameter. We will use
	// the names of the struct fields as the parameter names.

	gf := graphFunction{
		g:              g,
		name:           def.Name,
		paramType:      NamedParamsStruct,
		mode:           def.Mode,
		function:       graphFunc,
		method:         method,
		isSubscription: def.Mode == ModeSubscription,
	}

	mft := graphFunc.Type()
	// The error has already been checked earlier.
	returnType, _ := g.validateFunctionReturnTypes(mft, def)

	// For subscriptions, store the channel type
	if def.Mode == ModeSubscription && mft.NumOut() > 0 {
		for i := 0; i < mft.NumOut(); i++ {
			out := mft.Out(i)
			if out.Kind() == reflect.Chan {
				gf.channelType = out
				break
			}
		}
	}

	if returnType.typ == anyType && len(def.ReturnAnyOverride) > 0 {
		gf.baseReturnType = g.createImplicitTypeLookupUnion(unionNameGenerator(def), def.ReturnAnyOverride)
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
	nameMapping := map[string]functionParamNameMapping{}

	// Helper function to process a field and add it to nameMapping
	var processField func(field reflect.StructField, fieldIndex int, fieldPath []int)
	processField = func(field reflect.StructField, fieldIndex int, fieldPath []int) {
		name := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			if jsonTag == "-" {
				return
			}
			// Ignore anything after the first comma.
			name = strings.Split(jsonTag, ",")[0]
		}

		mapping := functionParamNameMapping{
			name:              name,
			paramIndex:        fieldIndex,
			fieldPath:         fieldPath,
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

	// Process all fields
	for i := 0; i < paramType.NumField(); i++ {
		field := paramType.Field(i)
		if field.Anonymous {
			// ANONYMOUS FIELD SUPPORT:
			// When we encounter an anonymous (embedded) field, we promote its fields
			// to the parent level for GraphQL arguments. This is similar to how Go
			// promotes methods from embedded types.
			//
			// Example input struct:
			//   type CommonInput struct {
			//       Limit  int
			//       Offset int
			//   }
			//   type SearchInput struct {
			//       CommonInput  // anonymous embedding
			//       Query string
			//   }
			//
			// In GraphQL, this allows: search(query: "test", limit: 10, offset: 0)
			// All fields from CommonInput are available as direct arguments.
			//
			// Note: We map promoted fields to the parent struct's field index,
			// so when populating the struct, we'll need to navigate to the embedded field.

			embeddedType := field.Type
			if embeddedType.Kind() == reflect.Ptr {
				embeddedType = embeddedType.Elem()
			}

			if embeddedType.Kind() == reflect.Struct {
				// Process fields of the embedded struct
				for j := 0; j < embeddedType.NumField(); j++ {
					embeddedField := embeddedType.Field(j)
					if embeddedField.Anonymous {
						// Skip nested anonymous fields for now
						// This could be supported recursively if needed
						continue
					}
					// For embedded fields, fieldPath contains [embeddedFieldIndex, fieldIndex]
					// This allows navigation: struct.Field(i).Field(j)
					processField(embeddedField, i, []int{j})
				}
			}
		} else {
			// Regular field, no navigation needed
			processField(field, i, nil)
		}
	}

	gf.paramsByName = nameMapping

	return gf
}

func (g *Graphy) createImplicitTypeLookupUnion(name string, types []any) *typeLookup {
	result := &typeLookup{
		name:                name,
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
		rt := returnTypes[0]

		// For subscriptions, validate that the return type is a channel
		if definition.Mode == ModeSubscription {
			if rt.Kind() != reflect.Chan {
				return nil, fmt.Errorf("subscription must return a channel, got %v", rt.Kind())
			}
			if rt.ChanDir() == reflect.SendDir {
				return nil, fmt.Errorf("subscription must return a receive-only or bidirectional channel")
			}
			// Get the element type of the channel
			elemType := rt.Elem()
			return g.typeLookup(elemType), nil
		}

		return g.typeLookup(rt), nil
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
			}
			gErr := NewGraphError(fmt.Sprintf("function %s panicked: %v", f.name, r), pos)
			gErr.AddExtension("stack", stack)
			retErr = gErr
		}
	}()

	// Get the parameters for the function call.
	// NOTE: params can be nil for field methods (methods accessed as fields without parentheses in GraphQL).
	// Example: "personalDetails" instead of "personalDetails()"
	// getCallParameters must handle this case correctly.
	paramValues, err := f.getCallParameters(ctx, req, params, methodTarget)
	if err != nil {
		var pos lexer.Position
		if params != nil {
			pos = params.Pos
		}
		return reflect.Value{}, AugmentGraphError(err, fmt.Sprintf("error getting call parameters for function %s", f.name), pos)
	}

	// PARAMETER VALIDATION:
	// Before calling the function, ensure all parameters are valid reflect.Values.
	// An invalid reflect.Value (created with reflect.Value{}) will cause a panic
	// when passed to Call(). This can happen if parameter initialization fails.
	//
	// This validation prevents the cryptic panic "reflect: Call using zero Value argument"
	// and provides a clearer error message indicating which parameter is invalid.
	for i, pv := range paramValues {
		if !pv.IsValid() {
			var pos lexer.Position
			if params != nil {
				pos = params.Pos
			}
			return reflect.Value{}, NewGraphError(fmt.Sprintf("invalid parameter at index %d for function %s", i, f.name), pos)
		}
	}

	gfv := f.function
	callResults := gfv.Call(paramValues)
	if len(callResults) == 0 {
		// We should never get here because all functions must return at least one value and an optional error.
		var pos lexer.Position
		if params != nil {
			pos = params.Pos
		}
		return reflect.Value{}, NewGraphError("function returned no values", pos, f.name)
	}

	var resultValues []reflect.Value
	for _, callResult := range callResults {
		if callResult.CanConvert(errorType) {
			if !callResult.IsNil() {
				err := callResult.Convert(errorType).Interface().(error)
				var pos lexer.Position
				if params != nil {
					pos = params.Pos
				}
				return reflect.Value{}, AugmentGraphError(err, fmt.Sprintf("function %s returned error", f.name), pos)
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
				var pos lexer.Position
				if params != nil {
					pos = params.Pos
				}
				return reflect.Value{}, NewGraphError(fmt.Sprintf("function %s returned multiple non-nil values", f.name), pos)
			}
			nonNilResult = resultValue
		}
	}
	if !nonNilResult.IsValid() {
		var pos lexer.Position
		if params != nil {
			pos = params.Pos
		}
		return reflect.Value{}, NewGraphError(fmt.Sprintf("function %s returned no non-nil values", f.name), pos)
	}
	return nonNilResult, nil
}

func (f *graphFunction) GenerateResult(ctx context.Context, req *request, obj reflect.Value, filter *resultFilter) (any, error) {
	// Process the results
	return f.processCallOutput(ctx, req, filter, obj)
}

// receiverValueForFunction converts the target value to the appropriate type for the method receiver.
// This handles the complexity of Go's method sets where a method might be defined on a pointer
// receiver but we have a value, or vice versa.
//
// EMBEDDED TYPE HANDLING:
// This function is particularly important for embedded types. When we navigate to an embedded
// field to call its method, we need to ensure the receiver type matches what the method expects.
// For example:
//   - Method defined on *Employee (pointer receiver)
//   - We have an Employee value from an embedded field
//   - We need to convert Employee to *Employee
//
// The function prioritizes using addressable values (via CanAddr) over creating new pointers,
// which preserves the connection to the original data structure.
func (f *graphFunction) receiverValueForFunction(target reflect.Value) reflect.Value {
	if !f.method {
		// There should be no way of getting here.
		panic("receiverValueForFunction called on non-method")
	}

	if !target.IsValid() {
		panic("receiverValueForFunction called with invalid target")
	}

	receiverType := f.function.Type().In(0)

	// This is an odd case -- in all circumstances, the type of the receiver is a pointer.
	// The way we dereference the value that is passed in is always a struct. Even through
	// detailed testing, there seems to be no cases where anything other than the first
	// case is needed. However, it's plausible that there are cases where the other cases
	// are needed. Those cases have not been found. See the TestGraphFunction_MethodCall
	// test case for the full matrix of cases that are tested.
	if receiverType.Kind() == reflect.Ptr && target.Kind() != reflect.Ptr {
		// Method expects pointer receiver but we have a value.
		// First try to take the address if the value is addressable.
		// This is preferable because it maintains the connection to the original struct.
		if target.CanAddr() {
			return target.Addr()
		}
		// If not addressable (e.g., value came from interface{} or map),
		// create a new pointer with a copy of the value.
		// Note: Changes made by the method won't affect the original in this case.
		ptrElem := reflect.New(target.Type())
		ptrElem.Elem().Set(target)
		return ptrElem
	} else if receiverType.Kind() == reflect.Struct && target.Kind() == reflect.Ptr {
		return target.Elem()
	} else if receiverType.Kind() == target.Kind() {
		return target
	}
	panic(fmt.Sprintf("receiverValueForFunction called with incompatible receiver type: want %v, got %v (target type: %v)",
		receiverType, target.Kind(), target.Type()))
}

func unionNameGenerator(def FunctionDefinition) string {
	if def.ReturnUnionName != "" {
		return def.ReturnUnionName
	} else {
		return def.Name + "ResultUnion"
	}
}
