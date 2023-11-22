package quickgraph

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type TypeKind int

const (
	TypeInput TypeKind = iota
	TypeOutput
)

func (g *Graphy) schemaForTypes(kind TypeKind, mapping typeNameMapping, types ...*typeLookup) string {

	completed := make(map[string]bool)

	typeQueue := make([]*typeLookup, len(types))

	copy(typeQueue, types)

	sb := strings.Builder{}
	for i := 0; i < len(typeQueue); i++ {
		if typeQueue[i] == nil {
			panic(fmt.Sprintf("typeQueue[%d] is nil", i))
		}
		name := mapping[typeQueue[i]]
		if completed[name] {
			continue
		}
		completed[name] = true
		t := typeQueue[i]
		if t.fundamental {
			continue
		}
		schema := g.schemaForType(kind, t, mapping)
		sb.WriteString(schema)
		sb.WriteString("\n")
	}

	return sb.String()
}

func (g *Graphy) schemaForEnumTypes(types ...*typeLookup) string {
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

	return sb.String()
}

func (g *Graphy) schemaForEnum(et *typeLookup) string {

	sb := strings.Builder{}

	enumValue := reflect.New(et.rootType)
	sev := enumValue.Convert(stringEnumValuesType)
	se := sev.Interface().(StringEnumValues)

	sb.WriteString("enum ")
	sb.WriteString(et.name)
	sb.WriteString(" {\n")

	for _, s := range se.EnumValues() {
		sb.WriteString("\t")
		sb.WriteString(s.Name) // TODO: Add deprecated support.
		sb.WriteString("\n")
	}
	sb.WriteString("}\n")
	return sb.String()
}

func (g *Graphy) schemaForType(kind TypeKind, t *typeLookup, mapping typeNameMapping) string {
	name := mapping[t]

	if len(t.union) > 0 {
		return g.schemaForUnion(name, t, mapping)
	}

	sb := &strings.Builder{}
	sb.WriteString(g.getSchemaTypePrefix(kind))
	sb.WriteString(name)
	sb.WriteString(g.getSchemaImplementedInterfaces(t, mapping))
	sb.WriteString(" {\n")
	sb.WriteString(g.getSchemaFields(t, kind, mapping))
	sb.WriteString("}\n")

	return sb.String()
}

func (g *Graphy) getSchemaTypePrefix(kind TypeKind) string {
	if kind == TypeInput {
		return "input "
	}
	return "type "
}

func (g *Graphy) getSchemaImplementedInterfaces(t *typeLookup, mapping typeNameMapping) string {
	if len(t.implements) == 0 {
		return ""
	}

	sb := &strings.Builder{}
	sb.WriteString(" implements")
	interfaceCount := 0
	for _, implementedType := range t.implements {
		if interfaceCount > 0 {
			sb.WriteString("& ")
		}
		interfaceCount++
		sb.WriteString(" ")
		sb.WriteString(mapping[implementedType])
	}

	return sb.String()
}

func (g *Graphy) getSchemaFields(t *typeLookup, kind TypeKind, mapping typeNameMapping) string {
	sb := &strings.Builder{}
	for _, name := range sortedKeys(t.fieldsLowercase) {
		field := t.fieldsLowercase[name]
		if len(field.fieldIndexes) > 1 {
			continue
		}

		fieldTypeString := g.getSchemaFieldType(&field, kind, mapping)
		if fieldTypeString == "" {
			continue
		}

		sb.WriteString("\t")
		sb.WriteString(field.name)
		sb.WriteString(fieldTypeString)

		if field.isDeprecated {
			sb.WriteString(" @deprecated(reason: \"")
			sb.WriteString(field.deprecatedReason)
			sb.WriteString("\")")
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

func (g *Graphy) getSchemaFieldType(field *fieldLookup, kind TypeKind, mapping typeNameMapping) string {
	switch field.fieldType {
	case FieldTypeField:
		return ": " + g.schemaRefForType(g.typeLookup(field.resultType), mapping)
	case FieldTypeGraphFunction:
		if kind == TypeOutput {
			return g.getSchemaGraphFunctionType(field, mapping)
		}
	}
	return ""
}

func (g *Graphy) getSchemaGraphFunctionType(field *fieldLookup, mapping typeNameMapping) string {
	sb := &strings.Builder{}
	if len(field.graphFunction.paramsByName) > 0 {
		sb.WriteString("(")
		sb.WriteString(g.schemaForFunctionParameters(field.graphFunction, mapping))
		sb.WriteString(")")
	}
	sb.WriteString(": ")
	sb.WriteString(g.schemaRefForType(field.graphFunction.baseReturnType, mapping))

	return sb.String()
}
func (g *Graphy) schemaForUnion(name string, t *typeLookup, mapping typeNameMapping) string {
	sb := strings.Builder{}
	sb.WriteString("union ")
	sb.WriteString(name)
	sb.WriteString(" =")
	unionCount := 0
	// Get the union names in alphabetical order.
	var unionNames []string
	for _, utl := range t.union {
		unionNames = append(unionNames, mapping[utl])
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
	}
	sb.WriteString("\n")
	return sb.String()
}

func (g *Graphy) schemaRefForType(t *typeLookup, mapping typeNameMapping) string {
	optional := t.isPointer
	array := t.isSlice
	optionalInner := t.isPointerSlice

	var baseType string
	if t.rootType == nil {
		baseType = t.name
	} else {
		switch t.rootType.Kind() {
		case reflect.String:
			if t.rootType.AssignableTo(stringEnumValuesType) {
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
			if t != nil {
				baseType = mapping[t]
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

	return work
}
