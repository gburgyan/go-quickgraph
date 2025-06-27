# Function Patterns

This guide covers all the patterns and best practices for registering and implementing GraphQL functions in go-quickgraph.

## Table of Contents
- [Function Registration Methods](#function-registration-methods)
- [Parameter Patterns](#parameter-patterns)
- [Return Value Patterns](#return-value-patterns)
- [Context and Authentication](#context-and-authentication)
- [Advanced Patterns](#advanced-patterns)
- [Subscription Patterns](#subscription-patterns)
- [Error Handling](#error-handling)
- [Performance Patterns](#performance-patterns)
- [Security Considerations](#security-considerations)
- [Best Practices](#best-practices)

## Function Registration Methods

go-quickgraph provides several ways to register functions as GraphQL operations:

### Basic Registration Methods

```go
// Register a query (read operation)
graphy.RegisterQuery(ctx, "getUser", getUserFunc, "userId")

// Register a mutation (write operation)
graphy.RegisterMutation(ctx, "createUser", createUserFunc, "input")

// Register a subscription (real-time updates)
graphy.RegisterSubscription(ctx, "userUpdates", userUpdatesFunc, "userId")
```

### Advanced Registration with FunctionDefinition

For full control over function behavior, use `RegisterFunction`:

```go
graphy.RegisterFunction(ctx, quickgraph.FunctionDefinition{
    Name:             "complexOperation",
    Function:         myFunction,
    ParameterNames:   []string{"param1", "param2"},
    ParameterMode:    quickgraph.NamedParams,
    Mode:             quickgraph.ModeQuery,
    Description:      strPtr("A complex operation with custom configuration"),
    DeprecatedReason: strPtr("Use newComplexOperation instead"),
    ReturnUnionName:  "MyCustomUnion",
})
```

## Parameter Patterns

go-quickgraph supports multiple parameter handling modes:

### AutoDetect Mode (Default)

The library automatically determines the parameter mode based on your function signature:

```go
// Single struct parameter → StructParams mode
func getUser(ctx context.Context, input GetUserInput) (*User, error)

// Multiple parameters → NamedParams mode with provided names
func getUser(ctx context.Context, id string, includeDeleted bool) (*User, error)
```

### StructParams Mode

Best for complex inputs with many fields:

```go
type CreateProductInput struct {
    Name        string   `json:"name"`
    Description *string  `json:"description,omitempty"`
    Price       float64  `json:"price"`
    Categories  []string `json:"categories"`
}

func createProduct(ctx context.Context, input CreateProductInput) (*Product, error) {
    // Validation happens automatically if input implements Validator
    // All fields are available as a single struct
}

// Register with automatic struct parameter detection
graphy.RegisterMutation(ctx, "createProduct", createProduct)
```

### NamedParams Mode

Best for simple functions with few parameters:

```go
func updateUserStatus(ctx context.Context, userId string, active bool, reason *string) (*User, error) {
    // Parameters are passed individually
}

// Register with explicit parameter names
graphy.RegisterMutation(ctx, "updateUserStatus", updateUserStatus, "userId", "active", "reason")
```

### PositionalParams Mode

For backward compatibility or when parameter names don't matter:

```go
func legacyOperation(ctx context.Context, arg1 string, arg2 int) (string, error) {
    // Parameters accessed as arg1, arg2 in GraphQL
}

// Register with positional parameter mode
graphy.RegisterFunction(ctx, quickgraph.FunctionDefinition{
    Name:          "legacyOperation",
    Function:      legacyOperation,
    ParameterMode: quickgraph.PositionalParams,
    Mode:          quickgraph.ModeQuery,
})
```

## Return Value Patterns

### Single Value Returns

The most common pattern - return a single value and optional error:

```go
func getUser(ctx context.Context, id string) (*User, error) {
    user, err := db.GetUser(id)
    if err != nil {
        return nil, err
    }
    return user, nil
}
```

### Union Type Returns

Return multiple pointer types where only one should be non-nil:

```go
func search(ctx context.Context, query string) (*UserResult, *ProductResult, *OrderResult, error) {
    // Return only one non-nil result based on search type
    if isUserSearch(query) {
        return &UserResult{Users: users}, nil, nil, nil
    }
    if isProductSearch(query) {
        return nil, &ProductResult{Products: products}, nil, nil
    }
    // ...
}
```

### Custom Union Names

Override the default union type name:

```go
graphy.RegisterFunction(ctx, quickgraph.FunctionDefinition{
    Name:            "search",
    Function:        search,
    ReturnUnionName: "SearchResult", // Instead of default "SearchUnion"
    // ...
})
```

### Any Return with Type Override

Return `any` but specify possible types for schema generation:

```go
func getDynamicContent(ctx context.Context, id string) (any, error) {
    // Returns different types based on ID
}

graphy.RegisterFunction(ctx, quickgraph.FunctionDefinition{
    Name:               "getDynamicContent",
    Function:           getDynamicContent,
    ReturnAnyOverride:  []any{Article{}, Video{}, Image{}},
    // ...
})
```

## Context and Authentication

### Context as First Parameter

Functions can accept `context.Context` to access request-scoped data:

```go
func getCurrentUser(ctx context.Context) (*User, error) {
    userID, ok := ctx.Value("userID").(string)
    if !ok {
        return nil, errors.New("not authenticated")
    }
    return getUserByID(userID)
}
```

### Authentication Patterns

#### Middleware-based Authentication

```go
// In your HTTP handler setup
graphHandler := authMiddleware(graph.HttpHandler())

func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        if userID, err := validateToken(token); err == nil {
            ctx := context.WithValue(r.Context(), "userID", userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        } else {
            next.ServeHTTP(w, r)
        }
    })
}
```

#### Function-level Authorization

```go
func deleteUser(ctx context.Context, userID string) (*User, error) {
    currentUser := ctx.Value("user").(*User)
    if !currentUser.IsAdmin {
        return nil, errors.New("admin access required")
    }
    // Proceed with deletion
}
```

## Advanced Patterns

### Method-based Field Resolvers

Methods on types are automatically exposed as GraphQL fields:

```go
type Product struct {
    ID         string
    CategoryID string
}

// This becomes a 'category' field in GraphQL that's resolved on demand
func (p *Product) Category(ctx context.Context) (*Category, error) {
    return getCategoryByID(ctx, p.CategoryID)
}

// This becomes a 'reviews' field with parameters
func (p *Product) Reviews(ctx context.Context, limit *int) ([]Review, error) {
    return getProductReviews(ctx, p.ID, limit)
}
```

### Type Discovery for Interfaces

Support runtime type resolution for GraphQL interfaces:

```go
type Employee interface {
    GetID() string
    GetName() string
}

type Developer struct {
    ID               string
    Name             string
    ProgrammingLangs []string
}

func (d *Developer) GetID() string   { return d.ID }
func (d *Developer) GetName() string { return d.Name }

// Enable type discovery
func (d *Developer) ActualType() any {
    return d // Return the concrete type
}
```

### Input Validation

Implement the `Validator` interface for automatic validation:

```go
type CreateUserInput struct {
    Email    string
    Password string
    Age      int
}

func (i CreateUserInput) Validate() error {
    if !isValidEmail(i.Email) {
        return errors.New("invalid email format")
    }
    if len(i.Password) < 8 {
        return errors.New("password must be at least 8 characters")
    }
    if i.Age < 18 {
        return errors.New("must be 18 or older")
    }
    return nil
}
```

### Custom Scalars in Functions

Register and use custom scalar types:

```go
type EmailAddress string

func (e EmailAddress) GraphQLTypeName() string { return "EmailAddress" }

func (e EmailAddress) Serialize() (any, error) {
    return string(e), nil
}

func (e *EmailAddress) Parse(input any) error {
    str, ok := input.(string)
    if !ok {
        return errors.New("email must be a string")
    }
    if !isValidEmail(str) {
        return errors.New("invalid email format")
    }
    *e = EmailAddress(str)
    return nil
}

// Use in functions
func updateEmail(ctx context.Context, email EmailAddress) (*User, error) {
    // Email is already validated by Parse method
}
```

## Subscription Patterns

### Basic Channel-based Subscription

```go
func userUpdates(ctx context.Context, userID string) (<-chan *User, error) {
    // Validate subscription request
    if !canSubscribeToUser(ctx, userID) {
        return nil, errors.New("unauthorized")
    }

    ch := make(chan *User, 10) // Buffer to prevent blocking
    
    go func() {
        defer close(ch)
        
        // Subscribe to internal event system
        sub := eventBus.Subscribe("user.updated." + userID)
        defer sub.Unsubscribe()
        
        for {
            select {
            case <-ctx.Done():
                return // Client disconnected
            case event := <-sub.Events:
                user := event.(*User)
                select {
                case ch <- user:
                    // Sent successfully
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    
    return ch, nil
}
```

### Broadcasting Pattern

Broadcast updates to multiple subscribers:

```go
var (
    subscribers = make(map[string][]chan<- *Product)
    subMutex    sync.RWMutex
)

func productUpdates(ctx context.Context, categoryID string) (<-chan *Product, error) {
    ch := make(chan *Product, 10)
    
    // Register subscriber
    subMutex.Lock()
    subscribers[categoryID] = append(subscribers[categoryID], ch)
    subMutex.Unlock()
    
    // Clean up on disconnect
    go func() {
        <-ctx.Done()
        subMutex.Lock()
        defer subMutex.Unlock()
        
        // Remove this channel from subscribers
        subs := subscribers[categoryID]
        for i, sub := range subs {
            if sub == ch {
                subscribers[categoryID] = append(subs[:i], subs[i+1:]...)
                break
            }
        }
        close(ch)
    }()
    
    return ch, nil
}

// Broadcast updates when products change
func broadcastProductUpdate(product *Product) {
    subMutex.RLock()
    subs := subscribers[product.CategoryID]
    subMutex.RUnlock()
    
    for _, ch := range subs {
        select {
        case ch <- product:
            // Sent successfully
        default:
            // Channel full, skip this subscriber
        }
    }
}
```

### WebSocket Authentication

Implement custom WebSocket authentication:

```go
type MyAuthenticator struct{}

func (a *MyAuthenticator) AuthenticateWebSocket(r *http.Request) (context.Context, error) {
    token := r.URL.Query().Get("token")
    if token == "" {
        token = r.Header.Get("Authorization")
    }
    
    userID, err := validateToken(token)
    if err != nil {
        return nil, err
    }
    
    ctx := context.WithValue(r.Context(), "userID", userID)
    return ctx, nil
}

// Configure in graphy
graphy.SetWebSocketAuthenticator(&MyAuthenticator{})
```

## Error Handling

### GraphQL Error Responses

Errors are automatically converted to GraphQL error format:

```go
func createUser(ctx context.Context, input CreateUserInput) (*User, error) {
    if exists, _ := userExists(input.Email); exists {
        return nil, errors.New("user with this email already exists")
    }
    // Error appears in GraphQL response errors array
}
```

### Production vs Development Mode

Configure error detail exposure:

```go
// Development mode - full error details
graphy := quickgraph.New()
graphy.SetProductionMode(false)

// Production mode - sanitized errors
graphy := quickgraph.New()
graphy.SetProductionMode(true)
graphy.SetErrorHandler(func(ctx context.Context, err error, callInfo quickgraph.CallInfo) {
    // Log full error details server-side
    log.Printf("GraphQL error: %v, Query: %s", err, callInfo.Query)
})
```

### Panic Recovery

Panics are automatically recovered and converted to errors:

```go
func riskyOperation(ctx context.Context) (string, error) {
    // If this panics, it becomes a GraphQL error
    return processData(), nil
}
```

## Performance Patterns

### Lazy Field Loading

Load expensive data only when requested:

```go
type User struct {
    ID    string
    Name  string
    // Don't load stats unless requested
}

func (u *User) Stats(ctx context.Context) (*UserStats, error) {
    // This runs only when 'stats' field is queried
    return calculateUserStats(u.ID)
}
```

### Request-scoped Caching

Cache data within a single request:

```go
type requestCache struct {
    users map[string]*User
    mu    sync.RWMutex
}

func getUserCached(ctx context.Context, id string) (*User, error) {
    cache := ctx.Value("cache").(*requestCache)
    
    cache.mu.RLock()
    if user, ok := cache.users[id]; ok {
        cache.mu.RUnlock()
        return user, nil
    }
    cache.mu.RUnlock()
    
    // Load from database
    user, err := db.GetUser(id)
    if err != nil {
        return nil, err
    }
    
    cache.mu.Lock()
    cache.users[id] = user
    cache.mu.Unlock()
    
    return user, nil
}
```

### Selective Field Population

Avoid loading data that might not be used:

```go
type Product struct {
    ID          string
    Name        string
    Description string
    // Expensive fields loaded on demand
}

func (p *Product) Images(ctx context.Context) ([]Image, error) {
    // Load images only when requested
    return loadProductImages(p.ID)
}

func (p *Product) Reviews(ctx context.Context, first *int) ([]Review, error) {
    // Load reviews with pagination
    limit := 10
    if first != nil {
        limit = *first
    }
    return loadProductReviews(p.ID, limit)
}
```

## Security Considerations

### Input Validation

Always validate inputs, especially for mutations:

```go
type UpdatePasswordInput struct {
    CurrentPassword string
    NewPassword     string
}

func (i UpdatePasswordInput) Validate() error {
    if len(i.NewPassword) < 12 {
        return errors.New("password must be at least 12 characters")
    }
    if i.NewPassword == i.CurrentPassword {
        return errors.New("new password must be different")
    }
    return nil
}
```

### Memory Limits

Configure memory limits to prevent DoS attacks:

```go
graphy.SetMemoryLimits(quickgraph.MemoryLimits{
    MaxHTTPBodySize:      10 * 1024 * 1024, // 10MB
    MaxVariableSize:      1024 * 1024,      // 1MB
    MaxSubscriptionBuffer: 1000,             // messages
})
```

### Authorization Patterns

Implement field-level authorization:

```go
func (u *User) Email(ctx context.Context) (*string, error) {
    currentUser := ctx.Value("user").(*User)
    if currentUser.ID != u.ID && !currentUser.IsAdmin {
        return nil, nil // Return nil for unauthorized access
    }
    return &u.email, nil
}
```

### WebSocket Security

Limit subscriptions per connection:

```go
graphy.SetQueryLimits(quickgraph.QueryLimits{
    MaxSubscriptionsPerConnection: 10,
})
```

## Best Practices

### 1. Organization Strategies

Group related handlers in dedicated files:

```go
// handlers/user.go
func RegisterUserHandlers(ctx context.Context, graphy *quickgraph.Graphy) {
    graphy.RegisterQuery(ctx, "getUser", getUser, "id")
    graphy.RegisterQuery(ctx, "searchUsers", searchUsers, "query")
    graphy.RegisterMutation(ctx, "createUser", createUser)
    graphy.RegisterMutation(ctx, "updateUser", updateUser)
    graphy.RegisterSubscription(ctx, "userUpdates", userUpdates, "userId")
}
```

### 2. Testing Approaches

Test functions independently:

```go
func TestGetUser(t *testing.T) {
    ctx := context.WithValue(context.Background(), "userID", "admin")
    
    user, err := getUser(ctx, "123")
    assert.NoError(t, err)
    assert.Equal(t, "123", user.ID)
}
```

### 3. Common Pitfalls to Avoid

1. **Not handling context cancellation** in subscriptions
2. **Forgetting to close channels** in subscription goroutines
3. **Not buffering subscription channels** (can cause deadlocks)
4. **Returning sensitive data** in production error messages
5. **Not validating inputs** before processing
6. **Loading all data upfront** instead of using lazy loading
7. **Not implementing proper authentication** for subscriptions

### 4. Production Checklist

- [ ] Enable production mode for error sanitization
- [ ] Configure memory limits
- [ ] Set up proper authentication middleware
- [ ] Implement request logging and monitoring
- [ ] Add input validation for all mutations
- [ ] Test subscription cleanup on disconnect
- [ ] Configure WebSocket authentication
- [ ] Set appropriate subscription limits
- [ ] Implement proper error handling
- [ ] Use context for cancellation propagation

### 5. Naming Conventions

- Use camelCase for GraphQL operation names
- Prefix mutations with verbs (create, update, delete)
- Use descriptive names for subscription operations
- Keep parameter names consistent with field names

### 6. Type Registration

Always register types that aren't directly returned:

```go
// Register types used in unions or interfaces
graphy.RegisterType(ctx, Developer{})
graphy.RegisterType(ctx, Manager{})

// Register custom scalars
graphy.RegisterType(ctx, EmailAddress(""))
graphy.RegisterType(ctx, Money{})
```

## Summary

go-quickgraph's function patterns provide a flexible, type-safe way to build GraphQL APIs in Go. By following these patterns and best practices, you can create maintainable, performant, and secure GraphQL services that leverage Go's strengths while providing an excellent developer experience.

For more examples, see the [sample application](../../go-quickgraph-sample) and [test files](../tests) in the repository.