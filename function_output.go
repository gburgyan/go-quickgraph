package quickgraph

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// processCallOutput takes a command and a slice of call results,
// processes the results based on the kind of value returned,
// and returns a single value and an error if there is any.
// Currently, it only supports slices, maps, and structs,
// and returns an error if the function returns a different kind of value.
func (f *GraphFunction) processCallOutput(ctx context.Context, req *Request, filter *ResultFilter, callResult reflect.Value) (any, error) {
	kind := callResult.Kind()
	if callResult.CanConvert(errorType) {
		return nil, fmt.Errorf("error calling function: %v", callResult.Convert(errorType).Interface().(error))
	}

	if kind == reflect.Interface {
		callResult = callResult.Elem()
		kind = callResult.Kind()
	}

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
				sr, err := f.processOutputStruct(ctx, req, filter, a)
				if err != nil {
					return nil, err
				}
				retVal = append(retVal, sr)
			}
			return retVal, nil
		}
	} else if kind == reflect.Map {
		// TODO: Handle maps?
		return nil, fmt.Errorf("return of map type not supported")
	} else if kind == reflect.Struct {
		sr, err := f.processOutputStruct(nil, req, filter, callResult.Interface())
		if err != nil {
			return nil, err
		}
		return sr, nil
	} else {
		return nil, fmt.Errorf("return type of %v not supported", kind)
	}

	// TODO: Better error handling.
	return nil, nil
}

// processOutputStruct takes a result filter and a struct, processes the struct according to the filter,
// and returns a map and an error if there is any. The map contains the processed fields of the struct.
func (f *GraphFunction) processOutputStruct(ctx context.Context, req *Request, filter *ResultFilter, anyStruct any) (map[string]any, error) {
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

	t := reflect.TypeOf(anyStruct)
	typeName := t.Name()
	fieldMap := f.g.MakeTypeFieldLookup(t)

	fieldsToProcess := []ResultField{}
	for _, field := range filter.Fields {
		fieldsToProcess = append(fieldsToProcess, field)
	}
	for _, fragmentCall := range filter.Fragments {
		var f *FragmentDef
		if fragmentCall.Inline != nil {
			f = fragmentCall.Inline
		} else if fragmentCall.FragmentRef != nil {
			f = req.Stub.Fragments[*fragmentCall.FragmentRef].Definition
		}
		if found, tl := fieldMap.ImplementsInterface(f.TypeName); found {
			fieldMap = tl
			for _, field := range f.Filter.Fields {
				fieldsToProcess = append(fieldsToProcess, field)
			}
		}
	}

	// Go through the result fields and map them to the struct fields.
	for _, field := range fieldsToProcess {
		if field.Name == "__typename" {
			r[field.Name] = typeName
		} else {
			fieldInfo, ok := fieldMap.GetField(field.Name)
			if !ok {
				// TODO: Is this an error?
				continue
			}
			// Todo: Check for directives. Either here or in Fetch.

			fieldAny, err := fieldInfo.Fetch(ctx, req, reflect.ValueOf(anyStruct), field.Params)
			if err != nil {
				return nil, err
			}
			if field.SubParts != nil {
				fieldVal := reflect.ValueOf(fieldAny)
				subPart, err := f.processCallOutput(ctx, req, field.SubParts, fieldVal)
				if err != nil {
					return nil, err
				}
				r[field.Name] = subPart
			} else {
				r[field.Name] = fieldAny
			}
		}
	}

	return r, nil
}

// deferenceUnionType takes a struct and checks if the struct is a union type.
// If it is, it finds the actual type of the struct and returns it.
// If the struct is not a union type it's simply returned as-is. If there is an
// error, it is returned.
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
