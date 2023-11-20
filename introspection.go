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
	Name           string     `json:"name"`
	Description    *string    `json:"description"`
	fieldsRaw      []__Field
	Interfaces     []*__Type `json:"interfaces"`
	PossibleTypes  []*__Type `json:"possibleTypes"`
	enumValuesRaw  []__EnumValue
	InputFields    []__InputValue
	OfType         *__Type `json:"ofType"`
	SpecifiedByUrl string  `json:"specifiedByUrl"`
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
	queries := &__Type{Kind: IntrospectionKindObject, Name: "__query"}
	mutations := &__Type{Kind: IntrospectionKindObject, Name: "__mutation"}

	is := &__Schema{
		Queries:          queries,
		Mutations:        mutations,
		Types:            []*__Type{},
		typeLookupByName: make(map[string]*__Type),
	}

	processorNames := keys(g.processors)
	sort.Strings(processorNames)

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
		}
	}

	typeNames := keys(is.typeLookupByName)
	sort.Strings(typeNames)

	for _, name := range typeNames {
		is.Types = append(is.Types, is.typeLookupByName[name])
	}

	is.Types = append(is.Types, queries, mutations)
	g.schemaBuffer.introspectionSchema = is
}

func (g *Graphy) getIntrospectionBaseType(is *__Schema, tl *typeLookup, io TypeKind) *__Type {
	var name string

	if io == TypeOutput || tl.fundamental {
		name = g.schemaBuffer.outputTypeNameLookup[tl]
	} else if io == TypeInput {
		name = g.schemaBuffer.inputTypeNameLookup[tl]
	} else {
		panic("unknown IO type")
	}

	if existing, ok := is.typeLookupByName[name]; ok {
		return existing
	}

	result := &__Type{Name: name}
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
	case io == TypeInput:
		result.Kind = IntrospectionKindInputObject
	default:
		result.Kind = IntrospectionKindObject
		g.addIntrospectionSchemaFields(is, tl, io, result)
		for _, impls := range sortedKeys(tl.implements) {
			result.Interfaces = append(result.Interfaces, g.getIntrospectionModifiedType(is, tl.implements[impls], io))
		}
	}

	return result
}

func (g *Graphy) addIntrospectionSchemaFields(is *__Schema, tl *typeLookup, io TypeKind, result *__Type) {
	for _, fieldName := range sortedKeys(tl.fields) {
		ft := tl.fields[fieldName]
		if ft.fieldType == FieldTypeField {
			if io == TypeOutput {
				field := __Field{
					Name:         fieldName,
					Type:         g.getIntrospectionModifiedType(is, g.typeLookup(ft.resultType), io),
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
	result := g.getIntrospectionModifiedType(is, f.baseReturnType, TypeOutput)

	var args []__InputValue
	for _, param := range f.indexMapping {
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

	// If the base type is a slice, wrap the introspection type as a list
	if tl.isSlice {
		// If the base type is not a pointer slice, also wrap the introspection type as a non-null type
		if !tl.isPointerSlice {
			ret = g.wrapType(ret, "required", IntrospectionKindNonNull)
		}
		ret = g.wrapType(ret, "list", IntrospectionKindList)
	}

	// If the base type is not a pointer, wrap the introspection type as a non-null type
	if !tl.isPointer {
		ret = g.wrapType(ret, "required", IntrospectionKindNonNull)
	}

	// Return the modified introspection type
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
	return &__Type{
		Name:   name,
		Kind:   kind,
		OfType: t,
	}
}
