package quickgraph

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test to verify that embedded types are correctly rendered as interfaces in GraphQL schema
func TestEmbeddedTypeBecomesInterface(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Define a base type that will be embedded
	type BaseEntity struct {
		ID        string
		CreatedAt string
		UpdatedAt string
	}

	// Define types that embed BaseEntity
	type User struct {
		BaseEntity
		Username string
		Email    string
	}

	type Product struct {
		BaseEntity
		Name  string
		Price float64
	}

	// Register functions that return these types
	g.RegisterQuery(ctx, "getUser", func(id string) *User {
		return &User{
			BaseEntity: BaseEntity{
				ID:        id,
				CreatedAt: "2024-01-01",
				UpdatedAt: "2024-01-02",
			},
			Username: "testuser",
			Email:    "test@example.com",
		}
	})

	g.RegisterQuery(ctx, "getProduct", func(id string) *Product {
		return &Product{
			BaseEntity: BaseEntity{
				ID:        id,
				CreatedAt: "2024-01-01",
				UpdatedAt: "2024-01-02",
			},
			Name:  "Test Product",
			Price: 99.99,
		}
	})

	// Get the schema
	schema := g.SchemaDefinition(ctx)

	// Verify that BaseEntity is rendered as an interface
	assert.Contains(t, schema, "interface BaseEntity {")

	// Verify that User implements BaseEntity
	assert.Contains(t, schema, "type User implements BaseEntity {")

	// Verify that Product implements BaseEntity
	assert.Contains(t, schema, "type Product implements BaseEntity {")

	// Verify that implementing types include all interface fields
	assert.Contains(t, schema, "type User implements BaseEntity {\n\tCreatedAt: String!\n\tEmail: String!\n\tID: String!\n\tUpdatedAt: String!\n\tUsername: String!\n}")
	assert.Contains(t, schema, "type Product implements BaseEntity {\n\tCreatedAt: String!\n\tID: String!\n\tName: String!\n\tPrice: Float!\n\tUpdatedAt: String!\n}")
}

// Test with multiple levels of embedding
func TestMultiLevelEmbeddingInterface(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Base type
	type Identifiable struct {
		ID string
	}

	// Mid-level type that embeds Identifiable
	type Timestamped struct {
		Identifiable
		CreatedAt string
		UpdatedAt string
	}

	// Concrete type that embeds Timestamped
	type Article struct {
		Timestamped
		Title   string
		Content string
	}

	g.RegisterQuery(ctx, "getArticle", func(id string) *Article {
		return &Article{
			Timestamped: Timestamped{
				Identifiable: Identifiable{ID: id},
				CreatedAt:    "2024-01-01",
				UpdatedAt:    "2024-01-02",
			},
			Title:   "Test Article",
			Content: "Test content",
		}
	})

	schema := g.SchemaDefinition(ctx)

	// Both Identifiable and Timestamped should be interfaces
	assert.Contains(t, schema, "interface Identifiable {")
	assert.Contains(t, schema, "interface Timestamped implements Identifiable {")

	// Article should implement both Identifiable and Timestamped
	assert.Contains(t, schema, "type Article implements Timestamped & Identifiable {")
}
