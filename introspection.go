package quickgraph

import (
	"context"
	"fmt"
	"reflect"
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
	FieldsRaw      []__Field  `json:"fields"`
	Interfaces     []*__Type  `json:"interfaces"`
	PossibleTypes  []*__Type  `json:"possibleTypes"`
	EnumValuesRaw  []__EnumValue
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
	Type         __Type  `json:"type"`
	DefaultValue *string `json:"defaultValue"`
}

func (it *__Type) Fields(includeDeprecated bool) []__Field {
	if includeDeprecated {
		return it.FieldsRaw
	}
	result := []__Field{}
	for _, field := range it.FieldsRaw {
		if !field.IsDeprecated {
			result = append(result, field)
		}
	}
	return result
}

func (it *__Type) EnumValues(includeDeprecated bool) []__EnumValue {
	if includeDeprecated {
		return it.EnumValuesRaw
	}
	result := []__EnumValue{}
	for _, enumValue := range it.EnumValuesRaw {
		if !enumValue.IsDeprecated {
			result = append(result, enumValue)
		}
	}
	return result
}

func (g *Graphy) EnableIntrospection(ctx context.Context) {
	schemaFunc := func() *__Schema {
		st := g.getSchemaTypes()
		return st.introspectionSchema
	}
	typesFunc := func(name string) (*__Type, error) {
		st := g.getSchemaTypes()
		tl, ok := st.introspectionTypes[name]
		if !ok {
			return nil, fmt.Errorf("type %s not found", name)
		}
		return tl, nil
	}
	g.RegisterQuery(ctx, "__schema", schemaFunc)
	g.RegisterQuery(ctx, "__type", typesFunc, "name")
}

func (g *Graphy) populateIntrospection(st *schemaTypes) {

	queries := &__Type{
		Kind: IntrospectionKindObject,
		Name: "__query",
	}
	mutations := &__Type{
		Kind: IntrospectionKindObject,
		Name: "__mutation",
	}

	is := &__Schema{
		Queries:          queries,
		Mutations:        mutations,
		Types:            []*__Type{},
		typeLookupByName: map[string]*__Type{},
	}

	for _, f := range g.processors {
		if strings.HasPrefix(f.name, "__") {
			continue
		}
		t := g.introspectionCall(is, &f)
		qf := __Field{
			Name: f.name,
			Type: t,
		}
		t.Name = f.name
		switch f.mode {
		case ModeQuery:
			queries.FieldsRaw = append(queries.FieldsRaw, qf)
		case ModeMutation:
			mutations.FieldsRaw = append(mutations.FieldsRaw, qf)
		}
	}

	for _, refType := range is.typeLookupByName {
		is.Types = append(is.Types, refType)
	}

	is.Types = append(is.Types, queries)
	is.Types = append(is.Types, mutations)

	g.schemaBuffer.introspectionSchema = is
}

func (g *Graphy) getIntrospectionType(is *__Schema, tl *typeLookup, io TypeKind) *__Type {
	if existing, ok := is.typeLookupByName[tl.name]; ok {
		return existing
	}
	result := &__Type{
		Name: tl.name,
	}
	is.typeLookupByName[tl.name] = result
	if tl.isSlice {
		result.Kind = IntrospectionKindList
		result.OfType = g.getIntrospectionType(is, g.typeLookup(tl.rootType), io)
		return result
	}
	if len(tl.union) > 0 {
		result.Kind = IntrospectionKindUnion
		for _, ul := range tl.union {
			result.PossibleTypes = append(result.PossibleTypes, g.getIntrospectionType(is, ul, io))
		}
		return result
	}
	if tl.rootType.Kind() == reflect.Interface {
		result.Kind = IntrospectionKindInterface
		// We don't have a good way of getting the objects that implement this interface.
		// TODO: Come up with something.
		return result
	}
	if tl.rootType.ConvertibleTo(stringEnumValuesType) {
		result.Kind = IntrospectionKindEnum

		// Create an instance of the enum type and get the values
		enumValue := reflect.New(tl.rootType)
		sev := enumValue.Convert(stringEnumValuesType)
		se := sev.Interface().(StringEnumValues)
		result.EnumValuesRaw = []__EnumValue{}
		for _, s := range se.EnumValues() {
			result.EnumValuesRaw = append(result.EnumValuesRaw, __EnumValue{
				Name: s,
			})
		}
		return result
	}
	if tl.fundamental {
		result.Kind = IntrospectionKindScalar
		return result
	}
	if io == TypeInput {
		result.Kind = IntrospectionKindInputObject
	} else {
		result.Kind = IntrospectionKindObject
	}
	for name, ft := range tl.fields {
		if ft.fieldType == FieldTypeField {
			result.FieldsRaw = append(result.FieldsRaw, __Field{
				Name: name,
				Type: g.getIntrospectionType(is, g.typeLookup(ft.resultType), io),
			})
		} else if ft.fieldType == FieldTypeGraphFunction {
			result.FieldsRaw = append(result.FieldsRaw, __Field{
				Name: name,
				Type: g.introspectionCall(is, ft.graphFunction),
			})
		}
	}
	return result
}

func (g *Graphy) introspectionCall(is *__Schema, f *graphFunction) *__Type {
	result := g.getIntrospectionType(is, f.baseReturnType, TypeOutput)
	for _, param := range f.nameMapping {
		result.FieldsRaw = append(result.FieldsRaw, __Field{
			Name: param.name,
			Type: g.getIntrospectionType(is, g.typeLookup(param.paramType), TypeInput),
		})
	}
	return result
}
