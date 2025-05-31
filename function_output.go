package quickgraph

import (
	"context"
	"fmt"
	"github.com/alecthomas/participle/v2/lexer"
	"reflect"
	"strconv"
	"strings"
)

// processCallOutput takes a command and a slice of call results,
// processes the results based on the kind of value returned,
// and returns a single value and an error if there is any.
// Currently, it only supports slices, maps, and structs,
// and returns an error if the function returns a different kind of value.
func (f *graphFunction) processCallOutput(ctx context.Context, req *request, filter *resultFilter, callResult reflect.Value) (any, error) {
	var pos lexer.Position
	if filter != nil {
		pos = filter.Pos
	}

	kind := callResult.Kind()
	if callResult.CanConvert(errorType) {
		// This should have been handled earlier. But just in case...
		panic("error should have been handled earlier")
	}

	if kind == reflect.Interface {
		callResult = callResult.Elem()
		kind = callResult.Kind()
	}

	if (kind == reflect.Pointer) && !callResult.IsNil() {
		// Check if the pointed-to type is a struct - if so, let processOutputStruct handle it
		// This preserves pointer types for type discovery
		elemKind := callResult.Elem().Kind()
		if elemKind == reflect.Struct {
			sr, err := f.processOutputStruct(ctx, req, filter, callResult)
			if err != nil {
				return nil, AugmentGraphError(err, fmt.Sprintf("error processing struct"), pos)
			}
			return sr, nil
		}
		// For non-struct pointers, dereference as before
		callResult = callResult.Elem()
		kind = callResult.Kind() // Update the kind
	}

	if kind == reflect.Slice {
		if !callResult.IsNil() {
			count := callResult.Len()

			// Check array size limit if configured
			if req.graphy.QueryLimits != nil && req.graphy.QueryLimits.MaxArraySize > 0 {
				if count > req.graphy.QueryLimits.MaxArraySize {
					// Truncate to max size and add warning
					count = req.graphy.QueryLimits.MaxArraySize
				}
			}

			retVal := []any{}
			for i := 0; i < count; i++ {
				a := callResult.Index(i)
				sr, err := f.processCallOutput(ctx, req, filter, a)
				if err != nil {
					return nil, AugmentGraphError(err, fmt.Sprintf("error processing slice element %v", i), pos, strconv.Itoa(i))
				}
				retVal = append(retVal, sr)
			}
			return retVal, nil
		}
		return []any{}, nil
	} else if kind == reflect.Map {
		// TODO: Handle maps?
		return nil, NewGraphError(fmt.Sprintf("maps not supported"), pos)
	} else if kind == reflect.Struct {
		// Check for custom scalar serialization before processing as struct
		if req != nil && req.graphy != nil {
			if scalar, exists := req.graphy.GetScalarByType(callResult.Type()); exists {
				serialized, err := scalar.Serialize(callResult.Interface())
				if err != nil {
					return nil, NewGraphError(fmt.Sprintf("failed to serialize %s: %v", scalar.Name, err), pos)
				}
				return serialized, nil
			}
		}

		sr, err := f.processOutputStruct(ctx, req, filter, callResult)
		if err != nil {
			return nil, AugmentGraphError(err, fmt.Sprintf("error processing struct"), pos)
		}
		return sr, nil
	} else {
		// Check for custom scalar serialization (only if graphy is available)
		if req != nil && req.graphy != nil {
			if scalar, exists := req.graphy.GetScalarByType(callResult.Type()); exists {
				serialized, err := scalar.Serialize(callResult.Interface())
				if err != nil {
					return nil, NewGraphError(fmt.Sprintf("failed to serialize %s: %v", scalar.Name, err), pos)
				}
				return serialized, nil
			}
		}
		return callResult.Interface(), nil
	}
}

// processOutputStruct takes a result filter and a reflect.Value of a struct, processes the struct according to the filter,
// and returns a map and an error if there is any. The map contains the processed fields of the struct.
//
// IMPORTANT: This function takes a reflect.Value instead of interface{} to preserve addressability.
// This is crucial for calling methods with pointer receivers on embedded types. When Go's reflection
// creates a new reflect.Value from an interface{} (via reflect.ValueOf), the resulting value is not
// addressable, which prevents taking its address for pointer receiver methods.
//
// Example scenario this fixes:
//
//	type Employee struct { Name string }
//	func (e *Employee) GetDetails() string { return e.Name }  // pointer receiver
//	type Developer struct { Employee }  // embedded by value
//
// When processing a Developer and trying to call GetDetails on the embedded Employee,
// we need the Employee field to be addressable so we can get *Employee for the method call.
func (f *graphFunction) processOutputStruct(ctx context.Context, req *request, filter *resultFilter, structValue reflect.Value) (any, error) {
	if !structValue.IsValid() {
		return nil, nil
	}

	// Handle nil maps
	if structValue.Kind() == reflect.Map && structValue.IsNil() {
		return nil, nil
	}

	// Dereference interfaces to get to the actual value
	for structValue.Kind() == reflect.Interface {
		if structValue.IsNil() {
			return nil, nil
		}
		structValue = structValue.Elem()
	}

	// Apply type discovery before dereferencing pointers
	// This is important because TypeDiscoverable might be implemented on pointer types
	if structValue.CanInterface() {
		if actual, ok := DiscoverType(structValue.Interface()); ok && actual != nil {
			// Get a reflect.Value from the discovered type
			actualValue := reflect.ValueOf(actual)
			// Only use the discovered type if it's valid and not nil
			if actualValue.IsValid() && (!actualValue.CanInterface() || actualValue.Interface() != nil) {
				// Check if we actually discovered a different type
				if actualValue.Type() != structValue.Type() {
					// fmt.Printf("DEBUG: After discovery - type: %v, value type: %T\n", actualValue.Type(), actual)
					structValue = actualValue
				}
			}
		}
	}

	// Now dereference pointers to get to the actual struct
	for structValue.Kind() == reflect.Ptr {
		if structValue.IsNil() {
			return nil, nil
		}
		structValue = structValue.Elem()
	}

	// Handle union types
	var err error
	structValue, err = deferenceUnionTypeValue(structValue)
	if err != nil {
		return nil, AugmentGraphError(err, "error dereferencing union type", filter.Pos)
	}

	// Get the type for lookup
	t := structValue.Type()
	typeName := t.Name()
	fieldMap := f.g.typeLookup(t)

	if filter == nil {
		return nil, NewGraphError(fmt.Sprintf("output filter is not present"), lexer.Position{})
	}

	fieldsToProcess := []resultField{}
	r := map[string]any{}

	for _, field := range filter.Fields {
		fieldsToProcess = append(fieldsToProcess, field)
	}
	for _, fragmentCall := range filter.Fragments {
		var f *fragmentDef
		if fragmentCall.Inline != nil {
			f = fragmentCall.Inline
		} else if fragmentCall.FragmentRef != nil {
			f = req.stub.fragments[*fragmentCall.FragmentRef].Definition
		}
		// Check if the current type matches the fragment type name OR implements it as an interface
		// This handles both concrete types (e.g., ... on Developer) and interfaces (e.g., ... on Employee)
		if strings.EqualFold(typeName, f.TypeName) {
			// Direct type match - use the current fieldMap
			for _, field := range f.Filter.Fields {
				fieldsToProcess = append(fieldsToProcess, field)
			}
		} else if found, _ := fieldMap.ImplementsInterface(f.TypeName); found {
			// Interface implementation match
			// Don't change fieldMap - the current type's fieldMap should already include
			// methods from embedded types
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
			// Todo: Check for directives. Either here or in fetch.

			fieldAny, err := fieldInfo.fetch(ctx, req, structValue, field.Params)
			if err != nil {
				return nil, AugmentGraphError(err, fmt.Sprintf("error fetching field %v", field.Name), field.Pos, field.Name)
			}
			if field.SubParts != nil {
				fieldVal := reflect.ValueOf(fieldAny)
				subPart, err := f.processCallOutput(ctx, req, field.SubParts, fieldVal)
				if err != nil {
					return nil, AugmentGraphError(err, fmt.Sprintf("error processing subpart %v", field.Name), field.Pos, field.Name)
				}
				r[field.Name] = subPart
			} else {
				r[field.Name] = fieldAny
			}
		}
	}

	return r, nil
}

// deferenceUnionTypeValue works with reflect.Value to preserve addressability while handling union types.
//
// Union types in GraphQL are represented as Go structs with multiple pointer fields, where exactly
// one field should be non-nil at runtime. This function finds and returns that non-nil field's value.
//
// Unlike the original deferenceUnionType function which works with interface{} values,
// this version maintains the reflect.Value chain, preserving addressability. This is critical
// when the union contains types with methods that have pointer receivers.
//
// Example union type:
//
//	type SearchResultUnion struct {
//	    User    *User
//	    Product *Product
//	    Order   *Order
//	}
//
// Only one of User, Product, or Order should be non-nil.
func deferenceUnionTypeValue(v reflect.Value) (reflect.Value, error) {
	if !v.IsValid() {
		return v, nil
	}

	// If the value is a union type, as indicated by its name ending in "Union"
	if !strings.HasSuffix(v.Type().Name(), "Union") {
		return v, nil
	}

	// Find the field that is not nil
	t := v.Type()
	found := false
	var result reflect.Value

	for i := 0; i < t.NumField(); i++ {
		field := v.Field(i)
		switch field.Kind() {
		case reflect.Map, reflect.Pointer, reflect.Interface, reflect.Slice:
			// These are allowed
		default:
			return reflect.Value{}, fmt.Errorf("fields in union type must be pointers, maps, slices, or interfaces")
		}

		if field.IsNil() {
			continue
		}

		if found {
			return reflect.Value{}, fmt.Errorf("more than one field in union type is not nil")
		}

		found = true
		// For pointer types, dereference
		if field.Kind() == reflect.Ptr {
			result = field.Elem()
			if !result.IsValid() {
				return reflect.Value{}, fmt.Errorf("union field %s points to invalid value", t.Field(i).Name)
			}
		} else {
			// For maps, slices, and interfaces, use them directly
			result = field
		}
	}

	if !found {
		return reflect.Value{}, fmt.Errorf("all fields in union type are nil")
	}

	return result, nil
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
			// For pointer types, we need to dereference safely
			if v.Field(i).Kind() == reflect.Ptr {
				elem := v.Field(i).Elem()
				if !elem.IsValid() {
					return nil, fmt.Errorf("union field %s points to invalid value", t.Field(i).Name)
				}
				anyStruct = elem.Interface()
			} else {
				// For maps, slices, and interfaces, use them directly
				anyStruct = v.Field(i).Interface()
			}
			found = true
		}
		if !found {
			return nil, fmt.Errorf("no fields in union type are not nil")
		}
	}
	return anyStruct, nil
}
