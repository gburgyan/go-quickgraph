package quickgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Import the test types from websocket_auth_test.go
// MockAuthWebSocketConn, AuthTestUser, and TestAuthenticator are defined there

func TestWebSocketSubscriptionLimits_MaxSubscriptionsPerConnection(t *testing.T) {
	tests := []struct {
		name                        string
		maxSubscriptionsPerConn     int
		subscriptionCount           int
		expectRejectedSubscriptions int
	}{
		{
			name:                        "Under limit",
			maxSubscriptionsPerConn:     5,
			subscriptionCount:           3,
			expectRejectedSubscriptions: 0,
		},
		{
			name:                        "At limit",
			maxSubscriptionsPerConn:     3,
			subscriptionCount:           3,
			expectRejectedSubscriptions: 0,
		},
		{
			name:                        "Over limit",
			maxSubscriptionsPerConn:     2,
			subscriptionCount:           4,
			expectRejectedSubscriptions: 2,
		},
		{
			name:                        "No limit configured",
			maxSubscriptionsPerConn:     0,
			subscriptionCount:           10,
			expectRejectedSubscriptions: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup GraphQL with memory limits
			g := &Graphy{}
			if tt.maxSubscriptionsPerConn > 0 {
				g.MemoryLimits = &MemoryLimits{
					MaxSubscriptionsPerConnection: tt.maxSubscriptionsPerConn,
				}
			}

			// Register test subscriptions
			for i := 0; i < tt.subscriptionCount; i++ {
				subName := fmt.Sprintf("testSub%d", i)
				testChan := make(chan string, 1)
				g.RegisterSubscription(context.Background(), subName, func(ctx context.Context) <-chan string {
					return testChan
				})
			}

			// Use NoOp authenticator for simplicity
			handler := NewGraphQLWebSocketHandler(g)

			conn := &MockAuthWebSocketConn{}
			conn.On("Close").Return(nil)
			conn.On("WriteMessage", mock.Anything).Return(nil)

			// Connection init
			initMsg := WebSocketMessage{
				Type:    GQLConnectionInit,
				Payload: json.RawMessage(`{}`),
			}
			initMsgBytes, _ := json.Marshal(initMsg)

			// Create subscription messages
			messages := [][]byte{initMsgBytes}
			for i := 0; i < tt.subscriptionCount; i++ {
				subMsg := WebSocketMessage{
					ID:      fmt.Sprintf("sub-%d", i),
					Type:    GQLSubscribe,
					Payload: json.RawMessage(fmt.Sprintf(`{"query": "subscription { testSub%d }"}`, i)),
				}
				subMsgBytes, _ := json.Marshal(subMsg)
				messages = append(messages, subMsgBytes)
			}

			// Setup mock to return messages in sequence
			for _, msg := range messages {
				conn.On("ReadMessage").Return(msg, nil).Once()
			}
			conn.On("ReadMessage").Return([]byte{}, fmt.Errorf("EOF")).Maybe()

			// Execute
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			go handler.HandleConnection(ctx, conn)
			time.Sleep(100 * time.Millisecond)
			cancel()

			// Verify results
			sentMessages := conn.GetMessages()

			// First message should be connection ack
			assert.True(t, len(sentMessages) >= 1)
			var ackMsg WebSocketMessage
			err := json.Unmarshal(sentMessages[0], &ackMsg)
			assert.NoError(t, err)
			assert.Equal(t, GQLConnectionAck, ackMsg.Type)

			// Count error messages for rejected subscriptions
			errorCount := 0
			for i := 1; i < len(sentMessages); i++ {
				var msg WebSocketMessage
				err := json.Unmarshal(sentMessages[i], &msg)
				assert.NoError(t, err)

				if msg.Type == GQLError {
					errorCount++
					// Verify it's a subscription limit error
					assert.Contains(t, string(msg.Payload), "maximum subscriptions per connection")
				}
			}

			assert.Equal(t, tt.expectRejectedSubscriptions, errorCount,
				"Number of rejected subscriptions should match expected")

			conn.AssertExpectations(t)
		})
	}
}

func TestWebSocketSubscriptionLimits_SubscriptionReplacement(t *testing.T) {
	// Test that replacing a subscription with the same ID doesn't count against limits
	g := &Graphy{
		MemoryLimits: &MemoryLimits{
			MaxSubscriptionsPerConnection: 2,
		},
	}

	// Register test subscriptions
	testChan1 := make(chan string, 1)
	testChan2 := make(chan string, 1)
	g.RegisterSubscription(context.Background(), "testSub1", func(ctx context.Context) <-chan string {
		return testChan1
	})
	g.RegisterSubscription(context.Background(), "testSub2", func(ctx context.Context) <-chan string {
		return testChan2
	})

	handler := NewGraphQLWebSocketHandler(g)
	conn := &MockAuthWebSocketConn{}
	conn.On("Close").Return(nil)
	conn.On("WriteMessage", mock.Anything).Return(nil)

	// Connection init
	initMsg := WebSocketMessage{
		Type:    GQLConnectionInit,
		Payload: json.RawMessage(`{}`),
	}
	initMsgBytes, _ := json.Marshal(initMsg)

	// First subscription
	sub1Msg := WebSocketMessage{
		ID:      "sub-1",
		Type:    GQLSubscribe,
		Payload: json.RawMessage(`{"query": "subscription { testSub1 }"}`),
	}
	sub1MsgBytes, _ := json.Marshal(sub1Msg)

	// Second subscription
	sub2Msg := WebSocketMessage{
		ID:      "sub-2",
		Type:    GQLSubscribe,
		Payload: json.RawMessage(`{"query": "subscription { testSub2 }"}`),
	}
	sub2MsgBytes, _ := json.Marshal(sub2Msg)

	// Replace first subscription (same ID)
	sub1ReplacementMsg := WebSocketMessage{
		ID:      "sub-1", // Same ID as first subscription
		Type:    GQLSubscribe,
		Payload: json.RawMessage(`{"query": "subscription { testSub2 }"}`), // Different query
	}
	sub1ReplacementMsgBytes, _ := json.Marshal(sub1ReplacementMsg)

	// Setup mock calls
	conn.On("ReadMessage").Return(initMsgBytes, nil).Once()
	conn.On("ReadMessage").Return(sub1MsgBytes, nil).Once()
	conn.On("ReadMessage").Return(sub2MsgBytes, nil).Once()
	conn.On("ReadMessage").Return(sub1ReplacementMsgBytes, nil).Once()
	conn.On("ReadMessage").Return([]byte{}, fmt.Errorf("EOF")).Maybe()

	// Execute
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go handler.HandleConnection(ctx, conn)
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Verify results
	sentMessages := conn.GetMessages()

	// Should have: connection ack + no error messages (all subscriptions should be allowed)
	errorCount := 0
	for _, msgBytes := range sentMessages {
		var msg WebSocketMessage
		err := json.Unmarshal(msgBytes, &msg)
		assert.NoError(t, err)

		if msg.Type == GQLError {
			errorCount++
		}
	}

	assert.Equal(t, 0, errorCount, "No subscriptions should be rejected when replacing existing ones")

	conn.AssertExpectations(t)
}

func TestWebSocketSubscriptionLimits_SubscriptionCancellation(t *testing.T) {
	// Test that cancelled subscriptions free up slots for new ones
	g := &Graphy{
		MemoryLimits: &MemoryLimits{
			MaxSubscriptionsPerConnection: 1, // Only allow 1 subscription
		},
	}

	testChan := make(chan string, 1)
	g.RegisterSubscription(context.Background(), "testSub", func(ctx context.Context) <-chan string {
		return testChan
	})

	handler := NewGraphQLWebSocketHandler(g)
	conn := &MockAuthWebSocketConn{}
	conn.On("Close").Return(nil)
	conn.On("WriteMessage", mock.Anything).Return(nil)

	// Connection init
	initMsg := WebSocketMessage{
		Type:    GQLConnectionInit,
		Payload: json.RawMessage(`{}`),
	}
	initMsgBytes, _ := json.Marshal(initMsg)

	// First subscription
	sub1Msg := WebSocketMessage{
		ID:      "sub-1",
		Type:    GQLSubscribe,
		Payload: json.RawMessage(`{"query": "subscription { testSub }"}`),
	}
	sub1MsgBytes, _ := json.Marshal(sub1Msg)

	// Cancel first subscription
	cancelMsg := WebSocketMessage{
		ID:   "sub-1",
		Type: GQLComplete,
	}
	cancelMsgBytes, _ := json.Marshal(cancelMsg)

	// Second subscription (should be allowed after cancellation)
	sub2Msg := WebSocketMessage{
		ID:      "sub-2",
		Type:    GQLSubscribe,
		Payload: json.RawMessage(`{"query": "subscription { testSub }"}`),
	}
	sub2MsgBytes, _ := json.Marshal(sub2Msg)

	// Setup mock calls
	conn.On("ReadMessage").Return(initMsgBytes, nil).Once()
	conn.On("ReadMessage").Return(sub1MsgBytes, nil).Once()
	conn.On("ReadMessage").Return(cancelMsgBytes, nil).Once()
	conn.On("ReadMessage").Return(sub2MsgBytes, nil).Once()
	conn.On("ReadMessage").Return([]byte{}, fmt.Errorf("EOF")).Maybe()

	// Execute
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go handler.HandleConnection(ctx, conn)
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Verify results - no subscription should be rejected
	sentMessages := conn.GetMessages()

	errorCount := 0
	for _, msgBytes := range sentMessages {
		var msg WebSocketMessage
		err := json.Unmarshal(msgBytes, &msg)
		assert.NoError(t, err)

		if msg.Type == GQLError {
			errorCount++
		}
	}

	assert.Equal(t, 0, errorCount, "Second subscription should be allowed after first is cancelled")

	conn.AssertExpectations(t)
}

func TestWebSocketSubscriptionLimits_CombinedWithAuthentication(t *testing.T) {
	// Test subscription limits work correctly with authentication
	g := &Graphy{
		MemoryLimits: &MemoryLimits{
			MaxSubscriptionsPerConnection: 2,
		},
	}

	// Register test subscription
	testChan := make(chan string, 1)
	g.RegisterSubscription(context.Background(), "testSub", func(ctx context.Context) <-chan string {
		return testChan
	})

	auth := NewTestAuthenticator(true) // Require authentication
	handler := NewGraphQLWebSocketHandlerWithAuth(g, auth)

	conn := &MockAuthWebSocketConn{}
	conn.On("Close").Return(nil)
	conn.On("WriteMessage", mock.Anything).Return(nil)

	// Connection init with valid token
	initMsg := WebSocketMessage{
		Type:    GQLConnectionInit,
		Payload: json.RawMessage(`{"token": "valid-token"}`),
	}
	initMsgBytes, _ := json.Marshal(initMsg)

	// Three subscription attempts (should reject the third)
	messages := [][]byte{initMsgBytes}
	for i := 0; i < 3; i++ {
		subMsg := WebSocketMessage{
			ID:      fmt.Sprintf("sub-%d", i),
			Type:    GQLSubscribe,
			Payload: json.RawMessage(`{"query": "subscription { testSub }"}`),
		}
		subMsgBytes, _ := json.Marshal(subMsg)
		messages = append(messages, subMsgBytes)
	}

	// Setup mock calls
	for _, msg := range messages {
		conn.On("ReadMessage").Return(msg, nil).Once()
	}
	conn.On("ReadMessage").Return([]byte{}, fmt.Errorf("EOF")).Maybe()

	// Execute
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go handler.HandleConnection(ctx, conn)
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Verify results
	sentMessages := conn.GetMessages()

	// Should have connection ack + 1 subscription limit error
	var ackFound bool
	limitErrorCount := 0

	for _, msgBytes := range sentMessages {
		var msg WebSocketMessage
		err := json.Unmarshal(msgBytes, &msg)
		assert.NoError(t, err)

		if msg.Type == GQLConnectionAck {
			ackFound = true
		} else if msg.Type == GQLError &&
			msg.ID == "sub-2" && // Third subscription
			string(msg.Payload) != "" {
			limitErrorCount++
		}
	}

	assert.True(t, ackFound, "Should receive connection ack")
	assert.Equal(t, 1, limitErrorCount, "Should reject one subscription due to limit")

	conn.AssertExpectations(t)
}

func TestWebSocketSubscriptionLimits_EdgeCases(t *testing.T) {
	t.Run("Zero limit means unlimited", func(t *testing.T) {
		g := &Graphy{
			MemoryLimits: &MemoryLimits{
				MaxSubscriptionsPerConnection: 0, // Unlimited
			},
		}

		testChan := make(chan string, 1)
		g.RegisterSubscription(context.Background(), "testSub", func(ctx context.Context) <-chan string {
			return testChan
		})

		handler := NewGraphQLWebSocketHandler(g)
		conn := &MockAuthWebSocketConn{}
		conn.On("Close").Return(nil)
		conn.On("WriteMessage", mock.Anything).Return(nil)

		// Try many subscriptions
		initMsg := WebSocketMessage{Type: GQLConnectionInit, Payload: json.RawMessage(`{}`)}
		initMsgBytes, _ := json.Marshal(initMsg)

		messages := [][]byte{initMsgBytes}
		for i := 0; i < 10; i++ { // Many subscriptions
			subMsg := WebSocketMessage{
				ID:      fmt.Sprintf("sub-%d", i),
				Type:    GQLSubscribe,
				Payload: json.RawMessage(`{"query": "subscription { testSub }"}`),
			}
			subMsgBytes, _ := json.Marshal(subMsg)
			messages = append(messages, subMsgBytes)
		}

		for _, msg := range messages {
			conn.On("ReadMessage").Return(msg, nil).Once()
		}
		conn.On("ReadMessage").Return([]byte{}, fmt.Errorf("EOF")).Maybe()

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		go handler.HandleConnection(ctx, conn)
		time.Sleep(100 * time.Millisecond)
		cancel()

		// Should have no limit errors
		sentMessages := conn.GetMessages()
		errorCount := 0
		for _, msgBytes := range sentMessages {
			var msg WebSocketMessage
			err := json.Unmarshal(msgBytes, &msg)
			assert.NoError(t, err)

			if msg.Type == GQLError {
				errorCount++
			}
		}

		assert.Equal(t, 0, errorCount, "No subscriptions should be rejected with unlimited setting")

		conn.AssertExpectations(t)
	})

	t.Run("Negative limit means unlimited", func(t *testing.T) {
		g := &Graphy{
			MemoryLimits: &MemoryLimits{
				MaxSubscriptionsPerConnection: -1, // Should be treated as unlimited
			},
		}

		testChan := make(chan string, 1)
		g.RegisterSubscription(context.Background(), "testSub", func(ctx context.Context) <-chan string {
			return testChan
		})

		handler := NewGraphQLWebSocketHandler(g)
		conn := &MockAuthWebSocketConn{}
		conn.On("Close").Return(nil)
		conn.On("WriteMessage", mock.Anything).Return(nil)

		// Connection init + subscription
		initMsg := WebSocketMessage{Type: GQLConnectionInit, Payload: json.RawMessage(`{}`)}
		initMsgBytes, _ := json.Marshal(initMsg)

		subMsg := WebSocketMessage{
			ID:      "sub-1",
			Type:    GQLSubscribe,
			Payload: json.RawMessage(`{"query": "subscription { testSub }"}`),
		}
		subMsgBytes, _ := json.Marshal(subMsg)

		conn.On("ReadMessage").Return(initMsgBytes, nil).Once()
		conn.On("ReadMessage").Return(subMsgBytes, nil).Once()
		conn.On("ReadMessage").Return([]byte{}, fmt.Errorf("EOF")).Maybe()

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		go handler.HandleConnection(ctx, conn)
		time.Sleep(50 * time.Millisecond)
		cancel()

		// Should work normally
		sentMessages := conn.GetMessages()
		assert.True(t, len(sentMessages) >= 1)

		var ackMsg WebSocketMessage
		err := json.Unmarshal(sentMessages[0], &ackMsg)
		assert.NoError(t, err)
		assert.Equal(t, GQLConnectionAck, ackMsg.Type)

		conn.AssertExpectations(t)
	})
}

func TestWebSocketSubscriptionLimits_ErrorMessages(t *testing.T) {
	// Test that error messages contain useful information
	g := &Graphy{
		MemoryLimits: &MemoryLimits{
			MaxSubscriptionsPerConnection: 1,
		},
	}

	testChan := make(chan string, 1)
	g.RegisterSubscription(context.Background(), "testSub", func(ctx context.Context) <-chan string {
		return testChan
	})

	handler := NewGraphQLWebSocketHandler(g)
	conn := &MockAuthWebSocketConn{}
	conn.On("Close").Return(nil)
	conn.On("WriteMessage", mock.Anything).Return(nil)

	// Connection init + 2 subscriptions (second should be rejected)
	initMsg := WebSocketMessage{Type: GQLConnectionInit, Payload: json.RawMessage(`{}`)}
	initMsgBytes, _ := json.Marshal(initMsg)

	sub1Msg := WebSocketMessage{
		ID:      "sub-1",
		Type:    GQLSubscribe,
		Payload: json.RawMessage(`{"query": "subscription { testSub }"}`),
	}
	sub1MsgBytes, _ := json.Marshal(sub1Msg)

	sub2Msg := WebSocketMessage{
		ID:      "sub-2",
		Type:    GQLSubscribe,
		Payload: json.RawMessage(`{"query": "subscription { testSub }"}`),
	}
	sub2MsgBytes, _ := json.Marshal(sub2Msg)

	conn.On("ReadMessage").Return(initMsgBytes, nil).Once()
	conn.On("ReadMessage").Return(sub1MsgBytes, nil).Once()
	conn.On("ReadMessage").Return(sub2MsgBytes, nil).Once()
	conn.On("ReadMessage").Return([]byte{}, fmt.Errorf("EOF")).Maybe()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go handler.HandleConnection(ctx, conn)
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Find the error message for the rejected subscription
	sentMessages := conn.GetMessages()
	var errorMsg WebSocketMessage
	found := false

	for _, msgBytes := range sentMessages {
		var msg WebSocketMessage
		err := json.Unmarshal(msgBytes, &msg)
		require.NoError(t, err)

		if msg.Type == GQLError && msg.ID == "sub-2" {
			errorMsg = msg
			found = true
			break
		}
	}

	require.True(t, found, "Should find error message for rejected subscription")

	// Verify error message content
	assert.Equal(t, "sub-2", errorMsg.ID)
	assert.Contains(t, string(errorMsg.Payload), "maximum subscriptions per connection")
	assert.Contains(t, string(errorMsg.Payload), "1") // The limit value
	assert.Contains(t, string(errorMsg.Payload), "exceeded")

	conn.AssertExpectations(t)
}
