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

func (g *Graphy) schemaForTypes(kind TypeKind, mapping typeNameMapping, inputMapping typeNameMapping, types ...*typeLookup) string {

	completed := make(map[string]bool)

	typeQueue := make([]*typeLookup, len(types))

	copy(typeQueue, types)

	sb := strings.Builder{}
	for i := 0; i < len(typeQueue); i++ {
		if typeQueue[i] == nil {
			panic(fmt.Sprintf("typeQueue[%d] is nil", i))
		}
		t := typeQueue[i]
		name := mapping[t]

		// Skip if already processed
		if completed[name] {
			continue
		}

		if t.fundamental {
			continue
		}

		// Check if this type should generate both interface and concrete type
		// Also check the root type if this is a variant (pointer or slice)
		t.mu.RLock()
		hasImplementedBy := len(t.implementedBy) > 0
		t.mu.RUnlock()
		if !hasImplementedBy && t.rootType != nil && t.rootType != t.typ {
			if rootLookup, ok := g.typeLookups[t.rootType]; ok {
				rootLookup.mu.RLock()
				hasImplementedBy = len(rootLookup.implementedBy) > 0
				rootLookup.mu.RUnlock()
			}
		}
		if kind == TypeOutput && hasImplementedBy {
			if t.interfaceOnly {
				// Generate only interface with original name (no I prefix)
				completed[name] = true
				schema := g.schemaForType(kind, t, mapping, inputMapping)
				sb.WriteString(schema)
				sb.WriteString("\n")
			} else {
				// Generate both interface (with I prefix) and concrete type
				interfaceName := "I" + name
				if !completed[interfaceName] {
					completed[interfaceName] = true
					interfaceSchema := g.schemaForInterface(t, interfaceName, mapping, inputMapping)
					sb.WriteString(interfaceSchema)
					sb.WriteString("\n")
				}

				// Generate concrete type with original name
				if !completed[name] {
					completed[name] = true
					concreteSchema := g.schemaForConcreteType(t, name, mapping, inputMapping)
					sb.WriteString(concreteSchema)
					sb.WriteString("\n")
				}
			}
		} else {
			// Generate single type as before
			completed[name] = true
			schema := g.schemaForType(kind, t, mapping, inputMapping)
			sb.WriteString(schema)
			sb.WriteString("\n")
		}
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

func (g *Graphy) schemaForType(kind TypeKind, t *typeLookup, mapping typeNameMapping, inputMapping typeNameMapping) string {
	name := mapping[t]

	if len(t.union) > 0 {
		return g.schemaForUnion(name, t, mapping)
	}

	sb := &strings.Builder{}
	sb.WriteString(g.getSchemaTypePrefix(kind, t))
	sb.WriteString(name)
	sb.WriteString(g.getSchemaImplementedInterfaces(t, mapping))
	sb.WriteString(" {\n")
	sb.WriteString(g.getSchemaFields(t, kind, mapping, inputMapping))
	sb.WriteString("}\n")

	return sb.String()
}

func (g *Graphy) schemaForInterface(t *typeLookup, interfaceName string, mapping typeNameMapping, inputMapping typeNameMapping) string {
	sb := &strings.Builder{}
	sb.WriteString("interface ")
	sb.WriteString(interfaceName)
	sb.WriteString(" {\n")
	sb.WriteString(g.getSchemaFields(t, TypeOutput, mapping, inputMapping))
	sb.WriteString("}\n")
	return sb.String()
}

func (g *Graphy) schemaForConcreteType(t *typeLookup, name string, mapping typeNameMapping, inputMapping typeNameMapping) string {
	sb := &strings.Builder{}
	sb.WriteString("type ")
	sb.WriteString(name)
	// This concrete type implements the interface
	sb.WriteString(" implements I")
	sb.WriteString(name)
	sb.WriteString(" {\n")
	sb.WriteString(g.getSchemaFields(t, TypeOutput, mapping, inputMapping))
	sb.WriteString("}\n")
	return sb.String()
}

func (g *Graphy) getSchemaTypePrefix(kind TypeKind, t *typeLookup) string {
	if kind == TypeInput {
		return "input "
	}
	// If this type is implemented by other types (i.e., it's embedded in other structs),
	// then it should be rendered as an interface in GraphQL

	// Check if this type has implementedBy relationships
	t.mu.RLock()
	hasImplementedBy := len(t.implementedBy) > 0
	t.mu.RUnlock()

	// If not, and this is a pointer or slice type, check the underlying type
	if !hasImplementedBy && t.rootType != nil {
		// Look up the non-pointer version of the type
		nonPtrType := t.rootType
		if nonPtrType.Kind() == reflect.Ptr {
			nonPtrType = nonPtrType.Elem()
		}

		// Get the typeLookup for the non-pointer type
		if baseTl := g.typeLookup(nonPtrType); baseTl != nil && baseTl != t {
			baseTl.mu.RLock()
			hasImplementedBy = len(baseTl.implementedBy) > 0
			baseTl.mu.RUnlock()
			// Found implementedBy relationships in the base type
		}
	}

	if hasImplementedBy {
		return "interface "
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

	// Sort interface names for deterministic output
	var interfaceNames []string
	interfacesByName := make(map[string]*typeLookup)

	t.mu.RLock()
	implementsCopy := make(map[string]*typeLookup)
	for k, v := range t.implements {
		implementsCopy[k] = v
	}
	t.mu.RUnlock()

	for _, implementedType := range implementsCopy {
		name := mapping[implementedType]
		// Check if we need to add I prefix
		implementedType.mu.RLock()
		hasImplementedBy := len(implementedType.implementedBy) > 0
		implementedType.mu.RUnlock()
		if !hasImplementedBy && implementedType.rootType != nil && implementedType.rootType != implementedType.typ {
			if rootLookup, ok := g.typeLookups[implementedType.rootType]; ok {
				rootLookup.mu.RLock()
				hasImplementedBy = len(rootLookup.implementedBy) > 0
				rootLookup.mu.RUnlock()
			}
		}
		if hasImplementedBy && !implementedType.interfaceOnly {
			name = "I" + name
		}
		interfaceNames = append(interfaceNames, name)
		interfacesByName[name] = implementedType
	}
	sort.Strings(interfaceNames)

	for _, name := range interfaceNames {
		if interfaceCount > 0 {
			sb.WriteString(" & ")
		} else {
			sb.WriteString(" ")
		}
		interfaceCount++
		sb.WriteString(name)
	}

	return sb.String()
}

func (g *Graphy) getSchemaFields(t *typeLookup, kind TypeKind, mapping typeNameMapping, inputMapping typeNameMapping) string {
	sb := &strings.Builder{}

	// Use fieldsLowercase with sortedKeys as in the original implementation
	// The fields already include inherited fields from embedded structs
	t.mu.RLock()
	fieldsCopy := make(map[string]fieldLookup)
	for k, v := range t.fieldsLowercase {
		fieldsCopy[k] = v
	}
	t.mu.RUnlock()

	for _, name := range sortedKeys(fieldsCopy) {
		field := fieldsCopy[name]

		// Note: We don't skip fields with len(fieldIndexes) > 1 because
		// embedded struct fields have multiple indexes (e.g., [0 0] for the first field
		// of the first embedded struct) and we want to include those in the schema

		fieldTypeString := g.getSchemaFieldType(&field, kind, mapping, inputMapping)
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

func (g *Graphy) getSchemaFieldType(field *fieldLookup, kind TypeKind, mapping typeNameMapping, inputMapping typeNameMapping) string {
	switch field.fieldType {
	case FieldTypeField:
		return ": " + g.schemaRefForType(g.typeLookup(field.resultType), mapping)
	case FieldTypeGraphFunction:
		if kind == TypeOutput {
			return g.getSchemaGraphFunctionType(field, mapping, inputMapping)
		}
	}
	return ""
}

func (g *Graphy) getSchemaGraphFunctionType(field *fieldLookup, outputMapping typeNameMapping, inputMapping typeNameMapping) string {
	sb := &strings.Builder{}
	if len(field.graphFunction.paramsByName) > 0 {
		sb.WriteString("(")
		// Use input mapping for function parameters
		mappingToUse := inputMapping
		if mappingToUse == nil {
			mappingToUse = outputMapping // fallback for backwards compatibility
		}
		sb.WriteString(g.schemaForFunctionParameters(field.graphFunction, mappingToUse))
		sb.WriteString(")")
	}
	sb.WriteString(": ")
	// Use output mapping for return type
	sb.WriteString(g.schemaRefForType(field.graphFunction.baseReturnType, outputMapping))

	return sb.String()
}
func (g *Graphy) schemaForUnion(name string, t *typeLookup, mapping typeNameMapping) string {
	sb := strings.Builder{}
	sb.WriteString("union ")
	sb.WriteString(name)
	sb.WriteString(" =")

	// Collect all concrete types for the union
	// If a union member is an interface, we need to include all its implementations
	concreteTypes := make(map[string]*typeLookup)
	for _, utl := range t.union {
		// Check if this type has implementedBy relationships
		// We need to dereference pointer types to check the actual type
		checkType := utl
		if utl.isPointer && utl.rootType != nil {
			// For pointer types, check if the underlying type has implementations
			baseType := g.typeLookup(utl.rootType)
			if baseType != nil {
				checkType = baseType
			}
		}

		checkType.mu.RLock()
		hasImplementations := len(checkType.implementedBy) > 0
		implementations := make([]*typeLookup, len(checkType.implementedBy))
		copy(implementations, checkType.implementedBy)
		checkType.mu.RUnlock()

		if hasImplementations {
			// This is an interface, add all its implementations
			for _, impl := range implementations {
				concreteTypes[impl.name] = impl
			}
			// If we're not interface-only, include the concrete type too
			if !checkType.interfaceOnly {
				concreteTypes[checkType.name] = checkType
			}
		} else {
			// This is a concrete type, add it directly
			concreteTypes[utl.name] = utl
		}
	}

	// Sort the concrete type names
	var typeNames []string
	for name := range concreteTypes {
		typeNames = append(typeNames, name)
	}
	sort.Strings(typeNames)

	// Build the union string
	for i, typeName := range typeNames {
		if i > 0 {
			sb.WriteString(" |")
		}
		sb.WriteString(" ")
		sb.WriteString(typeName)
	}

	sb.WriteString("\n")
	return sb.String()
}

func (g *Graphy) schemaRefForType(t *typeLookup, mapping typeNameMapping) string {
	optional := t.isPointer

	var baseType string
	if t.rootType == nil {
		baseType = t.name
	} else {
		// Check for custom scalar first
		if scalar, exists := g.GetScalarByType(t.rootType); exists {
			baseType = scalar.Name
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
					// Check if this type has implementations and should be referenced as an interface
					if len(t.implementedBy) > 0 && !t.interfaceOnly {
						baseType = "I" + baseType
					}
				}

			case reflect.Interface:
				// Interfaces are represented as the GraphQL any type
				// If the interface has specific implementations registered,
				// they would be handled through the union mechanism
				baseType = t.name
				if baseType == "" {
					baseType = "Any"
				}

			default:
				panic(fmt.Sprintf("unsupported type: %v", t.rootType.Kind()))
			}
		}
	}

	work := baseType

	if t.array != nil {
		work = g.wrapSchemaArray(work, t.array)
	}

	if !optional {
		work = work + "!"
	}

	return work
}

func (g *Graphy) wrapSchemaArray(work string, array *typeArrayModifier) string {
	if array.array != nil {
		work = g.wrapSchemaArray(work, array.array)
	}
	if !array.isPointer {
		work = work + "!"
	}
	return "[" + work + "]"
}

func (g *Graphy) schemaForScalarTypes() string {
	if g.scalars == nil {
		return ""
	}

	g.scalars.mu.RLock()
	defer g.scalars.mu.RUnlock()

	if len(g.scalars.byName) == 0 {
		return ""
	}

	sb := strings.Builder{}

	// Sort scalar names for deterministic output
	var scalarNames []string
	for name := range g.scalars.byName {
		scalarNames = append(scalarNames, name)
	}
	sort.Strings(scalarNames)

	for _, name := range scalarNames {
		scalar := g.scalars.byName[name]
		sb.WriteString("scalar ")
		sb.WriteString(scalar.Name)
		if scalar.Description != "" {
			// Add description as a comment for now
			// In a full GraphQL implementation, this would use the description syntax
			sb.WriteString(" # ")
			sb.WriteString(scalar.Description)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	return sb.String()
}
