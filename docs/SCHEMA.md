# Schema Generation

**Generate GraphQL schemas from your Go code** - Export SDL schemas for documentation, client code generation, and CI/CD integration.

## Overview

go-quickgraph automatically generates GraphQL schemas from your Go code structure. This powerful feature enables:

- **Schema Export** - Generate SDL (Schema Definition Language) files from your API
- **Client Code Generation** - Use exported schemas with tools like GraphQL Code Generator
- **Schema Versioning** - Track schema changes in your version control system
- **API Documentation** - Generate documentation from your GraphQL schema
- **CI/CD Integration** - Validate schema changes and ensure API compatibility

## How Schema Generation Works

The schema generation process analyzes your registered queries, mutations, subscriptions, and types to produce a complete GraphQL schema:

1. **Type Discovery** - Examines all registered functions and their parameters/return types
2. **Type Mapping** - Converts Go types to appropriate GraphQL types
3. **Relationship Resolution** - Handles interfaces, unions, and type implementations
4. **Schema Assembly** - Combines all elements into a valid GraphQL SDL

## Basic Usage

### Generating a Schema

Use the `SchemaDefinition()` method to generate your schema as a string. The schema includes any descriptions you've added to your types, fields, and functions:

```go
package main

import (
    "context"
    "fmt"
    "github.com/gburgyan/go-quickgraph"
)

type User struct {
    ID    string `graphy:"id"`
    Name  string `graphy:"name"`
    Email string `graphy:"email"`
}

func main() {
    ctx := context.Background()
    g := quickgraph.Graphy{}
    
    // Register your API
    g.RegisterQuery(ctx, "user", func(ctx context.Context, id string) *User {
        return &User{ID: id, Name: "Alice", Email: "alice@example.com"}
    }, "id")
    
    // Generate the schema
    schema := g.SchemaDefinition(ctx)
    fmt.Println(schema)
}
```

Output:
```graphql
type Query {
    user(id: String!): User
}

type User {
    id: String!
    name: String!
    email: String!
}
```

### Enabling Introspection

For GraphQL Playground and other tools to explore your schema:

```go
// Enable introspection queries
g.EnableIntrospection(ctx)

// Now clients can query __schema and __type
```

### Exporting Schema to a File

```go
package main

import (
    "context"
    "os"
    "github.com/gburgyan/go-quickgraph"
)

func exportSchema(g *quickgraph.Graphy) error {
    ctx := context.Background()
    schema := g.SchemaDefinition(ctx)
    
    return os.WriteFile("schema.graphql", []byte(schema), 0644)
}
```

## Schema Generation Features

### Type Mapping

go-quickgraph maps Go types to GraphQL types automatically:

| Go Type | GraphQL Type |
|---------|--------------|
| `string` | `String` |
| `int`, `int32`, `int64` | `Int` |
| `float32`, `float64` | `Float` |
| `bool` | `Boolean` |
| `struct` | `Object Type` |
| `interface` | `Interface` |
| `[]T` | `[T]` |
| `*T` | `T` (nullable) |
| `T` | `T!` (non-null) |

### Interfaces and Implementations

When a type implements an interface, go-quickgraph generates both interface and concrete types:

```go
type Node interface {
    GetID() string
}

type User struct {
    ID   string
    Name string
}

func (u User) GetID() string { return u.ID }

// Generates:
// interface INode {
//     id: String!
// }
// 
// type User implements INode {
//     id: String!
//     name: String!
// }
```

### Input vs Output Types

go-quickgraph automatically handles type name collisions between input and output types:

```go
type UserFilter struct {
    Name  string
    Email string
}

g.RegisterQuery(ctx, "users", func(filter UserFilter) []User {
    // ... implementation
}, "filter")

// Generates:
// type Query {
//     users(filter: UserFilterInput!): [User!]!
// }
// 
// input UserFilterInput {
//     name: String!
//     email: String!
// }
```

### Enums

Implement the `StringEnumValues` interface for enum support:

```go
type Status string

func (s Status) EnumValues() []quickgraph.EnumValueDefinition {
    return []quickgraph.EnumValueDefinition{
        {Name: "ACTIVE", Description: "User is active"},
        {Name: "INACTIVE", Description: "User is inactive"},
        {Name: "SUSPENDED", Description: "User is suspended", 
         IsDeprecated: true, DeprecationReason: "Use INACTIVE instead"},
    }
}
```

## CI/CD Integration

### Standalone Schema Generator

Create a server that can both serve GraphQL and generate schemas:

```go
// cmd/server/main.go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "github.com/gburgyan/go-quickgraph"
    "github.com/yourorg/yourapp/api"
)

func main() {
    var (
        generateSchema = flag.Bool("generate-schema", false, "Generate GraphQL schema and exit")
        output        = flag.String("output", "-", "Schema output file (- for stdout)")
        port          = flag.String("port", "8080", "Server port")
    )
    flag.Parse()
    
    // Initialize GraphQL server
    g := quickgraph.Graphy{}
    ctx := context.Background()
    
    // Register all your API endpoints
    api.RegisterQueries(&g, ctx)
    api.RegisterMutations(&g, ctx)
    api.RegisterSubscriptions(&g, ctx)
    api.RegisterTypes(&g, ctx)
    
    // Schema generation mode
    if *generateSchema {
        schema := g.SchemaDefinition(ctx)
        
        if *output == "-" {
            fmt.Print(schema)
        } else {
            if err := os.WriteFile(*output, []byte(schema), 0644); err != nil {
                log.Fatalf("Failed to write schema: %v", err)
            }
            fmt.Printf("Schema written to %s\n", *output)
        }
        os.Exit(0)
    }
    
    // Normal server mode
    g.EnableIntrospection(ctx)
    http.Handle("/graphql", g.HttpHandler())
    
    log.Printf("GraphQL server running at http://localhost:%s/graphql", *port)
    if err := http.ListenAndServe(":"+*port, nil); err != nil {
        log.Fatal(err)
    }
}
```

Usage:
```bash
# Run as a server
./server

# Generate schema to stdout
./server -generate-schema

# Generate schema to file
./server -generate-schema -output=schema.graphql

# Use in CI/CD
go run ./cmd/server -generate-schema > schema.graphql
```

### GitHub Actions Example

```yaml
name: Generate GraphQL Schema

on:
  push:
    branches: [main]
  pull_request:

jobs:
  schema:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Generate Schema
        run: |
          go run ./cmd/server -generate-schema -output=schema.graphql
      
      - name: Check Schema Changes
        run: |
          git diff --exit-code schema.graphql || \
          (echo "Schema has changed! Please commit schema.graphql" && exit 1)
      
      - name: Upload Schema Artifact
        uses: actions/upload-artifact@v3
        with:
          name: graphql-schema
          path: schema.graphql
```

### Docker Integration

```dockerfile
# Multi-stage build for schema generation
FROM golang:1.21 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server ./cmd/server
RUN ./server -generate-schema -output=/schema.graphql

# Runtime image
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
COPY --from=builder /schema.graphql .
CMD ["./server"]
```

## Best Practices

### 1. Generate at Build Time

Generate schemas during your build process rather than at runtime:

```makefile
# Makefile
.PHONY: schema
schema:
	@go run ./cmd/server -generate-schema -output=schema.graphql
	@echo "Schema generated: schema.graphql"

build: schema
	go build -o server ./cmd/server

.PHONY: run
run: build
	./server

.PHONY: check-schema
check-schema:
	@go run ./cmd/server -generate-schema -output=.schema.tmp
	@diff schema.graphql .schema.tmp || (echo "Schema out of date! Run 'make schema'" && rm .schema.tmp && exit 1)
	@rm .schema.tmp
	@echo "Schema is up to date"
```

### 2. Version Your Schema

Track schema changes in version control:

```bash
# Add to your .gitignore if generating dynamically
# schema.graphql

# OR commit it to track changes
git add schema.graphql
git commit -m "chore: update GraphQL schema"
```

### 3. Schema Validation

Use tools to validate schema changes:

```bash
# Install graphql-inspector
npm install -g @graphql-inspector/cli

# Check for breaking changes
graphql-inspector diff schema-old.graphql schema-new.graphql
```

### 4. Client Code Generation

Use your schema with client generators:

```json
// codegen.yml for GraphQL Code Generator
overwrite: true
schema: "./schema.graphql"
generates:
  ./src/generated/graphql.ts:
    plugins:
      - "typescript"
      - "typescript-operations"
```

### 5. Documentation and Descriptions

Add descriptions to your GraphQL schema:

#### Type Descriptions

Use `GraphTypeExtension` to add descriptions to types:

```go
type User struct {
    ID    string `graphy:"id"`
    Name  string `graphy:"name"`
    Email string `graphy:"email"`
}

func (User) GraphTypeExtension() quickgraph.GraphTypeInfo {
    return quickgraph.GraphTypeInfo{
        Description: "A user in the system",
    }
}
```

#### Field Descriptions

Use the `graphy` tag to add field descriptions:

```go
type User struct {
    ID    string `graphy:"id" graphy:"description=Unique user identifier"`
    Name  string `graphy:"name" graphy:"description=User's display name"`
    Email string `graphy:"email" graphy:"description=Email address,deprecated=Use primaryEmail instead"`
}
```

#### Function Descriptions

Add descriptions when registering functions:

```go
desc := "Retrieves a user by their ID"
g.RegisterFunction(ctx, quickgraph.FunctionDefinition{
    Name:        "getUser",
    Function:    getUserFunc,
    Description: &desc,
    Mode:        quickgraph.ModeQuery,
})
```

#### Enum Descriptions

Implement `StringEnumValues` with descriptions:

```go
type Status string

func (s Status) EnumValues() []quickgraph.EnumValue {
    return []quickgraph.EnumValue{
        {Name: "ACTIVE", Description: "User is currently active"},
        {Name: "INACTIVE", Description: "User is not active"},
        {Name: "SUSPENDED", Description: "User is suspended", 
         IsDeprecated: true, DeprecationReason: "Use INACTIVE"},
    }
}
```

Generated schema with descriptions:

```graphql
"""
A user in the system
"""
type User {
    """Unique user identifier"""
    id: String!
    
    """User's display name"""
    name: String!
    
    """Email address"""
    email: String! @deprecated(reason: "Use primaryEmail instead")
}

enum Status {
    """User is currently active"""
    ACTIVE
    
    """User is not active"""
    INACTIVE
    
    """User is suspended"""
    SUSPENDED @deprecated(reason: "Use INACTIVE")
}
```

## Advanced Topics

### Custom Type Names

Control GraphQL type names using `GraphTypeExtension`:

```go
type Product struct {
    SKU   string
    Price float64
}

func (Product) GraphTypeExtension() quickgraph.GraphTypeInfo {
    return quickgraph.GraphTypeInfo{
        Name:        "ProductItem",
        Description: "A product in our catalog",
    }
}
```


## Troubleshooting

### Common Issues

1. **Missing Types**: Ensure all types are reachable from registered functions
2. **Name Conflicts**: Input types automatically get "Input" suffix when needed
3. **Interface Issues**: Check that implementations properly satisfy interfaces

### Debugging Schema Generation

```go
// Enable verbose logging during development
g.EnableTiming = true

// Generate schema with timing information
schema := g.SchemaDefinition(ctx)
```

## Example: Complete Server with Schema Generation

Here's a production-ready example that combines serving GraphQL with schema generation capabilities:

```go
// cmd/server/main.go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "github.com/gburgyan/go-quickgraph"
    "github.com/yourorg/yourapp/graph"
)

type Config struct {
    GenerateSchema bool
    SchemaOutput   string
    SchemaFormat   string
    Port           string
    Production     bool
}

func main() {
    cfg := parseFlags()
    
    // Initialize GraphQL
    g := initializeGraphQL(cfg.Production)
    ctx := context.Background()
    
    // Schema generation mode
    if cfg.GenerateSchema {
        if err := generateSchema(g, ctx, cfg); err != nil {
            log.Fatal(err)
        }
        return
    }
    
    // Server mode
    startServer(g, ctx, cfg)
}

func parseFlags() Config {
    cfg := Config{}
    flag.BoolVar(&cfg.GenerateSchema, "generate-schema", false, "Generate schema and exit")
    flag.StringVar(&cfg.SchemaOutput, "output", "-", "Schema output file (- for stdout)")
    flag.StringVar(&cfg.SchemaFormat, "format", "sdl", "Schema format: sdl, json")
    flag.StringVar(&cfg.Port, "port", "8080", "Server port")
    flag.BoolVar(&cfg.Production, "production", false, "Run in production mode")
    flag.Parse()
    return cfg
}

func initializeGraphQL(production bool) *quickgraph.Graphy {
    g := &quickgraph.Graphy{
        ProductionMode: production,
        QueryLimits: &quickgraph.QueryLimits{
            MaxDepth:      10,
            MaxComplexity: 1000,
        },
        MemoryLimits: &quickgraph.MemoryLimits{
            MaxRequestBodySize: 1024 * 1024, // 1MB
        },
    }
    
    // Register API from your graph package
    graph.RegisterAPI(g)
    
    return g
}

func generateSchema(g *quickgraph.Graphy, ctx context.Context, cfg Config) error {
    var result string
    
    switch cfg.SchemaFormat {
    case "sdl":
        result = g.SchemaDefinition(ctx)
    case "json":
        g.EnableIntrospection(ctx)
        // Run introspection query
        query := `{
            __schema {
                types { name kind description }
                queryType { name }
                mutationType { name }
                subscriptionType { name }
            }
        }`
        resp, err := g.ProcessRequest(ctx, query, nil)
        if err != nil {
            return fmt.Errorf("introspection failed: %w", err)
        }
        data, err := json.MarshalIndent(resp, "", "  ")
        if err != nil {
            return fmt.Errorf("json marshal failed: %w", err)
        }
        result = string(data)
    default:
        return fmt.Errorf("unknown format: %s", cfg.SchemaFormat)
    }
    
    // Output
    if cfg.SchemaOutput == "-" {
        fmt.Print(result)
    } else {
        if err := os.WriteFile(cfg.SchemaOutput, []byte(result), 0644); err != nil {
            return fmt.Errorf("write failed: %w", err)
        }
        log.Printf("Schema written to %s\n", cfg.SchemaOutput)
    }
    
    return nil
}

func startServer(g *quickgraph.Graphy, ctx context.Context, cfg Config) {
    g.EnableIntrospection(ctx)
    
    // GraphQL endpoint
    http.Handle("/graphql", g.HttpHandler())
    
    // Health check
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })
    
    // Schema endpoint (optional)
    http.HandleFunc("/schema", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/plain")
        schema := g.SchemaDefinition(ctx)
        w.Write([]byte(schema))
    })
    
    log.Printf("GraphQL server running at http://localhost:%s/graphql", cfg.Port)
    if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
        log.Fatal(err)
    }
}
```

Usage in development:
```bash
# Run server
go run ./cmd/server

# Generate SDL schema
go run ./cmd/server -generate-schema

# Generate JSON introspection
go run ./cmd/server -generate-schema -format=json -output=schema.json

# Production mode with schema generation
go run ./cmd/server -production -generate-schema -output=prod-schema.graphql
```

## Next Steps

- Learn about [Custom Scalars](CUSTOM_SCALARS.md) for specialized types
- Explore [Performance](PERFORMANCE.md) optimization techniques
- Understand [Security](SECURITY_API.md) considerations for production use