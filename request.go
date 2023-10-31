package quickgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alecthomas/participle/v2/lexer"
	"reflect"
	"strings"
)

// RequestType is an enumeration of the types of requests. It can be a Query or a Mutation.
type RequestType int

const (
	RequestQuery RequestType = iota
	RequestMutation
)

// RequestStub represents a stub of a GraphQL-like request. It contains the Graphy instance,
// the mode of the request (Query or Mutation), the commands to execute, and the variables used in the request.
type RequestStub struct {
	graphy    *Graphy
	mode      RequestType
	commands  []command
	variables map[string]*requestVariable
	fragments map[string]fragment
}

// requestVariable represents a variable in a GraphQL-like request. It contains the variable name and its type.
type requestVariable struct {
	Name    string
	Type    reflect.Type
	Default *genericValue
}

// GraphRequestCache represents an interface for caching request stubs
// associated with graph requests. Implementations of this interface
// provide mechanisms to store and retrieve `RequestStub`s, allowing
// for optimizations and reduced processing times in graph operations.
// Note that the `RequestStub` is an internal representation of a
// graph request, and is not intended to be used directly by consumers.
// It is not serializable to JSON and needs to be kept in memory.
type GraphRequestCache interface {
	// GetRequestStub returns the request stub for a request. It should return nil if the request
	// is not cached. The error can either be the cached error or an error indicating a cache error.
	// In case the request is not cached, the returned *RequestStub should be nil.
	GetRequestStub(ctx context.Context, request string) (*RequestStub, error)

	// SetRequestStub sets the request stub for a request.
	SetRequestStub(ctx context.Context, request string, stub *RequestStub, err error)
}

// request represents a complete GraphQL-like request. It contains the Graphy instance, the request stub,
// and the actual variables used in the request.
type request struct {
	graphy    *Graphy
	stub      RequestStub
	variables map[string]reflect.Value
}

// newRequestStub creates a new request stub from a string representation of a GraphQL request.
// It parses the request, gathers and validates the variables used in the request, and determines
// the request type (Query or Mutation).
func (g *Graphy) newRequestStub(request string) (*RequestStub, error) {
	parsedCall, err := parseRequest(request)
	if err != nil {
		return nil, err
	}

	var mode RequestType
	switch strings.ToLower(parsedCall.Mode) {
	case "":
	case "query":
		mode = RequestQuery
	case "mutation":
		mode = RequestMutation
	default:
		return nil, NewGraphError(fmt.Sprintf("unknown/unsupported call mode %s", parsedCall.Mode), parsedCall.Pos)
	}

	// Validate that we have processors for all the commands.
	var missingCommands []command
	for _, command := range parsedCall.Commands {
		if processor, ok := g.processors[command.Name]; ok {
			if mode == RequestQuery && processor.mode == ModeMutation {
				return nil, NewGraphError(fmt.Sprintf("mutation %s used in query", command.Name), command.Pos)
			}
		} else {
			missingCommands = append(missingCommands, command)
		}
	}
	if len(missingCommands) > 0 {
		// Make a string slice of the command names.
		missingCommandNames := make([]string, len(missingCommands))
		for i, command := range missingCommands {
			missingCommandNames[i] = command.Name
		}
		return nil, UnknownCommandError{
			GraphError: GraphError{
				Message: "unknown command(s) in request: " + strings.Join(missingCommandNames, ", "),
				Locations: []ErrorLocation{
					lexerPositionError(missingCommands[0].Pos),
				},
			},
			Commands: missingCommandNames,
		}
	}

	fragments := map[string]fragment{}
	for _, fragment := range parsedCall.Fragments {
		fragments[fragment.Name] = fragment
	}

	// TODO: Use the fragments in the variable gathering.
	variableTypeMap, err := g.gatherRequestVariables(parsedCall, fragments)
	if err != nil {
		return nil, err
	}

	rs := RequestStub{
		graphy:    g,
		commands:  parsedCall.Commands,
		variables: variableTypeMap,
		fragments: fragments,
		mode:      mode,
	}

	return &rs, nil
}

// gatherRequestVariables gathers and validates the variables used in a GraphQL request.
// It ensures that the variables used across different commands are of the same type.
func (g *Graphy) gatherRequestVariables(parsedCall wrapper, fragments map[string]fragment) (map[string]*requestVariable, error) {
	// TODO: Look at the parsed arguments, find their types, then later verify that
	//  they are correct.

	// Find the commands in the request that use variables, extract the types
	// of the variables, and convert the variables to the correct type. Ensure that
	// there is consistency with the types in case two commands use the same variable.
	variableTypeMap := map[string]*requestVariable{}
	for _, command := range parsedCall.Commands {
		graphFunc, ok := g.processors[command.Name]
		if !ok {
			// This should have been caught earlier.
			return nil, NewGraphError(fmt.Sprintf("unknown command %s", command.Name), command.Pos)
		}
		anonArgs := false
		argIndex := 0
		if graphFunc.paramType == AnonymousParamsInline {
			anonArgs = true
		}
		if command.Parameters != nil {
			for _, parameter := range command.Parameters.Values {
				if parameter.Value.Variable != nil {
					varName := *parameter.Value.Variable
					var paramTarget functionNameMapping
					if anonArgs {
						paramTarget = graphFunc.indexMapping[argIndex]
						argIndex++
					} else {
						paramTarget = graphFunc.nameMapping[parameter.Name]
					}
					targetType := paramTarget.paramType
					if targetType == nil {
						panic(fmt.Sprintf("unknown parameter %s", parameter.Name))
					}

					err := g.addTypedInputVariable(varName, variableTypeMap, targetType)
					if err != nil {
						return nil, AugmentGraphError(err, fmt.Sprintf("error adding variable %s", varName), parameter.Pos)
					}
				}
			}
		}

		// Depth-first search into the result filter.
		typeLookup := graphFunc.baseReturnType

		err := g.addAndValidateResultVariables(typeLookup, command.ResultFilter, variableTypeMap, fragments)
		if err != nil {
			return nil, AugmentGraphError(err, fmt.Sprintf("error validating result filter for %s", command.Name), command.ResultFilter.Pos, command.Name)
		}
	}

	if parsedCall.OperationDef != nil {
		// Ensure that all the variables used in the operation definition are present.
		opVars := map[string]variableDef{}
		for _, variable := range parsedCall.OperationDef.Variables {
			// Trim off the leading $.
			name := variable.Name[1:]
			variable := variable
			opVars[name] = variable
		}

		for key, variable := range variableTypeMap {
			if reqVar, found := opVars[variable.Name]; found {
				// TODO: Validate the variable type.
				variableTypeMap[key].Default = reqVar.Value
				variable.Default = reqVar.Value
			} else {
				return nil, fmt.Errorf("variable %s is not defined in the operation", variable.Name)
			}
		}
	}

	return variableTypeMap, nil
}

func (g *Graphy) addTypedInputVariable(varName string, variableTypeMap map[string]*requestVariable, targetType reflect.Type) error {
	// Strip the leading $ from the variable name.
	varName = varName[1:]
	if existingVariable, found := variableTypeMap[varName]; found {
		if existingVariable.Type != targetType {
			return fmt.Errorf("variable %s is used with different types: existing type: %v, new type: %v", varName, existingVariable.Type, targetType)
		}
	} else {
		variableTypeMap[varName] = &requestVariable{
			Name: varName,
			Type: targetType,
		}
	}
	return nil
}

func (g *Graphy) addAndValidateResultVariables(typ *typeLookup, filter *resultFilter, variableTypeMap map[string]*requestVariable, fragments map[string]fragment) error {

	if filter == nil {
		return nil
	}

	for _, field := range filter.Fields {
		if len(typ.fields) == 0 {
			// This is a bit silly, but not an error.
			return nil
		}
		if field.Name == "__typename" {
			// This is a virtual field that is always present.
			continue
		}
		if pf, ok := typ.GetField(field.Name); ok {
			var commandField *resultField
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

			var childType *typeLookup
			if pf.fieldType == FieldTypeField {
				childType = g.typeLookup(pf.resultType)
				// Recurse
			} else if pf.fieldType == FieldTypeGraphFunction {
				gf := pf.graphFunction
				childType = gf.baseReturnType

				err := g.validateGraphFunctionParameters(commandField, gf, variableTypeMap)
				if err != nil {
					return AugmentGraphError(err, fmt.Sprintf("error validating parameters for %s", gf.name), field.Pos, gf.name)
				}
			}

			if childType != nil {
				// Recurse
				err := g.addAndValidateResultVariables(childType, field.SubParts, variableTypeMap, nil)
				if err != nil {
					return AugmentGraphError(err, fmt.Sprintf("error validating field for %s", field.Name), field.SubParts.Pos, field.Name)
				}
			}
		} else {
			return NewGraphError(fmt.Sprintf("unknown field %s", field.Name), field.Pos)
		}
	}

	// Recurse into the fragments.
	for _, fragment := range filter.Fragments {
		var fragmentDef *fragmentDef
		if fragment.Inline != nil {
			fragmentDef = fragment.Inline
		} else if fragment.FragmentRef != nil {
			fragmentDef = fragments[*fragment.FragmentRef].Definition
		} else {
			return fmt.Errorf("unknown fragment type")
		}
		if found, subTyp := typ.ImplementsInterface(fragmentDef.TypeName); found {
			err := g.addAndValidateResultVariables(subTyp, fragmentDef.Filter, variableTypeMap, fragments)
			if err != nil {
				return AugmentGraphError(err, fmt.Sprintf("error validating fragment %s", fragmentDef.TypeName), fragmentDef.Filter.Pos, fragmentDef.TypeName)
			}
		}
	}

	return nil
}

func (g *Graphy) validateGraphFunctionParameters(commandField *resultField, gf *graphFunction, variableTypeMap map[string]*requestVariable) error {
	// Validate the parameters.
	switch gf.paramType {
	case AnonymousParamsInline:
		return g.validateAnonymousFunctionParams(commandField, gf, variableTypeMap)
	case NamedParamsStruct:
		return g.validateNamedFunctionParams(commandField, gf, variableTypeMap)
	default:
		return fmt.Errorf("unknown function paramType %d", gf.paramType)
	}
}

func (g *Graphy) validateAnonymousFunctionParams(commandField *resultField, gf *graphFunction, variableTypeMap map[string]*requestVariable) error {
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
			return NewGraphError("missing parameters", commandField.Pos)
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
		//  special consideration that is needed is that pointers to objects are
		//  allowed -- and we have to ensure that objects that are cached are not
		//  changed between calls. Short-term, we can just not cache value types.
	}
	return nil
}

func (g *Graphy) validateNamedFunctionParams(commandField *resultField, gf *graphFunction, variableTypeMap map[string]*requestVariable) error {
	neededField := map[string]bool{}
	for _, param := range gf.nameMapping {
		neededField[param.name] = true
	}

	if commandField.Params != nil {
		for _, cfp := range commandField.Params.Values {
			targetType := gf.nameMapping[cfp.Name].paramType

			// We have the parameter, so remove it from the needed list.
			delete(neededField, cfp.Name)

			if cfp.Value.Variable != nil {
				varName := *cfp.Value.Variable
				// Strip the leading $ from the variable name.
				varName = varName[1:]

				err := g.validateFunctionVarParam(variableTypeMap, varName, targetType)
				if err != nil {
					return AugmentGraphError(err, fmt.Sprintf("error validating variable %s", varName), cfp.Pos)
				}
			}
			// Todo: Consider parsing, validating, and caching the value for value types. The
			//  special consideration that is needed is that pointers to objects are
			//  allowed -- and we have to ensure that objects that are cached are not
			//  changed between calls. Short-term, we can just not cache value types.
		}
	}

	// Ensure that all parameters are present.
	for name, val := range neededField {
		if val == true {
			return fmt.Errorf("missing parameter %s", name)
		}
	}

	return nil
}

func (g *Graphy) validateFunctionVarParam(variableTypeMap map[string]*requestVariable, varName string, targetType reflect.Type) error {
	if existingVariable, found := variableTypeMap[varName]; found {
		if existingVariable.Type != targetType {
			return fmt.Errorf("variable %s is used with different types: existing type: %v, new type: %v", varName, existingVariable.Type, targetType)
		}
	} else {
		variableTypeMap[varName] = &requestVariable{
			Name: varName,
			Type: targetType,
		}
	}
	return nil
}

// newRequest creates a new request from a request stub and a JSON string representing the variables used in the request.
// It unmarshals the variables and assigns them to the corresponding variables in the request.
func (rs *RequestStub) newRequest(variableJson string) (*request, error) {
	rawVariables := map[string]json.RawMessage{}
	if variableJson != "" {
		err := json.Unmarshal([]byte(variableJson), &rawVariables)
		if err != nil {
			return nil, transformJsonError(variableJson, err)
		}
	}

	// Now use the variable type map to convert the variables to the correct type.
	variables := map[string]reflect.Value{}
	for varName, variable := range rs.variables {
		// Get the RawMessage for the variable. Create a new instance of the variable type using reflection.
		// Then unmarshal the variable from JSON.
		variableValue := reflect.New(variable.Type)
		if variableJson, found := rawVariables[varName]; found {
			err := json.Unmarshal(variableJson, variableValue.Interface())
			if err != nil {
				return nil, AugmentGraphError(err, fmt.Sprintf("error parsing variable %s into type %s", varName, variable.Type.Name()), lexer.Position{})
			}
			variables[varName] = variableValue.Elem()
		} else if variable.Default != nil {
			err := parseInputIntoValue(nil, *variable.Default, variableValue.Elem())
			if err != nil {
				return nil, AugmentGraphError(err, fmt.Sprintf("error parsing default variable %s into type %s", varName, variable.Type.Name()), lexer.Position{})
			}
			variables[varName] = variableValue.Elem()
		} else {
			return nil, NewGraphError(fmt.Sprintf("variable %s not provided", varName), lexer.Position{})
		}
	}

	return &request{
		graphy:    rs.graphy,
		stub:      *rs,
		variables: variables,
	}, nil
}

type commandResult struct {
	name string
	obj  any
	err  error
}

// execute executes a GraphQL request. It looks up the appropriate processor for each command and invokes it.
// It returns the result of the request as a JSON string.
func (r *request) execute(ctx context.Context) (string, error) {
	result := map[string]any{}
	data := map[string]any{}
	var errColl []error
	result["data"] = data
	var retErr error

	var cmdResults []commandResult
	var parallel bool
	if r.stub.mode == RequestMutation {
		parallel = false
	} else {
		parallel = true
	}

	if parallel {
		resultChan := make(chan commandResult)
		// execute the commands in parallel.
		for _, cmd := range r.stub.commands {
			go func(cmd command) {
				resultChan <- r.executeCommand(ctx, cmd)
			}(cmd)
		}
		// Gather the results from the channel and put them in the cmdResults
		// slice.
		for len(cmdResults) < len(r.stub.commands) {
			select {
			case <-ctx.Done():
				cmdResults = append(cmdResults, commandResult{
					err: AugmentGraphError(ctx.Err(), "context timed out", lexer.Position{}),
				})
				break
			case cmdResult := <-resultChan:
				cmdResults = append(cmdResults, cmdResult)
			}
		}
	} else {
		for _, command := range r.stub.commands {
			ctxErr := ctx.Err()
			if ctxErr != nil {
				cmdResults = append(cmdResults, commandResult{
					err: AugmentGraphError(ctx.Err(), "context timed out", lexer.Position{}),
				})
				break
			}
			cmdResults = append(cmdResults, r.executeCommand(ctx, command))
		}
	}

	for _, cmdResult := range cmdResults {
		if cmdResult.err != nil {
			errColl = append(errColl, cmdResult.err)
			retErr = cmdResult.err
		}

		if cmdResult.name != "" {
			data[cmdResult.name] = cmdResult.obj
		}
	}

	if len(errColl) > 0 {
		result["errors"] = errColl
	}

	// Serialize the result to JSON.
	marshal, err := json.Marshal(result)
	if err != nil {
		// There should be no way for this to happen since we're using basic objects.
		return "", err
	}
	return string(marshal), retErr
}

func (r *request) executeCommand(ctx context.Context, command command) commandResult {
	processor, ok := r.graphy.processors[command.Name]
	if !ok {
		// This shouldn't happen since we validate the commands when we create the request stub.
		return commandResult{
			err: NewGraphError(fmt.Sprintf("unknown command %s", command.Name), command.Pos),
		}
	}

	obj, err := processor.Call(ctx, r, command.Parameters, reflect.Value{})
	if err != nil {
		return commandResult{
			err: AugmentGraphError(err, fmt.Sprintf("error calling %s", command.Name), command.Pos, command.Name),
		}
	}

	res, err := processor.GenerateResult(ctx, r, obj, command.ResultFilter)
	if err != nil {
		var pos lexer.Position
		if command.ResultFilter != nil {
			pos = command.ResultFilter.Pos
		} else {
			pos = command.Pos
		}
		return commandResult{
			err: AugmentGraphError(err, fmt.Sprintf("error generating result for %s", command.Name), pos, command.Name),
		}
	}

	name := command.Name
	if command.Alias != nil {
		name = *command.Alias
	}

	return commandResult{
		name: name,
		obj:  res,
	}
}
