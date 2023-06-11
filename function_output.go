package quickgraph

import (
	"fmt"
	"reflect"
	"strings"
)

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
