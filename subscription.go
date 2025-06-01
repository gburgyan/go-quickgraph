package quickgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
)

// SubscriptionMessage represents a single message in a subscription stream
type SubscriptionMessage struct {
	Data   any     `json:"data,omitempty"`
	Errors []error `json:"errors,omitempty"`
}

// SubscriptionResult represents the result of a subscription execution
type SubscriptionResult struct {
	// Channel that emits subscription messages
	Messages <-chan SubscriptionMessage
	// Cleanup function to be called when subscription is terminated
	Cleanup func()
}

// executeSubscription handles the execution of a subscription request
func (r *request) executeSubscription(ctx context.Context) (*SubscriptionResult, error) {
	if r.stub.mode != RequestSubscription {
		return nil, fmt.Errorf("executeSubscription called on non-subscription request")
	}

	if len(r.stub.commands) != 1 {
		return nil, fmt.Errorf("subscription must have exactly one root field")
	}

	command := r.stub.commands[0]
	processor, ok := r.graphy.processors[command.Name]
	if !ok {
		return nil, NewGraphError(fmt.Sprintf("unknown subscription %s", command.Name), command.Pos)
	}

	if !processor.isSubscription {
		return nil, NewGraphError(fmt.Sprintf("%s is not a subscription", command.Name), command.Pos)
	}

	// Call the subscription function to get the channel
	channelValue, err := processor.Call(ctx, r, command.Parameters, reflect.Value{})
	if err != nil {
		return nil, AugmentGraphError(err, fmt.Sprintf("error calling subscription %s", command.Name), command.Pos, command.Name)
	}

	// Verify we got a channel back
	if channelValue.Kind() != reflect.Chan {
		return nil, NewGraphError(fmt.Sprintf("subscription %s did not return a channel", command.Name), command.Pos)
	}

	// Create output channel for messages
	bufferSize := 0
	if r.graphy.MemoryLimits != nil {
		bufferSize = r.graphy.MemoryLimits.SubscriptionBufferSize
		if bufferSize < 0 {
			bufferSize = 0 // Treat negative values as unbuffered
		}
	}
	outChan := make(chan SubscriptionMessage, bufferSize)

	// Start goroutine to handle channel messages
	go func() {
		defer close(outChan)

		// Create select cases for both context cancellation and channel reception
		selectCases := []reflect.SelectCase{
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ctx.Done())},
			{Dir: reflect.SelectRecv, Chan: channelValue},
		}

		// Listen for values from the subscription channel
		for {
			chosen, value, ok := reflect.Select(selectCases)
			switch chosen {
			case 0: // ctx.Done()
				// Context cancelled, stop processing
				return
			case 1: // channelValue
				if !ok {
					// Channel closed, subscription complete
					return
				}

				// Generate the result for this value
				var name string
				if command.Alias != nil {
					name = *command.Alias
				} else {
					name = command.Name
				}

				// Process the value through the result filter
				result, err := processor.GenerateResult(ctx, r, value, command.ResultFilter)
				if err != nil {
					// Send error message
					outChan <- SubscriptionMessage{
						Errors: []error{AugmentGraphError(err, fmt.Sprintf("error generating result for subscription %s", command.Name), command.Pos, command.Name)},
					}
					continue
				}

				// Send data message
				data := map[string]any{name: result}
				outChan <- SubscriptionMessage{
					Data: data,
				}
			}
		}
	}()

	return &SubscriptionResult{
		Messages: outChan,
		Cleanup: func() {
			// Currently no cleanup needed, but this could be extended
			// to handle unsubscribe logic, resource cleanup, etc.
		},
	}, nil
}

// ProcessSubscription is a higher-level method that executes a subscription and returns a channel of JSON messages
func (g *Graphy) ProcessSubscription(ctx context.Context, request string, variableJson string) (<-chan string, error) {
	g.structureLock.RLock()
	defer g.structureLock.RUnlock()

	rs, err := g.getRequestStub(ctx, request)
	if err != nil {
		return nil, err
	}

	if rs.mode != RequestSubscription {
		return nil, fmt.Errorf("request is not a subscription")
	}

	newRequest, err := rs.newRequest(ctx, variableJson)
	if err != nil {
		return nil, err
	}

	subResult, err := newRequest.executeSubscription(ctx)
	if err != nil {
		return nil, err
	}

	// Create channel for JSON messages
	bufferSize := 0
	if g.MemoryLimits != nil {
		bufferSize = g.MemoryLimits.SubscriptionBufferSize
		if bufferSize < 0 {
			bufferSize = 0 // Treat negative values as unbuffered
		}
	}
	jsonChan := make(chan string, bufferSize)

	// Start goroutine to convert messages to JSON
	go func() {
		defer close(jsonChan)
		defer subResult.Cleanup()

		for msg := range subResult.Messages {
			// Convert message to GraphQL response format
			response := map[string]any{}
			if msg.Data != nil {
				response["data"] = msg.Data
			}
			if len(msg.Errors) > 0 {
				response["errors"] = msg.Errors
			}

			// Marshal to JSON
			jsonBytes, err := json.Marshal(response)
			if err != nil {
				// Send error response
				errorResponse := map[string]any{
					"errors": []error{fmt.Errorf("failed to marshal response: %v", err)},
				}
				if errorJson, _ := json.Marshal(errorResponse); errorJson != nil {
					jsonChan <- string(errorJson)
				}
				continue
			}

			jsonChan <- string(jsonBytes)
		}
	}()

	return jsonChan, nil
}
