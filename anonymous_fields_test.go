package quickgraph

import (
	"context"
	"encoding/json"
	"testing"
)

// Test types for anonymous field support
type PaginationInput struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type SearchInput struct {
	PaginationInput          // anonymous embedding
	Query           string   `json:"query"`
	Tags            []string `json:"tags"`
}

type FilterOptions struct {
	MinPrice *float64 `json:"minPrice"`
	MaxPrice *float64 `json:"maxPrice"`
}

type AdvancedSearchInput struct {
	PaginationInput        // anonymous embedding
	*FilterOptions         // anonymous pointer embedding
	Query           string `json:"query"`
	SortBy          string `json:"sortBy"`
}

type SearchResult struct {
	ID    string
	Title string
	Score float64
}

func TestAnonymousFieldsInStructParam(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Register a function that uses a struct with anonymous fields
	g.RegisterQuery(ctx, "Search", func(input SearchInput) []SearchResult {
		// Verify we can access fields from the embedded struct
		results := []SearchResult{
			{ID: "1", Title: "Result 1 for " + input.Query, Score: 0.9},
			{ID: "2", Title: "Result 2 for " + input.Query, Score: 0.8},
		}

		// Apply pagination from embedded fields
		start := input.Offset
		end := input.Offset + input.Limit
		if end > len(results) {
			end = len(results)
		}
		if start > len(results) {
			start = len(results)
		}

		return results[start:end]
	})

	// Test query with fields from both the main struct and embedded struct
	query := `{
		Search(query: "test", limit: 1, offset: 0, tags: ["tag1", "tag2"]) {
			ID
			Title
			Score
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	if err != nil {
		t.Fatalf("Failed to process request: %v", err)
	}

	// Parse and validate result
	var resultMap map[string]any
	if err := json.Unmarshal([]byte(result), &resultMap); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	data := resultMap["data"].(map[string]any)
	searchResults := data["Search"].([]any)

	if len(searchResults) != 1 {
		t.Errorf("Expected 1 result (due to limit), got %d", len(searchResults))
	}

	firstResult := searchResults[0].(map[string]any)
	if firstResult["Title"] != "Result 1 for test" {
		t.Errorf("Expected title 'Result 1 for test', got %v", firstResult["Title"])
	}
}

func TestAnonymousPointerFieldsInStructParam(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	// Register a function that uses a struct with anonymous pointer fields
	g.RegisterQuery(ctx, "AdvancedSearch", func(input AdvancedSearchInput) []SearchResult {
		results := []SearchResult{
			{ID: "1", Title: "Product A", Score: 100.0},
			{ID: "2", Title: "Product B", Score: 50.0},
			{ID: "3", Title: "Product C", Score: 150.0},
		}

		// Filter by price range if provided (from anonymous pointer field)
		if input.FilterOptions != nil {
			filtered := []SearchResult{}
			for _, r := range results {
				if input.MinPrice != nil && r.Score < *input.MinPrice {
					continue
				}
				if input.MaxPrice != nil && r.Score > *input.MaxPrice {
					continue
				}
				filtered = append(filtered, r)
			}
			results = filtered
		}

		// Apply pagination
		start := input.Offset
		end := input.Offset + input.Limit
		if end > len(results) {
			end = len(results)
		}
		if start > len(results) {
			start = len(results)
		}

		return results[start:end]
	})

	// Test query with optional fields from pointer embedding
	query := `{
		AdvancedSearch(
			query: "products",
			limit: 10,
			offset: 0,
			minPrice: 60.0,
			maxPrice: 120.0,
			sortBy: "price"
		) {
			ID
			Title
			Score
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	if err != nil {
		t.Fatalf("Failed to process request: %v", err)
	}

	// Parse and validate result
	var resultMap map[string]any
	if err := json.Unmarshal([]byte(result), &resultMap); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	data := resultMap["data"].(map[string]any)
	searchResults := data["AdvancedSearch"].([]any)

	// Should only have Product A (score 100) due to price filter
	if len(searchResults) != 1 {
		t.Errorf("Expected 1 filtered result, got %d", len(searchResults))
	}

	if len(searchResults) > 0 {
		firstResult := searchResults[0].(map[string]any)
		if firstResult["ID"] != "1" {
			t.Errorf("Expected Product A (ID: 1), got ID: %v", firstResult["ID"])
		}
	}
}

func TestRequiredFieldsInAnonymousEmbedding(t *testing.T) {
	ctx := context.Background()
	g := Graphy{}

	g.RegisterQuery(ctx, "SearchWithRequired", func(input SearchInput) string {
		return "OK"
	})

	// Test that required fields from embedded struct are enforced
	query := `{
		SearchWithRequired(query: "test") {
		}
	}`

	_, err := g.ProcessRequest(ctx, query, "")
	if err == nil {
		t.Error("Expected error for missing required fields, got none")
	}

	// Error should mention the missing required fields
	if err.Error() == "" || !contains(err.Error(), "limit") {
		t.Errorf("Expected error to mention missing 'limit' field, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
