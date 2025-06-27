package quickgraph

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
	"time"
)

// Test types for limit testing
type LimitTestNode struct {
	ID       string           `graphy:"id"`
	Name     string           `graphy:"name"`
	Children []*LimitTestNode `graphy:"children"`
	Parent   *LimitTestNode   `graphy:"parent"`
}

func createDeepNode(depth int) *LimitTestNode {
	if depth <= 0 {
		return &LimitTestNode{ID: "leaf", Name: "Leaf Node"}
	}
	return &LimitTestNode{
		ID:       fmt.Sprintf("node-%d", depth),
		Name:     fmt.Sprintf("Node %d", depth),
		Children: []*LimitTestNode{createDeepNode(depth - 1)},
	}
}

func createWideNode(width int) *LimitTestNode {
	children := make([]*LimitTestNode, width)
	for i := 0; i < width; i++ {
		children[i] = &LimitTestNode{
			ID:   fmt.Sprintf("child-%d", i),
			Name: fmt.Sprintf("Child %d", i),
		}
	}
	return &LimitTestNode{
		ID:       "root",
		Name:     "Root Node",
		Children: children,
	}
}

func TestQueryLimits_MaxDepth(t *testing.T) {
	ctx := context.Background()
	g := Graphy{
		QueryLimits: &QueryLimits{
			MaxDepth: 3,
		},
	}

	// Register a function that returns a deep structure
	g.RegisterQuery(ctx, "deepNode", func(ctx context.Context) *LimitTestNode {
		return createDeepNode(10) // Create a structure 10 levels deep
	})

	// Query that exceeds depth limit
	query := `{
		deepNode {
			id
			children {
				id
				children {
					id
					children {
						id
						children {
							id
						}
					}
				}
			}
		}
	}`

	_, err := g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed depth")
}

func TestQueryLimits_MaxFields(t *testing.T) {
	ctx := context.Background()
	g := Graphy{
		QueryLimits: &QueryLimits{
			MaxFields: 2,
		},
	}

	g.RegisterQuery(ctx, "node", func(ctx context.Context) *LimitTestNode {
		return &LimitTestNode{ID: "1", Name: "Test"}
	})

	// Query that requests too many fields
	query := `{
		node {
			id
			name
			children
		}
	}`

	_, err := g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed fields")
}

func TestQueryLimits_MaxAliases(t *testing.T) {
	ctx := context.Background()
	g := Graphy{
		QueryLimits: &QueryLimits{
			MaxAliases: 2,
		},
	}

	g.RegisterQuery(ctx, "node", func(ctx context.Context) *LimitTestNode {
		return &LimitTestNode{ID: "1", Name: "Test"}
	})

	// Query with too many aliases
	query := `{
		a: node { id }
		b: node { id }
		c: node { id }
	}`

	_, err := g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed aliases")
}

func TestQueryLimits_MaxArraySize(t *testing.T) {
	ctx := context.Background()
	g := Graphy{
		QueryLimits: &QueryLimits{
			MaxArraySize: 5,
		},
	}

	g.RegisterQuery(ctx, "wideNode", func(ctx context.Context) *LimitTestNode {
		return createWideNode(10) // Create node with 10 children
	})

	query := `{
		wideNode {
			children {
				id
			}
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)

	// Check that array was truncated to 5 elements
	assert.Contains(t, result, `"children":[`)
	// Count the number of "id" fields in the result
	idCount := strings.Count(result, `"id":`)
	assert.Equal(t, 5, idCount) // Should only have 5 children
}

func TestQueryLimits_MaxConcurrentResolvers(t *testing.T) {
	ctx := context.Background()
	g := Graphy{
		QueryLimits: &QueryLimits{
			MaxConcurrentResolvers: 2,
		},
	}

	slowFunc := func(ctx context.Context) *LimitTestNode {
		// Simulate slow operation
		time.Sleep(50 * time.Millisecond)
		return &LimitTestNode{ID: "1", Name: "Test"}
	}

	g.RegisterQuery(ctx, "slow", slowFunc)

	// Query with multiple aliases to trigger parallel execution
	query := `{
		a: slow { id }
		b: slow { id }
		c: slow { id }
		d: slow { id }
	}`

	start := time.Now()
	result, err := g.ProcessRequest(ctx, query, "")
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Contains(t, result, `"a":`)
	assert.Contains(t, result, `"b":`)
	assert.Contains(t, result, `"c":`)
	assert.Contains(t, result, `"d":`)

	// With max 2 concurrent, 4 queries of 50ms each should take at least 100ms
	// (2 batches of 2 parallel queries)
	assert.True(t, duration >= 100*time.Millisecond, "Expected duration >= 100ms, got %v", duration)
}

func TestQueryLimits_NoLimits(t *testing.T) {
	ctx := context.Background()
	g := Graphy{} // No limits set

	g.RegisterQuery(ctx, "deepNode", func(ctx context.Context) *LimitTestNode {
		return createDeepNode(10)
	})

	// Deep query should work without limits
	query := `{
		deepNode {
			id
			children {
				id
				children {
					id
					children {
						id
						children {
							id
						}
					}
				}
			}
		}
	}`

	result, err := g.ProcessRequest(ctx, query, "")
	assert.NoError(t, err)
	assert.Contains(t, result, `"id":"node-10"`)
}

func TestQueryLimits_DepthWithFragments(t *testing.T) {
	ctx := context.Background()
	g := Graphy{
		QueryLimits: &QueryLimits{
			MaxDepth: 3,
		},
	}

	g.RegisterQuery(ctx, "node", func(ctx context.Context) *LimitTestNode {
		return createDeepNode(10)
	})

	// Query using fragments that would exceed depth
	query := `{
		node {
			...nodeFields
		}
	}
	
	fragment nodeFields on LimitTestNode {
		id
		children {
			id
			children {
				id
				children {
					id
				}
			}
		}
	}`

	_, err := g.ProcessRequest(ctx, query, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum allowed depth")
}

func TestDefaultComplexityScorer(t *testing.T) {
	scorer := DefaultComplexityScorer{}

	// Test field scoring
	assert.Equal(t, 1, scorer.ScoreField("User", "name"))
	assert.Equal(t, 1, scorer.ScoreField("Post", "title"))

	// Test list scoring
	assert.Equal(t, 10, scorer.ScoreList(1, 10))
	assert.Equal(t, 50, scorer.ScoreList(5, 10))
	assert.Equal(t, 10, scorer.ScoreList(1, 0))  // Default size
	assert.Equal(t, 10, scorer.ScoreList(1, -1)) // Negative size
}
