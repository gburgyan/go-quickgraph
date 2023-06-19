package quickgraph

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

type EnumUnmarshaler interface {
	UnmarshalString(input string) (interface{}, error)
}

var enumUnmarshalerType = reflect.TypeOf((*EnumUnmarshaler)(nil)).Elem()

// getCallParameters returns the parameters to use when calling the function represented by this GraphFunction.
// The parameters are returned as a slice of reflect.Value that can be used to call the function.
// The request and command are used to populate the parameters to the function.
func (f *GraphFunction) getCallParameters(ctx context.Context, req *Request, command Command) ([]reflect.Value, error) {
	switch f.mode {
	case NamedParamsInline:
		return f.getCallParamsNamedInline(ctx, req, command)

	case AnonymousParamsInline:
		return f.getCallParamsAnonymousInline(ctx, req, command)

	case NamedParamsStruct:
		return f.getCallParamsNamedStruct(ctx, req, command)
	}
	return nil, fmt.Errorf("unknown function mode: %v", f.mode)
}

func (f *GraphFunction) getCallParamsNamedInline(ctx context.Context, req *Request, command Command) ([]reflect.Value, error) {
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
			err := parseInputIntoValue(req, param.Value, val)
			if err != nil {
				return nil, err
			}
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

func (f *GraphFunction) getCallParamsAnonymousInline(ctx context.Context, req *Request, command Command) ([]reflect.Value, error) {
	gft := reflect.TypeOf(f.function)

	// Make something to hold the parameters
	paramValues := make([]reflect.Value, gft.NumIn())

	// Go through all the input parameters and populate the values. If it's a context.Context,
	// use the context from the call. Also keep a count of the number of non-context parameters
	// and fill in those values from the command.
	normalParamCount := 0
	for i := 0; i < gft.NumIn(); i++ {
		if gft.In(i).ConvertibleTo(contextType) {
			paramValues[i] = reflect.ValueOf(ctx)
			continue
		} else {
			// This is a normal parameter, fill it in from the command.
			if normalParamCount >= len(command.Parameters.Values) {
				return nil, fmt.Errorf("too many parameters provided %d", normalParamCount)
			}
			val := reflect.New(gft.In(i)).Elem()
			err := parseInputIntoValue(req, command.Parameters.Values[normalParamCount].Value, val)
			if err != nil {
				return nil, err
			}
			paramValues[i] = val
		}
		normalParamCount++
	}
	if normalParamCount < len(command.Parameters.Values) {
		return nil, fmt.Errorf("too few parameters provided %d", normalParamCount)
	}

	return paramValues, nil
}

func (f *GraphFunction) getCallParamsNamedStruct(ctx context.Context, req *Request, command Command) ([]reflect.Value, error) {
	gft := reflect.TypeOf(f.function)

	// Make something to hold the parameters
	paramValues := make([]reflect.Value, gft.NumIn())

	// Go through all the input parameters and populate the values. If it's a context.Context,
	// use the context from the call. Grab the non-context parameter (only one based on the type)
	// and save that for later.
	var valueParam reflect.Value
	for i := 0; i < gft.NumIn(); i++ {
		if gft.In(i).ConvertibleTo(contextType) {
			paramValues[i] = reflect.ValueOf(ctx)
			continue
		} else if gft.In(i).Kind() == reflect.Ptr {
			// This is the value parameter, save it for later.
			valueParam = reflect.New(gft.In(i))
			paramValues[i] = valueParam.Elem()
			continue
		}
		panic(fmt.Errorf("invalid parameter type %v", gft.In(i)))
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
			err := parseInputIntoValue(req, param.Value, valueParam.Field(nameMapping.paramIndex))
			if err != nil {
				return nil, err
			}
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

// parseInputIntoValue interprets a GenericValue according to the type of the targetValue and assigns the result to targetValue.
// This method takes into account various types of input such as string, int, float, list, map, identifier, and GraphQL variable.
// It returns an error if the input cannot be parsed into the target type.
func parseInputIntoValue(req *Request, inValue GenericValue, targetValue reflect.Value) (err error) {
	// Catch panics and return them as errors.
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	if inValue.Variable != nil {
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
	} else if inValue.List != nil {
		err = parseListIntoValue(req, inValue, targetValue)
	} else if inValue.Map != nil {
		err = parseMapIntoValue(req, inValue, targetValue)
	} else {
		return fmt.Errorf("unknown value type: %v", inValue)
	}
	if err != nil {
		return err
	}

	return nil
}

// parseVariableIntoValue extracts the value of a variable from the provided request and assigns it to targetValue.
func parseVariableIntoValue(req *Request, variableName string, targetValue reflect.Value) error {
	value, ok := req.Variables[variableName]
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
	if value.Kind() == reflect.Ptr {
		// If the value is a pointer, dereference it.
		value = value.Elem()
	}
	// Make a pointer to the value type in case the receiver is a pointer.
	valueType := value.Type()
	valuePtr := reflect.New(valueType)

	if ok := valuePtr.CanConvert(enumUnmarshalerType); ok {
		// If it supports the EnumUnmarshaler interface, use that.
		enumUnmarshaler := valuePtr.Convert(enumUnmarshalerType).Interface().(EnumUnmarshaler)
		val, err := enumUnmarshaler.UnmarshalString(identifier)
		if err != nil {
			return err
		}
		value.Set(reflect.ValueOf(val))
		return nil
	} else if value.Kind() == reflect.String {
		// If the value is a string, set it.
		value.SetString(identifier)
		return nil
	}
	return fmt.Errorf("cannot unmarshal identifier %s into type: %v", identifier, value.Type())
}

// parseListIntoValue assigns a list of GenericValues to targetValue. Each item in the list is parsed into a value and assigned
// to the corresponding index in the slice represented by targetValue. If an item cannot be parsed, it returns an error.
func parseListIntoValue(req *Request, inVal GenericValue, targetValue reflect.Value) error {
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
func parseMapIntoValue(req *Request, inValue GenericValue, targetValue reflect.Value) error {
	// A map is a little more complicated. We need to loop through the fields of the target type
	// and set the values from the input map. This is how we initialize a struct from a map.
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
	return nil
}
