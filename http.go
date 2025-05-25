package quickgraph

import (
	"encoding/json"
	"github.com/gburgyan/go-timing"
	"log"
	"net/http"
	"runtime/debug"
)

type GraphHttpHandler struct {
	graphy *Graphy
}

func (g *Graphy) HttpHandler() http.Handler {
	return &GraphHttpHandler{
		graphy: g,
	}
}

type graphqlRequest struct {
	Query     string          `json:"query"`
	Variables json.RawMessage `json:"variables"`
}

func (g GraphHttpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// Recover from any panics to prevent server crashes
	defer func() {
		if r := recover(); r != nil {
			// Log the panic details internally (including stack trace)
			log.Printf("Panic in GraphQL HTTP handler: %v\nStack trace:\n%s", r, debug.Stack())

			// Return a generic error to the client (no internal details)
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(500)
			errorResponse := `{"errors":[{"message":"Internal server error"}]}`
			_, _ = writer.Write([]byte(errorResponse))
		}
	}()

	ctx := request.Context()
	var timingContext *timing.Context
	var complete timing.Complete

	if g.graphy.EnableTiming {
		timingContext, complete = timing.Start(ctx, "HttpHandler")
		ctx = timingContext
	}

	if request.Method == "GET" {
		if g.graphy.schemaEnabled {
			schema := g.graphy.SchemaDefinition(ctx)
			writer.WriteHeader(200)
			_, err := writer.Write([]byte(schema))
			if err != nil {
				log.Printf("Error writing response: %v", err)
			}
		} else {
			writer.WriteHeader(404)
			_, err := writer.Write([]byte("Not Found"))
			if err != nil {
				log.Printf("Error writing response: %v", err)
			}
		}
		return
	}

	var req graphqlRequest
	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		log.Printf("Error decoding request: %v", err)
		writer.WriteHeader(400)
		return
	}

	query := req.Query
	variables := string(req.Variables)

	// Process the request.
	res, err := g.graphy.ProcessRequest(ctx, query, variables)
	if err != nil {
		log.Printf("Error processing request: %v (will still return response)", err)
	}

	// Return the response string.
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(200) // Errors are in the response body, and there may be mixed errors and results.
	_, err = writer.Write([]byte(res))
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}

	if g.graphy.EnableTiming {
		complete()
		log.Printf("Timing: %v", timingContext.String())
	}
}
