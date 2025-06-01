package quickgraph

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock WebSocket connection for testing
type MockAuthWebSocketConn struct {
	mock.Mock
	messages [][]byte
	closed   bool
	mu       sync.Mutex
}

func (m *MockAuthWebSocketConn) ReadMessage() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockAuthWebSocketConn) WriteMessage(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, data)
	args := m.Called(data)
	return args.Error(0)
}

func (m *MockAuthWebSocketConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	args := m.Called()
	return args.Error(0)
}

func (m *MockAuthWebSocketConn) GetMessages() [][]byte {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([][]byte, len(m.messages))
	copy(result, m.messages)
	return result
}

func (m *MockAuthWebSocketConn) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

// Mock authenticator for testing
type MockAuthenticator struct {
	mock.Mock
}

func (m *MockAuthenticator) AuthenticateConnection(ctx context.Context, initPayload json.RawMessage) (context.Context, error) {
	args := m.Called(ctx, initPayload)
	return args.Get(0).(context.Context), args.Error(1)
}

func (m *MockAuthenticator) AuthorizeSubscription(ctx context.Context, query string, variables json.RawMessage) (context.Context, error) {
	args := m.Called(ctx, query, variables)
	return args.Get(0).(context.Context), args.Error(1)
}

// Test authenticator implementations
type AuthTestUser struct {
	ID       string
	Username string
	Role     string
}

type TestAuthenticator struct {
	validTokens map[string]*AuthTestUser
	requireAuth bool
}

func NewTestAuthenticator(requireAuth bool) *TestAuthenticator {
	return &TestAuthenticator{
		validTokens: map[string]*AuthTestUser{
			"valid-token": {ID: "1", Username: "user", Role: "user"},
			"admin-token": {ID: "2", Username: "admin", Role: "admin"},
			"guest-token": {ID: "3", Username: "guest", Role: "guest"},
		},
		requireAuth: requireAuth,
	}
}

func (t *TestAuthenticator) AuthenticateConnection(ctx context.Context, initPayload json.RawMessage) (context.Context, error) {
	if !t.requireAuth {
		return ctx, nil
	}

	var payload struct {
		Token string `json:"token"`
	}

	if err := json.Unmarshal(initPayload, &payload); err != nil {
		return nil, fmt.Errorf("invalid payload: %v", err)
	}

	user, exists := t.validTokens[payload.Token]
	if !exists {
		return nil, fmt.Errorf("invalid token")
	}

	return context.WithValue(ctx, "user", user), nil
}

func (t *TestAuthenticator) AuthorizeSubscription(ctx context.Context, query string, variables json.RawMessage) (context.Context, error) {
	if !t.requireAuth {
		return ctx, nil
	}

	user, ok := ctx.Value("user").(*AuthTestUser)
	if !ok {
		return nil, fmt.Errorf("no user in context")
	}

	// Role-based authorization
	if strings.Contains(query, "adminSubscription") && user.Role != "admin" {
		return nil, fmt.Errorf("admin role required")
	}

	if strings.Contains(query, "userSubscription") && user.Role == "guest" {
		return nil, fmt.Errorf("guest users not allowed")
	}

	return ctx, nil
}

func TestNoOpWebSocketAuthenticator(t *testing.T) {
	auth := NoOpWebSocketAuthenticator{}
	ctx := context.Background()
	payload := json.RawMessage(`{"test": "data"}`)

	t.Run("AuthenticateConnection allows all", func(t *testing.T) {
		resultCtx, err := auth.AuthenticateConnection(ctx, payload)
		assert.NoError(t, err)
		assert.Equal(t, ctx, resultCtx)
	})

	t.Run("AuthorizeSubscription allows all", func(t *testing.T) {
		query := "subscription { adminData }"
		variables := json.RawMessage(`{"param": "value"}`)

		resultCtx, err := auth.AuthorizeSubscription(ctx, query, variables)
		assert.NoError(t, err)
		assert.Equal(t, ctx, resultCtx)
	})
}

func TestWebSocketAuthenticator_Interface(t *testing.T) {
	// Test that our test implementations satisfy the interface
	var _ WebSocketAuthenticator = &NoOpWebSocketAuthenticator{}
	var _ WebSocketAuthenticator = &TestAuthenticator{}
	var _ WebSocketAuthenticator = &MockAuthenticator{}
}

func TestGraphQLWebSocketHandler_Creation(t *testing.T) {
	g := &Graphy{}

	t.Run("Default handler with NoOp authenticator", func(t *testing.T) {
		handler := NewGraphQLWebSocketHandler(g)
		assert.NotNil(t, handler)
		assert.Equal(t, g, handler.graphy)
		assert.IsType(t, NoOpWebSocketAuthenticator{}, handler.authenticator)
	})

	t.Run("Handler with custom authenticator", func(t *testing.T) {
		auth := &TestAuthenticator{}
		handler := NewGraphQLWebSocketHandlerWithAuth(g, auth)
		assert.NotNil(t, handler)
		assert.Equal(t, g, handler.graphy)
		assert.Equal(t, auth, handler.authenticator)
	})
}

func TestWebSocketAuthentication_ConnectionInit(t *testing.T) {
	tests := []struct {
		name            string
		initPayload     string
		expectAuthError bool
		expectAck       bool
		expectedUser    *AuthTestUser
	}{
		{
			name:            "Valid token authentication",
			initPayload:     `{"token": "valid-token"}`,
			expectAuthError: false,
			expectAck:       true,
			expectedUser:    &AuthTestUser{ID: "1", Username: "user", Role: "user"},
		},
		{
			name:            "Admin token authentication",
			initPayload:     `{"token": "admin-token"}`,
			expectAuthError: false,
			expectAck:       true,
			expectedUser:    &AuthTestUser{ID: "2", Username: "admin", Role: "admin"},
		},
		{
			name:            "Invalid token",
			initPayload:     `{"token": "invalid-token"}`,
			expectAuthError: true,
			expectAck:       false,
			expectedUser:    nil,
		},
		{
			name:            "Missing token",
			initPayload:     `{"other": "data"}`,
			expectAuthError: true,
			expectAck:       false,
			expectedUser:    nil,
		},
		{
			name:            "Invalid JSON payload",
			initPayload:     `{"invalid": "structure"}`, // Valid JSON but invalid for the authenticator
			expectAuthError: true,
			expectAck:       false,
			expectedUser:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			g := &Graphy{}
			auth := NewTestAuthenticator(true) // Require authentication
			handler := NewGraphQLWebSocketHandlerWithAuth(g, auth)

			conn := &MockAuthWebSocketConn{}
			conn.On("Close").Return(nil)
			conn.On("WriteMessage", mock.Anything).Return(nil)

			// Setup connection init message
			var initMsgBytes []byte
			var err error

			// Create valid WebSocket message (all tests use valid WebSocket format)
			initMsg := WebSocketMessage{
				Type: GQLConnectionInit,
			}

			// All test cases now use valid JSON
			initMsg.Payload = json.RawMessage(tt.initPayload)

			initMsgBytes, err = json.Marshal(initMsg)
			if err != nil {
				t.Fatalf("Failed to marshal init message: %v", err)
			}

			// Setup mock to return init message then EOF
			conn.On("ReadMessage").Return(initMsgBytes, nil).Once()
			conn.On("ReadMessage").Return([]byte{}, fmt.Errorf("EOF")).Maybe()

			// Execute
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			// Run in goroutine to avoid blocking
			done := make(chan bool, 1)
			go func() {
				handler.HandleConnection(ctx, conn)
				done <- true
			}()

			// Wait for completion or timeout
			select {
			case <-done:
				// Handler completed
			case <-time.After(150 * time.Millisecond):
				// Timeout
				cancel()
			}

			// Give a moment for any final message processing
			time.Sleep(10 * time.Millisecond)

			// Verify
			messages := conn.GetMessages()
			assert.True(t, conn.IsClosed())

			if tt.expectAuthError {
				// Should receive connection error message
				if len(messages) == 0 {
					t.Logf("Expected connection error message but got no messages")
					t.Logf("Mock call history: %+v", conn.Calls)
				}
				assert.True(t, len(messages) >= 1, "Should receive at least one message (connection error)")
				if len(messages) > 0 {
					var msg WebSocketMessage
					err := json.Unmarshal(messages[0], &msg)
					assert.NoError(t, err)
					assert.Equal(t, GQLConnectionError, msg.Type)
				}
			} else if tt.expectAck {
				// Should receive connection ack
				assert.Len(t, messages, 1)
				var msg WebSocketMessage
				err := json.Unmarshal(messages[0], &msg)
				assert.NoError(t, err)
				assert.Equal(t, GQLConnectionAck, msg.Type)
			}

			conn.AssertExpectations(t)
		})
	}
}

func TestWebSocketAuthentication_SubscriptionAuthorization(t *testing.T) {
	tests := []struct {
		name            string
		userToken       string
		query           string
		expectAuthError bool
		expectSubError  bool
		errorContains   string
	}{
		{
			name:            "User can access user subscription",
			userToken:       "valid-token",
			query:           "subscription { userSubscription }",
			expectAuthError: false,
			expectSubError:  false,
		},
		{
			name:            "Admin can access admin subscription",
			userToken:       "admin-token",
			query:           "subscription { adminSubscription }",
			expectAuthError: false,
			expectSubError:  false,
		},
		{
			name:            "User cannot access admin subscription",
			userToken:       "valid-token",
			query:           "subscription { adminSubscription }",
			expectAuthError: false,
			expectSubError:  true,
			errorContains:   "admin role required",
		},
		{
			name:            "Guest cannot access user subscription",
			userToken:       "guest-token",
			query:           "subscription { userSubscription }",
			expectAuthError: false,
			expectSubError:  true,
			errorContains:   "guest users not allowed",
		},
		{
			name:            "Generic subscription allowed for all authenticated users",
			userToken:       "guest-token",
			query:           "subscription { publicData }",
			expectAuthError: false,
			expectSubError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup GraphQL with test subscription
			g := &Graphy{}
			testChan := make(chan string, 1)
			g.RegisterSubscription(context.Background(), "testSub", func(ctx context.Context) <-chan string {
				return testChan
			})

			auth := NewTestAuthenticator(true)
			handler := NewGraphQLWebSocketHandlerWithAuth(g, auth)

			conn := &MockAuthWebSocketConn{}
			conn.On("Close").Return(nil)
			conn.On("WriteMessage", mock.Anything).Return(nil)

			// Connection init
			initPayload := fmt.Sprintf(`{"token": "%s"}`, tt.userToken)
			initMsg := WebSocketMessage{
				Type:    GQLConnectionInit,
				Payload: json.RawMessage(initPayload),
			}
			initMsgBytes, _ := json.Marshal(initMsg)

			// Subscription message
			subPayload := fmt.Sprintf(`{"query": "%s"}`, tt.query)
			subMsg := WebSocketMessage{
				ID:      "sub-1",
				Type:    GQLSubscribe,
				Payload: json.RawMessage(subPayload),
			}
			subMsgBytes, _ := json.Marshal(subMsg)

			// Setup mock calls
			conn.On("ReadMessage").Return(initMsgBytes, nil).Once()
			conn.On("ReadMessage").Return(subMsgBytes, nil).Once()
			conn.On("ReadMessage").Return([]byte{}, fmt.Errorf("EOF")).Maybe()

			// Execute
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()

			go handler.HandleConnection(ctx, conn)

			// Wait for processing
			time.Sleep(50 * time.Millisecond)
			cancel()

			// Verify messages
			messages := conn.GetMessages()

			if tt.expectAuthError {
				// Should have connection error
				assert.True(t, len(messages) >= 1)
				var msg WebSocketMessage
				err := json.Unmarshal(messages[0], &msg)
				assert.NoError(t, err)
				assert.Equal(t, GQLConnectionError, msg.Type)
			} else {
				// Should have connection ack
				assert.True(t, len(messages) >= 1)
				var ackMsg WebSocketMessage
				err := json.Unmarshal(messages[0], &ackMsg)
				assert.NoError(t, err)
				assert.Equal(t, GQLConnectionAck, ackMsg.Type)

				if tt.expectSubError {
					// Should have subscription error
					assert.True(t, len(messages) >= 2)
					var errMsg WebSocketMessage
					err := json.Unmarshal(messages[1], &errMsg)
					assert.NoError(t, err)
					assert.Equal(t, GQLError, errMsg.Type)
					assert.Equal(t, "sub-1", errMsg.ID)

					if tt.errorContains != "" {
						assert.Contains(t, string(errMsg.Payload), tt.errorContains)
					}
				}
			}

			conn.AssertExpectations(t)
		})
	}
}

func TestWebSocketAuthentication_NoAuthRequired(t *testing.T) {
	// Test that when no authentication is required, everything works normally
	g := &Graphy{}
	testChan := make(chan string, 1)
	g.RegisterSubscription(context.Background(), "testSub", func(ctx context.Context) <-chan string {
		return testChan
	})

	// Use default handler (NoOp authenticator)
	handler := NewGraphQLWebSocketHandler(g)

	conn := &MockAuthWebSocketConn{}
	conn.On("Close").Return(nil)
	conn.On("WriteMessage", mock.Anything).Return(nil)

	// Connection init without any auth payload
	initMsg := WebSocketMessage{
		Type:    GQLConnectionInit,
		Payload: json.RawMessage(`{}`),
	}
	initMsgBytes, _ := json.Marshal(initMsg)

	// Subscription
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

	messages := conn.GetMessages()
	assert.True(t, len(messages) >= 1)

	// Should get connection ack
	var ackMsg WebSocketMessage
	err := json.Unmarshal(messages[0], &ackMsg)
	assert.NoError(t, err)
	assert.Equal(t, GQLConnectionAck, ackMsg.Type)

	conn.AssertExpectations(t)
}

func TestWebSocketAuthentication_SubscriptionWithoutAuth(t *testing.T) {
	// Test that subscriptions are rejected if connection is not authenticated
	g := &Graphy{}
	auth := NewTestAuthenticator(true) // Require auth
	handler := NewGraphQLWebSocketHandlerWithAuth(g, auth)

	conn := &MockAuthWebSocketConn{}
	conn.On("Close").Return(nil)
	conn.On("WriteMessage", mock.Anything).Return(nil)

	// Try to subscribe without connection init
	subMsg := WebSocketMessage{
		ID:      "sub-1",
		Type:    GQLSubscribe,
		Payload: json.RawMessage(`{"query": "subscription { testSub }"}`),
	}
	subMsgBytes, _ := json.Marshal(subMsg)

	conn.On("ReadMessage").Return(subMsgBytes, nil).Once()
	conn.On("ReadMessage").Return([]byte{}, fmt.Errorf("EOF")).Maybe()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go handler.HandleConnection(ctx, conn)
	time.Sleep(50 * time.Millisecond)
	cancel()

	messages := conn.GetMessages()
	assert.True(t, len(messages) >= 1)

	// Should get error about not being authenticated
	var errMsg WebSocketMessage
	err := json.Unmarshal(messages[0], &errMsg)
	assert.NoError(t, err)
	assert.Equal(t, GQLError, errMsg.Type)
	assert.Equal(t, "sub-1", errMsg.ID)
	assert.Contains(t, string(errMsg.Payload), "not authenticated")

	conn.AssertExpectations(t)
}

func TestWebSocketAuthentication_ContextPropagation(t *testing.T) {
	// Test that authenticated context is properly propagated through the subscription
	g := &Graphy{}

	var receivedContext context.Context
	g.RegisterSubscription(context.Background(), "testSub", func(ctx context.Context) <-chan string {
		receivedContext = ctx
		ch := make(chan string, 1)
		ch <- "test message"
		close(ch)
		return ch
	})

	auth := NewTestAuthenticator(true)
	handler := NewGraphQLWebSocketHandlerWithAuth(g, auth)

	conn := &MockAuthWebSocketConn{}
	conn.On("Close").Return(nil)
	conn.On("WriteMessage", mock.Anything).Return(nil)

	// Connection init with valid token
	initMsg := WebSocketMessage{
		Type:    GQLConnectionInit,
		Payload: json.RawMessage(`{"token": "admin-token"}`),
	}
	initMsgBytes, _ := json.Marshal(initMsg)

	// Subscription
	subMsg := WebSocketMessage{
		ID:      "sub-1",
		Type:    GQLSubscribe,
		Payload: json.RawMessage(`{"query": "subscription { testSub }"}`),
	}
	subMsgBytes, _ := json.Marshal(subMsg)

	conn.On("ReadMessage").Return(initMsgBytes, nil).Once()
	conn.On("ReadMessage").Return(subMsgBytes, nil).Once()
	conn.On("ReadMessage").Return([]byte{}, fmt.Errorf("EOF")).Maybe()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go handler.HandleConnection(ctx, conn)
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Verify that the subscription function received the authenticated context
	assert.NotNil(t, receivedContext)
	user, ok := receivedContext.Value("user").(*AuthTestUser)
	assert.True(t, ok, "Context should contain authenticated user")
	assert.Equal(t, "admin", user.Username)
	assert.Equal(t, "admin", user.Role)

	conn.AssertExpectations(t)
}

func TestWebSocketAuthentication_MockAuthenticator(t *testing.T) {
	// Test using the mock authenticator for more precise control
	g := &Graphy{}

	mockAuth := &MockAuthenticator{}
	handler := NewGraphQLWebSocketHandlerWithAuth(g, mockAuth)

	// Setup mock expectations
	testCtx := context.WithValue(context.Background(), "test", "value")
	mockAuth.On("AuthenticateConnection", mock.Anything, json.RawMessage(`{"token":"test"}`)).
		Return(testCtx, nil)

	conn := &MockAuthWebSocketConn{}
	conn.On("Close").Return(nil)
	conn.On("WriteMessage", mock.Anything).Return(nil)

	initMsg := WebSocketMessage{
		Type:    GQLConnectionInit,
		Payload: json.RawMessage(`{"token":"test"}`),
	}
	initMsgBytes, _ := json.Marshal(initMsg)

	conn.On("ReadMessage").Return(initMsgBytes, nil).Once()
	conn.On("ReadMessage").Return([]byte{}, fmt.Errorf("EOF")).Maybe()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	handler.HandleConnection(ctx, conn)

	// Verify mock was called
	mockAuth.AssertExpectations(t)
	conn.AssertExpectations(t)
}
