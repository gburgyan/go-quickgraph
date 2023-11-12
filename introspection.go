package quickgraph

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

type introspectionSchema struct {
	// Description string `json:"description"`
	Queries   *introspectionType `json:"queryType"`
	Mutations *introspectionType `json:"mutationType"`

	Types []*introspectionType `json:"types"`

	typeLookupByName map[string]*introspectionType
}

type introspectionType struct {
	Kind introspectionKind `json:"kind"`
	Name string            `json:"name"`
	// Description string `json:"description"`
	FieldsRaw      []introspectionField `json:"fields"`
	Interfaces     []*introspectionType `json:"interfaces"`
	PossibleTypes  []*introspectionType `json:"possibleTypes"`
	EnumValuesRaw  []introspectionEnumValue
	InputFields    []introspectionInputValue
	OfType         *introspectionType `json:"ofType"`
	SpecifiedByUrl string             `json:"specifiedByUrl"`
}

type introspectionEnumValue struct {
	Name string `json:"name"`
	// Description string `json:"description"`
	IsDeprecated      bool    `json:"isDeprecated"`
	DeprecationReason *string `json:"deprecationReason"`
}

type introspectionField struct {
	Name string `json:"name"`
	// Description string `json:"description"
	Args              []introspectionInputValue `json:"args"`
	Type              *introspectionType        `json:"type"`
	IsDeprecated      bool                      `json:"isDeprecated"`
	DeprecationReason *string                   `json:"deprecationReason"`
}

type introspectionKind string

const (
	IntrospectionKindScalar      introspectionKind = "SCALAR"
	IntrospectionKindObject      introspectionKind = "OBJECT"
	IntrospectionKindInterface   introspectionKind = "INTERFACE"
	IntrospectionKindUnion       introspectionKind = "UNION"
	IntrospectionKindEnum        introspectionKind = "ENUM"
	IntrospectionKindInputObject introspectionKind = "INPUT_OBJECT"
	IntrospectionKindList        introspectionKind = "LIST"
	IntrospectionKindNonNull     introspectionKind = "NON_NULL"
)

type introspectionInputValue struct {
	Name string `json:"name"`
	// Description string `json:"description"`
	Type introspectionType `json:"type"`
	// DefaultValue string `json:"defaultValue"`
}

func (it *introspectionType) Fields(includeDeprecated bool) []introspectionField {
	if includeDeprecated {
		return it.FieldsRaw
	}
	result := []introspectionField{}
	for _, field := range it.FieldsRaw {
		if !field.IsDeprecated {
			result = append(result, field)
		}
	}
	return result
}

func (it *introspectionType) EnumValues(includeDeprecated bool) []introspectionEnumValue {
	if includeDeprecated {
		return it.EnumValuesRaw
	}
	result := []introspectionEnumValue{}
	for _, enumValue := range it.EnumValuesRaw {
		if !enumValue.IsDeprecated {
			result = append(result, enumValue)
		}
	}
	return result
}

func (g *Graphy) EnableIntrospection(ctx context.Context) {
	schemaFunc := func() *introspectionSchema {
		st := g.getSchemaTypes()
		return st.introspectionSchema
	}
	typesFunc := func(name string) (*introspectionType, error) {
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

	queries := &introspectionType{
		Kind: IntrospectionKindObject,
		Name: "Query",
	}
	mutations := &introspectionType{
		Kind: IntrospectionKindObject,
		Name: "Mutation",
	}

	is := &introspectionSchema{
		Queries:          queries,
		Mutations:        mutations,
		Types:            []*introspectionType{},
		typeLookupByName: map[string]*introspectionType{},
	}

	for _, f := range g.processors {
		if strings.HasPrefix(f.name, "__") {
			continue
		}
		t := g.introspectionCall(is, &f)
		qf := introspectionField{
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

	g.schemaBuffer.introspectionSchema = is
}

func (g *Graphy) getIntrospectionType(is *introspectionSchema, tl *typeLookup, io TypeKind) *introspectionType {
	if existing, ok := is.typeLookupByName[tl.name]; ok {
		return existing
	}
	result := &introspectionType{
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
		result.EnumValuesRaw = []introspectionEnumValue{}
		for _, s := range se.EnumValues() {
			result.EnumValuesRaw = append(result.EnumValuesRaw, introspectionEnumValue{
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
			result.FieldsRaw = append(result.FieldsRaw, introspectionField{
				Name: name,
				Type: g.getIntrospectionType(is, g.typeLookup(ft.resultType), io),
			})
		} else if ft.fieldType == FieldTypeGraphFunction {
			result.FieldsRaw = append(result.FieldsRaw, introspectionField{
				Name: name,
				Type: g.introspectionCall(is, ft.graphFunction),
			})
		}
	}
	return result
}

func (g *Graphy) introspectionCall(is *introspectionSchema, f *graphFunction) *introspectionType {
	result := g.getIntrospectionType(is, f.baseReturnType, TypeOutput)
	for _, param := range f.nameMapping {
		result.FieldsRaw = append(result.FieldsRaw, introspectionField{
			Name: param.name,
			Type: g.getIntrospectionType(is, g.typeLookup(param.paramType), TypeInput),
		})
	}
	return result
}
