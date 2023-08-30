package quickgraph

import (
	"context"
	"reflect"
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
	enumTypes := []reflect.Type{}

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
			for _, outType := range fOuput {
				if outType.AssignableTo(stringEnumValuesType) {
					enumTypes = append(enumTypes, outType)
				} else {
					outTypeLookup := g.typeLookup(outType)
					outputTypes = append(outputTypes, outTypeLookup)
				}
			}
			sb.WriteString(funcParams)

			sb.WriteString("): ")
			schemaRef, _ := g.schemaRefForType(function.rawReturnType)
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

func (g *Graphy) schemaForFunctionParameters(f *GraphFunction) (string, []reflect.Type, error) {
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
		schemaRef, _ := g.schemaRefForType(param.paramType)
		sb.WriteString(schemaRef)
	}

	ret := []reflect.Type{
		f.rawReturnType,
	}

	return sb.String(), ret, nil
}
