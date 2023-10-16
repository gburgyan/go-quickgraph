package quickgraph

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

type fieldType int

const (
	FieldTypeField fieldType = iota
	FieldTypeGraphFunction
	FieldTypeUnion
	FieldTypeEnum
)

type fieldLookup struct {
	fieldType     fieldType
	name          string
	resultType    reflect.Type
	fieldIndexes  []int
	graphFunction *graphFunction
}

type typeLookup struct {
	typ                 reflect.Type
	rootType            reflect.Type
	isSlice             bool
	isPointer           bool
	isPointerSlice      bool
	name                string
	fundamental         bool
	fields              map[string]fieldLookup
	fieldsLowercase     map[string]fieldLookup
	implements          map[string]*typeLookup
	implementsLowercase map[string]*typeLookup
	union               map[string]*typeLookup
	unionLowercase      map[string]*typeLookup
}

func (tl *typeLookup) GetField(name string) (fieldLookup, bool) {
	result, ok := tl.fields[name]
	if !ok {
		result, ok = tl.fieldsLowercase[strings.ToLower(name)]
	}
	return result, ok
}

func (tl *typeLookup) ImplementsInterface(name string) (bool, *typeLookup) {
	if strings.ToLower(name) == strings.ToLower(tl.name) {
		return true, tl
	}
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

// processFieldLookup is a helper function for makeTypeFieldLookup. It recursively processes
// a given type, populating the result map with field lookups. It takes into account JSON
// tags for naming and field exclusion.
func (g *Graphy) processFieldLookup(typ reflect.Type, prevIndex []int, tl *typeLookup) {
	name := typ.Name()
	if strings.HasSuffix(name, "Union") {
		g.processUnionFieldLookup(typ, prevIndex, tl, name)
	} else {
		g.processBaseTypeFieldLookup(typ, prevIndex, tl)
	}
}

func (g *Graphy) processUnionFieldLookup(typ reflect.Type, prevIndex []int, tl *typeLookup, name string) {
	name = name[:len(name)-5]
	tl.name = name

	// The convention for this is to have anonymous fields for each type in the union.

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// TODO: Add some sanity checking here. Right now it's is bit too loose.
		fieldType := field.Type
		fieldTypeLookup := g.typeLookup(fieldType)
		tl.union[fieldTypeLookup.name] = fieldTypeLookup
		// If the lowercase version of the field name is not already in the map,
		// add it.
		if _, ok := tl.unionLowercase[strings.ToLower(name)]; !ok {
			tl.unionLowercase[strings.ToLower(name)] = fieldTypeLookup
		}
	}
}

func (g *Graphy) processBaseTypeFieldLookup(typ reflect.Type, prevIndex []int, tl *typeLookup) {
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

			tfl := fieldLookup{
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

	g.addGraphMethodsForType(typ, prevIndex, tl)

	// if typ is a struct, make a pointer to it to account for receiver pointers.
	if typ.Kind() == reflect.Struct {
		typ = reflect.PtrTo(typ)
		g.addGraphMethodsForType(typ, prevIndex, tl)
	} else if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		g.addGraphMethodsForType(typ, prevIndex, tl)
	}

	// Process the anonymous fields.
	for _, fn := range deferredAnonymous {
		fn()
	}
}

func (g *Graphy) addGraphMethodsForType(typ reflect.Type, index []int, tl *typeLookup) {
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

		err := g.validateGraphFunction(m.Func, m.Name, true)
		if err == nil {
			// Todo: Make this take a reflect.Type instead of an any.
			gf := g.newGraphFunction(FunctionDefinition{
				Name:     m.Name,
				Function: m.Func,
			}, true)
			// TODO: There seems to be a reflection issue where functions from
			//  an anonymous struct are not properly recognized as being from
			//  that struct. We need to figure out what's going on so when emitting
			//  the schema we can properly identify the type and not output the
			//  function multiple times. Basically, if an anonymous struct member
			//  has a function, that will presently be output as a function of
			//  both the struct as well as the type that includes it as anonymous.
			tfl := fieldLookup{
				name:          m.Name,
				resultType:    gf.rawReturnType,
				fieldIndexes:  index,
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

// fetch fetches a value from a given reflect.Value using the field indexes.
// It walks the field indexes in order to find the nested field if necessary.
func (t *fieldLookup) fetch(ctx context.Context, req *request, v reflect.Value, params *parameterList) (any, error) {
	switch t.fieldType {
	case FieldTypeField:
		return t.fetchField(v)
	case FieldTypeGraphFunction:
		return t.fetchGraphFunction(ctx, req, v, params)
	}
	// This should never happen, but return an error if we get here.
	return nil, NewGraphError(fmt.Sprintf("unknown field type: %v", t.fieldType), params.Pos)
}

func (t *fieldLookup) fetchField(v reflect.Value) (any, error) {
	for _, i := range t.fieldIndexes {
		v = v.Field(i)
	}
	return v.Interface(), nil
}

func (t *fieldLookup) fetchGraphFunction(ctx context.Context, req *request, v reflect.Value, params *parameterList) (any, error) {
	obj, err := t.graphFunction.Call(ctx, req, params, v)
	if err != nil {
		return nil, AugmentGraphError(err, "error calling graph function", params.Pos)
	}
	return obj.Interface(), nil
}
