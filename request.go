package quickgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

type RequestType int

const (
	Query RequestType = iota
	Mutation
)

type RequestStub struct {
	Graphy    *Graphy
	Mode      RequestType
	Commands  []Command
	Variables map[string]RequestVariable
}

type RequestVariable struct {
	Name string
	Type reflect.Type
}

type Request struct {
	Graphy    *Graphy
	Stub      RequestStub
	Variables map[string]reflect.Value
}

func (g *Graphy) NewRequestStub(request string) (*RequestStub, error) {
	parsedCall, err := ParseRequest(request)
	if err != nil {
		return nil, err
	}

	variableTypeMap, err2 := g.GatherRequestVariables(parsedCall)
	if err2 != nil {
		return nil, err2
	}

	rs := RequestStub{
		Graphy:    g,
		Commands:  parsedCall.Commands,
		Variables: variableTypeMap,
	}

	switch parsedCall.Mode {
	case "":
	case "query":
		rs.Mode = Query
	case "mutation":
		rs.Mode = Mutation
	default:
		return nil, fmt.Errorf("unknown mode %s", parsedCall.Mode)
	}

	return &rs, nil
}

func (g *Graphy) GatherRequestVariables(parsedCall Wrapper) (map[string]RequestVariable, error) {
	// TODO: Look at the parsed arguments, find their types, then later verify that
	// they are correct.

	// Find the commands in the request that use variables, extract the types
	// of the variables, and convert the variables to the correct type. Ensure that
	// there is consistency with the types in case two commands use the same variable.
	variableTypeMap := map[string]RequestVariable{}
	for _, command := range parsedCall.Commands {
		processor, ok := g.processors[command.Name]
		if !ok {
			return nil, fmt.Errorf("unknown command %s", command.Name)
		}

		if command.Parameters != nil {
			for _, parameter := range command.Parameters.Values {
				if parameter.Value.Variable != nil {
					varName := *parameter.Value.Variable
					// String the leading $ from the variable name.
					varName = varName[1:]
					paramTarget := processor.nameMapping[parameter.Name]
					targetType := paramTarget.paramType
					if existingVariable, found := variableTypeMap[varName]; found {
						if existingVariable.Type != targetType {
							return nil, fmt.Errorf("variable %s is used with different types", varName)
						}
					} else {
						variableTypeMap[varName] = RequestVariable{
							Name: varName,
							Type: targetType,
						}
					}
				}
			}
		}

		// TODO: Dive into the result filter and find variables there.
	}
	return variableTypeMap, nil
}

func (rs *RequestStub) NewRequest(variableJson string) (*Request, error) {
	rawVariables := map[string]json.RawMessage{}
	if variableJson != "" {
		err := json.Unmarshal([]byte(variableJson), &rawVariables)
		if err != nil {
			return nil, err
		}
	}

	// Now use the variable type map to convert the variables to the correct type.
	variables := map[string]reflect.Value{}
	for varName, variable := range rs.Variables {
		// Get the RawMessage for the variable. Create a new instance of the variable type using reflection.
		// Then unmarshal the variable from JSON.
		variableJson, found := rawVariables[varName]
		if !found {
			return nil, fmt.Errorf("variable %s not provided", varName)
		}
		variableValue := reflect.New(variable.Type)
		err := json.Unmarshal(variableJson, variableValue.Interface())
		if err != nil {
			return nil, err
		}
		variables[varName] = variableValue.Elem()
	}

	return &Request{
		Graphy:    rs.Graphy,
		Stub:      *rs,
		Variables: variables,
	}, nil
}

func (r *Request) Execute(ctx context.Context) (string, error) {
	// TODO: Deal with all commands.
	command := r.Stub.Commands[0]

	result := map[string]any{}
	data := map[string]any{}
	result["data"] = data

	// TODO: In query mode, we can run all these in parallel.

	// Find the processor
	if processor, ok := r.Graphy.processors[command.Name]; ok {
		// TODO: Variables
		r, err := processor.Call(ctx, r, command)
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
