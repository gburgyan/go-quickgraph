package quickgraph

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/alecthomas/participle/v2/lexer"
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

	// Initialize all parameters to their zero values first
	for _, nameMapping := range f.paramsByName {
		if paramValues[nameMapping.paramIndex].Kind() == reflect.Invalid {
			// Only initialize if not already set (e.g., context or receiver)
			paramValues[nameMapping.paramIndex] = reflect.Zero(nameMapping.paramType)
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
				err := parseInputIntoValue(ctx, req, param.Value, val)
				if err != nil {
					return nil, err
				}
				paramValues[nameMapping.paramIndex] = val
				delete(requiredParams, param.Name)
			}
		}
	}

	if len(requiredParams) > 0 {
		return nil, fmt.Errorf("missing required parameters: %v", strings.Join(keys(requiredParams), ", "))
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
				if normalParamCount < len(params.Values) {
					err := parseInputIntoValue(ctx, req, params.Values[normalParamCount].Value, val)
					if err != nil {
						return nil, err
					}
				} else if val.Type().Kind() != reflect.Ptr {
					// Required parameter not provided
					return nil, fmt.Errorf("missing required parameter at position %d", normalParamCount)
				}
			}
			normalParamCount++
		}
	}
	inParamCount := 0
	if params != nil {
		inParamCount = len(params.Values)
	}
	if inParamCount > normalParamCount {
		return nil, fmt.Errorf("too many parameters provided: expected %d, got %d", normalParamCount, inParamCount)
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
		} else if gft.In(i).Kind() == reflect.Ptr && gft.In(i).Elem().Kind() == reflect.Struct {
			// This is a pointer to struct parameter
			valueParam = reflect.New(gft.In(i).Elem())
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
				// Navigate to the target field using the field path
				// If valueParam is a pointer, dereference it first
				fieldTarget := valueParam
				if fieldTarget.Kind() == reflect.Ptr {
					fieldTarget = fieldTarget.Elem()
				}
				targetField := fieldTarget.Field(nameMapping.paramIndex)

				// If we have a fieldPath, navigate through embedded fields
				// This handles the case where a field is promoted from an anonymous embedded struct
				if len(nameMapping.fieldPath) > 0 {
					// First ensure the embedded struct is initialized if it's a pointer
					if targetField.Kind() == reflect.Ptr && targetField.IsNil() {
						targetField.Set(reflect.New(targetField.Type().Elem()))
					}

					// Navigate to the actual field within the embedded struct
					if targetField.Kind() == reflect.Ptr {
						targetField = targetField.Elem()
					}
					for _, idx := range nameMapping.fieldPath {
						targetField = targetField.Field(idx)
					}
				}

				err := parseInputIntoValue(ctx, req, param.Value, targetField)
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

	// Validate the complete struct after all fields are populated
	for i := startIndex; i < gft.NumIn(); i++ {
		if paramValues[i].IsValid() && !gft.In(i).ConvertibleTo(contextType) {
			// Check if this parameter implements Validator or ValidatorWithContext
			paramValue := paramValues[i]
			if paramValue.CanInterface() {
				if validator, ok := paramValue.Interface().(ValidatorWithContext); ok && ctx != nil {
					if err := validator.ValidateWithContext(ctx); err != nil {
						return nil, NewGraphError(fmt.Sprintf("validation failed: %v", err), lexer.Position{})
					}
				} else if validator, ok := paramValue.Interface().(Validator); ok {
					if err := validator.Validate(); err != nil {
						return nil, NewGraphError(fmt.Sprintf("validation failed: %v", err), lexer.Position{})
					}
				}
			}
		}
	}

	return paramValues, nil
}

// parseInputIntoValue interprets a genericValue according to the type of the targetValue and assigns the result to targetValue.
// This method takes into account various types of input such as string, int, float, list, map, identifier, and GraphQL variable.
// It returns an error if the input cannot be parsed into the target type.
func parseInputIntoValue(ctx context.Context, req *request, inValue genericValue, targetValue reflect.Value) (err error) {
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
		// Make a new instance of the object, set the target to that new instance, then dereference.
		typ = typ.Elem()
		instance := reflect.New(typ)
		targetValue.Set(instance)
		targetValue = targetValue.Elem()
	}
	isSlice := typ.Kind() == reflect.Slice
	isStruct := typ.Kind() == reflect.Struct
	if inValue.Variable != nil {
		if req == nil {
			return fmt.Errorf("variable %s provided but no request", *inValue.Variable)
		}
		// Parse and validate the variable name
		variableName, err := parseVariableName(*inValue.Variable)
		if err != nil {
			return err
		}
		err = parseVariableIntoValue(req, variableName, targetValue)
	} else if inValue.String != nil {
		// The string value has quotes around it, remove them.
		literalValue := (*inValue.String)[1 : len(*inValue.String)-1]
		err = parseStringIntoValue(req, literalValue, targetValue, inValue.Pos)
	} else if inValue.Identifier != nil {
		// This is where we handle enums. We have to look up the value based on the field.
		// This will only work with enums that are strings.
		err = parseIdentifierIntoValue(*inValue.Identifier, targetValue)
	} else if inValue.Int != nil {
		i := *inValue.Int
		err = parseIntIntoValue(req, i, targetValue)
	} else if inValue.Float != nil {
		f := *inValue.Float
		err = parseFloatIntoValue(req, f, targetValue)
	} else if inValue.List != nil || isSlice {
		err = parseListIntoValue(ctx, req, inValue, targetValue)
	} else if inValue.Map != nil || isStruct {
		err = parseMapIntoValue(ctx, req, inValue, targetValue)
	} else {
		// This should never occur as this should be a parse error
		// that gets caught by the parser.
		return fmt.Errorf("no input found to parse into value")
	}
	if err != nil {
		return err
	}

	// Check if the parsed value implements the Validator or ValidatorWithContext interface
	// First check if the value itself (or pointer) implements the interfaces
	if targetValue.CanInterface() {
		// Check pointer type first since methods might be defined on pointer receivers
		if targetValue.Kind() == reflect.Ptr && !targetValue.IsNil() {
			if validator, ok := targetValue.Interface().(ValidatorWithContext); ok && ctx != nil {
				if err := validator.ValidateWithContext(ctx); err != nil {
					return AugmentGraphError(err, "validation failed", inValue.Pos)
				}
			} else if validator, ok := targetValue.Interface().(Validator); ok {
				if err := validator.Validate(); err != nil {
					return AugmentGraphError(err, "validation failed", inValue.Pos)
				}
			}
		}

		// Also check the dereferenced value if it's a pointer
		checkValue := targetValue
		if checkValue.Kind() == reflect.Ptr && !checkValue.IsNil() {
			checkValue = checkValue.Elem()
		}

		if checkValue.CanInterface() {
			if validator, ok := checkValue.Interface().(ValidatorWithContext); ok && ctx != nil {
				if err := validator.ValidateWithContext(ctx); err != nil {
					return AugmentGraphError(err, "validation failed", inValue.Pos)
				}
			} else if validator, ok := checkValue.Interface().(Validator); ok {
				if err := validator.Validate(); err != nil {
					return AugmentGraphError(err, "validation failed", inValue.Pos)
				}
			}
		}
	}

	return nil
}

// parseVariableIntoValue extracts the value of a variable from the provided request and assigns it to targetValue.
func parseVariableIntoValue(req *request, variableName string, targetValue reflect.Value) error {
	value, ok := req.variables[variableName]
	if !ok {
		return fmt.Errorf("variable %v not found", variableName)
	}
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	targetValue.Set(value)
	return nil
}

// parseStringIntoValue interprets the provided string and assigns it to targetValue.
func parseStringIntoValue(req *request, s string, targetValue reflect.Value, pos lexer.Position) error {
	// Check for custom scalar parsing first (only if graphy is available)
	if req != nil && req.graphy != nil {
		if scalar, exists := req.graphy.GetScalarByType(targetValue.Type()); exists {
			parsed, err := scalar.ParseLiteral(s)
			if err != nil {
				return NewGraphError(fmt.Sprintf("failed to parse %s literal: %v", scalar.Name, err), pos)
			}
			targetValue.Set(reflect.ValueOf(parsed))
			return nil
		}
	}

	// Default string handling
	targetValue.SetString(s)
	return nil
}

// parseIntIntoValue converts an int64 to the appropriate type and assigns it to targetValue.
func parseIntIntoValue(req *request, i int64, targetValue reflect.Value) error {
	// Check for custom scalar parsing first (only if graphy is available)
	if req != nil && req.graphy != nil {
		if scalar, exists := req.graphy.GetScalarByType(targetValue.Type()); exists {
			parsed, err := scalar.ParseValue(i)
			if err != nil {
				return fmt.Errorf("failed to parse %s from int %d: %v", scalar.Name, i, err)
			}
			targetValue.Set(reflect.ValueOf(parsed))
			return nil
		}
	}

	// Check bounds based on the target type to prevent overflow
	switch targetValue.Kind() {
	case reflect.Int8:
		if i < math.MinInt8 || i > math.MaxInt8 {
			return fmt.Errorf("value %d overflows int8", i)
		}
	case reflect.Int16:
		if i < math.MinInt16 || i > math.MaxInt16 {
			return fmt.Errorf("value %d overflows int16", i)
		}
	case reflect.Int32:
		if i < math.MinInt32 || i > math.MaxInt32 {
			return fmt.Errorf("value %d overflows int32", i)
		}
	case reflect.Uint:
		if i < 0 || (targetValue.Type().Size() == 4 && i > math.MaxUint32) {
			return fmt.Errorf("value %d overflows uint", i)
		}
		targetValue.SetUint(uint64(i))
		return nil
	case reflect.Uint8:
		if i < 0 || i > math.MaxUint8 {
			return fmt.Errorf("value %d overflows uint8", i)
		}
		targetValue.SetUint(uint64(i))
		return nil
	case reflect.Uint16:
		if i < 0 || i > math.MaxUint16 {
			return fmt.Errorf("value %d overflows uint16", i)
		}
		targetValue.SetUint(uint64(i))
		return nil
	case reflect.Uint32:
		if i < 0 || i > math.MaxUint32 {
			return fmt.Errorf("value %d overflows uint32", i)
		}
		targetValue.SetUint(uint64(i))
		return nil
	case reflect.Uint64:
		if i < 0 {
			return fmt.Errorf("value %d cannot be negative for uint64", i)
		}
		targetValue.SetUint(uint64(i))
		return nil
	case reflect.Float32, reflect.Float64:
		// Convert int to float
		targetValue.SetFloat(float64(i))
		return nil
	}
	targetValue.SetInt(i)
	return nil
}

// parseFloatIntoValue converts a float64 to the appropriate type and assigns it to targetValue.
func parseFloatIntoValue(req *request, f float64, targetValue reflect.Value) error {
	// Check for custom scalar parsing first (only if graphy is available)
	if req != nil && req.graphy != nil {
		if scalar, exists := req.graphy.GetScalarByType(targetValue.Type()); exists {
			parsed, err := scalar.ParseValue(f)
			if err != nil {
				return fmt.Errorf("failed to parse %s from float %f: %v", scalar.Name, f, err)
			}
			targetValue.Set(reflect.ValueOf(parsed))
			return nil
		}
	}

	// Default float handling
	targetValue.SetFloat(f)
	return nil
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
		// This assumes the value is a string or something that
		// simply wraps a string. No extra checks are done as
		// that is also handled at the reflection layer. If this
		// is not a string, it will panic, that will get caught,
		// and the error will be returned.
		value.SetString(identifier)
		return nil
	}
}

func unmarshalWithEnumUnmarshaler(identifier string, value reflect.Value) (bool, error) {
	// Make a pointer to the value type in case the receiver is a pointer.
	interfaceVal := value
	valueType := interfaceVal.Type()
	if interfaceVal.Kind() == reflect.Ptr && interfaceVal.IsNil() {
		// Create something for the pointer to point to.
		instance := reflect.New(valueType.Elem())
		interfaceVal.Set(instance)

		// Now dereference the pointer.
		valueType = interfaceVal.Type().Elem()
		interfaceVal = interfaceVal.Elem()
	} else if interfaceVal.Kind() == reflect.Ptr {
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
func parseListIntoValue(ctx context.Context, req *request, inVal genericValue, targetValue reflect.Value) error {
	targetType := targetValue.Type()
	targetValue.Set(reflect.MakeSlice(targetType, len(inVal.List), len(inVal.List)))
	for i, listItem := range inVal.List {
		err := parseInputIntoValue(ctx, req, listItem, targetValue.Index(i))
		if err != nil {
			return err
		}
	}
	return nil
}

// parseMapIntoValue assigns a map of GenericValues to the struct represented by targetValue. Each field in the input map is parsed
// into a value and set on the struct field that has a matching "json" tag or field name. If a required field is missing from the
// input map, it returns an error.
func parseMapIntoValue(ctx context.Context, req *request, inValue genericValue, targetValue reflect.Value) error {
	// A map is a little more complicated. We need to loop through the fields of the target type
	// and set the values from the input map. This is how we initialize a struct from a map.
	targetType := targetValue.Type()
	// TODO: Cache this so we don't have to reconstruct it every time.
	// Loop through the fields of the target type and make a map of the fields by tags.
	// Priority: graphy tag > json tag > field name
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
		fieldName := ""
		graphyProvidedName := false

		// Check graphy tag first for metadata and name
		if graphyTag, ok := field.Tag.Lookup("graphy"); ok {
			graphyParts := strings.Split(graphyTag, ",")
			if graphyParts[0] == "-" {
				continue
			}
			// Handle both simple name and name=value format
			for _, part := range graphyParts {
				parts := strings.SplitN(part, "=", 2)
				if len(parts) == 1 && parts[0] != "" && !strings.Contains(parts[0], "=") {
					// Simple name without = sign
					fieldName = parts[0]
					graphyProvidedName = true
					break
				} else if len(parts) == 2 && parts[0] == "name" {
					// name=value format
					value := strings.TrimSpace(parts[1])
					if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
						value = value[1 : len(value)-1]
					}
					fieldName = value
					graphyProvidedName = true
					break
				}
			}
		}

		// If graphy didn't provide a name, fall back to json tag
		if !graphyProvidedName {
			if jsonTag, ok := field.Tag.Lookup("json"); ok {
				// Only use the name from the tag if it's not "-" and ignore anything after a comma.
				if jsonTag == "-" {
					continue
				}
				fieldName = strings.Split(jsonTag, ",")[0]
			}
		}

		// Use the determined field name (or field.Name if no tag)
		if fieldName != "" {
			fieldMap[fieldName] = field
		} else {
			fieldMap[field.Name] = field
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
			err := parseInputIntoValue(ctx, req, namedValue.Value, fieldValue)
			if err != nil {
				return AugmentGraphError(err, fmt.Sprintf("error setting field %s", fieldName), inValue.Pos, fieldName)
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
