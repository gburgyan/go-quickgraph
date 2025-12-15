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
    CORSSettings: &quickgraph.CORSSettings{
        AllowedOrigins:   []string{"https://myapp.com"},
        AllowedMethods:   []string{"POST", "OPTIONS"},
        AllowedHeaders:   []string{"Content-Type", "Authorization"},
        AllowCredentials: true,
        MaxAge:           3600,
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
- ✅ **CORS Configuration** - Flexible cross-origin access control
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

## CORS Configuration

Cross-Origin Resource Sharing (CORS) is essential for allowing web browsers to access your GraphQL API from different domains. go-quickgraph provides comprehensive CORS configuration through the `CORSSettings` struct.

### Basic CORS Setup

```go
g := &Graphy{
    // Enable default CORS settings (allows all origins)
    CORSSettings: DefaultCORSSettings(),
}
```

### Custom CORS Configuration

```go
g := &Graphy{
    CORSSettings: &CORSSettings{
        AllowedOrigins:   []string{"https://myapp.com", "https://admin.myapp.com"},
        AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
        AllowedHeaders:   []string{"Content-Type", "Authorization"},
        ExposedHeaders:   []string{"X-Custom-Header"},
        AllowCredentials: true,
        MaxAge:           3600, // 1 hour cache for preflight requests
        EnableForAllResponses: true, // Add CORS headers to all responses, not just OPTIONS
    },
}
```

### CORS Configuration Options

```go
type CORSSettings struct {
    // AllowedOrigins specifies allowed origins for CORS requests
    // Use "*" to allow all origins, or specify exact origins
    AllowedOrigins []string

    // AllowedMethods specifies allowed HTTP methods
    // Default: ["GET", "POST", "OPTIONS"]
    AllowedMethods []string

    // AllowedHeaders specifies allowed request headers
    // Default: ["Content-Type", "Authorization"]
    AllowedHeaders []string

    // ExposedHeaders specifies response headers browsers can access
    // Optional - only set if you need to expose custom headers
    ExposedHeaders []string

    // AllowCredentials indicates whether credentials are allowed
    // Important: Cannot be true if AllowedOrigins contains "*"
    AllowCredentials bool

    // MaxAge specifies preflight cache duration in seconds
    // Default: 86400 (24 hours)
    MaxAge int

    // EnableForAllResponses adds CORS headers to all responses
    // If false, only OPTIONS preflight requests get CORS headers
    EnableForAllResponses bool
}
```

### Production CORS Examples

#### GraphQL Playground Development
```go
// Allow GraphQL Playground access during development
g := &Graphy{
    CORSSettings: &CORSSettings{
        AllowedOrigins: []string{"*"}, // Allow all origins
        AllowedMethods: []string{"GET", "POST", "OPTIONS"},
        AllowedHeaders: []string{"Content-Type", "Authorization"},
    },
}
```

#### Production Web Application
```go
// Strict CORS for production
g := &Graphy{
    CORSSettings: &CORSSettings{
        AllowedOrigins:        []string{"https://myapp.com"},
        AllowedMethods:        []string{"POST", "OPTIONS"},
        AllowedHeaders:        []string{"Content-Type", "Authorization"},
        AllowCredentials:      true,
        MaxAge:                3600,
        EnableForAllResponses: true,
    },
}
```

#### Multiple Frontend Environments
```go
// Support multiple frontend deployments
g := &Graphy{
    CORSSettings: &CORSSettings{
        AllowedOrigins: []string{
            "https://app.mycompany.com",
            "https://staging.mycompany.com", 
            "https://admin.mycompany.com",
        },
        AllowedMethods:        []string{"GET", "POST", "OPTIONS"},
        AllowedHeaders:        []string{"Content-Type", "Authorization", "X-API-Key"},
        ExposedHeaders:        []string{"X-Request-ID", "X-Rate-Limit-Remaining"},
        AllowCredentials:      true,
        EnableForAllResponses: true,
    },
}
```

### CORS Security Considerations

#### 1. Avoid Wildcard Origins in Production
```go
// ❌ Dangerous in production
CORSSettings: &CORSSettings{
    AllowedOrigins:   []string{"*"},
    AllowCredentials: true, // This combination is invalid
}

// ✅ Specify exact origins
CORSSettings: &CORSSettings{
    AllowedOrigins:   []string{"https://myapp.com"},
    AllowCredentials: true,
}
```

#### 2. Minimize Exposed Headers
```go
// ❌ Don't expose sensitive headers unnecessarily
ExposedHeaders: []string{"Authorization", "X-Internal-Token"}

// ✅ Only expose what the frontend needs
ExposedHeaders: []string{"X-Request-ID"}
```

#### 3. Use HTTPS Origins
```go
// ❌ HTTP origins in production
AllowedOrigins: []string{"http://myapp.com"}

// ✅ HTTPS only in production
AllowedOrigins: []string{"https://myapp.com"}
```

### Disabling CORS

```go
// CORS is disabled by default
g := &Graphy{
    CORSSettings: nil, // No CORS headers will be added
}
```

When CORS is disabled:
- No `Access-Control-*` headers are added to responses
- Browsers will block cross-origin requests
- Useful for same-origin deployments or API-only servers

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
        Authorization string `graphy:"authorization"`
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

## Production Error Sanitization

### ⚠️ Critical: Preventing Information Disclosure

One of the most critical security features in go-quickgraph is **error sanitization** in production mode. Without proper configuration, GraphQL errors can expose sensitive internal information including:

- Stack traces with file paths and line numbers
- Function names and internal implementation details  
- Memory addresses and pointer information
- Runtime error messages with system internals
- Reflection call details and Go internals

### Automatic Error Sanitization

```go
g := &quickgraph.Graphy{
    ProductionMode: true, // 🔒 CRITICAL: Enables error sanitization
}
```

When `ProductionMode: true` is set:

- **Sensitive error messages** are replaced with generic messages
- **Sensitive extensions** (stack traces, function names) are filtered from client responses
- **Production messages** override development messages in error responses
- **Internal error details** are still logged via error handlers for debugging

### Error Message Examples

#### Development Mode (ProductionMode: false)
```json
{
  "errors": [{
    "message": "function getUserProfile panicked: runtime error: nil pointer dereference",
    "path": ["getUserProfile"],
    "extensions": {
      "stack": "goroutine 1 [running]:\nmain.go:123\nhandler.go:456",
      "function_name": "getUserProfile", 
      "panic_value": "runtime error: nil pointer dereference"
    }
  }]
}
```

#### Production Mode (ProductionMode: true)
```json
{
  "errors": [{
    "message": "Internal server error",
    "path": ["getUserProfile"]
  }]
}
```

### Built-in Sensitive Pattern Detection

go-quickgraph automatically detects and sanitizes messages containing:

#### Runtime Errors & Panics
- `panic:` messages
- `runtime error:` messages  
- `stack trace` references
- `goroutine` information

#### Go Internals
- `reflect.` method calls
- Source locations like `main.go:123`
- Memory addresses like `0x40e020`
- Function signatures like `func(string, int)`
- Method calls like `pkg.Method()`

#### System Information
- File system paths (`/app/src/handlers.go`)
- Windows paths (`C:\\Program Files\\app\\main.go`)
- Interface type references (`interface {}`)
- Nil pointer and slice bounds errors

### Production Error Creation Functions

#### NewGraphErrorProduction
```go
// Creates sanitized errors for production use
err := quickgraph.NewGraphErrorProduction(
    "function getUserData panicked: nil pointer", // Original message
    pos,     // Position info
    "users", "profile", // Path segments
)
// Result: "An error occurred while processing the request"
```

#### NewGraphErrorWithProduction 
```go
// Creates errors with separate dev/prod messages
err := quickgraph.NewGraphErrorWithProduction(
    "function getUserData panicked: nil pointer", // Development message
    "Internal server error",                     // Production message
    pos,
    "users", "profile",
)
```

#### NewPanicError
```go
// Automatically creates sanitized panic errors
err := quickgraph.NewPanicError(
    "getUserData",           // Function name
    "nil pointer dereference", // Panic value
    pos,
    "users", "profile",
)
// Development: "function getUserData panicked: nil pointer dereference"
// Production: "Internal server error"
```

### Sensitive Extensions Filtering

```go
// Add sensitive debugging information
gErr := quickgraph.GraphError{
    Message: "Database connection failed",
    ProductionMessage: "Internal server error",
    Extensions: map[string]string{
        "query_id": "12345",        // Safe - included in production
        "timestamp": "2024-01-01",  // Safe - included in production
    },
    SensitiveExtensions: map[string]string{
        "stack_trace":   "goroutine 1 [running]...",  // Filtered in production
        "function_name": "connectToDatabase",          // Filtered in production
        "db_host":       "internal-db.company.com",   // Filtered in production
        "panic_value":   "connection refused",        // Filtered in production
    },
}

// In production mode, only "query_id" and "timestamp" are sent to client
```

### JSON Marshaling for Production

```go
// Development marshaling - includes all information
jsonBytes, err := gErr.MarshalJSON()

// Production marshaling - filters sensitive information
jsonBytes, err := gErr.MarshalJSONProduction()

// Framework automatically uses production marshaling when ProductionMode: true
```

### Custom Error Sanitization

For advanced use cases, customize the error sanitization patterns:

```go
// Create custom sanitizer
customSanitizer := &quickgraph.ErrorSanitizer{
    RegexPatterns: []*regexp.Regexp{
        regexp.MustCompile(`\bsecret\b`),     // Flag "secret" as sensitive
        regexp.MustCompile(`\bapi[_-]key\b`), // Flag API key references
        regexp.MustCompile(`\bpassword\b`),   // Flag password references
    },
    StringPatterns: []string{
        "confidential",    // Simple substring matching
        "internal-only",
    },
    CustomSanitizer: func(message string) bool {
        // Custom logic for sensitivity detection
        return strings.Contains(message, "SENSITIVE_DATA")
    },
}

// Apply custom sanitizer (primarily for testing)
quickgraph.SetErrorSanitizer(customSanitizer)
```

### Error Handler Integration

```go
// Error handlers receive full error details regardless of production mode
g.SetErrorHandler(quickgraph.ErrorHandlerFunc(func(ctx context.Context, category quickgraph.ErrorCategory, err error, details map[string]interface{}) {
    // Log complete error information for internal debugging
    logger.WithFields(logrus.Fields{
        "category": string(category),
        "error":    err.Error(),           // Full error message
        "details":  details,              // All extensions including sensitive ones
        "path":     details["path"],
        "function": details["function_name"], // Available even in production
        "stack":    details["stack"],         // Full stack trace available
    }).Error("GraphQL error occurred")
    
    // Send alerts for critical errors
    if strings.Contains(err.Error(), "panic") {
        alertManager.SendAlert("GraphQL Panic", err.Error())
    }
}))
```

### Testing Error Sanitization

```go
func TestErrorSanitization(t *testing.T) {
    // Test development mode shows details
    g := &quickgraph.Graphy{ProductionMode: false}
    g.RegisterQuery(context.Background(), "testPanic", panicFunc)
    
    result, err := g.ProcessRequest(context.Background(), `{testPanic}`, "{}")
    assert.Contains(t, result, "testPanic panicked")    // Function name visible
    assert.Contains(t, result, "panic details")        // Panic details visible
    assert.Contains(t, result, "stack")               // Stack trace in extensions
    
    // Test production mode sanitizes details
    g.ProductionMode = true
    result, err = g.ProcessRequest(context.Background(), `{testPanic}`, "{}")
    assert.NotContains(t, result, "testPanic panicked") // Function name hidden
    assert.NotContains(t, result, "panic details")     // Panic details hidden
    assert.NotContains(t, result, "stack")             // Stack trace filtered
    assert.Contains(t, result, "Internal server error") // Generic message shown
}
```

### Error Sanitization Best Practices

#### 1. Always Enable Production Mode
```go
// ❌ Dangerous - exposes internal details
g := &quickgraph.Graphy{
    ProductionMode: false, // Never use false in production
}

// ✅ Secure - sanitizes error responses
g := &quickgraph.Graphy{
    ProductionMode: true, // Always true in production
}
```

#### 2. Use Appropriate Error Creation Functions
```go
// ❌ Manual error creation without sanitization
return nil, fmt.Errorf("database connection failed: %s", internalDetails)

// ✅ Use production-aware error functions
return nil, quickgraph.NewGraphErrorWithProduction(
    fmt.Sprintf("database connection failed: %s", internalDetails), // Dev message
    "Database temporarily unavailable", // Production message
    pos, "getUserData",
)
```

#### 3. Add Sensitive Information to SensitiveExtensions
```go
// ❌ Sensitive data in regular extensions
gErr.Extensions["db_password"] = password
gErr.Extensions["api_key"] = apiKey

// ✅ Sensitive data in SensitiveExtensions
gErr.AddSensitiveExtension("db_password", password)
gErr.AddSensitiveExtension("api_key", apiKey)
```

#### 4. Monitor Error Handler Output
```go
// Error handlers receive full details for monitoring
g.SetErrorHandler(quickgraph.ErrorHandlerFunc(func(ctx context.Context, category quickgraph.ErrorCategory, err error, details map[string]interface{}) {
    // Full error information available for internal logging
    // Use this for debugging, monitoring, and alerting
    internalLogger.Log(err.Error(), details)
    
    // Track error patterns
    if strings.Contains(err.Error(), "panic") {
        metrics.IncrementCounter("graphql.panics")
    }
}))
```

### Security Impact

**Without Error Sanitization** (ProductionMode: false):
- Attackers can see internal file structures
- Function names reveal implementation details
- Stack traces show code paths and vulnerabilities
- Memory addresses can aid in memory exploitation
- Database connection strings may be exposed

**With Error Sanitization** (ProductionMode: true):
- Clients receive only generic error messages
- Internal details are hidden from attackers
- Debugging information still available via error handlers
- Maintains security while preserving functionality

### Configuration Examples

#### Development Environment
```go
// Show detailed errors for debugging
g := &quickgraph.Graphy{
    ProductionMode: false,
    EnableIntrospection: true,
}
```

#### Staging Environment
```go
// Production-like error handling with detailed logging
g := &quickgraph.Graphy{
    ProductionMode: true,
    EnableIntrospection: false,
}
g.SetErrorHandler(DetailedLoggingErrorHandler)
```

#### Production Environment
```go
// Maximum security with monitoring
g := &quickgraph.Graphy{
    ProductionMode: true,
    EnableIntrospection: false,
    QueryLimits: productionQueryLimits,
    MemoryLimits: productionMemoryLimits,
}
g.SetErrorHandler(MonitoringErrorHandler)
```

### Troubleshooting Production Errors

Since production mode sanitizes client-facing errors, use error handlers for debugging:

```go
// Comprehensive error handler for production debugging
type ProductionErrorHandler struct {
    logger *logrus.Logger
    alerts AlertManager
}

func (p *ProductionErrorHandler) HandleError(ctx context.Context, category quickgraph.ErrorCategory, err error, details map[string]interface{}) {
    // Create structured log entry
    entry := p.logger.WithFields(logrus.Fields{
        "category": string(category),
        "error":    err.Error(),
        "path":     details["path"],
        "query":    details["query"],
        "user":     getUserFromContext(ctx),
    })
    
    // Add available debugging information
    if stack, ok := details["stack"]; ok {
        entry = entry.WithField("stack_trace", stack)
    }
    if function, ok := details["function_name"]; ok {
        entry = entry.WithField("function", function)
    }
    
    // Log with appropriate level
    switch category {
    case quickgraph.ErrorCategoryInternal:
        entry.Error("Internal GraphQL error")
        p.alerts.SendAlert("GraphQL Internal Error", err.Error())
    case quickgraph.ErrorCategoryExecution:
        entry.Warn("GraphQL execution error")
    case quickgraph.ErrorCategoryValidation:
        entry.Info("GraphQL validation error")
    }
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

- [ ] **Enable Production Mode**: Set `ProductionMode: true` (CRITICAL for error sanitization)
- [ ] **Configure Memory Limits**: Set appropriate `MemoryLimits`
- [ ] **Configure Query Limits**: Set `QueryLimits` to prevent DoS
- [ ] **WebSocket Authentication**: Replace default `NoOpWebSocketAuthenticator`
- [ ] **Disable Introspection**: Only enable in development environments
- [ ] **Error Handler Setup**: Implement production error handler for debugging
- [ ] **Verify Error Sanitization**: Test that sensitive information is not exposed

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

## Common GraphQL Vulnerabilities

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