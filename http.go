package quickgraph

import (
	"encoding/json"
	"fmt"
	"github.com/gburgyan/go-timing"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
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

// setCORSHeaders sets CORS headers based on the configured CORSSettings
func (g *GraphHttpHandler) setCORSHeaders(writer http.ResponseWriter, forOptionsRequest bool) {
	cors := g.graphy.CORSSettings
	if cors == nil {
		return
	}

	// Set CORS headers for OPTIONS requests or if EnableForAllResponses is true
	if forOptionsRequest || cors.EnableForAllResponses {
		// Set allowed origins
		origins := cors.getEffectiveOrigins()
		writer.Header().Set("Access-Control-Allow-Origin", strings.Join(origins, ", "))

		// Set allowed methods
		methods := cors.getEffectiveMethods()
		writer.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ", "))

		// Set allowed headers
		headers := cors.getEffectiveHeaders()
		writer.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ", "))

		// Set exposed headers if specified
		if len(cors.ExposedHeaders) > 0 {
			writer.Header().Set("Access-Control-Expose-Headers", strings.Join(cors.ExposedHeaders, ", "))
		}

		// Set credentials if allowed
		if cors.AllowCredentials {
			writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		// Set max age for OPTIONS requests
		if forOptionsRequest {
			maxAge := cors.getEffectiveMaxAge()
			writer.Header().Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
		}
	}
}

func (g GraphHttpHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	// Recover from any panics to prevent server crashes
	defer func() {
		if r := recover(); r != nil {
			// Create panic error for HTTP handler
			err := GraphError{
				Message:           "Internal server error",
				ProductionMessage: "Internal server error",
				InnerError:        fmt.Errorf("panic: %v", r),
			}

			// Add sensitive information that will be filtered in production
			if err.SensitiveExtensions == nil {
				err.SensitiveExtensions = make(map[string]string)
			}
			err.SensitiveExtensions["stack_trace"] = string(debug.Stack())
			err.SensitiveExtensions["panic_value"] = fmt.Sprintf("%v", r)

			// Create details for error handler
			details := map[string]interface{}{
				"operation":      "http_handler_panic",
				"request_method": request.Method,
				"request_path":   request.URL.Path,
				"panic_value":    fmt.Sprintf("%v", r),
				"stack_trace":    string(debug.Stack()),
			}

			// Report through error handler (this will include all details)
			g.graphy.handleError(request.Context(), ErrorCategoryInternal, err, details)

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

	if request.Method == "OPTIONS" {
		// Handle CORS preflight requests
		g.setCORSHeaders(writer, true)
		writer.WriteHeader(204)
		return
	}

	if request.Method == "GET" {
		g.setCORSHeaders(writer, false)
		if g.graphy.schemaEnabled {
			schema := g.graphy.SchemaDefinition(ctx)
			writer.WriteHeader(200)
			_, err := writer.Write([]byte(schema))
			if err != nil {
				g.graphy.handleError(ctx, ErrorCategoryHTTP, err, map[string]interface{}{
					"operation":      "write_schema_response",
					"request_method": request.Method,
				})
			}
		} else {
			writer.WriteHeader(404)
			_, err := writer.Write([]byte("Not Found"))
			if err != nil {
				g.graphy.handleError(ctx, ErrorCategoryHTTP, err, map[string]interface{}{
					"operation":      "write_not_found_response",
					"request_method": request.Method,
				})
			}
		}
		return
	}

	var req graphqlRequest

	// Apply memory limits if configured
	var bodyReader io.Reader = request.Body
	if g.graphy.MemoryLimits != nil && g.graphy.MemoryLimits.MaxRequestBodySize > 0 {
		bodyReader = io.LimitReader(request.Body, g.graphy.MemoryLimits.MaxRequestBodySize)
	}

	err := json.NewDecoder(bodyReader).Decode(&req)
	if err != nil {
		g.graphy.handleError(ctx, ErrorCategoryHTTP, err, map[string]interface{}{
			"operation":      "decode_request_body",
			"request_method": request.Method,
		})
		writer.WriteHeader(400)
		return
	}

	query := req.Query
	variables := string(req.Variables)

	// Process the request.
	res, err := g.graphy.ProcessRequest(ctx, query, variables)
	if err != nil {
		g.graphy.handleError(ctx, ErrorCategoryExecution, err, map[string]interface{}{
			"operation": "process_request",
			"query":     query,
			"variables": variables,
		})
	}

	// Return the response string.
	g.setCORSHeaders(writer, false)
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(200) // Errors are in the response body, and there may be mixed errors and results.

	_, err = writer.Write([]byte(res))
	if err != nil {
		g.graphy.handleError(ctx, ErrorCategoryHTTP, err, map[string]interface{}{
			"operation":      "write_response",
			"request_method": request.Method,
		})
	}

	if g.graphy.EnableTiming {
		complete()
		// Keep timing logs as they are - they're not errors
		log.Printf("Timing: %v", timingContext.String())
	}
}
