![Build status](https://github.com/gburgyan/go-quickgraph/actions/workflows/go.yml/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/gburgyan/go-quickgraph)](https://goreportcard.com/report/github.com/gburgyan/go-quickgraph) [![PkgGoDev](https://pkg.go.dev/badge/github.com/gburgyan/go-quickgraph)](https://pkg.go.dev/github.com/gburgyan/go-quickgraph)

# go-quickgraph

**A code-first GraphQL library for Go** - Write regular Go functions and structs, get a full GraphQL API.

## Quick Start

**📖 [Read the Quickstart Guide](docs/QUICKSTART.md)** - Learn go-quickgraph in 5 minutes with practical examples.

```bash
go get github.com/gburgyan/go-quickgraph
```

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "github.com/gburgyan/go-quickgraph"
)

type User struct {
    Name  string `graphy:"name"`
    Email string `graphy:"email"`
}

func main() {
    ctx := context.Background()
    g := quickgraph.Graphy{
        // Optional: Add security configuration
        MemoryLimits: &quickgraph.MemoryLimits{
            MaxRequestBodySize:            1024 * 1024, // 1MB
            MaxVariableSize:               64 * 1024,   // 64KB
            MaxSubscriptionsPerConnection: 10,          // Per-connection limit
        },
        QueryLimits: &quickgraph.QueryLimits{
            MaxDepth:      10,   // Query depth limit
            MaxComplexity: 1000, // Query complexity limit
        },
        // Optional: Configure CORS for web clients
        CORSSettings: quickgraph.DefaultCORSSettings(),
    }
    
    // Register a function - it becomes a GraphQL query automatically!
    g.RegisterQuery(ctx, "user", func(ctx context.Context, id int) *User {
        return &User{Name: "Alice", Email: "alice@example.com"}
    }, "id")
    
    // Enable GraphQL Playground and serve over HTTP
    g.EnableIntrospection(ctx)
    http.Handle("/graphql", g.HttpHandler())
    
    fmt.Println("🚀 GraphQL server running at http://localhost:8080/graphql")
    http.ListenAndServe(":8080", nil)
}
```

That's it! Visit http://localhost:8080/graphql to explore your API with GraphQL Playground.

Try this query:
```graphql
{
  user(id: 1) {
    name
    email
  }
}
```

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

## Key Features

- **🔥 Code-First**: Generate schemas from Go code, not the other way around
- **⚡ Fast**: Aggressive caching and optimized reflection usage
- **🛡️ Secure**: Built-in DoS protection, memory limits, and WebSocket authentication
- **🌐 Web-Ready**: Configurable CORS support for GraphQL playgrounds and web clients
- **🔄 Real-time**: WebSocket subscriptions with channel-based streaming
- **🎯 Type-Safe**: Full Go type checking with automatic GraphQL type generation
- **🚀 Production Ready**: Thread-safe, caching, error handling, and observability

## Documentation

### Getting Started
- **[Quickstart Guide](docs/QUICKSTART.md)** - Learn go-quickgraph in 5 minutes (start here!)
- [Installation & Setup](docs/GETTING_STARTED.md) - Your first GraphQL API in 5 minutes
- [Core Concepts](docs/CORE_CONCEPTS.md) - Understanding the code-first approach
- [Basic Operations](docs/BASIC_OPERATIONS.md) - Queries, mutations, and simple types

### Building APIs  
- [Type System Guide](docs/TYPE_SYSTEM.md) - Structs, interfaces, unions, and enums
- [Function Patterns](docs/FUNCTION_PATTERNS.md) - Parameter handling and return types
- [Custom Scalars](docs/CUSTOM_SCALARS.md) - DateTime, Money, and custom types
- [Real-time Subscriptions](docs/SUBSCRIPTIONS.md) - WebSocket streaming with channels

### Advanced Usage
- [Security Guide](docs/SECURITY_API.md) - Memory limits, authentication, and DoS protection
- [Authentication & Authorization](docs/AUTH_PATTERNS.md) - Securing your GraphQL API
- [Performance & Caching](docs/PERFORMANCE.md) - Optimization and DoS protection  
- [Schema Generation](docs/SCHEMA.md) - Introspection and schema customization
- [Error Handling](docs/ERROR_HANDLING.md) - Graceful error management

### Examples & Recipes
- [Common Patterns](docs/COMMON_PATTERNS.md) - Pagination, DataLoader, file uploads
- [Migration Guide](docs/MIGRATION.md) - Moving from other GraphQL libraries
- [Troubleshooting](docs/TROUBLESHOOTING.md) - Common issues and solutions

## Examples

Check out the comprehensive [sample application](https://github.com/gburgyan/go-quickgraph-sample) that demonstrates:

- **Full CRUD Operations** - Products, employees, widgets
- **Advanced Type System** - Interfaces, unions, custom scalars
- **Real-time Features** - WebSocket subscriptions  
- **Security** - Authentication, authorization, query limits
- **Performance** - Caching, batching, optimization

Run the sample:
```bash
git clone https://github.com/gburgyan/go-quickgraph-sample
cd go-quickgraph-sample
go run ./cmd/server
# Visit http://localhost:8080/graphql
```

## Requirements

- Go 1.20 or later
- No external dependencies for core functionality
- Optional: WebSocket library for subscriptions (gorilla/websocket recommended)

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

- 📖 [Documentation](docs/) - Comprehensive guides and API reference
- 💡 [Sample App](https://github.com/gburgyan/go-quickgraph-sample) - Working examples
- 🐛 [Issues](https://github.com/gburgyan/go-quickgraph/issues) - Bug reports and feature requests
- 💬 [Discussions](https://github.com/gburgyan/go-quickgraph/discussions) - Questions and community