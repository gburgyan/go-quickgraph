# Core Concepts

This guide explains the fundamental concepts behind go-quickgraph's code-first approach to GraphQL.

## The Code-First Philosophy

go-quickgraph inverts the traditional GraphQL development workflow:

### Traditional Schema-First
```
GraphQL Schema → Code Generation → Implementation → Maintenance
```

### go-quickgraph Code-First
```
Go Code → Automatic Schema Generation → Ready to Use
```

**Why This Matters:**
- Your Go code is the single source of truth
- No need to keep schema and code in sync
- Full compile-time type safety
- Natural Go development workflow

## Core Components

### 1. The Graphy Object

The `Graphy` struct is your GraphQL server. It manages:
- Function registration
- Type discovery and caching
- Request processing
- Schema generation

```go
g := quickgraph.Graphy{
    EnableTiming: true,           // Optional: request timing
    RequestCache: &MyCache{},     // Optional: parsed query caching
    QueryLimits:  &QueryLimits{}, // Optional: DoS protection
}
```

### 2. Function Registration

Functions become GraphQL operations through registration:

```go
// Queries (read operations)
g.RegisterQuery(ctx, "user", GetUser, "id")
g.RegisterQuery(ctx, "users", GetAllUsers)

// Mutations (write operations)  
g.RegisterMutation(ctx, "createUser", CreateUser, "input")

// Subscriptions (real-time streams)
g.RegisterSubscription(ctx, "userUpdates", SubscribeToUsers, "filter")
```

### 3. Automatic Type Discovery

go-quickgraph uses Go reflection to discover types:

```go
type User struct {
    ID    int      `graphy:"id"`
    Name  string   `graphy:"name"`
    Posts []Post   `graphy:"posts"`
}

// This automatically becomes:
// type User {
//   id: Int!
//   name: String!
//   posts: [Post!]!
// }
```

## How It Works Under the Hood

### 1. Request Processing Pipeline

```
GraphQL Request → Parse → Validate → Execute → Response
```

1. **Parse**: Convert GraphQL query to internal representation
2. **Validate**: Check against registered functions and types
3. **Execute**: Call your Go functions with proper parameters
4. **Response**: Format results as GraphQL response

### 2. Type Mapping

go-quickgraph automatically maps Go types to GraphQL types:

| Go Type | GraphQL Type | Notes |
|---------|--------------|-------|
| `string` | `String!` | Non-nil pointer = nullable |
| `*string` | `String` | Pointer = nullable |
| `int`, `int32`, `int64` | `Int!` | All int types → GraphQL Int |
| `float32`, `float64` | `Float!` | All float types → GraphQL Float |
| `bool` | `Boolean!` | |
| `[]T` | `[T!]!` | Slice of non-nullable items |
| `[]*T` | `[T]!` | Slice of nullable items |
| `struct` | `type` | Object type with fields |

### 3. Function Parameter Resolution

Functions can accept parameters in different ways:

**Struct-based (Recommended for 3+ parameters):**
```go
type UserInput struct {
    Name  string `graphy:"name"`
    Email string `graphy:"email"`
    Age   *int   `graphy:"age"` // Optional field
}

func CreateUser(ctx context.Context, input UserInput) (*User, error) {
    // GraphQL: createUser(name: "Alice", email: "alice@example.com", age: 25)
}
```

**Named Parameters:**
```go
func GetUser(ctx context.Context, id int, includeDeleted bool) (*User, error) {
    // Specify parameter names during registration
}

g.RegisterQuery(ctx, "user", GetUser, "id", "includeDeleted")
// GraphQL: user(id: 123, includeDeleted: false)
```

**Positional Parameters:**
```go
func SearchUsers(ctx context.Context, query string, limit int) ([]*User, error) {
    // Without names, uses arg1, arg2, etc.
}

g.RegisterQuery(ctx, "searchUsers", SearchUsers)
// GraphQL: searchUsers(arg1: "alice", arg2: 10)
```

## Context Usage

Always accept `context.Context` as the first parameter:

```go
func GetUser(ctx context.Context, id int) (*User, error) {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Access authentication info
    userID := ctx.Value("userID").(string)
    
    // Use for database timeouts
    user, err := db.GetUserWithContext(ctx, id)
    return user, err
}
```

**Context Benefits:**
- Request cancellation and timeouts
- Authentication and authorization
- Request tracing and logging
- Database context propagation

## Error Handling

go-quickgraph follows Go's idiomatic error handling:

```go
func GetUser(ctx context.Context, id int) (*User, error) {
    if id <= 0 {
        return nil, fmt.Errorf("invalid user ID: %d", id)
    }
    
    user, err := database.GetUser(id)
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    
    return user, nil
}
```

**Error Response:**
```json
{
  "data": {"user": null},
  "errors": [{
    "message": "invalid user ID: -1",
    "path": ["user"],
    "locations": [{"line": 2, "column": 3}]
  }]
}
```

## Schema Generation

go-quickgraph generates GraphQL schemas automatically:

```go
// Get the generated schema
schema, err := g.SchemaDefinition(ctx)
if err != nil {
    log.Fatal(err)
}

fmt.Println(schema) // Full GraphQL SDL
```

**Schema includes:**
- All registered functions as queries/mutations/subscriptions
- All discovered types from function signatures
- Custom scalars and their descriptions
- Interface and union types
- Enum types with valid values

## Performance Considerations

### Caching

go-quickgraph caches aggressively:

```go
type MyCache struct {
    cache map[string]*CachedRequest
    mu    sync.RWMutex
}

func (c *MyCache) GetRequestStub(ctx context.Context, query string) (*RequestStub, error) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    // Return cached parsed query
}

func (c *MyCache) SetRequestStub(ctx context.Context, query string, stub *RequestStub, err error) {
    c.mu.Lock()
    defer c.mu.Unlock()  
    // Cache the parsed query
}

g := quickgraph.Graphy{RequestCache: &MyCache{}}
```

### Reflection Optimization

- Type information is cached after first use
- Function signatures are analyzed once at registration
- Schema generation is cached until functions change

## Thread Safety

go-quickgraph is thread-safe for concurrent use:

- `Graphy` is safe for concurrent requests after setup
- Registration must happen before serving requests
- Internal caches use proper synchronization

## Best Practices

### 1. Initialize Once
```go
// ✅ Create once, use many times
var graphqlHandler http.Handler

func init() {
    g := quickgraph.Graphy{}
    // Register all functions...
    g.EnableIntrospection(ctx)
    graphqlHandler = g.HttpHandler()
}

func main() {
    http.Handle("/graphql", graphqlHandler)
    http.ListenAndServe(":8080", nil)
}
```

### 2. Consider Struct Tags for Field Names

go-quickgraph uses struct tags to control GraphQL field names and metadata:

```go
// ✅ Recommended: Use graphy tags for new code
type User struct {
    ID   int    `graphy:"id,description=User identifier"`
    Name string `graphy:"name,description=User's full name"`
}

// ✅ Backward Compatible: JSON tags still work for legacy code
type LegacyUser struct {
    ID   int    `json:"id"`    // GraphQL field: "id"
    Name string `json:"name"`  // GraphQL field: "name"
}

// ✅ Without tags - uses Go field names
type User struct {
    ID   int     // GraphQL field: "ID"  
    Name string  // GraphQL field: "Name"
}
```

**Tag Support:**
- **`graphy` tags** (recommended): Full support for field names, descriptions, deprecation, etc.
- **`json` tags** (legacy support): Recognized for field naming to ease migration from existing codebases
- **Tag priority**: `graphy` tags take precedence over `json` tags when both are present

**Migration Note:** If you have existing Go structs with `json` tags, they'll work seamlessly with go-quickgraph. This backward compatibility feature makes it easy to adopt go-quickgraph in existing projects without modifying all your type definitions. However, for new code and when adding GraphQL-specific metadata (descriptions, deprecation notices), use `graphy` tags.

### 3. Handle Errors Gracefully
```go
// ✅ Return meaningful errors
func GetUser(ctx context.Context, id int) (*User, error) {
    if id <= 0 {
        return nil, fmt.Errorf("user ID must be positive, got %d", id)
    }
    // ...
}

// ❌ Panic or return invalid data
func GetUser(ctx context.Context, id int) *User {
    return &users[id] // Panic if ID doesn't exist
}
```

### 4. Use Struct Inputs for Complex Parameters
```go
// ✅ Easy to extend and maintain
type CreateUserInput struct {
    Name     string   `graphy:"name"`
    Email    string   `graphy:"email"`
    Roles    []string `graphy:"roles"`
    Settings *UserSettings `graphy:"settings"`
}

// ❌ Too many individual parameters
func CreateUser(ctx context.Context, name, email string, roles []string, settings *UserSettings) (*User, error)
```

## Next Steps

- **[Basic Operations](BASIC_OPERATIONS.md)** - Learn about queries, mutations, and basic types
- **[Function Patterns](FUNCTION_PATTERNS.md)** - Master different parameter and return value patterns
- **[Type System Guide](TYPE_SYSTEM.md)** - Build complex schemas with interfaces and unions