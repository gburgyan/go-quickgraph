# Error Handling

go-quickgraph provides comprehensive error handling that follows GraphQL specifications while offering production-ready security features and developer-friendly debugging capabilities.

## Table of Contents
- [Basic Error Handling](#basic-error-handling)
- [Error Types](#error-types)
- [Production vs Development Mode](#production-vs-development-mode)
- [Error Handler Interface](#error-handler-interface)
- [Validation Errors](#validation-errors)
- [Panic Recovery](#panic-recovery)
- [WebSocket and Subscription Errors](#websocket-and-subscription-errors)
- [Best Practices](#best-practices)

## Basic Error Handling

GraphQL functions follow Go's standard error return pattern:

```go
// Query with error handling
func GetEmployee(id int) (*Employee, error) {
    employeeMux.RLock()
    defer employeeMux.RUnlock()
    
    for _, emp := range employees {
        if emp.ID == id {
            return emp, nil
        }
    }
    
    return nil, fmt.Errorf("employee with id %d not found", id)
}

// Mutation with error handling
func CreateEmployee(input EmployeeInput) (*Employee, error) {
    if input.Name == "" {
        return nil, errors.New("employee name is required")
    }
    
    if input.Salary < 0 {
        return nil, errors.New("salary cannot be negative")
    }
    
    // Create employee...
    return employee, nil
}
```

Errors are automatically converted to GraphQL-compliant error responses:

```json
{
  "errors": [{
    "message": "employee with id 999 not found",
    "path": ["GetEmployee"]
  }]
}
```

## Error Types

### GraphError

The core error type that provides structured error information:

```go
type GraphError struct {
    Message             string            `json:"message"`
    Locations           []ErrorLocation   `json:"locations,omitempty"`
    Path                []string          `json:"path,omitempty"`
    Extensions          map[string]string `json:"extensions,omitempty"`
    InnerError          error             `json:"-"`
    ProductionMessage   string            `json:"-"`
    SensitiveExtensions map[string]string `json:"-"`
}
```

### Creating GraphErrors

```go
// Basic GraphError
err := NewGraphError("validation failed", pos, "fieldName")

// GraphError with production message
err := NewGraphErrorWithProduction(
    "detailed debug info",      // Development message
    "validation error",         // Production message
    pos,
    "fieldName",
)

// Add extensions for additional context
err.AddExtension("code", "VALIDATION_ERROR")
err.AddExtension("field", "email")

// Add sensitive extensions (filtered in production)
err.AddSensitiveExtension("stack", stackTrace)
err.AddSensitiveExtension("query", rawQuery)
```

### Error Categories

Errors are categorized for better monitoring and handling:

```go
const (
    ErrorCategoryValidation ErrorCategory = "validation"  // Invalid queries, missing variables
    ErrorCategoryExecution  ErrorCategory = "execution"   // Runtime errors
    ErrorCategoryWebSocket  ErrorCategory = "websocket"   // WebSocket protocol errors
    ErrorCategoryHTTP       ErrorCategory = "http"        // HTTP protocol errors
    ErrorCategoryInternal   ErrorCategory = "internal"    // Internal library errors
)
```

## Production vs Development Mode

### Configuration

```go
// Production configuration - CRITICAL for deployments
g := &Graphy{
    ProductionMode: true,  // ⚠️ MUST enable for production
    // ... other configuration
}

// Development configuration (default)
g := &Graphy{
    ProductionMode: false, // Detailed errors for debugging
}
```

### Production Mode Features

In production mode, error messages are sanitized to prevent information disclosure:

| Feature | Development Mode | Production Mode |
|---------|-----------------|-----------------|
| Error Messages | Full details with stack traces | Generic "Internal server error" |
| Panic Details | Function names and values | Hidden from clients |
| Stack Traces | Included in response | Filtered out |
| Sensitive Extensions | Visible | Removed from response |
| Error Handler | Receives full details | Receives full details |

Example production response:
```json
{
  "errors": [{
    "message": "Internal server error",
    "path": ["someQuery"]
  }]
}
```

The same error in development mode:
```json
{
  "errors": [{
    "message": "function someQuery panicked: runtime error: index out of range [5] with length 3",
    "path": ["someQuery"],
    "extensions": {
      "stack": "goroutine 1 [running]:\n...",
      "function_name": "someQuery",
      "panic_value": "runtime error: index out of range [5] with length 3"
    }
  }]
}
```

## Error Handler Interface

Implement custom error logging and monitoring:

```go
type ErrorHandler interface {
    HandleError(
        ctx context.Context,
        category ErrorCategory,
        err error,
        details map[string]interface{},
    )
}

// Example implementation
type MyErrorHandler struct {
    logger *log.Logger
}

func (h *MyErrorHandler) HandleError(
    ctx context.Context,
    category ErrorCategory,
    err error,
    details map[string]interface{},
) {
    // Log based on category
    switch category {
    case ErrorCategoryValidation:
        h.logger.Printf("Validation error: %v", err)
    case ErrorCategoryExecution:
        h.logger.Printf("Execution error: %v, details: %+v", err, details)
    case ErrorCategoryInternal:
        // Alert on internal errors
        h.logger.Printf("CRITICAL: Internal error: %v, details: %+v", err, details)
    }
    
    // Extract GraphError for detailed information
    var gErr GraphError
    if errors.As(err, &gErr) {
        // Access sensitive information for logging
        if stack, ok := gErr.SensitiveExtensions["stack"]; ok {
            h.logger.Printf("Stack trace: %s", stack)
        }
    }
}

// Register the handler
graphy.SetErrorHandler(&MyErrorHandler{logger: log.New(...)})
```

The error handler always receives full error details regardless of production mode, enabling proper monitoring while keeping client responses secure.

## Validation Errors

### Input Validation

Implement the `Validator` interface for custom validation:

```go
type ProductInput struct {
    Name  string  `graphy:"name"`
    Price float64 `graphy:"price"`
}

func (p ProductInput) Validate() error {
    if p.Name == "" {
        return errors.New("product name is required")
    }
    if p.Price <= 0 {
        return errors.New("product price must be positive")
    }
    return nil
}

// Context-aware validation
func (p ProductInput) ValidateWithContext(ctx context.Context) error {
    // Access user permissions from context
    if !hasPermission(ctx, "create_product") {
        return errors.New("insufficient permissions to create product")
    }
    return p.Validate()
}
```

Validation errors include location information:
```json
{
  "errors": [{
    "message": "product price must be positive",
    "locations": [{"line": 3, "column": 15}],
    "path": ["CreateProduct", "input", "price"]
  }]
}
```

### Query Validation

Built-in validations include:
- Unknown fields
- Type mismatches
- Missing required arguments
- Invalid enum values

## Panic Recovery

Functions that panic are automatically recovered and converted to errors:

```go
func DangerousOperation() string {
    // This panic will be caught and converted to an error
    panic("something went wrong")
}
```

In production mode:
```json
{
  "errors": [{
    "message": "Internal server error",
    "path": ["DangerousOperation"]
  }]
}
```

The error handler receives full panic details:
```go
details := map[string]interface{}{
    "stack": "full stack trace...",
    "function_name": "DangerousOperation",
    "panic_value": "something went wrong",
}
```

## WebSocket and Subscription Errors

### Connection Errors

```go
type MyWebSocketAuth struct{}

func (a *MyWebSocketAuth) AuthenticateConnection(
    ctx context.Context,
    payload json.RawMessage,
) (context.Context, error) {
    // Return error to reject connection
    return nil, errors.New("invalid authentication token")
}
```

### Subscription Errors

Errors in subscription streams:

```go
func SubscribeToUpdates(ctx context.Context) (<-chan Update, error) {
    // Initial validation
    if !hasPermission(ctx, "subscribe") {
        return nil, errors.New("subscription access denied")
    }
    
    ch := make(chan Update)
    go func() {
        defer close(ch)
        
        for {
            select {
            case <-ctx.Done():
                return
            case update := <-updates:
                // Errors during streaming are sent as error messages
                if err := validateUpdate(update); err != nil {
                    // This will be sent as an error frame
                    panic(err)
                }
                ch <- update
            }
        }
    }()
    
    return ch, nil
}
```

## Best Practices

### 1. Use Appropriate Error Messages

```go
// ❌ Bad: Exposing internal details
return nil, fmt.Errorf("failed to query database: %v", dbErr)

// ✅ Good: User-friendly message
return nil, errors.New("unable to retrieve employee information")
```

### 2. Leverage GraphError for Rich Context

```go
func ProcessOrder(orderID string) (*Order, error) {
    order, err := db.GetOrder(orderID)
    if err != nil {
        // Wrap with context
        return nil, AugmentGraphError(
            err,
            "order not found",
            lexer.Position{},
            "ProcessOrder", "orderID",
        )
    }
    return order, nil
}
```

### 3. Implement Comprehensive Error Handling

```go
// Custom error types for different scenarios
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}

// Use in functions
func UpdateProduct(id int, input ProductInput) (*Product, error) {
    if input.Price < 0 {
        return nil, ValidationError{
            Field:   "price",
            Message: "price cannot be negative",
        }
    }
    // ...
}
```

### 4. Monitor Errors in Production

```go
type MetricsErrorHandler struct {
    metrics *prometheus.CounterVec
}

func (h *MetricsErrorHandler) HandleError(
    ctx context.Context,
    category ErrorCategory,
    err error,
    details map[string]interface{},
) {
    // Increment metrics
    h.metrics.WithLabelValues(
        string(category),
        getErrorType(err),
    ).Inc()
    
    // Log for debugging
    if category == ErrorCategoryInternal {
        log.Printf("Internal error: %+v", details)
    }
}
```

### 5. Test Error Scenarios

```go
func TestErrorHandling(t *testing.T) {
    g := &Graphy{ProductionMode: true}
    
    // Register function that returns error
    g.RegisterQuery(ctx, "failingQuery", func() (string, error) {
        return "", errors.New("expected error")
    })
    
    result, err := g.ProcessRequest(ctx, `{failingQuery}`, "{}")
    
    // Verify error is properly formatted
    assert.Error(t, err)
    assert.Contains(t, result, "errors")
    assert.NotContains(t, result, "expected error") // Sanitized in production
}
```

## Summary

go-quickgraph's error handling provides:
- **GraphQL Compliance**: Errors follow the GraphQL specification
- **Production Safety**: Automatic sanitization of sensitive information
- **Developer Experience**: Detailed errors during development
- **Monitoring Support**: Error handlers receive full details for logging
- **Flexible Extension**: Custom error types and handlers
- **Security by Default**: Production mode prevents information disclosure

Always enable `ProductionMode: true` before deploying to production environments to ensure error messages don't leak sensitive implementation details to clients.