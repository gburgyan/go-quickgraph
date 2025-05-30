package quickgraph

import (
	"context"
	"fmt"
	"github.com/alecthomas/participle/v2/lexer"
	"reflect"
	"strings"
	"sync"
)

type fieldType int

const (
	FieldTypeField fieldType = iota
	FieldTypeGraphFunction
)

type fieldLookup struct {
	fieldType     fieldType
	name          string
	resultType    reflect.Type
	fieldIndexes  []int
	graphFunction *graphFunction

	isDeprecated     bool
	deprecatedReason string
}

type typeLookup struct {
	mu                  sync.RWMutex // Protects all mutable fields
	typ                 reflect.Type
	rootType            reflect.Type
	isPointer           bool
	array               *typeArrayModifier
	name                string
	fundamental         bool
	fields              map[string]fieldLookup
	fieldsLowercase     map[string]fieldLookup
	implements          map[string]*typeLookup
	implementsLowercase map[string]*typeLookup
	implementedBy       []*typeLookup
	union               map[string]*typeLookup
	unionLowercase      map[string]*typeLookup

	description      *string
	isDeprecated     bool
	deprecatedReason string

	// interfaceOnly indicates that this type should only generate an interface,
	// not both interface and concrete type. Used for opt-out behavior.
	interfaceOnly bool
}

type typeArrayModifier struct {
	isPointer bool
	array     *typeArrayModifier
}

func (tl *typeLookup) GetField(name string) (fieldLookup, bool) {
	tl.mu.RLock()
	defer tl.mu.RUnlock()

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

// populateTypeLookup is a helper function for makeTypeFieldLookup. It recursively processes
// a given type, populating the result map with field lookups. It takes into account JSON
// tags for naming and field exclusion.
func (g *Graphy) populateTypeLookup(typ reflect.Type, prevIndex []int, tl *typeLookup) {
	name := tl.name

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
		fieldTypeLookup := g.typeLookup(field.Type)
		tl.mu.Lock()
		tl.union[fieldTypeLookup.name] = fieldTypeLookup
		// If the lowercase version of the field name is not already in the map,
		// add it.
		if _, ok := tl.unionLowercase[strings.ToLower(name)]; !ok {
			tl.unionLowercase[strings.ToLower(name)] = fieldTypeLookup
		}
		tl.mu.Unlock()
	}
}

func (g *Graphy) processBaseTypeFieldLookup(typ reflect.Type, prevIndex []int, tl *typeLookup) {
	// List of functions to process for the anonymous fields.
	var deferredAnonymous []func()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		index := append(prevIndex, i)
		if field.Anonymous {
			// Queue up the anonymous field for processing later.
			// Capture loop variables to avoid closure issues
			capturedField := field
			capturedIndex := index
			deferredAnonymous = append(deferredAnonymous, func() {
				g.populateTypeLookup(capturedField.Type, capturedIndex, tl)
			})
			// Get the name of the type of the field.
			name := field.Type.Name()

			anonLookup := g.typeLookup(field.Type)

			tl.mu.Lock()
			tl.implements[name] = anonLookup
			tl.implementsLowercase[strings.ToLower(name)] = anonLookup
			tl.mu.Unlock()

			// When establishing the implementedBy relationship, always use the root type
			// This ensures that MyType, *MyType, and []MyType all share the same relationships
			rootAnonLookup := anonLookup
			if anonLookup.rootType != anonLookup.typ {
				// This is a variant (pointer or slice), get the root type lookup
				rootAnonLookup = g.typeLookup(anonLookup.rootType)
			}
			rootAnonLookup.mu.Lock()
			rootAnonLookup.implementedBy = append(rootAnonLookup.implementedBy, tl)
			rootAnonLookup.mu.Unlock()
		} else {

			tfl := g.baseFieldLookup(field, index)

			if tfl.name == "" {
				continue
			}

			// If we already have a field with that name, ignore it.
			tl.mu.Lock()
			if _, ok := tl.fields[tfl.name]; ok {
				tl.mu.Unlock()
				continue
			}

			// TODO: Add enum support here. Special processing for strings that implement
			//  the StringEnumValues interface.

			tl.fields[tfl.name] = tfl
			// If the lowercase version of the field name is not already in the map,
			// add it.
			if _, ok := tl.fieldsLowercase[strings.ToLower(tfl.name)]; !ok {
				tl.fieldsLowercase[strings.ToLower(tfl.name)] = tfl
			}
			tl.mu.Unlock()
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
		// There should be no way of getting here as the upstream code
		// should only pass in a struct, not a pointer to a struct. But
		// just in case, handle it.
		typ = typ.Elem()
		g.addGraphMethodsForType(typ, prevIndex, tl)
	}

	// Process the anonymous fields.
	for _, fn := range deferredAnonymous {
		fn()
	}
}

func (g *Graphy) baseFieldLookup(field reflect.StructField, index []int) fieldLookup {
	// If there's a json tag on the field, use that for the name of the field.
	// Otherwise, use the name of the field.
	// If there's a json tag with a "-" value, ignore the field.
	tfl := fieldLookup{
		name:         field.Name,
		resultType:   field.Type,
		fieldIndexes: index,
		fieldType:    FieldTypeField,
	}

	if jsonTag := field.Tag.Get("json"); jsonTag != "" {
		jsonParts := strings.Split(jsonTag, ",")
		if jsonParts[0] == "-" {
			return fieldLookup{}
		}
		if jsonParts[0] != "" {
			tfl.name = jsonParts[0]
		}
	}

	if graphyTag := field.Tag.Get("graphy"); graphyTag != "" {
		graphyParts := strings.Split(graphyTag, ",")

		// Check if the first part is "-" which means exclude the field
		if len(graphyParts) > 0 && graphyParts[0] == "-" {
			return fieldLookup{}
		}

		// First part, if it has no special meaning, is the name of the field.
		// All the parts are name=value pairs (except the first part, which can be special).
		// If there are quotes around the value, they are stripped.
		// The special parts are:
		//  - name: the name of the field
		//  - deprecated: if exists, the field is deprecated with the value as the reason

		for _, part := range graphyParts {
			parts := strings.Split(part, "=")
			if len(parts) == 1 {
				tfl.name = parts[0]
			} else {
				// If the value is quoted, strip the quotes.
				switch parts[0] {
				case "name":
					tfl.name = parts[1]
				case "deprecated":
					tfl.isDeprecated = true
					tfl.deprecatedReason = parts[1]
				}
			}
		}
	}

	return tfl
}

func (g *Graphy) addGraphMethodsForType(typ reflect.Type, index []int, tl *typeLookup) {
	functionDefs := map[string]FunctionDefinition{}
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		if _, found := ignoredFunctions[m.Name]; found {
			// Ignore functions that are designed to be ignored as
			// framework functions.
			continue
		}
		fd := FunctionDefinition{
			Name:     m.Name,
			Function: m.Func,
		}
		functionDefs[m.Name] = fd
	}
	// Check if the type or its element type implements GraphTypeExtension
	checkType := typ
	if typ.Kind() == reflect.Ptr {
		checkType = typ.Elem()
	}

	if checkType.Implements(graphTypeExtensionType) || reflect.PtrTo(checkType).Implements(graphTypeExtensionType) {
		var gtei GraphTypeExtension
		if checkType.Implements(graphTypeExtensionType) {
			// Type implements the interface directly (value receiver)
			gtev := reflect.New(checkType).Elem()
			gtei = gtev.Interface().(GraphTypeExtension)
		} else {
			// Pointer to type implements the interface
			gtev := reflect.New(checkType)
			gtei = gtev.Interface().(GraphTypeExtension)
		}
		typeExtension := gtei.GraphTypeExtension()
		for _, override := range typeExtension.FunctionDefinitions {
			functionDefs[override.Name] = override
		}
	}

	for _, funcDef := range functionDefs {
		// Gather the inputs and outputs of the function.
		var method reflect.Type
		var function reflect.Value
		if f, ok := funcDef.Function.(reflect.Value); ok {
			method = f.Type()
			function = f
		} else {
			method = reflect.TypeOf(funcDef.Function)
			function = reflect.ValueOf(funcDef.Function)
		}

		var inTypes []reflect.Type
		var outTypes []reflect.Type
		for j := 0; j < method.NumIn(); j++ {
			inTypes = append(inTypes, method.In(j))
		}
		for j := 0; j < method.NumOut(); j++ {
			outTypes = append(outTypes, method.Out(j))
		}

		err := g.validateGraphFunction(function, funcDef.Name, true)
		if err == nil {
			// Todo: Make this take a reflect.Type instead of an any.
			gf := g.newGraphFunction(funcDef, true)
			// TODO: There seems to be a reflection issue where functions from
			//  an anonymous struct are not properly recognized as being from
			//  that struct. We need to figure out what's going on so when emitting
			//  the schema we can properly identify the type and not output the
			//  function multiple times. Basically, if an anonymous struct member
			//  has a function, that will presently be output as a function of
			//  both the struct as well as the type that includes it as anonymous.
			tfl := fieldLookup{
				name:          funcDef.Name,
				resultType:    gf.rawReturnType,
				fieldIndexes:  index,
				fieldType:     FieldTypeGraphFunction,
				graphFunction: &gf,
			}
			tl.mu.Lock()
			tl.fields[funcDef.Name] = tfl
			// If the lowercase version of the field name is not already in the map,
			// add it.
			if _, ok := tl.fieldsLowercase[strings.ToLower(funcDef.Name)]; !ok {
				tl.fieldsLowercase[strings.ToLower(funcDef.Name)] = tfl
			}
			tl.mu.Unlock()
		}
	}
}

// fetch fetches a value from a given reflect.Value using the field indexes.
// It walks the field indexes in order to find the nested field if necessary.
//
// This is the entry point for field resolution. It handles two types of fields:
// 1. Regular struct fields (FieldTypeField) - simply navigates to and returns the field value
// 2. Methods/functions (FieldTypeGraphFunction) - calls fetchGraphFunction to execute the method
//
// The distinction is important because methods on embedded types require special handling
// to ensure the correct receiver type is used when calling the method.
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
	// params can be nil for field methods (methods called as fields in GraphQL without parentheses)
	// We need to handle this gracefully throughout this function
	var pos lexer.Position
	if params != nil {
		pos = params.Pos
	}

	if t.graphFunction == nil {
		return nil, NewGraphError("field is not a function", pos)
	}

	// EMBEDDED TYPE METHOD HANDLING:
	// When a method is defined on an embedded type, we need to navigate through the struct
	// hierarchy to reach the correct receiver. The fieldIndexes slice contains the path
	// through nested fields to reach the embedded type.
	//
	// Example scenario:
	//   type Employee struct { Name string }
	//   func (e *Employee) GetDetails() string { ... }
	//   type Developer struct {
	//       Employee  // embedded at field index 0
	//       Language string
	//   }
	//
	// To call GetDetails on a Developer, we need to:
	// 1. Navigate to field index 0 (the embedded Employee)
	// 2. Ensure the value is suitable for the method's receiver type
	for i, idx := range t.fieldIndexes {
		if !v.IsValid() {
			return nil, NewGraphError(fmt.Sprintf("invalid value at field index %d", i), pos)
		}
		// Handle both struct and pointer to struct
		// We may encounter pointers when dealing with pointer fields or interface{} values
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return nil, NewGraphError(fmt.Sprintf("nil pointer at field index %d", i), pos)
			}
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return nil, NewGraphError(fmt.Sprintf("expected struct at field index %d, got %v", i, v.Kind()), pos)
		}
		if idx >= v.NumField() {
			return nil, NewGraphError(fmt.Sprintf("field index %d out of range (num fields: %d)", idx, v.NumField()), pos)
		}
		// Navigate to the embedded field. This maintains addressability if the parent was addressable.
		v = v.Field(idx)
	}

	if !v.IsValid() {
		return nil, NewGraphError("invalid value after field navigation", pos)
	}

	// Debug: log the value state before calling
	// fmt.Printf("DEBUG fetchGraphFunction: v.Type=%v, v.Kind=%v, v.CanAddr=%v, method=%v\n",
	//     v.Type(), v.Kind(), v.CanAddr(), t.name)

	// Call the method with the navigated value as the receiver
	// The Call function will handle ensuring the receiver type matches what the method expects
	// (e.g., converting Employee to *Employee if needed)
	obj, err := t.graphFunction.Call(ctx, req, params, v)
	if err != nil {
		var pos lexer.Position
		if params != nil {
			pos = params.Pos
		}
		return nil, AugmentGraphError(err, "error calling graph function", pos)
	}
	return obj.Interface(), nil
}

func (t *typeLookup) String() string {
	return fmt.Sprintf("typeLookup: %v", t.typ)
}
