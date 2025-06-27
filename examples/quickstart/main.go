package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gburgyan/go-quickgraph"
)

// Domain types - just regular Go structs
type User struct {
	ID        int       `graphy:"id"`
	Email     string    `graphy:"email"`
	Name      string    `graphy:"name"`
	CreatedAt time.Time `graphy:"createdAt"`
}

type Post struct {
	ID        int       `graphy:"id"`
	Title     string    `graphy:"title"`
	Content   string    `graphy:"content"`
	AuthorID  int       `graphy:"authorId"`
	Price     float64   `graphy:"price"` // Price in dollars
	CreatedAt time.Time `graphy:"createdAt"`
}

// Simple in-memory storage
var (
	users = []User{
		{ID: 1, Email: "alice@example.com", Name: "Alice", CreatedAt: time.Now()},
		{ID: 2, Email: "bob@example.com", Name: "Bob", CreatedAt: time.Now()},
	}
	posts = []Post{
		{ID: 1, Title: "Hello World", Content: "My first post", AuthorID: 1, Price: 9.99, CreatedAt: time.Now()},
		{ID: 2, Title: "GraphQL is Great", Content: "Learning about GraphQL", AuthorID: 2, Price: 14.99, CreatedAt: time.Now()},
	}
	mu sync.RWMutex
)

// Queries - just functions that return data
func GetUser(id int) (*User, error) {
	mu.RLock()
	defer mu.RUnlock()

	for _, user := range users {
		if user.ID == id {
			return &user, nil
		}
	}
	return nil, errors.New("user not found")
}

func GetUsers() []User {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]User, len(users))
	copy(result, users)
	return result
}

func GetPosts() []Post {
	mu.RLock()
	defer mu.RUnlock()

	result := make([]Post, len(posts))
	copy(result, posts)
	return result
}

// Input type with validation
type CreatePostInput struct {
	Title    string  `graphy:"title"`
	Content  string  `graphy:"content"`
	AuthorID int     `graphy:"authorId"`
	Price    float64 `graphy:"price"`
}

func (input CreatePostInput) Validate() error {
	if input.Title == "" {
		return errors.New("title is required")
	}
	if len(input.Title) > 100 {
		return errors.New("title must be 100 characters or less")
	}
	if input.Content == "" {
		return errors.New("content is required")
	}
	if input.Price < 0 {
		return errors.New("price cannot be negative")
	}
	if input.Price > 1000 {
		return errors.New("price cannot exceed $1000")
	}
	return nil
}

// Mutations - functions that modify data
func CreatePost(input CreatePostInput) (*Post, error) {
	mu.Lock()
	defer mu.Unlock()

	// Validate author exists
	validAuthor := false
	for _, user := range users {
		if user.ID == input.AuthorID {
			validAuthor = true
			break
		}
	}
	if !validAuthor {
		return nil, errors.New("invalid author")
	}

	post := Post{
		ID:        len(posts) + 1,
		Title:     input.Title,
		Content:   input.Content,
		AuthorID:  input.AuthorID,
		Price:     input.Price,
		CreatedAt: time.Now(),
	}
	posts = append(posts, post)
	return &post, nil
}

// Relationships - methods on types become fields
func (p Post) Author() (*User, error) {
	return GetUser(p.AuthorID)
}

func (u User) Posts() []Post {
	mu.RLock()
	defer mu.RUnlock()

	var userPosts []Post
	for _, post := range posts {
		if post.AuthorID == u.ID {
			userPosts = append(userPosts, post)
		}
	}
	return userPosts
}

// Demonstrate context usage for auth
func GetMyPosts(ctx context.Context) ([]Post, error) {
	// In a real app, you'd get the user from context (set by middleware)
	// For demo, we'll just use the first user
	user := users[0]
	return user.Posts(), nil
}

func main() {
	ctx := context.Background()
	g := quickgraph.Graphy{}

	// Register queries
	g.RegisterQuery(ctx, "user", GetUser, "id")
	g.RegisterQuery(ctx, "users", GetUsers)
	g.RegisterQuery(ctx, "posts", GetPosts)
	g.RegisterQuery(ctx, "myPosts", GetMyPosts)

	// Register mutations
	g.RegisterMutation(ctx, "createPost", CreatePost, "input")

	// Enable GraphQL Playground
	g.EnableIntrospection(ctx)

	// Configure CORS for GraphQL Playground
	g.CORSSettings = quickgraph.DefaultCORSSettings()

	// Start server
	http.Handle("/graphql", g.HttpHandler())
	fmt.Println("🚀 GraphQL server running at http://localhost:8080/graphql")
	fmt.Println("\nTry these queries in the GraphQL Playground:")

	fmt.Println("# Get all users")
	fmt.Println(`{
  users {
    id
    name
    email
    posts {
      title
      price
    }
  }
}`)

	fmt.Println("\n# Get all posts with authors")
	fmt.Println(`{
  posts {
    title
    content
    price
    author {
      name
      email
    }
  }
}`)

	fmt.Println("\n# Create a new post")
	fmt.Println(`mutation {
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
}`)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
