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
			sb.WriteString("(")

			funcParams, fOuput, fInput, err := g.schemaForFunctionParameters(function)
			if err != nil {
				return "", err
			}
			sb.WriteString(funcParams)
			sb.WriteString("): ")
			schemaRef, _ := g.schemaRefForType(function.baseReturnType)
			outputTypes = append(outputTypes, function.baseReturnType)
			inputTypes = append(inputTypes, fInput...)

			sb.WriteString(schemaRef)
			sb.WriteString("\n")

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
		sb.WriteString("}\n\n")
	}

	inputSchema, iEnumTypes, err := g.schemaForTypes(TypeInput, inputTypes...)
	if err != nil {
		return "", err
	}
	enumTypes = append(enumTypes, iEnumTypes...)
	sb.WriteString(inputSchema)

	outputSchema, oEnumTypes, err := g.schemaForTypes(TypeOutput, outputTypes...)
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

func (g *Graphy) schemaForFunctionParameters(f *graphFunction) (string, []*typeLookup, []*typeLookup, error) {
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
		schemaRef, _ := g.schemaRefForType(paramTl)
		sb.WriteString(schemaRef)
		paramLookups = append(paramLookups, paramTl)
	}

	refLookups := []*typeLookup{
		f.baseReturnType,
	}

	return sb.String(), refLookups, paramLookups, nil
}
