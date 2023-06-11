package quickgraph

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// getCallParameters returns the parameters to use when calling the function represented by this GraphFunction.
// The parameters are returned as a slice of reflect.Value that can be used to call the function.
// The request and command are used to populate the parameters to the function.
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

// parseMappingIntoValue parses the given mapping into the given value. A GenericValue represents
// an input that is of some indeterminate type. The value is parsed and converted into the type
// of the targetValue.
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
