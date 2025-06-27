# Quickstart Example

This is the complete, runnable example from the [Quickstart Guide](../../docs/QUICKSTART.md).

## Running the Example

```bash
# From this directory
go run .

# Or from the root
go run ./examples/quickstart
```

Then visit http://localhost:8080/graphql to explore the API with GraphQL Playground.

## Features Demonstrated

- **Queries**: `user`, `users`, `posts`, `myPosts`
- **Mutations**: `createPost` with input validation
- **Relationships**: Posts have authors, users have posts
- **Input Validation**: CreatePostInput validates all fields
- **Context Usage**: `myPosts` demonstrates context parameter
- **Built-in Scalars**: DateTime support via time.Time

## Example Queries

### Get all users with their posts
```graphql
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
```

### Get all posts with authors
```graphql
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
```

### Create a new post
```graphql
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

### Get a specific user
```graphql
{
  user(id: 1) {
    name
    email
    createdAt
    posts {
      title
      content
      price
    }
  }
}
```

## Code Structure

The example demonstrates:

1. **Domain Types**: Regular Go structs with `graphy` tags
2. **Queries**: Simple functions that return data
3. **Mutations**: Functions that modify data and return results
4. **Input Validation**: Validate() method on input types
5. **Relationships**: Methods on types become GraphQL fields
6. **Thread Safety**: Using sync.RWMutex for concurrent access

This is a minimal but complete example showing how to build a GraphQL API with go-quickgraph using only standard Go patterns.