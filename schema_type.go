package quickgraph

import (
	"reflect"
	"sort"
	"strings"
)

func (g *Graphy) schemaForType(t *TypeLookup) (string, error) {
	sb := strings.Builder{}
	sb.WriteString("type ")
	sb.WriteString(t.name)
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
			sb.WriteString("\t")
			sb.WriteString(field.name)
			sb.WriteString(": ")
			sb.WriteString(g.schemaForTypeLookup(field.resultType))
			sb.WriteString("\n")
		}
	}
	sb.WriteString("}\n")
	return sb.String(), nil
}

func (g *Graphy) schemaForTypeLookup(t reflect.Type) string {
	optional := false
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		optional = true
	}
	array := false
	optionalInner := false
	if t.Kind() == reflect.Slice {
		t = t.Elem()
		array = true
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
			optionalInner = true
		}
	}

	var baseType string
	switch t.Kind() {
	case reflect.String:
		baseType = "String"
	case reflect.Int:
	case reflect.Int8:
	case reflect.Int16:
	case reflect.Int32:
	case reflect.Int64:
	case reflect.Uint:
	case reflect.Uint8:
	case reflect.Uint16:
	case reflect.Uint32:
	case reflect.Uint64:
		baseType = "Int"

	case reflect.Float32:
	case reflect.Float64:
		baseType = "Float"

	case reflect.Bool:
		baseType = "Boolean"

	case reflect.Struct:
		tl := g.typeLookup(t)
		if tl != nil {
			// TODO: Handle same type name in different packages.
			baseType = tl.name
		}

	default:
		panic("unsupported type")
	}

	work := baseType
	if !optionalInner {
		work = work + "!"
	}
	if array {
		work = "[" + work + "]"
	}
	if optional {
		work = work + "!"
	}

	return work
}
