package quickgraph

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryLimits_HTTPRequestBodySize(t *testing.T) {
	tests := []struct {
		name           string
		maxBodySize    int64
		requestBody    string
		expectError    bool
		expectResponse bool
	}{
		{
			name:           "Small request under limit",
			maxBodySize:    1000,
			requestBody:    `{"query": "{ hello }"}`,
			expectError:    false,
			expectResponse: true,
		},
		{
			name:           "Request exactly at limit",
			maxBodySize:    100,
			requestBody:    strings.Repeat("x", 100), // Exactly 100 bytes
			expectError:    true,                     // Invalid JSON will cause error
			expectResponse: false,
		},
		{
			name:           "Request over limit",
			maxBodySize:    50,
			requestBody:    `{"query": "{ hello }", "variables": {"data": "` + strings.Repeat("x", 100) + `"}}`,
			expectError:    true,
			expectResponse: false,
		},
		{
			name:           "No limit configured (unlimited)",
			maxBodySize:    0,
			requestBody:    `{"query": "{ hello }", "variables": {"data": "` + strings.Repeat("x", 1000) + `"}}`,
			expectError:    false,
			expectResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create graphy with memory limits
			g := &Graphy{}
			if tt.maxBodySize > 0 {
				g.MemoryLimits = &MemoryLimits{
					MaxRequestBodySize: tt.maxBodySize,
				}
			}

			// Register a simple query for testing
			g.RegisterQuery(context.Background(), "hello", func(ctx context.Context) string {
				return "world"
			})

			// Create HTTP handler
			handler := g.HttpHandler()

			// Create request
			req := httptest.NewRequest("POST", "/graphql", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Record response
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			if tt.expectError {
				// Should either be a 400 (bad request) or return GraphQL error
				assert.True(t, w.Code == 400 || w.Code == 200, "Expected 400 or 200 status code")
				if w.Code == 200 {
					// Check if response contains error
					var resp map[string]interface{}
					err := json.Unmarshal(w.Body.Bytes(), &resp)
					assert.NoError(t, err)
					assert.Contains(t, resp, "errors", "Response should contain errors")
				}
			} else if tt.expectResponse {
				assert.Equal(t, 200, w.Code)
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Contains(t, resp, "data", "Response should contain data")
			}
		})
	}
}

func TestMemoryLimits_VariableJSONSize(t *testing.T) {
	tests := []struct {
		name            string
		maxVariableSize int64
		variables       string
		expectError     bool
		errorMessage    string
	}{
		{
			name:            "Small variables under limit",
			maxVariableSize: 1000,
			variables:       `{"input": "test"}`,
			expectError:     false,
		},
		{
			name:            "Variables exactly at limit",
			maxVariableSize: 30,
			variables:       `{"input":"` + strings.Repeat("a", 10) + `"}`, // Exactly around 30 bytes
			expectError:     false,
		},
		{
			name:            "Variables over limit",
			maxVariableSize: 50,
			variables:       `{"input":"` + strings.Repeat("x", 100) + `"}`,
			expectError:     true,
			errorMessage:    "variable payload size",
		},
		{
			name:            "No limit configured (unlimited)",
			maxVariableSize: 0,
			variables:       `{"input":"` + strings.Repeat("x", 1000) + `"}`,
			expectError:     false,
		},
		{
			name:            "Empty variables with query that doesn't use variables",
			maxVariableSize: 10,
			variables:       "",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create graphy with memory limits
			g := &Graphy{}
			if tt.maxVariableSize > 0 {
				g.MemoryLimits = &MemoryLimits{
					MaxVariableSize: tt.maxVariableSize,
				}
			}

			// Register a simple query that uses variables (make input optional with pointer)
			g.RegisterQuery(context.Background(), "echo", func(ctx context.Context, input *string) string {
				if input == nil {
					return "default"
				}
				return *input
			}, "input")

			// Register a simple query without variables
			g.RegisterQuery(context.Background(), "hello", func(ctx context.Context) string {
				return "world"
			})

			// Choose query based on whether we have variables
			var query string
			if tt.variables == "" {
				query = `{ hello }`
			} else {
				query = `query Echo($input: String) { echo(input: $input) }`
			}
			stub, err := g.newRequestStub(query)
			require.NoError(t, err)

			// Create request with variables
			req, err := stub.newRequest(context.Background(), tt.variables)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
				assert.Nil(t, req)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, req)
			}
		})
	}
}

func TestMemoryLimits_SubscriptionBuffering(t *testing.T) {
	tests := []struct {
		name               string
		subscriptionBuffer int
		messageCount       int
		expectBlocking     bool
	}{
		{
			name:               "Unbuffered channel (default)",
			subscriptionBuffer: 0,
			messageCount:       1,
			expectBlocking:     false, // Single message should work
		},
		{
			name:               "Buffered channel with capacity",
			subscriptionBuffer: 5,
			messageCount:       3,
			expectBlocking:     false, // Under capacity
		},
		{
			name:               "Buffered channel at capacity",
			subscriptionBuffer: 2,
			messageCount:       2,
			expectBlocking:     false, // Exactly at capacity
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create graphy with memory limits
			g := &Graphy{}
			if tt.subscriptionBuffer >= 0 {
				g.MemoryLimits = &MemoryLimits{
					SubscriptionBufferSize: tt.subscriptionBuffer,
				}
			}

			// Create a test subscription that emits messages
			messageChan := make(chan string, 10) // Large buffer for test setup
			g.RegisterSubscription(context.Background(), "testSub", func(ctx context.Context) <-chan string {
				return messageChan
			})

			// Create subscription request
			query := `subscription { testSub }`
			stub, err := g.newRequestStub(query)
			require.NoError(t, err)

			req, err := stub.newRequest(context.Background(), "")
			require.NoError(t, err)

			// Execute subscription
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			result, err := req.executeSubscription(ctx)
			require.NoError(t, err)
			require.NotNil(t, result)

			// Send test messages
			for i := 0; i < tt.messageCount; i++ {
				select {
				case messageChan <- "test message":
					// Message sent successfully
				case <-time.After(100 * time.Millisecond):
					if tt.expectBlocking {
						t.Log("Message sending blocked as expected")
						return
					}
					t.Fatal("Message sending unexpectedly blocked")
				}
			}

			// Close the source channel to end subscription
			close(messageChan)

			// Verify we can receive the messages
			receivedCount := 0
			for msg := range result.Messages {
				require.NotNil(t, msg.Data)
				receivedCount++
				if receivedCount >= tt.messageCount {
					break
				}
			}

			assert.Equal(t, tt.messageCount, receivedCount, "Should receive all sent messages")
		})
	}
}

func TestMemoryLimits_ProcessSubscriptionBuffering(t *testing.T) {
	// Test the JSON channel buffering in ProcessSubscription
	g := &Graphy{
		MemoryLimits: &MemoryLimits{
			SubscriptionBufferSize: 3,
		},
	}

	// Create a test subscription
	messageChan := make(chan string, 10)
	g.RegisterSubscription(context.Background(), "testSub", func(ctx context.Context) <-chan string {
		return messageChan
	})

	// Start subscription
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	query := `subscription { testSub }`
	jsonChan, err := g.ProcessSubscription(ctx, query, "")
	require.NoError(t, err)

	// Send messages and verify buffering
	testMessages := []string{"msg1", "msg2", "msg3"}

	// Send messages to source
	for _, msg := range testMessages {
		messageChan <- msg
	}
	close(messageChan)

	// Receive JSON messages
	var receivedMessages []string
	for jsonMsg := range jsonChan {
		receivedMessages = append(receivedMessages, jsonMsg)
	}

	assert.Equal(t, len(testMessages), len(receivedMessages), "Should receive all messages")

	// Verify JSON structure
	for i, jsonMsg := range receivedMessages {
		var resp map[string]interface{}
		err := json.Unmarshal([]byte(jsonMsg), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp, "data")

		data := resp["data"].(map[string]interface{})
		assert.Equal(t, testMessages[i], data["testSub"])
	}
}

func TestMemoryLimits_NoLimitsConfigured(t *testing.T) {
	// Test that everything works normally when no memory limits are configured
	g := &Graphy{} // No MemoryLimits set

	// Register a simple query
	g.RegisterQuery(context.Background(), "hello", func(ctx context.Context) string {
		return "world"
	})

	// Test HTTP request with large body (should work without limits)
	largeBody := `{"query": "{ hello }", "variables": {"data": "` + strings.Repeat("x", 10000) + `"}}`

	req := httptest.NewRequest("POST", "/graphql", bytes.NewBufferString(largeBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler := g.HttpHandler()
	handler.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, resp, "data")
}

func TestMemoryLimits_EdgeCases(t *testing.T) {
	t.Run("Zero byte limit", func(t *testing.T) {
		g := &Graphy{
			MemoryLimits: &MemoryLimits{
				MaxRequestBodySize: 0, // Unlimited
			},
		}

		g.RegisterQuery(context.Background(), "hello", func(ctx context.Context) string {
			return "world"
		})

		// Even with zero limit (unlimited), normal requests should work
		req := httptest.NewRequest("POST", "/graphql", bytes.NewBufferString(`{"query": "{ hello }"}`))
		w := httptest.NewRecorder()
		g.HttpHandler().ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})

	t.Run("Negative buffer size", func(t *testing.T) {
		g := &Graphy{
			MemoryLimits: &MemoryLimits{
				SubscriptionBufferSize: -1, // Should be treated as 0 (unbuffered)
			},
		}

		messageChan := make(chan string, 1)
		g.RegisterSubscription(context.Background(), "testSub", func(ctx context.Context) <-chan string {
			return messageChan
		})

		query := `subscription { testSub }`
		stub, err := g.newRequestStub(query)
		require.NoError(t, err)

		req, err := stub.newRequest(context.Background(), "")
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		result, err := req.executeSubscription(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// Helper function to create a limited reader for testing
func createLimitedReader(data string, limit int64) io.Reader {
	return io.LimitReader(strings.NewReader(data), limit)
}

func TestLimitReader_Behavior(t *testing.T) {
	// Test the io.LimitReader behavior we rely on
	data := "Hello, World!"

	t.Run("Under limit", func(t *testing.T) {
		limited := io.LimitReader(strings.NewReader(data), 20)
		result, err := io.ReadAll(limited)
		assert.NoError(t, err)
		assert.Equal(t, data, string(result))
	})

	t.Run("Over limit", func(t *testing.T) {
		limited := io.LimitReader(strings.NewReader(data), 5)
		result, err := io.ReadAll(limited)
		assert.NoError(t, err)
		assert.Equal(t, "Hello", string(result))
		assert.Len(t, result, 5)
	})

	t.Run("Exact limit", func(t *testing.T) {
		limited := io.LimitReader(strings.NewReader(data), int64(len(data)))
		result, err := io.ReadAll(limited)
		assert.NoError(t, err)
		assert.Equal(t, data, string(result))
	})
}
