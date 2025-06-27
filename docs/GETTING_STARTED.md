# Getting Started with go-quickgraph

This guide will get you up and running with go-quickgraph in under 5 minutes.

## Installation

```bash
go get github.com/gburgyan/go-quickgraph
```

**Requirements:**
- Go 1.21 or later
- No additional dependencies required for basic usage

## Your First GraphQL API

Let's build a simple book store API step by step.

### Step 1: Define Your Data Types

```go
package main

import (
    "context"
    "net/http"
    "github.com/gburgyan/go-quickgraph"
)

// Define your domain types - just regular Go structs!
type Book struct {
    ID     int    `graphy:"id"`
    Title  string `graphy:"title"`
    Author string `graphy:"author"`
    Year   int    `graphy:"year"`
}

type Author struct {
    ID   int    `graphy:"id"`
    Name string `graphy:"name"`
    Bio  string `graphy:"bio"`
}
```

### Step 2: Create Handler Functions

Write regular Go functions - no special interfaces to implement:

```go
// Sample data (in real apps, this would come from a database)
var books = []Book{
    {ID: 1, Title: "The Go Programming Language", Author: "Alan Donovan", Year: 2015},
    {ID: 2, Title: "Clean Code", Author: "Robert Martin", Year: 2008},
}

var authors = []Author{
    {ID: 1, Name: "Alan Donovan", Bio: "Co-author of The Go Programming Language"},
    {ID: 2, Name: "Robert Martin", Bio: "Software craftsman and author"},
}

// Query: Get a book by ID
func GetBook(ctx context.Context, id int) (*Book, error) {
    for _, book := range books {
        if book.ID == id {
            return &book, nil
        }
    }
    return nil, fmt.Errorf("book with ID %d not found", id)
}

// Query: Get all books
func GetBooks(ctx context.Context) []Book {
    return books
}

// Mutation: Add a new book
func AddBook(ctx context.Context, input BookInput) (*Book, error) {
    newBook := Book{
        ID:     len(books) + 1,
        Title:  input.Title,
        Author: input.Author,
        Year:   input.Year,
    }
    books = append(books, newBook)
    return &newBook, nil
}

// Input type for mutations
type BookInput struct {
    Title  string `graphy:"title"`
    Author string `graphy:"author"`
    Year   int    `graphy:"year"`
}
```

### Step 3: Register Functions and Start Server

```go
func main() {
    ctx := context.Background()
    
    // Create the GraphQL server
    g := quickgraph.Graphy{}
    
    // Register your functions as GraphQL operations
    g.RegisterQuery(ctx, "book", GetBook, "id")
    g.RegisterQuery(ctx, "books", GetBooks)
    g.RegisterMutation(ctx, "addBook", AddBook, "input")
    
    // Enable GraphQL Playground for development
    g.EnableIntrospection(ctx)
    
    // Start the server
    http.Handle("/graphql", g.HttpHandler())
    
    fmt.Println("🚀 GraphQL server running at http://localhost:8080/graphql")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Step 4: Test Your API

Visit http://localhost:8080/graphql and try these queries:

**Get all books:**
```graphql
{
  books {
    id
    title
    author
    year
  }
}
```

**Get a specific book:**
```graphql
{
  book(id: 1) {
    title
    author
  }
}
```

**Add a new book:**
```graphql
mutation {
  addBook(input: {
    title: "Effective Go"
    author: "The Go Team"
    year: 2023
  }) {
    id
    title
  }
}
```

## What Just Happened?

1. **No Schema Definition**: go-quickgraph generated the GraphQL schema automatically from your Go types
2. **Type Safety**: All operations are fully type-checked at compile time
3. **Zero Boilerplate**: Your functions became GraphQL operations with just a single registration call
4. **Automatic Serialization**: Go structs are automatically converted to GraphQL types

## Generated Schema

go-quickgraph automatically generated this GraphQL schema from your Go code:

```graphql
type Book {
  id: Int!
  title: String!
  author: String!
  year: Int!
}

type Query {
  book(id: Int!): Book
  books: [Book!]!
}

type Mutation {
  addBook(input: BookInput!): Book
}

input BookInput {
  title: String!
  author: String!
  year: Int!
}
```

## Next Steps

Now that you have a basic GraphQL API running, explore these topics:

### Essential Concepts
- **[Core Concepts](CORE_CONCEPTS.md)** - Understand how go-quickgraph works
- **[Basic Operations](BASIC_OPERATIONS.md)** - Queries, mutations, and error handling
- **[Function Patterns](FUNCTION_PATTERNS.md)** - Different ways to structure your handlers

### Common Use Cases
- **[Type System Guide](TYPE_SYSTEM.md)** - Interfaces, unions, enums, and relationships
- **[Custom Scalars](CUSTOM_SCALARS.md)** - DateTime, Money, and validation
- **[Authentication](AUTH_PATTERNS.md)** - Securing your API

### Advanced Features
- **[Subscriptions](SUBSCRIPTIONS.md)** - Real-time updates with WebSockets
- **[Performance](PERFORMANCE.md)** - Caching and optimization
- **[Schema Customization](SCHEMA.md)** - Advanced schema generation

## Common Pitfalls

### 1. JSON Tags for Field Control
```go
// ✅ Without json tags - uses Go field names
type User struct {
    Name  string  // GraphQL field: "Name"
    Email string  // GraphQL field: "Email"
}

// ✅ With json tags - more control over GraphQL field names
type User struct {
    Name  string `graphy:"name"`   // GraphQL field: "name"
    Email string `graphy:"email"`  // GraphQL field: "email"
}
```

### 2. Not Handling Errors
```go
// ❌ Panics may crash your server
func GetUser(id int) User {
    return users[id] // Panic if ID doesn't exist
}

// ✅ Return errors for GraphQL error handling
func GetUser(id int) (*User, error) {
    if id >= len(users) {
        return nil, fmt.Errorf("user not found")
    }
    return &users[id], nil
}
```

### 3. Forgetting Context
```go
// ❌ No context for cancellation or auth
func GetUser(id int) *User { ... }

// ✅ Always accept context for production apps
func GetUser(ctx context.Context, id int) *User { ... }
```

## Example Applications

- **[Complete Sample App](https://github.com/gburgyan/go-quickgraph-sample)** - Full-featured example with authentication, subscriptions, and more
- **[Simple REST Migration](examples/rest-to-graphql)** - Converting a REST API to GraphQL
- **[Real-time Chat](examples/chat-app)** - WebSocket subscriptions example

Ready to dive deeper? Continue to [Core Concepts](CORE_CONCEPTS.md) to understand how go-quickgraph works under the hood.