package quickgraph

import (
	"context"
	"sort"
	"strings"
)

func (g *Graphy) SchemaDefinition(ctx context.Context) (string, error) {
	sb := strings.Builder{}

	procByMode := map[GraphFunctionMode][]*GraphFunction{}

	for _, function := range g.processors {
		byMode, ok := procByMode[function.mode]
		if !ok {
			byMode = []*GraphFunction{}
			procByMode[function.mode] = byMode
		}
		procByMode[function.mode] = append(byMode, &function)
	}

	outputTypes := []*TypeLookup{}
	enumTypes := []*TypeLookup{}

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

		for _, function := range functions {
			sb.WriteString("\t")
			sb.WriteString(function.name)
			sb.WriteString("(")

			funcParams, fOuput, err := g.schemaForFunctionParameters(function)
			if err != nil {
				return "", err
			}
			for _, outTypeLookup := range fOuput {
				if outTypeLookup.rootType != nil {
					if outTypeLookup.rootType.AssignableTo(stringEnumValuesType) {
						enumTypes = append(enumTypes, outTypeLookup)
					}
				} else {
					outputTypes = append(outputTypes, outTypeLookup)
				}
			}
			sb.WriteString(funcParams)

			sb.WriteString("): ")
			schemaRef, _ := g.schemaRefForType(function.baseReturnType)
			outputTypes = append(outputTypes, function.baseReturnType)
			sb.WriteString(schemaRef)
			sb.WriteString("\n")
		}
		sb.WriteString("}\n\n")
	}

	outputSchema, oEnumTypes, err := g.schemaForOutputTypes(outputTypes...)
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

func (g *Graphy) schemaForFunctionParameters(f *GraphFunction) (string, []*TypeLookup, error) {
	sb := strings.Builder{}

	mappings := []FunctionNameMapping{}
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
		schemaRef, _ := g.schemaRefForType(g.typeLookup(param.paramType))
		sb.WriteString(schemaRef)
	}

	ret := []*TypeLookup{
		f.baseReturnType,
	}

	return sb.String(), ret, nil
}
