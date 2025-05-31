# Custom Scalar Types in go-quickgraph

This document describes how to use custom scalar types in the go-quickgraph library.

## Overview

Custom scalars allow you to map Go types to GraphQL scalar types with custom serialization and parsing logic. This is useful for types like `time.Time`, custom ID types, or any complex data structures that should be represented as scalars in GraphQL.

## Basic Usage

### Registering a Custom Scalar

Use `RegisterScalar()` to register a custom scalar type:

```go
import (
    "context"
    "reflect"
    "time"
    "github.com/gburgyan/go-quickgraph"
)

// Define your custom type
type UserID string

func main() {
    ctx := context.Background()
    graphy := &quickgraph.Graphy{}

    // Register the scalar
    err := graphy.RegisterScalar(ctx, quickgraph.ScalarDefinition{
        Name:   "UserID",
        GoType: reflect.TypeOf(UserID("")),
        Description: "Unique identifier for users",
        Serialize: func(value interface{}) (interface{}, error) {
            if uid, ok := value.(UserID); ok {
                return string(uid), nil
            }
            return nil, fmt.Errorf("expected UserID, got %T", value)
        },
        ParseValue: func(value interface{}) (interface{}, error) {
            if str, ok := value.(string); ok {
                return UserID(str), nil
            }
            return nil, fmt.Errorf("expected string, got %T", value)
        },
    })
    if err != nil {
        panic(err)
    }
}
```

### ScalarDefinition Fields

- **Name**: The GraphQL scalar type name that will appear in the schema
- **GoType**: The Go type that this scalar represents
- **Description**: Optional description for the scalar type
- **Serialize**: Function to convert Go value to JSON-serializable value for output
- **ParseValue**: Function to convert JSON value from variables to Go value
- **ParseLiteral**: Optional function to convert GraphQL literal values. If not provided, `ParseValue` is used as fallback

## Built-in Scalars

The library provides built-in scalar registrations for common types:

### DateTime Scalar

```go
err := graphy.RegisterDateTimeScalar(ctx)
```

This registers a `DateTime` scalar for `time.Time` that uses RFC3339 format:

```graphql
scalar DateTime # RFC3339 formatted date-time string

type Event {
    startTime: DateTime!
    endTime: DateTime
}
```

### JSON Scalar

```go
err := graphy.RegisterJSONScalar(ctx)
```

This registers a `JSON` scalar for `map[string]interface{}` that passes through arbitrary JSON data:

```graphql
scalar JSON # Arbitrary JSON data

type Product {
    metadata: JSON
}
```

## Advanced Examples

### Custom ID Type with Validation

```go
type ProductID string

func (id ProductID) Validate() error {
    if len(id) == 0 {
        return fmt.Errorf("product ID cannot be empty")
    }
    if !strings.HasPrefix(string(id), "prod_") {
        return fmt.Errorf("product ID must start with 'prod_'")
    }
    return nil
}

err := graphy.RegisterScalar(ctx, quickgraph.ScalarDefinition{
    Name:   "ProductID",
    GoType: reflect.TypeOf(ProductID("")),
    Serialize: func(value interface{}) (interface{}, error) {
        if pid, ok := value.(ProductID); ok {
            return string(pid), nil
        }
        return nil, fmt.Errorf("expected ProductID, got %T", value)
    },
    ParseValue: func(value interface{}) (interface{}, error) {
        if str, ok := value.(string); ok {
            pid := ProductID(str)
            if err := pid.Validate(); err != nil {
                return nil, err
            }
            return pid, nil
        }
        return nil, fmt.Errorf("expected string, got %T", value)
    },
})
```

### Currency Type

```go
type Money struct {
    Amount   int64  // in cents
    Currency string
}

err := graphy.RegisterScalar(ctx, quickgraph.ScalarDefinition{
    Name:   "Money",
    GoType: reflect.TypeOf(Money{}),
    Description: "Monetary amount with currency",
    Serialize: func(value interface{}) (interface{}, error) {
        if money, ok := value.(Money); ok {
            return fmt.Sprintf("%.2f %s", float64(money.Amount)/100, money.Currency), nil
        }
        return nil, fmt.Errorf("expected Money, got %T", value)
    },
    ParseValue: func(value interface{}) (interface{}, error) {
        if str, ok := value.(string); ok {
            // Parse "12.34 USD" format
            parts := strings.Split(str, " ")
            if len(parts) != 2 {
                return nil, fmt.Errorf("invalid money format, expected 'amount currency'")
            }
            amount, err := strconv.ParseFloat(parts[0], 64)
            if err != nil {
                return nil, fmt.Errorf("invalid amount: %v", err)
            }
            return Money{
                Amount:   int64(amount * 100),
                Currency: parts[1],
            }, nil
        }
        return nil, fmt.Errorf("expected string, got %T", value)
    },
})
```

### URL Type

```go
import "net/url"

err := graphy.RegisterScalar(ctx, quickgraph.ScalarDefinition{
    Name:   "URL",
    GoType: reflect.TypeOf(url.URL{}),
    Description: "Valid URL string",
    Serialize: func(value interface{}) (interface{}, error) {
        if u, ok := value.(url.URL); ok {
            return u.String(), nil
        }
        if u, ok := value.(*url.URL); ok && u != nil {
            return u.String(), nil
        }
        return nil, fmt.Errorf("expected url.URL, got %T", value)
    },
    ParseValue: func(value interface{}) (interface{}, error) {
        if str, ok := value.(string); ok {
            u, err := url.Parse(str)
            if err != nil {
                return nil, fmt.Errorf("invalid URL: %v", err)
            }
            return *u, nil
        }
        return nil, fmt.Errorf("expected string, got %T", value)
    },
})
```

## Using Scalars in Functions

Once registered, scalars can be used in function parameters and return values:

```go
// Register functions that use custom scalars
graphy.RegisterQuery(ctx, "getUser", func(id UserID) *User {
    // Function implementation
    return findUserByID(id)
}, "id")

graphy.RegisterMutation(ctx, "createProduct", func(input ProductInput) ProductID {
    // Function implementation
    return ProductID("prod_123")
}, "input")

// Custom scalars appear in the generated schema
// type Query {
//     getUser(id: UserID!): User
// }
// 
// type Mutation {
//     createProduct(input: ProductInput!): ProductID!
// }
//
// scalar UserID
// scalar ProductID
```

## Schema Generation

Custom scalars automatically appear in the generated GraphQL schema:

```graphql
scalar DateTime # RFC3339 formatted date-time string
scalar UserID
scalar ProductID
scalar Money # Monetary amount with currency
```

The scalars are sorted alphabetically in the schema output.

## Error Handling

Scalar parsing and serialization functions should return descriptive errors:

```go
ParseValue: func(value interface{}) (interface{}, error) {
    if str, ok := value.(string); ok {
        if len(str) == 0 {
            return nil, fmt.Errorf("UserID cannot be empty")
        }
        return UserID(str), nil
    }
    return nil, fmt.Errorf("expected string for UserID, got %T", value)
},
```

These errors will be included in GraphQL error responses with proper location information.

## Best Practices

1. **Validation**: Include validation logic in `ParseValue` to ensure data integrity
2. **Type Safety**: Always check types in serialize/parse functions
3. **Error Messages**: Provide clear, descriptive error messages
4. **Naming**: Use descriptive GraphQL scalar names that follow GraphQL conventions
5. **Documentation**: Include descriptions for custom scalars
6. **Registration Order**: Register scalars before registering functions that use them

## Limitations

- Custom scalars work best with simple Go types (strings, numbers, simple structs)
- Complex struct types may have issues with field selection in GraphQL queries
- Circular references in scalar types are not supported
- Map types are not directly supported (use JSON scalar instead)

## Management Functions

### Get Scalar Information

```go
// Get scalar by name
scalar, exists := graphy.GetScalarByName("UserID")
if exists {
    fmt.Printf("Scalar %s maps to Go type %v\n", scalar.Name, scalar.GoType)
}

// Get scalar by Go type
scalar, exists = graphy.GetScalarByType(reflect.TypeOf(UserID("")))

// Get all registered scalars
scalars := graphy.GetRegisteredScalars()
for name, scalar := range scalars {
    fmt.Printf("Scalar %s: %s\n", name, scalar.Description)
}
```

### Error Checking

The `RegisterScalar` function validates scalar definitions:

- Scalar name must not be empty
- Scalar name must follow GraphQL naming conventions
- GoType must not be nil
- Serialize and ParseValue functions must not be nil
- No name or type conflicts with existing scalars

```go
err := graphy.RegisterScalar(ctx, definition)
if err != nil {
    // Handle registration error
    fmt.Printf("Failed to register scalar: %v\n", err)
}
```