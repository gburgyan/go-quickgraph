# Migration Guide to go-quickgraph

This guide helps you migrate from other GraphQL libraries to go-quickgraph's code-first approach. Whether you're coming from gqlgen, graphql-go, or another framework, this guide provides step-by-step instructions and examples.

## Why Migrate to go-quickgraph?

**The Simplicity Advantage:**
- **No schema files** - Your Go code IS the schema
- **No code generation** - No build steps, no generated files to manage
- **No resolver interfaces** - Just write regular Go functions
- **Native Go experience** - Full IDE support, type safety, and refactoring tools
- **Minimal boilerplate** - Register a function, get a GraphQL endpoint

## Table of Contents

1. [Quick Library Comparison](#quick-library-comparison)
2. [Migration in 5 Minutes](#migration-in-5-minutes)
3. [Understanding the Paradigm Shift](#understanding-the-paradigm-shift)
4. [Migration from gqlgen](#migration-from-gqlgen)
5. [Migration from graphql-go/graphql](#migration-from-graphql-gographql)
6. [Migration from graph-gophers/graphql-go](#migration-from-graph-gophersgraphql-go)
7. [Common Migration Patterns](#common-migration-patterns)
8. [Migration Checklists](#migration-checklists)
9. [Best Practices](#best-practices)
10. [Quick Reference](#quick-reference)
11. [Complete Migration Example](#complete-migration-example)

## Quick Library Comparison

Here's how go-quickgraph compares to popular Go GraphQL libraries:

| Feature | gqlgen | graphql-go/graphql | graph-gophers | go-quickgraph |
|---------|--------|-------------------|--------------|---------------|
| **Approach** | Schema-first | Code-first (verbose) | Schema-first | Code-first (simple) |
| **Schema Definition** | `.graphql` files | Go code (verbose) | `.graphql` files | Go structs |
| **Code Generation** | Required | Not required | Not required | Not required |
| **Type Safety** | Generated types | Runtime checks | Runtime checks | Compile-time |
| **Resolver Pattern** | Interface implementation | Verbose field definitions | Interface methods | Regular functions |
| **Boilerplate** | High | Very High | Medium | Minimal |
| **Setup Complexity** | Complex | Complex | Medium | Simple |
| **Build Steps** | Multiple | None | None | None |
| **Hot Reload** | Requires regeneration | Yes | Yes | Yes |
| **Learning Curve** | Steep | Steep | Medium | Gentle |

### Lines of Code Comparison

For a simple user query with posts:

| Library | Schema | Type Definitions | Resolvers | Setup | Total |
|---------|--------|-----------------|-----------|--------|-------|
| gqlgen | 15 lines | Generated | 30 lines | 20 lines | 65+ lines |
| graphql-go | N/A | 50 lines | 40 lines | 30 lines | 120 lines |
| graph-gophers | 15 lines | 20 lines | 35 lines | 25 lines | 95 lines |
| **go-quickgraph** | **0 lines** | **10 lines** | **15 lines** | **10 lines** | **35 lines** |

## Migration in 5 Minutes

**The fastest way to understand go-quickgraph? See a complete migration:**

### Step 1: Replace Your Schema with Go Types

**Before (any schema-first library):**
```graphql
type User {
  id: ID!
  name: String!
  email: String!
  posts: [Post!]!
}

type Post {
  id: ID!
  title: String!
  content: String!
  author: User!
}

type Query {
  user(id: ID!): User
  posts(limit: Int): [Post!]!
}
```

**After (go-quickgraph):**
```go
// Just regular Go structs!
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Methods become GraphQL fields
func (u *User) Posts(ctx context.Context) ([]*Post, error) {
    return getPostsByUserID(ctx, u.ID)
}

type Post struct {
    ID       string `json:"id"`
    Title    string `json:"title"`
    Content  string `json:"content"`
    AuthorID string `json:"authorId"`
}

func (p *Post) Author(ctx context.Context) (*User, error) {
    return getUserByID(ctx, p.AuthorID)
}
```

### Step 2: Replace Resolvers with Functions

**Before (resolver interfaces):**
```go
func (r *queryResolver) User(ctx context.Context, id string) (*User, error) {
    return r.db.GetUser(ctx, id)
}

func (r *queryResolver) Posts(ctx context.Context, limit *int) ([]*Post, error) {
    l := 10
    if limit != nil {
        l = *limit
    }
    return r.db.GetPosts(ctx, l)
}
```

**After (just functions):**
```go
// Plain Go functions - no interfaces!
func GetUser(ctx context.Context, id string) (*User, error) {
    db := ctx.Value("db").(*Database)
    return db.GetUser(ctx, id)
}

func GetPosts(ctx context.Context, limit *int) ([]*Post, error) {
    db := ctx.Value("db").(*Database)
    l := 10
    if limit != nil {
        l = *limit
    }
    return db.GetPosts(ctx, l)
}
```

### Step 3: Wire It Up

**Before (complex setup):**
```go
// gqlgen example
srv := handler.NewDefaultServer(
    generated.NewExecutableSchema(
        generated.Config{
            Resolvers: &graph.Resolver{DB: db},
        },
    ),
)
// Plus configuration files, generated code, etc.
```

**After (simple registration):**
```go
// Create server
g := quickgraph.Graphy{}

// Register your functions
g.RegisterQuery(ctx, "user", GetUser, "id")
g.RegisterQuery(ctx, "posts", GetPosts, "limit")

// Enable introspection
g.EnableIntrospection(ctx)

// Serve
http.Handle("/graphql", g.HttpHandler())
```

### That's It! 🎉

You now have a fully functional GraphQL API with:
- ✅ Type-safe schema generated from your Go code
- ✅ Automatic query validation
- ✅ Built-in introspection
- ✅ No schema files
- ✅ No code generation
- ✅ No resolver interfaces

**Try it:**
```graphql
{
  user(id: "123") {
    name
    email
    posts {
      title
      content
    }
  }
}
```

## Understanding the Paradigm Shift

### Traditional Schema-First Approach
```graphql
# schema.graphql
type User {
  id: ID!
  name: String!
  email: String!
}

type Query {
  user(id: ID!): User
  users(limit: Int): [User!]!
}
```

Then generate code and implement resolvers...

### go-quickgraph Code-First Approach
```go
// Just write Go code!
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func GetUser(ctx context.Context, id string) (*User, error) {
    // Your implementation
}

// Register and done!
g.RegisterQuery(ctx, "user", GetUser, "id")
```

## Migration from gqlgen

gqlgen is a popular schema-first GraphQL library. Here's how to migrate:

### 1. Remove Schema Files and Generated Code

**Before (gqlgen):**
```
project/
├── gqlgen.yml
├── schema.graphql
├── graph/
│   ├── generated/
│   ├── model/
│   └── resolver.go
```

**After (go-quickgraph):**
```
project/
├── types.go      # Your domain types
├── queries.go    # Query functions
├── mutations.go  # Mutation functions
└── main.go       # Setup and registration
```

### 2. Convert Schema Types to Go Structs

**Before (gqlgen schema.graphql):**
```graphql
type Product {
  id: ID!
  name: String!
  price: Float!
  description: String
  inStock: Boolean!
  category: Category!
  tags: [String!]!
}

type Category {
  id: ID!
  name: String!
}

input ProductInput {
  name: String!
  price: Float!
  description: String
  categoryId: ID!
  tags: [String!]
}
```

**After (go-quickgraph types.go):**
```go
type Product struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Price       float64   `json:"price"`
    Description *string   `json:"description"` // Pointer = nullable
    InStock     bool      `json:"inStock"`
    Category    *Category `json:"category"`
    Tags        []string  `json:"tags"`
}

type Category struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type ProductInput struct {
    Name        string   `json:"name"`
    Price       float64  `json:"price"`
    Description *string  `json:"description"`
    CategoryID  string   `json:"categoryId"`
    Tags        []string `json:"tags"`
}
```

### 3. Convert Resolvers to Functions

**Before (gqlgen resolver.go):**
```go
type Resolver struct {
    db *sql.DB
}

func (r *queryResolver) Product(ctx context.Context, id string) (*model.Product, error) {
    product, err := r.db.GetProduct(ctx, id)
    if err != nil {
        return nil, err
    }
    return product, nil
}

func (r *queryResolver) Products(ctx context.Context, limit *int) ([]*model.Product, error) {
    l := 10
    if limit != nil {
        l = *limit
    }
    return r.db.ListProducts(ctx, l)
}

func (r *mutationResolver) CreateProduct(ctx context.Context, input model.ProductInput) (*model.Product, error) {
    product := &model.Product{
        Name:        input.Name,
        Price:       input.Price,
        Description: input.Description,
    }
    if err := r.db.CreateProduct(ctx, product); err != nil {
        return nil, err
    }
    return product, nil
}

func (r *productResolver) Category(ctx context.Context, obj *model.Product) (*model.Category, error) {
    return r.db.GetCategory(ctx, obj.CategoryID)
}
```

**After (go-quickgraph queries.go & mutations.go):**
```go
// queries.go
func GetProduct(ctx context.Context, id string) (*Product, error) {
    db := ctx.Value("db").(*sql.DB)
    product, err := db.GetProduct(ctx, id)
    if err != nil {
        return nil, err
    }
    // Lazy loading handled by methods
    return product, nil
}

func GetProducts(ctx context.Context, limit *int) ([]*Product, error) {
    db := ctx.Value("db").(*sql.DB)
    l := 10
    if limit != nil {
        l = *limit
    }
    return db.ListProducts(ctx, l)
}

// mutations.go
func CreateProduct(ctx context.Context, input ProductInput) (*Product, error) {
    db := ctx.Value("db").(*sql.DB)
    product := &Product{
        Name:        input.Name,
        Price:       input.Price,
        Description: input.Description,
    }
    if err := db.CreateProduct(ctx, product); err != nil {
        return nil, err
    }
    return product, nil
}

// Add method to Product for lazy loading
func (p *Product) Category(ctx context.Context) (*Category, error) {
    if p.Category != nil {
        return p.Category, nil
    }
    db := ctx.Value("db").(*sql.DB)
    return db.GetCategory(ctx, p.CategoryID)
}
```

### 4. Update Server Setup

**Before (gqlgen main.go):**
```go
package main

import (
    "log"
    "net/http"
    
    "github.com/99designs/gqlgen/graphql/handler"
    "github.com/99designs/gqlgen/graphql/playground"
    "myapp/graph"
    "myapp/graph/generated"
)

func main() {
    db := initDB()
    
    srv := handler.NewDefaultServer(
        generated.NewExecutableSchema(
            generated.Config{
                Resolvers: &graph.Resolver{DB: db},
            },
        ),
    )
    
    http.Handle("/", playground.Handler("GraphQL playground", "/query"))
    http.Handle("/query", srv)
    
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

**After (go-quickgraph main.go):**
```go
package main

import (
    "context"
    "log"
    "net/http"
    
    "github.com/gburgyan/go-quickgraph"
)

func main() {
    db := initDB()
    
    g := quickgraph.Graphy{
        MemoryLimits: quickgraph.DefaultMemoryLimits(),
        QueryLimits:  quickgraph.DefaultQueryLimits(),
    }
    
    // Create context with DB
    ctx := context.WithValue(context.Background(), "db", db)
    
    // Register queries
    g.RegisterQuery(ctx, "product", GetProduct, "id")
    g.RegisterQuery(ctx, "products", GetProducts, "limit")
    
    // Register mutations
    g.RegisterMutation(ctx, "createProduct", CreateProduct, "input")
    
    // Enable introspection for development
    g.EnableIntrospection(ctx)
    
    http.Handle("/query", g.HttpHandler())
    
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### 5. Handle Subscriptions

**Before (gqlgen):**
```go
func (r *subscriptionResolver) ProductUpdated(ctx context.Context, productID string) (<-chan *model.Product, error) {
    ch := make(chan *model.Product, 1)
    
    go func() {
        defer close(ch)
        for {
            select {
            case <-ctx.Done():
                return
            case product := <-r.productUpdates:
                if product.ID == productID {
                    ch <- product
                }
            }
        }
    }()
    
    return ch, nil
}
```

**After (go-quickgraph):**
```go
func ProductUpdated(ctx context.Context, productID string) (<-chan *Product, error) {
    ch := make(chan *Product, 1)
    updates := ctx.Value("productUpdates").(chan *Product)
    
    go func() {
        defer close(ch)
        for {
            select {
            case <-ctx.Done():
                return
            case product := <-updates:
                if product.ID == productID {
                    ch <- product
                }
            }
        }
    }()
    
    return ch, nil
}

// Register subscription
g.RegisterSubscription(ctx, "productUpdated", ProductUpdated, "productID")
```

## Migration from graphql-go/graphql

graphql-go/graphql uses a programmatic approach to building schemas. Here's how to migrate:

### 1. Convert Type Definitions

**Before (graphql-go):**
```go
var userType = graphql.NewObject(graphql.ObjectConfig{
    Name: "User",
    Fields: graphql.Fields{
        "id": &graphql.Field{
            Type: graphql.NewNonNull(graphql.ID),
        },
        "name": &graphql.Field{
            Type: graphql.NewNonNull(graphql.String),
        },
        "email": &graphql.Field{
            Type: graphql.NewNonNull(graphql.String),
        },
        "posts": &graphql.Field{
            Type: graphql.NewList(postType),
            Resolve: func(p graphql.ResolveParams) (interface{}, error) {
                user := p.Source.(*User)
                return getPostsByUserID(user.ID)
            },
        },
    },
})

var postType = graphql.NewObject(graphql.ObjectConfig{
    Name: "Post",
    Fields: graphql.Fields{
        "id":      &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
        "title":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
        "content": &graphql.Field{Type: graphql.String},
    },
})
```

**After (go-quickgraph):**
```go
type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Method for lazy loading
func (u *User) Posts(ctx context.Context) ([]*Post, error) {
    return getPostsByUserID(ctx, u.ID)
}

type Post struct {
    ID      string  `json:"id"`
    Title   string  `json:"title"`
    Content *string `json:"content"` // Nullable
}
```

### 2. Convert Query Definitions

**Before (graphql-go):**
```go
var queryType = graphql.NewObject(graphql.ObjectConfig{
    Name: "Query",
    Fields: graphql.Fields{
        "user": &graphql.Field{
            Type: userType,
            Args: graphql.FieldConfigArgument{
                "id": &graphql.ArgumentConfig{
                    Type: graphql.NewNonNull(graphql.ID),
                },
            },
            Resolve: func(p graphql.ResolveParams) (interface{}, error) {
                id := p.Args["id"].(string)
                return getUserByID(id)
            },
        },
        "users": &graphql.Field{
            Type: graphql.NewList(userType),
            Args: graphql.FieldConfigArgument{
                "limit": &graphql.ArgumentConfig{
                    Type:         graphql.Int,
                    DefaultValue: 10,
                },
            },
            Resolve: func(p graphql.ResolveParams) (interface{}, error) {
                limit := p.Args["limit"].(int)
                return getUsers(limit)
            },
        },
    },
})
```

**After (go-quickgraph):**
```go
func GetUser(ctx context.Context, id string) (*User, error) {
    return getUserByID(ctx, id)
}

func GetUsers(ctx context.Context, limit *int) ([]*User, error) {
    l := 10
    if limit != nil {
        l = *limit
    }
    return getUsers(ctx, l)
}

// Registration
g.RegisterQuery(ctx, "user", GetUser, "id")
g.RegisterQuery(ctx, "users", GetUsers, "limit")
```

### 3. Convert Mutations

**Before (graphql-go):**
```go
var mutationType = graphql.NewObject(graphql.ObjectConfig{
    Name: "Mutation",
    Fields: graphql.Fields{
        "createUser": &graphql.Field{
            Type: userType,
            Args: graphql.FieldConfigArgument{
                "input": &graphql.ArgumentConfig{
                    Type: graphql.NewNonNull(userInputType),
                },
            },
            Resolve: func(p graphql.ResolveParams) (interface{}, error) {
                input := p.Args["input"].(map[string]interface{})
                user := &User{
                    Name:  input["name"].(string),
                    Email: input["email"].(string),
                }
                return createUser(user)
            },
        },
    },
})
```

**After (go-quickgraph):**
```go
type CreateUserInput struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func CreateUser(ctx context.Context, input CreateUserInput) (*User, error) {
    user := &User{
        Name:  input.Name,
        Email: input.Email,
    }
    return createUser(ctx, user)
}

// Registration
g.RegisterMutation(ctx, "createUser", CreateUser, "input")
```

### 4. Update Server Setup

**Before (graphql-go):**
```go
schema, _ := graphql.NewSchema(graphql.SchemaConfig{
    Query:    queryType,
    Mutation: mutationType,
})

h := handler.New(&handler.Config{
    Schema:   &schema,
    Pretty:   true,
    GraphiQL: true,
})

http.Handle("/graphql", h)
http.ListenAndServe(":8080", nil)
```

**After (go-quickgraph):**
```go
g := quickgraph.Graphy{}

// Register all operations
g.RegisterQuery(ctx, "user", GetUser, "id")
g.RegisterQuery(ctx, "users", GetUsers, "limit")
g.RegisterMutation(ctx, "createUser", CreateUser, "input")

g.EnableIntrospection(ctx)

http.Handle("/graphql", g.HttpHandler())
http.ListenAndServe(":8080", nil)
```

## Migration from graph-gophers/graphql-go

graph-gophers/graphql-go is another schema-first library. Here's the migration path:

### 1. Convert Schema to Go Types

**Before (graph-gophers schema):**
```graphql
schema {
    query: Query
    mutation: Mutation
}

type Query {
    user(id: ID!): User
    users(filter: UserFilter): [User!]!
}

type Mutation {
    createUser(input: UserInput!): User!
}

type User {
    id: ID!
    name: String!
    email: String!
    role: UserRole!
    createdAt: DateTime!
}

enum UserRole {
    ADMIN
    USER
    GUEST
}

input UserFilter {
    role: UserRole
    createdAfter: DateTime
}

input UserInput {
    name: String!
    email: String!
    role: UserRole!
}

scalar DateTime
```

**After (go-quickgraph):**
```go
type User struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    Role      UserRole  `json:"role"`
    CreatedAt time.Time `json:"createdAt"`
}

type UserRole string

func (r UserRole) EnumValues() []string {
    return []string{"ADMIN", "USER", "GUEST"}
}

type UserFilter struct {
    Role         *UserRole  `json:"role"`
    CreatedAfter *time.Time `json:"createdAfter"`
}

type UserInput struct {
    Name  string   `json:"name"`
    Email string   `json:"email"`
    Role  UserRole `json:"role"`
}
```

### 2. Convert Resolver Interface

**Before (graph-gophers):**
```go
type Resolver struct {
    db *Database
}

func (r *Resolver) User(ctx context.Context, args struct{ ID graphql.ID }) (*UserResolver, error) {
    user, err := r.db.GetUser(string(args.ID))
    if err != nil {
        return nil, err
    }
    return &UserResolver{user: user}, nil
}

func (r *Resolver) Users(ctx context.Context, args struct{ Filter *UserFilterInput }) ([]*UserResolver, error) {
    users, err := r.db.GetUsers(args.Filter)
    if err != nil {
        return nil, err
    }
    
    resolvers := make([]*UserResolver, len(users))
    for i, user := range users {
        resolvers[i] = &UserResolver{user: user}
    }
    return resolvers, nil
}

type UserResolver struct {
    user *User
}

func (r *UserResolver) ID() graphql.ID {
    return graphql.ID(r.user.ID)
}

func (r *UserResolver) Name() string {
    return r.user.Name
}

func (r *UserResolver) Email() string {
    return r.user.Email
}
```

**After (go-quickgraph):**
```go
func GetUser(ctx context.Context, id string) (*User, error) {
    db := ctx.Value("db").(*Database)
    return db.GetUser(id)
}

func GetUsers(ctx context.Context, filter *UserFilter) ([]*User, error) {
    db := ctx.Value("db").(*Database)
    return db.GetUsers(filter)
}

func CreateUser(ctx context.Context, input UserInput) (*User, error) {
    db := ctx.Value("db").(*Database)
    user := &User{
        ID:        generateID(),
        Name:      input.Name,
        Email:     input.Email,
        Role:      input.Role,
        CreatedAt: time.Now(),
    }
    if err := db.CreateUser(user); err != nil {
        return nil, err
    }
    return user, nil
}
```

### 3. Register DateTime Scalar

**Before (graph-gophers):**
```go
// Custom scalar implementation required
```

**After (go-quickgraph):**
```go
// Built-in support
g.RegisterDateTimeScalar(ctx)
```

## Common Migration Patterns

### Pattern 1: Basic Type Definition

**gqlgen (schema + generated code):**
```graphql
# schema.graphql
type User {
  id: ID!
  name: String!
  email: String!
  bio: String
  age: Int
}
```
Plus generated Go code you can't modify directly.

**graphql-go/graphql (verbose code):**
```go
var userType = graphql.NewObject(graphql.ObjectConfig{
    Name: "User",
    Fields: graphql.Fields{
        "id": &graphql.Field{
            Type: graphql.NewNonNull(graphql.ID),
        },
        "name": &graphql.Field{
            Type: graphql.NewNonNull(graphql.String),
        },
        "email": &graphql.Field{
            Type: graphql.NewNonNull(graphql.String),
        },
        "bio": &graphql.Field{
            Type: graphql.String, // Nullable
        },
        "age": &graphql.Field{
            Type: graphql.Int, // Nullable
        },
    },
})
```

**go-quickgraph (simple struct):**
```go
type User struct {
    ID    string  `json:"id"`
    Name  string  `json:"name"`
    Email string  `json:"email"`
    Bio   *string `json:"bio"`  // Pointer = nullable
    Age   *int    `json:"age"`  // Pointer = nullable
}
```

### Pattern 2: Resolver/Function Implementation

**gqlgen (resolver interface):**
```go
type Resolver struct{
    db *sql.DB
}

type QueryResolver interface {
    User(ctx context.Context, id string) (*model.User, error)
}

func (r *queryResolver) User(ctx context.Context, id string) (*model.User, error) {
    return r.db.GetUser(ctx, id)
}
```

**graphql-go/graphql (field resolver):**
```go
"user": &graphql.Field{
    Type: userType,
    Args: graphql.FieldConfigArgument{
        "id": &graphql.ArgumentConfig{
            Type: graphql.NewNonNull(graphql.String),
        },
    },
    Resolve: func(p graphql.ResolveParams) (interface{}, error) {
        id := p.Args["id"].(string)
        return db.GetUser(p.Context, id)
    },
}
```

**go-quickgraph (just a function):**
```go
func GetUser(ctx context.Context, id string) (*User, error) {
    db := ctx.Value("db").(*sql.DB)
    return db.GetUser(ctx, id)
}

// Register it
g.RegisterQuery(ctx, "user", GetUser, "id")
```

### Pattern 3: Input Validation

**gqlgen (manual validation in resolver):**
```go
func (r *mutationResolver) CreateUser(ctx context.Context, input model.NewUser) (*model.User, error) {
    // Manual validation
    if input.Name == "" {
        return nil, fmt.Errorf("name is required")
    }
    if len(input.Name) < 2 {
        return nil, fmt.Errorf("name too short")
    }
    if !strings.Contains(input.Email, "@") {
        return nil, fmt.Errorf("invalid email")
    }
    
    // Create user
    return r.db.CreateUser(ctx, input)
}
```

**graphql-go/graphql (validation in resolver):**
```go
Resolve: func(p graphql.ResolveParams) (interface{}, error) {
    input := p.Args["input"].(map[string]interface{})
    
    // Extract and validate
    name := input["name"].(string)
    if name == "" {
        return nil, fmt.Errorf("name is required")
    }
    
    email := input["email"].(string)
    if !strings.Contains(email, "@") {
        return nil, fmt.Errorf("invalid email")
    }
    
    return createUser(name, email)
}
```

**go-quickgraph (automatic validation):**
```go
type CreateUserInput struct {
    Name     string `json:"name"`
    Email    string `json:"email"`
    Password string `json:"password"`
}

// Implement Validator interface - runs automatically before resolver
func (i CreateUserInput) Validate() error {
    if i.Name == "" {
        return fmt.Errorf("name is required")
    }
    if len(i.Name) < 2 {
        return fmt.Errorf("name must be at least 2 characters")
    }
    if !strings.Contains(i.Email, "@") {
        return fmt.Errorf("invalid email format")
    }
    if len(i.Password) < 8 {
        return fmt.Errorf("password must be at least 8 characters")
    }
    return nil
}

// Your resolver doesn't need to call Validate()
func CreateUser(ctx context.Context, input CreateUserInput) (*User, error) {
    // Validation already happened automatically
    return saveUser(&User{
        Name:  input.Name,
        Email: input.Email,
    })
}
```

### Pattern 4: Relationships/Lazy Loading

**gqlgen (field resolver):**
```go
// Separate resolver for User.posts field
func (r *userResolver) Posts(ctx context.Context, obj *model.User) ([]*model.Post, error) {
    return r.db.GetPostsByUserID(ctx, obj.ID)
}

// Must implement UserResolver interface
type userResolver struct{ *Resolver }
```

**graphql-go/graphql (field configuration):**
```go
"posts": &graphql.Field{
    Type: graphql.NewList(postType),
    Resolve: func(p graphql.ResolveParams) (interface{}, error) {
        user := p.Source.(*User)
        return db.GetPostsByUserID(p.Context, user.ID)
    },
}
```

**go-quickgraph (method on struct):**
```go
// Just add a method to your struct
func (u *User) Posts(ctx context.Context) ([]*Post, error) {
    return getPostsByUserID(ctx, u.ID)
}
// That's it! No configuration needed
```

### Pattern 5: DataLoader Pattern

**Traditional (with dataloader library):**
```go
type Loaders struct {
    UserByID *dataloader.Loader
}

func NewLoaders() *Loaders {
    return &Loaders{
        UserByID: dataloader.NewBatchedLoader(batchGetUsers),
    }
}
```

**go-quickgraph (using context and methods):**
```go
type Post struct {
    ID       string `json:"id"`
    AuthorID string `json:"authorId"`
}

// Lazy loading with caching via context
func (p *Post) Author(ctx context.Context) (*User, error) {
    // Check context cache first
    cache := ctx.Value("userCache").(map[string]*User)
    if user, ok := cache[p.AuthorID]; ok {
        return user, nil
    }
    
    // Load and cache
    user, err := getUserByID(ctx, p.AuthorID)
    if err != nil {
        return nil, err
    }
    cache[p.AuthorID] = user
    return user, nil
}
```

### Pattern 6: Error Handling

**gqlgen (multiple error return paths):**
```go
func (r *queryResolver) User(ctx context.Context, id string) (*model.User, error) {
    if id == "" {
        return nil, &gqlerror.Error{
            Message: "User ID is required",
            Extensions: map[string]interface{}{
                "code": "BAD_USER_INPUT",
            },
        }
    }
    
    user, err := r.db.GetUser(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    return user, nil
}
```

**graphql-go/graphql (mixed error handling):**
```go
Resolve: func(p graphql.ResolveParams) (interface{}, error) {
    id, ok := p.Args["id"].(string)
    if !ok || id == "" {
        return nil, errors.New("invalid user ID")
    }
    
    return getUserByID(p.Context, id)
}
```

**go-quickgraph (simple Go errors):**
```go
func GetUser(ctx context.Context, id string) (*User, error) {
    if id == "" {
        return nil, fmt.Errorf("user ID is required")
    }
    
    user, err := getUserByID(ctx, id)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("user not found: %s", id)
    }
    return user, err
}
```

### Pattern 7: Pagination

**Traditional:**
```graphql
type Query {
    users(first: Int, after: String): UserConnection!
}

type UserConnection {
    edges: [UserEdge!]!
    pageInfo: PageInfo!
}
```

**go-quickgraph:**
```go
type UserConnection struct {
    Edges    []*UserEdge `json:"edges"`
    PageInfo *PageInfo  `json:"pageInfo"`
}

type UserEdge struct {
    Node   *User  `json:"node"`
    Cursor string `json:"cursor"`
}

type PageInfo struct {
    HasNextPage     bool   `json:"hasNextPage"`
    HasPreviousPage bool   `json:"hasPreviousPage"`
    StartCursor     string `json:"startCursor"`
    EndCursor       string `json:"endCursor"`
}

func GetUsers(ctx context.Context, first *int, after *string) (*UserConnection, error) {
    limit := 10
    if first != nil {
        limit = *first
    }
    
    users, hasNext, err := fetchUsersWithCursor(ctx, limit, after)
    if err != nil {
        return nil, err
    }
    
    edges := make([]*UserEdge, len(users))
    for i, user := range users {
        edges[i] = &UserEdge{
            Node:   user,
            Cursor: encodeCursor(user.ID),
        }
    }
    
    return &UserConnection{
        Edges: edges,
        PageInfo: &PageInfo{
            HasNextPage: hasNext,
            StartCursor: edges[0].Cursor,
            EndCursor:   edges[len(edges)-1].Cursor,
        },
    }, nil
}
```

### Pattern 8: Authorization

**Traditional (middleware/directives):**
```graphql
type Query {
    adminUsers: [User!]! @hasRole(role: "ADMIN")
}
```

**go-quickgraph (in function):**
```go
func GetAdminUsers(ctx context.Context) ([]*User, error) {
    // Check authorization
    currentUser := ctx.Value("user").(*User)
    if currentUser.Role != "ADMIN" {
        return nil, fmt.Errorf("unauthorized: admin access required")
    }
    
    return fetchAdminUsers(ctx)
}

// Or use validation
type AdminQuery struct {
    Action string `json:"action"`
}

func (q AdminQuery) ValidateWithContext(ctx context.Context) error {
    user := ctx.Value("user").(*User)
    if user.Role != "ADMIN" {
        return fmt.Errorf("admin access required")
    }
    return nil
}
```

### Pattern 9: Custom Scalars

**Traditional:**
- Define scalar in schema
- Implement marshal/unmarshal methods
- Register with schema

**go-quickgraph:**
```go
// Define custom type
type Money struct {
    Amount   int64
    Currency string
}

// Register scalar
g.RegisterScalar(ctx, quickgraph.ScalarDefinition{
    Name:        "Money",
    GoType:      reflect.TypeOf(Money{}),
    Description: "Money represented as minor units",
    Serialize: func(value interface{}) (interface{}, error) {
        money := value.(Money)
        return fmt.Sprintf("%d %s", money.Amount, money.Currency), nil
    },
    ParseValue: func(value interface{}) (interface{}, error) {
        str := value.(string)
        var amount int64
        var currency string
        fmt.Sscanf(str, "%d %s", &amount, &currency)
        return Money{Amount: amount, Currency: currency}, nil
    },
})

// Use in types
type Product struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Price Money  `json:"price"`
}
```

## Migration Checklists

### gqlgen Migration Checklist

- [ ] **Remove all generated files and folders**
  - `graph/generated/`
  - `graph/model/`
  - Any `*.generated.go` files

- [ ] **Delete configuration files**
  - `gqlgen.yml`
  - `tools.go` (if only used for gqlgen)

- [ ] **Convert schema files to Go structs**
  - Each GraphQL type → Go struct
  - Non-nullable fields → regular types
  - Nullable fields → pointer types
  - Lists → slices

- [ ] **Transform resolvers to functions**
  - Remove resolver receiver methods
  - Convert to standalone functions
  - Add context as first parameter
  - Access dependencies via context

- [ ] **Handle field resolvers**
  - Convert to methods on structs
  - Use lazy loading pattern
  - Add context parameter

- [ ] **Update imports**
  - Remove gqlgen imports
  - Add `github.com/gburgyan/go-quickgraph`

- [ ] **Simplify server setup**
  - Remove schema configuration
  - Register functions directly
  - Enable introspection

### graphql-go/graphql Migration Checklist

- [ ] **Remove verbose type definitions**
  - Delete all `graphql.NewObject()` calls
  - Remove field configuration maps
  - Delete resolver functions in field definitions

- [ ] **Convert to simple structs**
  - Replace type objects with Go structs
  - Use json tags for field names
  - Add methods for relationships

- [ ] **Simplify query/mutation definitions**
  - Remove query type object
  - Remove mutation type object
  - Convert to regular functions

- [ ] **Clean up argument handling**
  - Remove `Args` configuration
  - Use function parameters directly
  - Let go-quickgraph handle type conversion

- [ ] **Update server initialization**
  - Remove schema building
  - Register functions instead
  - Simplify HTTP handler

### graph-gophers/graphql-go Migration Checklist

- [ ] **Remove schema string/file**
  - Delete `.graphql` files
  - Remove embedded schema strings

- [ ] **Eliminate resolver interfaces**
  - Remove resolver structs
  - Remove method receivers
  - Convert to standalone functions

- [ ] **Simplify type handling**
  - Remove resolver wrapper types
  - Return domain types directly
  - Use built-in scalar support

- [ ] **Update argument structs**
  - Remove anonymous structs in parameters
  - Use named parameter types
  - Simplify input handling

- [ ] **Convert custom scalars**
  - Remove marshal/unmarshal methods
  - Use go-quickgraph's scalar registration

## Best Practices

### 1. Context Usage

Always pass context as the first parameter:
```go
// ✅ Good
func GetUser(ctx context.Context, id string) (*User, error)

// ❌ Bad
func GetUser(id string) (*User, error)
```

### 2. Error Handling

Return meaningful errors:
```go
func GetUser(ctx context.Context, id string) (*User, error) {
    if id == "" {
        return nil, fmt.Errorf("user ID is required")
    }
    
    user, err := db.GetUser(ctx, id)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("user not found: %s", id)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user: %w", err)
    }
    
    return user, nil
}
```

### 3. Input Validation

go-quickgraph provides native validation support. When your input types implement `Validator` or `ValidatorWithContext`, validation runs automatically before your resolver:

```go
type CreatePostInput struct {
    Title   string   `json:"title"`
    Content string   `json:"content"`
    Tags    []string `json:"tags"`
}

// Implement Validator - runs automatically before resolver
func (i CreatePostInput) Validate() error {
    if i.Title == "" {
        return fmt.Errorf("title is required")
    }
    if len(i.Title) > 200 {
        return fmt.Errorf("title must be less than 200 characters")
    }
    if len(i.Tags) > 10 {
        return fmt.Errorf("maximum 10 tags allowed")
    }
    return nil
}

// Your resolver doesn't need to call Validate()
func CreatePost(ctx context.Context, input CreatePostInput) (*Post, error) {
    // Validation already happened automatically
    return savePost(&Post{
        Title:   input.Title,
        Content: input.Content,
        Tags:    input.Tags,
    })
}
```

### 4. Lazy Loading

Use methods for related data:
```go
type User struct {
    ID string `json:"id"`
}

// Lazy load posts
func (u *User) Posts(ctx context.Context, limit *int) ([]*Post, error) {
    l := 10
    if limit != nil {
        l = *limit
    }
    return getPostsByUserID(ctx, u.ID, l)
}

// Lazy load with caching
func (u *User) Profile(ctx context.Context) (*Profile, error) {
    cache := ctx.Value("profileCache").(map[string]*Profile)
    if profile, ok := cache[u.ID]; ok {
        return profile, nil
    }
    
    profile, err := getProfileByUserID(ctx, u.ID)
    if err != nil {
        return nil, err
    }
    
    cache[u.ID] = profile
    return profile, nil
}
```

### 5. Testing

Test your GraphQL operations:
```go
func TestGetUser(t *testing.T) {
    g := quickgraph.Graphy{}
    ctx := context.Background()
    
    // Register function
    g.RegisterQuery(ctx, "user", GetUser, "id")
    
    // Execute query
    query := `query { user(id: "123") { id name email } }`
    result := g.Execute(ctx, query, nil)
    
    // Check result
    if len(result.Errors) > 0 {
        t.Fatalf("unexpected errors: %v", result.Errors)
    }
    
    // Verify data
    data := result.Data.(map[string]interface{})
    user := data["user"].(map[string]interface{})
    assert.Equal(t, "123", user["id"])
}
```

### 6. Gradual Migration

You can run both systems side-by-side:
```go
// Serve both endpoints during migration
http.Handle("/graphql", oldGraphQLHandler)      // Existing
http.Handle("/graphql-new", g.HttpHandler())    // go-quickgraph

// Or use feature flags
if useNewGraphQL {
    http.Handle("/graphql", g.HttpHandler())
} else {
    http.Handle("/graphql", oldHandler)
}
```

## Quick Reference

### Type Mapping

| GraphQL Type | Traditional Schema | go-quickgraph |
|--------------|-------------------|---------------|
| String! | `String!` | `string` |
| String | `String` | `*string` |
| Int! | `Int!` | `int` |
| Float! | `Float!` | `float64` |
| Boolean! | `Boolean!` | `bool` |
| [String!]! | `[String!]!` | `[]string` |
| [String]! | `[String]!` | `[]*string` |
| Custom Object | `type User {...}` | `type User struct {...}` |
| Input Object | `input UserInput {...}` | `type UserInput struct {...}` |
| Enum | `enum Role {...}` | `type Role string` + `EnumValues()` |
| Interface | `interface Node {...}` | Embedded struct |
| Union | `union Result = ...` | Multiple return values |

### Operation Registration

| Operation | Traditional | go-quickgraph |
|-----------|------------|---------------|
| Query | Schema + Resolver | `g.RegisterQuery(ctx, name, func, params...)` |
| Mutation | Schema + Resolver | `g.RegisterMutation(ctx, name, func, params...)` |
| Subscription | Schema + Resolver | `g.RegisterSubscription(ctx, name, func, params...)` |

### Common Patterns

| Pattern | Traditional | go-quickgraph |
|---------|------------|---------------|
| Optional Field | `field: Type` | `Field *Type` |
| Required Field | `field: Type!` | `Field Type` |
| List | `[Type!]!` | `[]Type` |
| Validation | Middleware/Directives | `Validate()` method |
| Auth Check | Directives/Context | `ValidateWithContext()` |
| Lazy Loading | DataLoader | Type methods |
| Custom Scalar | Schema + Implementation | `RegisterScalar()` |

## Complete Migration Example

Let's migrate a complete blog API from gqlgen to go-quickgraph:

### Original gqlgen Schema

```graphql
# schema.graphql
type Query {
  post(id: ID!): Post
  posts(filter: PostFilter, pagination: PaginationInput): PostConnection!
  me: User
}

type Mutation {
  createPost(input: CreatePostInput!): Post!
  updatePost(id: ID!, input: UpdatePostInput!): Post!
  deletePost(id: ID!): Boolean!
  login(email: String!, password: String!): AuthPayload!
}

type Subscription {
  postCreated(authorId: ID): Post!
}

type Post {
  id: ID!
  title: String!
  content: String!
  author: User!
  tags: [String!]!
  published: Boolean!
  createdAt: DateTime!
  updatedAt: DateTime!
  comments: [Comment!]!
}

type User {
  id: ID!
  name: String!
  email: String!
  posts: [Post!]!
  role: UserRole!
}

type Comment {
  id: ID!
  content: String!
  author: User!
  post: Post!
  createdAt: DateTime!
}

enum UserRole {
  ADMIN
  EDITOR
  VIEWER
}

type PostConnection {
  edges: [PostEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type PostEdge {
  node: Post!
  cursor: String!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

type AuthPayload {
  token: String!
  user: User!
}

input CreatePostInput {
  title: String!
  content: String!
  tags: [String!]!
  published: Boolean
}

input UpdatePostInput {
  title: String
  content: String
  tags: [String!]
  published: Boolean
}

input PostFilter {
  authorId: ID
  published: Boolean
  tag: String
}

input PaginationInput {
  first: Int
  after: String
}

scalar DateTime
```

### Migrated go-quickgraph Implementation

```go
// types.go
package main

import (
    "context"
    "fmt"
    "time"
)

// Domain Types
type Post struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    Content   string    `json:"content"`
    AuthorID  string    `json:"authorId"`
    Tags      []string  `json:"tags"`
    Published bool      `json:"published"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}

// Lazy load author
func (p *Post) Author(ctx context.Context) (*User, error) {
    return getUserByID(ctx, p.AuthorID)
}

// Lazy load comments
func (p *Post) Comments(ctx context.Context) ([]*Comment, error) {
    return getCommentsByPostID(ctx, p.ID)
}

type User struct {
    ID    string   `json:"id"`
    Name  string   `json:"name"`
    Email string   `json:"email"`
    Role  UserRole `json:"role"`
}

// Lazy load posts
func (u *User) Posts(ctx context.Context) ([]*Post, error) {
    return getPostsByUserID(ctx, u.ID)
}

type Comment struct {
    ID        string    `json:"id"`
    Content   string    `json:"content"`
    AuthorID  string    `json:"authorId"`
    PostID    string    `json:"postId"`
    CreatedAt time.Time `json:"createdAt"`
}

func (c *Comment) Author(ctx context.Context) (*User, error) {
    return getUserByID(ctx, c.AuthorID)
}

func (c *Comment) Post(ctx context.Context) (*Post, error) {
    return getPostByID(ctx, c.PostID)
}

// Enum
type UserRole string

func (r UserRole) EnumValues() []string {
    return []string{"ADMIN", "EDITOR", "VIEWER"}
}

// Connection Types
type PostConnection struct {
    Edges      []*PostEdge `json:"edges"`
    PageInfo   *PageInfo   `json:"pageInfo"`
    TotalCount int         `json:"totalCount"`
}

type PostEdge struct {
    Node   *Post  `json:"node"`
    Cursor string `json:"cursor"`
}

type PageInfo struct {
    HasNextPage     bool    `json:"hasNextPage"`
    HasPreviousPage bool    `json:"hasPreviousPage"`
    StartCursor     *string `json:"startCursor"`
    EndCursor       *string `json:"endCursor"`
}

// Input Types
type CreatePostInput struct {
    Title     string   `json:"title"`
    Content   string   `json:"content"`
    Tags      []string `json:"tags"`
    Published *bool    `json:"published"`
}

// Validation runs automatically before CreatePost resolver
func (i CreatePostInput) Validate() error {
    if i.Title == "" {
        return fmt.Errorf("title is required")
    }
    if len(i.Title) > 200 {
        return fmt.Errorf("title must be less than 200 characters")
    }
    if i.Content == "" {
        return fmt.Errorf("content is required")
    }
    return nil
}

type UpdatePostInput struct {
    Title     *string   `json:"title"`
    Content   *string   `json:"content"`
    Tags      *[]string `json:"tags"`
    Published *bool     `json:"published"`
}

// Context-aware validation for authorization checks
func (i UpdatePostInput) ValidateWithContext(ctx context.Context) error {
    // At least one field must be provided
    if i.Title == nil && i.Content == nil && i.Tags == nil && i.Published == nil {
        return fmt.Errorf("at least one field must be provided for update")
    }
    return nil
}

type PostFilter struct {
    AuthorID  *string `json:"authorId"`
    Published *bool   `json:"published"`
    Tag       *string `json:"tag"`
}

type PaginationInput struct {
    First *int    `json:"first"`
    After *string `json:"after"`
}

type AuthPayload struct {
    Token string `json:"token"`
    User  *User  `json:"user"`
}

// queries.go
func GetPost(ctx context.Context, id string) (*Post, error) {
    return getPostByID(ctx, id)
}

func GetPosts(ctx context.Context, filter *PostFilter, pagination *PaginationInput) (*PostConnection, error) {
    // Default pagination
    limit := 10
    cursor := ""
    
    if pagination != nil {
        if pagination.First != nil {
            limit = *pagination.First
        }
        if pagination.After != nil {
            cursor = *pagination.After
        }
    }
    
    // Apply filters and fetch posts
    posts, total, hasNext, err := fetchPosts(ctx, filter, limit, cursor)
    if err != nil {
        return nil, err
    }
    
    // Build edges
    edges := make([]*PostEdge, len(posts))
    for i, post := range posts {
        edges[i] = &PostEdge{
            Node:   post,
            Cursor: encodeCursor(post.ID),
        }
    }
    
    // Build page info
    pageInfo := &PageInfo{
        HasNextPage:     hasNext,
        HasPreviousPage: cursor != "",
    }
    if len(edges) > 0 {
        start := edges[0].Cursor
        end := edges[len(edges)-1].Cursor
        pageInfo.StartCursor = &start
        pageInfo.EndCursor = &end
    }
    
    return &PostConnection{
        Edges:      edges,
        PageInfo:   pageInfo,
        TotalCount: total,
    }, nil
}

func Me(ctx context.Context) (*User, error) {
    userID := ctx.Value("userID")
    if userID == nil {
        return nil, fmt.Errorf("not authenticated")
    }
    return getUserByID(ctx, userID.(string))
}

// mutations.go
func CreatePost(ctx context.Context, input CreatePostInput) (*Post, error) {
    userID := ctx.Value("userID")
    if userID == nil {
        return nil, fmt.Errorf("authentication required")
    }
    
    post := &Post{
        ID:        generateID(),
        Title:     input.Title,
        Content:   input.Content,
        AuthorID:  userID.(string),
        Tags:      input.Tags,
        Published: false,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    
    if input.Published != nil {
        post.Published = *input.Published
    }
    
    if err := savePost(ctx, post); err != nil {
        return nil, fmt.Errorf("failed to create post: %w", err)
    }
    
    // Trigger subscription
    postCreatedChannel <- post
    
    return post, nil
}

func UpdatePost(ctx context.Context, id string, input UpdatePostInput) (*Post, error) {
    userID := ctx.Value("userID")
    if userID == nil {
        return nil, fmt.Errorf("authentication required")
    }
    
    post, err := getPostByID(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // Check ownership
    if post.AuthorID != userID.(string) {
        user, _ := getUserByID(ctx, userID.(string))
        if user.Role != "ADMIN" {
            return nil, fmt.Errorf("unauthorized to update this post")
        }
    }
    
    // Apply updates
    if input.Title != nil {
        post.Title = *input.Title
    }
    if input.Content != nil {
        post.Content = *input.Content
    }
    if input.Tags != nil {
        post.Tags = *input.Tags
    }
    if input.Published != nil {
        post.Published = *input.Published
    }
    post.UpdatedAt = time.Now()
    
    if err := savePost(ctx, post); err != nil {
        return nil, fmt.Errorf("failed to update post: %w", err)
    }
    
    return post, nil
}

func DeletePost(ctx context.Context, id string) (bool, error) {
    userID := ctx.Value("userID")
    if userID == nil {
        return false, fmt.Errorf("authentication required")
    }
    
    post, err := getPostByID(ctx, id)
    if err != nil {
        return false, err
    }
    
    // Check ownership
    if post.AuthorID != userID.(string) {
        user, _ := getUserByID(ctx, userID.(string))
        if user.Role != "ADMIN" {
            return false, fmt.Errorf("unauthorized to delete this post")
        }
    }
    
    if err := deletePostByID(ctx, id); err != nil {
        return false, fmt.Errorf("failed to delete post: %w", err)
    }
    
    return true, nil
}

func Login(ctx context.Context, email string, password string) (*AuthPayload, error) {
    user, err := authenticateUser(ctx, email, password)
    if err != nil {
        return nil, fmt.Errorf("invalid credentials")
    }
    
    token, err := generateJWT(user.ID)
    if err != nil {
        return nil, fmt.Errorf("failed to generate token: %w", err)
    }
    
    return &AuthPayload{
        Token: token,
        User:  user,
    }, nil
}

// subscriptions.go
var postCreatedChannel = make(chan *Post, 100)

func PostCreated(ctx context.Context, authorID *string) (<-chan *Post, error) {
    ch := make(chan *Post)
    
    go func() {
        defer close(ch)
        for {
            select {
            case <-ctx.Done():
                return
            case post := <-postCreatedChannel:
                // Filter by author if specified
                if authorID != nil && post.AuthorID != *authorID {
                    continue
                }
                
                select {
                case ch <- post:
                case <-ctx.Done():
                    return
                }
            }
        }
    }()
    
    return ch, nil
}

// main.go
package main

import (
    "context"
    "log"
    "net/http"
    
    "github.com/gburgyan/go-quickgraph"
)

func main() {
    // Initialize database
    db := initDatabase()
    
    // Create GraphQL server
    g := quickgraph.Graphy{
        MemoryLimits: &quickgraph.MemoryLimits{
            MaxRequestBodySize: 1024 * 1024, // 1MB
        },
        QueryLimits: &quickgraph.QueryLimits{
            MaxDepth:      10,
            MaxComplexity: 1000,
        },
        CORSSettings: quickgraph.DefaultCORSSettings(),
    }
    
    // Base context with database
    ctx := context.WithValue(context.Background(), "db", db)
    
    // Register DateTime scalar
    g.RegisterDateTimeScalar(ctx)
    
    // Register queries
    g.RegisterQuery(ctx, "post", GetPost, "id")
    g.RegisterQuery(ctx, "posts", GetPosts, "filter", "pagination")
    g.RegisterQuery(ctx, "me", Me)
    
    // Register mutations
    g.RegisterMutation(ctx, "createPost", CreatePost, "input")
    g.RegisterMutation(ctx, "updatePost", UpdatePost, "id", "input")
    g.RegisterMutation(ctx, "deletePost", DeletePost, "id")
    g.RegisterMutation(ctx, "login", Login, "email", "password")
    
    // Register subscriptions
    g.RegisterSubscription(ctx, "postCreated", PostCreated, "authorId")
    
    // Enable introspection for development
    g.EnableIntrospection(ctx)
    
    // Auth middleware
    authHandler := func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := r.Context()
            
            // Extract token from Authorization header
            token := extractToken(r)
            if token != "" {
                if userID, err := validateJWT(token); err == nil {
                    ctx = context.WithValue(ctx, "userID", userID)
                }
            }
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
    
    // Setup routes
    http.Handle("/graphql", authHandler(g.HttpHandler()))
    http.Handle("/subscriptions", authHandler(g.WebsocketHandler()))
    
    log.Println("🚀 GraphQL server running at http://localhost:8080/graphql")
    log.Println("🔌 WebSocket subscriptions at ws://localhost:8080/subscriptions")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

## Conclusion

Migrating to go-quickgraph simplifies your GraphQL development by:

1. **Eliminating schema files** - Your Go code is the schema
2. **Removing code generation** - No build steps or generated files
3. **Improving type safety** - Full compile-time checking
4. **Reducing boilerplate** - Direct function registration
5. **Maintaining flexibility** - All GraphQL features still available

The code-first approach aligns naturally with Go's philosophy of simplicity and explicitness, making your GraphQL APIs easier to build, test, and maintain.

For more examples and advanced patterns, check out:
- [Sample Application](https://github.com/gburgyan/go-quickgraph-sample)
- [API Documentation](https://pkg.go.dev/github.com/gburgyan/go-quickgraph)
- [GitHub Repository](https://github.com/gburgyan/go-quickgraph)