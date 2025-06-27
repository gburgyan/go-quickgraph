# Type System Guide

go-quickgraph supports the full GraphQL type system using idiomatic Go patterns. This guide covers structs, interfaces, unions, enums, and advanced type relationships.

## Basic Types

### Structs → Object Types

Go structs automatically become GraphQL object types:

```go
type User struct {
    ID       int       `graphy:"id"`
    Name     string    `graphy:"name"`
    Email    *string   `graphy:"email"`    // Optional field (nullable)
    Posts    []Post    `graphy:"posts"`    // Array of non-nullable Posts
    Settings *Settings `graphy:"settings"` // Optional nested object
}

type Post struct {
    ID    int    `graphy:"id"`
    Title string `graphy:"title"`
    Body  string `graphy:"body"`
}

type Settings struct {
    Theme       string `graphy:"theme"`
    Newsletters bool   `graphy:"newsletters"`
}
```

**Generated GraphQL:**
```graphql
type User {
  id: Int!
  name: String!
  email: String          # Nullable because of *string
  posts: [Post!]!        # Non-null array of non-null Posts
  settings: Settings     # Nullable nested object
}

type Post {
  id: Int!
  title: String!
  body: String!
}

type Settings {
  theme: String!
  newsletters: Boolean!
}
```

### Struct Field Tags

go-quickgraph uses the `graphy` tag to control GraphQL field naming and metadata. The library also supports `json` tags for backward compatibility.

```go
type Product struct {
    // Primary: graphy tag for field naming and metadata
    ID          int     `graphy:"id,description=Unique product identifier"`
    
    // Standard field naming
    Name        string  `graphy:"productName,description=Product display name"`
    
    // Standard field
    Price       float64 `graphy:"price"`
    
    // Advanced graphy tag features
    Deprecated  string  `graphy:"oldField,deprecated=Use newField instead"`
    
    // Exclude fields
    Internal    string  `graphy:"-"`        // Excluded from GraphQL schema
    Secret      string  `graphy:"-"`          // Also excluded
}
```

**Tag Priority:**
1. `graphy` tag (highest priority - recommended)
2. `json` tag (fallback for backward compatibility)
3. Field name (if no tags)

**Graphy Tag Format:**
- Simple: `graphy:"fieldName"`
- With metadata: `graphy:"fieldName,description=Field description"`
- Multiple attributes: `graphy:"name=fieldName,deprecated=Reason,description=Desc"`
- Exclude field: `graphy:"-"`

### Field Methods

Add methods to structs to create computed fields:

```go
type User struct {
    ID        int    `graphy:"id"`
    FirstName string `graphy:"firstName"`
    LastName  string `graphy:"lastName"`
}

// Method becomes a GraphQL field
func (u *User) FullName() string {
    return u.FirstName + " " + u.LastName
}

// Method with parameters becomes a field with arguments
func (u *User) Posts(ctx context.Context, limit *int) ([]Post, error) {
    maxPosts := 10
    if limit != nil && *limit > 0 {
        maxPosts = *limit
    }
    return getPostsByUser(u.ID, maxPosts)
}
```

**Generated GraphQL:**
```graphql
type User {
  id: Int!
  firstName: String!
  lastName: String!
  fullName: String!                    # Computed field
  posts(limit: Int): [Post!]!          # Field with arguments
}
```

## Enums

### String-Based Enums

Implement the `StringEnumValues` interface:

```go
type UserRole string

func (r UserRole) EnumValues() []string {
    return []string{"ADMIN", "USER", "MODERATOR"}
}

// Use in your types
type User struct {
    ID   int      `graphy:"id"`
    Name string   `graphy:"name"`
    Role UserRole `graphy:"role"`
}
```

**Generated GraphQL:**
```graphql
enum UserRole {
  ADMIN
  USER
  MODERATOR
}

type User {
  id: Int!
  name: String!
  role: UserRole!
}
```

### Validation

Enums automatically validate input:

```go
// This query will fail with "invalid enum value INVALID_ROLE"
mutation {
  createUser(input: {name: "Alice", role: "INVALID_ROLE"}) {
    id
  }
}
```

## Interfaces

Use Go's anonymous struct embedding to create GraphQL interfaces:

```go
// Base type becomes the interface
type Node struct {
    ID   string `graphy:"id"`
    Type string `graphy:"type"`
}

// Types that embed Node implement the Node interface
type User struct {
    Node                              // Anonymous embedding
    Name  string `graphy:"name"`
    Email string `graphy:"email"`
}

type Post struct {
    Node                              // Anonymous embedding
    Title string `graphy:"title"`
    Body  string `graphy:"body"`
}
```

**Generated GraphQL:**
```graphql
interface INode {
  id: String!
  type: String!
}

type Node implements INode {
  id: String!
  type: String!
}

type User implements INode {
  id: String!
  type: String!
  name: String!
  email: String!
}

type Post implements INode {
  id: String!
  type: String!
  title: String!
  body: String!
}
```

### Interface-Only Types

Sometimes you only want the interface, not the concrete type:

```go
type BaseEntity struct {
    ID        string    `graphy:"id"`
    CreatedAt time.Time `graphy:"createdAt"`
}

// Opt out of concrete type generation
func (b BaseEntity) GraphTypeExtension() GraphTypeInfo {
    return GraphTypeInfo{
        Name:          "BaseEntity",
        InterfaceOnly: true,
    }
}

type User struct {
    BaseEntity
    Name string `graphy:"name"`
}

type Product struct {
    BaseEntity  
    Name  string  `graphy:"name"`
    Price float64 `graphy:"price"`
}
```

**Generated GraphQL:**
```graphql
interface BaseEntity {
  id: String!
  createdAt: DateTime!
}

type User implements BaseEntity {
  id: String!
  createdAt: DateTime!
  name: String!
}

type Product implements BaseEntity {
  id: String!
  createdAt: DateTime!
  name: String!
  price: Float!
}
```

### Using Interfaces in Queries

```graphql
# Query for interface type with fragments
{
  searchNodes(query: "alice") {
    id
    type
    ... on User {
      name
      email
    }
    ... on Post {
      title
      body
    }
  }
}
```

## Unions

### Explicit Union Types

Create union types by ending struct names with "Union":

```go
type SearchResultUnion struct {
    User    *User    
    Post    *Post
    Product *Product
}

func Search(ctx context.Context, query string) (*SearchResultUnion, error) {
    // Return only one non-nil field
    if isUserQuery(query) {
        user := findUser(query)
        return &SearchResultUnion{User: user}, nil
    } else if isPostQuery(query) {
        post := findPost(query)
        return &SearchResultUnion{Post: post}, nil
    }
    // etc.
}
```

**Generated GraphQL:**
```graphql
union SearchResult = User | Post | Product

type Query {
  search(query: String!): SearchResult
}
```

### Implicit Unions (Multiple Return Values)

Functions returning multiple pointers create implicit unions:

```go
func GetContent(ctx context.Context, id string) (*Article, *Video, error) {
    content := findContent(id)
    switch content.Type {
    case "article":
        return &Article{...}, nil, nil
    case "video":
        return nil, &Video{...}, nil
    default:
        return nil, nil, fmt.Errorf("content not found")
    }
}
```

**Generated GraphQL:**
```graphql
union GetContentResultUnion = Article | Video

type Query {
  getContent(id: String!): GetContentResultUnion
}
```

### Interface Expansion in Unions

When a union contains an interface type, it automatically expands to include all implementations:

```go
// Interface type
type Employee struct {
    ID   int    `graphy:"id"`
    Name string `graphy:"name"`
}

// Implementations
type Developer struct {
    Employee
    Languages []string `graphy:"languages"`
}

type Manager struct {
    Employee
    Department string `graphy:"department"`
}

// Union with interface
type SearchResultUnion struct {
    Employee *Employee  // Expands to Developer, Manager, and Employee
    Product  *Product
}

// Register all types so they appear in schema
func setupSchema(g *quickgraph.Graphy) {
    g.RegisterTypes(ctx, Developer{}, Manager{})
}
```

**Generated GraphQL:**
```graphql
union SearchResult = Developer | Employee | Manager | Product
```

## Advanced Type Patterns

### Optional Fields with Pointers

Use pointers to make fields optional (nullable):

```go
type User struct {
    ID       int     `graphy:"id"`
    Name     string  `graphy:"name"`      // Required
    Email    *string `graphy:"email"`     // Optional
    Age      *int    `graphy:"age"`       // Optional
    IsActive bool    `graphy:"isActive"`  // Required (defaults to false)
}
```

### Nested Objects and Arrays

```go
type User struct {
    ID       int       `graphy:"id"`
    Profile  Profile   `graphy:"profile"`     // Required nested object
    Posts    []Post    `graphy:"posts"`       // Required array of required items
    Comments []*Comment `graphy:"comments"`    // Required array of optional items
    Tags     *[]string `graphy:"tags"`        // Optional array of required strings
}

type Profile struct {
    Bio       *string `graphy:"bio"`
    AvatarURL *string `graphy:"avatarUrl"`
}
```

### Circular References

Handle circular references carefully:

```go
type User struct {
    ID   int    `graphy:"id"`
    Name string `graphy:"name"`
}

// Add relationships as methods to avoid infinite recursion
func (u *User) Posts(ctx context.Context) ([]Post, error) {
    return getPostsByUserID(u.ID)
}

type Post struct {
    ID    int    `graphy:"id"`
    Title string `graphy:"title"`
}

func (p *Post) Author(ctx context.Context) (*User, error) {
    return getUserByPostID(p.ID)
}
```

### Type Discovery and Runtime Resolution

For polymorphic returns, use type discovery:

```go
type Content struct {
    ID   string `graphy:"id"`
    Type string `graphy:"type"`
    // Private field for actual type
    actualType interface{} `graphy:"-"`
}

// Enable runtime type resolution
func (c *Content) ActualType() interface{} {
    return c.actualType
}

// Constructor that sets actual type
func NewArticle(id, title, body string) *Article {
    a := &Article{
        Content: Content{ID: id, Type: "article"},
        Title:   title,
        Body:    body,
    }
    a.Content.actualType = a // Enable type discovery
    return a
}

type Article struct {
    Content
    Title string `graphy:"title"`
    Body  string `graphy:"body"`
}

type Video struct {
    Content
    Duration int    `graphy:"duration"`
    URL      string `graphy:"url"`
}

// Function can return base type
func GetContent(ctx context.Context, id string) (*Content, error) {
    // Load from database and create appropriate type
    if contentType == "article" {
        article := NewArticle(id, title, body)
        return &article.Content, nil
    }
    // etc.
}
```

## Input Types

### Struct Input Types

Go structs become GraphQL input types when used as function parameters:

```go
type CreateUserInput struct {
    Name     string   `graphy:"name"`
    Email    string   `graphy:"email"`
    Role     UserRole `graphy:"role"`
    Settings *UserSettingsInput `graphy:"settings"`
}

type UserSettingsInput struct {
    Theme       string `graphy:"theme"`
    Newsletters bool   `graphy:"newsletters"`
}

func CreateUser(ctx context.Context, input CreateUserInput) (*User, error) {
    // Implementation
}
```

**Generated GraphQL:**
```graphql
input CreateUserInput {
  name: String!
  email: String!
  role: UserRole!
  settings: UserSettingsInput
}

input UserSettingsInput {
  theme: String!
  newsletters: Boolean!
}

type Mutation {
  createUser(input: CreateUserInput!): User
}
```

### Anonymous Fields in Inputs

Share common fields across input types:

```go
type PaginationInput struct {
    Limit  int `graphy:"limit"`
    Offset int `graphy:"offset"`
}

type UserSearchInput struct {
    PaginationInput      // Fields promoted to top level
    Query   string       `graphy:"query"`
    Role    *UserRole    `graphy:"role"`
}

func SearchUsers(ctx context.Context, input UserSearchInput) ([]*User, error) {
    // Can access input.Limit, input.Offset directly
}
```

**GraphQL Usage:**
```graphql
{
  searchUsers(query: "alice", limit: 10, offset: 0, role: ADMIN) {
    id
    name
  }
}
```

## Best Practices

### 1. Use Descriptive Types
```go
// ✅ Clear, specific types
type CreateUserInput struct {
    Name  string   `graphy:"name"`
    Email string   `graphy:"email"`
    Role  UserRole `graphy:"role"`
}

// ❌ Generic, unclear types
type UserData struct {
    Field1 string `graphy:"field1"`
    Field2 string `graphy:"field2"`
    Field3 string `graphy:"field3"`
}
```

### 2. Leverage Pointers for Optionality
```go
// ✅ Clear optional vs required fields
type UpdateUserInput struct {
    Name     *string   `graphy:"name"`     // Optional update
    Email    *string   `graphy:"email"`    // Optional update
    IsActive *bool     `graphy:"isActive"` // Optional update (including false)
}

// ❌ Required fields for partial updates
type UpdateUserInput struct {
    Name     string `graphy:"name"`     // Must always provide
    Email    string `graphy:"email"`    // Must always provide
    IsActive bool   `graphy:"isActive"` // Can't distinguish false from unset
}
```

### 3. Use Methods for Computed Fields
```go
// ✅ Dynamic, context-aware fields
func (u *User) PostCount(ctx context.Context) (int, error) {
    return database.CountPostsByUser(ctx, u.ID)
}

// ❌ Static fields that may be stale
type User struct {
    ID        int `graphy:"id"`
    PostCount int `graphy:"postCount"` // May be outdated
}
```

### 4. Design for GraphQL Queries
```go
// ✅ Efficient for GraphQL selection
type User struct {
    ID      int     `graphy:"id"`
    Name    string  `graphy:"name"`
    Profile Profile `graphy:"profile"` // Separate object for complex data
}

func (u *User) Posts(ctx context.Context, limit *int) ([]Post, error) {
    // Only load when requested
}

// ❌ Always loads everything
type User struct {
    ID           int      `graphy:"id"`
    Name         string   `graphy:"name"`
    Bio          string   `graphy:"bio"`
    AvatarURL    string   `graphy:"avatarUrl"`
    AllUserPosts []Post   `graphy:"posts"` // Always loaded, inefficient
}
```

## Next Steps

- **[Custom Scalars](CUSTOM_SCALARS.md)** - DateTime, Money, and validation
- **[Function Patterns](FUNCTION_PATTERNS.md)** - Parameter handling and return patterns
- **[Subscriptions](SUBSCRIPTIONS.md)** - Real-time updates with complex types