package quickgraph

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test queryLimitContext creation and basic functionality
func TestNewQueryLimitContext(t *testing.T) {
	tests := []struct {
		name   string
		limits *QueryLimits
		isNil  bool
	}{
		{
			name:   "nil limits returns nil context",
			limits: nil,
			isNil:  true,
		},
		{
			name:   "empty limits returns valid context",
			limits: &QueryLimits{},
			isNil:  false,
		},
		{
			name: "limits with values returns valid context",
			limits: &QueryLimits{
				MaxDepth:               10,
				MaxFields:              50,
				MaxConcurrentResolvers: 5,
			},
			isNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newQueryLimitContext(tt.limits)
			if tt.isNil {
				assert.Nil(t, ctx)
			} else {
				assert.NotNil(t, ctx)
				assert.Equal(t, tt.limits, ctx.limits)
				assert.NotNil(t, ctx.fieldCount)

				// Check resolver limiter creation
				if tt.limits != nil && tt.limits.MaxConcurrentResolvers > 0 {
					assert.NotNil(t, ctx.resolverLimiter)
					assert.Equal(t, tt.limits.MaxConcurrentResolvers, cap(ctx.resolverLimiter))
				} else {
					assert.Nil(t, ctx.resolverLimiter)
				}
			}
		})
	}
}

// Test depth checking
func TestQueryLimitContext_CheckDepth(t *testing.T) {
	tests := []struct {
		name      string
		limits    *QueryLimits
		depth     int
		wantError bool
	}{
		{
			name:      "nil context allows any depth",
			limits:    nil,
			depth:     1000,
			wantError: false,
		},
		{
			name:      "zero max depth allows any depth",
			limits:    &QueryLimits{MaxDepth: 0},
			depth:     1000,
			wantError: false,
		},
		{
			name:      "depth within limit",
			limits:    &QueryLimits{MaxDepth: 5},
			depth:     3,
			wantError: false,
		},
		{
			name:      "depth at limit",
			limits:    &QueryLimits{MaxDepth: 5},
			depth:     5,
			wantError: false,
		},
		{
			name:      "depth exceeds limit",
			limits:    &QueryLimits{MaxDepth: 5},
			depth:     6,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newQueryLimitContext(tt.limits)
			err := ctx.checkDepth(tt.depth)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "exceeds maximum allowed depth")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test field count checking
func TestQueryLimitContext_CheckFieldCount(t *testing.T) {
	tests := []struct {
		name      string
		limits    *QueryLimits
		depth     int
		count     int
		wantError bool
	}{
		{
			name:      "nil context allows any field count",
			limits:    nil,
			depth:     1,
			count:     1000,
			wantError: false,
		},
		{
			name:      "zero max fields allows any count",
			limits:    &QueryLimits{MaxFields: 0},
			depth:     1,
			count:     1000,
			wantError: false,
		},
		{
			name:      "field count within limit",
			limits:    &QueryLimits{MaxFields: 10},
			depth:     1,
			count:     5,
			wantError: false,
		},
		{
			name:      "field count at limit",
			limits:    &QueryLimits{MaxFields: 10},
			depth:     1,
			count:     10,
			wantError: false,
		},
		{
			name:      "field count exceeds limit",
			limits:    &QueryLimits{MaxFields: 10},
			depth:     1,
			count:     11,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newQueryLimitContext(tt.limits)
			err := ctx.checkFieldCount(tt.depth, tt.count)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "exceeds maximum allowed fields")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test alias count checking
func TestQueryLimitContext_CheckAliasCount(t *testing.T) {
	tests := []struct {
		name      string
		limits    *QueryLimits
		count     int
		wantError bool
	}{
		{
			name:      "nil context allows any alias count",
			limits:    nil,
			count:     1000,
			wantError: false,
		},
		{
			name:      "zero max aliases allows any count",
			limits:    &QueryLimits{MaxAliases: 0},
			count:     1000,
			wantError: false,
		},
		{
			name:      "alias count within limit",
			limits:    &QueryLimits{MaxAliases: 20},
			count:     10,
			wantError: false,
		},
		{
			name:      "alias count at limit",
			limits:    &QueryLimits{MaxAliases: 20},
			count:     20,
			wantError: false,
		},
		{
			name:      "alias count exceeds limit",
			limits:    &QueryLimits{MaxAliases: 20},
			count:     21,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newQueryLimitContext(tt.limits)
			err := ctx.checkAliasCount(tt.count)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "exceeds maximum allowed aliases")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test complexity checking
func TestQueryLimitContext_CheckComplexity(t *testing.T) {
	tests := []struct {
		name           string
		limits         *QueryLimits
		initialCost    int
		additionalCost int
		wantError      bool
	}{
		{
			name:           "nil context allows any complexity",
			limits:         nil,
			additionalCost: 1000,
			wantError:      false,
		},
		{
			name:           "zero max complexity allows any cost",
			limits:         &QueryLimits{MaxComplexity: 0},
			additionalCost: 1000,
			wantError:      false,
		},
		{
			name:           "complexity within limit",
			limits:         &QueryLimits{MaxComplexity: 100},
			additionalCost: 50,
			wantError:      false,
		},
		{
			name:           "complexity at limit",
			limits:         &QueryLimits{MaxComplexity: 100},
			additionalCost: 100,
			wantError:      false,
		},
		{
			name:           "complexity exceeds limit",
			limits:         &QueryLimits{MaxComplexity: 100},
			additionalCost: 101,
			wantError:      true,
		},
		{
			name:           "cumulative complexity",
			limits:         &QueryLimits{MaxComplexity: 100},
			initialCost:    80,
			additionalCost: 21,
			wantError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newQueryLimitContext(tt.limits)

			// Add initial cost if specified
			if tt.initialCost > 0 && ctx != nil {
				err := ctx.checkComplexity(tt.initialCost)
				assert.NoError(t, err)
			}

			err := ctx.checkComplexity(tt.additionalCost)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "exceeds maximum allowed complexity")
			} else {
				assert.NoError(t, err)
				// Verify complexity was updated
				if ctx != nil && ctx.limits.MaxComplexity > 0 {
					assert.Equal(t, tt.initialCost+tt.additionalCost, ctx.complexity)
				}
			}
		})
	}
}

// Test array size checking
func TestQueryLimitContext_CheckArraySize(t *testing.T) {
	tests := []struct {
		name      string
		limits    *QueryLimits
		size      int
		wantError bool
	}{
		{
			name:      "nil context allows any array size",
			limits:    nil,
			size:      10000,
			wantError: false,
		},
		{
			name:      "zero max array size allows any size",
			limits:    &QueryLimits{MaxArraySize: 0},
			size:      10000,
			wantError: false,
		},
		{
			name:      "array size within limit",
			limits:    &QueryLimits{MaxArraySize: 1000},
			size:      500,
			wantError: false,
		},
		{
			name:      "array size at limit",
			limits:    &QueryLimits{MaxArraySize: 1000},
			size:      1000,
			wantError: false,
		},
		{
			name:      "array size exceeds limit",
			limits:    &QueryLimits{MaxArraySize: 1000},
			size:      1001,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newQueryLimitContext(tt.limits)
			err := ctx.checkArraySize(tt.size)

			if tt.wantError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "exceeds maximum allowed size")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test resolver limiter
func TestQueryLimitContext_ResolverLimiter(t *testing.T) {
	t.Run("nil context", func(t *testing.T) {
		var ctx *queryLimitContext
		// Should not panic
		ctx.acquireResolver()
		ctx.releaseResolver()
	})

	t.Run("no limiter configured", func(t *testing.T) {
		ctx := newQueryLimitContext(&QueryLimits{})
		// Should not panic
		ctx.acquireResolver()
		ctx.releaseResolver()
	})

	t.Run("limiter configured", func(t *testing.T) {
		ctx := newQueryLimitContext(&QueryLimits{
			MaxConcurrentResolvers: 2,
		})

		// Should be able to acquire 2
		ctx.acquireResolver()
		ctx.acquireResolver()

		// Third should block, so test with goroutine
		acquired := make(chan bool, 1)
		go func() {
			ctx.acquireResolver()
			acquired <- true
		}()

		// Should timeout waiting
		select {
		case <-acquired:
			t.Fatal("Should not have acquired third resolver")
		case <-time.After(100 * time.Millisecond):
			// Expected
		}

		// Release one
		ctx.releaseResolver()

		// Now should acquire
		select {
		case <-acquired:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Should have acquired resolver after release")
		}
	})
}

// Test concurrent resolver guard
func TestConcurrentResolverGuard(t *testing.T) {
	t.Run("nil guard", func(t *testing.T) {
		guard := newConcurrentResolverGuard(0)
		assert.Nil(t, guard)

		// Should not panic
		err := guard.acquire()
		assert.NoError(t, err)
		guard.release()
	})

	t.Run("negative max", func(t *testing.T) {
		guard := newConcurrentResolverGuard(-1)
		assert.Nil(t, guard)
	})

	t.Run("basic functionality", func(t *testing.T) {
		guard := newConcurrentResolverGuard(3)
		assert.NotNil(t, guard)

		// Acquire 3
		for i := 0; i < 3; i++ {
			err := guard.acquire()
			assert.NoError(t, err)
		}

		// 4th should fail
		err := guard.acquire()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "maximum concurrent resolvers")

		// Release one
		guard.release()

		// Should now succeed
		err = guard.acquire()
		assert.NoError(t, err)
	})

	t.Run("concurrent operations", func(t *testing.T) {
		guard := newConcurrentResolverGuard(10)

		var wg sync.WaitGroup
		errors := make(chan error, 20)

		// Try to acquire 20 slots from 20 goroutines
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := guard.acquire()
				if err != nil {
					errors <- err
				} else {
					// Hold for a bit then release
					time.Sleep(10 * time.Millisecond)
					guard.release()
				}
			}()
		}

		wg.Wait()
		close(errors)

		// Should have exactly 10 errors
		errorCount := 0
		for err := range errors {
			errorCount++
			assert.Contains(t, err.Error(), "maximum concurrent resolvers")
		}
		assert.Equal(t, 10, errorCount)

		// All should be released now
		assert.Equal(t, int32(0), guard.activeCount)
	})
}

// Test custom complexity scorer
type customComplexityScorer struct {
	fieldScores map[string]int
}

func (c customComplexityScorer) ScoreField(typeName, fieldName string) int {
	key := fmt.Sprintf("%s.%s", typeName, fieldName)
	if score, ok := c.fieldScores[key]; ok {
		return score
	}
	return 1
}

func (c customComplexityScorer) ScoreList(itemScore int, estimatedSize int) int {
	// Custom formula: square the item score and multiply by size
	return itemScore * itemScore * estimatedSize
}

func TestCustomComplexityScorer(t *testing.T) {
	scorer := customComplexityScorer{
		fieldScores: map[string]int{
			"User.friends":    5,
			"User.posts":      10,
			"Post.comments":   3,
			"Comment.replies": 2,
		},
	}

	// Test field scoring
	assert.Equal(t, 5, scorer.ScoreField("User", "friends"))
	assert.Equal(t, 10, scorer.ScoreField("User", "posts"))
	assert.Equal(t, 1, scorer.ScoreField("User", "name")) // default

	// Test list scoring
	assert.Equal(t, 25, scorer.ScoreList(5, 1))   // 5^2 * 1
	assert.Equal(t, 400, scorer.ScoreList(10, 4)) // 10^2 * 4
}

// Test edge cases and error messages
func TestQueryLimitContext_ErrorMessages(t *testing.T) {
	t.Run("depth error message", func(t *testing.T) {
		ctx := newQueryLimitContext(&QueryLimits{MaxDepth: 5})
		err := ctx.checkDepth(7)
		assert.Equal(t, "query depth 7 exceeds maximum allowed depth of 5", err.Error())
	})

	t.Run("field count error message", func(t *testing.T) {
		ctx := newQueryLimitContext(&QueryLimits{MaxFields: 10})
		err := ctx.checkFieldCount(3, 15)
		assert.Equal(t, "field count 15 at depth 3 exceeds maximum allowed fields of 10", err.Error())
	})

	t.Run("alias count error message", func(t *testing.T) {
		ctx := newQueryLimitContext(&QueryLimits{MaxAliases: 5})
		err := ctx.checkAliasCount(8)
		assert.Equal(t, "alias count 8 exceeds maximum allowed aliases of 5", err.Error())
	})

	t.Run("complexity error message", func(t *testing.T) {
		ctx := newQueryLimitContext(&QueryLimits{MaxComplexity: 100})
		err := ctx.checkComplexity(150)
		assert.Equal(t, "query complexity 150 exceeds maximum allowed complexity of 100", err.Error())
	})

	t.Run("array size error message", func(t *testing.T) {
		ctx := newQueryLimitContext(&QueryLimits{MaxArraySize: 1000})
		err := ctx.checkArraySize(2000)
		assert.Equal(t, "array size 2000 exceeds maximum allowed size of 1000", err.Error())
	})

	t.Run("concurrent resolver error message", func(t *testing.T) {
		guard := newConcurrentResolverGuard(5)
		// Fill up all slots
		for i := 0; i < 5; i++ {
			err := guard.acquire()
			assert.NoError(t, err)
		}
		err := guard.acquire()
		assert.Equal(t, "maximum concurrent resolvers (5) reached", err.Error())
	})
}
