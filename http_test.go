package quickgraph

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type errorWriter struct {
	status int
}

func (e *errorWriter) Header() http.Header {
	return map[string][]string{}
}

func (e *errorWriter) Write(i []byte) (int, error) {
	return 0, errors.New("error")
}

func (e *errorWriter) WriteHeader(status int) {
	e.status = status
}

func TestGraphHttpHandler_ServeHTTP_GetSchema(t *testing.T) {
	g := Graphy{EnableTiming: true}
	ctx := context.Background()
	g.RegisterQuery(ctx, "greeting", func(ctx context.Context, name string) (string, error) {
		return "Hello, " + name, nil
	}, "name")

	g.EnableIntrospection(ctx)
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

	errorWriter := &errorWriter{}
	h.ServeHTTP(errorWriter, req)
	// Verify the error is logged.
}

func TestGraphHttpHandler_ServeHTTP_GetSchema_Error(t *testing.T) {
	g := Graphy{EnableTiming: true}
	ctx := context.Background()
	g.RegisterQuery(ctx, "greeting", func(ctx context.Context, name string) (string, error) {
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

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected status 404; got %v", res.Status)
	}

	errorWriter := errorWriter{}
	h.ServeHTTP(&errorWriter, req)
	// Verify the error is logged.
}

func TestGraphHttpHandler_ServeHTTP_PostQuery(t *testing.T) {
	g := Graphy{EnableTiming: true}
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

	bodyReader = bytes.NewReader(body)
	req = httptest.NewRequest("POST", "/", bodyReader)
	errorWriter := errorWriter{}
	h.ServeHTTP(&errorWriter, req)
	// Verify the error is logged.
}

func TestGraphHttpHandler_ServeHTTP_PostQuery_BadJSON(t *testing.T) {
	g := Graphy{EnableTiming: true}
	g.RegisterQuery(nil, "greeting", func(ctx context.Context, name string) (string, error) {
		return "Hello, " + name, nil
	}, "name")

	h := g.HttpHandler()

	// Construct a request with a query and variables.
	query := `query Greeting($name: String!) {
	greeting(name: $name)
}`
	variables := `={
	"name": "World"
} bad json`

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

	// Verify the error is logged.
	assert.Equal(t, http.StatusBadRequest, res.StatusCode)
}

func TestGraphHttpHandler_ServeHTTP_PostQuery_Error(t *testing.T) {
	g := Graphy{EnableTiming: true}
	g.RegisterQuery(nil, "greeting", func(ctx context.Context, name string) (string, error) {
		return "", errors.New("expected error")
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

	assert.Equal(t, `{"data":{},"errors":[{"message":"function greeting returned error: expected error","locations":[{"line":2,"column":11}],"path":["greeting"]}]}`, string(resBody))
}

func TestGraphHttpHandler_ServeHTTP_Options_NoCORS(t *testing.T) {
	g := Graphy{EnableTiming: true}
	// No CORSSettings configured - should not add CORS headers
	g.RegisterQuery(nil, "greeting", func(ctx context.Context, name string) (string, error) {
		return "Hello, " + name, nil
	}, "name")

	h := g.HttpHandler()

	req, err := http.NewRequest("OPTIONS", "/", nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	res := rec.Result()

	// Verify the status code is 204 No Content
	assert.Equal(t, http.StatusNoContent, res.StatusCode)

	// Verify no CORS headers are set when CORSSettings is nil
	assert.Empty(t, res.Header.Get("Access-Control-Allow-Origin"))
	assert.Empty(t, res.Header.Get("Access-Control-Allow-Methods"))
	assert.Empty(t, res.Header.Get("Access-Control-Allow-Headers"))
	assert.Empty(t, res.Header.Get("Access-Control-Max-Age"))

	// Verify no body is returned
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("could not read response body: %v", err)
	}
	assert.Empty(t, body)
}

func TestGraphHttpHandler_ServeHTTP_Options_DefaultCORS(t *testing.T) {
	g := Graphy{
		EnableTiming: true,
		CORSSettings: DefaultCORSSettings(),
	}
	g.RegisterQuery(nil, "greeting", func(ctx context.Context, name string) (string, error) {
		return "Hello, " + name, nil
	}, "name")

	h := g.HttpHandler()

	req, err := http.NewRequest("OPTIONS", "/", nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	res := rec.Result()

	// Verify the status code is 204 No Content
	assert.Equal(t, http.StatusNoContent, res.StatusCode)

	// Verify default CORS headers are set correctly
	assert.Equal(t, "*", res.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "GET, POST, OPTIONS", res.Header.Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, Authorization", res.Header.Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "86400", res.Header.Get("Access-Control-Max-Age"))
	assert.Empty(t, res.Header.Get("Access-Control-Allow-Credentials")) // Should not be set when false

	// Verify no body is returned
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("could not read response body: %v", err)
	}
	assert.Empty(t, body)
}

func TestGraphHttpHandler_ServeHTTP_Options_WithErrorWriter(t *testing.T) {
	g := Graphy{
		EnableTiming: true,
		CORSSettings: DefaultCORSSettings(),
	}
	g.RegisterQuery(nil, "greeting", func(ctx context.Context, name string) (string, error) {
		return "Hello, " + name, nil
	}, "name")

	h := g.HttpHandler()

	req, err := http.NewRequest("OPTIONS", "/", nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}

	errorWriter := &errorWriter{}
	h.ServeHTTP(errorWriter, req)

	// Verify the status code is set to 204 even with error writer
	assert.Equal(t, http.StatusNoContent, errorWriter.status)
}

func TestGraphHttpHandler_ServeHTTP_Options_CustomCORS(t *testing.T) {
	g := Graphy{
		EnableTiming: true,
		CORSSettings: &CORSSettings{
			AllowedOrigins:   []string{"https://example.com", "https://app.example.com"},
			AllowedMethods:   []string{"POST", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "X-Custom-Header"},
			ExposedHeaders:   []string{"X-Response-ID"},
			AllowCredentials: true,
			MaxAge:           3600,
		},
	}
	g.RegisterQuery(nil, "greeting", func(ctx context.Context, name string) (string, error) {
		return "Hello, " + name, nil
	}, "name")

	h := g.HttpHandler()

	req, err := http.NewRequest("OPTIONS", "/", nil)
	if err != nil {
		t.Fatalf("could not create request: %v", err)
	}
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	res := rec.Result()

	// Verify the status code is 204 No Content
	assert.Equal(t, http.StatusNoContent, res.StatusCode)

	// Verify custom CORS headers are set correctly
	assert.Equal(t, "https://example.com, https://app.example.com", res.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "POST, OPTIONS", res.Header.Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type, X-Custom-Header", res.Header.Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "X-Response-ID", res.Header.Get("Access-Control-Expose-Headers"))
	assert.Equal(t, "true", res.Header.Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "3600", res.Header.Get("Access-Control-Max-Age"))
}

func TestGraphHttpHandler_ServeHTTP_POST_CORS(t *testing.T) {
	g := Graphy{
		EnableTiming: true,
		CORSSettings: &CORSSettings{
			AllowedOrigins:        []string{"https://example.com"},
			AllowedMethods:        []string{"POST", "OPTIONS"},
			AllowedHeaders:        []string{"Content-Type"},
			EnableForAllResponses: true,
		},
	}
	g.RegisterQuery(nil, "greeting", func(ctx context.Context, name string) (string, error) {
		return "Hello, " + name, nil
	}, "name")

	h := g.HttpHandler()

	// Construct a POST request
	query := `query { greeting(name: "World") }`
	graphRequest := graphqlRequest{
		Query:     query,
		Variables: json.RawMessage("{}"),
	}
	body, _ := json.Marshal(graphRequest)
	bodyReader := bytes.NewReader(body)

	req, _ := http.NewRequest("POST", "/", bodyReader)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()

	// Verify the request succeeded
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// Verify CORS headers are set on POST response when EnableForAllResponses is true
	assert.Equal(t, "https://example.com", res.Header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "POST, OPTIONS", res.Header.Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "Content-Type", res.Header.Get("Access-Control-Allow-Headers"))
	// Max-Age should not be set for non-OPTIONS requests
	assert.Empty(t, res.Header.Get("Access-Control-Max-Age"))
}

func TestGraphHttpHandler_ServeHTTP_POST_CORS_DisabledForResponses(t *testing.T) {
	g := Graphy{
		EnableTiming: true,
		CORSSettings: &CORSSettings{
			AllowedOrigins:        []string{"https://example.com"},
			AllowedMethods:        []string{"POST", "OPTIONS"},
			AllowedHeaders:        []string{"Content-Type"},
			EnableForAllResponses: false, // Default behavior
		},
	}
	g.RegisterQuery(nil, "greeting", func(ctx context.Context, name string) (string, error) {
		return "Hello, " + name, nil
	}, "name")

	h := g.HttpHandler()

	// Construct a POST request
	query := `query { greeting(name: "World") }`
	graphRequest := graphqlRequest{
		Query:     query,
		Variables: json.RawMessage("{}"),
	}
	body, _ := json.Marshal(graphRequest)
	bodyReader := bytes.NewReader(body)

	req, _ := http.NewRequest("POST", "/", bodyReader)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := rec.Result()

	// Verify the request succeeded
	assert.Equal(t, http.StatusOK, res.StatusCode)

	// Verify CORS headers are NOT set on POST response when EnableForAllResponses is false
	assert.Empty(t, res.Header.Get("Access-Control-Allow-Origin"))
	assert.Empty(t, res.Header.Get("Access-Control-Allow-Methods"))
	assert.Empty(t, res.Header.Get("Access-Control-Allow-Headers"))
}
