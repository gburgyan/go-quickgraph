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
	includeDeprecated := false
	if includeDeprecatedOpt != nil {
		includeDeprecated = *includeDeprecatedOpt
	}

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
	includeDeprecated := false
	if includeDeprecatedOpt != nil {
		includeDeprecated = *includeDeprecatedOpt
	}

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

	processorNames := keys(g.processors)
	// Sort the processors by name
	sort.Slice(processorNames, func(i, j int) bool {
		return processorNames[i] < processorNames[j]
	})

	for _, name := range processorNames {
		f := g.processors[name]
		if strings.HasPrefix(f.name, "__") {
			continue
		}
		t, args := g.introspectionCall(is, &f)
		qf := __Field{
			Name: f.name,
			Type: t,
			Args: args,
		}
		switch f.mode {
		case ModeQuery:
			queries.fieldsRaw = append(queries.fieldsRaw, qf)
		case ModeMutation:
			mutations.fieldsRaw = append(mutations.fieldsRaw, qf)
		}
	}

	typeNames := keys(is.typeLookupByName)
	// Sort the types by name
	sort.Slice(typeNames, func(i, j int) bool {
		return typeNames[i] < typeNames[j]
	})
	for _, name := range typeNames {
		refType := is.typeLookupByName[name]
		is.Types = append(is.Types, refType)
	}

	is.Types = append(is.Types, queries)
	is.Types = append(is.Types, mutations)

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

	result := &__Type{
		Name: name,
	}

	is.typeLookupByName[name] = result
	if len(tl.union) > 0 {
		result.Kind = IntrospectionKindUnion

		unionNames := keys(tl.union)
		// Sort the union names by name
		sort.Slice(unionNames, func(i, j int) bool {
			return unionNames[i] < unionNames[j]
		})

		for _, name := range unionNames {
			ul := tl.union[name]
			result.PossibleTypes = append(result.PossibleTypes, g.getIntrospectionModifiedType(is, ul, io))
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
		result.enumValuesRaw = []__EnumValue{}
		for _, s := range se.EnumValues() {
			result.enumValuesRaw = append(result.enumValuesRaw, __EnumValue{
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
	fieldNames := keys(tl.fields)
	// Sort the fields by name
	sort.Slice(fieldNames, func(i, j int) bool {
		return fieldNames[i] < fieldNames[j]
	})

	for _, fieldName := range fieldNames {
		ft := tl.fields[fieldName]
		if ft.fieldType == FieldTypeField {
			if io == TypeOutput {
				result.fieldsRaw = append(result.fieldsRaw, __Field{
					Name: fieldName,
					Type: g.getIntrospectionModifiedType(is, g.typeLookup(ft.resultType), io),
				})
			} else {
				result.InputFields = append(result.InputFields, __InputValue{
					Name: fieldName,
					Type: g.getIntrospectionModifiedType(is, g.typeLookup(ft.resultType), io),
				})
			}
		} else if ft.fieldType == FieldTypeGraphFunction {
			call, args := g.introspectionCall(is, ft.graphFunction)
			result.fieldsRaw = append(result.fieldsRaw, __Field{
				Name: fieldName,
				Type: call,
				Args: args,
			})
		}
	}
	return result
}

func (g *Graphy) introspectionCall(is *__Schema, f *graphFunction) (*__Type, []__InputValue) {
	result := g.getIntrospectionModifiedType(is, f.baseReturnType, TypeOutput)
	var args []__InputValue
	for _, param := range f.nameMapping {
		args = append(args, __InputValue{
			Name: param.name,
			Type: g.getIntrospectionModifiedType(is, g.typeLookup(param.paramType), TypeInput),
		})
	}
	return result, args
}

func (g *Graphy) getIntrospectionModifiedType(is *__Schema, tl *typeLookup, io TypeKind) *__Type {
	ret := g.getIntrospectionBaseType(is, tl, io)

	if tl.isSlice {
		if !tl.isPointerSlice {
			wrapper := &__Type{
				Name:   "required",
				Kind:   IntrospectionKindNonNull,
				OfType: ret,
			}
			ret = wrapper
		}

		wrapper := &__Type{
			Name:   "list",
			Kind:   IntrospectionKindList,
			OfType: ret,
		}
		ret = wrapper
	}

	if !tl.isPointer {
		wrapper := &__Type{
			Name:   "required",
			Kind:   IntrospectionKindNonNull,
			OfType: ret,
		}
		ret = wrapper
	}

	return ret
}
