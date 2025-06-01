# Security API Reference

This document provides detailed API reference for all security-related features in go-quickgraph.

## Table of Contents

- [Production Mode](#production-mode)
- [MemoryLimits](#memorylimits)
- [WebSocketAuthenticator Interface](#websocketauthenticator-interface)
- [WebSocket Handler Functions](#websocket-handler-functions)
- [Built-in Authenticators](#built-in-authenticators)
- [Testing Utilities](#testing-utilities)
- [Production Security Checklist](#production-security-checklist)

## Production Mode

The `ProductionMode` configuration controls error reporting behavior to prevent information disclosure in production environments.

### Configuration

```go
type Graphy struct {
    ProductionMode bool // Controls error sanitization behavior
    // ... other fields
}
```

### Field

#### ProductionMode
- **Type**: `bool`
- **Default**: `false` (development mode)
- **Purpose**: Controls whether sensitive error information is exposed to clients
- **Critical**: **MUST** be set to `true` for production deployments

### Behavior

#### Development Mode (ProductionMode: false)
- **Client Responses**: Include detailed error information (stack traces, function names, panic details)
- **Error Handler**: Receives full error details
- **Extensions**: All error extensions are preserved in client responses
- **Use Case**: Development and debugging environments

#### Production Mode (ProductionMode: true)
- **Client Responses**: Sanitized generic error messages ("Internal server error")
- **Error Handler**: Still receives full error details for logging/monitoring
- **Extensions**: Sensitive extensions filtered from client responses
- **Use Case**: Production deployments to prevent information disclosure

### Sanitized Information

When `ProductionMode: true`, the following information is filtered from client responses:

- **Stack Traces**: Complete stack traces removed
- **Function Names**: GraphQL function names in error messages
- **Panic Details**: Specific panic values and types  
- **Inner Errors**: Underlying Go error details
- **Sensitive Extensions**: Keys like `stack`, `function_name`, `panic_value`

### Dual Error Handling

Production mode implements dual error handling:

1. **Client Response**: Sanitized errors safe for external consumption
2. **Error Handler**: Full error details for internal logging and monitoring

```go
// Example: Function panic in production mode

// Client receives:
{
  "errors": [{
    "message": "Internal server error",
    "locations": [{"line": 2, "column": 10}],
    "path": ["userProfile"]
  }]
}

// Error handler receives:
Error: function getUserProfile panicked: nil pointer dereference
Details: {
  "operation": "function_panic",
  "function_name": "getUserProfile", 
  "panic_value": "nil pointer dereference",
  "stack_trace": "goroutine 1 [running]:\n...",
  "request_method": "graphql"
}
```

### Usage Examples

#### Basic Configuration

```go
// Production deployment - REQUIRED
g := &Graphy{
    ProductionMode: true,  // ⚠️ CRITICAL for production
    MemoryLimits: &MemoryLimits{...},
    QueryLimits: &QueryLimits{...},
}

// Development environment
g := &Graphy{
    ProductionMode: false, // Default: detailed errors
}
```

#### With Error Handler

```go
// Production configuration with monitoring
g := &Graphy{
    ProductionMode: true,
}

// Error handler receives full details even in production mode
g.SetErrorHandler(ErrorHandlerFunc(func(ctx context.Context, category ErrorCategory, err error, details map[string]interface{}) {
    switch category {
    case ErrorCategoryExecution:
        // Log full panic details for monitoring
        log.Printf("EXECUTION ERROR: %v - Details: %+v", err, details)
        
        // Send to monitoring system
        if functionName, ok := details["function_name"]; ok {
            metrics.IncrementCounter("graphql.function.panic", map[string]string{
                "function": functionName.(string),
            })
        }
        
    case ErrorCategoryInternal:
        // Log internal errors with full context
        log.Printf("INTERNAL ERROR: %v - Details: %+v", err, details)
    }
}))
```

#### Environment-Based Configuration

```go
// Configure based on environment
productionMode := os.Getenv("ENV") == "production"

g := &Graphy{
    ProductionMode: productionMode,
    MemoryLimits: &MemoryLimits{
        MaxRequestBodySize: getRequestSizeLimit(),
        MaxVariableSize:    getVariableSizeLimit(),
    },
}

if productionMode {
    // Production-specific error handling
    g.SetErrorHandler(createProductionErrorHandler())
} else {
    // Development-specific error handling
    g.SetErrorHandler(createDevelopmentErrorHandler())
}
```

### Security Considerations

#### Information Disclosure Prevention
- **Stack Traces**: Can reveal application structure, file paths, and internal implementation
- **Function Names**: Expose GraphQL schema implementation details
- **Panic Messages**: May contain sensitive runtime information
- **File Paths**: Stack traces can reveal server directory structure

#### Monitoring and Debugging
- Error handlers always receive complete information for debugging
- Production incidents can be fully investigated through logs
- Monitoring systems receive detailed error context
- Client responses remain secure

### Testing Production Mode

```go
func TestProductionModeErrorSanitization(t *testing.T) {
    // Test function that panics
    panicFunc := func(ctx context.Context) (string, error) {
        panic("sensitive internal error")
    }

    g := &Graphy{ProductionMode: true}
    g.RegisterQuery(context.Background(), "testPanic", panicFunc)

    result, err := g.ProcessRequest(context.Background(), `{testPanic}`, "{}")
    
    // Client response should be sanitized
    assert.Contains(t, result, "Internal server error")
    assert.NotContains(t, result, "sensitive internal error")
    assert.NotContains(t, result, "testPanic panicked")
    
    // Error should still exist for logging
    assert.Error(t, err)
}
```

## MemoryLimits

The `MemoryLimits` struct provides comprehensive protection against memory exhaustion attacks.

```go
type MemoryLimits struct {
    MaxRequestBodySize            int64 // HTTP request body size limit in bytes
    MaxVariableSize              int64  // GraphQL variables JSON size limit in bytes
    SubscriptionBufferSize       int    // Channel buffer size for subscriptions
    MaxWebSocketConnections      int    // Global WebSocket connection limit (customer-implemented)
    MaxSubscriptionsPerConnection int    // Per-connection subscription limit
}
```

### Fields

#### MaxRequestBodySize
- **Type**: `int64`
- **Purpose**: Limits the size of HTTP request bodies to prevent large payload attacks
- **Implementation**: Uses `io.LimitReader` to enforce limits at the HTTP layer
- **Behavior**:
  - `0`: Unlimited (not recommended for production)
  - `> 0`: Enforces the specified byte limit
- **Example**: `1024 * 1024` (1MB limit)

#### MaxVariableSize
- **Type**: `int64`
- **Purpose**: Limits the size of GraphQL variables JSON after parsing
- **Implementation**: Validates during request processing after HTTP body parsing
- **Behavior**:
  - `0`: Unlimited
  - `> 0`: Enforces the specified byte limit on variables JSON
- **Example**: `64 * 1024` (64KB limit)

#### SubscriptionBufferSize
- **Type**: `int`
- **Purpose**: Controls memory usage of subscription channels
- **Implementation**: Used as buffer size when creating subscription channels
- **Behavior**:
  - `0`: Unbuffered channels (blocks on send)
  - `> 0`: Buffered channels with specified capacity
  - `< 0`: Treated as unbuffered (safe default)
- **Example**: `100` (100-message buffer)

#### MaxWebSocketConnections
- **Type**: `int`
- **Purpose**: Global limit for WebSocket connections (customer-implemented)
- **Implementation**: Not enforced by library; provided for customer implementation
- **Usage**: Track connections at HTTP upgrade layer
- **Example**: See [connection tracking example](#connection-tracking-example)

#### MaxSubscriptionsPerConnection
- **Type**: `int`
- **Purpose**: Limits subscriptions per WebSocket connection
- **Implementation**: Enforced in WebSocket handler
- **Behavior**:
  - `0`: Unlimited
  - `> 0`: Enforces the specified limit per connection
- **Example**: `10` (10 subscriptions per connection)

### Usage Example

```go
g := &Graphy{
    MemoryLimits: &MemoryLimits{
        MaxRequestBodySize:            1024 * 1024, // 1MB HTTP bodies
        MaxVariableSize:               64 * 1024,   // 64KB variables
        SubscriptionBufferSize:        100,         // 100-message buffers
        MaxWebSocketConnections:       1000,        // Global connection limit
        MaxSubscriptionsPerConnection: 10,          // 10 subs per connection
    },
}
```

## WebSocketAuthenticator Interface

The `WebSocketAuthenticator` interface provides flexible authentication for WebSocket subscriptions.

```go
type WebSocketAuthenticator interface {
    AuthenticateConnection(ctx context.Context, initPayload json.RawMessage) (context.Context, error)
    AuthorizeSubscription(ctx context.Context, query string, variables json.RawMessage) (context.Context, error)
}
```

### Methods

#### AuthenticateConnection
Authenticates a WebSocket connection during the initial handshake.

```go
AuthenticateConnection(ctx context.Context, initPayload json.RawMessage) (context.Context, error)
```

**Parameters**:
- `ctx`: The base context for the connection
- `initPayload`: JSON payload from the client's connection_init message

**Returns**:
- `context.Context`: Authenticated context with user/client information
- `error`: Authentication error (closes connection if non-nil)

**Behavior**:
- Called once per WebSocket connection during `connection_init`
- Authentication failure sends `connection_error` and closes connection
- Successful authentication sends `connection_ack`
- Returned context is used for all subsequent operations on this connection

**Example**:
```go
func (a *MyAuth) AuthenticateConnection(ctx context.Context, payload json.RawMessage) (context.Context, error) {
    var authData struct {
        Token string `json:"token"`
    }
    
    if err := json.Unmarshal(payload, &authData); err != nil {
        return nil, fmt.Errorf("invalid payload: %v", err)
    }
    
    user, err := validateToken(authData.Token)
    if err != nil {
        return nil, fmt.Errorf("invalid token: %v", err)
    }
    
    return context.WithValue(ctx, "user", user), nil
}
```

#### AuthorizeSubscription
Authorizes individual subscription requests.

```go
AuthorizeSubscription(ctx context.Context, query string, variables json.RawMessage) (context.Context, error)
```

**Parameters**:
- `ctx`: Authenticated context from `AuthenticateConnection`
- `query`: GraphQL subscription query string
- `variables`: GraphQL variables for the subscription

**Returns**:
- `context.Context`: Authorized context (may add additional info)
- `error`: Authorization error (rejects subscription if non-nil)

**Behavior**:
- Called for each subscription request on an authenticated connection
- Authorization failure sends `error` message for that subscription
- Can implement role-based access control, rate limiting, etc.
- Returned context is passed to the subscription function

**Example**:
```go
func (a *MyAuth) AuthorizeSubscription(ctx context.Context, query string, variables json.RawMessage) (context.Context, error) {
    user, ok := ctx.Value("user").(*User)
    if !ok {
        return nil, fmt.Errorf("no authenticated user")
    }
    
    // Role-based authorization
    if strings.Contains(query, "adminSubscription") && user.Role != "admin" {
        return nil, fmt.Errorf("admin access required")
    }
    
    return ctx, nil
}
```

## WebSocket Handler Functions

### NewGraphQLWebSocketHandler
Creates a WebSocket handler with no authentication (default).

```go
func NewGraphQLWebSocketHandler(graphy *Graphy) *GraphQLWebSocketHandler
```

**Parameters**:
- `graphy`: The Graphy instance

**Returns**:
- `*GraphQLWebSocketHandler`: Handler with `NoOpWebSocketAuthenticator`

**Usage**:
```go
handler := NewGraphQLWebSocketHandler(g)
```

### NewGraphQLWebSocketHandlerWithAuth
Creates a WebSocket handler with custom authentication.

```go
func NewGraphQLWebSocketHandlerWithAuth(graphy *Graphy, authenticator WebSocketAuthenticator) *GraphQLWebSocketHandler
```

**Parameters**:
- `graphy`: The Graphy instance
- `authenticator`: Custom authentication implementation

**Returns**:
- `*GraphQLWebSocketHandler`: Handler with specified authenticator

**Usage**:
```go
auth := &MyAuthenticator{}
handler := NewGraphQLWebSocketHandlerWithAuth(g, auth)
```

### HandleConnection
Handles a WebSocket connection using the graphql-ws protocol.

```go
func (h *GraphQLWebSocketHandler) HandleConnection(ctx context.Context, conn SimpleWebSocketConn)
```

**Parameters**:
- `ctx`: Context for the connection (with timeout/cancellation)
- `conn`: WebSocket connection implementing `SimpleWebSocketConn`

**Behavior**:
- Implements full graphql-ws protocol
- Handles connection init, subscriptions, and cleanup
- Enforces subscription limits
- Automatically cleans up on disconnect

**Usage**:
```go
// Typically called by WebSocket upgrader
ctx, cancel := context.WithTimeout(r.Context(), 1*time.Hour)
defer cancel()
handler.HandleConnection(ctx, conn)
```

## Built-in Authenticators

### NoOpWebSocketAuthenticator
Default authenticator that allows all connections and subscriptions.

```go
type NoOpWebSocketAuthenticator struct{}

func (n NoOpWebSocketAuthenticator) AuthenticateConnection(ctx context.Context, initPayload json.RawMessage) (context.Context, error) {
    return ctx, nil // Always allow
}

func (n NoOpWebSocketAuthenticator) AuthorizeSubscription(ctx context.Context, query string, variables json.RawMessage) (context.Context, error) {
    return ctx, nil // Always allow
}
```

**Usage**:
- Used automatically by `NewGraphQLWebSocketHandler`
- Provides backward compatibility
- Suitable for development or when authentication is handled externally

## Connection Tracking Example

For implementing global WebSocket connection limits:

```go
type ConnectionTracker struct {
    mu             sync.Mutex
    count          int
    maxConnections int
}

func (ct *ConnectionTracker) CanConnect() bool {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    return ct.count < ct.maxConnections
}

func (ct *ConnectionTracker) OnConnect() {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    ct.count++
}

func (ct *ConnectionTracker) OnDisconnect() {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    ct.count--
}

// Usage in HTTP handler
func (h *MyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Header.Get("Upgrade") == "websocket" {
        if !h.tracker.CanConnect() {
            http.Error(w, "Too many connections", http.StatusTooManyRequests)
            return
        }
        
        conn, err := h.upgrader.Upgrade(w, r)
        if err != nil {
            return
        }
        
        h.tracker.OnConnect()
        defer h.tracker.OnDisconnect()
        
        h.wsHandler.HandleConnection(r.Context(), conn)
    }
}
```

## Testing Utilities

### Mock WebSocket Connection
For testing WebSocket authentication:

```go
type MockWebSocketConn struct {
    mock.Mock
    messages [][]byte
    closed   bool
    mu       sync.Mutex
}

func (m *MockWebSocketConn) ReadMessage() ([]byte, error) {
    args := m.Called()
    return args.Get(0).([]byte), args.Error(1)
}

func (m *MockWebSocketConn) WriteMessage(data []byte) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.messages = append(m.messages, data)
    args := m.Called(data)
    return args.Error(0)
}

func (m *MockWebSocketConn) Close() error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.closed = true
    args := m.Called()
    return args.Error(0)
}

func (m *MockWebSocketConn) GetMessages() [][]byte {
    m.mu.Lock()
    defer m.mu.Unlock()
    result := make([][]byte, len(m.messages))
    copy(result, m.messages)
    return result
}
```

### Test Authenticator
Example authenticator for testing:

```go
type TestAuthenticator struct {
    validTokens map[string]*User
    requireAuth bool
}

func NewTestAuthenticator(requireAuth bool) *TestAuthenticator {
    return &TestAuthenticator{
        validTokens: map[string]*User{
            "valid-token": {ID: "1", Username: "user", Role: "user"},
            "admin-token": {ID: "2", Username: "admin", Role: "admin"},
        },
        requireAuth: requireAuth,
    }
}

func (t *TestAuthenticator) AuthenticateConnection(ctx context.Context, payload json.RawMessage) (context.Context, error) {
    if !t.requireAuth {
        return ctx, nil
    }
    
    var authData struct {
        Token string `json:"token"`
    }
    
    if err := json.Unmarshal(payload, &authData); err != nil {
        return nil, fmt.Errorf("invalid payload: %v", err)
    }
    
    user, exists := t.validTokens[authData.Token]
    if !exists {
        return nil, fmt.Errorf("invalid token")
    }
    
    return context.WithValue(ctx, "user", user), nil
}
```

## Error Handling

### Authentication Errors
When `AuthenticateConnection` returns an error:
1. A `connection_error` message is sent to the client
2. The WebSocket connection is closed
3. Error is logged via the error handling system

### Authorization Errors
When `AuthorizeSubscription` returns an error:
1. An `error` message is sent for that specific subscription
2. The connection remains open for other operations
3. Error is logged with subscription context

### Memory Limit Errors
When memory limits are exceeded:
1. HTTP requests: Return 400 Bad Request or JSON error response
2. Variables: Return GraphQL error response
3. Subscriptions: Enforce limits during subscription creation

## Protocol Compliance

The WebSocket implementation follows the [graphql-ws protocol](https://github.com/enisdenjo/graphql-ws/blob/master/PROTOCOL.md):

### Message Types
- **Client → Server**: `connection_init`, `subscribe`, `complete`
- **Server → Client**: `connection_ack`, `connection_error`, `next`, `error`, `complete`

### Connection Flow
1. Client sends `connection_init` with auth payload
2. Server calls `AuthenticateConnection`
3. Server responds with `connection_ack` or `connection_error`
4. Client sends `subscribe` messages
5. Server calls `AuthorizeSubscription` for each subscription
6. Server sends `next`, `error`, or `complete` messages

## Production Security Checklist

Before deploying go-quickgraph to production, ensure all security features are properly configured:

### Required Configuration

- [ ] **Production Mode**: Set `ProductionMode: true` in Graphy configuration
- [ ] **Error Handler**: Configure error handler to capture and log full error details
- [ ] **Memory Limits**: Configure appropriate `MemoryLimits` for your environment
- [ ] **Query Limits**: Set `QueryLimits` to prevent DoS attacks
- [ ] **WebSocket Authentication**: Implement `WebSocketAuthenticator` for subscription security
- [ ] **Connection Limits**: Implement global WebSocket connection tracking and limits

### Error Handling Verification

- [ ] **Production Mode Testing**: Verify client responses are sanitized in production mode
- [ ] **Error Handler Testing**: Confirm error handlers receive full details for monitoring
- [ ] **Panic Handling**: Test that function panics are properly handled and logged
- [ ] **Information Disclosure**: Ensure no sensitive information leaks to clients

### Memory and Performance

- [ ] **HTTP Body Limits**: Set `MaxRequestBodySize` to prevent large payload attacks
- [ ] **Variable Limits**: Configure `MaxVariableSize` for GraphQL variables
- [ ] **Subscription Buffering**: Set appropriate `SubscriptionBufferSize`
- [ ] **Connection Monitoring**: Implement connection count tracking and alerting

### Authentication and Authorization

- [ ] **WebSocket Auth**: Test connection authentication with valid/invalid credentials  
- [ ] **Subscription Auth**: Verify subscription-level authorization works correctly
- [ ] **Role-Based Access**: Test role-based access control if implemented
- [ ] **Token Validation**: Ensure token validation handles edge cases properly

### Monitoring and Observability

- [ ] **Error Logging**: Verify all error categories are properly logged
- [ ] **Metrics Collection**: Implement metrics for panics, errors, and performance
- [ ] **Alerting**: Set up alerts for high error rates or security violations
- [ ] **Audit Logging**: Log authentication and authorization events

### Example Production Configuration

```go
// Complete production configuration
g := &Graphy{
    ProductionMode: true, // ⚠️ CRITICAL: Enable error sanitization
    
    MemoryLimits: &MemoryLimits{
        MaxRequestBodySize:            1024 * 1024,     // 1MB HTTP bodies
        MaxVariableSize:               64 * 1024,       // 64KB variables  
        SubscriptionBufferSize:        100,             // 100-message buffers
        MaxWebSocketConnections:       1000,            // Global connection limit
        MaxSubscriptionsPerConnection: 10,              // Per-connection limit
    },
    
    QueryLimits: &QueryLimits{
        MaxDepth:               10,    // Query nesting depth
        MaxComplexity:          1000,  // Query complexity score
        MaxFields:              50,    // Fields per level
        MaxAliases:             30,    // Alias count
        MaxArraySize:           1000,  // Array size in responses
        MaxConcurrentResolvers: 100,   // Concurrent execution limit
    },
}

// Production error handler with monitoring
g.SetErrorHandler(ErrorHandlerFunc(func(ctx context.Context, category ErrorCategory, err error, details map[string]interface{}) {
    // Log full details for internal debugging
    logger.WithFields(logrus.Fields{
        "category": string(category),
        "error":    err.Error(),
        "details":  details,
    }).Error("GraphQL error occurred")
    
    // Send metrics to monitoring system
    metrics.IncrementCounter("graphql.errors", map[string]string{
        "category": string(category),
    })
    
    // Alert on critical errors
    if category == ErrorCategoryInternal {
        alerting.SendAlert("GraphQL internal error", err.Error())
    }
}))

// WebSocket handler with authentication
auth := &ProductionAuthenticator{
    tokenValidator: tokenValidator,
    roleChecker:    roleChecker,
}
wsHandler := NewGraphQLWebSocketHandlerWithAuth(g, auth)
```

This API provides comprehensive security features while maintaining flexibility for different authentication and authorization patterns.