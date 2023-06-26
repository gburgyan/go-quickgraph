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
	fieldType     FieldType
	name          string
	resultType    reflect.Type
	fieldIndexes  []int
	graphFunction *GraphFunction
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

	addGraphMethodsForType(typ, result)

	// if typ is a struct, make a pointer to it to account for receiver pointers.
	if typ.Kind() == reflect.Struct {
		typ = reflect.PtrTo(typ)
		addGraphMethodsForType(typ, result)
	} else if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		addGraphMethodsForType(typ, result)
	}
}

func addGraphMethodsForType(typ reflect.Type, result map[string]TypeFieldLookup) {
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

		if isValidGraphFunction(m.Func, true) {
			// Todo: Make this take a reflect.Type instead of an any.
			gf := NewGraphFunction(m.Name, m.Func, true)
			tfl := TypeFieldLookup{
				name:          m.Name,
				resultType:    m.Type,
				fieldIndexes:  nil,
				fieldType:     FieldTypeGraphFunction,
				graphFunction: &gf,
			}
			result[m.Name] = tfl
		}
	}
}

// Fetch fetches a value from a given reflect.Value using the field indexes.
// It walks the field indexes in order to find the nested field if necessary.
func (t *TypeFieldLookup) Fetch(ctx context.Context, req *Request, v reflect.Value, params *ParameterList) (any, error) {
	switch t.fieldType {
	case FieldTypeField:
		return t.fetchField(v)
	case FieldTypeGraphFunction:
		return t.fetchGraphFunction(ctx, req, v, params)
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

func (t *TypeFieldLookup) fetchGraphFunction(ctx context.Context, req *Request, v reflect.Value, params *ParameterList) (any, error) {
	obj, err := t.graphFunction.Call(ctx, req, params, v)
	if err != nil {
		return nil, err
	}
	return obj.Interface(), nil
}
