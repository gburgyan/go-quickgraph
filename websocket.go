package quickgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

// GraphQLWebSocketHandler handles GraphQL subscriptions over WebSocket
type GraphQLWebSocketHandler struct {
	graphy *Graphy
}

// NewGraphQLWebSocketHandler creates a new WebSocket handler for GraphQL subscriptions
func NewGraphQLWebSocketHandler(graphy *Graphy) *GraphQLWebSocketHandler {
	return &GraphQLWebSocketHandler{
		graphy: graphy,
	}
}

// HandleConnection handles a WebSocket connection using the graphql-ws protocol
func (h *GraphQLWebSocketHandler) HandleConnection(ctx context.Context, conn SimpleWebSocketConn) {
	defer conn.Close()

	// Tracks active subscriptions
	subscriptions := make(map[string]context.CancelFunc)
	mu := sync.Mutex{}

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
				log.Printf("WebSocket read error: %v", err)
			}
			return
		}

		var msg WebSocketMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("Failed to parse WebSocket message: %v", err)
			continue
		}

		switch msg.Type {
		case GQLConnectionInit:
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
			// Parse subscription request
			var payload struct {
				Query     string          `json:"query"`
				Variables json.RawMessage `json:"variables"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				h.sendError(conn, msg.ID, fmt.Errorf("invalid subscribe payload: %v", err))
				continue
			}

			// Cancel any existing subscription with the same ID
			mu.Lock()
			if cancel, exists := subscriptions[msg.ID]; exists {
				cancel()
			}
			mu.Unlock()

			// Create subscription context
			subCtx, cancel := context.WithCancel(ctx)
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
			log.Printf("WebSocket upgrade failed: %v", err)
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
