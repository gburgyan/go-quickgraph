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

	// Add built-in scalar types to the introspection
	g.addBuiltInScalars(is)

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
	// Do a final pass to collect any interface types that were created during processing
	// but not yet added to is.Types
	for name, typ := range is.typeLookupByName {
		found := false
		for _, existing := range is.Types {
			if existing.Name != nil && *existing.Name == name {
				found = true
				break
			}
		}
		if !found {
			is.Types = append(is.Types, typ)
		}
	}

	// Post-process to ensure concrete types with interfaces are properly linked
	// This handles cases like Employee implementing IEmployee
	for _, typ := range is.Types {
		if typ.Kind == IntrospectionKindObject && typ.Name != nil {
			typeName := *typ.Name
			interfaceName := "I" + typeName
			if interfaceType, ok := is.typeLookupByName[interfaceName]; ok {
				if interfaceType.Kind == IntrospectionKindInterface {
					// Check if this type already implements the interface
					implementsInterface := false
					for _, iface := range typ.Interfaces {
						if iface.Name != nil && *iface.Name == interfaceName {
							implementsInterface = true
							break
						}
					}
					if !implementsInterface {
						typ.Interfaces = append(typ.Interfaces, interfaceType)
					}

					// Check if this type is in the interface's possible types
					inPossibleTypes := false
					for _, pt := range interfaceType.PossibleTypes {
						if pt.Name != nil && *pt.Name == typeName {
							inPossibleTypes = true
							break
						}
					}
					if !inPossibleTypes {
						interfaceType.PossibleTypes = append(interfaceType.PossibleTypes, typ)
					}
				}
			}
		}
	}

	g.schemaBuffer.introspectionSchema = is
}

// getIntrospectionBaseType creates a GraphQL introspection type from a type lookup.
// This is the main entry point for converting internal type representations to GraphQL introspection format.
// It handles the complete type creation workflow including name resolution, caching, and delegation to specialized creators.
func (g *Graphy) getIntrospectionBaseType(is *__Schema, tl *typeLookup, io TypeKind) *__Type {
	// Resolve the GraphQL type name
	name := g.resolveIntrospectionTypeName(tl, io)

	// Handle interface/concrete type split for types with implementations
	if io == TypeOutput && len(tl.implementedBy) > 0 && !tl.interfaceOnly {
		return g.handleInterfaceConcreteTypeSplit(is, tl, io, name)
	}

	// Return existing type if already created
	if existing, ok := is.typeLookupByName[name]; ok {
		return existing
	}

	// Create new type and add to lookup
	result := &__Type{Name: &name}
	is.typeLookupByName[name] = result

	// Delegate to specialized type creators based on type characteristics
	switch {
	case len(tl.union) > 0:
		g.createUnionIntrospectionType(is, tl, io, result)
	case tl.rootType != nil && tl.rootType.ConvertibleTo(stringEnumValuesType):
		g.createEnumIntrospectionType(tl, result)
	case tl.fundamental:
		result.Kind = IntrospectionKindScalar
	case io == TypeInput:
		result.Kind = IntrospectionKindInputObject
		g.addIntrospectionSchemaFields(is, tl, io, result)
	default:
		g.createObjectIntrospectionType(is, tl, io, result)
	}

	return result
}

// resolveIntrospectionTypeName determines the GraphQL type name for a type lookup.
// It handles enum types, fundamental types, and input/output type distinctions.
// Returns the appropriate name based on the type's characteristics and usage context.
func (g *Graphy) resolveIntrospectionTypeName(tl *typeLookup, io TypeKind) string {
	if tl.rootType != nil && tl.rootType.ConvertibleTo(stringEnumValuesType) {
		return g.schemaBuffer.enumTypeNameLookup[tl]
	}
	if tl.fundamental {
		if otlName, ok := g.schemaBuffer.outputTypeNameLookup[tl]; ok {
			return otlName
		}
		// Check for custom scalar first
		if scalar, exists := g.GetScalarByType(tl.rootType); exists {
			return scalar.Name
		}
		// If no custom scalar found but it's fundamental, try built-in scalar names
		if tl.rootType != nil {
			result := introspectionScalarName(tl)
			if result == "" {
				// This is a fundamental type that's not a basic Go type and not a registered custom scalar
				// This shouldn't happen in normal operation
				return fmt.Sprintf("UnknownFundamental_%v", tl.rootType.Kind())
			}
			return result
		}
		// If rootType is nil, we have a problem - return a fallback name with debug info
		return fmt.Sprintf("UnknownScalar_%v", tl.typ)
	}
	if io == TypeOutput {
		return g.schemaBuffer.outputTypeNameLookup[tl]
	}
	if io == TypeInput {
		return g.schemaBuffer.inputTypeNameLookup[tl]
	}
	// This should never happen with current TypeKind enum values
	panic("unknown IO type")
}

// handleInterfaceConcreteTypeSplit manages types that need both interface and concrete representations.
// This occurs when a type has implementations but is not marked as interface-only.
// It creates an interface type with "I" prefix and returns a concrete type that implements it.
func (g *Graphy) handleInterfaceConcreteTypeSplit(is *__Schema, tl *typeLookup, io TypeKind, name string) *__Type {
	interfaceName := "I" + name

	// Create interface type if it doesn't exist
	if _, ok := is.typeLookupByName[interfaceName]; !ok {
		interfaceType := &__Type{
			Name: &interfaceName,
			Kind: IntrospectionKindInterface,
		}
		is.typeLookupByName[interfaceName] = interfaceType

		// Add fields to interface
		g.addIntrospectionSchemaFields(is, tl, io, interfaceType)

		// Add concrete type as possible implementation
		concreteType := &__Type{
			Name: &name,
			Kind: IntrospectionKindObject,
		}
		interfaceType.PossibleTypes = append(interfaceType.PossibleTypes, concreteType)

		// Add other implementations
		for _, impl := range tl.implementedBy {
			implType := g.getIntrospectionBaseType(is, impl, io)
			interfaceType.PossibleTypes = append(interfaceType.PossibleTypes, implType)
		}
	}

	// Return existing concrete type or create new one
	if existing, ok := is.typeLookupByName[name]; ok {
		return existing
	}

	result := &__Type{Name: &name}
	is.typeLookupByName[name] = result
	return result
}

// createUnionIntrospectionType populates a union type with its possible concrete types.
// It resolves all union members, handling interface implementations correctly,
// and ensures only concrete types (not interfaces) are included in the union.
func (g *Graphy) createUnionIntrospectionType(is *__Schema, tl *typeLookup, io TypeKind, result *__Type) {
	result.Kind = IntrospectionKindUnion
	concreteTypes := make(map[string]*__Type)

	for _, name := range sortedKeys(tl.union) {
		ul := tl.union[name]
		baseType := g.getIntrospectionBaseType(is, ul, io)

		// Check if this type has implementations (is an interface)
		hasImplementations := len(ul.implementedBy) > 0
		if !hasImplementations && ul.rootType != nil && ul.rootType != ul.typ {
			if rootLookup, ok := g.typeLookups[ul.rootType]; ok {
				hasImplementations = len(rootLookup.implementedBy) > 0
			}
		}

		if hasImplementations {
			// For interface types, add concrete implementations to union
			checkType := ul
			if ul.rootType != nil && ul.rootType != ul.typ {
				if rootLookup, ok := g.typeLookups[ul.rootType]; ok {
					checkType = rootLookup
				}
			}

			// Add all implementations
			for _, impl := range checkType.implementedBy {
				implType := g.getIntrospectionBaseType(is, impl, io)
				if implType.Name != nil {
					concreteTypes[*implType.Name] = implType
				}
			}

			// Include concrete type if not interface-only
			if !checkType.interfaceOnly && baseType.Name != nil {
				concreteTypes[*baseType.Name] = baseType
			}
		} else {
			// For regular types, add directly to union
			if baseType.Name != nil {
				concreteTypes[*baseType.Name] = baseType
			}
		}
	}

	// Sort and add concrete types to union
	var typeNames []string
	for name := range concreteTypes {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)

	for _, name := range typeNames {
		result.PossibleTypes = append(result.PossibleTypes, concreteTypes[name])
	}
}

// createEnumIntrospectionType populates an enum type with its values and metadata.
// It extracts enum values from types implementing StringEnumValues interface,
// including descriptions and deprecation information.
func (g *Graphy) createEnumIntrospectionType(tl *typeLookup, result *__Type) {
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
}

// createObjectIntrospectionType creates an object type with fields and interface relationships.
// It handles the complex logic of interface implementations and embedded interfaces,
// ensuring proper GraphQL schema representation of Go type relationships.
func (g *Graphy) createObjectIntrospectionType(is *__Schema, tl *typeLookup, io TypeKind, result *__Type) {
	result.Kind = IntrospectionKindObject
	g.addIntrospectionSchemaFields(is, tl, io, result)

	// Add interface relationships for embedded interfaces
	for _, impls := range sortedKeys(tl.implements) {
		implTl := tl.implements[impls]
		interfaceName := g.resolveEmbeddedInterfaceName(implTl, io)

		if interfaceType, ok := is.typeLookupByName[interfaceName]; ok {
			result.Interfaces = append(result.Interfaces, interfaceType)
			interfaceType.PossibleTypes = append(interfaceType.PossibleTypes, result)
		} else {
			// Create missing interface type
			interfaceType := &__Type{
				Name: &interfaceName,
				Kind: IntrospectionKindInterface,
			}
			is.typeLookupByName[interfaceName] = interfaceType
			g.addIntrospectionSchemaFields(is, implTl, io, interfaceType)
			interfaceType.PossibleTypes = []*__Type{result}
			result.Interfaces = append(result.Interfaces, interfaceType)
		}
	}
}

// resolveEmbeddedInterfaceName determines the correct interface name for embedded types.
// It applies the "I" prefix pattern when appropriate and handles type name resolution.
func (g *Graphy) resolveEmbeddedInterfaceName(implTl *typeLookup, io TypeKind) string {
	var interfaceName string
	if implTl.rootType != nil && implTl.rootType.ConvertibleTo(stringEnumValuesType) {
		interfaceName = g.schemaBuffer.enumTypeNameLookup[implTl]
	} else if io == TypeOutput || implTl.fundamental {
		interfaceName = g.schemaBuffer.outputTypeNameLookup[implTl]
	} else {
		interfaceName = g.schemaBuffer.inputTypeNameLookup[implTl]
	}

	// Apply "I" prefix pattern for interface types with implementations
	if len(implTl.implementedBy) > 0 && !implTl.interfaceOnly {
		interfaceName = "I" + interfaceName
	}

	return interfaceName
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
		// For types that don't match basic Go types but are marked as fundamental,
		// return an empty string to indicate they should be handled by custom scalar logic
		return ""
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
	checkType := tl
	if tl.rootType != nil && tl.rootType != tl.typ {
		// For pointer/slice types, check the underlying type
		if rootLookup, ok := g.typeLookups[tl.rootType]; ok {
			checkType = rootLookup
		}
	}

	if io == TypeOutput && len(checkType.implementedBy) > 0 && !checkType.interfaceOnly {
		// Ensure we have the schema buffer (it should be available from populateIntrospection)
		if g.schemaBuffer != nil && g.schemaBuffer.outputTypeNameLookup != nil {
			baseName := g.schemaBuffer.outputTypeNameLookup[checkType]
			interfaceName := "I" + baseName
			if interfaceType, ok := is.typeLookupByName[interfaceName]; ok {
				ret = interfaceType
			}
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

// addBuiltInScalars adds the standard GraphQL scalar types to the introspection schema.
// These are the built-in scalar types that are always available in GraphQL.
func (g *Graphy) addBuiltInScalars(is *__Schema) {
	// Define the built-in scalar types
	builtInScalars := []string{"String", "Int", "Float", "Boolean", "ID"}

	for _, scalarName := range builtInScalars {
		name := scalarName // Create a new variable to avoid pointer issues
		scalarType := &__Type{
			Name: &name,
			Kind: IntrospectionKindScalar,
		}
		is.typeLookupByName[name] = scalarType
	}
}
