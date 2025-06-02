# Security API Reference

This document provides comprehensive security guidance for go-quickgraph, including configuration, best practices, and implementation examples for production deployments.

## ⚠️ Critical: Production Mode Configuration

**MOST IMPORTANT**: Always enable production mode for production deployments to prevent information disclosure.

```go
g := &quickgraph.Graphy{
    ProductionMode: true,  // 🔒 CRITICAL: Sanitizes error messages
    QueryLimits: &quickgraph.QueryLimits{
        MaxDepth:     10,
        MaxFields:    100,
        MaxAliases:   15,
        MaxComplexity: 1000,
    },
    MemoryLimits: &quickgraph.MemoryLimits{
        MaxRequestBodySize:            1024 * 1024,  // 1MB
        MaxVariableSize:               64 * 1024,    // 64KB
        SubscriptionBufferSize:        100,
        MaxSubscriptionsPerConnection: 10,
    },
}
```

**What Production Mode Prevents:**
- Stack traces in client responses
- Internal function names in error messages
- Panic details and sensitive extensions
- Information disclosure attacks

**Without production mode**, clients receive detailed error information including stack traces and internal implementation details.

## Security Features Overview

### Built-in Protection

- ✅ **Query Complexity Limits** - Depth, field count, alias, and complexity protection
- ✅ **Memory Exhaustion Prevention** - HTTP body and variable size limits
- ✅ **Information Disclosure Protection** - Production mode error sanitization  
- ✅ **WebSocket Authentication** - Pluggable authentication framework
- ✅ **Input Validation** - Type-safe parameter validation
- ✅ **Panic Recovery** - Comprehensive error handling

### Security Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   HTTP Layer    │    │  GraphQL Layer   │    │ Application     │
│                 │    │                  │    │ Layer           │
│ • Rate Limiting │───▶│ • Query Limits   │───▶│ • Authentication│
│ • Body Limits   │    │ • Memory Limits  │    │ • Authorization │
│ • CORS          │    │ • Input Valid.   │    │ • Business Logic│
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Query & Memory Limits

### QueryLimits Configuration

Protects against GraphQL-specific DoS attacks:

```go
type QueryLimits struct {
    MaxDepth                int // Prevents deep nesting attacks
    MaxComplexity          int // Overall query cost limits  
    MaxFields              int // Wide query protection
    MaxAliases             int // Alias amplification protection
    MaxArraySize           int // Large result protection
    MaxConcurrentResolvers int // Concurrency limits
    ComplexityScorer       ComplexityScorer // Custom complexity calculation
}
```

**Attack Prevention Examples:**

```graphql
# Blocked by MaxDepth: 10
query DeepNestingAttack {
  user {
    posts {
      comments {
        replies {
          # ... continues 20+ levels deep
        }
      }
    }
  }
}

# Blocked by MaxAliases: 15
query AliasAmplificationAttack {
  a1: user { name }
  a2: user { name }
  # ... continues with 50+ aliases
}
```

### MemoryLimits Configuration

Prevents memory exhaustion attacks:

```go
type MemoryLimits struct {
    MaxRequestBodySize            int64 // HTTP body size limit (bytes)
    MaxVariableSize              int64  // GraphQL variables JSON limit (bytes)
    SubscriptionBufferSize       int    // Channel buffer size
    MaxSubscriptionsPerConnection int    // Per-connection subscription limit
}
```

**Key Features:**
- **HTTP Body Limiting**: Uses `io.LimitReader` to prevent large payloads
- **Variable Validation**: Limits JSON variable size after parsing
- **Subscription Control**: Manages channel buffers and connection limits

## WebSocket Security

### Authentication Interface

Implement the `WebSocketAuthenticator` interface for secure WebSocket subscriptions:

```go
type WebSocketAuthenticator interface {
    AuthenticateConnection(ctx context.Context, initPayload json.RawMessage) (context.Context, error)
    AuthorizeSubscription(ctx context.Context, query string, variables json.RawMessage) (context.Context, error)
}
```

### JWT Authentication Example

```go
type JWTWebSocketAuthenticator struct {
    jwtSecret []byte
}

func (j *JWTWebSocketAuthenticator) AuthenticateConnection(ctx context.Context, initPayload json.RawMessage) (context.Context, error) {
    var payload struct {
        Authorization string `json:"authorization"`
    }
    
    if err := json.Unmarshal(initPayload, &payload); err != nil {
        return nil, fmt.Errorf("invalid connection payload")
    }
    
    token := strings.TrimPrefix(payload.Authorization, "Bearer ")
    claims, err := j.validateJWT(token)
    if err != nil {
        return nil, fmt.Errorf("authentication failed: %v", err)
    }
    
    // Add user to context
    ctx = context.WithValue(ctx, "user", claims)
    return ctx, nil
}

func (j *JWTWebSocketAuthenticator) AuthorizeSubscription(ctx context.Context, query string, variables json.RawMessage) (context.Context, error) {
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

// Use the authenticator
auth := &JWTWebSocketAuthenticator{jwtSecret: []byte("your-secret")}
handler := quickgraph.NewGraphQLWebSocketHandlerWithAuth(graphy, auth)
```

### Connection Management

Implement global connection tracking:

```go
type ConnectionTracker struct {
    mu          sync.Mutex
    connections map[string]time.Time
    maxConn     int
}

func (ct *ConnectionTracker) AddConnection(id string) error {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    
    if len(ct.connections) >= ct.maxConn {
        return fmt.Errorf("maximum connections exceeded")
    }
    
    ct.connections[id] = time.Now()
    return nil
}

func (ct *ConnectionTracker) RemoveConnection(id string) {
    ct.mu.Lock()
    defer ct.mu.Unlock()
    delete(ct.connections, id)
}
```

## HTTP Layer Security

### Authentication Middleware

```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Authorization header required", http.StatusUnauthorized)
            return
        }
        
        token := strings.TrimPrefix(authHeader, "Bearer ")
        user, err := validateJWT(token)
        if err != nil {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }
        
        // Add user to request context
        ctx := context.WithValue(r.Context(), "user", user)
        r = r.WithContext(ctx)
        
        next.ServeHTTP(w, r)
    })
}

// Apply to GraphQL handler
http.Handle("/graphql", AuthMiddleware(graphy.HttpHandler()))
```

### Rate Limiting

```go
func RateLimitMiddleware(requestsPerMinute int) func(http.Handler) http.Handler {
    limiter := rate.NewLimiter(rate.Limit(requestsPerMinute/60), requestsPerMinute)
    
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Apply rate limiting
http.Handle("/graphql", RateLimitMiddleware(60)(graphy.HttpHandler()))
```

### Security Headers

```go
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Prevent clickjacking
        w.Header().Set("X-Frame-Options", "DENY")
        
        // Content type sniffing protection
        w.Header().Set("X-Content-Type-Options", "nosniff")
        
        // XSS protection
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        
        // HSTS (if using HTTPS)
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        
        // Content Security Policy
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        
        next.ServeHTTP(w, r)
    })
}
```

## Application Layer Security

### Field-Level Authorization

```go
func (e *Employee) PersonalDetails(ctx context.Context) (*PersonalInfo, error) {
    currentUser, ok := ctx.Value("user").(*User)
    if !ok {
        return nil, errors.New("authentication required")
    }
    
    // Allow if admin or viewing own data
    if currentUser.Role == "admin" || currentUser.Email == e.Email {
        return &PersonalInfo{
            Salary: e.Salary,
            Email:  e.Email,
        }, nil
    }
    
    return nil, errors.New("not authorized to view personal details")
}
```

### Input Validation

```go
func ValidateProductInput(input ProductInput) error {
    if len(input.Name) > 100 {
        return fmt.Errorf("product name too long")
    }
    
    if input.Price < 0 {
        return fmt.Errorf("price cannot be negative")
    }
    
    // Prevent injection attacks
    if containsDangerousPatterns(input.Description) {
        return fmt.Errorf("invalid characters in description")
    }
    
    return nil
}

func containsDangerousPatterns(input string) bool {
    dangerous := []string{"'", "\"", ";", "--", "/*", "*/", "xp_", "sp_"}
    lower := strings.ToLower(input)
    for _, pattern := range dangerous {
        if strings.Contains(lower, pattern) {
            return true
        }
    }
    return false
}
```

### Custom Scalar Security

```go
type SecureEmailScalar struct{}

func (s *SecureEmailScalar) ParseLiteral(value string) (interface{}, error) {
    // Validate email format
    if !isValidEmail(value) {
        return nil, fmt.Errorf("invalid email format")
    }
    
    // Additional security checks
    if containsSQLInjectionPatterns(value) {
        return nil, fmt.Errorf("invalid characters in email")
    }
    
    return value, nil
}

func isValidEmail(email string) bool {
    emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    return emailRegex.MatchString(email)
}
```

## Error Handling & Monitoring

### Production Error Handler

```go
type SecurityErrorHandler struct{}

func (s *SecurityErrorHandler) HandleError(ctx context.Context, category quickgraph.ErrorCategory, err error, metadata map[string]interface{}) {
    // Log full error details internally
    log.Printf("[%s] %v - Metadata: %+v", category, err, metadata)
    
    // Send alert for security-related errors
    if category == quickgraph.ErrorCategoryWebSocket {
        s.sendSecurityAlert(err, metadata)
    }
}

func (s *SecurityErrorHandler) sendSecurityAlert(err error, metadata map[string]interface{}) {
    // Implement alerting logic (email, Slack, etc.)
}

// Configure error handler
graphy.ErrorHandler = &SecurityErrorHandler{}
```

### Metrics Collection

```go
type SecurityMetrics struct {
    FailedAuth       int64
    BlockedQueries   int64
    RateLimitHits    int64
    WebSocketAbuse   int64
}

func (s *SecurityErrorHandler) HandleError(ctx context.Context, category quickgraph.ErrorCategory, err error, metadata map[string]interface{}) {
    // Increment security metrics
    switch category {
    case quickgraph.ErrorCategoryValidation:
        if strings.Contains(err.Error(), "exceeds maximum") {
            atomic.AddInt64(&s.metrics.BlockedQueries, 1)
        }
    case quickgraph.ErrorCategoryWebSocket:
        if strings.Contains(err.Error(), "authentication") {
            atomic.AddInt64(&s.metrics.FailedAuth, 1)
        }
    }
    
    // Alert on suspicious activity
    if s.metrics.FailedAuth > 100 {
        s.sendAlert("High number of authentication failures")
    }
}
```

## Security Best Practices

### 1. Defense in Depth

Implement security at multiple layers:

```
Internet → CDN/WAF → Load Balancer → Rate Limiter → Auth Middleware → GraphQL Limits → Application Logic
```

### 2. Principle of Least Privilege

```go
// Grant minimal necessary permissions
func (u *User) CanAccessField(fieldName string, resourceID string) bool {
    switch u.Role {
    case "admin":
        return true
    case "user":
        return u.ID == resourceID && fieldName != "salary"
    default:
        return false
    }
}
```

### 3. Secure Defaults

```go
// Use secure defaults in configuration
func NewSecureGraphy() *quickgraph.Graphy {
    return &quickgraph.Graphy{
        ProductionMode: true,  // Secure by default
        QueryLimits: &quickgraph.QueryLimits{
            MaxDepth:     10,
            MaxFields:    50,
            MaxAliases:   10,
            MaxComplexity: 500,
        },
        MemoryLimits: &quickgraph.MemoryLimits{
            MaxRequestBodySize: 512 * 1024,  // 512KB
            MaxVariableSize:    64 * 1024,   // 64KB
        },
    }
}
```

### 4. Regular Security Reviews

- **Code Reviews** - Security-focused code review process
- **Dependency Scanning** - Regular vulnerability scanning of dependencies
- **Penetration Testing** - Periodic security testing by external experts
- **Security Audits** - Annual comprehensive security audits

## Security Testing

### Automated Security Tests

```go
func TestSecurityLimits(t *testing.T) {
    g := &quickgraph.Graphy{
        ProductionMode: true,
        QueryLimits: &quickgraph.QueryLimits{
            MaxDepth: 5,
            MaxFields: 10,
        },
    }
    
    // Test depth limit
    deepQuery := buildDeepQuery(10) // Creates 10-level deep query
    _, err := g.ProcessRequest(context.Background(), deepQuery, "")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "exceeds maximum allowed depth")
    
    // Test field limit
    wideQuery := buildWideQuery(20) // Creates query with 20 fields
    _, err = g.ProcessRequest(context.Background(), wideQuery, "")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "exceeds maximum allowed fields")
}

func TestAuthenticationBypass(t *testing.T) {
    // Test that unauthenticated requests are rejected
    // Test that invalid tokens are rejected
    // Test that expired tokens are rejected
}

func TestWebSocketSecurity(t *testing.T) {
    // Test connection authentication
    // Test subscription authorization
    // Test subscription limits
    // Test connection cleanup
}
```

### Security Test Commands

```bash
# Run all security-related tests
go test -v -run "TestMemoryLimits|TestWebSocketAuthentication|TestWebSocketSubscriptionLimits"

# Run memory limit tests
go test -v -run "TestMemoryLimits"

# Run with race detection
go test -race ./...
```

## Production Security Checklist

### ⚠️ Critical Configuration

- [ ] **Enable Production Mode**: Set `ProductionMode: true` 
- [ ] **Configure Memory Limits**: Set appropriate `MemoryLimits`
- [ ] **Configure Query Limits**: Set `QueryLimits` to prevent DoS
- [ ] **WebSocket Authentication**: Replace default `NoOpWebSocketAuthenticator`
- [ ] **Disable Introspection**: Only enable in development environments

### Authentication & Authorization

- [ ] **JWT Validation**: Implement proper token validation
- [ ] **Role-Based Access**: Test role-based authorization
- [ ] **Field-Level Security**: Verify field-level access controls
- [ ] **Session Management**: Implement secure session handling

### Monitoring & Alerting

- [ ] **Error Logging**: Configure error handler for full error capture
- [ ] **Security Metrics**: Track authentication failures and blocked queries
- [ ] **Alerting**: Set up alerts for security violations
- [ ] **Audit Logging**: Log authentication and authorization events

### Network Security

- [ ] **HTTPS Only**: Use TLS for all communication
- [ ] **Security Headers**: Implement security headers middleware
- [ ] **CORS Configuration**: Configure appropriate CORS policies
- [ ] **Rate Limiting**: Implement request rate limiting

### Example Production Configuration

```go
// Complete production configuration
g := &Graphy{
    ProductionMode: true, // ⚠️ CRITICAL: Enable error sanitization
    
    MemoryLimits: &MemoryLimits{
        MaxRequestBodySize:            1024 * 1024,     // 1MB HTTP bodies
        MaxVariableSize:               64 * 1024,       // 64KB variables  
        SubscriptionBufferSize:        100,             // 100-message buffers
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

## Getting Help

### Security Issues

If you discover a security vulnerability:

1. **DO NOT** create a public GitHub issue
2. Email security concerns to the maintainers privately
3. Provide detailed information about the vulnerability
4. Allow time for the issue to be addressed before public disclosure

### Security Questions

For security-related implementation questions:

1. Review this documentation thoroughly
2. Check the example implementations in the repository
3. Join the community discussions
4. Consult with security professionals for complex implementations

**Remember**: Security is a shared responsibility. While go-quickgraph provides robust security foundations, proper configuration and implementation are critical for production deployments.