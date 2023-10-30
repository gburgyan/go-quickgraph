package quickgraph

import (
	"context"
	"sort"
	"strings"
)

func (g *Graphy) SchemaDefinition(ctx context.Context) (string, error) {
	sb := strings.Builder{}

	procByMode := map[GraphFunctionMode][]*graphFunction{}

	for _, function := range g.processors {
		function := function
		byMode, ok := procByMode[function.mode]
		if !ok {
			byMode = []*graphFunction{}
			procByMode[function.mode] = byMode
		}
		procByMode[function.mode] = append(byMode, &function)
	}

	outputTypes := []*typeLookup{}
	inputTypes := []*typeLookup{}
	enumTypes := []*typeLookup{}

	for _, functions := range procByMode {
		for _, function := range functions {
			_, fOuput, fInput := g.schemaForFunctionParameters(function, nil)

			outputTypes = append(outputTypes, function.baseReturnType)
			inputTypes = append(inputTypes, fInput...)

			for _, outTypeLookup := range fOuput {
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
	}

	inputTypes = g.expandTypeLookups(inputTypes)
	outputTypes = g.expandTypeLookups(outputTypes)

	inputMapping, outputMapping := solveInputOutputNameMapping(inputTypes, outputTypes)

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
				funcParams, _, _ := g.schemaForFunctionParameters(function, inputMapping)
				sb.WriteString(funcParams)
				sb.WriteString(")")
			}

			sb.WriteString(": ")
			schemaRef, _ := g.schemaRefForType(function.baseReturnType, outputMapping)

			sb.WriteString(schemaRef)
			sb.WriteString("\n")
		}
		sb.WriteString("}\n\n")
	}

	inputSchema, iEnumTypes, err := g.schemaForTypes(TypeInput, inputMapping, inputTypes...)
	if err != nil {
		return "", err
	}
	enumTypes = append(enumTypes, iEnumTypes...)
	sb.WriteString(inputSchema)

	outputSchema, oEnumTypes, err := g.schemaForTypes(TypeOutput, outputMapping, outputTypes...)
	if err != nil {
		return "", err
	}
	enumTypes = append(enumTypes, oEnumTypes...)

	sb.WriteString(outputSchema)

	enumSchema, err := g.schemaForEnumTypes(enumTypes...)
	if err != nil {
		return "", err
	}
	sb.WriteString(enumSchema)

	return sb.String(), nil
}

type typeNameMapping map[*typeLookup]string

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
	expandedTypes := []*typeLookup{}
	for tl := range expandedTypeMap {
		expandedTypes = append(expandedTypes, tl)
	}

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

func (g *Graphy) schemaForFunctionParameters(f *graphFunction, mapping typeNameMapping) (string, []*typeLookup, []*typeLookup) {
	sb := strings.Builder{}

	mappings := []functionNameMapping{}
	for _, param := range f.nameMapping {
		mappings = append(mappings, param)
	}
	// Sort by index
	sort.Slice(mappings, func(i, j int) bool {
		return mappings[i].paramIndex < mappings[j].paramIndex
	})

	var paramLookups []*typeLookup

	for i, param := range mappings {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(param.name)
		sb.WriteString(": ")
		paramTl := g.typeLookup(param.paramType)
		schemaRef, _ := g.schemaRefForType(paramTl, mapping)
		sb.WriteString(schemaRef)
		paramLookups = append(paramLookups, paramTl)
	}

	refLookups := []*typeLookup{
		f.baseReturnType,
	}

	return sb.String(), refLookups, paramLookups
}
