# Real-time Subscriptions

GraphQL subscriptions enable real-time communication between your server and clients. go-quickgraph uses Go channels to provide a natural, type-safe way to stream data updates over WebSockets.

## Quick Start

Here's a simple subscription that sends the current time every second:

```go
type TimeUpdate struct {
    Timestamp time.Time `json:"timestamp"`
    Formatted string    `json:"formatted"`
}

func CurrentTime(ctx context.Context, intervalMs int) (<-chan TimeUpdate, error) {
    if intervalMs < 100 {
        return nil, fmt.Errorf("interval must be at least 100ms")
    }
    
    updates := make(chan TimeUpdate)
    
    go func() {
        defer close(updates)
        ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
        defer ticker.Stop()
        
        for {
            select {
            case <-ctx.Done():
                return // Client disconnected
            case now := <-ticker.C:
                updates <- TimeUpdate{
                    Timestamp: now,
                    Formatted: now.Format("2006-01-02 15:04:05"),
                }
            }
        }
    }()
    
    return updates, nil
}

// Register the subscription
g.RegisterSubscription(ctx, "currentTime", CurrentTime, "intervalMs")
```

**Client Usage:**
```graphql
subscription {
  currentTime(intervalMs: 1000) {
    timestamp
    formatted
  }
}
```

## How Subscriptions Work

### Channel-Based Streaming

Subscription functions return Go channels that stream data:

```go
// Function signature options:
func MySubscription(ctx context.Context, params...) <-chan DataType
func MySubscription(ctx context.Context, params...) (<-chan DataType, error)
```

### WebSocket Transport

Subscriptions use WebSockets with the `graphql-ws` protocol:

1. Client connects to `ws://localhost:8080/graphql`
2. Protocol handshake (`connection_init`, `connection_ack`)
3. Client sends subscription with `start` message
4. Server streams `data` messages
5. Either party can send `stop`/`complete`

## Setting Up WebSocket Support

### HTTP Server with WebSocket Upgrade

```go
import (
    "github.com/gorilla/websocket"
    "github.com/gburgyan/go-quickgraph"
)

// Create WebSocket upgrader
type GorillaWebSocketUpgrader struct {
    upgrader websocket.Upgrader
}

func (u *GorillaWebSocketUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (quickgraph.SimpleWebSocketConn, error) {
    conn, err := u.upgrader.Upgrade(w, r, nil)
    if err != nil {
        return nil, err
    }
    return &GorillaWebSocketAdapter{conn: conn}, nil
}

// Adapter for gorilla/websocket
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

// Use in your server
func main() {
    g := &quickgraph.Graphy{}
    
    // Register subscriptions
    g.RegisterSubscription(ctx, "currentTime", CurrentTime, "intervalMs")
    
    // Create WebSocket-enabled handler
    upgrader := &GorillaWebSocketUpgrader{
        upgrader: websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool { return true },
        },
    }
    
    handler := g.HttpHandlerWithWebSocket(upgrader)
    http.Handle("/graphql", handler)
    
    log.Println("Server running at http://localhost:8080/graphql")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Subscription Patterns

### Data Change Notifications

Stream updates when data changes:

```go
type ProductUpdate struct {
    Product   Product    `json:"product"`
    Action    string     `json:"action"` // "created", "updated", "deleted"
    Timestamp time.Time  `json:"timestamp"`
}

var productUpdates = make(chan ProductUpdate, 100) // Buffered channel

func ProductUpdates(ctx context.Context) <-chan ProductUpdate {
    updates := make(chan ProductUpdate)
    
    go func() {
        defer close(updates)
        for {
            select {
            case <-ctx.Done():
                return
            case update := <-productUpdates:
                updates <- update
            }
        }
    }()
    
    return updates
}

// Trigger updates from mutations
func CreateProduct(ctx context.Context, input ProductInput) (*Product, error) {
    product := &Product{
        ID:    generateID(),
        Name:  input.Name,
        Price: input.Price,
    }
    
    // Save to database
    err := database.SaveProduct(product)
    if err != nil {
        return nil, err
    }
    
    // Broadcast update to subscribers
    select {
    case productUpdates <- ProductUpdate{
        Product:   *product,
        Action:    "created",
        Timestamp: time.Now(),
    }:
    default:
        // Channel full, skip this update
    }
    
    return product, nil
}
```

### Filtered Subscriptions

Allow clients to filter subscription data:

```go
func ProductUpdatesByCategory(ctx context.Context, categoryID int) (<-chan ProductUpdate, error) {
    if categoryID <= 0 {
        return nil, fmt.Errorf("invalid category ID")
    }
    
    updates := make(chan ProductUpdate)
    
    go func() {
        defer close(updates)
        for {
            select {
            case <-ctx.Done():
                return
            case update := <-productUpdates:
                // Only send updates for the requested category
                if update.Product.CategoryID == categoryID {
                    updates <- update
                }
            }
        }
    }()
    
    return updates, nil
}

g.RegisterSubscription(ctx, "productUpdatesByCategory", ProductUpdatesByCategory, "categoryID")
```

**Client Usage:**
```graphql
subscription {
  productUpdatesByCategory(categoryID: 1) {
    product {
      id
      name
      price
    }
    action
    timestamp
  }
}
```

### Order Status Tracking

Track the progress of long-running operations:

```go
type OrderStatus string

const (
    OrderStatusPending    OrderStatus = "PENDING"
    OrderStatusProcessing OrderStatus = "PROCESSING"
    OrderStatusShipped    OrderStatus = "SHIPPED"
    OrderStatusDelivered  OrderStatus = "DELIVERED"
    OrderStatusCancelled  OrderStatus = "CANCELLED"
)

type OrderUpdate struct {
    OrderID   string      `json:"orderID"`
    Status    OrderStatus `json:"status"`
    Message   string      `json:"message"`
    Timestamp time.Time   `json:"timestamp"`
}

var orderChannels = make(map[string][]chan OrderUpdate)
var orderChannelsMutex sync.RWMutex

func OrderStatusUpdates(ctx context.Context, orderID string) (<-chan OrderUpdate, error) {
    if orderID == "" {
        return nil, fmt.Errorf("order ID is required")
    }
    
    updates := make(chan OrderUpdate, 10)
    
    // Register this channel for order updates
    orderChannelsMutex.Lock()
    orderChannels[orderID] = append(orderChannels[orderID], updates)
    orderChannelsMutex.Unlock()
    
    // Clean up when subscription ends
    go func() {
        <-ctx.Done()
        orderChannelsMutex.Lock()
        defer orderChannelsMutex.Unlock()
        
        // Remove this channel from the list
        channels := orderChannels[orderID]
        for i, ch := range channels {
            if ch == updates {
                orderChannels[orderID] = append(channels[:i], channels[i+1:]...)
                break
            }
        }
        close(updates)
    }()
    
    return updates, nil
}

// Function to broadcast order updates
func BroadcastOrderUpdate(orderID string, status OrderStatus, message string) {
    update := OrderUpdate{
        OrderID:   orderID,
        Status:    status,
        Message:   message,
        Timestamp: time.Now(),
    }
    
    orderChannelsMutex.RLock()
    channels := orderChannels[orderID]
    orderChannelsMutex.RUnlock()
    
    for _, ch := range channels {
        select {
        case ch <- update:
        default:
            // Channel full or closed, skip
        }
    }
}

// Use in order processing
func ProcessOrder(orderID string) {
    BroadcastOrderUpdate(orderID, OrderStatusProcessing, "Order is being processed")
    
    // Do processing...
    time.Sleep(2 * time.Second)
    
    BroadcastOrderUpdate(orderID, OrderStatusShipped, "Order has been shipped")
    
    // More processing...
    time.Sleep(5 * time.Second)
    
    BroadcastOrderUpdate(orderID, OrderStatusDelivered, "Order has been delivered")
}
```

## Advanced Patterns

### Subscription with Authentication

Check authentication in subscription functions:

```go
type UserUpdate struct {
    User      User      `json:"user"`
    Action    string    `json:"action"`
    Timestamp time.Time `json:"timestamp"`
}

func UserUpdates(ctx context.Context) (<-chan UserUpdate, error) {
    // Check authentication
    userID, ok := ctx.Value("userID").(string)
    if !ok {
        return nil, fmt.Errorf("authentication required")
    }
    
    // Check authorization (admin only)
    user, err := getUser(userID)
    if err != nil || user.Role != "admin" {
        return nil, fmt.Errorf("admin access required")
    }
    
    updates := make(chan UserUpdate)
    
    go func() {
        defer close(updates)
        for {
            select {
            case <-ctx.Done():
                return
            case update := <-globalUserUpdates:
                updates <- update
            }
        }
    }()
    
    return updates, nil
}
```

### Subscription Middleware

Create reusable subscription patterns:

```go
// Generic filtered subscription helper
func FilteredSubscription[T any](
    ctx context.Context,
    source <-chan T,
    filter func(T) bool,
) <-chan T {
    filtered := make(chan T)
    
    go func() {
        defer close(filtered)
        for {
            select {
            case <-ctx.Done():
                return
            case item, ok := <-source:
                if !ok {
                    return // Source closed
                }
                if filter(item) {
                    filtered <- item
                }
            }
        }
    }()
    
    return filtered
}

// Use the helper
func ProductUpdatesForUser(ctx context.Context, userID string) (<-chan ProductUpdate, error) {
    user, err := getUser(userID)
    if err != nil {
        return nil, err
    }
    
    return FilteredSubscription(ctx, productUpdates, func(update ProductUpdate) bool {
        // Only show products the user can see
        return canUserSeeProduct(user, update.Product)
    }), nil
}
```

### Subscription Batching

Batch multiple updates to reduce client load:

```go
type BatchedUpdate struct {
    Updates   []ProductUpdate `json:"updates"`
    Count     int             `json:"count"`
    Timestamp time.Time       `json:"timestamp"`
}

func BatchedProductUpdates(ctx context.Context, batchSize int, intervalMs int) (<-chan BatchedUpdate, error) {
    if batchSize <= 0 {
        batchSize = 10
    }
    if intervalMs < 100 {
        intervalMs = 1000
    }
    
    batched := make(chan BatchedUpdate)
    
    go func() {
        defer close(batched)
        
        var batch []ProductUpdate
        ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
        defer ticker.Stop()
        
        flushBatch := func() {
            if len(batch) > 0 {
                batched <- BatchedUpdate{
                    Updates:   batch,
                    Count:     len(batch),
                    Timestamp: time.Now(),
                }
                batch = nil
            }
        }
        
        for {
            select {
            case <-ctx.Done():
                flushBatch()
                return
            case <-ticker.C:
                flushBatch()
            case update := <-productUpdates:
                batch = append(batch, update)
                if len(batch) >= batchSize {
                    flushBatch()
                }
            }
        }
    }()
    
    return batched, nil
}
```

## Client Integration

### JavaScript Client

```javascript
import { createClient } from 'graphql-ws';

const client = createClient({
  url: 'ws://localhost:8080/graphql',
});

// Subscribe to product updates
const unsubscribe = client.subscribe(
  {
    query: `
      subscription OnProductUpdates($categoryID: Int!) {
        productUpdatesByCategory(categoryID: $categoryID) {
          product {
            id
            name
            price
          }
          action
          timestamp
        }
      }
    `,
    variables: { categoryID: 1 },
  },
  {
    next: (data) => {
      console.log('Product update:', data);
    },
    error: (err) => {
      console.error('Subscription error:', err);
    },
    complete: () => {
      console.log('Subscription completed');
    },
  }
);

// Clean up
setTimeout(() => unsubscribe(), 30000);
```

### Go Client

```go
import (
    "github.com/gorilla/websocket"
    "encoding/json"
)

type SubscriptionMessage struct {
    ID      string      `json:"id"`
    Type    string      `json:"type"`
    Payload interface{} `json:"payload,omitempty"`
}

func subscribeToProductUpdates() {
    conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8080/graphql", nil)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()
    
    // Send connection init
    conn.WriteJSON(SubscriptionMessage{
        Type: "connection_init",
    })
    
    // Send subscription
    conn.WriteJSON(SubscriptionMessage{
        ID:   "1",
        Type: "start",
        Payload: map[string]interface{}{
            "query": `
                subscription {
                    productUpdates {
                        product { id name price }
                        action
                        timestamp
                    }
                }
            `,
        },
    })
    
    // Read messages
    for {
        var msg SubscriptionMessage
        err := conn.ReadJSON(&msg)
        if err != nil {
            log.Println("Read error:", err)
            break
        }
        
        switch msg.Type {
        case "connection_ack":
            log.Println("Connection acknowledged")
        case "data":
            log.Printf("Received update: %+v", msg.Payload)
        case "error":
            log.Printf("Subscription error: %+v", msg.Payload)
        case "complete":
            log.Println("Subscription completed")
            return
        }
    }
}
```

## Error Handling and Best Practices

### Graceful Error Handling

```go
func RobustSubscription(ctx context.Context) (<-chan DataUpdate, error) {
    updates := make(chan DataUpdate)
    
    go func() {
        defer func() {
            if r := recover(); r != nil {
                log.Printf("Subscription panic recovered: %v", r)
            }
            close(updates)
        }()
        
        for {
            select {
            case <-ctx.Done():
                return
            case update, ok := <-dataSource:
                if !ok {
                    return // Source closed
                }
                
                // Handle potential errors in update processing
                processedUpdate, err := processUpdate(update)
                if err != nil {
                    log.Printf("Error processing update: %v", err)
                    continue
                }
                
                select {
                case updates <- processedUpdate:
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    
    return updates, nil
}
```

### Resource Management

```go
// ✅ Always close channels
go func() {
    defer close(updates) // Prevents goroutine leaks
    // ...
}()

// ✅ Respect context cancellation
select {
case <-ctx.Done():
    return // Stop processing immediately
case data := <-source:
    // Process data
}

// ✅ Use buffered channels for bursty data
updates := make(chan Update, 100) // Buffer prevents blocking

// ✅ Handle channel send failures
select {
case ch <- update:
    // Sent successfully
default:
    // Channel full or closed - handle gracefully
    log.Printf("Failed to send update, channel may be closed")
}
```

### Performance Considerations

```go
// ✅ Efficient broadcast pattern
type Hub struct {
    clients map[chan Update]bool
    mutex   sync.RWMutex
}

func (h *Hub) AddClient(ch chan Update) {
    h.mutex.Lock()
    defer h.mutex.Unlock()
    h.clients[ch] = true
}

func (h *Hub) RemoveClient(ch chan Update) {
    h.mutex.Lock()
    defer h.mutex.Unlock()
    delete(h.clients, ch)
    close(ch)
}

func (h *Hub) Broadcast(update Update) {
    h.mutex.RLock()
    defer h.mutex.RUnlock()
    
    for ch := range h.clients {
        select {
        case ch <- update:
        default:
            // Channel blocked, consider removing slow clients
        }
    }
}
```

## Testing Subscriptions

### Unit Testing

```go
func TestCurrentTimeSubscription(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    updates, err := CurrentTime(ctx, 100)
    require.NoError(t, err)
    
    // Read first update
    select {
    case update := <-updates:
        assert.NotZero(t, update.Timestamp)
        assert.NotEmpty(t, update.Formatted)
    case <-time.After(200 * time.Millisecond):
        t.Fatal("Expected update within 200ms")
    }
    
    // Cancel and verify channel closes
    cancel()
    select {
    case _, ok := <-updates:
        assert.False(t, ok, "Channel should be closed")
    case <-time.After(100 * time.Millisecond):
        t.Fatal("Channel should close within 100ms")
    }
}
```

### Integration Testing

```go
func TestSubscriptionOverWebSocket(t *testing.T) {
    // Set up test server
    g := &quickgraph.Graphy{}
    g.RegisterSubscription(ctx, "currentTime", CurrentTime, "intervalMs")
    
    server := httptest.NewServer(g.HttpHandlerWithWebSocket(testUpgrader))
    defer server.Close()
    
    // Connect with WebSocket client
    url := "ws" + strings.TrimPrefix(server.URL, "http") + "/graphql"
    conn, _, err := websocket.DefaultDialer.Dial(url, nil)
    require.NoError(t, err)
    defer conn.Close()
    
    // Test subscription protocol
    // ... test implementation
}
```

## Examples and Sample Code

Check out the [sample application](https://github.com/gburgyan/go-quickgraph-sample) for complete working examples:

- **Real-time Chat**: Message streaming with user filtering
- **Order Tracking**: Status updates for long-running processes  
- **Live Dashboard**: Real-time metrics and alerts
- **Notification System**: User-specific notification delivery

## Next Steps

- **[Authentication](AUTH_PATTERNS.md)** - Securing subscription endpoints
- **[Performance](PERFORMANCE.md)** - Optimizing real-time data delivery
- **[Error Handling](ERROR_HANDLING.md)** - Robust error management in streaming