package quickgraph

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

type FieldType int

const (
	FieldTypeField FieldType = iota
	FieldTypeGraphFunction
)

type TypeFieldLookup struct {
	fieldType    FieldType
	name         string
	resultType   reflect.Type
	fieldIndexes []int
}

// MakeTypeFieldLookup creates a lookup of fields for a given type. It performs
// a depth-first search of the type, including anonymous fields. It creates the lookup
// using either the json tag name or the field name.
func MakeTypeFieldLookup(typ reflect.Type) map[string]TypeFieldLookup {
	// Do a depth-first search of the type to find all of the fields.
	// Include the anonymous fields in this search and treat them as if
	// they were part of the current type in a flattened manner.
	result := map[string]TypeFieldLookup{}
	processFieldLookup(typ, nil, result)
	return result
}

// processFieldLookup is a helper function for MakeTypeFieldLookup. It recursively processes
// a given type, populating the result map with field lookups. It takes into account JSON
// tags for naming and field exclusion.
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
				name:         fieldName,
				resultType:   field.Type,
				fieldIndexes: index,
				fieldType:    FieldTypeField,
			}
			result[fieldName] = tfl
		}
	}

	// TODO: Handle functions as well as those can fulfil parameterized calls.

	// A function is more complicated. In all cases a function may take a context
	// parameter and must return some concrete type. The function may also return an
	// error. If the function takes exactly one non-context parameter, it will be
	// treated as an unnamed parameter and any input will be passed to it. If a function
	// needs more complicated parameterization, the parameter must be a struct with
	// fields that match the input.

	// Loop through the methods of the type and find any that match the above criteria.

	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)

		// Gather the inputs and outputs of the function.
		inTypes := []reflect.Type{}
		outTypes := []reflect.Type{}
		for j := 0; j < m.Type.NumIn(); j++ {
			inTypes = append(inTypes, m.Type.In(j))
		}
		for j := 0; j < m.Type.NumOut(); j++ {
			outTypes = append(outTypes, m.Type.Out(j))
		}

		// Make a proxy function without the this parameter that's required for method calls.
		// This allows us to call the function without having to pass in the struct.
		proxyFunc := reflect.FuncOf(inTypes[1:], outTypes, false)

		// Todo: Change the isValidGraphFunction to take a reflect.Type instead of an any.
		tempFunc := reflect.MakeFunc(proxyFunc, func(args []reflect.Value) []reflect.Value {
			return []reflect.Value{}
		})

		if isValidGraphFunction(tempFunc.Interface(), true) {
			// Todo: Make this take a reflect.Type instead of an any.
			newStructGraphFunction(m.Name, tempFunc.Interface(), typ)
			tfl := TypeFieldLookup{
				name:         m.Name,
				resultType:   m.Type,
				fieldIndexes: nil,
				fieldType:    FieldTypeGraphFunction,
			}
			result[m.Name] = tfl
		}
	}
}

// Fetch fetches a value from a given reflect.Value using the field indexes.
// It walks the field indexes in order to find the nested field if necessary.
func (t *TypeFieldLookup) Fetch(ctx context.Context, req *Request, v reflect.Value) (any, error) {
	switch t.fieldType {
	case FieldTypeField:
		return t.fetchField(v)
	case FieldTypeGraphFunction:
		return t.fetchGraphFunction(req, v)
	}
	// Return error
	return nil, fmt.Errorf("unknown field type: %v", t.fieldType)
}

func (t *TypeFieldLookup) fetchField(v reflect.Value) (any, error) {
	for _, i := range t.fieldIndexes {
		v = v.Field(i)
	}
	return v.Interface(), nil
}

func (t *TypeFieldLookup) fetchGraphFunction(req *Request, v reflect.Value) (any, error) {
	panic("not implemented")
}
