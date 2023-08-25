package quickgraph

import (
	"context"
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
			//for i, param := range function. {
			//
			//}
			sb.WriteString("): ")
			schemaRef, _ := g.schemaRefForType(function.returnType.typ)
			outputTypes = append(outputTypes, function.returnType)
			sb.WriteString(schemaRef)
			sb.WriteString("\n")
		}
		sb.WriteString("}\n\n")
	}

	types, err := g.schemaForOutputTypes(outputTypes...)
	if err != nil {
		return "", err
	}

	sb.WriteString(types)

	return sb.String(), nil
}
