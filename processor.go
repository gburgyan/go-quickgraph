package quickgraph

import (
	"context"
	"fmt"
	"reflect"
	"strings"
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

type TypeFieldLookup struct {
	name         string
	typ          reflect.Type
	fieldIndexes []int
}

func MakeTypeFieldLookup(typ reflect.Type) map[string]TypeFieldLookup {
	// Do a depth-first search of the type to find all of the fields.
	// Include the anonymous fields in this search and treat them as if
	// they were part of the current type in a flattened manner.
	result := map[string]TypeFieldLookup{}
	processFieldLookup(typ, nil, result)
	return result
}

func processFieldLookup(typ reflect.Type, prevIndex []int, result map[string]TypeFieldLookup) {
	for i := 0; i < typ.NumField(); i++ {

		field := typ.Field(i)
		// If there's a json tag on the field, use that for the name of the field.
		// Otherwise, use the name of the field.
		// If there's a json tag with a "-" value, ignore the field.
		// If there's a json tag with a "omitempty" value, ignore the field.
		fieldName := field.Name
		jsonTag := field.Tag.Get("json")
		if jsonTag != "" {
			jsonParts := strings.Split(jsonTag, ",")
			if jsonParts[0] == "-" {
				continue
			}
			if jsonParts[0] != "" {
				fieldName = jsonParts[0]
			}
		}

		// If we already have a field with that name, ignore it.
		if _, ok := result[fieldName]; ok {
			continue
		}
		index := append(prevIndex, i)
		if field.Anonymous {
			processFieldLookup(field.Type, index, result)
		} else {
			tfl := TypeFieldLookup{
				name:         field.Name,
				typ:          field.Type,
				fieldIndexes: index,
			}
			result[field.Name] = tfl
		}
	}
}

func (t *TypeFieldLookup) Fetch(v reflect.Value) (any, error) {
	for _, i := range t.fieldIndexes {
		v = v.Field(i)
	}
	return v.Interface(), nil
}

func NewGraphFunction(name string, mutatorFunc any, names ...string) GraphFunction {
	mft := reflect.TypeOf(mutatorFunc)
	if mft.Kind() != reflect.Func {
		panic("mutatorFunc must be a func")
	}

	returnType := validateFunctionReturnTypes(mft)

	// Check the parameters of the mutatorFunc. The count of names must match the count of parameters.
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
		function:    mutatorFunc,
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

func (f *GraphFunction) getCallParameters(ctx context.Context, req *Request, command Command) ([]reflect.Value, error) {
	gft := reflect.TypeOf(f.function)

	// Make something to hold the parameters
	paramValues := make([]reflect.Value, gft.NumIn())

	// Go through all the input parameters and populate the values. If it's a context.Context,
	// use the context from the call.
	for i := 0; i < gft.NumIn(); i++ {
		if gft.In(i).ConvertibleTo(contextType) {
			paramValues[i] = reflect.ValueOf(ctx)
			continue
		}
	}

	// Make a map of the parameters that are required
	requiredParams := map[string]bool{}
	for _, nameMapping := range f.nameMapping {
		if nameMapping.required {
			requiredParams[nameMapping.name] = true
		}
	}

	parsedParams := command.Parameters
	for _, param := range parsedParams.Values {
		if nameMapping, ok := f.nameMapping[param.Name]; ok {
			val := reflect.New(nameMapping.paramType).Elem()
			parseMappingIntoValue(req, param.Value, val)
			paramValues[nameMapping.paramIndex] = val
			delete(requiredParams, param.Name)
		}
	}
	if len(requiredParams) > 0 {
		missingParams := []string{}
		for paramName := range requiredParams {
			missingParams = append(missingParams, paramName)
		}
		return nil, fmt.Errorf("missing required parameters: %v", strings.Join(missingParams, ", "))
	}
	return paramValues, nil
}

func processStruct(filter *ResultFilter, anyStruct any) (map[string]any, error) {
	r := map[string]any{}

	// If the anyStruct is a pointer, dereference it.
	if reflect.TypeOf(anyStruct).Kind() == reflect.Ptr {
		if reflect.ValueOf(anyStruct).IsNil() {
			return nil, nil
		}
		anyStruct = reflect.ValueOf(anyStruct).Elem().Interface()
	}

	kind := reflect.TypeOf(anyStruct).Kind()
	if kind == reflect.Map && reflect.ValueOf(anyStruct).IsNil() {
		return nil, nil
	}

	anyStruct, err := deferenceUnionType(anyStruct)
	if err != nil {
		return nil, err
	}

	// Create map of field names, as specified by the json tag or field name, to index.
	// This is used to map the fields in the struct to the fields in the result.
	// TODO: Cache this.
	t := reflect.TypeOf(anyStruct)
	typeName := t.Name()
	fieldMap := MakeTypeFieldLookup(t)

	fieldsToProcess := []ResultField{}
	for _, field := range filter.Fields {
		fieldsToProcess = append(fieldsToProcess, field)
	}
	for _, union := range filter.UnionLookup {
		if union.TypeName == typeName {
			for _, field := range union.Fields.Fields {
				fieldsToProcess = append(fieldsToProcess, field)
			}
		}
	}

	// Go through the result fields and map them to the struct fields.
	for _, field := range fieldsToProcess {
		if field.Params != nil {
			// TODO: Deal with parameterized fields.
		} else if field.Name == "__typename" {
			r[field.Name] = typeName
		} else {
			if fieldInfo, ok := fieldMap[field.Name]; ok {
				fieldVal, err := fieldInfo.Fetch(reflect.ValueOf(anyStruct))
				if err != nil {
					return nil, err
				}
				if field.SubParts != nil {
					subPart, err := processStruct(field.SubParts, fieldVal)
					if err != nil {
						return nil, err
					}
					r[field.Name] = subPart
				} else {
					r[field.Name] = fieldVal
				}
			} else {
				// TODO: Is this an error?
			}
		}
	}

	return r, nil
}

func deferenceUnionType(anyStruct any) (any, error) {
	// If the anyStruct is a union type, as indicated by its name ending in "Union", then
	// we need to get the actual type of the struct. We do this by finding the field that is
	// not nil. The expectation is that there will be only one field that is not nil. Further
	// the fields must be pointers so we can check for nil.
	if strings.HasSuffix(reflect.TypeOf(anyStruct).Name(), "Union") {
		// Find the field that is not nil.
		t := reflect.TypeOf(anyStruct)
		v := reflect.ValueOf(anyStruct)
		found := false
		for i := 0; i < t.NumField(); i++ {
			switch v.Field(i).Kind() {
			case reflect.Map, reflect.Pointer, reflect.Interface, reflect.Slice:
				break

			default:
				return nil, fmt.Errorf("fields in union type must be pointers, maps, slices, or interfaces")
			}
			if v.Field(i).IsNil() {
				continue
			}
			if found {
				return nil, fmt.Errorf("more than one field in union type is not nil")
			}
			anyStruct = v.Field(i).Elem().Interface()
			found = true
		}
		if !found {
			return nil, fmt.Errorf("no fields in union type are not nil")
		}
	}
	return anyStruct, nil
}

func parseMappingIntoValue(req *Request, inValue GenericValue, targetValue reflect.Value) {
	// TODO: Better error detection and handling.
	if inValue.Variable != nil {
		// Strip the $ from the variable name.
		variableName := (*inValue.Variable)[1:]
		value := req.Variables[variableName]
		targetValue.Set(value)
	} else if inValue.String != nil {
		// The string value has quotes around it, remove them.
		literalValue := (*inValue.String)[1 : len(*inValue.String)-1]
		if targetValue.Kind() == reflect.Ptr {
			// Create a pointer to the target type and set the value.
			ttp := targetValue.Type()
			tt := ttp.Elem()
			val := reflect.New(tt)
			val.Elem().SetString(literalValue)
			targetValue.Set(val)
		} else {
			targetValue.SetString(literalValue)
		}
	} else if inValue.List != nil {
		targetType := targetValue.Type()
		targetValue.Set(reflect.MakeSlice(targetType, len(inValue.List), len(inValue.List)))
		for i, listItem := range inValue.List {
			parseMappingIntoValue(req, listItem, targetValue.Index(i))
		}
	} else if inValue.Map != nil {
		targetType := targetValue.Type()
		// TODO: Cache this so we don't have to reconstruct it ever time.
		// Loop through the fields of the target type and make a map of the fields by "json" tag.
		fieldMap := map[string]reflect.StructField{}
		requiredFields := map[string]bool{}
		for i := 0; i < targetType.NumField(); i++ {
			field := targetType.Field(i)
			if tag, ok := field.Tag.Lookup("json"); ok {
				fieldMap[tag] = field
			}
			if field.Type.Kind() != reflect.Ptr {
				requiredFields[field.Name] = true
			}
		}

		for _, namedValue := range inValue.Map {
			var fieldValue reflect.Value
			var fieldName string
			if targetField, ok := fieldMap[namedValue.Name]; ok {
				fieldValue = targetValue.FieldByName(targetField.Name)
				fieldName = targetField.Name
			}
			if fieldValue.Kind() == 0 {
				fieldValue = targetValue.FieldByName(namedValue.Name)
				fieldName = namedValue.Name
			}
			if fieldValue.Kind() != 0 {
				parseMappingIntoValue(req, namedValue.Value, fieldValue)
				delete(requiredFields, fieldName)
			} else {
				// TODO: Handle this better. Warning? Error?
			}
		}

		if len(requiredFields) > 0 {
			missingFields := strings.Builder{}
			for fieldName := range requiredFields {
				if missingFields.Len() > 0 {
					missingFields.WriteString(", ")
				}
				missingFields.WriteString(fieldName)
			}
			panic("missing required fields: " + missingFields.String())
		}
	} else {
		panic("not implemented")
	}
}
