package quickgraph

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type __Directive struct {
	Name         string   `json:"name"`
	Description  *string  `json:"description"`
	Locations    []string `json:"locations"`
	Args         []__InputValue
	IsRepeatable bool `json:"isRepeatable"`
}

type __Schema struct {
	Description  *string `json:"description"`
	Queries      *__Type `json:"queryType"`
	Mutations    *__Type `json:"mutationType"`
	Subscription *__Type `json:"subscriptionType"`

	Types      []*__Type      `json:"types"`
	Directives []*__Directive `json:"directives"`

	typeLookupByName map[string]*__Type
}

type __Type struct {
	Kind           __TypeKind `json:"kind"`
	Name           *string    `json:"name"`
	Description    *string    `json:"description"`
	fieldsRaw      []__Field
	Interfaces     []*__Type `json:"interfaces"`
	PossibleTypes  []*__Type `json:"possibleTypes"`
	enumValuesRaw  []__EnumValue
	InputFields    []__InputValue
	OfType         *__Type `json:"ofType"`
	SpecifiedByUrl *string `json:"specifiedByUrl"`
}

type __EnumValue struct {
	Name              string  `json:"name"`
	Description       *string `json:"description"`
	IsDeprecated      bool    `json:"isDeprecated"`
	DeprecationReason *string `json:"deprecationReason"`
}

type __Field struct {
	Name              string         `json:"name"`
	Description       *string        `json:"description"`
	Args              []__InputValue `json:"args"`
	Type              *__Type        `json:"type"`
	IsDeprecated      bool           `json:"isDeprecated"`
	DeprecationReason *string        `json:"deprecationReason"`
}

type __TypeKind string

const (
	IntrospectionKindScalar      __TypeKind = "SCALAR"
	IntrospectionKindObject      __TypeKind = "OBJECT"
	IntrospectionKindInterface   __TypeKind = "INTERFACE"
	IntrospectionKindUnion       __TypeKind = "UNION"
	IntrospectionKindEnum        __TypeKind = "ENUM"
	IntrospectionKindInputObject __TypeKind = "INPUT_OBJECT"
	IntrospectionKindList        __TypeKind = "LIST"
	IntrospectionKindNonNull     __TypeKind = "NON_NULL"
)

type __InputValue struct {
	Name         string  `json:"name"`
	Description  *string `json:"description"`
	Type         *__Type `json:"type"`
	DefaultValue *string `json:"defaultValue"`
}

func (it *__Type) Fields(includeDeprecatedOpt *bool) []__Field {
	includeDeprecated := includeDeprecatedOpt != nil && *includeDeprecatedOpt

	result := []__Field{}

	fields := it.fieldsRaw
	// Sort the fields by name
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].Name < fields[j].Name
	})

	for _, field := range fields {
		field := field
		if !field.IsDeprecated || includeDeprecated {
			result = append(result, field)
		}
	}
	return result
}

func (it *__Type) EnumValues(includeDeprecatedOpt *bool) []__EnumValue {
	includeDeprecated := includeDeprecatedOpt != nil && *includeDeprecatedOpt

	result := []__EnumValue{}
	// Sort the enum values by name
	values := it.enumValuesRaw
	sort.Slice(values, func(i, j int) bool {
		return values[i].Name < values[j].Name
	})

	for _, enumValue := range values {
		enumValue := enumValue
		if !enumValue.IsDeprecated || includeDeprecated {
			result = append(result, enumValue)
		}
	}
	return result
}

func (g *Graphy) EnableIntrospection(ctx context.Context) {
	g.schemaEnabled = true
	schemaFunc := func() *__Schema {
		st := g.getSchemaTypes()
		return st.introspectionSchema
	}
	typesFunc := func(name string) (*__Type, error) {
		st := g.getSchemaTypes()
		tl, ok := st.introspectionSchema.typeLookupByName[name]
		if !ok {
			return nil, fmt.Errorf("type %s not found", name)
		}
		return tl, nil
	}
	g.RegisterQuery(ctx, "__schema", schemaFunc)
	g.RegisterQuery(ctx, "__type", typesFunc, "name")
}

func (g *Graphy) populateIntrospection(st *schemaTypes) {
	queryName := "__query"
	mutationName := "__mutation"
	subscriptionName := "__subscription"
	queries := &__Type{Kind: IntrospectionKindObject, Name: &queryName}
	mutations := &__Type{Kind: IntrospectionKindObject, Name: &mutationName}
	subscriptions := &__Type{Kind: IntrospectionKindObject, Name: &subscriptionName}

	is := &__Schema{
		Queries:          queries,
		Mutations:        mutations,
		Subscription:     subscriptions,
		Types:            []*__Type{},
		typeLookupByName: make(map[string]*__Type),
	}

	processorNames := keys(g.processors)
	sort.Strings(processorNames)

	hasSubscriptions := false
	for _, name := range processorNames {
		f := g.processors[name]
		if strings.HasPrefix(f.name, "__") {
			continue
		}
		t, args := g.introspectionCall(is, &f)
		qf := __Field{Name: f.name, Type: t, Args: args}

		switch f.mode {
		case ModeQuery:
			queries.fieldsRaw = append(queries.fieldsRaw, qf)
		case ModeMutation:
			mutations.fieldsRaw = append(mutations.fieldsRaw, qf)
		case ModeSubscription:
			subscriptions.fieldsRaw = append(subscriptions.fieldsRaw, qf)
			hasSubscriptions = true
		}
	}

	typeNames := keys(is.typeLookupByName)
	sort.Strings(typeNames)

	for _, name := range typeNames {
		is.Types = append(is.Types, is.typeLookupByName[name])
	}

	// Add the root types to the lookup map
	is.typeLookupByName["__query"] = queries
	is.typeLookupByName["__mutation"] = mutations

	is.Types = append(is.Types, queries, mutations)
	if hasSubscriptions {
		is.typeLookupByName["__subscription"] = subscriptions
		is.Types = append(is.Types, subscriptions)
	} else {
		// If no subscriptions are registered, set it to nil
		is.Subscription = nil
	}
	g.schemaBuffer.introspectionSchema = is
}

func (g *Graphy) getIntrospectionBaseType(is *__Schema, tl *typeLookup, io TypeKind) *__Type {
	var name string

	if tl.rootType != nil && tl.rootType.ConvertibleTo(stringEnumValuesType) {
		name = g.schemaBuffer.enumTypeNameLookup[tl]
	} else if tl.fundamental {
		if otlName, ok := g.schemaBuffer.outputTypeNameLookup[tl]; ok {
			name = otlName
		} else {
			name = introspectionScalarName(tl)
		}
	} else if io == TypeOutput || tl.fundamental {
		name = g.schemaBuffer.outputTypeNameLookup[tl]
	} else if io == TypeInput {
		name = g.schemaBuffer.inputTypeNameLookup[tl]
	} else {
		panic("unknown IO type")
	}

	// Check if we need to handle interface/concrete type split
	hasImplementedBy := len(tl.implementedBy) > 0
	if io == TypeOutput && hasImplementedBy && !tl.interfaceOnly {
		// This type needs both interface and concrete type like in schema generation
		// First ensure the interface with "I" prefix exists
		interfaceName := "I" + name
		if _, ok := is.typeLookupByName[interfaceName]; !ok {
			interfaceType := &__Type{
				Name: &interfaceName,
				Kind: IntrospectionKindInterface,
			}
			is.typeLookupByName[interfaceName] = interfaceType
			// Add fields to interface
			g.addIntrospectionSchemaFields(is, tl, io, interfaceType)
			// Add possible types (implementations) including the concrete type
			// First add the concrete type itself
			concreteType := &__Type{
				Name: &name,
				Kind: IntrospectionKindObject,
			}
			interfaceType.PossibleTypes = append(interfaceType.PossibleTypes, concreteType)
			// Then add other implementations
			for _, impl := range tl.implementedBy {
				implType := g.getIntrospectionBaseType(is, impl, io)
				interfaceType.PossibleTypes = append(interfaceType.PossibleTypes, implType)
			}
		}

		// Now handle the concrete type
		if existing, ok := is.typeLookupByName[name]; ok {
			return existing
		}

		// Create the concrete type
		result := &__Type{Name: &name}
		is.typeLookupByName[name] = result
		// The concrete type will have its kind set by the caller
		return result
	}

	if existing, ok := is.typeLookupByName[name]; ok {
		return existing
	}

	result := &__Type{Name: &name}
	is.typeLookupByName[name] = result

	switch {
	case len(tl.union) > 0:
		result.Kind = IntrospectionKindUnion
		for _, name := range sortedKeys(tl.union) {
			ul := tl.union[name]
			result.PossibleTypes = append(result.PossibleTypes, g.getIntrospectionModifiedType(is, ul, io))
		}
	case len(tl.implementedBy) > 0:
		result.Kind = IntrospectionKindInterface
		g.addIntrospectionSchemaFields(is, tl, io, result)
		impls := tl.implementedBy
		sort.Slice(impls, func(i, j int) bool {
			return impls[i].name < impls[j].name
		})
		for _, impl := range impls {
			implType := g.getIntrospectionBaseType(is, impl, io)
			result.PossibleTypes = append(result.PossibleTypes, implType)
		}
	case tl.rootType.ConvertibleTo(stringEnumValuesType):
		result.Kind = IntrospectionKindEnum
		enumValue := reflect.New(tl.rootType)
		sev := enumValue.Convert(stringEnumValuesType)
		se := sev.Interface().(StringEnumValues)
		for _, s := range se.EnumValues() {
			s := s
			value := __EnumValue{
				Name: s.Name,
			}
			if s.Description != "" {
				value.Description = &s.Description
			}
			if s.IsDeprecated {
				value.IsDeprecated = true
				value.DeprecationReason = &s.DeprecationReason
			}
			result.enumValuesRaw = append(result.enumValuesRaw, value)
		}

	case tl.fundamental:
		result.Kind = IntrospectionKindScalar
		// Name is already set as &name above

	case io == TypeInput:
		result.Kind = IntrospectionKindInputObject
		g.addIntrospectionSchemaFields(is, tl, io, result)

	default:
		result.Kind = IntrospectionKindObject
		g.addIntrospectionSchemaFields(is, tl, io, result)

		// Handle interface references
		for _, impls := range sortedKeys(tl.implements) {
			implTl := tl.implements[impls]
			// Get the interface name (with "I" prefix if applicable)
			var interfaceName string
			if implTl.rootType != nil && implTl.rootType.ConvertibleTo(stringEnumValuesType) {
				interfaceName = g.schemaBuffer.enumTypeNameLookup[implTl]
			} else if io == TypeOutput || implTl.fundamental {
				interfaceName = g.schemaBuffer.outputTypeNameLookup[implTl]
			} else {
				interfaceName = g.schemaBuffer.inputTypeNameLookup[implTl]
			}

			// Check if this interface uses the "I" prefix pattern
			if len(implTl.implementedBy) > 0 && !implTl.interfaceOnly {
				interfaceName = "I" + interfaceName
			}

			// Look up the interface type by name
			if interfaceType, ok := is.typeLookupByName[interfaceName]; ok {
				result.Interfaces = append(result.Interfaces, interfaceType)
			} else {
				// If not found, create a reference to it
				interfaceType := &__Type{
					Name: &interfaceName,
					Kind: IntrospectionKindInterface,
				}
				result.Interfaces = append(result.Interfaces, interfaceType)
			}
		}

		// If this type has implementedBy, it should also implement its own interface
		if len(tl.implementedBy) > 0 && !tl.interfaceOnly {
			typeName := g.schemaBuffer.outputTypeNameLookup[tl]
			interfaceName := "I" + typeName
			if interfaceType, ok := is.typeLookupByName[interfaceName]; ok {
				result.Interfaces = append(result.Interfaces, interfaceType)
			}
		}
	}

	return result
}

func introspectionScalarName(tl *typeLookup) string {
	kind := tl.rootType.Kind()
	switch kind {
	case reflect.Bool:
		return "Boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "Int"
	case reflect.Float32, reflect.Float64:
		return "Float"
	case reflect.String:
		return "String"
	default:
		panic("unknown scalar type")
	}
}

func (g *Graphy) addIntrospectionSchemaFields(is *__Schema, tl *typeLookup, io TypeKind, result *__Type) {
	tl.mu.RLock()
	fieldsCopy := make(map[string]fieldLookup)
	for k, v := range tl.fields {
		fieldsCopy[k] = v
	}
	tl.mu.RUnlock()

	for _, fieldName := range sortedKeys(fieldsCopy) {
		ft := fieldsCopy[fieldName]
		if ft.fieldType == FieldTypeField {
			if io == TypeOutput {
				// Let the type lookup handle whether this should be an interface
				fieldType := g.getIntrospectionModifiedType(is, g.typeLookup(ft.resultType), io)

				field := __Field{
					Name:         fieldName,
					Type:         fieldType,
					IsDeprecated: ft.isDeprecated,
				}
				if ft.isDeprecated {
					field.DeprecationReason = &ft.deprecatedReason
				}
				result.fieldsRaw = append(result.fieldsRaw, field)
			} else {
				input := __InputValue{
					Name: fieldName,
					Type: g.getIntrospectionModifiedType(is, g.typeLookup(ft.resultType), io),
				}
				result.InputFields = append(result.InputFields, input)
			}
		} else if ft.fieldType == FieldTypeGraphFunction {
			call, args := g.introspectionCall(is, ft.graphFunction)
			result.fieldsRaw = append(result.fieldsRaw, __Field{Name: fieldName, Type: call, Args: args})
		}
	}
}

func (g *Graphy) introspectionCall(is *__Schema, f *graphFunction) (*__Type, []__InputValue) {
	// Get the base return type
	result := g.getIntrospectionModifiedType(is, f.baseReturnType, TypeOutput)

	var args []__InputValue
	for _, param := range f.paramsByIndex {
		args = append(args, __InputValue{
			Name: param.name,
			Type: g.getIntrospectionModifiedType(is, g.typeLookup(param.paramType), TypeInput),
		})
	}
	return result, args
}

// getIntrospectionModifiedType is a method of the Graphy struct. It is used to generate a modified
// introspection type based on a given base type and its characteristics. This is used in the process of
// generating the introspection schema of a GraphQL server.
//
// The method takes three parameters:
// - is: a pointer to the __Schema struct representing the introspection schema.
// - tl: a pointer to the typeLookup struct representing the base type.
// - io: a TypeKind value representing whether the base type is used for input or output.
//
// The method first calls the getIntrospectionBaseType method to get the introspection type of the base type.
//
// If the base type is a slice, the method wraps the introspection type as a list. If the base type is not
// a pointer slice, the method also wraps the introspection type as a non-null type.
//
// If the base type is not a pointer, the method wraps the introspection type as a non-null type.
//
// The method returns a pointer to the modified introspection type.
func (g *Graphy) getIntrospectionModifiedType(is *__Schema, tl *typeLookup, io TypeKind) *__Type {
	// Get the introspection type of the base type
	ret := g.getIntrospectionBaseType(is, tl, io)

	// For output types with implementations (not interface-only), we should return the interface type
	// This matches the behavior in schemaRefForType
	if io == TypeOutput && len(tl.implementedBy) > 0 && !tl.interfaceOnly {
		baseName := g.schemaBuffer.outputTypeNameLookup[tl]
		interfaceName := "I" + baseName
		if interfaceType, ok := is.typeLookupByName[interfaceName]; ok {
			ret = interfaceType
		}
	}

	// If the base type is a slice, wrap the introspection type as a list
	if tl.array != nil {
		ret = g.wrapArrayTypes(ret, tl.array)
	}

	// If the base type is not a pointer, wrap the introspection type as a non-null type
	if !tl.isPointer {
		ret = g.wrapType(ret, "", IntrospectionKindNonNull) // Empty name for NON_NULL wrapper
	}

	// Return the modified introspection type
	return ret
}

func (g *Graphy) wrapArrayTypes(ret *__Type, array *typeArrayModifier) *__Type {
	if array.array != nil {
		ret = g.wrapArrayTypes(ret, array.array)
	}
	if !array.isPointer {
		ret = g.wrapType(ret, "", IntrospectionKindNonNull) // Empty name for NON_NULL wrapper
	}
	ret = g.wrapType(ret, "", IntrospectionKindList) // Empty name for LIST wrapper
	return ret
}

// wrapType is a method of the Graphy struct. It is used to create a new __Type struct
// that wraps a given type with a given name and kind. This is used in the process of
// generating the introspection schema of a GraphQL server, specifically when modifying
// a base type to represent a list or a non-null type.
//
// The method takes three parameters:
// - t: a pointer to the __Type struct to be wrapped.
// - name: a string representing the name of the new type. This is typically "list" or "required".
// - kind: a __TypeKind value representing the kind of the new type. This is typically IntrospectionKindList or IntrospectionKindNonNull.
//
// The method returns a pointer to the new __Type struct. The new type has the given name and kind,
// and its OfType field is set to the given type. This represents that the new type is a list of
// or a non-null version of the given type.
func (g *Graphy) wrapType(t *__Type, name string, kind __TypeKind) *__Type {
	// NON_NULL and LIST wrapper types should have null names in GraphQL introspection
	if kind == IntrospectionKindNonNull || kind == IntrospectionKindList {
		return &__Type{
			Name:   nil, // Null name for wrapper types per GraphQL spec
			Kind:   kind,
			OfType: t,
		}
	}
	// For other types, use the provided name
	return &__Type{
		Name:   &name,
		Kind:   kind,
		OfType: t,
	}
}
