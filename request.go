package quickgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

// RequestType is an enumeration of the types of requests. It can be a Query or a Mutation.
type RequestType int

const (
	Query RequestType = iota
	Mutation
)

// RequestStub represents a stub of a GraphQL-like request. It contains the Graphy instance,
// the mode of the request (Query or Mutation), the commands to execute, and the variables used in the request.
type RequestStub struct {
	Graphy    *Graphy
	Mode      RequestType
	Commands  []Command
	Variables map[string]RequestVariable
	Fragments map[string]Fragment
}

// RequestVariable represents a variable in a GraphQL-like request. It contains the variable name and its type.
type RequestVariable struct {
	Name string
	Type reflect.Type
}

// Request represents a complete GraphQL-like request. It contains the Graphy instance, the request stub,
// and the actual variables used in the request.
type Request struct {
	Graphy    *Graphy
	Stub      RequestStub
	Variables map[string]reflect.Value
}

// NewRequestStub creates a new request stub from a string representation of a GraphQL request.
// It parses the request, gathers and validates the variables used in the request, and determines
// the request type (Query or Mutation).
func (g *Graphy) NewRequestStub(request string) (*RequestStub, error) {
	parsedCall, err := ParseRequest(request)
	if err != nil {
		return nil, err
	}

	fragments := map[string]Fragment{}
	for _, fragment := range parsedCall.Fragments {
		// TODO: Validate the fragments.
		fragments[fragment.Name] = fragment
	}

	// TODO: Use the fragments in the variable gathering.
	variableTypeMap, err := g.GatherRequestVariables(parsedCall)
	if err != nil {
		return nil, err
	}

	rs := RequestStub{
		Graphy:    g,
		Commands:  parsedCall.Commands,
		Variables: variableTypeMap,
		Fragments: fragments,
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

// GatherRequestVariables gathers and validates the variables used in a GraphQL request.
// It ensures that the variables used across different commands are of the same type.
func (g *Graphy) GatherRequestVariables(parsedCall Wrapper) (map[string]RequestVariable, error) {
	// TODO: Look at the parsed arguments, find their types, then later verify that
	// they are correct.

	// Find the commands in the request that use variables, extract the types
	// of the variables, and convert the variables to the correct type. Ensure that
	// there is consistency with the types in case two commands use the same variable.
	variableTypeMap := map[string]RequestVariable{}
	for _, command := range parsedCall.Commands {
		graphFunc, ok := g.processors[command.Name]
		if !ok {
			return nil, fmt.Errorf("unknown command %s", command.Name)
		}

		if command.Parameters != nil {
			for _, parameter := range command.Parameters.Values {
				if parameter.Value.Variable != nil {
					varName := *parameter.Value.Variable
					// String the leading $ from the variable name.
					varName = varName[1:]
					paramTarget := graphFunc.nameMapping[parameter.Name]
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

		// Depth first search into the result filter.
		typeLookup := g.TypeLookup(graphFunc.returnType)
		err := g.addTypeVariables(typeLookup, command.ResultFilter, variableTypeMap)
		if err != nil {
			return nil, err
		}
	}
	return variableTypeMap, nil
}

func (g *Graphy) addTypeVariables(typ *TypeLookup, filter *ResultFilter, variableTypeMap map[string]RequestVariable) error {

	if filter == nil {
		return nil
	}

	for _, field := range filter.Fields {
		if typ == nil {
			return fmt.Errorf("type is nil")
		}
		if typ.fields == nil {
			return fmt.Errorf("type has no fields")
		}
		if field.Name == "__typename" {
			// This is a virtual field that is always present.
			continue
		}
		if pf, ok := typ.GetField(field.Name); ok {
			var commandField *ResultField
			for _, resultField := range filter.Fields {
				if resultField.Name == field.Name {
					commandField = &resultField
					break
				}
			}
			if commandField == nil {
				// Todo: Warning?
				continue
			}
			// Todo: handle union types
			var childType *TypeLookup
			if pf.fieldType == FieldTypeField {
				childType = g.TypeLookup(pf.resultType)
				// Recurse
			} else if pf.fieldType == FieldTypeGraphFunction {
				gf := pf.graphFunction
				childType = g.TypeLookup(gf.returnType)

				err := g.validateGraphFunctionParameters(commandField, gf, variableTypeMap)
				if err != nil {
					return err
				}
			}

			if childType != nil {
				// Recurse
				err := g.addTypeVariables(childType, field.SubParts, variableTypeMap)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("unknown field %s", field.Name)
		}
	}
	return nil
}

func (g *Graphy) validateGraphFunctionParameters(commandField *ResultField, gf *GraphFunction, variableTypeMap map[string]RequestVariable) error {
	// Validate the parameters.
	switch gf.mode {
	case AnonymousParamsInline:
		return g.validateAnonymousFunctionParams(commandField, gf, variableTypeMap)
	case NamedParamsStruct:
		return g.validateNamedFunctionParams(commandField, gf, variableTypeMap)
	default:
		return fmt.Errorf("unknown function mode %d", gf.mode)
	}
}

func (g *Graphy) validateAnonymousFunctionParams(commandField *ResultField, gf *GraphFunction, variableTypeMap map[string]RequestVariable) error {
	// Ensure that the number of parameters is correct.
	// TODO: If the parameters are all pointers, then they are optional.

	if commandField.Params == nil && gf.function.Type().NumIn() != 1 {
		// If all of the parameters are pointers, then they are optional and we're OK.
		allOptional := true
		for i := 1; i < gf.function.Type().NumIn(); i++ {
			if gf.function.Type().In(i).Kind() != reflect.Ptr {
				allOptional = false
				break
			}
		}
		if !allOptional {
			return fmt.Errorf("missing parameters")
		}
		return nil
	}
	paramCount := 0
	if commandField.Params != nil {
		paramCount = len(commandField.Params.Values)
	}
	if paramCount != gf.function.Type().NumIn()-1 {
		return fmt.Errorf("wrong number of parameters")
	}
	paramIndex := 1 // Skip the first parameter which is the receiver.
	if commandField.Params == nil {
		return nil
	}
	for _, cfp := range commandField.Params.Values {
		targetType := gf.function.Type().In(paramIndex)
		paramIndex++

		// Ensure that the parameter is the correct type.
		if cfp.Value.Variable != nil {
			varName := *cfp.Value.Variable
			// Strip the leading $ from the variable name.
			varName = varName[1:]

			err := g.validateFunctionVarParam(variableTypeMap, varName, targetType)
			if err != nil {
				return err
			}
		}
		// Todo: Consider parsing, validating, and caching the value for value types. The
		// special consideration that is needed is that pointers to objects are
		// allowed -- and we have to ensure that objects that are cached are not
		// changed between calls. Short-term, we can just not cache value types.
	}
	return nil
}

func (g *Graphy) validateNamedFunctionParams(commandField *ResultField, gf *GraphFunction, variableTypeMap map[string]RequestVariable) error {
	neededField := map[string]bool{}
	for _, param := range gf.nameMapping {
		neededField[param.name] = true
	}

	for _, cfp := range commandField.Params.Values {
		targetType := gf.nameMapping[cfp.Name].paramType

		if cfp.Value.Variable != nil {
			varName := *cfp.Value.Variable
			// Strip the leading $ from the variable name.
			varName = varName[1:]

			err := g.validateFunctionVarParam(variableTypeMap, varName, targetType)
			if err != nil {
				return err
			}
		}
		// Todo: Consider parsing, validating, and caching the value for value types. The
		// special consideration that is needed is that pointers to objects are
		// allowed -- and we have to ensure that objects that are cached are not
		// changed between calls. Short-term, we can just not cache value types.
	}

	// Ensure that all parameters are present.
	for name, val := range neededField {
		if val == true {
			return fmt.Errorf("missing parameter %s", name)
		}
	}

	return nil
}

func (g *Graphy) validateFunctionVarParam(variableTypeMap map[string]RequestVariable, varName string, targetType reflect.Type) error {
	if existingVariable, found := variableTypeMap[varName]; found {
		if existingVariable.Type != targetType {
			return fmt.Errorf("variable %s is used with different types", varName)
		}
	} else {
		variableTypeMap[varName] = RequestVariable{
			Name: varName,
			Type: targetType,
		}
	}
	return nil
}

// NewRequest creates a new request from a request stub and a JSON string representing the variables used in the request.
// It unmarshals the variables and assigns them to the corresponding variables in the request.
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
			return nil, fmt.Errorf("variable %s into type %s: %s", varName, variable.Type.Name(), err)
		}
		variables[varName] = variableValue.Elem()
	}

	return &Request{
		Graphy:    rs.Graphy,
		Stub:      *rs,
		Variables: variables,
	}, nil
}

// Execute executes a GraphQL request. It looks up the appropriate processor for each command and invokes it.
// It returns the result of the request as a JSON string.
func (r *Request) Execute(ctx context.Context) (string, error) {
	result := map[string]any{}
	data := map[string]any{}
	result["data"] = data

	for _, command := range r.Stub.Commands {
		// TODO: In query mode, we can run all these in parallel.
		// Find the processor
		if processor, ok := r.Graphy.processors[command.Name]; ok {
			obj, err := processor.Call(ctx, r, command.Parameters, reflect.Value{})
			if err != nil {
				return "", err
			}
			res, err := processor.GenerateResult(ctx, r, obj, command.ResultFilter)
			if err != nil {
				return "", err
			}
			name := command.Name
			if command.Alias != nil {
				name = *command.Alias
			}
			data[name] = res
		} else {
			// TODO: Make this better
			return "", fmt.Errorf("unknown command %s", command.Name)
		}
	}

	// Serialize the result to JSON.
	marshal, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(marshal), nil
}
