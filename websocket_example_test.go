package quickgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// ExampleWebSocketAdapter shows how to create a WebSocket adapter for gorilla/websocket
type ExampleWebSocketAdapter struct {
	// This would wrap gorilla/websocket.Conn or similar
	// For testing, we'll use channels to simulate
	incoming chan []byte
	outgoing chan []byte
	closed   bool
}

func (e *ExampleWebSocketAdapter) ReadMessage() ([]byte, error) {
	data, ok := <-e.incoming
	if !ok {
		return nil, fmt.Errorf("connection closed")
	}
	return data, nil
}

func (e *ExampleWebSocketAdapter) WriteMessage(data []byte) error {
	if e.closed {
		return fmt.Errorf("connection closed")
	}
	e.outgoing <- data
	return nil
}

func (e *ExampleWebSocketAdapter) Close() error {
	e.closed = true
	close(e.outgoing)
	return nil
}

// ExampleWebSocketUpgrader shows how to implement the upgrader interface
type ExampleWebSocketUpgrader struct{}

func (u *ExampleWebSocketUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (SimpleWebSocketConn, error) {
	// In a real implementation, this would use gorilla/websocket:
	// upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	// conn, err := upgrader.Upgrade(w, r, nil)
	// return &WebSocketAdapter{conn: conn}, err

	// For testing, return our mock adapter
	return &ExampleWebSocketAdapter{
		incoming: make(chan []byte, 10),
		outgoing: make(chan []byte, 10),
	}, nil
}

// TestWebSocketIntegrationExample demonstrates the WebSocket integration
func TestWebSocketIntegrationExample(t *testing.T) {
	// Create a Graphy instance with a subscription
	g := Graphy{}
	ctx := context.Background()

	// Register a time ticker subscription
	g.RegisterSubscription(ctx, "currentTime", func(ctx context.Context, intervalMs int) (<-chan TimeUpdate, error) {
		if intervalMs < 100 {
			intervalMs = 100 // Minimum interval
		}

		ch := make(chan TimeUpdate)

		go func() {
			defer close(ch)
			ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case t := <-ticker.C:
					select {
					case ch <- TimeUpdate{
						Timestamp: t.Unix(),
						Formatted: t.Format(time.RFC3339),
					}:
					case <-ctx.Done():
						return
					}
				}
			}
		}()

		return ch, nil
	})

	// Create HTTP handler with WebSocket support
	upgrader := &ExampleWebSocketUpgrader{}
	_ = g.HttpHandlerWithWebSocket(upgrader) // handler would be used in real server

	// In a real server:
	// handler := g.HttpHandlerWithWebSocket(upgrader)
	// http.Handle("/graphql", handler)
	// http.ListenAndServe(":8080", nil)

	// For this example, we'll simulate the WebSocket protocol
	adapter := &ExampleWebSocketAdapter{
		incoming: make(chan []byte, 10),
		outgoing: make(chan []byte, 10),
	}

	// Simulate WebSocket connection
	go func() {
		wsHandler := NewGraphQLWebSocketHandler(&g)
		wsHandler.HandleConnection(ctx, adapter)
	}()

	// Send connection init
	initMsg := WebSocketMessage{
		Type: GQLConnectionInit,
	}
	initData, _ := json.Marshal(initMsg)
	adapter.incoming <- initData

	// Wait for connection ack
	select {
	case msg := <-adapter.outgoing:
		var ack WebSocketMessage
		json.Unmarshal(msg, &ack)
		if ack.Type != GQLConnectionAck {
			t.Fatalf("Expected connection ack, got %s", ack.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for connection ack")
	}

	// Subscribe to time updates
	subMsg := WebSocketMessage{
		ID:   "sub1",
		Type: GQLSubscribe,
		Payload: json.RawMessage(`{
			"query": "subscription { currentTime(intervalMs: 100) { timestamp formatted } }"
		}`),
	}
	subData, _ := json.Marshal(subMsg)
	adapter.incoming <- subData

	// Receive a few time updates
	receivedCount := 0
	timeout := time.After(500 * time.Millisecond)

loop:
	for {
		select {
		case msg := <-adapter.outgoing:
			var wsMsg WebSocketMessage
			json.Unmarshal(msg, &wsMsg)
			if wsMsg.Type == GQLNext {
				receivedCount++
				if receivedCount >= 3 {
					break loop
				}
			}
		case <-timeout:
			break loop
		}
	}

	// Complete the subscription
	completeMsg := WebSocketMessage{
		ID:   "sub1",
		Type: GQLComplete,
	}
	completeData, _ := json.Marshal(completeMsg)
	adapter.incoming <- completeData

	// Close connection
	close(adapter.incoming)

	// Verify we received some updates
	if receivedCount < 1 {
		t.Fatal("Did not receive any time updates")
	}
}

// TimeUpdate represents a time update message
type TimeUpdate struct {
	Timestamp int64
	Formatted string
}

// Example of how to use with gorilla/websocket (commented out to avoid dependency):
/*
import "github.com/gorilla/websocket"

type GorillaWebSocketAdapter struct {
	conn *websocket.Conn
}

func (a *GorillaWebSocketAdapter) ReadMessage() ([]byte, error) {
	_, data, err := a.conn.ReadMessage()
	return data, err
}

func (a *GorillaWebSocketAdapter) WriteMessage(data []byte) error {
	return a.conn.WriteMessage(websocket.TextMessage, data)
}

func (a *GorillaWebSocketAdapter) Close() error {
	return a.conn.Close()
}

type GorillaWebSocketUpgrader struct {
	upgrader websocket.Upgrader
}

func NewGorillaUpgrader() *GorillaWebSocketUpgrader {
	return &GorillaWebSocketUpgrader{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Configure origin checking as needed
				return true
			},
		},
	}
}

func (u *GorillaWebSocketUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (SimpleWebSocketConn, error) {
	conn, err := u.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &GorillaWebSocketAdapter{conn: conn}, nil
}

// Usage:
func main() {
	g := &Graphy{}
	// ... register subscriptions ...

	upgrader := NewGorillaUpgrader()
	handler := g.HttpHandlerWithWebSocket(upgrader)

	http.Handle("/graphql", handler)
	http.ListenAndServe(":8080", nil)
}
*/
