package quickgraph

import (
	"context"
	"reflect"
	"strings"
)

type GraphFunction struct {
	name        string
	function    any
	nameMapping map[string]FunctionNameMapping
}

type FunctionNameMapping struct {
	name       string
	paramIndex int
	paramType  reflect.Type
}

func NewGraphFunction(name string, mutatorFunc any, names ...string) GraphFunction {
	mft := reflect.TypeOf(mutatorFunc)
	if mft.Kind() != reflect.Func {
		panic("mutatorFunc must be a func")
	}

	// Check the parameters of the mutatorFunc. The count of names must match the count of parameters.
	// Note that context.Context is not a parameter that is counted.
	nameMapping := map[string]FunctionNameMapping{}
	for i := 0; i < mft.NumIn(); i++ {
		if mft.In(i).ConvertibleTo(contextType) {
			continue
		}
		if len(nameMapping) >= len(names) {
			panic("too few names provided")
		}
		name := names[len(nameMapping)]
		nameMapping[name] = FunctionNameMapping{
			name:       name,
			paramIndex: i,
			paramType:  mft.In(i),
		}
	}
	if len(nameMapping) != len(names) {
		panic("the count of names must match the count of parameters")
	}

	gf := GraphFunction{
		name:        name,
		function:    mutatorFunc,
		nameMapping: nameMapping,
	}
	return gf
}

func (f *GraphFunction) Call(ctx context.Context, req *Request, command Command) (any, error) {

	gft := reflect.TypeOf(f.function)
	gfv := reflect.ValueOf(f.function)

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

	// TODO: Make sure all required parameters are present.
	parsedParams := command.Parameters
	for _, param := range parsedParams.Values {
		if nameMapping, ok := f.nameMapping[param.Name]; ok {
			val := reflect.New(nameMapping.paramType).Elem()
			parseMappingIntoValue(req, param.Value, val)
			paramValues[nameMapping.paramIndex] = val
		}
	}

	callResults := gfv.Call(paramValues)
	if len(callResults) == 0 {
		return nil, nil
	}

	// TODO: Tighten this up to deal with the return types better.
	for _, callResult := range callResults {
		if callResult.CanConvert(errorType) {
			return nil, callResult.Interface().(error)
		}
	}
	for _, callResult := range callResults {
		kind := callResult.Kind()
		if (kind == reflect.Interface || kind == reflect.Pointer) && !callResult.IsNil() {
			return callResult.Interface(), nil
		}
		if kind == reflect.Slice {
			if !callResult.IsNil() {
				var retVal []any
				count := callResult.Len()
				for i := 0; i < count; i++ {
					a := callResult.Index(i).Interface()
					sr, err := processStruct(command.ResultFilter.Fields, a)
					if err != nil {
						return nil, err
					}
					retVal = append(retVal, sr)
				}
				return retVal, nil
			}

		} else if kind == reflect.Map {
			// TODO: Is this needed?
			if !callResult.IsNil() {
				return callResult.Interface(), nil
			}
		} else if kind == reflect.Struct {
			sr, err := processStruct(command.ResultFilter.Fields, callResult.Interface())
			if err != nil {
				return nil, err
			}
			return sr, nil
		}
	}

	// TODO: Better error handling.

	return nil, nil
}

func processStruct(filter []ResultField, anyStruct any) (any, error) {
	r := map[string]any{}

	// If the anyStruct is a pointer, dereference it.
	if reflect.TypeOf(anyStruct).Kind() == reflect.Ptr {
		if reflect.ValueOf(anyStruct).IsNil() {
			return nil, nil
		}
		anyStruct = reflect.ValueOf(anyStruct).Elem().Interface()
	}

	// Create map of field names, as specified by the json tag or field name, to index.
	// This is used to map the fields in the struct to the fields in the result.
	// TODO: Cache this.
	fieldMap := map[string]int{}
	t := reflect.TypeOf(anyStruct)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			fieldName = jsonTag
		}
		fieldMap[fieldName] = i
	}

	// Go through the result fields and map them to the struct fields.
	for _, field := range filter {
		if field.Params != nil {
			// TODO: Deal with parameterized fields.
		} else if field.Name == "__typename" {
			// TODO: Magic field to get type.
		} else {
			if index, ok := fieldMap[field.Name]; ok {
				if len(field.SubParts) > 0 {
					subPart, err := processStruct(field.SubParts, reflect.ValueOf(anyStruct).Field(index).Interface())
					if err != nil {
						return nil, err
					}
					r[field.Name] = subPart
				} else {
					fieldValue := reflect.ValueOf(anyStruct).Field(index)
					r[field.Name] = fieldValue.Interface()
				}
			} else {
				// TODO: Is this an error?
			}
		}
	}

	return r, nil
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
