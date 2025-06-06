package quickgraph

import (
	"fmt"
	"sync/atomic"
)

// QueryLimits defines optional limits to prevent DoS attacks.
// All limits are optional - zero values mean unlimited.
type QueryLimits struct {
	// MaxDepth limits nesting depth of queries (0 = unlimited)
	MaxDepth int

	// MaxComplexity limits total "cost" of a query (0 = unlimited)
	MaxComplexity int

	// MaxFields limits fields selected at any level (0 = unlimited)
	MaxFields int

	// MaxAliases limits number of aliases in a query (0 = unlimited)
	MaxAliases int

	// MaxArraySize limits size of arrays returned (0 = unlimited)
	MaxArraySize int

	// MaxConcurrentResolvers limits parallel goroutines (0 = unlimited)
	MaxConcurrentResolvers int

	// ComplexityScorer allows custom complexity calculation
	ComplexityScorer ComplexityScorer
}

// MemoryLimits defines optional limits to prevent memory exhaustion attacks.
// All limits are optional - zero values mean unlimited.
type MemoryLimits struct {
	// MaxRequestBodySize limits the size of HTTP request bodies in bytes (0 = unlimited)
	MaxRequestBodySize int64

	// MaxVariableSize limits the size of variable JSON payloads in bytes (0 = unlimited)
	MaxVariableSize int64

	// SubscriptionBufferSize sets the buffer size for subscription channels (0 = unbuffered)
	SubscriptionBufferSize int

	// MaxWebSocketConnections limits concurrent WebSocket connections (0 = unlimited)
	// This is customer-implemented. Left as a placeholder for customer use.
	MaxWebSocketConnections int

	// MaxSubscriptionsPerConnection limits subscriptions per WebSocket connection (0 = unlimited)
	MaxSubscriptionsPerConnection int
}

// CORSSettings defines CORS configuration for HTTP responses.
// Nil CORSSettings means no CORS headers will be added.
type CORSSettings struct {
	// AllowedOrigins specifies allowed origins for CORS requests
	// Use "*" to allow all origins, or specify exact origins like "https://example.com"
	// If empty and CORSSettings is not nil, defaults to "*"
	AllowedOrigins []string

	// AllowedMethods specifies allowed HTTP methods for CORS requests
	// If empty and CORSSettings is not nil, defaults to ["GET", "POST", "OPTIONS"]
	AllowedMethods []string

	// AllowedHeaders specifies allowed request headers for CORS requests
	// If empty and CORSSettings is not nil, defaults to ["Content-Type", "Authorization"]
	AllowedHeaders []string

	// ExposedHeaders specifies response headers that browsers can access
	// Optional - if empty, no Access-Control-Expose-Headers header is set
	ExposedHeaders []string

	// AllowCredentials indicates whether credentials are allowed in CORS requests
	// Defaults to false
	AllowCredentials bool

	// MaxAge specifies how long browsers can cache preflight responses in seconds
	// If 0 and CORSSettings is not nil, defaults to 86400 (24 hours)
	MaxAge int

	// EnableForAllResponses determines whether CORS headers are added to all responses
	// If true, CORS headers are added to GET, POST, and other responses
	// If false, CORS headers are only added to OPTIONS responses
	// Defaults to false
	EnableForAllResponses bool
}

// ComplexityScorer allows custom complexity scoring for queries
type ComplexityScorer interface {
	// ScoreField returns the complexity score for accessing a field
	ScoreField(typeName, fieldName string) int

	// ScoreList returns the complexity score for a list field
	ScoreList(itemScore int, estimatedSize int) int
}

// DefaultComplexityScorer provides basic complexity scoring
type DefaultComplexityScorer struct{}

func (d DefaultComplexityScorer) ScoreField(typeName, fieldName string) int {
	// Default: each field costs 1
	return 1
}

func (d DefaultComplexityScorer) ScoreList(itemScore int, estimatedSize int) int {
	// Default: list cost is item cost * estimated size
	if estimatedSize <= 0 {
		estimatedSize = 10 // Conservative default
	}
	return itemScore * estimatedSize
}

// queryLimitContext tracks limit usage during query execution
type queryLimitContext struct {
	limits          *QueryLimits
	depth           int
	complexity      int
	fieldCount      map[int]int // tracks fields per depth level
	aliasCount      int
	resolverLimiter chan struct{} // semaphore for concurrent resolvers
}

func newQueryLimitContext(limits *QueryLimits) *queryLimitContext {
	if limits == nil {
		return nil
	}

	ctx := &queryLimitContext{
		limits:     limits,
		fieldCount: make(map[int]int),
	}

	// Initialize resolver limiter if needed
	if limits.MaxConcurrentResolvers > 0 {
		ctx.resolverLimiter = make(chan struct{}, limits.MaxConcurrentResolvers)
	}

	return ctx
}

// checkDepth validates depth limit hasn't been exceeded
func (q *queryLimitContext) checkDepth(depth int) error {
	if q == nil || q.limits.MaxDepth <= 0 {
		return nil
	}
	if depth > q.limits.MaxDepth {
		return fmt.Errorf("query depth %d exceeds maximum allowed depth of %d", depth, q.limits.MaxDepth)
	}
	return nil
}

// checkFieldCount validates field count at current depth
func (q *queryLimitContext) checkFieldCount(depth int, count int) error {
	if q == nil || q.limits.MaxFields <= 0 {
		return nil
	}
	if count > q.limits.MaxFields {
		return fmt.Errorf("field count %d at depth %d exceeds maximum allowed fields of %d", count, depth, q.limits.MaxFields)
	}
	return nil
}

// checkAliasCount validates total alias count
func (q *queryLimitContext) checkAliasCount(count int) error {
	if q == nil || q.limits.MaxAliases <= 0 {
		return nil
	}
	if count > q.limits.MaxAliases {
		return fmt.Errorf("alias count %d exceeds maximum allowed aliases of %d", count, q.limits.MaxAliases)
	}
	return nil
}

// checkComplexity validates total query complexity
func (q *queryLimitContext) checkComplexity(additionalCost int) error {
	if q == nil || q.limits.MaxComplexity <= 0 {
		return nil
	}
	newComplexity := q.complexity + additionalCost
	if newComplexity > q.limits.MaxComplexity {
		return fmt.Errorf("query complexity %d exceeds maximum allowed complexity of %d", newComplexity, q.limits.MaxComplexity)
	}
	q.complexity = newComplexity
	return nil
}

// acquireResolver gets permission to spawn a resolver goroutine
func (q *queryLimitContext) acquireResolver() {
	if q == nil || q.resolverLimiter == nil {
		return
	}
	q.resolverLimiter <- struct{}{}
}

// releaseResolver releases a resolver goroutine slot
func (q *queryLimitContext) releaseResolver() {
	if q == nil || q.resolverLimiter == nil {
		return
	}
	<-q.resolverLimiter
}

// checkArraySize validates array size during execution
func (q *queryLimitContext) checkArraySize(size int) error {
	if q == nil || q.limits.MaxArraySize <= 0 {
		return nil
	}
	if size > q.limits.MaxArraySize {
		return fmt.Errorf("array size %d exceeds maximum allowed size of %d", size, q.limits.MaxArraySize)
	}
	return nil
}

// concurrentResolverGuard manages concurrent resolver limits
type concurrentResolverGuard struct {
	activeCount int32
	maxCount    int32
}

func newConcurrentResolverGuard(max int) *concurrentResolverGuard {
	if max <= 0 {
		return nil
	}
	return &concurrentResolverGuard{maxCount: int32(max)}
}

func (c *concurrentResolverGuard) acquire() error {
	if c == nil {
		return nil
	}

	// Try to increment activeCount
	for {
		current := atomic.LoadInt32(&c.activeCount)
		if current >= c.maxCount {
			return fmt.Errorf("maximum concurrent resolvers (%d) reached", c.maxCount)
		}
		if atomic.CompareAndSwapInt32(&c.activeCount, current, current+1) {
			return nil
		}
	}
}

func (c *concurrentResolverGuard) release() {
	if c == nil {
		return
	}
	atomic.AddInt32(&c.activeCount, -1)
}

// DefaultCORSSettings returns a CORSSettings struct with sensible defaults for GraphQL APIs
func DefaultCORSSettings() *CORSSettings {
	return &CORSSettings{
		AllowedOrigins:        []string{"*"},
		AllowedMethods:        []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:        []string{"Content-Type", "Authorization"},
		AllowCredentials:      false,
		MaxAge:                86400, // 24 hours
		EnableForAllResponses: false,
	}
}

// getEffectiveOrigins returns the origins to use, applying defaults if needed
func (c *CORSSettings) getEffectiveOrigins() []string {
	if len(c.AllowedOrigins) > 0 {
		return c.AllowedOrigins
	}
	return []string{"*"}
}

// getEffectiveMethods returns the methods to use, applying defaults if needed
func (c *CORSSettings) getEffectiveMethods() []string {
	if len(c.AllowedMethods) > 0 {
		return c.AllowedMethods
	}
	return []string{"GET", "POST", "OPTIONS"}
}

// getEffectiveHeaders returns the headers to use, applying defaults if needed
func (c *CORSSettings) getEffectiveHeaders() []string {
	if len(c.AllowedHeaders) > 0 {
		return c.AllowedHeaders
	}
	return []string{"Content-Type", "Authorization"}
}

// getEffectiveMaxAge returns the max age to use, applying defaults if needed
func (c *CORSSettings) getEffectiveMaxAge() int {
	if c.MaxAge > 0 {
		return c.MaxAge
	}
	return 86400 // 24 hours
}
