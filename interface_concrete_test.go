package quickgraph

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Helper function to run a query and get the result data
func runQueryGetData(t *testing.T, g *Graphy, ctx context.Context, query string) map[string]any {
	resultJSON, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)

	var result map[string]any
	err = json.Unmarshal([]byte(resultJSON), &result)
	assert.NoError(t, err)

	data, ok := result["data"].(map[string]any)
	assert.True(t, ok, "result should have 'data' field")
	return data
}

// Test interfaces with concrete types that have additional fields
func TestInterfaceWithConcreteTypeFields(t *testing.T) {
	ctx := context.Background()

	// Define concrete types with additional fields
	type Dog struct {
		Name    string `json:"name"`
		Age     int    `json:"age"`
		Breed   string `json:"breed"`
		GoodBoy bool   `json:"goodBoy"`
	}

	type Cat struct {
		Name   string `json:"name"`
		Age    int    `json:"age"`
		Color  string `json:"color"`
		Indoor bool   `json:"indoor"`
	}

	// Create test data
	dog := &Dog{
		Name:    "Buddy",
		Age:     3,
		Breed:   "Golden Retriever",
		GoodBoy: true,
	}

	cat := &Cat{
		Name:   "Whiskers",
		Age:    5,
		Color:  "Orange",
		Indoor: true,
	}

	// Function that returns an interface but actually returns concrete types
	getPet := func(ctx context.Context, petType string) any {
		if petType == "dog" {
			return dog
		} else if petType == "cat" {
			return cat
		}
		return nil
	}

	g := Graphy{}
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:              "getPet",
		Function:          getPet,
		ReturnAnyOverride: []any{Dog{}, Cat{}},
	})

	// Test 1: Query with inline fragment for Dog
	query1 := `{
		getPet(petType: "dog") {
			name
			age
			... on Dog {
				breed
				goodBoy
			}
		}
	}`

	result1 := runQueryGetData(t, &g, ctx, query1)
	assert.Equal(t, map[string]any{
		"getPet": map[string]any{
			"name":    "Buddy",
			"age":     float64(3),
			"breed":   "Golden Retriever",
			"goodBoy": true,
		},
	}, result1)

	// Test 2: Query with inline fragment for Cat
	query2 := `{
		getPet(petType: "cat") {
			name
			age
			... on Cat {
				color
				indoor
			}
		}
	}`

	result2 := runQueryGetData(t, &g, ctx, query2)
	assert.Equal(t, map[string]any{
		"getPet": map[string]any{
			"name":   "Whiskers",
			"age":    float64(5),
			"color":  "Orange",
			"indoor": true,
		},
	}, result2)

	// Test 3: Query without type-specific fields (should work for both)
	query3 := `{
		getPet(petType: "dog") {
			name
			age
		}
	}`

	result3 := runQueryGetData(t, &g, ctx, query3)
	assert.Equal(t, map[string]any{
		"getPet": map[string]any{
			"name": "Buddy",
			"age":  float64(3),
		},
	}, result3)

	// Test 4: Query with __typename to verify concrete type detection
	query4 := `{
		getPet(petType: "dog") {
			__typename
			name
			... on Dog {
				breed
			}
		}
	}`

	result4 := runQueryGetData(t, &g, ctx, query4)
	assert.Equal(t, map[string]any{
		"getPet": map[string]any{
			"__typename": "Dog",
			"name":       "Buddy",
			"breed":      "Golden Retriever",
		},
	}, result4)
}

// Test with embedded structs (more like GraphQL interfaces)
func TestEmbeddedStructInterface(t *testing.T) {
	ctx := context.Background()

	// Base "interface" as embedded struct
	type Character struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	type Human struct {
		Character
		Height float64 `json:"height"`
	}

	type Alien struct {
		Character
		Planet    string `json:"planet"`
		Tentacles int    `json:"tentacles"`
	}

	// Function returns any but actual types are concrete
	getCharacter := func(ctx context.Context, id string) any {
		if id == "human1" {
			return &Human{
				Character: Character{ID: "human1", Name: "John"},
				Height:    1.8,
			}
		}
		return &Alien{
			Character: Character{ID: "alien1", Name: "Zorg"},
			Planet:    "Mars",
			Tentacles: 8,
		}
	}

	g := Graphy{}
	g.RegisterFunction(ctx, FunctionDefinition{
		Name:              "getCharacter",
		Function:          getCharacter,
		ReturnAnyOverride: []any{Human{}, Alien{}},
	})

	// Test querying with fragments
	query := `{
		getCharacter(id: "human1") {
			id
			name
			... on Human {
				height
			}
		}
	}`

	result := runQueryGetData(t, &g, ctx, query)
	assert.Equal(t, map[string]any{
		"getCharacter": map[string]any{
			"id":     "human1",
			"name":   "John",
			"height": 1.8,
		},
	}, result)

	// Test with Alien
	query2 := `{
		getCharacter(id: "alien1") {
			id
			name
			... on Alien {
				planet
				tentacles
			}
		}
	}`

	result2 := runQueryGetData(t, &g, ctx, query2)
	assert.Equal(t, map[string]any{
		"getCharacter": map[string]any{
			"id":        "alien1",
			"name":      "Zorg",
			"planet":    "Mars",
			"tentacles": float64(8),
		},
	}, result2)
}
