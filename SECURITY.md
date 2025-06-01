# Security Guide

This document provides comprehensive guidance on securing GraphQL services built with go-quickgraph.

## Table of Contents

- [Overview](#overview)
- [Memory Limits](#memory-limits)
- [WebSocket Authentication](#websocket-authentication)
- [Built-in DoS Protection](#built-in-dos-protection)
- [Production Security Checklist](#production-security-checklist)
- [Common Vulnerabilities](#common-vulnerabilities)
- [Best Practices](#best-practices)

## Overview

go-quickgraph includes multiple layers of security features designed to protect against common GraphQL vulnerabilities:

- **Memory exhaustion attacks** through configurable resource limits
- **Subscription flooding** via per-connection limits
- **Unauthorized access** through flexible authentication interfaces
- **Query complexity attacks** via built-in depth and complexity analysis

The security model is designed to be **customer-flexible** - providing secure defaults while allowing customization for specific use cases.

## Memory Limits

The `MemoryLimits` struct provides comprehensive protection against memory-based attacks:

```go
type MemoryLimits struct {
    MaxRequestBodySize            int64 // HTTP request body size limit (bytes)
    MaxVariableSize              int64  // GraphQL variables JSON size limit (bytes)
    SubscriptionBufferSize       int    // Channel buffer size for subscriptions
    MaxWebSocketConnections      int    // Global WebSocket connection limit (customer-implemented)
    MaxSubscriptionsPerConnection int    // Per-connection subscription limit
}
```

### Configuration Examples

#### Basic Setup
```go
g := &Graphy{
    MemoryLimits: &MemoryLimits{
        MaxRequestBodySize:            1024 * 1024,  // 1MB
        MaxVariableSize:               64 * 1024,    // 64KB
        SubscriptionBufferSize:        100,          // Buffered channels
        MaxSubscriptionsPerConnection: 10,           // 10 subs per connection
    },
}
```

#### High-Traffic Production
```go
g := &Graphy{
    MemoryLimits: &MemoryLimits{
        MaxRequestBodySize:            512 * 1024,   // 512KB (stricter)
        MaxVariableSize:               32 * 1024,    // 32KB (stricter)
        SubscriptionBufferSize:        50,           // Smaller buffers
        MaxSubscriptionsPerConnection: 5,            // Fewer subs per connection
    },
}
```

#### Development/Testing
```go
g := &Graphy{
    MemoryLimits: &MemoryLimits{
        MaxRequestBodySize:            10 * 1024 * 1024, // 10MB (relaxed)
        MaxVariableSize:               1024 * 1024,      // 1MB (relaxed)
        SubscriptionBufferSize:        1000,             // Large buffers
        MaxSubscriptionsPerConnection: 100,              // Many subscriptions
    },
}
```

### Understanding the Limits

#### HTTP Body Size Limiting
- Uses `io.LimitReader` to enforce limits at the HTTP layer
- Prevents large payload attacks before JSON parsing
- Set to `0` for unlimited (not recommended in production)

```go
// Implementation detail (automatic)
var bodyReader io.Reader = request.Body
if g.MemoryLimits != nil && g.MemoryLimits.MaxRequestBodySize > 0 {
    bodyReader = io.LimitReader(request.Body, g.MemoryLimits.MaxRequestBodySize)
}
```

#### Variable Size Validation
- Validates the size of the variables JSON after HTTP body parsing
- Catches cases where small HTTP bodies contain large variable objects
- Applied during request processing, not HTTP parsing

#### Subscription Buffer Management
- Controls memory usage of subscription channels
- `0` = unbuffered channels (blocks on send)
- `> 0` = buffered channels with specified capacity
- `< 0` = treated as unbuffered (safe default)

## WebSocket Authentication

The `WebSocketAuthenticator` interface provides flexible authentication for WebSocket subscriptions:

```go
type WebSocketAuthenticator interface {
    AuthenticateConnection(ctx context.Context, initPayload json.RawMessage) (context.Context, error)
    AuthorizeSubscription(ctx context.Context, query string, variables json.RawMessage) (context.Context, error)
}
```

### Implementation Examples

#### JWT Token Authentication
```go
type JWTAuthenticator struct {
    secretKey []byte
}

func (j *JWTAuthenticator) AuthenticateConnection(ctx context.Context, payload json.RawMessage) (context.Context, error) {
    var initData struct {
        Token string `json:"token"`
    }
    
    if err := json.Unmarshal(payload, &initData); err != nil {
        return nil, fmt.Errorf("invalid payload format: %v", err)
    }
    
    // Validate JWT token
    token, err := jwt.Parse(initData.Token, func(token *jwt.Token) (interface{}, error) {
        return j.secretKey, nil
    })
    
    if err != nil || !token.Valid {
        return nil, fmt.Errorf("invalid token")
    }
    
    // Extract user info from claims
    if claims, ok := token.Claims.(jwt.MapClaims); ok {
        user := &User{
            ID:   claims["user_id"].(string),
            Role: claims["role"].(string),
        }
        return context.WithValue(ctx, "user", user), nil
    }
    
    return nil, fmt.Errorf("invalid token claims")
}

func (j *JWTAuthenticator) AuthorizeSubscription(ctx context.Context, query string, variables json.RawMessage) (context.Context, error) {
    user, ok := ctx.Value("user").(*User)
    if !ok {
        return nil, fmt.Errorf("no authenticated user")
    }
    
    // Role-based authorization
    if strings.Contains(query, "adminSubscription") && user.Role != "admin" {
        return nil, fmt.Errorf("admin access required")
    }
    
    if strings.Contains(query, "premiumSubscription") && user.Role == "free" {
        return nil, fmt.Errorf("premium subscription required")
    }
    
    return ctx, nil
}
```

#### API Key Authentication
```go
type APIKeyAuthenticator struct {
    validKeys map[string]*Client
}

func (a *APIKeyAuthenticator) AuthenticateConnection(ctx context.Context, payload json.RawMessage) (context.Context, error) {
    var authData struct {
        APIKey string `json:"apiKey"`
    }
    
    if err := json.Unmarshal(payload, &authData); err != nil {
        return nil, fmt.Errorf("invalid auth payload")
    }
    
    client, exists := a.validKeys[authData.APIKey]
    if !exists {
        return nil, fmt.Errorf("invalid API key")
    }
    
    if client.IsExpired() {
        return nil, fmt.Errorf("API key expired")
    }
    
    return context.WithValue(ctx, "client", client), nil
}

func (a *APIKeyAuthenticator) AuthorizeSubscription(ctx context.Context, query string, variables json.RawMessage) (context.Context, error) {
    client, ok := ctx.Value("client").(*Client)
    if !ok {
        return nil, fmt.Errorf("no authenticated client")
    }
    
    // Check subscription limits
    if client.SubscriptionTier == "basic" && strings.Contains(query, "realTimeData") {
        return nil, fmt.Errorf("real-time subscriptions not available for basic tier")
    }
    
    return ctx, nil
}
```

#### Rate-Limited Authentication
```go
type RateLimitedAuthenticator struct {
    baseAuth WebSocketAuthenticator
    limiter  map[string]*rate.Limiter
    mu       sync.RWMutex
}

func (r *RateLimitedAuthenticator) AuthenticateConnection(ctx context.Context, payload json.RawMessage) (context.Context, error) {
    // Extract client identifier (IP, user ID, etc.)
    clientID := r.getClientID(ctx, payload)
    
    // Check rate limit
    if !r.checkRateLimit(clientID) {
        return nil, fmt.Errorf("authentication rate limit exceeded")
    }
    
    // Delegate to base authenticator
    return r.baseAuth.AuthenticateConnection(ctx, payload)
}

func (r *RateLimitedAuthenticator) checkRateLimit(clientID string) bool {
    r.mu.RLock()
    limiter, exists := r.limiter[clientID]
    r.mu.RUnlock()
    
    if !exists {
        r.mu.Lock()
        limiter = rate.NewLimiter(rate.Every(time.Minute), 10) // 10 attempts per minute
        r.limiter[clientID] = limiter
        r.mu.Unlock()
    }
    
    return limiter.Allow()
}
```

### WebSocket Handler Setup

```go
// Default handler (no authentication)
handler := NewGraphQLWebSocketHandler(graphy)

// With custom authentication
auth := &JWTAuthenticator{secretKey: []byte("your-secret")}
handler := NewGraphQLWebSocketHandlerWithAuth(graphy, auth)

// Integration with HTTP server
upgrader := &YourWebSocketUpgrader{} // Implement WebSocketUpgrader interface
httpHandler := graphy.HttpHandlerWithWebSocket(upgrader)
```

## Built-in DoS Protection

### Query Limits (Pre-existing)
```go
g := &Graphy{
    QueryLimits: &QueryLimits{
        MaxDepth:      10,   // Maximum query nesting depth
        MaxComplexity: 1000, // Maximum query complexity score
    },
}
```

### Request Processing Limits
- **Parsing Timeout**: Queries that take too long to parse are rejected
- **Execution Timeout**: Use context timeouts for long-running operations
- **Memory Bounds**: All memory limits work together to prevent exhaustion

### Subscription Protection
- **Connection Limits**: Prevent too many concurrent connections
- **Per-Connection Limits**: Prevent subscription flooding from single clients
- **Buffer Limits**: Control memory usage of subscription channels
- **Cleanup Guarantees**: Automatic cleanup on disconnect prevents resource leaks

## Production Security Checklist

### Essential Configuration
- [ ] Set `MaxRequestBodySize` (recommended: 1MB or less)
- [ ] Set `MaxVariableSize` (recommended: 64KB or less)
- [ ] Configure `MaxSubscriptionsPerConnection` (recommended: 10 or less)
- [ ] Set `SubscriptionBufferSize` based on expected load
- [ ] Implement WebSocket authentication if using subscriptions
- [ ] Configure `QueryLimits` for depth and complexity

### Infrastructure Security
- [ ] Use HTTPS/WSS in production
- [ ] Implement rate limiting at reverse proxy/load balancer
- [ ] Set up monitoring for memory usage and connection counts
- [ ] Configure proper CORS policies
- [ ] Implement request logging and monitoring
- [ ] Set up alerts for unusual traffic patterns

### Application Security
- [ ] Validate all input in GraphQL resolvers
- [ ] Implement proper authorization in resolvers
- [ ] Use parameterized queries to prevent injection
- [ ] Audit third-party dependencies regularly
- [ ] Implement proper error handling (don't leak sensitive info)
- [ ] Use secure session management

### WebSocket Specific
- [ ] Authenticate all WebSocket connections
- [ ] Implement connection cleanup on authentication failure
- [ ] Monitor subscription creation/cancellation patterns
- [ ] Set reasonable WebSocket timeouts
- [ ] Implement proper connection state management

## Common Vulnerabilities

### 1. Query Depth Attack
**Attack**: Deeply nested queries that consume excessive processing time
```graphql
query {
  user {
    friends {
      friends {
        friends {
          # ... 100 levels deep
        }
      }
    }
  }
}
```
**Protection**: Set `QueryLimits.MaxDepth`

### 2. Query Complexity Attack
**Attack**: Queries with high computational complexity
```graphql
query {
  users(first: 1000) {
    posts(first: 1000) {
      comments(first: 1000) {
        author { name }
      }
    }
  }
}
```
**Protection**: Set `QueryLimits.MaxComplexity`

### 3. Memory Exhaustion
**Attack**: Large request payloads or variables
```json
{
  "query": "mutation($data: String!) { process(data: $data) }",
  "variables": {
    "data": "A".repeat(10000000)
  }
}
```
**Protection**: Set `MemoryLimits.MaxRequestBodySize` and `MaxVariableSize`

### 4. Subscription Flooding
**Attack**: Opening many subscriptions to exhaust server resources
```javascript
// Client opens 1000+ concurrent subscriptions
for (let i = 0; i < 1000; i++) {
  client.subscribe({ query: "subscription { realTimeData }" });
}
```
**Protection**: Set `MemoryLimits.MaxSubscriptionsPerConnection`

### 5. Unauthorized Access
**Attack**: Accessing restricted subscriptions without authentication
```graphql
subscription {
  adminNotifications {
    message
    sensitiveData
  }
}
```
**Protection**: Implement `WebSocketAuthenticator` with proper authorization

## Best Practices

### Security Configuration
1. **Start Restrictive**: Begin with strict limits and relax as needed
2. **Monitor in Production**: Track memory usage, connection counts, and query patterns
3. **Layer Security**: Combine multiple protection mechanisms
4. **Test Thoroughly**: Use the comprehensive test suite to validate security

### Authentication
1. **Fail Securely**: Default to denying access when authentication fails
2. **Context Propagation**: Use context values to pass authenticated user info
3. **Token Validation**: Always validate tokens/credentials properly
4. **Rate Limiting**: Implement rate limiting for authentication attempts

### Subscription Management
1. **Resource Cleanup**: Ensure subscriptions are properly cleaned up
2. **Authorization**: Check permissions for each subscription request
3. **Monitoring**: Track subscription creation and cancellation patterns
4. **Buffer Sizing**: Choose buffer sizes based on expected message rates

### Error Handling
1. **Don't Leak Info**: Avoid exposing internal errors to clients
2. **Log Security Events**: Log authentication failures and suspicious activity
3. **Graceful Degradation**: Handle resource exhaustion gracefully
4. **Clear Error Messages**: Provide helpful error messages for legitimate issues

### Testing Security
```go
// Example security test
func TestSecurityLimits(t *testing.T) {
    g := &Graphy{
        MemoryLimits: &MemoryLimits{
            MaxRequestBodySize: 1024,
            MaxVariableSize:   512,
        },
    }
    
    // Test that large requests are rejected
    largePayload := strings.Repeat("A", 2048)
    resp := testGraphQLRequest(g, largePayload)
    assert.Contains(t, resp.Errors, "request too large")
}
```

This security configuration provides a solid foundation for production GraphQL services while maintaining the flexibility to customize for specific requirements.