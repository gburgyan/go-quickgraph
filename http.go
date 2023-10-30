package quickgraph

import (
	"encoding/json"
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
	// Get the query and variables from the request form data.

	if request.Method == "GET" {
		schema := g.graphy.SchemaDefinition(request.Context())
		writer.WriteHeader(200)
		_, err := writer.Write([]byte(schema))
		if err != nil {
			panic(err)
		}
		return
	}

	var req graphqlRequest
	err := json.NewDecoder(request.Body).Decode(&req)

	query := req.Query
	variables := string(req.Variables)

	// Process the request.
	res, err := g.graphy.ProcessRequest(request.Context(), query, variables)
	if err != nil {
		// TODO: Log the error here, but the response still has a GraphQL response that can be returned.
	}
	// Return the response string.
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(200) // Errors are in the response body, and there may be mixed errors and results.
	_, err = writer.Write([]byte(res))
	if err != nil {
		panic(err)
	}
}
