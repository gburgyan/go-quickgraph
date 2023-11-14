package quickgraph

import (
	"context"
	"sort"
	"strings"
)

type usageMap map[*typeLookup]bool

type typeNameLookup map[string]*typeLookup
type typeNameMapping map[*typeLookup]string

type schemaTypes struct {
	inputTypes  []*typeLookup
	outputTypes []*typeLookup
	enumTypes   []*typeLookup

	inputTypeNameLookup  typeNameMapping
	outputTypeNameLookup typeNameMapping
	enumTypeNameLookup   typeNameMapping

	inputTypesByName  typeNameLookup
	outputTypesByName typeNameLookup
	enumTypesByName   typeNameLookup

	introspectionSchema *__Schema
}

func (g *Graphy) SchemaDefinition(ctx context.Context) string {
	g.structureLock.RLock()
	defer g.structureLock.RUnlock()

	st := g.getSchemaTypes()

	sb := strings.Builder{}

	procByMode := map[GraphFunctionMode][]*graphFunction{}

	for _, function := range g.processors {
		function := function
		if strings.HasPrefix(function.name, "__") {
			continue
		}
		byMode, ok := procByMode[function.mode]
		if !ok {
			byMode = []*graphFunction{}
			procByMode[function.mode] = byMode
		}
		procByMode[function.mode] = append(byMode, &function)
	}

	for mode, functions := range procByMode {
		sb.WriteString("type ")
		switch mode {
		case ModeQuery:
			sb.WriteString("Query")
		case ModeMutation:
			sb.WriteString("Mutation")
		default:
			panic("unknown mode")
		}
		sb.WriteString(" {\n")

		// Sort the functions by name
		sort.Slice(functions, func(i, j int) bool {
			return functions[i].name < functions[j].name
		})

		for _, function := range functions {
			sb.WriteString("\t")
			sb.WriteString(function.name)
			if len(function.nameMapping) > 0 {
				sb.WriteString("(")
				funcParams := g.schemaForFunctionParameters(function, st.inputTypeNameLookup)
				sb.WriteString(funcParams)
				sb.WriteString(")")
			}

			sb.WriteString(": ")
			schemaRef := g.schemaRefForType(function.baseReturnType, st.outputTypeNameLookup)

			sb.WriteString(schemaRef)
			sb.WriteString("\n")
		}
		sb.WriteString("}\n\n")
	}

	inputSchema := g.schemaForTypes(TypeInput, st.inputTypeNameLookup, st.inputTypes...)
	sb.WriteString(inputSchema)

	outputSchema := g.schemaForTypes(TypeOutput, st.outputTypeNameLookup, st.outputTypes...)
	sb.WriteString(outputSchema)

	enumSchema := g.schemaForEnumTypes(st.enumTypes...)
	sb.WriteString(enumSchema)

	return sb.String()
}

func (g *Graphy) getSchemaTypes() *schemaTypes {
	// We're already in a structure lock, so we are good making this check without
	// a lock.
	if g.schemaBuffer != nil {
		return g.schemaBuffer
	}

	// Only one goroutine should be able to get here at a time.
	g.schemaLock.Lock()
	defer g.schemaLock.Unlock()

	// Check again in case it was set while waiting for the lock
	if g.schemaBuffer != nil {
		return g.schemaBuffer
	}

	var outputTypes []*typeLookup
	var inputTypes []*typeLookup
	var enumTypes []*typeLookup

	for _, proc := range g.processors {
		if strings.HasPrefix(proc.name, "__") {
			// These are internal functions, so we can skip them.
			continue
		}
		function := &proc
		inputMap := make(usageMap)
		outputMap := make(usageMap)

		g.functionIO(function, inputMap, outputMap)

		fInput := keys(inputMap)
		fOutput := keys(outputMap)

		// Add these to the global lists.
		outputTypes = append(outputTypes, fOutput...)
		inputTypes = append(inputTypes, fInput...)

		for _, outTypeLookup := range fOutput {
			if outTypeLookup.rootType != nil {
				if outTypeLookup.rootType.AssignableTo(stringEnumValuesType) {
					enumTypes = append(enumTypes, outTypeLookup)
				}
			} else {
				outputTypes = append(outputTypes, outTypeLookup)
			}
		}

		for _, inTypeLookup := range fInput {
			if inTypeLookup.rootType != nil {
				if inTypeLookup.rootType.AssignableTo(stringEnumValuesType) {
					enumTypes = append(enumTypes, inTypeLookup)
				}
			} else {
				inputTypes = append(inputTypes, inTypeLookup)
			}
		}
	}

	inputTypes = g.expandTypeLookups(inputTypes)
	outputTypes = g.expandTypeLookups(outputTypes)

	inputMapping, outputMapping := solveInputOutputNameMapping(inputTypes, outputTypes)
	enumMapping := typeNameMapping{}
	for _, enumType := range enumTypes {
		enumMapping[enumType] = enumType.name
	}

	g.schemaBuffer = &schemaTypes{
		inputTypes:  inputTypes,
		outputTypes: outputTypes,
		enumTypes:   enumTypes,

		inputTypeNameLookup:  inputMapping,
		outputTypeNameLookup: outputMapping,
		enumTypeNameLookup:   enumMapping,

		inputTypesByName:  makeTypeNameLookup(inputMapping),
		outputTypesByName: makeTypeNameLookup(outputMapping),
		enumTypesByName:   makeTypeNameLookup(enumMapping),
	}

	g.populateIntrospection(g.schemaBuffer)

	return g.schemaBuffer
}

func makeTypeNameLookup(t typeNameMapping) typeNameLookup {
	result := make(typeNameLookup)
	for tl, name := range t {
		result[name] = tl
	}
	return result
}

func solveInputOutputNameMapping(inputTypes []*typeLookup, outputTypes []*typeLookup) (typeNameMapping, typeNameMapping) {
	// TODO: Handle same type name in different packages.

	inputMapping := make(typeNameMapping)
	outputMapping := make(typeNameMapping)

	outputNames := map[string]bool{}

	// Populate outputMapping and check for name collisions along the way
	for _, outputType := range outputTypes {
		outputMapping[outputType] = outputType.name
		outputNames[outputType.name] = true
	}

	// Populate inputMapping, checking for name collisions and resolving them by appending "Input"
	for _, inputType := range inputTypes {
		name := inputType.name
		_, exists := outputNames[name]
		if exists {
			// If a collision is found, append "Input" to the input type name
			name += "Input"
		}
		inputMapping[inputType] = name
	}

	return inputMapping, outputMapping
}

func (g *Graphy) expandTypeLookups(types []*typeLookup) []*typeLookup {
	expandedTypeMap := map[*typeLookup]bool{}
	for _, tl := range types {
		expandedTypeMap = g.recursiveAddTypeLookup(tl, expandedTypeMap)
	}
	expandedTypes := keys(expandedTypeMap)

	// Sort by name
	sort.Slice(expandedTypes, func(i, j int) bool {
		return expandedTypes[i].name < expandedTypes[j].name
	})

	return expandedTypes
}

func (g *Graphy) recursiveAddTypeLookup(tl *typeLookup, typeMap map[*typeLookup]bool) map[*typeLookup]bool {
	if typeMap[tl] {
		return typeMap
	}
	typeMap[tl] = true
	for _, tl := range tl.implements {
		typeMap = g.recursiveAddTypeLookup(tl, typeMap)
	}
	for _, tl := range tl.union {
		typeMap = g.recursiveAddTypeLookup(tl, typeMap)
	}
	for _, fl := range tl.fields {
		ftl := g.typeLookup(fl.resultType)
		typeMap = g.recursiveAddTypeLookup(ftl, typeMap)
	}
	return typeMap
}

func (g *Graphy) schemaForFunctionParameters(f *graphFunction, mapping typeNameMapping) string {
	sb := strings.Builder{}

	mappings := []functionNameMapping{}
	for _, param := range f.nameMapping {
		mappings = append(mappings, param)
	}
	// Sort by index
	sort.Slice(mappings, func(i, j int) bool {
		return mappings[i].paramIndex < mappings[j].paramIndex
	})

	for i, param := range mappings {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(param.name)
		sb.WriteString(": ")
		paramTl := g.typeLookup(param.paramType)
		schemaRef := g.schemaRefForType(paramTl, mapping)
		sb.WriteString(schemaRef)
	}

	return sb.String()
}

func (g *Graphy) functionIO(f *graphFunction, inputTypes, outputTypes usageMap) {

	for _, param := range f.nameMapping {
		g.typeIO(g.typeLookup(param.paramType), TypeInput, inputTypes, outputTypes)
	}

	g.typeIO(f.baseReturnType, TypeOutput, inputTypes, outputTypes)
}

func (g *Graphy) typeIO(tl *typeLookup, io TypeKind, inputTypes, outputTypes usageMap) {
	if io == TypeInput {
		if inputTypes[tl] {
			return
		}
		inputTypes[tl] = true
	} else {
		if outputTypes[tl] {
			return
		}
		outputTypes[tl] = true
	}

	for _, fl := range tl.fields {
		switch fl.fieldType {
		case FieldTypeField:
			g.typeIO(g.typeLookup(fl.resultType), io, inputTypes, outputTypes)

		case FieldTypeGraphFunction:
			g.functionIO(fl.graphFunction, inputTypes, outputTypes)

		default:
			panic("unknown field type")
		}
	}

	for _, tl := range tl.implements {
		g.typeIO(tl, io, inputTypes, outputTypes)
	}

	for _, tl := range tl.union {
		g.typeIO(tl, io, inputTypes, outputTypes)
	}
}
