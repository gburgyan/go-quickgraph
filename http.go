package quickgraph

import (
	"encoding/json"
	"log"
	"net/http"
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
	if request.Method == "GET" {
		if g.graphy.schemaEnabled {
			schema := g.graphy.SchemaDefinition(request.Context())
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
	res, err := g.graphy.ProcessRequest(request.Context(), query, variables)
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
}
