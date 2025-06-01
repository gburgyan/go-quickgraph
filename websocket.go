package quickgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// WebSocketHandler is an interface for WebSocket implementations
// This allows users to plug in their preferred WebSocket library
type WebSocketHandler interface {
	// HandleWebSocket upgrades an HTTP connection to WebSocket and handles GraphQL subscriptions
	HandleWebSocket(w http.ResponseWriter, r *http.Request, graphy *Graphy)
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// GraphQLWSMessage types based on graphql-ws protocol
const (
	// Client -> Server
	GQLConnectionInit      = "connection_init"
	GQLConnectionTerminate = "connection_terminate"
	GQLSubscribe           = "subscribe"
	GQLComplete            = "complete"

	// Server -> Client
	GQLConnectionAck   = "connection_ack"
	GQLConnectionError = "connection_error"
	GQLNext            = "next"
	GQLError           = "error"
	GQLComplete_       = "complete"
)

// SimpleWebSocketConn is a minimal interface for WebSocket connections
// that different WebSocket libraries can implement
type SimpleWebSocketConn interface {
	ReadMessage() ([]byte, error)
	WriteMessage(data []byte) error
	Close() error
}

// WebSocketAuthenticator provides a flexible interface for authenticating WebSocket connections.
// Customers can implement this interface to add their own authentication logic.
type WebSocketAuthenticator interface {
	// AuthenticateConnection is called when a WebSocket connection is established.
	// It receives the connection initialization payload and should return:
	// - An authenticated context if authentication succeeds
	// - An error if authentication fails (this will close the connection)
	// The returned context will be used for all subsequent operations on this connection.
	AuthenticateConnection(ctx context.Context, initPayload json.RawMessage) (context.Context, error)

	// AuthorizeSubscription is called before each subscription is created.
	// It receives the authenticated context and subscription request and should return:
	// - An authorized context if the subscription is allowed
	// - An error if the subscription should be denied
	// This allows for fine-grained per-subscription authorization.
	AuthorizeSubscription(ctx context.Context, query string, variables json.RawMessage) (context.Context, error)
}

// NoOpWebSocketAuthenticator is a default implementation that allows all connections and subscriptions.
// This maintains backward compatibility for customers who don't need authentication.
type NoOpWebSocketAuthenticator struct{}

func (n NoOpWebSocketAuthenticator) AuthenticateConnection(ctx context.Context, initPayload json.RawMessage) (context.Context, error) {
	return ctx, nil
}

func (n NoOpWebSocketAuthenticator) AuthorizeSubscription(ctx context.Context, query string, variables json.RawMessage) (context.Context, error) {
	return ctx, nil
}

// GraphQLWebSocketHandler handles GraphQL subscriptions over WebSocket
type GraphQLWebSocketHandler struct {
	graphy        *Graphy
	authenticator WebSocketAuthenticator
}

// NewGraphQLWebSocketHandler creates a new WebSocket handler for GraphQL subscriptions
func NewGraphQLWebSocketHandler(graphy *Graphy) *GraphQLWebSocketHandler {
	return &GraphQLWebSocketHandler{
		graphy:        graphy,
		authenticator: NoOpWebSocketAuthenticator{}, // Default to no authentication
	}
}

// NewGraphQLWebSocketHandlerWithAuth creates a new WebSocket handler with custom authentication
func NewGraphQLWebSocketHandlerWithAuth(graphy *Graphy, authenticator WebSocketAuthenticator) *GraphQLWebSocketHandler {
	return &GraphQLWebSocketHandler{
		graphy:        graphy,
		authenticator: authenticator,
	}
}

// NOTE: For global WebSocket connection limits (MemoryLimits.MaxWebSocketConnections),
// customers should implement connection tracking at the HTTP upgrade layer.
// Example implementation:
//
// type ConnectionTracker struct {
//     mu sync.Mutex
//     count int
//     maxConnections int
// }
//
// func (ct *ConnectionTracker) CanConnect() bool {
//     ct.mu.Lock()
//     defer ct.mu.Unlock()
//     return ct.count < ct.maxConnections
// }
//
// func (ct *ConnectionTracker) OnConnect() {
//     ct.mu.Lock()
//     defer ct.mu.Unlock()
//     ct.count++
// }
//
// func (ct *ConnectionTracker) OnDisconnect() {
//     ct.mu.Lock()
//     defer ct.mu.Unlock()
//     ct.count--
// }

// HandleConnection handles a WebSocket connection using the graphql-ws protocol
func (h *GraphQLWebSocketHandler) HandleConnection(ctx context.Context, conn SimpleWebSocketConn) {
	defer conn.Close()

	// Tracks active subscriptions
	subscriptions := make(map[string]context.CancelFunc)
	mu := sync.Mutex{}

	// Track authentication state
	var authenticatedCtx context.Context
	authenticated := false

	// Cleanup all subscriptions on disconnect
	defer func() {
		mu.Lock()
		for _, cancel := range subscriptions {
			cancel()
		}
		mu.Unlock()
	}()

	// Handle incoming messages
	for {
		data, err := conn.ReadMessage()
		if err != nil {
			if err != io.EOF {
				h.graphy.handleError(ctx, ErrorCategoryWebSocket, err, map[string]interface{}{
					"operation": "read_websocket_message",
				})
			}
			return
		}

		var msg WebSocketMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			h.graphy.handleError(ctx, ErrorCategoryWebSocket, err, map[string]interface{}{
				"operation":   "parse_websocket_message",
				"raw_message": string(data),
			})
			continue
		}

		switch msg.Type {
		case GQLConnectionInit:
			// Authenticate the connection
			authCtx, err := h.authenticator.AuthenticateConnection(ctx, msg.Payload)
			if err != nil {
				// Send connection error and close
				errorPayload := map[string]string{
					"message": fmt.Sprintf("Authentication failed: %s", err.Error()),
				}
				payloadBytes, _ := json.Marshal(errorPayload)
				errMsg := WebSocketMessage{
					Type:    GQLConnectionError,
					Payload: json.RawMessage(payloadBytes),
				}
				if data, marshalErr := json.Marshal(errMsg); marshalErr == nil {
					conn.WriteMessage(data)
				}
				return
			}

			// Authentication successful
			authenticatedCtx = authCtx
			authenticated = true

			// Send connection acknowledgment
			ack := WebSocketMessage{
				Type: GQLConnectionAck,
			}
			if data, err := json.Marshal(ack); err == nil {
				conn.WriteMessage(data)
			}

		case GQLConnectionTerminate:
			return

		case GQLSubscribe:
			// Check if connection is authenticated
			if !authenticated {
				h.sendError(conn, msg.ID, fmt.Errorf("connection not authenticated"))
				continue
			}

			// Parse subscription request
			var payload struct {
				Query     string          `json:"query"`
				Variables json.RawMessage `json:"variables"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				h.sendError(conn, msg.ID, fmt.Errorf("invalid subscribe payload: %v", err))
				continue
			}

			// Authorize the subscription
			authorizedCtx, err := h.authenticator.AuthorizeSubscription(authenticatedCtx, payload.Query, payload.Variables)
			if err != nil {
				h.sendError(conn, msg.ID, fmt.Errorf("subscription not authorized: %v", err))
				continue
			}

			// Check subscription limits
			mu.Lock()
			// Cancel any existing subscription with the same ID first
			if cancel, exists := subscriptions[msg.ID]; exists {
				cancel()
				delete(subscriptions, msg.ID)
			}

			// Check if we're at the subscription limit
			if h.graphy.MemoryLimits != nil && h.graphy.MemoryLimits.MaxSubscriptionsPerConnection > 0 {
				if len(subscriptions) >= h.graphy.MemoryLimits.MaxSubscriptionsPerConnection {
					mu.Unlock()
					h.sendError(conn, msg.ID, fmt.Errorf("maximum subscriptions per connection (%d) exceeded", h.graphy.MemoryLimits.MaxSubscriptionsPerConnection))
					continue
				}
			}
			mu.Unlock()

			// Create subscription context
			subCtx, cancel := context.WithCancel(authorizedCtx)
			mu.Lock()
			subscriptions[msg.ID] = cancel
			mu.Unlock()

			// Start subscription in goroutine
			go func(id string) {
				defer func() {
					mu.Lock()
					delete(subscriptions, id)
					mu.Unlock()

					// Send complete message
					complete := WebSocketMessage{
						ID:   id,
						Type: GQLComplete_,
					}
					if data, err := json.Marshal(complete); err == nil {
						conn.WriteMessage(data)
					}
				}()

				// Process subscription
				msgChan, err := h.graphy.ProcessSubscription(subCtx, payload.Query, string(payload.Variables))
				if err != nil {
					// Categorize the error appropriately
					h.graphy.handleError(subCtx, ErrorCategoryValidation, err, map[string]interface{}{
						"operation":       "process_subscription",
						"subscription_id": id,
						"query":           payload.Query,
						"variables":       string(payload.Variables),
					})
					h.sendError(conn, id, err)
					return
				}

				// Forward messages
				for jsonMsg := range msgChan {
					next := WebSocketMessage{
						ID:      id,
						Type:    GQLNext,
						Payload: json.RawMessage(jsonMsg),
					}
					if data, err := json.Marshal(next); err == nil {
						if err := conn.WriteMessage(data); err != nil {
							// Connection closed
							return
						}
					}
				}
			}(msg.ID)

		case GQLComplete:
			// Cancel subscription
			mu.Lock()
			if cancel, exists := subscriptions[msg.ID]; exists {
				cancel()
				delete(subscriptions, msg.ID)
			}
			mu.Unlock()
		}
	}
}

func (h *GraphQLWebSocketHandler) sendError(conn SimpleWebSocketConn, id string, err error) {
	errMsg := WebSocketMessage{
		ID:      id,
		Type:    GQLError,
		Payload: json.RawMessage(fmt.Sprintf(`{"errors":[{"message":"%s"}]}`, err.Error())),
	}
	if data, err := json.Marshal(errMsg); err == nil {
		conn.WriteMessage(data)
	}
}

// WebSocketUpgrader is an interface for upgrading HTTP connections to WebSocket
// This allows different WebSocket libraries to be used
type WebSocketUpgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request) (SimpleWebSocketConn, error)
}

// AddWebSocketSupport adds WebSocket support to the existing HTTP handler
// This requires the user to provide a WebSocketUpgrader implementation
func (g *Graphy) HttpHandlerWithWebSocket(upgrader WebSocketUpgrader) http.Handler {
	return &GraphHttpHandlerWithWS{
		graphy:    g,
		upgrader:  upgrader,
		wsHandler: NewGraphQLWebSocketHandler(g),
	}
}

type GraphHttpHandlerWithWS struct {
	graphy    *Graphy
	upgrader  WebSocketUpgrader
	wsHandler *GraphQLWebSocketHandler
}

func (h *GraphHttpHandlerWithWS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Check if this is a WebSocket upgrade request
	if r.Header.Get("Upgrade") == "websocket" {
		conn, err := h.upgrader.Upgrade(w, r)
		if err != nil {
			h.graphy.handleError(r.Context(), ErrorCategoryWebSocket, err, map[string]interface{}{
				"operation":      "websocket_upgrade",
				"request_method": r.Method,
				"request_path":   r.URL.Path,
			})
			http.Error(w, "WebSocket upgrade failed", http.StatusBadRequest)
			return
		}

		// Set a reasonable timeout for the WebSocket connection
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Hour)
		defer cancel()

		h.wsHandler.HandleConnection(ctx, conn)
		return
	}

	// Fall back to regular HTTP handler
	httpHandler := h.graphy.HttpHandler()
	httpHandler.ServeHTTP(w, r)
}
