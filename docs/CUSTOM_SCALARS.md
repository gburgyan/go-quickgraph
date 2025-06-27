# Custom Scalars

Custom scalars allow you to extend GraphQL's type system with your own data types. go-quickgraph makes it easy to create validated, serializable custom types like DateTime, Money, Email, and more.

## Quick Start

Here's a simple custom scalar for user IDs with validation:

```go
type UserID string

func (id UserID) Validate() error {
    if len(id) == 0 {
        return fmt.Errorf("user ID cannot be empty")
    }
    if !strings.HasPrefix(string(id), "user_") {
        return fmt.Errorf("user ID must start with 'user_'")
    }
    return nil
}

// Register the scalar
err := g.RegisterScalar(ctx, quickgraph.ScalarDefinition{
    Name:        "UserID",
    GoType:      reflect.TypeOf(UserID("")),
    Description: "Unique identifier for users (format: user_xxxxx)",
    Serialize: func(value interface{}) (interface{}, error) {
        if uid, ok := value.(UserID); ok {
            return string(uid), nil
        }
        return nil, fmt.Errorf("expected UserID, got %T", value)
    },
    ParseValue: func(value interface{}) (interface{}, error) {
        if str, ok := value.(string); ok {
            uid := UserID(str)
            if err := uid.Validate(); err != nil {
                return nil, err
            }
            return uid, nil
        }
        return nil, fmt.Errorf("expected string, got %T", value)
    },
})
```

Now use it in your functions:
```go
func GetUser(ctx context.Context, id UserID) (*User, error) {
    // id is already validated by the scalar
    return database.GetUser(string(id))
}

g.RegisterQuery(ctx, "user", GetUser, "id")
```

## Built-in Scalars

go-quickgraph provides ready-to-use scalars for common types:

### DateTime Scalar

```go
err := g.RegisterDateTimeScalar(ctx)
if err != nil {
    log.Fatal(err)
}

type Event struct {
    ID        int       `graphy:"id"`
    Name      string    `graphy:"name"`
    StartTime time.Time `graphy:"startTime"` // Uses DateTime scalar
}
```

**Features:**
- RFC3339 format (`2023-12-25T10:30:00Z`)
- Automatic time zone handling
- Validation of date format

**GraphQL Usage:**
```graphql
mutation {
  createEvent(
    name: "Conference"
    startTime: "2023-12-25T10:30:00Z"
  ) {
    id
    startTime
  }
}
```

### JSON Scalar

```go
err := g.RegisterJSONScalar(ctx)
if err != nil {
    log.Fatal(err)
}

type Product struct {
    ID       int                    `graphy:"id"`
    Name     string                 `graphy:"name"`
    Metadata map[string]interface{} `graphy:"metadata"` // Uses JSON scalar
}
```

**Features:**
- Applies for `map[string]interface{}` if registered
- Passes through arbitrary JSON data
- No validation (any valid JSON)
- Useful for flexible metadata fields

## Creating Custom Scalars

### ScalarDefinition Fields

When registering a custom scalar, you provide:

```go
type ScalarDefinition struct {
    Name         string                                           // GraphQL type name
    GoType       reflect.Type                                     // Go type this represents
    Description  string                                           // Optional description
    Serialize    func(value interface{}) (interface{}, error)     // Go → JSON
    ParseValue   func(value interface{}) (interface{}, error)     // Variables → Go
    ParseLiteral func(value interface{}) (interface{}, error)     // Optional: Query literals → Go
}
```

- **Serialize**: Convert Go values to JSON for output
- **ParseValue**: Convert JSON values from GraphQL variables to Go values  
- **ParseLiteral**: Convert literal values from GraphQL queries to Go values (optional)

### ParseValue vs ParseLiteral

**ParseValue** handles values from GraphQL **variables** (JSON input):
```graphql
# Variable comes from JSON: {"userId": "user_123"}
query GetUser($userId: UserID!) {
  user(id: $userId) { name }
}
```

**ParseLiteral** handles values written directly in **query literals**:
```graphql
# Literal value in query string
query GetUser {
  user(id: "user_123") { name }  
}
```

**When to provide ParseLiteral:**

1. **Different Input Formats**: Literals and variables use different representations
2. **Validation Differences**: Different rules for compile-time vs runtime values  
3. **Format Conversion**: Literals need different parsing than JSON-decoded values

**Example: DateTime with Different Formats**
```go
ParseValue: func(value interface{}) (interface{}, error) {
    // Variables: Unix timestamp from JSON
    if timestamp, ok := value.(float64); ok {
        return time.Unix(int64(timestamp), 0), nil
    }
    return nil, fmt.Errorf("expected timestamp")
},
ParseLiteral: func(value interface{}) (interface{}, error) {
    // Literals: ISO date string in query
    if str, ok := value.(string); ok {
        return time.Parse("2006-01-02", str)
    }
    return nil, fmt.Errorf("expected date string")
}
```

**Most scalars don't need ParseLiteral** - if not provided, ParseValue is used as fallback.

### Example: Money Scalar with Different Literal/Variable Formats

```go
type Money int64 // Amount in cents

func (m Money) String() string {
    return fmt.Sprintf("$%.2f", float64(m)/100)
}

err := g.RegisterScalar(ctx, quickgraph.ScalarDefinition{
    Name:        "Money",
    GoType:      reflect.TypeOf(Money(0)),
    Description: "Monetary amount - variables as cents (int), literals as dollars (string)",
    Serialize: func(value interface{}) (interface{}, error) {
        if money, ok := value.(Money); ok {
            return int64(money), nil // Always output cents as integer
        }
        return nil, fmt.Errorf("expected Money, got %T", value)
    },
    ParseValue: func(value interface{}) (interface{}, error) {
        // Variables: expect cents as integer from JSON
        switch v := value.(type) {
        case int:
            return Money(v), nil
        case int64:
            return Money(v), nil
        case float64:
            return Money(v), nil // Allow float from JSON
        default:
            return nil, fmt.Errorf("expected number for Money variable, got %T", value)
        }
    },
    ParseLiteral: func(value interface{}) (interface{}, error) {
        // Literals: expect dollar format like "$25.00" in query string
        if str, ok := value.(string); ok {
            if !strings.HasPrefix(str, "$") {
                return nil, fmt.Errorf("money literal must start with $, got: %s", str)
            }
            amount, err := strconv.ParseFloat(str[1:], 64)
            if err != nil {
                return nil, fmt.Errorf("invalid money format: %s", str)
            }
            return Money(amount * 100), nil // Convert dollars to cents
        }
        return nil, fmt.Errorf("expected string for Money literal, got %T", value)
    },
})
```

**Usage:**
```graphql
# Using variables (cents as integers)
query GetProductsInRange($minPrice: Money!, $maxPrice: Money!) {
  products(minPrice: $minPrice, maxPrice: $maxPrice) { name price }
}
# Variables: {"minPrice": 1000, "maxPrice": 5000}

# Using literals (dollars as strings)  
query GetExpensiveProducts {
  products(minPrice: "$10.00", maxPrice: "$50.00") { name price }
}
```

### Simple Example: UserID Scalar

Most scalars only need ParseValue since literals and variables use the same format:

```go
type UserID string

func (id UserID) Validate() error {
    if len(id) == 0 {
        return fmt.Errorf("user ID cannot be empty")
    }
    if !strings.HasPrefix(string(id), "user_") {
        return fmt.Errorf("user ID must start with 'user_'")
    }
    return nil
}

err := g.RegisterScalar(ctx, quickgraph.ScalarDefinition{
    Name:        "UserID",
    GoType:      reflect.TypeOf(UserID("")),
    Description: "Unique identifier for users (format: user_xxxxx)",
    Serialize: func(value interface{}) (interface{}, error) {
        if uid, ok := value.(UserID); ok {
            return string(uid), nil
        }
        return nil, fmt.Errorf("expected UserID, got %T", value)
    },
    ParseValue: func(value interface{}) (interface{}, error) {
        // Works for both variables and literals since both are strings
        if str, ok := value.(string); ok {
            uid := UserID(str)
            if err := uid.Validate(); err != nil {
                return nil, err
            }
            return uid, nil
        }
        return nil, fmt.Errorf("expected string, got %T", value)
    },
    // ParseLiteral not needed - same logic as ParseValue
})
```

**Usage in Types:**
```go
type User struct {
    ID    UserID `graphy:"id"`
    Name  string `graphy:"name"`
    Email string `graphy:"email"`
}

func GetUser(ctx context.Context, id UserID) (*User, error) {
    // id is already validated by the scalar
    return database.GetUser(string(id))
}

func CreateUser(ctx context.Context, name string) (*User, error) {
    // Generate new UserID
    id := UserID("user_" + generateID())
    return &User{ID: id, Name: name}, nil
}
```

**GraphQL Usage:**
```graphql
# Both variables and literals use the same string format
query {
  user(id: "user_12345") {
    id
    name
  }
}

# With variables
query GetUser($userId: UserID!) {
  user(id: $userId) {
    id  
    name
  }
}
# Variables: {"userId": "user_12345"}
```

### Example: Email Scalar with Validation

```go
type EmailAddress string

func (e EmailAddress) Validate() error {
    email := string(e)
    if !strings.Contains(email, "@") {
        return fmt.Errorf("invalid email format: missing @")
    }
    parts := strings.Split(email, "@")
    if len(parts) != 2 {
        return fmt.Errorf("invalid email format: multiple @ symbols")
    }
    if len(parts[0]) == 0 {
        return fmt.Errorf("invalid email format: missing local part")
    }
    if len(parts[1]) == 0 || !strings.Contains(parts[1], ".") {
        return fmt.Errorf("invalid email format: missing or invalid domain")
    }
    return nil
}

err := g.RegisterScalar(ctx, quickgraph.ScalarDefinition{
    Name:        "EmailAddress",
    GoType:      reflect.TypeOf(EmailAddress("")),
    Description: "Valid email address",
    Serialize: func(value interface{}) (interface{}, error) {
        if email, ok := value.(EmailAddress); ok {
            return string(email), nil
        }
        return nil, fmt.Errorf("expected EmailAddress, got %T", value)
    },
    ParseValue: func(value interface{}) (interface{}, error) {
        if str, ok := value.(string); ok {
            email := EmailAddress(str)
            if err := email.Validate(); err != nil {
                return nil, err
            }
            return email, nil
        }
        return nil, fmt.Errorf("expected string, got %T", value)
    },
})
```

### Example: Color Scalar

```go
type HexColor string

func (c HexColor) Validate() error {
    color := string(c)
    if !strings.HasPrefix(color, "#") {
        return fmt.Errorf("hex color must start with #")
    }
    
    hex := color[1:]
    if len(hex) != 3 && len(hex) != 6 {
        return fmt.Errorf("hex color must be #RGB or #RRGGBB format")
    }
    
    for _, r := range hex {
        if !((r >= '0' && r <= '9') || (r >= 'A' && r <= 'F') || (r >= 'a' && r <= 'f')) {
            return fmt.Errorf("invalid hex character: %c", r)
        }
    }
    return nil
}

// Normalize short form to long form
func (c HexColor) Normalize() HexColor {
    color := string(c)
    if len(color) == 4 { // #RGB
        r, g, b := color[1], color[2], color[3]
        return HexColor(fmt.Sprintf("#%c%c%c%c%c%c", r, r, g, g, b, b))
    }
    return c
}

err := g.RegisterScalar(ctx, quickgraph.ScalarDefinition{
    Name:        "HexColor",
    GoType:      reflect.TypeOf(HexColor("")),
    Description: "Hexadecimal color representation (e.g., #FF0000 or #F00)",
    Serialize: func(value interface{}) (interface{}, error) {
        if color, ok := value.(HexColor); ok {
            return string(color.Normalize()), nil // Always return normalized form
        }
        return nil, fmt.Errorf("expected HexColor, got %T", value)
    },
    ParseValue: func(value interface{}) (interface{}, error) {
        if str, ok := value.(string); ok {
            color := HexColor(str)
            if err := color.Validate(); err != nil {
                return nil, err
            }
            return color.Normalize(), nil
        }
        return nil, fmt.Errorf("expected string, got %T", value)
    },
})
```

## Advanced Patterns

### Composite Scalars

Create scalars that represent complex data as strings:

```go
type Coordinates struct {
    Lat float64
    Lng float64
}

type GeoPoint Coordinates

func (g GeoPoint) String() string {
    return fmt.Sprintf("%.6f,%.6f", g.Lat, g.Lng)
}

func ParseGeoPoint(s string) (GeoPoint, error) {
    parts := strings.Split(s, ",")
    if len(parts) != 2 {
        return GeoPoint{}, fmt.Errorf("invalid format: expected lat,lng")
    }
    
    lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
    if err != nil {
        return GeoPoint{}, fmt.Errorf("invalid latitude: %v", err)
    }
    
    lng, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
    if err != nil {
        return GeoPoint{}, fmt.Errorf("invalid longitude: %v", err)
    }
    
    return GeoPoint{Lat: lat, Lng: lng}, nil
}

err := g.RegisterScalar(ctx, quickgraph.ScalarDefinition{
    Name:        "GeoPoint",
    GoType:      reflect.TypeOf(GeoPoint{}),
    Description: "Geographic coordinates as lat,lng string",
    Serialize: func(value interface{}) (interface{}, error) {
        if point, ok := value.(GeoPoint); ok {
            return point.String(), nil
        }
        return nil, fmt.Errorf("expected GeoPoint, got %T", value)
    },
    ParseValue: func(value interface{}) (interface{}, error) {
        if str, ok := value.(string); ok {
            return ParseGeoPoint(str)
        }
        return nil, fmt.Errorf("expected string, got %T", value)
    },
})
```

### Pointer Support

Support both value and pointer types:

```go
err := g.RegisterScalar(ctx, quickgraph.ScalarDefinition{
    Name:   "UserID",
    GoType: reflect.TypeOf(UserID("")),
    Serialize: func(value interface{}) (interface{}, error) {
        switch v := value.(type) {
        case UserID:
            return string(v), nil
        case *UserID:
            if v != nil {
                return string(*v), nil
            }
            return nil, nil // Handle nil pointers
        default:
            return nil, fmt.Errorf("expected UserID or *UserID, got %T", value)
        }
    },
    ParseValue: func(value interface{}) (interface{}, error) {
        if str, ok := value.(string); ok {
            uid := UserID(str)
            if err := uid.Validate(); err != nil {
                return nil, err
            }
            return uid, nil
        }
        return nil, fmt.Errorf("expected string, got %T", value)
    },
})
```

## Schema Integration

### Generated Schema

Custom scalars automatically appear in your GraphQL schema:

```graphql
scalar DateTime # RFC3339 formatted date-time string
scalar EmailAddress # Valid email address
scalar HexColor # Hexadecimal color representation (e.g., #FF0000 or #F00)
scalar Money # Monetary amount in cents
scalar UserID # Unique identifier for users (format: user_xxxxx)

type User {
  id: UserID!
  email: EmailAddress!
  createdAt: DateTime!
}

type Product {
  id: String!
  name: String!
  price: Money!
  color: HexColor
}
```

### Introspection

Custom scalars support GraphQL introspection:

```graphql
query {
  __type(name: "Money") {
    name
    kind
    description
  }
}
```

## Error Handling

### Validation Errors

Scalar validation errors become GraphQL errors:

```graphql
# Invalid email
mutation {
  createUser(email: "invalid-email") {
    id
  }
}
```

**Response:**
```json
{
  "data": null,
  "errors": [{
    "message": "invalid email format: missing @",
    "locations": [{"line": 2, "column": 15}],
    "path": ["createUser"]
  }]
}
```

### Error Best Practices

```go
// ✅ Descriptive, actionable errors
ParseValue: func(value interface{}) (interface{}, error) {
    if str, ok := value.(string); ok {
        if len(str) == 0 {
            return nil, fmt.Errorf("UserID cannot be empty")
        }
        if !strings.HasPrefix(str, "user_") {
            return nil, fmt.Errorf("UserID must start with 'user_', got: %s", str)
        }
        return UserID(str), nil
    }
    return nil, fmt.Errorf("expected string for UserID, got %T", value)
}

// ❌ Generic, unhelpful errors
ParseValue: func(value interface{}) (interface{}, error) {
    // ... validation ...
    return nil, fmt.Errorf("invalid")
}
```

## Management Functions

### Checking Registered Scalars

```go
// Check if scalar exists
scalar, exists := g.GetScalarByName("UserID")
if exists {
    fmt.Printf("Scalar %s maps to Go type %v\n", scalar.Name, scalar.GoType)
}

// Get scalar by Go type
scalar, exists = g.GetScalarByType(reflect.TypeOf(UserID("")))

// Get all registered scalars
scalars := g.GetRegisteredScalars()
for name, scalar := range scalars {
    fmt.Printf("Scalar %s: %s\n", name, scalar.Description)
}
```

### Registration Validation

The `RegisterScalar` function validates definitions:

```go
err := g.RegisterScalar(ctx, definition)
if err != nil {
    // Handle registration errors:
    // - Empty name
    // - Invalid GraphQL name format
    // - Nil GoType, Serialize, or ParseValue
    // - Duplicate name or type registration
    log.Printf("Failed to register scalar: %v", err)
}
```

## Best Practices

### 1. Validate Early and Clearly
```go
// ✅ Clear validation with helpful messages
func (e EmailAddress) Validate() error {
    email := string(e)
    if len(email) == 0 {
        return fmt.Errorf("email address cannot be empty")
    }
    if !strings.Contains(email, "@") {
        return fmt.Errorf("email address must contain @")
    }
    // More specific validation...
}

// ❌ Generic validation
func (e EmailAddress) Validate() error {
    if !isValidEmail(string(e)) {
        return fmt.Errorf("invalid email")
    }
}
```

### 2. Support Multiple Input Formats
```go
// ✅ Flexible input parsing
ParseValue: func(value interface{}) (interface{}, error) {
    switch v := value.(type) {
    case string:
        return ParseMoney(v) // "$25.00"
    case int:
        return Money(v), nil // 2500 (cents)
    case float64:
        return Money(v * 100), nil // 25.00 (dollars)
    default:
        return nil, fmt.Errorf("expected string, int, or float64")
    }
}
```

### 3. Consistent Serialization
```go
// ✅ Always output in the same format
Serialize: func(value interface{}) (interface{}, error) {
    if color, ok := value.(HexColor); ok {
        return string(color.Normalize()), nil // Always 6-digit format
    }
    return nil, fmt.Errorf("expected HexColor")
}

// ❌ Inconsistent output format
Serialize: func(value interface{}) (interface{}, error) {
    // Sometimes returns #F00, sometimes #FF0000
}
```

### 4. Register Before Use
```go
// ✅ Register scalars before functions that use them
func setupGraphQL() *quickgraph.Graphy {
    g := &quickgraph.Graphy{}
    
    // Register scalars first
    g.RegisterDateTimeScalar(ctx)
    g.RegisterScalar(ctx, moneyScalarDef)
    g.RegisterScalar(ctx, emailScalarDef)
    
    // Then register functions
    g.RegisterQuery(ctx, "user", GetUser, "id")
    g.RegisterMutation(ctx, "createProduct", CreateProduct, "input")
    
    return g
}
```

## Examples from Sample App

See the [sample application](https://github.com/gburgyan/go-quickgraph-sample) for working examples:

- **Money Scalar**: Monetary amounts with validation
- **HexColor Scalar**: Color values with format validation
- **EmailAddress Scalar**: Email validation with detailed error messages
- **DateTime Integration**: Using built-in DateTime scalar
- **Complex Validation**: Custom business rules in scalar parsing

## Next Steps

- **[Function Patterns](FUNCTION_PATTERNS.md)** - Using custom scalars in function parameters
- **[Type System Guide](TYPE_SYSTEM.md)** - Combining scalars with complex types
- **[Performance](PERFORMANCE.md)** - Optimizing scalar serialization and validation