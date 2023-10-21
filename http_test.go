package quickgraph

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGraphHttpHandler_ServeHTTP_GetSchema(t *testing.T) {
	g := Graphy{}
	g.RegisterQuery(nil, "greeting", func(ctx context.Context, name string) (string, error) {
		return "Hello, " + name, nil
	}, "name")

	h := g.HttpHandler()

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	res := rec.Result()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", res.Status)
	}

	genSchema, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("could not read response: %v", err)
	}

	schema := `type Query {
	greeting(name: String!): String!
}

`
	// Here you can also assert the content of the response if you know what the schema should look like
	assert.Equal(t, schema, string(genSchema))
}

func TestGraphHttpHandler_ServeHTTP_PostQuery(t *testing.T) {
	g := Graphy{}
	g.RegisterQuery(nil, "greeting", func(ctx context.Context, name string) (string, error) {
		return "Hello, " + name, nil
	}, "name")

	h := g.HttpHandler()

	// Construct a request with a query and variables.
	query := `query Greeting($name: String!) {
	greeting(name: $name)
}`
	variables := `{
	"name": "World"
}`

	graphRequest := graphqlRequest{
		Query:     query,
		Variables: json.RawMessage(variables),
	}

	body, _ := json.Marshal(graphRequest)

	// Make a reader for the body
	bodyReader := bytes.NewReader(body)

	req, _ := http.NewRequest("POST", "/", bodyReader)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()
	resBody, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", res.Status)
	}

	assert.Equal(t, `{"data":{"greeting":"Hello, World"}}`, string(resBody))
}
