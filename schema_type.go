package quickgraph

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

func (g *Graphy) schemaForOutputTypes(types ...*TypeLookup) (string, []*TypeLookup, error) {

	completed := make(map[string]bool)

	typeQueue := make([]*TypeLookup, len(types))
	var enumQueue []*TypeLookup

	copy(typeQueue, types)

	sb := strings.Builder{}
	for i := 0; i < len(typeQueue); i++ {
		if typeQueue[i] == nil {
			panic(fmt.Sprintf("typeQueue[%d] is nil", i))
		}
		if completed[typeQueue[i].name] {
			continue
		}
		completed[typeQueue[i].name] = true
		t := typeQueue[i]
		if t.fundamental {
			continue
		}
		schema, extra, err := g.schemaForOutputType(t)
		if err != nil {
			return "", nil, err
		}
		for _, et := range extra {
			if et.rootType.Kind() != reflect.Invalid && et.rootType.AssignableTo(stringEnumValuesType) {
				enumQueue = append(enumQueue, et)
			} else {
				etl := et
				if !completed[et.name] && etl != nil {
					typeQueue = append(typeQueue, etl)
				}
			}
		}
		sb.WriteString(schema)
		sb.WriteString("\n")
	}

	return sb.String(), enumQueue, nil
}

func (g *Graphy) schemaForEnumTypes(types ...*TypeLookup) (string, error) {
	sb := strings.Builder{}

	completed := make(map[string]bool)

	for _, et := range types {
		enumName := et.name
		if completed[enumName] {
			continue
		}
		completed[enumName] = true

		sb.WriteString(g.schemaForEnum(et))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

func (g *Graphy) schemaForEnum(et *TypeLookup) string {

	sb := strings.Builder{}

	enumValue := reflect.New(et.rootType)
	sev := enumValue.Convert(stringEnumValuesType)
	se := sev.Interface().(StringEnumValues)

	sb.WriteString("enum ")
	sb.WriteString(et.name)
	sb.WriteString(" {\n")

	for _, s := range se.EnumValues() {
		sb.WriteString("\t")
		sb.WriteString(s)
		sb.WriteString("\n")
	}
	sb.WriteString("}\n")
	return sb.String()
}

func (g *Graphy) schemaForOutputType(t *TypeLookup) (string, []*TypeLookup, error) {
	var extraTypes []*TypeLookup

	// TODO: this can use some refactoring -- the function seems too complex as it is.
	if len(t.union) > 0 {
		sb := strings.Builder{}
		sb.WriteString("union ")
		sb.WriteString(t.name)
		sb.WriteString(" =")
		unionCount := 0
		// Get the union names in alphabetical order.
		var unionNames []string
		for n := range t.union {
			unionNames = append(unionNames, n)
		}
		sort.Strings(unionNames)
		for _, unionName := range unionNames {
			unionType := t.union[unionName]
			sb.WriteString(" ")
			if unionCount > 0 {
				sb.WriteString("| ")
			}
			unionCount++
			sb.WriteString(unionType.name)
			extraTypes = append(extraTypes, unionType)
		}
		sb.WriteString("\n")
		return sb.String(), extraTypes, nil
	}

	sb := strings.Builder{}
	sb.WriteString("type ")
	sb.WriteString(t.name)

	if len(t.implements) > 0 {
		sb.WriteString(" implements")
		interfaceCount := 0
		for _, implementedType := range t.implements {
			sb.WriteString(" ")
			if interfaceCount > 0 {
				sb.WriteString("& ")
			}
			interfaceCount++
			sb.WriteString(implementedType.name)
			extraTypes = append(extraTypes, implementedType)
		}
	}

	sb.WriteString(" {\n")

	// Get the field names in alphabetical order.
	var fieldNames []string
	for n := range t.fieldsLowercase {
		fieldNames = append(fieldNames, n)
	}
	sort.Strings(fieldNames)

	for _, name := range fieldNames {
		field := t.fieldsLowercase[name]
		if field.fieldType == FieldTypeField {
			if len(field.fieldIndexes) > 1 {
				// These are going to be either union or implemented interfaces. These need
				// to be handled differently.
				continue
			}
			typeString, extraType := g.schemaRefForType(g.typeLookup(field.resultType))
			if extraType != nil {
				extraTypes = append(extraTypes, extraType)
			}
			sb.WriteString("\t")
			sb.WriteString(field.name)
			sb.WriteString(": ")
			sb.WriteString(typeString)
			sb.WriteString("\n")
		} else if field.fieldType == FieldTypeGraphFunction {
			if len(field.fieldIndexes) > 1 {
				// These are going to be either union or implemented interfaces. These need
				// to be handled differently.
				continue
			}
			sb.WriteString("\t")
			sb.WriteString(field.name)
			sb.WriteString("(")
			funcParams, fEnums, err := g.schemaForFunctionParameters(field.graphFunction)
			if err != nil {
				return "", nil, err
			}
			extraTypes = append(extraTypes, fEnums...)
			sb.WriteString(funcParams)
			sb.WriteString("): ")
			schemaRef, _ := g.schemaRefForType(field.graphFunction.baseReturnType)
			sb.WriteString(schemaRef)
			sb.WriteString("\n")
		} else {
			panic("unknown field type")
		}
	}

	sb.WriteString("}\n")
	return sb.String(), extraTypes, nil
}

func (g *Graphy) schemaRefForType(t *TypeLookup) (string, *TypeLookup) {
	var extraType *TypeLookup

	optional := t.isPointer
	array := t.isSlice
	optionalInner := t.isPointerSlice

	var baseType string
	if t.rootType == nil {
		baseType = t.name
		extraType = t
	} else {
		switch t.rootType.Kind() {
		case reflect.String:
			if t.rootType.AssignableTo(stringEnumValuesType) {
				extraType = t
				baseType = t.name
			} else {
				baseType = "String"
			}

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			baseType = "Int"

		case reflect.Float32, reflect.Float64:
			baseType = "Float"

		case reflect.Bool:
			baseType = "Boolean"

		case reflect.Struct:
			tl := t
			if tl != nil {
				// TODO: Handle same type name in different packages.
				baseType = tl.name
			}

		default:
			panic("unsupported type")
		}
	}

	work := baseType
	if array {
		if !optionalInner {
			work = work + "!"
		}
		work = "[" + work + "]"
	}
	if !optional {
		work = work + "!"
	}

	return work, extraType
}
