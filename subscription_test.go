package quickgraph

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Message represents a chat message
type Message struct {
	ID        string
	Text      string
	User      string
	Timestamp time.Time
}

// TestSubscriptionBasic tests basic subscription functionality
func TestSubscriptionBasic(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Register a simple subscription that emits messages
	g.RegisterSubscription(ctx, "messageAdded", func(ctx context.Context, roomID string) (<-chan Message, error) {
		ch := make(chan Message, 10)

		// Simulate messages being added
		go func() {
			defer close(ch)
			for i := 0; i < 3; i++ {
				select {
				case <-ctx.Done():
					return
				case ch <- Message{
					ID:        fmt.Sprintf("msg-%d", i+1),
					Text:      fmt.Sprintf("Hello from room %s, message %d", roomID, i+1),
					User:      "testuser",
					Timestamp: time.Now(),
				}:
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()

		return ch, nil
	})

	// Test subscription execution
	query := `subscription { messageAdded(roomID: "general") { id text user } }`
	msgChan, err := g.ProcessSubscription(ctx, query, "")
	assert.NoError(t, err)

	// Collect messages
	var messages []string
	for msg := range msgChan {
		messages = append(messages, msg)
	}

	// Verify we got 3 messages
	assert.Len(t, messages, 3)
}

// TestSubscriptionWithVariables tests subscriptions with variables
func TestSubscriptionWithVariables(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Track subscription calls
	subscriptionCalls := 0

	g.RegisterSubscription(ctx, "userStatusChanged", func(ctx context.Context, userID string) (<-chan UserStatus, error) {
		subscriptionCalls++
		ch := make(chan UserStatus, 1)

		go func() {
			defer close(ch)
			// Emit one status change
			select {
			case <-ctx.Done():
				return
			case ch <- UserStatus{
				UserID: userID,
				Status: "online",
			}:
			}
		}()

		return ch, nil
	})

	// Test with variables
	query := `subscription StatusSub($user: String!) { userStatusChanged(userID: $user) { userID status } }`
	variables := `{"user": "user123"}`

	msgChan, err := g.ProcessSubscription(ctx, query, variables)
	assert.NoError(t, err)

	// Get first message
	msg := <-msgChan
	assert.Contains(t, msg, `"userID":"user123"`)
	assert.Contains(t, msg, `"status":"online"`)

	// Verify subscription was called once
	assert.Equal(t, 1, subscriptionCalls)
}

// UserStatus represents a user's online status
type UserStatus struct {
	UserID string
	Status string
}

// TestSubscriptionContextCancellation tests that subscriptions respect context cancellation
func TestSubscriptionContextCancellation(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	g.RegisterSubscription(ctx, "infiniteCounter", func(ctx context.Context) (<-chan int, error) {
		ch := make(chan int)

		go func() {
			defer close(ch)
			counter := 0
			for {
				select {
				case <-ctx.Done():
					return
				case ch <- counter:
					counter++
					time.Sleep(10 * time.Millisecond)
				}
			}
		}()

		return ch, nil
	})

	// Create cancellable context
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	query := `subscription { infiniteCounter }`
	msgChan, err := g.ProcessSubscription(subCtx, query, "")
	assert.NoError(t, err)

	// Collect a few messages
	messageCount := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	for range msgChan {
		messageCount++
	}

	// Should have received some messages but not infinite
	assert.Greater(t, messageCount, 0)
	assert.Less(t, messageCount, 10)
}

// TestSubscriptionError tests error handling in subscriptions
func TestSubscriptionError(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Register a subscription that returns an error
	g.RegisterSubscription(ctx, "errorSubscription", func(ctx context.Context) (<-chan string, error) {
		return nil, fmt.Errorf("subscription setup failed")
	})

	query := `subscription { errorSubscription }`
	_, err := g.ProcessSubscription(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subscription setup failed")
}

// TestSubscriptionSchema tests that subscriptions appear in the schema
func TestSubscriptionSchema(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	// Enable introspection
	g.EnableIntrospection(ctx)

	// Register a subscription with named parameter
	g.RegisterSubscription(ctx, "orderUpdated", func(ctx context.Context, orderID string) (<-chan Order, error) {
		ch := make(chan Order)
		close(ch) // Just for schema test
		return ch, nil
	}, "orderID")

	// Get schema
	schema := g.SchemaDefinition(ctx)

	// Verify subscription type is in schema
	assert.Contains(t, schema, "type Subscription {")
	assert.Contains(t, schema, "orderUpdated(orderID: String!): Order")
}

// Order represents an order for schema testing
type Order struct {
	ID     string
	Status string
	Total  float64
}

// TestMultipleSubscriptionsNotAllowed tests that only one subscription per request is allowed
func TestMultipleSubscriptionsNotAllowed(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	g.RegisterSubscription(ctx, "sub1", func(ctx context.Context) (<-chan string, error) {
		ch := make(chan string)
		close(ch)
		return ch, nil
	})

	g.RegisterSubscription(ctx, "sub2", func(ctx context.Context) (<-chan string, error) {
		ch := make(chan string)
		close(ch)
		return ch, nil
	})

	// Try to execute multiple subscriptions
	query := `subscription { sub1 sub2 }`
	_, err := g.getRequestStub(ctx, query)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subscriptions can only have one root field")
}

// TestSubscriptionNotAllowedInQuery tests that subscriptions can't be used in queries
func TestSubscriptionNotAllowedInQuery(t *testing.T) {
	g := Graphy{}
	ctx := context.Background()

	g.RegisterSubscription(ctx, "updates", func(ctx context.Context) (<-chan string, error) {
		ch := make(chan string)
		close(ch)
		return ch, nil
	})

	// Try to use subscription in a query
	query := `query { updates }`
	_, err := g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subscription updates used in query")
}
