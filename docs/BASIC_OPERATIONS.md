# Basic Operations

This guide covers the fundamentals of GraphQL queries and mutations in go-quickgraph. You'll learn how to create, register, and test basic GraphQL operations.

## Queries (Read Operations)

Queries are read-only operations that fetch data. They map directly to Go functions that return data.

### Simple Query

```go
type User struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}

func GetUser(ctx context.Context, id int) (*User, error) {
    // Simulate database lookup
    users := []User{
        {ID: 1, Name: "Alice"},
        {ID: 2, Name: "Bob"},
    }
    
    for _, user := range users {
        if user.ID == id {
            return &user, nil
        }
    }
    
    return nil, fmt.Errorf("user with ID %d not found", id)
}

// Register the query
g.RegisterQuery(ctx, "user", GetUser, "id")
```

**GraphQL Usage:**
```graphql
{
  user(id: 1) {
    id
    name
  }
}
```

**Response:**
```json
{
  "data": {
    "user": {
      "id": 1,
      "name": "Alice"
    }
  }
}
```

### Query Without Parameters

```go
func GetAllUsers(ctx context.Context) ([]User, error) {
    // Return all users
    return getAllUsersFromDB()
}

g.RegisterQuery(ctx, "users", GetAllUsers)
```

**GraphQL Usage:**
```graphql
{
  users {
    id
    name
  }
}
```

### Query with Multiple Parameters

```go
func SearchUsers(ctx context.Context, query string, limit int, offset int) ([]User, error) {
    if limit <= 0 {
        limit = 10 // Default limit
    }
    if limit > 100 {
        limit = 100 // Max limit
    }
    
    return searchUsersInDB(query, limit, offset)
}

g.RegisterQuery(ctx, "searchUsers", SearchUsers, "query", "limit", "offset")
```

**GraphQL Usage:**
```graphql
{
  searchUsers(query: "alice", limit: 5, offset: 0) {
    id
    name
  }
}
```

### Query with Optional Parameters

Use pointers to make parameters optional:

```go
func GetUsers(ctx context.Context, limit *int, role *string) ([]User, error) {
    // Set defaults for nil parameters
    maxLimit := 10
    if limit != nil {
        maxLimit = *limit
    }
    
    var roleFilter string
    if role != nil {
        roleFilter = *role
    }
    
    return filterUsers(maxLimit, roleFilter)
}

g.RegisterQuery(ctx, "users", GetUsers, "limit", "role")
```

**GraphQL Usage:**
```graphql
# All parameters optional
{ users { id name } }

# With some parameters
{ users(limit: 5) { id name } }

# With all parameters
{ users(limit: 20, role: "admin") { id name } }
```

## Mutations (Write Operations)

Mutations are write operations that modify data. They also map to Go functions but are registered differently.

### Simple Mutation

```go
type CreateUserInput struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

func CreateUser(ctx context.Context, input CreateUserInput) (*User, error) {
    // Validation
    if input.Name == "" {
        return nil, fmt.Errorf("name is required")
    }
    if input.Email == "" {
        return nil, fmt.Errorf("email is required")
    }
    
    // Create new user
    user := &User{
        ID:    generateID(),
        Name:  input.Name,
        Email: input.Email,
    }
    
    // Save to database
    err := saveUserToDB(user)
    if err != nil {
        return nil, fmt.Errorf("failed to save user: %w", err)
    }
    
    return user, nil
}

g.RegisterMutation(ctx, "createUser", CreateUser, "input")
```

**GraphQL Usage:**
```graphql
mutation {
  createUser(input: {
    name: "Charlie"
    email: "charlie@example.com"
  }) {
    id
    name
    email
  }
}
```

### Update Mutation

```go
type UpdateUserInput struct {
    ID    int     `json:"id"`
    Name  *string `json:"name"`  // Optional update
    Email *string `json:"email"` // Optional update
}

func UpdateUser(ctx context.Context, input UpdateUserInput) (*User, error) {
    // Find existing user
    user, err := getUserByID(input.ID)
    if err != nil {
        return nil, fmt.Errorf("user not found: %w", err)
    }
    
    // Update only provided fields
    if input.Name != nil {
        user.Name = *input.Name
    }
    if input.Email != nil {
        user.Email = *input.Email
    }
    
    // Save changes
    err = saveUserToDB(user)
    if err != nil {
        return nil, fmt.Errorf("failed to update user: %w", err)
    }
    
    return user, nil
}

g.RegisterMutation(ctx, "updateUser", UpdateUser, "input")
```

**GraphQL Usage:**
```graphql
mutation {
  updateUser(input: {
    id: 1
    name: "Alice Updated"
  }) {
    id
    name
    email
  }
}
```

### Delete Mutation

```go
func DeleteUser(ctx context.Context, id int) (bool, error) {
    err := deleteUserFromDB(id)
    if err != nil {
        return false, fmt.Errorf("failed to delete user: %w", err)
    }
    return true, nil
}

g.RegisterMutation(ctx, "deleteUser", DeleteUser, "id")
```

**GraphQL Usage:**
```graphql
mutation {
  deleteUser(id: 1)
}
```

## Function Parameter Patterns

### Struct-Based Parameters (Recommended)

Use structs for complex input - fields become GraphQL arguments:

```go
type CreatePostInput struct {
    Title    string   `json:"title"`
    Body     string   `json:"body"`
    AuthorID int      `json:"authorID"`
    Tags     []string `json:"tags"`
    Draft    *bool    `json:"draft"` // Optional field
}

func CreatePost(ctx context.Context, input CreatePostInput) (*Post, error) {
    isDraft := false
    if input.Draft != nil {
        isDraft = *input.Draft
    }
    
    post := &Post{
        ID:       generateID(),
        Title:    input.Title,
        Body:     input.Body,
        AuthorID: input.AuthorID,
        Tags:     input.Tags,
        Draft:    isDraft,
    }
    
    return savePost(post)
}

g.RegisterMutation(ctx, "createPost", CreatePost, "input")
```

**GraphQL Schema Generated:**
```graphql
input CreatePostInput {
  title: String!
  body: String!
  authorID: Int!
  tags: [String!]!
  draft: Boolean
}

type Mutation {
  createPost(input: CreatePostInput!): Post
}
```

### Named Parameters

For simple functions with few parameters:

```go
func GetPost(ctx context.Context, id int, includeDrafts bool) (*Post, error) {
    return getPostFromDB(id, includeDrafts)
}

g.RegisterQuery(ctx, "post", GetPost, "id", "includeDrafts")
```

**GraphQL Usage:**
```graphql
{
  post(id: 1, includeDrafts: false) {
    title
    body
  }
}
```

### Positional Parameters

When parameter names don't matter:

```go
func SearchPosts(ctx context.Context, query string, limit int) ([]Post, error) {
    return searchPostsInDB(query, limit)
}

// No parameter names provided - uses arg1, arg2, etc.
g.RegisterQuery(ctx, "searchPosts", SearchPosts)
```

**GraphQL Usage:**
```graphql
{
  searchPosts(arg1: "golang", arg2: 10) {
    title
  }
}
```

## Return Value Patterns

### Single Object

```go
func GetUser(ctx context.Context, id int) (*User, error) {
    return getUserFromDB(id) // Returns pointer for nullable result
}
```

### Array of Objects

```go
func GetUsers(ctx context.Context) ([]User, error) {
    return getAllUsers() // Slice for array
}

func GetUserPointers(ctx context.Context) ([]*User, error) {
    return getAllUserPointers() // Slice of pointers for nullable items
}
```

### Multiple Return Values (Union Types)

```go
func SearchContent(ctx context.Context, query string) (*Post, *User, error) {
    // Return only one non-nil value
    if isPostQuery(query) {
        post, err := findPost(query)
        return post, nil, err
    } else {
        user, err := findUser(query)
        return nil, user, err
    }
}
```

**Generated Union Schema:**
```graphql
union SearchContentResultUnion = Post | User

type Query {
  searchContent(query: String!): SearchContentResultUnion
}
```

## Error Handling

### Returning Errors

Always return meaningful errors:

```go
func GetUser(ctx context.Context, id int) (*User, error) {
    if id <= 0 {
        return nil, fmt.Errorf("user ID must be positive, got %d", id)
    }
    
    user, err := database.GetUser(id)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("user with ID %d not found", id)
        }
        return nil, fmt.Errorf("database error: %w", err)
    }
    
    return user, nil
}
```

**Error Response:**
```json
{
  "data": {
    "user": null
  },
  "errors": [
    {
      "message": "user with ID -1 not found",
      "locations": [{"line": 2, "column": 3}],
      "path": ["user"]
    }
  ]
}
```

### Validation Errors

```go
type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func CreateUser(ctx context.Context, input CreateUserInput) (*User, error) {
    if input.Name == "" {
        return nil, ValidationError{
            Field:   "name",
            Message: "name is required",
        }
    }
    
    if !isValidEmail(input.Email) {
        return nil, ValidationError{
            Field:   "email", 
            Message: "must be a valid email address",
        }
    }
    
    // Continue with creation...
}
```

## Context Usage

### Authentication

```go
func GetCurrentUser(ctx context.Context) (*User, error) {
    userID, ok := ctx.Value("userID").(string)
    if !ok {
        return nil, fmt.Errorf("authentication required")
    }
    
    return getUserByID(userID)
}

func UpdateProfile(ctx context.Context, input ProfileInput) (*User, error) {
    // Verify user can update this profile
    currentUserID := ctx.Value("userID").(string)
    if currentUserID != input.UserID {
        return nil, fmt.Errorf("cannot update another user's profile")
    }
    
    return updateUserProfile(input)
}
```

### Request Cancellation

```go
func LongRunningQuery(ctx context.Context) (*Result, error) {
    // Check for cancellation periodically
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Use context with database operations
    result, err := database.QueryWithContext(ctx, "SELECT ...")
    return result, err
}
```

### Request Metadata

```go
func GetUserWithMetrics(ctx context.Context, id int) (*User, error) {
    // Add timing metrics
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        log.Printf("GetUser took %v", duration)
    }()
    
    // Check feature flags
    if ctx.Value("feature:newUserFormat") == "enabled" {
        return getUserWithNewFormat(id)
    }
    
    return getUserStandard(id)
}
```

## Input Validation

### Basic Validation

```go
func CreatePost(ctx context.Context, input CreatePostInput) (*Post, error) {
    // Required field validation
    if strings.TrimSpace(input.Title) == "" {
        return nil, fmt.Errorf("title cannot be empty")
    }
    
    // Length validation
    if len(input.Title) > 200 {
        return nil, fmt.Errorf("title cannot exceed 200 characters")
    }
    
    // Business logic validation
    if input.AuthorID <= 0 {
        return nil, fmt.Errorf("invalid author ID")
    }
    
    // Check if author exists
    _, err := getUserByID(input.AuthorID)
    if err != nil {
        return nil, fmt.Errorf("author not found: %w", err)
    }
    
    return createPost(input)
}
```

### Custom Validators

```go
type UserInput struct {
    Name     string `json:"name" validate:"required,min=2,max=50"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"min=13,max=120"`
}

func (u UserInput) Validate() error {
    if u.Name == "" {
        return fmt.Errorf("name is required")
    }
    if len(u.Name) < 2 {
        return fmt.Errorf("name must be at least 2 characters")
    }
    if !isValidEmail(u.Email) {
        return fmt.Errorf("invalid email format")
    }
    if u.Age < 13 || u.Age > 120 {
        return fmt.Errorf("age must be between 13 and 120")
    }
    return nil
}

func CreateUser(ctx context.Context, input UserInput) (*User, error) {
    if err := input.Validate(); err != nil {
        return nil, err
    }
    
    return createUser(input)
}
```

## Testing Operations

### Unit Testing Queries

```go
func TestGetUser(t *testing.T) {
    ctx := context.Background()
    
    // Test successful case
    user, err := GetUser(ctx, 1)
    assert.NoError(t, err)
    assert.Equal(t, 1, user.ID)
    assert.Equal(t, "Alice", user.Name)
    
    // Test error case
    user, err = GetUser(ctx, 999)
    assert.Error(t, err)
    assert.Nil(t, user)
    assert.Contains(t, err.Error(), "not found")
}
```

### Integration Testing

```go
func TestCreateUserMutation(t *testing.T) {
    ctx := context.Background()
    g := &quickgraph.Graphy{}
    g.RegisterMutation(ctx, "createUser", CreateUser, "input")
    
    query := `
        mutation {
            createUser(input: {
                name: "Test User"
                email: "test@example.com"
            }) {
                id
                name
                email
            }
        }
    `
    
    result, err := g.ProcessRequest(ctx, query, "")
    assert.NoError(t, err)
    assert.Contains(t, result, `"name":"Test User"`)
    assert.Contains(t, result, `"email":"test@example.com"`)
}
```

### Testing with Variables

```go
func TestQueryWithVariables(t *testing.T) {
    query := `
        query GetUser($id: Int!) {
            user(id: $id) {
                id
                name
            }
        }
    `
    
    variables := `{"id": 1}`
    
    result, err := g.ProcessRequest(ctx, query, variables)
    assert.NoError(t, err)
    // Assert result contains expected data
}
```

## Best Practices

### 1. Use Meaningful Names
```go
// ✅ Clear, descriptive names
g.RegisterQuery(ctx, "userById", GetUserByID, "id")
g.RegisterMutation(ctx, "createBlogPost", CreateBlogPost, "input")

// ❌ Generic, unclear names
g.RegisterQuery(ctx, "get", GetSomething, "id")
g.RegisterMutation(ctx, "do", DoSomething, "data")
```

### 2. Validate Input Early
```go
// ✅ Validate at the start of functions
func CreateUser(ctx context.Context, input UserInput) (*User, error) {
    if err := input.Validate(); err != nil {
        return nil, err
    }
    // Continue with business logic...
}

// ❌ Skip validation
func CreateUser(ctx context.Context, input UserInput) (*User, error) {
    // Directly process without validation
    return &User{Name: input.Name}, nil
}
```

### 3. Return Appropriate Types
```go
// ✅ Pointer for nullable results
func GetUser(ctx context.Context, id int) (*User, error) {
    // Can return nil if user not found
}

// ✅ Slice for arrays
func GetUsers(ctx context.Context) ([]User, error) {
    // Returns empty slice if no users
}

// ❌ Value type that can't be null
func GetUser(ctx context.Context, id int) (User, error) {
    // Can't represent "user not found" naturally
}
```

### 4. Handle Context Properly
```go
// ✅ Always accept context as first parameter
func GetUser(ctx context.Context, id int) (*User, error) {
    return database.GetUserWithContext(ctx, id)
}

// ❌ No context support
func GetUser(id int) (*User, error) {
    // Can't be cancelled or carry metadata
}
```

## Next Steps

- **[Function Patterns](FUNCTION_PATTERNS.md)** - Advanced parameter and return patterns
- **[Type System Guide](TYPE_SYSTEM.md)** - Complex types, interfaces, and unions
- **[Custom Scalars](CUSTOM_SCALARS.md)** - Creating custom data types with validation