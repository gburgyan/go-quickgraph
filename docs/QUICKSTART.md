# go-quickgraph Quickstart

**Write Go. Get GraphQL. No magic.**

This guide shows you how to use your existing Go knowledge to build a GraphQL API in minutes.

## Zero to GraphQL in 30 Seconds

```go
package main

import (
    "context"
    "net/http"
    "github.com/gburgyan/go-quickgraph"
)

type Todo struct {
    ID   int    `graphy:"id"`
    Text string `graphy:"text"`
    Done bool   `graphy:"done"`
}

var todos = []Todo{{ID: 1, Text: "Write quickstart", Done: true}}

func GetTodos() []Todo {
    return todos
}

func main() {
    ctx := context.Background()
    g := quickgraph.Graphy{}
    
    g.RegisterQuery(ctx, "todos", GetTodos)
    g.EnableIntrospection(ctx)
    
    http.Handle("/graphql", g.HttpHandler())
    http.ListenAndServe(":8080", nil)
}
```

That's it! Visit http://localhost:8080/graphql and run:
```graphql
{ todos { id text done } }
```

## Just Write Go Code

### Your Functions Become GraphQL Operations

**Queries** - Functions that return data:
```go
func GetTodo(id int) (*Todo, error) {
    for _, todo := range todos {
        if todo.ID == id {
            return &todo, nil
        }
    }
    return nil, errors.New("todo not found")
}

// Register it
g.RegisterQuery(ctx, "todo", GetTodo, "id")
```

GraphQL:
```graphql
{ todo(id: 1) { text done } }
```

**Mutations** - Functions that change data:
```go
func CreateTodo(text string) (*Todo, error) {
    todo := Todo{
        ID:   len(todos) + 1,
        Text: text,
        Done: false,
    }
    todos = append(todos, todo)
    return &todo, nil
}

// Register it
g.RegisterMutation(ctx, "createTodo", CreateTodo, "text")
```

GraphQL:
```graphql
mutation { createTodo(text: "Learn GraphQL") { id text } }
```

### Use Your Existing Structs

Just add `graphy` tags to control field names (or use your existing `json` tags if your structs already have them - they work too!):
```go
type User struct {
    ID        int       `graphy:"id"`  // or `json:"id"` if it's already there
    FirstName string    `graphy:"firstName"`
    LastName  string    `graphy:"surname"`
    Email     string    `graphy:"email"`
    CreatedAt time.Time `graphy:"createdAt"`
}
```

### Error Handling - The Go Way

Return errors like you always do:
```go
func UpdateTodo(id int, text string, done bool) (*Todo, error) {
    for i, todo := range todos {
        if todo.ID == id {
            todos[i].Text = text
            todos[i].Done = done
            return &todos[i], nil
        }
    }
    return nil, fmt.Errorf("todo %d not found", id)
}
```

Errors appear in the GraphQL response:
```json
{
  "errors": [{
    "message": "todo 99 not found",
    "path": ["updateTodo"]
  }]
}
```

### Context for Auth & Cancellation

Use context as the first parameter when needed:
```go
func GetMyTodos(ctx context.Context) ([]Todo, error) {
    // Get user from context (set by middleware)
    user, ok := ctx.Value("user").(User)
    if !ok {
        return nil, errors.New("unauthorized")
    }
    
    // Return user's todos
    return getUserTodos(user.ID), nil
}

g.RegisterQuery(ctx, "myTodos", GetMyTodos)
```

## Common Go Patterns

### Input Types with Validation

Define input structs with a `Validate()` method:
```go
type CreateUserInput struct {
    FirstName string `graphy:"firstName"`
    LastName  string `graphy:"lastName"`
    Email     string `graphy:"email"`
    Age       int    `graphy:"age"`
}

func (input CreateUserInput) Validate() error {
    if input.FirstName == "" || input.LastName == "" {
        return errors.New("name is required")
    }
    if !strings.Contains(input.Email, "@") {
        return errors.New("invalid email")
    }
    if input.Age < 0 || input.Age > 150 {
        return errors.New("invalid age")
    }
    return nil
}

func CreateUser(input CreateUserInput) (*User, error) {
    // Validation happens automatically before this function is called
    user := User{
        ID:        generateID(),
        FirstName: input.FirstName,
        LastName:  input.LastName,
        Email:     input.Email,
    }
    saveUser(user)
    return &user, nil
}
```

### Optional Fields

Use pointers for nullable fields:
```go
type UpdateProfileInput struct {
    FirstName *string `graphy:"firstName"` // Optional
    LastName  *string `graphy:"lastName"`  // Optional
    Bio       *string `graphy:"bio"`       // Optional
}

func UpdateProfile(ctx context.Context, input UpdateProfileInput) (*User, error) {
    user := getCurrentUser(ctx)
    
    // Only update provided fields
    if input.FirstName != nil {
        user.FirstName = *input.FirstName
    }
    if input.LastName != nil {
        user.LastName = *input.LastName
    }
    if input.Bio != nil {
        user.Bio = input.Bio // Bio is already *string
    }
    
    return &user, nil
}
```

### Enums

Create string types with an `EnumValues()` method:
```go
type Status string

const (
    StatusActive   Status = "ACTIVE"
    StatusInactive Status = "INACTIVE"
    StatusPending  Status = "PENDING"
)

func (Status) EnumValues() []quickgraph.EnumValue {
    return []quickgraph.EnumValue{
        {Name: "ACTIVE", Description: "User is active"},
        {Name: "INACTIVE", Description: "User is inactive"},
        {Name: "PENDING", Description: "User is pending approval"},
    }
}

type User struct {
    ID     int    `graphy:"id"`
    Name   string `graphy:"name"`
    Status Status `graphy:"status"`
}
```

### Custom Scalars

For special types like DateTime, see the [Custom Scalars Guide](CUSTOM_SCALARS.md). Here's a quick example:

```go
// The library includes built-in support for time.Time as DateTime
type Event struct {
    ID        int       `graphy:"id"`
    Name      string    `graphy:"name"`
    StartTime time.Time `graphy:"startTime"` // Automatically serialized as DateTime
}

// For custom scalars, use RegisterScalar
g.RegisterScalar(ctx, quickgraph.ScalarDefinition{
    Name:   "EmailAddress",
    GoType: reflect.TypeOf(""),
    ParseValue: func(value interface{}) (interface{}, error) {
        email, ok := value.(string)
        if !ok || !strings.Contains(email, "@") {
            return nil, errors.New("invalid email address")
        }
        return email, nil
    },
    Serialize: func(value interface{}) (interface{}, error) {
        return value, nil // Already a string
    },
})
```

### Relationships and Nested Data

Return nested structs or add methods to types:
```go
type Post struct {
    ID       int    `graphy:"id"`
    Title    string `graphy:"title"`
    AuthorID int    `graphy:"authorId"`
}

// Method on Post becomes a GraphQL field
func (p Post) Author() (*User, error) {
    return GetUser(p.AuthorID)
}

// Method for reverse relationship
func (u User) Posts() ([]Post, error) {
    return getPostsByUser(u.ID)
}
```

GraphQL:
```graphql
{
  post(id: 1) {
    title
    author {
      name
      posts {
        title
      }
    }
  }
}
```

## Type System - It Just Works

Your Go types automatically become GraphQL types:

| Go Type | GraphQL Type | Example |
|---------|--------------|---------|
| `string` | `String!` | `Name string` |
| `int`, `int64` | `Int!` | `Age int` |
| `float64` | `Float!` | `Price float64` |
| `bool` | `Boolean!` | `Active bool` |
| `*string` | `String` | `Bio *string` (nullable) |
| `[]Post` | `[Post!]!` | `Posts []Post` |
| `[]*Post` | `[Post]!` | `Posts []*Post` |
| `time.Time` | `DateTime!` | `CreatedAt time.Time` |
| `interface{}` | `Any` | `Metadata interface{}` |

## Quick Start a Real App

Here's a minimal but complete example:

```go
package main

import (
    "context"
    "errors"
    "net/http"
    "sync"
    "time"
    
    "github.com/gburgyan/go-quickgraph"
)

// Domain types - just regular Go structs
type User struct {
    ID        int       `graphy:"id"`
    Email     string    `graphy:"email"`
    Name      string    `graphy:"name"`
    CreatedAt time.Time `graphy:"createdAt"`
}

type Post struct {
    ID        int       `graphy:"id"`
    Title     string    `graphy:"title"`
    Content   string    `graphy:"content"`
    AuthorID  int       `graphy:"authorId"`
    Price     float64   `graphy:"price"`
    CreatedAt time.Time `graphy:"createdAt"`
}

// Simple in-memory storage
var (
    users = []User{
        {ID: 1, Email: "alice@example.com", Name: "Alice", CreatedAt: time.Now()},
        {ID: 2, Email: "bob@example.com", Name: "Bob", CreatedAt: time.Now()},
    }
    posts = []Post{
        {ID: 1, Title: "Hello World", Content: "My first post", AuthorID: 1, Price: 9.99, CreatedAt: time.Now()},
        {ID: 2, Title: "GraphQL is Great", Content: "Learning about GraphQL", AuthorID: 2, Price: 14.99, CreatedAt: time.Now()},
    }
    mu sync.RWMutex
)

// Queries - just functions that return data
func GetUser(id int) (*User, error) {
    mu.RLock()
    defer mu.RUnlock()
    
    for _, user := range users {
        if user.ID == id {
            return &user, nil
        }
    }
    return nil, errors.New("user not found")
}

func GetUsers() []User {
    mu.RLock()
    defer mu.RUnlock()
    
    result := make([]User, len(users))
    copy(result, users)
    return result
}

func GetPosts() []Post {
    mu.RLock()
    defer mu.RUnlock()
    
    result := make([]Post, len(posts))
    copy(result, posts)
    return result
}

// Input type with validation
type CreatePostInput struct {
    Title    string  `graphy:"title"`
    Content  string  `graphy:"content"`
    AuthorID int     `graphy:"authorId"`
    Price    float64 `graphy:"price"`
}

func (input CreatePostInput) Validate() error {
    if input.Title == "" {
        return errors.New("title is required")
    }
    if input.Content == "" {
        return errors.New("content is required")
    }
    if input.Price < 0 {
        return errors.New("price cannot be negative")
    }
    return nil
}

// Mutations - functions that modify data
func CreatePost(input CreatePostInput) (*Post, error) {
    mu.Lock()
    defer mu.Unlock()
    
    // Validate author exists
    validAuthor := false
    for _, user := range users {
        if user.ID == input.AuthorID {
            validAuthor = true
            break
        }
    }
    if !validAuthor {
        return nil, errors.New("invalid author")
    }
    
    post := Post{
        ID:        len(posts) + 1,
        Title:     input.Title,
        Content:   input.Content,
        AuthorID:  input.AuthorID,
        Price:     input.Price,
        CreatedAt: time.Now(),
    }
    posts = append(posts, post)
    return &post, nil
}

// Relationships - methods on types become fields
func (p Post) Author() (*User, error) {
    return GetUser(p.AuthorID)
}

func (u User) Posts() []Post {
    mu.RLock()
    defer mu.RUnlock()

    var userPosts []Post
    for _, post := range posts {
        if post.AuthorID == u.ID {
            userPosts = append(userPosts, post)
        }
    }
    return userPosts
}

func main() {
    ctx := context.Background()
    g := quickgraph.Graphy{}
    
    // Register your functions
    g.RegisterQuery(ctx, "user", GetUser, "id")
    g.RegisterQuery(ctx, "users", GetUsers)
    g.RegisterQuery(ctx, "posts", GetPosts)
    g.RegisterMutation(ctx, "createPost", CreatePost, "input")
    
    // Enable GraphQL Playground
    g.EnableIntrospection(ctx)
    
    // Start server
    http.Handle("/graphql", g.HttpHandler())
    println("GraphQL server at http://localhost:8080/graphql")
    http.ListenAndServe(":8080", nil)
}
```

Try these queries at http://localhost:8080/graphql:

```graphql
# Get all users with their posts  
{
  users {
    id
    name
    email
    posts {
      title
      price
    }
  }
}

# Get all posts with authors
{
  posts {
    title
    content
    price
    author {
      name
      email
    }
  }
}

# Create a new post
mutation {
  createPost(input: {
    title: "GraphQL is Easy"
    content: "Just write Go!"
    authorId: 1
    price: 19.99
  }) {
    id
    title
    price
    createdAt
  }
}
```

## Try the Complete Example

The code from this quickstart is available as a runnable example:

```bash
cd examples/quickstart
go run .
# Visit http://localhost:8080/graphql
```

## What's Next?

You now know the essentials! For more advanced topics:

- **[Subscriptions](SUBSCRIPTIONS.md)** - Real-time updates with channels
- **[Security](SECURITY_API.md)** - Production security settings
- **[Performance](PERFORMANCE.md)** - Caching and optimization
- **[Type System](TYPE_SYSTEM.md)** - Interfaces, unions, and advanced types
- **[Custom Scalars](CUSTOM_SCALARS.md)** - Create your own scalar types

## Production Checklist

Before deploying, check the [Security Guide](SECURITY_API.md). Key settings:

```go
g := quickgraph.Graphy{
    ProductionMode: true,  // Sanitize errors
    MemoryLimits: &quickgraph.MemoryLimits{
        MaxRequestBodySize: 1024 * 1024, // 1MB
    },
    QueryLimits: &quickgraph.QueryLimits{
        MaxDepth: 10,
    },
}
```

---

**Remember**: It's just Go code. If you know Go, you know go-quickgraph.