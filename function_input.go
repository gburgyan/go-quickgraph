package quickgraph

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// getCallParameters returns the parameters to use when calling the function represented by this graphFunction.
// The parameters are returned as a slice of reflect.Value that can be used to call the function.
// The request and command are used to populate the parameters to the function.
func (f *graphFunction) getCallParameters(ctx context.Context, req *request, paramList *parameterList, target reflect.Value) ([]reflect.Value, error) {
	switch f.paramType {
	case NamedParamsInline:
		return f.getCallParamsNamedInline(ctx, req, paramList, target)

	case AnonymousParamsInline:
		return f.getCallParamsAnonymousInline(ctx, req, paramList, target)

	case NamedParamsStruct:
		return f.getCallParamsNamedStruct(ctx, req, paramList, target)
	}
	return nil, fmt.Errorf("unknown function paramType: %v", f.paramType)
}

func (f *graphFunction) getCallParamsNamedInline(ctx context.Context, req *request, params *parameterList, target reflect.Value) ([]reflect.Value, error) {
	gft := f.function.Type()

	// Make something to hold the parameters
	paramValues := make([]reflect.Value, gft.NumIn())

	startIndex := 0
	if f.method {
		// If this is a method, the first parameter is the receiver.
		paramValues[0] = f.receiverValueForFunction(target)
		startIndex = 1
	}

	// Go through all the input parameters and populate the values. If it's a context.Context,
	// use the context from the call.
	for i := startIndex; i < gft.NumIn(); i++ {
		if gft.In(i).ConvertibleTo(contextType) {
			paramValues[i] = reflect.ValueOf(ctx)
			continue
		}
	}

	// Make a map of the parameters that are required
	requiredParams := map[string]bool{}
	for _, nameMapping := range f.paramsByName {
		if nameMapping.required {
			requiredParams[nameMapping.name] = true
		}
	}

	parsedParams := params
	if parsedParams != nil {
		for _, param := range parsedParams.Values {
			if nameMapping, ok := f.paramsByName[param.Name]; ok {
				val := reflect.New(nameMapping.paramType).Elem()
				err := parseInputIntoValue(req, param.Value, val)
				if err != nil {
					return nil, err
				}
				paramValues[nameMapping.paramIndex] = val
				delete(requiredParams, param.Name)
			}
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

func (f *graphFunction) getCallParamsAnonymousInline(ctx context.Context, req *request, params *parameterList, target reflect.Value) ([]reflect.Value, error) {
	gft := f.function.Type()

	// Make something to hold the parameters
	paramValues := make([]reflect.Value, gft.NumIn())

	startIndex := 0
	if f.method {
		// If this is a method, the first parameter is the receiver.
		paramValues[0] = f.receiverValueForFunction(target)
		startIndex = 1
	}

	// Go through all the input parameters and populate the values. If it's a context.Context,
	// use the context from the call. Also keep a count of the number of non-context parameters
	// and fill in those values from the command.
	normalParamCount := 0
	for i := startIndex; i < gft.NumIn(); i++ {
		if gft.In(i).ConvertibleTo(contextType) {
			paramValues[i] = reflect.ValueOf(ctx)
			continue
		} else {
			// This is a normal parameter, fill it in from the command.
			val := reflect.New(gft.In(i)).Elem()
			paramValues[i] = val

			if params == nil {
				if val.Type().Kind() != reflect.Ptr {
					return nil, fmt.Errorf("missing parameter in function")
				}
			} else {
				if normalParamCount >= len(params.Values) {
					return nil, fmt.Errorf("too many parameters provided %d", normalParamCount)
				}
				err := parseInputIntoValue(req, params.Values[normalParamCount].Value, val)
				if err != nil {
					return nil, err
				}
			}

		}
		normalParamCount++
	}
	inParamCount := 0
	if params != nil {
		inParamCount = len(params.Values)
	}
	if normalParamCount < inParamCount {
		return nil, fmt.Errorf("too few parameters provided %d", normalParamCount)
	}

	return paramValues, nil
}

func (f *graphFunction) getCallParamsNamedStruct(ctx context.Context, req *request, params *parameterList, target reflect.Value) ([]reflect.Value, error) {
	gft := f.function.Type()

	// Make something to hold the parameters
	paramValues := make([]reflect.Value, gft.NumIn())

	startIndex := 0
	if f.method {
		// If this is a method, the first parameter is the receiver.
		paramValues[0] = f.receiverValueForFunction(target)
		startIndex = 1
	}

	// Go through all the input parameters and populate the values. If it's a context.Context,
	// use the context from the call. Grab the non-context parameter (only one based on the type)
	// and save that for later.
	var valueParam reflect.Value
	for i := startIndex; i < gft.NumIn(); i++ {
		if gft.In(i).ConvertibleTo(contextType) {
			paramValues[i] = reflect.ValueOf(ctx)
			continue
		} else if gft.In(i).Kind() == reflect.Struct {
			// This is the value parameter, save it for later.
			valueParam = reflect.New(gft.In(i)).Elem()
			paramValues[i] = valueParam
			continue
		}
		panic(fmt.Errorf("invalid parameter type %v", gft.In(i)))
	}

	// Make a map of the parameters that are required
	requiredParams := map[string]bool{}
	for _, nameMapping := range f.paramsByName {
		if nameMapping.required {
			requiredParams[nameMapping.name] = true
		}
	}

	parsedParams := params
	if parsedParams != nil {
		for _, param := range parsedParams.Values {
			if nameMapping, ok := f.paramsByName[param.Name]; ok {
				err := parseInputIntoValue(req, param.Value, valueParam.Field(nameMapping.paramIndex))
				if err != nil {
					return nil, err
				}
				delete(requiredParams, param.Name)
			}
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

// parseInputIntoValue interprets a genericValue according to the type of the targetValue and assigns the result to targetValue.
// This method takes into account various types of input such as string, int, float, list, map, identifier, and GraphQL variable.
// It returns an error if the input cannot be parsed into the target type.
func parseInputIntoValue(req *request, inValue genericValue, targetValue reflect.Value) (err error) {
	// Catch panics and return them as errors.
	defer func() {
		if r := recover(); r != nil {
			e := fmt.Errorf("panic: %v", r)
			err = AugmentGraphError(e, "", inValue.Pos)
		}
	}()

	typ := targetValue.Type()
	isPtr := typ.Kind() == reflect.Ptr
	if isPtr {
		typ = typ.Elem()
	}
	isSlice := typ.Kind() == reflect.Slice
	isStruct := typ.Kind() == reflect.Struct
	if inValue.Variable != nil {
		if req == nil {
			return fmt.Errorf("variable %s provided but no request", *inValue.Variable)
		}
		// Strip the $ from the variable name.
		variableName := (*inValue.Variable)[1:]
		err = parseVariableIntoValue(req, variableName, targetValue)
	} else if inValue.String != nil {
		// The string value has quotes around it, remove them.
		literalValue := (*inValue.String)[1 : len(*inValue.String)-1]
		parseStringIntoValue(literalValue, targetValue)
	} else if inValue.Identifier != nil {
		// This is where we handle enums. We have to look up the value based on the field.
		// This will only work with enums that are strings.
		err = parseIdentifierIntoValue(*inValue.Identifier, targetValue)
	} else if inValue.Int != nil {
		i := *inValue.Int
		parseIntIntoValue(i, targetValue)
	} else if inValue.Float != nil {
		f := *inValue.Float
		parseFloatIntoValue(f, targetValue)
	} else if inValue.List != nil || isSlice {
		err = parseListIntoValue(req, inValue, targetValue)
	} else if inValue.Map != nil || isStruct {
		err = parseMapIntoValue(req, inValue, targetValue)
	} else {
		// This should never occur as this should be a parse error
		// that gets caught by the parser.
		return fmt.Errorf("no input found to parse into value")
	}
	if err != nil {
		return err
	}

	return nil
}

// parseVariableIntoValue extracts the value of a variable from the provided request and assigns it to targetValue.
func parseVariableIntoValue(req *request, variableName string, targetValue reflect.Value) error {
	value, ok := req.variables[variableName]
	if !ok {
		return fmt.Errorf("variable %v not found", variableName)
	}
	targetValue.Set(value)
	return nil
}

// parseStringIntoValue interprets the provided string and assigns it to targetValue.
func parseStringIntoValue(s string, targetValue reflect.Value) {
	if targetValue.Kind() == reflect.Ptr {
		// Create a pointer to the target type and set the value.
		ttp := targetValue.Type()
		tt := ttp.Elem()
		val := reflect.New(tt)
		val.Elem().SetString(s)
		targetValue.Set(val)
	} else {
		targetValue.SetString(s)
	}
}

// parseIntIntoValue converts an int64 to the appropriate type and assigns it to targetValue.
func parseIntIntoValue(i int64, targetValue reflect.Value) {
	if targetValue.Kind() == reflect.Ptr {
		// Create a pointer to the target type and set the value.
		ttp := targetValue.Type()
		tt := ttp.Elem()
		val := reflect.New(tt)
		val.Elem().SetInt(i)
		targetValue.Set(val)
	} else {
		targetValue.SetInt(i)
	}
}

// parseFloatIntoValue converts a float64 to the appropriate type and assigns it to targetValue.
func parseFloatIntoValue(f float64, targetValue reflect.Value) {
	if targetValue.Kind() == reflect.Ptr {
		// Create a pointer to the target type and set the value.
		ttp := targetValue.Type()
		tt := ttp.Elem()
		val := reflect.New(tt)
		val.Elem().SetFloat(f)
		targetValue.Set(val)
	} else {
		targetValue.SetFloat(f)
	}
}

// parseIdentifierIntoValue attempts to interpret an identifier and assign its corresponding value to targetValue. It supports
// EnumUnmarshaler interface and strings. Returns an error if it cannot unmarshal the identifier.
func parseIdentifierIntoValue(identifier string, value reflect.Value) error {

	done, err := unmarshalWithEnumUnmarshaler(identifier, value)
	if done {
		return err
	}

	ptr := false
	kind := value.Kind()
	if kind == reflect.Ptr {
		kind = value.Type().Elem().Kind()
		ptr = true
	}

	if kind == reflect.Bool {
		// If the value is a bool, set it.
		var b bool
		lowerIdent := strings.ToLower(identifier)
		if lowerIdent == "true" {
			b = true
		} else if lowerIdent == "false" {
			b = false
		} else {
			return fmt.Errorf("cannot unmarshal identifier %s into type: %v", identifier, value.Type())
		}
		if ptr {
			value.Set(reflect.ValueOf(&b))
		} else {
			value.SetBool(b)
		}
		return nil
	}

	if value.Type() == stringType {
		// If the value is a string, set it. There is no
		// possibility that this is a pointer since we're testing
		// for the string type explicitly.
		value.SetString(identifier)
		return nil
	} else {
		if ptr {
			strValue := reflect.New(value.Type().Elem())
			strValue.Elem().SetString(identifier)
			value.Set(strValue)
		} else {
			// This assumes the value is a string or something that
			// simply wraps a string. No extra checks are done as
			// that is also handled at the reflection layer. If this
			// is not a string, it will panic, that will get caught,
			// and the error will be returned.
			value.SetString(identifier)
		}
		return nil
	}
}

func unmarshalWithEnumUnmarshaler(identifier string, value reflect.Value) (bool, error) {
	// Make a pointer to the value type in case the receiver is a pointer.
	interfaceVal := value
	valueType := interfaceVal.Type()
	if interfaceVal.Kind() == reflect.Ptr {
		// If the value is a pointer, dereference it.
		valueType = interfaceVal.Type().Elem()
		interfaceVal = interfaceVal.Elem()
	}
	destinationVal := reflect.New(valueType)
	if ok := value.CanConvert(enumUnmarshalerType); ok {
		// If it supports the EnumUnmarshaler interface, use that.
		enumUnmarshaler := destinationVal.Convert(enumUnmarshalerType).Interface().(EnumUnmarshaler)
		val, err := enumUnmarshaler.UnmarshalString(identifier)
		if err != nil {
			return true, err
		}
		interfaceVal.Set(reflect.ValueOf(val))
		return true, nil
	} else if ok := value.CanConvert(stringEnumValuesType); ok {
		// If it supports the StringEnumValues interface, use that.
		stringEnumValues := destinationVal.Convert(stringEnumValuesType).Interface().(StringEnumValues)
		enumValues := stringEnumValues.EnumValues()
		for _, enumValue := range enumValues {
			if enumValue.Name == identifier {
				interfaceVal.SetString(identifier)
				return true, nil
			}
		}
		return true, fmt.Errorf("invalid enum value %s", identifier)
	}
	return false, nil
}

// parseListIntoValue assigns a list of GenericValues to targetValue. Each item in the list is parsed into a value and assigned
// to the corresponding index in the slice represented by targetValue. If an item cannot be parsed, it returns an error.
func parseListIntoValue(req *request, inVal genericValue, targetValue reflect.Value) error {
	targetType := targetValue.Type()
	targetValue.Set(reflect.MakeSlice(targetType, len(inVal.List), len(inVal.List)))
	for i, listItem := range inVal.List {
		err := parseInputIntoValue(req, listItem, targetValue.Index(i))
		if err != nil {
			return err
		}
	}
	return nil
}

// parseMapIntoValue assigns a map of GenericValues to the struct represented by targetValue. Each field in the input map is parsed
// into a value and set on the struct field that has a matching "json" tag or field name. If a required field is missing from the
// input map, it returns an error.
func parseMapIntoValue(req *request, inValue genericValue, targetValue reflect.Value) error {
	// A map is a little more complicated. We need to loop through the fields of the target type
	// and set the values from the input map. This is how we initialize a struct from a map.
	targetType := targetValue.Type()
	// TODO: Cache this so we don't have to reconstruct it every time.
	// Loop through the fields of the target type and make a map of the fields by "json" tag.
	fieldMap := map[string]reflect.StructField{}
	requiredFields := map[string]bool{}

	if targetType.Kind() == reflect.Ptr {
		isNilPtr := targetValue.IsNil()
		targetType = targetType.Elem()
		if isNilPtr {
			targetValue.Set(reflect.New(targetType))
		}
		targetValue = targetValue.Elem()
	}

	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		if tag, ok := field.Tag.Lookup("json"); ok {
			fieldMap[tag] = field
		}
		if field.Type.Kind() != reflect.Ptr {
			requiredFields[field.Name] = true
		}
	}

	// Loop through the fields of the input map and set the values in the target value.
	for _, namedValue := range inValue.Map {
		var fieldValue reflect.Value
		var fieldName string

		if targetField, ok := fieldMap[namedValue.Name]; ok {
			// If we find this in the fieldMap, the field has a defined JSON tag, so use that.
			fieldValue = targetValue.FieldByName(targetField.Name)
			fieldName = targetField.Name
		}
		if fieldValue.Kind() == reflect.Invalid {
			// If we didn't find it in the fieldMap, the field doesn't have a defined JSON tag, so
			// try to find it by name in the structure.
			fieldValue = targetValue.FieldByName(namedValue.Name)
			fieldName = namedValue.Name
		}

		if fieldValue.Kind() != reflect.Invalid {
			// We have found the field, so parse the value into it.
			err := parseInputIntoValue(req, namedValue.Value, fieldValue)
			if err != nil {
				return err
			}
			delete(requiredFields, fieldName)
		} else {
			return NewGraphError(fmt.Sprintf("field %s not found in input struct", namedValue.Name), namedValue.Pos, namedValue.Name)
		}
	}

	if len(requiredFields) > 0 {
		missingFields := strings.Join(keys(requiredFields), ", ")
		return NewGraphError("missing required fields: "+missingFields, inValue.Pos)
	}
	return nil
}
