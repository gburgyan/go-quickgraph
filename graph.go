package quickgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

type Graphy struct {
	processors map[string]GraphFunction
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()

func (g *Graphy) RegisterProcessorWithParamNames(ctx context.Context, name string, mutatorFunc any, names ...string) {
	gf := NewGraphFunction(name, mutatorFunc, names...)
	if g.processors == nil {
		g.processors = map[string]GraphFunction{}
	}
	g.processors[name] = gf
}

func (g *Graphy) ProcessRequest(ctx context.Context, request string, variableJson string) (string, error) {
	parsedCall, err := ParseRequest(request)
	if err != nil {
		return "", err
	}

	// TODO: Variables
	rawVariables := map[string]json.RawMessage{}
	if variableJson != "" {
		err = json.Unmarshal([]byte(variableJson), &rawVariables)
		if err != nil {
			return "", err
		}
	}

	// Find the commands in the request that use variables, extract the types
	// of the variables, and convert the variables to the correct type. Ensure that
	// there is consistency with the types in case two commands use the same variable.
	variableTypeMap := map[string]reflect.Type{}
	for _, command := range parsedCall.Command {
		processor, ok := g.processors[command.Name]
		if !ok {
			return "", fmt.Errorf("unknown command %s", command.Name)
		}

		if command.Parameters != nil {
			for _, parameter := range command.Parameters.Values {
				if parameter.Value.Variable != nil {
					varName := *parameter.Value.Variable
					paramTarget := processor.nameMapping[parameter.Name]
					targetType := paramTarget.paramType
					if existingType, found := variableTypeMap[varName]; found {
						if existingType != targetType {
							return "", fmt.Errorf("variable %s is used with different types", varName)
						}
					} else {
						variableTypeMap[varName] = targetType
					}
				}
			}
		}
	}
	// Now use the variable type map to convert the variables to the correct type.
	variables := map[string]any{}
	for varName, varType := range variableTypeMap {
		// Use reflection to create a new instance of the variable type.
		variable := reflect.New(varType).Interface()
		// Deserialize the variable from JSON.
		variableJson, found := rawVariables[varName]
		if !found {
			return "", fmt.Errorf("variable %s not provided", varName)
		}
		// Unmarshal the variable from JSON.
		err = json.Unmarshal(variableJson, variable)
		if err != nil {
			return "", err
		}
		variables[varName] = variable
	}

	command := parsedCall.Command[0]

	result := map[string]any{}
	data := map[string]any{}
	result["data"] = data

	// TODO: In query mode, we can run all these in parallel.

	// Find the processor
	if processor, ok := g.processors[command.Name]; ok {
		// TODO: Variables
		r, err := processor.Call(ctx, command)
		if err != nil {
			return "", err
		}
		name := command.Name
		if command.Alias != nil {
			name = *command.Alias
		}
		data[name] = r
	}

	// Serialize the result to JSON.
	marshal, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(marshal), nil
}
