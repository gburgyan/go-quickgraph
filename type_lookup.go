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
	FieldTypeUnion
	FieldTypeEnum
)

type FieldLookup struct {
	// TODO: Add support for enums.
	fieldType     FieldType
	name          string
	resultType    reflect.Type
	fieldIndexes  []int
	graphFunction *GraphFunction
}

type TypeLookup struct {
	typ                 reflect.Type
	name                string
	fields              map[string]FieldLookup
	fieldsLowercase     map[string]FieldLookup
	implements          map[string]*TypeLookup
	implementsLowercase map[string]*TypeLookup
	union               map[string]*TypeLookup
	unionLowercase      map[string]*TypeLookup
}

func (tl *TypeLookup) GetField(name string) (FieldLookup, bool) {
	result, ok := tl.fields[name]
	if !ok {
		result, ok = tl.fieldsLowercase[strings.ToLower(name)]
	}
	return result, ok
	//if ok {
	//	return result, ok
	//}
	//for _, tl := range tl.union {
	//	result, ok = tl.GetField(name)
	//	if ok {
	//		return result, ok
	//	}
	//}
	//return FieldLookup{}, false
}

func (tl *TypeLookup) ImplementsInterface(name string) (bool, *TypeLookup) {
	_, found := tl.implementsLowercase[strings.ToLower(name)]
	if found {
		return true, tl
	}
	for _, tl := range tl.union {
		found, tl := tl.ImplementsInterface(name)
		if found {
			return true, tl
		}
	}
	return false, nil
}

// makeTypeFieldLookup creates a lookup of fields for a given type. It performs
// a depth-first search of the type, including anonymous fields. It creates the lookup
// using either the json tag name or the field name.
func (g *Graphy) makeTypeFieldLookup(typ reflect.Type) *TypeLookup {
	// Do a depth-first search of the type to find all the fields.
	// Include the anonymous fields in this search and treat them as if
	// they were part of the current type in a flattened manner.
	result := &TypeLookup{
		typ:                 typ,
		name:                typ.Name(),
		fields:              make(map[string]FieldLookup),
		fieldsLowercase:     map[string]FieldLookup{},
		implements:          map[string]*TypeLookup{},
		implementsLowercase: map[string]*TypeLookup{},
		union:               map[string]*TypeLookup{},
		unionLowercase:      map[string]*TypeLookup{},
	}
	g.processFieldLookup(typ, nil, result)
	return result
}

// processFieldLookup is a helper function for makeTypeFieldLookup. It recursively processes
// a given type, populating the result map with field lookups. It takes into account JSON
// tags for naming and field exclusion.
func (g *Graphy) processFieldLookup(typ reflect.Type, prevIndex []int, tl *TypeLookup) {
	name := typ.Name()
	if strings.HasSuffix(name, "Union") {
		g.processUnionFieldLookup(typ, prevIndex, tl, name)
	} else {
		g.processBaseTypeFieldLookup(typ, prevIndex, tl)
	}
}

func (g *Graphy) processUnionFieldLookup(typ reflect.Type, prevIndex []int, tl *TypeLookup, name string) {
	name = name[:len(name)-5]
	// The convention for this is to have anonymous fields for each type in the union.

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.Anonymous {
			// For union types, we only care about the anonymous fields.
			continue
		}
		fieldType := field.Type
		fieldTypeLookup := g.typeLookup(fieldType)
		tl.union[name] = fieldTypeLookup
		// If the lowercase version of the field name is not already in the map,
		// add it.
		if _, ok := tl.unionLowercase[strings.ToLower(name)]; !ok {
			tl.unionLowercase[strings.ToLower(name)] = fieldTypeLookup
		}
	}
}

func (g *Graphy) processBaseTypeFieldLookup(typ reflect.Type, prevIndex []int, tl *TypeLookup) {
	// List of functions to process for the anonymous fields.
	var deferredAnonymous []func()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		// If there's a json tag on the field, use that for the name of the field.
		// Otherwise, use the name of the field.
		// If there's a json tag with a "-" value, ignore the field.
		// If there's a json tag with an "omitempty" value, ignore the field.
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
		if _, ok := tl.fields[fieldName]; ok {
			continue
		}
		index := append(prevIndex, i)
		if field.Anonymous {
			// Queue up the anonymous field for processing later.
			deferredAnonymous = append(deferredAnonymous, func() {
				g.processFieldLookup(field.Type, index, tl)
			})
			// Get the name of the type of the field.
			name := field.Type.Name()

			anonLookup := g.typeLookup(field.Type)

			tl.implements[name] = anonLookup
			tl.implementsLowercase[strings.ToLower(name)] = anonLookup
		} else {
			// TODO: Add enum support here. Special processing for strings that implement
			//  the StringEnumValues interface.

			tfl := FieldLookup{
				name:         fieldName,
				resultType:   field.Type,
				fieldIndexes: index,
				fieldType:    FieldTypeField,
			}
			tl.fields[fieldName] = tfl
			// If the lowercase version of the field name is not already in the map,
			// add it.
			if _, ok := tl.fieldsLowercase[strings.ToLower(fieldName)]; !ok {
				tl.fieldsLowercase[strings.ToLower(fieldName)] = tfl
			}
		}
	}

	// A function is more complicated. In all cases a function may take a context
	// parameter and must return some concrete type. The function may also return an
	// error. If the function takes exactly one non-context parameter, it will be
	// treated as an unnamed parameter and any input will be passed to it. If a function
	// needs more complicated parameterization, the parameter must be a struct with
	// fields that match the input.

	// Loop through the methods of the type and find any that match the above criteria.

	g.addGraphMethodsForType(typ, tl)

	// if typ is a struct, make a pointer to it to account for receiver pointers.
	if typ.Kind() == reflect.Struct {
		typ = reflect.PtrTo(typ)
		g.addGraphMethodsForType(typ, tl)
	} else if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		g.addGraphMethodsForType(typ, tl)
	}

	// Process the anonymous fields.
	for _, fn := range deferredAnonymous {
		fn()
	}
}

func (g *Graphy) addGraphMethodsForType(typ reflect.Type, tl *TypeLookup) {
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)

		// Gather the inputs and outputs of the function.
		var inTypes []reflect.Type
		var outTypes []reflect.Type
		for j := 0; j < m.Type.NumIn(); j++ {
			inTypes = append(inTypes, m.Type.In(j))
		}
		for j := 0; j < m.Type.NumOut(); j++ {
			outTypes = append(outTypes, m.Type.Out(j))
		}

		if g.isValidGraphFunction(m.Func, true) {
			// Todo: Make this take a reflect.Type instead of an any.
			gf := g.newGraphFunction(FunctionDefinition{
				Name:     m.Name,
				Function: m.Func,
			}, true)
			tfl := FieldLookup{
				name:          m.Name,
				resultType:    m.Type,
				fieldIndexes:  nil,
				fieldType:     FieldTypeGraphFunction,
				graphFunction: &gf,
			}
			tl.fields[m.Name] = tfl
			// If the lowercase version of the field name is not already in the map,
			// add it.
			if _, ok := tl.fieldsLowercase[strings.ToLower(m.Name)]; !ok {
				tl.fieldsLowercase[strings.ToLower(m.Name)] = tfl
			}
		}
	}
}

// Fetch fetches a value from a given reflect.Value using the field indexes.
// It walks the field indexes in order to find the nested field if necessary.
func (t *FieldLookup) Fetch(ctx context.Context, req *Request, v reflect.Value, params *ParameterList) (any, error) {
	switch t.fieldType {
	case FieldTypeField:
		return t.fetchField(v)
	case FieldTypeGraphFunction:
		return t.fetchGraphFunction(ctx, req, v, params)
	}
	// Return error
	return nil, fmt.Errorf("unknown field type: %v", t.fieldType)
}

func (t *FieldLookup) fetchField(v reflect.Value) (any, error) {
	for _, i := range t.fieldIndexes {
		v = v.Field(i)
	}
	return v.Interface(), nil
}

func (t *FieldLookup) fetchGraphFunction(ctx context.Context, req *Request, v reflect.Value, params *ParameterList) (any, error) {
	obj, err := t.graphFunction.Call(ctx, req, params, v)
	if err != nil {
		return nil, err
	}
	return obj.Interface(), nil
}
