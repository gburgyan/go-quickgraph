package quickgraph

import (
	"reflect"
	"strings"
)

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
	// TODO: Handle functions as well as those can fulfil parameterized calls.

}

func (t *TypeFieldLookup) Fetch(v reflect.Value) (any, error) {
	for _, i := range t.fieldIndexes {
		v = v.Field(i)
	}
	return v.Interface(), nil
}
