![Build status](https://github.com/gburgyan/go-quickgraph/actions/workflows/go.yml/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/gburgyan/go-quickgraph)](https://goreportcard.com/report/github.com/gburgyan/go-quickgraph) [![PkgGoDev](https://pkg.go.dev/badge/github.com/gburgyan/go-quickgraph)](https://pkg.go.dev/github.com/gburgyan/go-quickgraph)

# go-quickgraph

**A code-first GraphQL library for Go** - Write regular Go functions and structs, get a full GraphQL API.

## Quick Start

```bash
go get github.com/gburgyan/go-quickgraph
```

```go
package main

import (
    "context"
    "fmt"
    "github.com/gburgyan/go-quickgraph"
)

type User struct {
    Name  string `json:"name"
    Email string `json:"email"`
}

func main() {
    ctx := context.Background()
    g := quickgraph.Graphy{}
    
    // Register a simple query - just a regular Go function!
    g.RegisterQuery(ctx, "user", func(ctx context.Context, id int) *User {
        return &User{Name: "Alice", Email: "alice@example.com"}
    }, "id")
    
    // That's it! You have a GraphQL API
    query := `{ user(id: 1) { name email } }`
    result, _ := g.ProcessRequest(ctx, query, "")
    fmt.Println(result) // {"data":{"user":{"name":"Alice","email":"alice@example.com"}}}
}
```

Want to serve it over HTTP? Just add:

```go
g.EnableIntrospection(ctx) // Enable GraphQL playground
http.Handle("/graphql", g.HttpHandler())
http.ListenAndServe(":8080", nil)
```

Now visit http://localhost:8080/graphql to explore your API with GraphQL playground!

## Custom HTTP Implementation

The built-in `GraphHttpHandler` implementation can be used as a template if you need custom functionality like special logging, context injection, authentication middleware, or custom error handling. The actual GraphQL processing is handled by `ProcessRequest` - everything in `GraphHttpHandler` is just a standard HTTP handler implementation that you can customize as needed.

## Why Code-First?

**Traditional Schema-First Approach:**
1. Write GraphQL schema (SDL)
2. Generate Go code from schema
3. Implement resolver interfaces
4. Keep schema and code in sync

**go-quickgraph Code-First Approach:**
1. Write regular Go functions and structs
2. That's it - GraphQL schema is generated automatically!

### Benefits

✅ **No Schema Files** - Your Go code is the single source of truth

✅ **Type Safety** - Full Go type checking, no runtime type mismatches

✅ **IDE Support** - Autocomplete, refactoring, and all your favorite Go tools work naturally

✅ **No Code Generation** - No build steps, no generated files to maintain

✅ **Minimal Boilerplate** - Register a function, it becomes a GraphQL operation

## About

`go-quickgraph` is a code-first GraphQL library that generates your GraphQL schema from your Go code. Write idiomatic Go, get a fully-compliant GraphQL API.

## Table of Contents

- [Quick Start](#quick-start)
- [Why Code-First?](#why-code-first)
- [Installation](#installation)
- [Basic Examples](#basic-examples)
- [Design Goals](#design-goals)
- [Common Patterns](#common-patterns)
- [Advanced Features](#advanced-features)
- [API Reference](#api-reference)

# Design Goals

* Minimal boilerplate code. Write Go functions and structs, and the library will take care of the rest.
* Ability to process all valid GraphQL queries. This includes queries with fragments, aliases, variables, and directives.
* Generation of a GraphQL schema from the set-up GraphQL environment.
* Be as idiomatic as possible. This means using Go's native types and idioms as much as possible. If you are familiar with Go, you should be able to use this library without having to learn a lot of new concepts.
* If we need to relax something in the processing of a GraphQL query, err on the side of preserving Go idioms, even if it means that we are not 100% compliant with the GraphQL specification.
* Attempt to allow anything legal in GraphQL to be expressed in Go. Make the common things as easy as possible.
* Be as fast as practical. Cache aggressively.

The last point is critical in thinking about how the library is built. It relies heavily on Go reflection to figure out what the interfaces are of the functions and structs. Certain things, like function parameter names, are not available at runtime, so we have to make some compromises. If, for instance, we have a function that takes two non-context arguments, we can process requests assuming that the order of the parameters in the request matches the order from the function. This is not 100% compliant with the GraphQL specification, but it is a reasonable compromise to make in order to preserve Go idioms and minimize boilerplate code.

# Installation

```bash
go get github.com/gburgyan/go-quickgraph
```

Requires Go 1.21 or later.

# Basic Examples

## 1. Simple Query

The simplest possible GraphQL API - just register a Go function:

```go
// Define your data types
type Product struct {
    ID    int     `json:"id"`
    Name  string  `json:"name"`
    Price float64 `json:"price"`
}

// Create and configure your GraphQL API
g := quickgraph.Graphy{}
g.RegisterQuery(ctx, "product", func(ctx context.Context, id int) *Product {
    // Your business logic here
    return &Product{ID: id, Name: "Widget", Price: 9.99}
}, "id")
```

That's it! This creates a GraphQL query:
```graphql
query {
  product(id: 123) {
    id
    name
    price
  }
}
```

## 2. Query with Multiple Parameters

```go
// Using multiple named parameters
g.RegisterQuery(ctx, "search", func(ctx context.Context, query string, limit int) []Product {
    // Search logic here
    return searchProducts(query, limit)
}, "query", "limit")

// Or use a struct for parameters (recommended for 3+ params)
type SearchInput struct {
    Query  string   `json:"query"`
    Limit  int      `json:"limit"`
    Filter []string `json:"filter"`
}

g.RegisterQuery(ctx, "searchProducts", func(ctx context.Context, input SearchInput) []Product {
    // Search with complex parameters
    return advancedSearch(input)
}, "input")
```

## 3. Mutations

```go
type CreateUserInput struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

// Mutations follow the same pattern as queries
g.RegisterMutation(ctx, "createUser", func(ctx context.Context, input CreateUserInput) (*User, error) {
    // Validation
    if input.Email == "" {
        return nil, fmt.Errorf("email is required")
    }
    
    // Create user
    user := &User{
        ID:    generateID(),
        Name:  input.Name,
        Email: input.Email,
    }
    saveUser(user)
    return user, nil
}, "input")
```

## 4. Working with Enums

```go
type Status string

// Implement EnumValues for GraphQL schema generation
func (s Status) EnumValues() []string {
    return []string{"ACTIVE", "INACTIVE", "PENDING"}
}

type User struct {
    Name   string `json:"name"`
    Status Status `json:"status"`
}
```

## 5. Nested Objects and Relationships

```go
type Author struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

type Book struct {
    Title  string  `json:"title"`
    Author *Author `json:"author"`  // Nested object
}

// Methods on structs become GraphQL fields
func (a *Author) Books(ctx context.Context) []Book {
    // Load books for this author
    return getBooksByAuthor(a.ID)
}

// Register the query
g.RegisterQuery(ctx, "author", getAuthorByID, "id")
```

This automatically generates the schema:
```graphql
type Author {
  id: Int!
  name: String!
  books: [Book!]!
}

type Book {
  title: String!
  author: Author
}
```

## 6. Subscriptions (Real-time Updates)

```go
// Subscriptions return channels
g.RegisterSubscription(ctx, "orderStatus", func(ctx context.Context, orderID string) (<-chan OrderUpdate, error) {
    updates := make(chan OrderUpdate)
    
    go func() {
        defer close(updates)
        // Send updates when order status changes
        for {
            select {
            case <-ctx.Done():
                return
            case update := <-getOrderUpdates(orderID):
                updates <- update
            }
        }
    }()
    
    return updates, nil
}, "orderID")
```

## Complete Example

Here's a complete example from the GraphQL [documentation examples](https://graphql.org/learn/queries/):

```go
package main

import (
    "context"
    "net/http"
    "github.com/gburgyan/go-quickgraph"
)

type Character struct {
    Id        string       `json:"id"`
    Name      string       `json:"name"`
    Friends   []*Character `json:"friends"`
    AppearsIn []Episode    `json:"appearsIn"`
}

type Episode string

func (e Episode) EnumValues() []string {
    return []string{"NEWHOPE", "EMPIRE", "JEDI"}
}

func GetHero(ctx context.Context, episode Episode) *Character {
    // Return different heroes based on episode
    if episode == "EMPIRE" {
        return &Character{
            Id:        "1000",
            Name:      "Luke Skywalker",
            AppearsIn: []Episode{"NEWHOPE", "EMPIRE", "JEDI"},
        }
    }
    return &Character{
        Id:        "2001",
        Name:      "R2-D2",
        AppearsIn: []Episode{"NEWHOPE", "EMPIRE", "JEDI"},
    }
}

func main() {
    ctx := context.Background()
    
    // Create GraphQL server
    g := quickgraph.Graphy{}
    g.RegisterQuery(ctx, "hero", GetHero, "episode")
    g.EnableIntrospection(ctx)
    
    // Serve GraphQL API
    http.Handle("/graphql", g.HttpHandler())
    http.ListenAndServe(":8080", nil)
}
```

For more examples, check out the [go-quickgraph-sample](https://github.com/gburgyan/go-quickgraph-sample) repository.

# Theory of Operation

## Initialization of the Graphy Object

As the above example illustrates, the first step is to create a `Graphy` object. The intent is that an instance of this object is set up at service initialization time, and then used to process requests. The `Graphy` object is thread-safe, so it can be used concurrently.

The process is to add all the functions that can be used to service queries and mutations. The library will use reflection to figure out what the function signatures are, and will use that information to process the requests. The library will also use reflection to figure out what the types are of the parameters and return values, and will use that information to process the requests and generate the GraphQL schema.

Functions are simple Go `func`s that handle the processing. There are two broad "flavors" of functions: struct-based and anonymous. Since Go doesn't permit reflection of the parameters of functions we a few options. If the function takes one non-`Context` `struct` parameter, it will be treated as struct-based. The fields of the `struct` will be used to figure out the function signature. Otherwise, the function is an anonymous function. During the addition of a function, the ordered listing of parameter names can be passed. This will be covered in more detail further in this `README`.

Once all the functions are added, the `Graphy` object is ready to be used.

## Processing of a Request

Internally, a request is processed in four primary phases:

### Query Processing:

The inbound query is parsed into a `RequestStub` object. Since a request can have variables, the determination of the variable types can be done once, then cached. This is important because in complex cases, this can be an expensive operation as both the input and potential outputs must be considered.

### Variable Processing

If there are variables to be processed, they are unmarshalled using JSON. The result objects are then mapped into the variables that have been defined by the preceding step. Variables can represent both simple scalar types and more complex object graphs.

### Function Execution

Once the variables have been defined, the query or mutator functions can be called. This will call a function that was initially registered with the `Graphy` instance. 

### Output Generation

The most complex part of the overall request processing is the generation of the result object graph. All aspects of the standard GraphQL query language are supported. See the section on the [type systems](#type-system) for more information about how this operates.

Another aspect that GraphQL supports is operations that function on the returned data. This library models that as receiver functions on results variables:

```go

func (c *Character) FriendsConnection(first int) *FriendsConnection {
	// Processing goes here...
    return result
}

type FriendsConnection struct {
    TotalCount int               `json:"totalCount"`
    Edges      []*ConnectionEdge `json:"edges"`
}
```

This, in addition to the earlier example, will expose a function on the result object that is represented by a schema such as:

```graphql
type Character {
	appearsIn: [episode!]!
	friends: [Character]!
	FriendsConnection(arg1: Int!): FriendsConnection
	id: String!
	name: String!
}
```

There are several important things here:

* All public fields of the `Character` struct are reflected in the schema.
* The `json` tags are respected for the field naming
* The `FriendsConnection` receiver function is an [anonymous function](#anonymous-functions).

There is a further discussion on [schemata](#schema-generation) later on.

## Error handling

There are two general places where errors can occur: setup and runtime. During setup, the library will generally `panic` as this is something that should fail fast and indicates a structural problem with the program itself. At runtime, there should be no way that the system can `panic`.

When calling `result, err := g.ProcessRequest(ctx, input, "{variables...})` the result should contain something that can be returned to a GraphQL client in all cases. The errors, if any, are formatted the way that a GraphQL client would expect. So, for instance:

```json
{
  "data": {},
  "errors": [
    {
      "message": "error getting call parameters for function hero: invalid enum value INVALID",
      "locations": [
        {
          "line": 3,
          "column": 8
        }
      ],
      "path": [
        "hero"
      ]
    }
  ]
}
```

The error is also returned by the function itself and should be able to be handled normally.

# Functions

Functions are used in two ways in the processing of a `Graphy` request:

* Handling the primary query processing
* Providing processing while rendering the results from the output from the primary function

Any time a request is processed, it gets mapped to the function that is registered for that name. This is done with one of the `Register*` functions on the `Graphy` object.

In an effort to make it simple for developers to create these functions, there are several ways to add functions and `Graphy` that each have some advantages and disadvantages.

In all cases, if there is a `context.Context` parameter, that will be passed into the function.

## Anonymous Functions

Anonymous functions look like typical Golang functions and simply take a normal parameter list.

Given this simple function:

```go
func GetCourses(ctx context.Context, categories []*string) []*Course
```

You can register the processor with:

```go
g := Graphy{}
g.RegisterQuery(ctx, "courses", GetCourses)
```

The downside of this approach is based on a limitation in reflection in Golang: there is no way to get the _names_ of the parameters. In this case `Graphy` will run assuming the parameters are positional. This is a variance from orthodox GraphQL behavior as it expects named parameters. If a schema is generated from the `Graphy`, these will be rendered as `arg1`, `arg2`, etc. Those are purely placeholders for the schema and the processing of the request will still be treated positionally; the names of the passed parameters are ignored.

If the named parameters are needed, you can do that via:

```go
g := Graphy{}
g.RegisterQuery(ctx, "courses", GetCourses, "categories")
```

Once you tell `Graphy` the names of the paramters to expect, it can function with named parameters as is typical for GraphQL.

Finally, you can also register the processor with the most flexible function:

```go
g := Graphy{}
g.RegisterFunction(ctx, graphy.FunctionDefinition{
	Name: "courses",
	Function: GetCourses,
	ParameterNames: {"categories"},
	Mode: graphy.ModeQuery,
})
```

This is also how you can register a mutation instead of the default query.

A function may also take zero non-`Context` parameters, in which case it simply gets run.

## Struct Functions

The alternate way that a function can be defined is by passing in a struct as the parameter. The structure must be passed as value. In addition to the struct parameter, it may also, optionally, have a `context.Context` parameter.

For example:

```go
type CourseInput struct {
	Categories []string
}

func GetCourses(ctx context.Context, in CourseInput) []*Course
{
	// Implementation
}

g := Graphy{}
g.RegisterQuery(ctx, "courses", GetCourses)
```

Since reflection can be used to get the names of the members of the structure, that information will be used to get the names of the parameters that will be exposed for the function.

The same `RegisterFunction` function can also be used as described above to define additional aspects of the function, such as if it's a mutator.

### Anonymous Fields (Embedded Structs)

Struct functions support Go's anonymous fields (embedded structs), allowing you to compose input types and promote fields to the parent level in GraphQL. This is particularly useful for sharing common fields across multiple input types.

When a struct contains anonymous fields, the fields from the embedded struct are promoted and become direct arguments in the GraphQL function:

```go
// Common fields that multiple inputs might share
type PaginationInput struct {
    Limit  int `json:"limit"`
    Offset int `json:"offset"`
}

// SearchInput embeds PaginationInput
type SearchInput struct {
    PaginationInput  // anonymous embedding
    Query   string   `json:"query"`
    Tags    []string `json:"tags"`
}

func Search(ctx context.Context, input SearchInput) []SearchResult {
    // Can access input.Limit, input.Offset directly
    // as well as input.Query and input.Tags
}

g.RegisterQuery(ctx, "search", Search)
```

In GraphQL, this allows queries like:
```graphql
{
    search(query: "golang", limit: 10, offset: 0, tags: ["tutorial"]) {
        id
        title
    }
}
```

Notice how `limit` and `offset` are promoted to be direct arguments alongside `query` and `tags`, rather than being nested in a sub-object.

#### Pointer Embedding

You can also embed structs via pointers, which makes all the embedded fields optional:

```go
type FilterOptions struct {
    MinPrice *float64 `json:"minPrice"`
    MaxPrice *float64 `json:"maxPrice"`
}

type ProductSearchInput struct {
    PaginationInput      // Required fields from value embedding
    *FilterOptions       // Optional fields from pointer embedding
    Query       string   `json:"query"`
    SortBy      string   `json:"sortBy"`
}

func SearchProducts(ctx context.Context, input ProductSearchInput) []Product {
    // FilterOptions is automatically initialized if any of its fields are provided
    if input.FilterOptions != nil && input.MinPrice != nil {
        // Apply minimum price filter
    }
}
```

This creates a GraphQL function where `limit` and `offset` are required (from `PaginationInput`), while `minPrice` and `maxPrice` are optional (from `*FilterOptions`).

#### Benefits

1. **Code Reuse**: Common fields like pagination can be defined once and embedded in multiple input types
2. **Cleaner APIs**: Fields are promoted to the top level, avoiding nested input objects
3. **Go Idiomatic**: Follows Go's composition patterns naturally
4. **Backward Compatible**: Adding fields to an embedded struct automatically makes them available in all structs that embed it

#### Limitations

- Nested anonymous fields (anonymous fields within anonymous fields) are not currently supported
- Field name conflicts between the parent and embedded structs follow Go's rules (parent wins)
- JSON tags on embedded struct fields are respected

## Return Values

Regardless of how the function is defined, it is required to return a struct, a pointer to a struct, or a slice of either. It may optionally return an `error` as well. The returned value will be used to populate the response to the GraphQL calls. The shape of the response object will be used to construct the schema of the `Graphy` in case that is used.

There is a special case where a function can return an `any` type. This is valid from a runtime perspective as the type of the object can be determined at runtime, but it precludes schema generation for the result as the type of the result cannot be determined by the signature of the function.

### GraphQL Interfaces and Unions

GraphQL supports interfaces and unions, which allow a field to return different concrete types. `go-quickgraph` provides multiple approaches to handle these cases:

#### Approach 1: Multiple Return Values (Union-like)

The simplest approach is to return multiple pointers where only one is non-nil:

```go
func CreateContent(input ContentInput) (*Article, *Video, error) {
    if input.Type == "article" {
        article := &Article{Title: input.Title, Body: input.Body}
        return article, nil, nil
    } else {
        video := &Video{Title: input.Title, Duration: input.Duration}
        return nil, video, nil
    }
}
```

**Pros:**
- Simple and explicit
- Clear at the function signature what types can be returned
- No additional setup required

**Cons:**
- Function signatures become unwieldy with many types
- Callers must check multiple return values
- Not idiomatic Go for functions with many possible return types

#### Approach 2: Explicit Union Types

Define a struct with "Union" suffix containing pointers to possible types:

```go
type ContentResultUnion struct {
    Article *Article
    Video   *Video
}

func CreateContent(input ContentInput) (ContentResultUnion, error) {
    if input.Type == "article" {
        return ContentResultUnion{Article: &Article{Title: input.Title}}, nil
    } else {
        return ContentResultUnion{Video: &Video{Title: input.Title}}, nil
    }
}
```

**Pros:**
- Clean function signatures
- Type-safe at compile time
- Easy to extend with new types

**Cons:**
- Requires defining union types
- Callers must check which field is non-nil

#### Approach 3: Interface with Type Discovery

For GraphQL interfaces (common fields with different implementations), use embedded structs with optional type discovery:

```go
// Base type with common fields
type Content struct {
    ID    string
    Title string
    // Optional: enable type discovery
    actualType interface{} `json:"-" graphy:"-"`
}

// Implement TypeDiscoverable for runtime type resolution
func (c *Content) ActualType() interface{} {
    if c.actualType != nil {
        return c.actualType
    }
    return c
}

// Concrete types embed the base type
type Article struct {
    Content
    Body string
}

type Video struct {
    Content
    Duration int
}

// Constructor enables type discovery
func NewArticle(id, title, body string) *Article {
    a := &Article{
        Content: Content{ID: id, Title: title},
        Body:    body,
    }
    a.Content.actualType = a // Enable type discovery
    return a
}

// Functions can return the base type
func GetContent(id string) (*Content, error) {
    // In practice, load from database
    if id == "article-1" {
        article := NewArticle(id, "My Article", "Article body...")
        return &article.Content, nil
    } else {
        video := NewVideo(id, "My Video", 120)
        return &video.Content, nil
    }
}
```

**Pros:**
- Natural Go interfaces and embedding
- Functions return single, clean types
- Supports GraphQL fragments and introspection
- Optional - only add type discovery where needed
- Zero overhead for types that don't need discovery

**Cons:**
- Requires constructor functions for discoverable types
- Additional field in base types (8 bytes per instance)
- Slightly more complex setup

#### When to Use Each Approach

1. **Multiple Return Values**: Best for simple cases with 2-3 types, especially for mutations that create different types based on input.

2. **Union Types**: Ideal when you have a clear set of unrelated types that can be returned, like search results that might return Products, Users, or Orders.

3. **Type Discovery**: Perfect for GraphQL interfaces where types share common fields and behavior. This approach scales well and provides the best GraphQL experience with proper fragment support.

The type discovery approach is particularly powerful because it allows your Go code to follow natural patterns (returning interface or base types) while still providing full GraphQL type information at runtime.

## Output Functions

When calling a function to service a request, that function returns the value that is processed into the response -- that part is obvious. Another feature is that those objects can have functions on them as well. This plays into the overall Graph functionality that is exposed by `Graphy`. These receiver functions follow the same pattern as above.

Additionally, they get transformed into schemas exactly as expected. If a receiver function takes nothing (or only a `context.Context` object), then it gets exposed as a field. If the field is referenced, then the function is invoked and the output generation continues. If the function takes parameters, then it's exposed as a function with parameters both for the request as well as the schema:

```go
type Human struct {
	Character
	HeightMeters float64 `json:"HeightMeters"`
}

func (h *Human) Height(units *string) float64 {
	if units == nil {
		return h.HeightMeters
	}
	if *units == "FOOT" {
		return roundToPrecision(h.HeightMeters*3.28084, 7)
	}
	return h.HeightMeters
}
```

Since the `Height` function takes a pointer to a `string` as a parameter, it's treated as optional.

In this case both of the following queries will work:

```graphql
{
  Human(id: "1000") {
    name
    height
  }
}
```

```graphql
{
    Human(id: "1000") {
        name
        height(unit: FOOT)
    }
}
```

## Function Parameters

Regardless of how the function is invoked, the parameters for the function come from either the base query itself or variables that are passed in along with the query. `Graphy` supports both scalar types, as well as more complex types including complex, and even nested, structures, as well as slices of those objects.

## Explicit Parameter Modes

By default, `go-quickgraph` automatically detects how to handle function parameters based on the function signature. However, you can explicitly specify the parameter mode for clearer intent and better error messages:

```go
type ParameterMode int

const (
    AutoDetect      // Default: automatic detection (backward compatible)
    StructParams    // Explicitly use a struct for parameters
    NamedParams     // Explicitly use named inline parameters
    PositionalParams // Explicitly use positional parameters
)
```

### StructParams Mode

Use when you want parameters to be fields of a struct:

```go
type UserInput struct {
    ID     string  `json:"id"`
    Name   *string `json:"name"`    // optional
    Active *bool   `json:"active"`  // optional
}

func GetUser(ctx context.Context, input UserInput) *User {
    // Implementation
}

g.RegisterFunction(ctx, FunctionDefinition{
    Name:          "getUser",
    Function:      GetUser,
    ParameterMode: StructParams,
})
```

GraphQL query:
```graphql
{
    getUser(id: "123", name: "John") {
        id
        name
    }
}
```

### NamedParams Mode

Use when you want multiple named parameters:

```go
func GetUser(ctx context.Context, id string, name *string, active *bool) *User {
    // Implementation
}

g.RegisterFunction(ctx, FunctionDefinition{
    Name:           "getUser",
    Function:       GetUser,
    ParameterMode:  NamedParams,
    ParameterNames: []string{"id", "name", "active"},
})
```

GraphQL query uses the provided names:
```graphql
{
    getUser(id: "123", active: true) {
        id
        name
    }
}
```

### PositionalParams Mode

Use when parameter order matters more than names:

```go
func SearchUsers(ctx context.Context, query string, limit int) []*User {
    // Implementation
}

g.RegisterFunction(ctx, FunctionDefinition{
    Name:          "searchUsers",
    Function:      SearchUsers,
    ParameterMode: PositionalParams,
})
```

GraphQL query uses positional arguments (arg1, arg2, etc.):
```graphql
{
    searchUsers(arg1: "john", arg2: 10) {
        id
        name
    }
}
```

### Benefits of Explicit Modes

1. **Clear Intent**: Your code explicitly states how parameters should be handled
2. **Better Validation**: Get specific error messages when configuration doesn't match the function signature
3. **No Ambiguity**: Avoid confusion about which mode was auto-selected
4. **Self-Documenting**: The parameter mode serves as documentation

### Validation Rules

- **StructParams**: Requires exactly one non-context struct parameter
- **NamedParams**: Requires ParameterNames to be set and match parameter count
- **PositionalParams**: Cannot be used with ParameterNames (will panic)
- **AutoDetect**: Falls back to original behavior for backward compatibility

### AutoDetect Behavior (Default)

When `ParameterMode` is not specified or set to `AutoDetect`, the library uses these rules to determine parameter handling:

1. **No parameters** (excluding context.Context):
   - Function is called without arguments
   - Used for resolver functions that don't need input

2. **Single struct parameter**:
   - If the single parameter is a struct AND no ParameterNames are provided
   - Struct fields become GraphQL arguments (uses `StructParams` mode)
   - Example: `func GetUser(ctx context.Context, input UserInput) *User`

3. **Single non-struct parameter OR struct with ParameterNames**:
   - Treated as anonymous inline parameter (uses `AnonymousParamsInline` mode)
   - If ParameterNames provided, uses those names; otherwise uses `arg1`
   - Example: `func GetUserByID(ctx context.Context, id string) *User`

4. **Multiple parameters**:
   - Always treated as anonymous inline parameters (uses `AnonymousParamsInline` mode)
   - If ParameterNames provided, uses those names; otherwise uses `arg1`, `arg2`, etc.
   - Example: `func SearchUsers(ctx context.Context, query string, limit int) []*User`

**Important Notes**:
- `context.Context` parameters are always ignored when counting parameters
- Pointer types indicate optional parameters (can be nil)
- Non-pointer types are required parameters

Example of AutoDetect in action:
```go
// Case 1: Struct parameter - fields become arguments
type UserInput struct {
    ID   string
    Name *string // optional
}
g.RegisterQuery(ctx, "getUser", func(ctx context.Context, input UserInput) *User {
    // GraphQL: getUser(id: "123", name: "John")
})

// Case 2: Single non-struct - becomes arg1
g.RegisterQuery(ctx, "getUserByID", func(ctx context.Context, id string) *User {
    // GraphQL: getUserByID(arg1: "123")
})

// Case 3: Multiple parameters - become arg1, arg2, etc.
g.RegisterQuery(ctx, "searchUsers", func(ctx context.Context, query string, limit int) []*User {
    // GraphQL: searchUsers(arg1: "john", arg2: 10)
})

// Case 4: With parameter names - uses provided names
g.RegisterFunction(ctx, FunctionDefinition{
    Name:           "searchUsers",
    Function:       searchUsersFunc,
    ParameterNames: []string{"query", "limit"},
    // GraphQL: searchUsers(query: "john", limit: 10)
})
```

# Type System

The way `Graphy` works with types is intended to be as transparent to the user as possible. The normal types, scalars, structs, and slices all work as expected. This applies to both input in the form of parameters being sent in to functions and the results of those functions.

Maps, presently, are not supported.

## Enums

Go doesn't have a native way of representing enumerations in a way that is open to be used for reflection. To get around this, `Graphy` provides a few different ways of exposing enumerations.

### Simple strings

### `EnumUnmarshaler` interface

If you need to have function inputs that map to specific non-string inputs, you can implement the `EnumUnmarshaler` interface:

```go
// EnumUnmarshaler provides an interface for types that can unmarshal
// a string representation into their enumerated type. This is useful
// for types that need to convert a string, typically from external sources
// like JSON or XML, into a specific enumerated type in Go.
//
// UnmarshalString should return the appropriate enumerated value for the
// given input string, or an error if the input is not valid for the enumeration.
type EnumUnmarshaler interface {
	UnmarshalString(input string) (interface{}, error)
}
```

`UnmarshalString` is called with the supplied identifier, and is responsible for converting that into whatever type is needed. If the identifier cannot be converted, simply return an error.

The downside of this is that there is no way to communicate to the schema what are the _valid_ values for the enumeration.

Example:

```go
type MyEnum string

const (
	EnumVal1 MyEnum = "EnumVal1"
	EnumVal2 MyEnum = "EnumVal2"
	EnumVal3 MyEnum = "EnumVal3"
)

func (e *MyEnum) UnmarshalString(input string) (interface{}, error) {
	switch input {
	case "EnumVal1":
		return EnumVal1, nil
	case "EnumVal2":
		return EnumVal2, nil
	case "EnumVal3":
		return EnumVal3, nil
	default:
		return nil, fmt.Errorf("invalid enum value %s", input)
	}
}
```

In this case, the enum type is a string, but that's not a requirement.

### `StringEnumValues`

Another way of dealing with enumerations is to treat them as strings, but with a layer of validation applied. You can implement the `StringEnumValues` interface to say what are the valid values for a given type.

```go
// StringEnumValues provides an interface for types that can return
// a list of valid string representations for their enumeration.
// This can be useful in scenarios like validation or auto-generation
// of documentation where a list of valid enum values is required.
//
// EnumValues should return a slice of strings representing the valid values
// for the enumeration.
type StringEnumValues interface {
	EnumValues() []string
}
```

These strings are used both for input validation and schema generation. The limitation is that the inputs and outputs that use this type need to be of a string type.

An example from the tests:

```go
type episode string

func (e episode) EnumValues() []string {
	return []string{
		"NEWHOPE",
		"EMPIRE",
		"JEDI",
	}
}
```

## Interfaces

Interfaces, in this case, are referring to how GraphQL uses the term "interface." The way that a type can implement an interface, as well as select the output filtering based on the type of object that is being returned.

The way this is modeled in this library is by using anonymous fields on a struct type to show an "is-a" relationship.

So, for instance:

```go
type Character struct {
	Id        string       `json:"id"`
	Name      string       `json:"name"`
	Friends   []*Character `json:"friends"`
	AppearsIn []episode    `json:"appearsIn"`
}

type Human struct {
	Character
	HeightMeters float64 `json:"HeightMeters"`
}
```

In this case, a `Human` is a subtype of `Character`. The schema generated from this is:

```graphql
type Human implements ICharacter {
	FriendsConnection(arg1: Int!): FriendsConnection
	HeightMeters: Float!
	appearsIn: [episode!]!
	friends: [Character]!
	id: String!
	name: String!
}

interface ICharacter {
	appearsIn: [episode!]!
	friends: [Character]!
	id: String!
	name: String!
}

type Character implements ICharacter {
	appearsIn: [episode!]!
	friends: [Character]!
	id: String!
	name: String!
}
```

### Interface and Concrete Type Generation

When a type is embedded in other structs (making it an "interface" in GraphQL terms), `go-quickgraph` by default generates both:
- An interface with an "I" prefix (e.g., `interface ICharacter`)
- A concrete type with the original name (e.g., `type Character implements ICharacter`)

This allows you to:
1. Query for the interface type when you want polymorphic behavior
2. Return the base type directly when needed
3. Use the base type in unions alongside its implementations

For example:
```go
// Base type that will become both interface and concrete type
type Vehicle struct {
    ID    string
    Model string
    Year  int
}

// Types that embed Vehicle
type Car struct {
    Vehicle
    Doors int
}

type Motorcycle struct {
    Vehicle
    Type string
}

// Union that can include the base Vehicle type
type SearchResultUnion struct {
    Vehicle    *Vehicle     // Can return a generic Vehicle
    Car        *Car         // Or a specific Car
    Motorcycle *Motorcycle  // Or a specific Motorcycle
}
```

Generated schema:
```graphql
interface IVehicle {
    ID: String!
    Model: String!
    Year: Int!
}

type Vehicle implements IVehicle {
    ID: String!
    Model: String!
    Year: Int!
}

type Car implements IVehicle {
    ID: String!
    Model: String!
    Year: Int!
    Doors: Int!
}

union SearchResult = Car | Motorcycle | Vehicle
```

### Opting Out of Concrete Type Generation

Sometimes you may want only an interface without the concrete type. You can opt out by implementing the `GraphTypeExtension` interface:

```go
type BaseComponent struct {
    ID   int
    Name string
}

// Opt out of concrete type generation
func (b BaseComponent) GraphTypeExtension() GraphTypeInfo {
    return GraphTypeInfo{
        Name:          "BaseComponent",
        InterfaceOnly: true,
    }
}

// Types that embed BaseComponent
type Button struct {
    BaseComponent
    Label string
}

type TextInput struct {
    BaseComponent
    Placeholder string
}
```

With `InterfaceOnly: true`, the schema generates only the interface:
```graphql
interface BaseComponent {
    ID: Int!
    Name: String!
}

type Button implements BaseComponent {
    ID: Int!
    Name: String!
    Label: String!
}

type TextInput implements BaseComponent {
    ID: Int!
    Name: String!
    Placeholder: String!
}
```

## Unions

Another aspect of GraphQL that doesn't cleanly map to Go is the concept of unions -- where a value can be one of several distinct types.

This is handled by one of two ways: implicit and explicit unions.

### Implicit Unions

Implicit unions are created by functions that return multiple pointers to results. Of course only one of those result pointers can be non-nil. For example:

```go
type resultA struct {
	OutStringA string
}
type resultB struct {
	OutStringB string
}
func Implicit(ctx context.Context, selector string) (*resultA, *resultB, error) {
	// implementation
}
```

This will generate a schema that looks like:

```graphql
type Query {
	Implicit(arg1: String!): ImplicitResultUnion!
}

union ImplicitResultUnion = resultA | resultB

type resultA {
	OutStringA: String!
}

type resultB {
	OutStringB: String!
}
```

If you need a custom-named enum, you can register the function like:

```go
g.RegisterFunction(ctx, FunctionDefinition{
	Name:            "CustomResultFunc",
	Function:        function,
	ReturnUnionName: "MyUnion",
})
```

In which case the name of the union is `MyUnion`.

### Explicit Unions

You can also name a type ending with the string `Union` and that type will be treated as a union. The members of that type must all be pointers. The result of the evaluation of the union must have a single non-nil value, and that is the implied type of the result.

#### Interface Expansion in Unions

When a union contains an interface type (a type that is embedded in other types), the union will automatically expand to include all concrete implementations of that interface:

```go
// Interface type (embedded by other types)
type Employee struct {
    ID   int
    Name string
}

// Concrete types that embed Employee
type Developer struct {
    Employee  // Makes Developer implement the Employee "interface"
    Languages []string
}

type Manager struct {
    Employee   // Makes Manager implement the Employee "interface"
    Department string
}

// Union that includes the interface
type SearchResultUnion struct {
    Employee *Employee  // Will expand to Developer, Manager, and Employee itself
    Product  *Product
    Widget   *Widget
}

// In the generated schema:
// union SearchResult = Developer | Employee | Manager | Product | Widget
```

#### Registering Types for Schema Generation

Types that aren't directly returned by any GraphQL function need to be explicitly registered to appear in the schema:

```go
graph := quickgraph.Graphy{}

// Register queries and mutations
graph.RegisterQuery(ctx, "search", SearchHandler)

// Explicitly register types that aren't directly returned
// This ensures Developer appears in the schema and unions
graph.RegisterTypes(ctx, Developer{})

// Now the SearchResult union will correctly include Developer
```

This is particularly important when:
- A type only appears as part of an interface (e.g., Developer is only returned as Employee)
- You want to ensure all possible union members are included in the schema
- Types are dynamically resolved at runtime but need to be in the schema

# Common Patterns

This section covers common patterns and best practices when building GraphQL APIs with go-quickgraph.

## Authentication and Authorization

### Pattern 1: Context-based Authentication

```go
// Middleware to add user to context
func authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        user, err := validateToken(token)
        if err != nil {
            http.Error(w, "Unauthorized", 401)
            return
        }
        ctx := context.WithValue(r.Context(), "user", user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Use authentication in your resolvers
func GetMyProfile(ctx context.Context) (*User, error) {
    user, ok := ctx.Value("user").(*User)
    if !ok {
        return nil, fmt.Errorf("unauthorized")
    }
    return user, nil
}

// Set up the server
http.Handle("/graphql", authMiddleware(g.HttpHandler()))
```

### Pattern 2: Field-level Authorization

```go
type User struct {
    ID    string
    Name  string
    Email string // Only visible to self or admin
}

// Use methods for sensitive fields
func (u *User) Email(ctx context.Context) (*string, error) {
    currentUser := ctx.Value("user").(*User)
    if currentUser.ID != u.ID && !currentUser.IsAdmin {
        return nil, nil // Return nil for unauthorized access
    }
    return &u.Email, nil
}
```

## Pagination

### Pattern 1: Offset-based Pagination

```go
type ProductsInput struct {
    Limit  int `json:"limit"`
    Offset int `json:"offset"`
}

type ProductList struct {
    Items      []Product `json:"items"`
    TotalCount int       `json:"totalCount"`
    HasMore    bool      `json:"hasMore"`
}

func GetProducts(ctx context.Context, input ProductsInput) (*ProductList, error) {
    if input.Limit <= 0 {
        input.Limit = 20 // Default
    }
    if input.Limit > 100 {
        input.Limit = 100 // Max
    }
    
    products, total := db.GetProducts(input.Offset, input.Limit)
    
    return &ProductList{
        Items:      products,
        TotalCount: total,
        HasMore:    input.Offset+len(products) < total,
    }, nil
}
```

### Pattern 2: Cursor-based Pagination (Relay-style)

```go
type Connection struct {
    Edges    []Edge   `json:"edges"`
    PageInfo PageInfo `json:"pageInfo"`
}

type Edge struct {
    Node   interface{} `json:"node"`
    Cursor string      `json:"cursor"`
}

type PageInfo struct {
    HasNextPage     bool   `json:"hasNextPage"`
    HasPreviousPage bool   `json:"hasPreviousPage"`
    StartCursor     string `json:"startCursor"`
    EndCursor       string `json:"endCursor"`
}

// Implement for your types
func GetUserConnection(ctx context.Context, first int, after string) (*Connection, error) {
    // Decode cursor, fetch data, build connection
    // See go-quickgraph-sample for full implementation
}
```

## Error Handling

### Pattern 1: Structured Errors

```go
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func CreateUser(ctx context.Context, input CreateUserInput) (*User, error) {
    if input.Email == "" {
        return nil, ValidationError{
            Field:   "email",
            Message: "Email is required",
        }
    }
    // Create user...
}
```

### Pattern 2: Multiple Errors

```go
type MultiError struct {
    Errors []error `json:"errors"`
}

func (m MultiError) Error() string {
    return fmt.Sprintf("%d validation errors", len(m.Errors))
}
```

## N+1 Query Prevention

### Pattern 1: DataLoader Pattern

```go
// Create a batch loader
type UserLoader struct {
    mu    sync.Mutex
    batch map[string]*User
    wait  sync.WaitGroup
}

func (l *UserLoader) Load(ctx context.Context, id string) (*User, error) {
    l.mu.Lock()
    if l.batch == nil {
        l.batch = make(map[string]*User)
        l.wait.Add(1)
        
        // Batch load after current tick
        go func() {
            time.Sleep(1 * time.Millisecond)
            l.loadBatch()
        }()
    }
    l.batch[id] = nil // Mark as requested
    l.mu.Unlock()
    
    l.wait.Wait() // Wait for batch to complete
    
    l.mu.Lock()
    user := l.batch[id]
    l.mu.Unlock()
    
    return user, nil
}

// Add loader to context in your HTTP handler
func (g *Graphy) HttpHandlerWithLoader() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := context.WithValue(r.Context(), "userLoader", &UserLoader{})
        g.HttpHandler().ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### Pattern 2: Eager Loading

```go
type Post struct {
    ID       string
    AuthorID string
    author   *User // Private field for caching
}

// Lazy load author
func (p *Post) Author(ctx context.Context) (*User, error) {
    if p.author != nil {
        return p.author, nil
    }
    
    // Use dataloader from context if available
    if loader, ok := ctx.Value("userLoader").(*UserLoader); ok {
        return loader.Load(ctx, p.AuthorID)
    }
    
    // Fallback to direct load
    return getUserByID(p.AuthorID)
}

// For queries that return multiple posts
func GetPosts(ctx context.Context) ([]Post, error) {
    posts := fetchPosts()
    
    // Optionally preload authors if requested
    if shouldPreloadAuthors(ctx) {
        authorIDs := collectAuthorIDs(posts)
        authors := batchLoadUsers(authorIDs)
        // Attach authors to posts...
    }
    
    return posts, nil
}
```

## File Uploads

```go
// GraphQL doesn't have a native file type, so use a custom scalar
type Upload struct {
    Filename string
    Size     int64
    Content  io.Reader
}

// In your HTTP handler, process multipart form data
func HandleGraphQLWithUploads(g *Graphy) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method == "POST" && strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
            // Parse multipart form
            // Extract operations and map variables
            // Process with g.ProcessRequest()
        } else {
            g.HttpHandler().ServeHTTP(w, r)
        }
    }
}
```

## Testing

### Pattern 1: Unit Testing Resolvers

```go
func TestGetUser(t *testing.T) {
    ctx := context.Background()
    g := quickgraph.Graphy{}
    
    // Mock your data layer
    mockDB := &MockDB{
        users: map[string]*User{
            "1": {ID: "1", Name: "Alice"},
        },
    }
    
    g.RegisterQuery(ctx, "user", func(ctx context.Context, id string) *User {
        return mockDB.GetUser(id)
    }, "id")
    
    query := `{ user(id: "1") { name } }`
    result, err := g.ProcessRequest(ctx, query, "")
    
    assert.NoError(t, err)
    assert.Contains(t, result, `"name":"Alice"`)
}
```

### Pattern 2: Integration Testing

```go
func TestGraphQLAPI(t *testing.T) {
    // Set up your full GraphQL server
    g := setupGraphy()
    server := httptest.NewServer(g.HttpHandler())
    defer server.Close()
    
    // Make HTTP requests
    query := `{"query": "{ users { name } }"}`
    resp, err := http.Post(server.URL+"/graphql", "application/json", strings.NewReader(query))
    
    // Assert response
    assert.Equal(t, 200, resp.StatusCode)
}
```

## Performance Optimization

### Pattern 1: Query Result Caching

```go
// The Graphy object already caches parsed queries
// You can add result caching for expensive operations

var resultCache = make(map[string]interface{})
var cacheMu sync.RWMutex

func GetExpensiveData(ctx context.Context, id string) (*Data, error) {
    cacheKey := fmt.Sprintf("data:%s", id)
    
    cacheMu.RLock()
    if cached, ok := resultCache[cacheKey]; ok {
        cacheMu.RUnlock()
        return cached.(*Data), nil
    }
    cacheMu.RUnlock()
    
    // Compute expensive data
    data := computeExpensiveData(id)
    
    cacheMu.Lock()
    resultCache[cacheKey] = data
    cacheMu.Unlock()
    
    return data, nil
}
```

### Pattern 2: Selective Field Resolution

```go
// Only compute expensive fields when requested
func (u *User) Statistics(ctx context.Context) (*UserStats, error) {
    // This method only runs if the client requests the statistics field
    return calculateUserStats(u.ID)
}
```

For more examples and patterns, see the [go-quickgraph-sample](https://github.com/gburgyan/go-quickgraph-sample) repository.

# Subscriptions

`go-quickgraph` supports GraphQL subscriptions, allowing clients to receive real-time updates when data changes. Subscriptions follow the same code-first approach as queries and mutations.

## Basic Subscription Example

```go
type Message struct {
    ID        string
    Text      string
    User      string
    Timestamp time.Time
}

// Register a subscription that returns a channel
g.RegisterSubscription(ctx, "messageAdded", func(ctx context.Context, roomID string) (<-chan Message, error) {
    ch := make(chan Message)
    
    // Set up your event source (message broker, database, etc.)
    go func() {
        defer close(ch)
        
        // Example: Subscribe to a message broker
        subscription := messageBroker.Subscribe(roomID)
        defer subscription.Close()
        
        for {
            select {
            case <-ctx.Done():
                return
            case msg := <-subscription.Messages():
                ch <- Message{
                    ID:        msg.ID,
                    Text:      msg.Text,
                    User:      msg.User,
                    Timestamp: msg.Timestamp,
                }
            }
        }
    }()
    
    return ch, nil
})
```

## Subscription Functions

Subscription functions must:
1. Return a receive-only channel (`<-chan T`) as the first return value
2. Optionally return an error as the second return value
3. Close the channel when the subscription ends
4. Respect context cancellation

The channel element type `T` follows the same rules as query/mutation return types.

Examples:
```go
// With error return (useful for setup validation)
func Subscribe(ctx context.Context, topic string) (<-chan Event, error) {
    if !isValidTopic(topic) {
        return nil, fmt.Errorf("invalid topic: %s", topic)
    }
    ch := make(chan Event)
    // ... setup subscription ...
    return ch, nil
}

// Without error return (simpler when setup can't fail)
func Subscribe(ctx context.Context, interval time.Duration) <-chan time.Time {
    ch := make(chan time.Time)
    go func() {
        defer close(ch)
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case t := <-ticker.C:
                ch <- t
            }
        }
    }()
    return ch
}
```

## WebSocket Support

Subscriptions require a WebSocket transport. `go-quickgraph` provides a pluggable WebSocket interface that works with any WebSocket library:

```go
// Example with gorilla/websocket
import "github.com/gorilla/websocket"

// Create adapter for your WebSocket library
type GorillaWebSocketAdapter struct {
    conn *websocket.Conn
}

func (a *GorillaWebSocketAdapter) ReadMessage() ([]byte, error) {
    _, data, err := a.conn.ReadMessage()
    return data, err
}

func (a *GorillaWebSocketAdapter) WriteMessage(data []byte) error {
    return a.conn.WriteMessage(websocket.TextMessage, data)
}

func (a *GorillaWebSocketAdapter) Close() error {
    return a.conn.Close()
}

// Create upgrader
type GorillaWebSocketUpgrader struct {
    upgrader websocket.Upgrader
}

func (u *GorillaWebSocketUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (quickgraph.SimpleWebSocketConn, error) {
    conn, err := u.upgrader.Upgrade(w, r, nil)
    if err != nil {
        return nil, err
    }
    return &GorillaWebSocketAdapter{conn: conn}, nil
}

// Use in your server
upgrader := &GorillaWebSocketUpgrader{
    upgrader: websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool { return true },
    },
}

handler := g.HttpHandlerWithWebSocket(upgrader)
http.Handle("/graphql", handler)
```

## Client Usage

Clients connect using the standard graphql-ws protocol:

```javascript
// JavaScript example with graphql-ws
import { createClient } from 'graphql-ws';

const client = createClient({
  url: 'ws://localhost:8080/graphql',
});

client.subscribe(
  {
    query: `
      subscription OnMessageAdded($room: String!) {
        messageAdded(roomID: $room) {
          id
          text
          user
          timestamp
        }
      }
    `,
    variables: { room: 'general' },
  },
  {
    next: (data) => console.log('New message:', data),
    error: (err) => console.error('Error:', err),
    complete: () => console.log('Subscription complete'),
  }
);
```

## Subscription Schema

Subscriptions appear in the GraphQL schema alongside queries and mutations:

```graphql
type Subscription {
    messageAdded(roomID: String!): Message!
    orderUpdated(orderID: String!): Order!
    userStatusChanged(userID: String!): UserStatus!
}
```

## Best Practices

1. **Always close channels**: Prevent goroutine leaks by closing channels when done
2. **Handle context cancellation**: Check `ctx.Done()` to stop processing
3. **Use buffered channels**: For bursty events, use buffered channels to prevent blocking
4. **Error handling**: Return errors during setup; for streaming errors, close the channel
5. **Resource cleanup**: Use `defer` for cleanup operations

## Limitations

- Only one subscription field per request (GraphQL spec requirement)
- Subscriptions cannot be mixed with queries or mutations
- Subscriptions require WebSocket transport (HTTP-only not supported)

# Schema Generation

Once a `graphy` is set up with all the query and mutation handlers, you can call:

```go
schema, err := g.SchemaDefinition(ctx)
```

This will create a GraphQL schema that represents the state of the `graphy` object. Explore the `schema_type_test.go` test file for more examples of generated schemata.

## Introspection

By calling `graph.EnableIntrospection(ctx)` you also enable the introspection queries. Internally this is handled by the schema generation subsystem. This also turns on implicit schema generation by the built-in HTTP handler.

## Limitations

* If there are multiple types with the same name, but from different packages, the results will not be valid.
 
# Caching

Caching is an optional feature of the graph processing. To enable it, simply set the `RequestCache` on the `Graphy` object. The cache is an implementation of the `GraphRequestCache` interface. If this is not set, the graphy functionality will not cache anything.

The cache is used to cache the result of parsing the request. This is a `RequestStub` as well as any errors that were present in parsing the errors. The request stub contains everything that was prepared to run the request except the variables that were passed in. This process involves a lot of reflection, so this is a comparatively expensive operation. By caching this processing, we gain a roughly 10x speedup.

We cache errors as well because a request that can't be fulfilled by the `Graphy` library will continue to be an error even if it submitted again -- there is no reason to reprocess the request to simply get back to the answer of error.

The internals of the `RequestStub` is only in-memory and not externally serializable.

## Example implementation

Using a simple cache library `github.com/patrickmn/go-cache`, here's a simple implementation:

```go
type SimpleGraphRequestCache struct {
	cache *cache.Cache
}

type simpleGraphRequestCacheEntry struct {
	request string
	stub    *RequestStub
	err     error
}

func (d *SimpleGraphRequestCache) SetRequestStub(ctx context.Context, request string, stub *RequestStub, err error) {
	setErr := d.cache.Add(request, &simpleGraphRequestCacheEntry{
		request: request,
		stub:    stub,
		err:     err,
	}, time.Hour)
	if setErr != nil {
		// Log this error, but don't return it.
		// Potentially disable the cache if this recurs continuously.
	}
}

func (d *SimpleGraphRequestCache) GetRequestStub(ctx context.Context, request string) (*RequestStub, error) {
	value, found := d.cache.Get(request)
	if !found {
		return nil, nil
	}
	entry, ok := value.(*simpleGraphRequestCacheEntry)
	if !ok {
		return nil, nil
	}
	return entry.stub, entry.err
}
```

Since each unique request, independent of the variables, can be cached, it's important to have a working eviction policy to prevent a denial of service attack from exhausting memory.

## Internal caching

Internally `Graphy` will cache much of the results of reflection operations. These relate to the types that are used for input and output. Since these have a one-to-one relationship to the internal types of the running system, they are cached by `Graphy` for the lifetime of the object; it can't grow out of bounds and cannot be subject to a denial of service attack. 

# Query Limits and DoS Protection

`go-quickgraph` provides optional query limits to protect against denial-of-service attacks and resource exhaustion. These limits are opt-in and can be configured on the `Graphy` instance.

## Configuring Query Limits

```go
graph := quickgraph.Graphy{
    QueryLimits: &quickgraph.QueryLimits{
        MaxDepth:               10,    // Maximum query nesting depth
        MaxFields:              50,    // Maximum fields per level
        MaxAliases:             20,    // Maximum number of aliases
        MaxArraySize:           1000,  // Maximum array elements returned
        MaxConcurrentResolvers: 100,   // Maximum parallel goroutines
    },
}
```

All limits are optional - zero or unset values mean unlimited.

## Available Limits

### MaxDepth
Prevents deeply nested queries that could cause excessive processing or stack overflow:
```graphql
# This query would be rejected if MaxDepth is set to 3
{
  user {
    posts {
      comments {
        author {  # Depth 4 - would exceed limit
          name
        }
      }
    }
  }
}
```

### MaxFields
Limits the number of fields that can be requested at any single level:
```graphql
# This would be rejected if MaxFields is set to 3
{
  user {
    id        # Field 1
    name      # Field 2
    email     # Field 3
    posts     # Field 4 - would exceed limit
  }
}
```

### MaxAliases
Prevents alias amplification attacks:
```graphql
# This would be rejected if MaxAliases is set to 2
{
  a1: expensive { ... }  # Alias 1
  a2: expensive { ... }  # Alias 2
  a3: expensive { ... }  # Alias 3 - would exceed limit
}
```

### MaxArraySize
Truncates large arrays to prevent memory exhaustion:
```go
// If a resolver returns 10,000 items but MaxArraySize is 100,
// only the first 100 items will be processed and returned
```

### MaxConcurrentResolvers
Limits the number of goroutines spawned for parallel query execution:
```go
// With MaxConcurrentResolvers set to 10, at most 10 resolver
// functions will execute in parallel, even if the query has
// 100 aliased fields
```

## Error Handling

When a limit is exceeded, the query is rejected with a descriptive error:
```json
{
  "errors": [{
    "message": "query depth 5 exceeds maximum allowed depth of 3"
  }]
}
```

## Performance Considerations

- Depth and field limits are checked during query parsing (early validation)
- Array size limits are applied during execution (late validation)
- Concurrent resolver limits may cause some queries to execute sequentially instead of in parallel

## Example: Production Configuration

```go
func main() {
    ctx := context.Background()
    
    graph := quickgraph.Graphy{
        QueryLimits: &quickgraph.QueryLimits{
            MaxDepth:               15,    // Reasonable for most schemas
            MaxFields:              100,   // Prevent overly broad queries
            MaxAliases:             30,    // Prevent alias abuse
            MaxArraySize:           5000,  // Limit memory usage
            MaxConcurrentResolvers: 50,    // Prevent goroutine explosion
        },
    }
    
    graph.RegisterQuery(ctx, "users", GetUsers)
    graph.EnableIntrospection(ctx)
    
    http.Handle("/graphql", graph.HttpHandler())
    http.ListenAndServe(":8080", nil)
}
```

# Dealing with unknown commands

A frequent requirement is to implement a strangler pattern to start taking requests for things that can be processed, but to forward requests that can't be processed to another service. This is enabled by the processing pipeline by returning a `UnknownCommandError`. Since the processing of the request can be cached, this can be a fail-fast scenario so that the request could be forwarded to another service for processing. 

# Benchmarks

Given this relatively complex query:

```graphql
mutation CreateReviewForEpisode($ep: Episode!, $review: ReviewInput!) {
  createReview(episode: $ep, review: $review) {
    stars
    commentary
  }
}
```

with this variable JSON:

```json
{
  "ep": "JEDI",
  "review": {
    "stars": 5,
    "commentary": "This is a great movie!"
  }
}
```

with caching enabled, the framework overhead is less than 4.8µs on an Apple M1 Pro processor. This includes parsing the variable JSON, calling the function for `CreateReviewForEpisode`, and processing the output. The vast majority of the overhead, roughly 75% of the time, isn't the library itself, but rather the unmarshalling of the variable JSON as well as marshaling the result to be returned.

See the `benchmark_test.go` benchmarks for more tests and to evaluate this on your hardware.

While potentially caching the variable JSON would be possible, the decision was made that it's likely not worthwhile as the variables are what are most likely to change between requests negating any benefits of caching.

# General Limitations

## Interfaces

While interfaces are handled appropriately when processing responses, due to limitations in the Go reflection system there is no way to find all types that implement that interface. Because of this, when looking at the generated schemata or using the introspection system, there may be implementing types that are not known. 

## Validation of Input

Queries ignore the type specifiers on the input. The types are always inferred from the actual function inputs.