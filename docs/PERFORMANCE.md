# Performance & Optimization

This guide covers performance optimization, caching strategies, and DoS protection for production go-quickgraph applications.

## Quick Performance Wins

### 1. Enable Request Caching

Caching parsed queries provides ~10x speedup for repeated requests:

```go
import "github.com/patrickmn/go-cache"

type SimpleGraphRequestCache struct {
    cache *cache.Cache
}

func (c *SimpleGraphRequestCache) GetRequestStub(ctx context.Context, request string) (*quickgraph.RequestStub, error) {
    if cached, found := c.cache.Get(request); found {
        if entry, ok := cached.(*CacheEntry); ok {
            return entry.Stub, entry.Err
        }
    }
    return nil, nil
}

func (c *SimpleGraphRequestCache) SetRequestStub(ctx context.Context, request string, stub *quickgraph.RequestStub, err error) {
    c.cache.Set(request, &CacheEntry{Stub: stub, Err: err}, cache.DefaultExpiration)
}

type CacheEntry struct {
    Stub *quickgraph.RequestStub
    Err  error
}

// Use the cache
g := quickgraph.Graphy{
    RequestCache: &SimpleGraphRequestCache{
        cache: cache.New(5*time.Minute, 10*time.Minute),
    },
}
```

### 2. Use DoS Protection

Prevent resource exhaustion with query limits:

```go
g := quickgraph.Graphy{
    QueryLimits: &quickgraph.QueryLimits{
        MaxDepth:               10,   // Prevent deeply nested queries
        MaxFields:              100,  // Limit fields per level
        MaxAliases:             50,   // Prevent alias amplification
        MaxArraySize:           1000, // Limit array results
        MaxConcurrentResolvers: 50,   // Control parallel execution
        MaxComplexity:          1000, // Overall query cost
    },
}
```

### 3. Enable Timing

Monitor performance with built-in timing:

```go
g := quickgraph.Graphy{
    EnableTiming: true,
}

// Timing data appears in GraphQL extensions
{
  "data": {...},
  "extensions": {
    "timing": {
      "parsing": "245µs",
      "validation": "89µs", 
      "execution": "2.3ms",
      "total": "2.634ms"
    }
  }
}
```

## Caching Strategies

### Application-Level Caching

Cache expensive function results:

```go
type ProductCache struct {
    cache map[int]*Product
    mutex sync.RWMutex
    ttl   time.Duration
}

func (c *ProductCache) Get(id int) (*Product, bool) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    product, exists := c.cache[id]
    return product, exists
}

func (c *ProductCache) Set(id int, product *Product) {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    c.cache[id] = product
    
    // Simple TTL cleanup
    go func() {
        time.Sleep(c.ttl)
        c.mutex.Lock()
        delete(c.cache, id)
        c.mutex.Unlock()
    }()
}

var productCache = &ProductCache{
    cache: make(map[int]*Product),
    ttl:   5 * time.Minute,
}

func GetProduct(ctx context.Context, id int) (*Product, error) {
    // Check cache first
    if product, found := productCache.Get(id); found {
        return product, nil
    }
    
    // Load from database
    product, err := database.GetProduct(id)
    if err != nil {
        return nil, err
    }
    
    // Cache the result
    productCache.Set(id, product)
    return product, nil
}
```

### Context-Based Caching

Use request context for per-request caching:

```go
type RequestCache struct {
    users    map[int]*User
    products map[int]*Product
    mutex    sync.RWMutex
}

func GetRequestCache(ctx context.Context) *RequestCache {
    if cache, ok := ctx.Value("requestCache").(*RequestCache); ok {
        return cache
    }
    // Return empty cache if not found
    return &RequestCache{
        users:    make(map[int]*User),
        products: make(map[int]*Product),
    }
}

func GetUserCached(ctx context.Context, id int) (*User, error) {
    cache := GetRequestCache(ctx)
    
    cache.mutex.RLock()
    if user, exists := cache.users[id]; exists {
        cache.mutex.RUnlock()
        return user, nil
    }
    cache.mutex.RUnlock()
    
    // Load from database
    user, err := database.GetUser(id)
    if err != nil {
        return nil, err
    }
    
    // Cache in request context
    cache.mutex.Lock()
    cache.users[id] = user
    cache.mutex.Unlock()
    
    return user, nil
}

// HTTP middleware to inject request cache
func requestCacheMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        cache := &RequestCache{
            users:    make(map[int]*User),
            products: make(map[int]*Product),
        }
        ctx := context.WithValue(r.Context(), "requestCache", cache)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### DataLoader Pattern

Batch and cache database queries to solve N+1 problems:

```go
type UserLoader struct {
    fetch    func([]int) ([]*User, error)
    waiting  map[int][]chan *UserResult
    batching map[int]*User
    mutex    sync.Mutex
    maxBatch int
    delay    time.Duration
}

type UserResult struct {
    User  *User
    Error error
}

func NewUserLoader(fetchFunc func([]int) ([]*User, error)) *UserLoader {
    return &UserLoader{
        fetch:    fetchFunc,
        waiting:  make(map[int][]chan *UserResult),
        batching: make(map[int]*User),
        maxBatch: 100,
        delay:    time.Millisecond,
    }
}

func (ul *UserLoader) Load(ctx context.Context, id int) (*User, error) {
    result := make(chan *UserResult, 1)
    
    ul.mutex.Lock()
    ul.waiting[id] = append(ul.waiting[id], result)
    
    if len(ul.waiting) == 1 {
        // First request - start batch timer
        go ul.processBatch()
    }
    ul.mutex.Unlock()
    
    select {
    case res := <-result:
        return res.User, res.Error
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

func (ul *UserLoader) processBatch() {
    time.Sleep(ul.delay)
    
    ul.mutex.Lock()
    if len(ul.waiting) == 0 {
        ul.mutex.Unlock()
        return
    }
    
    // Collect IDs to fetch
    var ids []int
    waitingCopy := make(map[int][]chan *UserResult)
    for id, channels := range ul.waiting {
        ids = append(ids, id)
        waitingCopy[id] = channels
    }
    ul.waiting = make(map[int][]chan *UserResult)
    ul.mutex.Unlock()
    
    // Batch fetch
    users, err := ul.fetch(ids)
    
    // Create result map
    userMap := make(map[int]*User)
    if err == nil {
        for _, user := range users {
            if user != nil {
                userMap[user.ID] = user
            }
        }
    }
    
    // Send results to waiting channels
    for id, channels := range waitingCopy {
        result := &UserResult{}
        if err != nil {
            result.Error = err
        } else if user, found := userMap[id]; found {
            result.User = user
        } else {
            result.Error = fmt.Errorf("user %d not found", id)
        }
        
        for _, ch := range channels {
            ch <- result
            close(ch)
        }
    }
}

// Use in HTTP middleware
func dataLoaderMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userLoader := NewUserLoader(func(ids []int) ([]*User, error) {
            return database.GetUsersByIDs(ids)
        })
        
        ctx := context.WithValue(r.Context(), "userLoader", userLoader)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

// Use in resolvers
func (p *Post) Author(ctx context.Context) (*User, error) {
    if loader, ok := ctx.Value("userLoader").(*UserLoader); ok {
        return loader.Load(ctx, p.AuthorID)
    }
    // Fallback to direct load
    return database.GetUser(p.AuthorID)
}
```

## Query Analysis and Optimization

### Query Complexity Analysis

Implement custom complexity calculation:

```go
type ComplexityAnalyzer struct {
    maxComplexity int
    typeCosts     map[string]int
    fieldCosts    map[string]int
}

func (ca *ComplexityAnalyzer) AnalyzeQuery(query string) (int, error) {
    // Parse query and calculate cost
    totalCost := 0
    
    // Example: each field costs 1, each nested level costs 2x
    // Arrays multiply cost by estimated size
    
    if totalCost > ca.maxComplexity {
        return totalCost, fmt.Errorf("query complexity %d exceeds limit %d", 
            totalCost, ca.maxComplexity)
    }
    
    return totalCost, nil
}

// Middleware to check query complexity
func complexityMiddleware(analyzer *ComplexityAnalyzer) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if r.Method == "POST" {
                body, _ := io.ReadAll(r.Body)
                r.Body = io.NopCloser(bytes.NewReader(body))
                
                var req struct {
                    Query string `json:"query"`
                }
                if json.Unmarshal(body, &req) == nil {
                    if cost, err := analyzer.AnalyzeQuery(req.Query); err != nil {
                        http.Error(w, err.Error(), http.StatusBadRequest)
                        return
                    } else if cost > 100 {
                        log.Printf("High complexity query: %d", cost)
                    }
                }
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### Selective Field Loading

Load expensive fields only when requested using method patterns:

```go
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
    
    // Don't preload expensive data
    cachedStats *UserStats
}

// Only compute/load when GraphQL query requests this field
func (u *User) Stats(ctx context.Context) (*UserStats, error) {
    if u.cachedStats != nil {
        return u.cachedStats, nil
    }
    
    // Expensive computation only happens if field is requested
    stats, err := calculateUserStats(u.ID)
    if err != nil {
        return nil, err
    }
    
    u.cachedStats = stats // Cache for this request
    return stats, nil
}

// Only load posts when GraphQL query requests this field
func (u *User) Posts(ctx context.Context, limit *int) ([]Post, error) {
    maxLimit := 10
    if limit != nil && *limit > 0 {
        maxLimit = *limit
    }
    
    return database.GetPostsByUser(u.ID, maxLimit)
}
```

### Lazy Loading

Load expensive fields only when requested:

```go
type User struct {
    ID       int     `json:"id"`
    Name     string  `json:"name"`
    Email    string  `json:"email"`
    
    // Expensive fields loaded lazily
    posts    []Post  // Not included in JSON by default
    stats    *UserStats
}

// Lazy load posts only if requested
func (u *User) Posts(ctx context.Context, limit *int) ([]Post, error) {
    maxLimit := 10
    if limit != nil && *limit > 0 {
        maxLimit = *limit
    }
    
    return database.GetPostsByUser(u.ID, maxLimit)
}

// Lazy load stats only if requested
func (u *User) Stats(ctx context.Context) (*UserStats, error) {
    if u.stats != nil {
        return u.stats, nil
    }
    
    stats, err := calculateUserStats(u.ID)
    if err != nil {
        return nil, err
    }
    
    u.stats = stats // Cache for this request
    return stats, nil
}
```

## DoS Protection

### Query Depth Limits

```go
// Built-in depth limiting
g.QueryLimits = &quickgraph.QueryLimits{
    MaxDepth: 10, // Prevents queries like user.posts.author.posts.author...
}

// Custom depth analysis
func analyzeDepth(query string) int {
    // Parse and count maximum nesting depth
    return depth
}
```

### Field Count Limits

```go
// Prevent overly broad queries
g.QueryLimits = &quickgraph.QueryLimits{
    MaxFields: 100, // Limit fields per selection set
}

// Example of rejected query:
// {
//   user {
//     field1 field2 field3 ... field101  # Too many fields
//   }
// }
```

### Alias Limits

```go
// Prevent alias amplification attacks
g.QueryLimits = &quickgraph.QueryLimits{
    MaxAliases: 50,
}

// Example of rejected query:
// {
//   a1: expensiveQuery { ... }
//   a2: expensiveQuery { ... }
//   ... (50 more aliases)
// }
```

### Rate Limiting

```go
import "golang.org/x/time/rate"

type RateLimiter struct {
    limiter *rate.Limiter
}

func NewRateLimiter(requestsPerSecond int) *RateLimiter {
    return &RateLimiter{
        limiter: rate.NewLimiter(rate.Limit(requestsPerSecond), requestsPerSecond),
    }
}

func (rl *RateLimiter) Allow() bool {
    return rl.limiter.Allow()
}

// Rate limiting middleware
func rateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if !limiter.Allow() {
                http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

## Memory Optimization

### Efficient Data Structures

```go
// ✅ Use pointers for large or optional structs
type User struct {
    Profile *UserProfile `json:"profile"` // Only allocated when needed
}

// ✅ Embed small structs instead of using pointers
type User struct {
    Timestamps // Embedded, no pointer overhead
    Name string
}

type Timestamps struct {
    CreatedAt time.Time
    UpdatedAt time.Time
}

// ✅ Use slices for collections
type User struct {
    Tags     []string   `json:"tags"`     // Dynamic collection
    PostIDs  []int      `json:"postIds"`  // Variable length
}
```

### Memory Pool for Frequent Allocations

```go
var requestPool = sync.Pool{
    New: func() interface{} {
        return &RequestContext{
            Cache: make(map[string]interface{}),
        }
    },
}

type RequestContext struct {
    Cache map[string]interface{}
}

func (rc *RequestContext) Reset() {
    for k := range rc.Cache {
        delete(rc.Cache, k)
    }
}

// Use pool in HTTP handler
func graphqlHandler(g *quickgraph.Graphy) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        reqCtx := requestPool.Get().(*RequestContext)
        defer func() {
            reqCtx.Reset()
            requestPool.Put(reqCtx)
        }()
        
        ctx := context.WithValue(r.Context(), "requestContext", reqCtx)
        g.HttpHandler().ServeHTTP(w, r.WithContext(ctx))
    }
}
```

## Database Optimization

### Connection Pooling

```go
import "database/sql"

func setupDB() *sql.DB {
    db, err := sql.Open("postgres", connectionString)
    if err != nil {
        log.Fatal(err)
    }
    
    // Optimize connection pool
    db.SetMaxOpenConns(25)           // Maximum open connections
    db.SetMaxIdleConns(5)            // Maximum idle connections
    db.SetConnMaxLifetime(time.Hour) // Maximum connection lifetime
    db.SetConnMaxIdleTime(30 * time.Minute) // Maximum idle time
    
    return db
}
```

### Prepared Statements

```go
type UserQueries struct {
    getUser      *sql.Stmt
    getUserBatch *sql.Stmt
    updateUser   *sql.Stmt
}

func NewUserQueries(db *sql.DB) (*UserQueries, error) {
    getUser, err := db.Prepare("SELECT id, name, email FROM users WHERE id = $1")
    if err != nil {
        return nil, err
    }
    
    getUserBatch, err := db.Prepare("SELECT id, name, email FROM users WHERE id = ANY($1)")
    if err != nil {
        return nil, err
    }
    
    return &UserQueries{
        getUser:      getUser,
        getUserBatch: getUserBatch,
    }, nil
}

func (uq *UserQueries) GetUser(id int) (*User, error) {
    var user User
    err := uq.getUser.QueryRow(id).Scan(&user.ID, &user.Name, &user.Email)
    return &user, err
}
```

### Batch Queries

```go
func GetUsersByIDs(ids []int) ([]*User, error) {
    if len(ids) == 0 {
        return nil, nil
    }
    
    // Use batch query instead of N individual queries
    query := `
        SELECT id, name, email 
        FROM users 
        WHERE id = ANY($1)
    `
    
    rows, err := db.Query(query, pq.Array(ids))
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var users []*User
    for rows.Next() {
        var user User
        err := rows.Scan(&user.ID, &user.Name, &user.Email)
        if err != nil {
            return nil, err
        }
        users = append(users, &user)
    }
    
    return users, nil
}
```

## Monitoring and Profiling

### Custom Metrics

```go
import "github.com/prometheus/client_golang/prometheus"

var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "graphql_request_duration_seconds",
            Help: "GraphQL request duration",
        },
        []string{"operation", "status"},
    )
    
    requestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "graphql_requests_total", 
            Help: "Total GraphQL requests",
        },
        []string{"operation", "status"},
    )
)

func init() {
    prometheus.MustRegister(requestDuration)
    prometheus.MustRegister(requestsTotal)
}

// Metrics middleware
func metricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        next.ServeHTTP(w, r)
        
        duration := time.Since(start)
        requestDuration.WithLabelValues("query", "success").Observe(duration.Seconds())
        requestsTotal.WithLabelValues("query", "success").Inc()
    })
}
```

### Performance Profiling

```go
import _ "net/http/pprof"

func main() {
    // Enable pprof endpoints
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    
    // Your GraphQL server
    g := setupGraphQL()
    http.Handle("/graphql", g.HttpHandler())
    http.ListenAndServe(":8080", nil)
}

// Access profiling at:
// http://localhost:6060/debug/pprof/
// go tool pprof http://localhost:6060/debug/pprof/heap
// go tool pprof http://localhost:6060/debug/pprof/profile
```

## Benchmarking

### Function Benchmarks

```go
func BenchmarkGetUser(b *testing.B) {
    ctx := context.Background()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := GetUser(ctx, 1)
        if err != nil {
            b.Fatal(err)
        }
    }
}

func BenchmarkQueryProcessing(b *testing.B) {
    g := setupTestGraphy()
    ctx := context.Background()
    query := `{ user(id: 1) { name email } }`
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := g.ProcessRequest(ctx, query, "")
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

### Load Testing

```go
// Simple load test
func TestConcurrentRequests(t *testing.T) {
    g := setupTestGraphy()
    server := httptest.NewServer(g.HttpHandler())
    defer server.Close()
    
    const numRequests = 100
    const concurrency = 10
    
    semaphore := make(chan struct{}, concurrency)
    var wg sync.WaitGroup
    
    for i := 0; i < numRequests; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            semaphore <- struct{}{}
            defer func() { <-semaphore }()
            
            resp, err := http.Post(server.URL+"/graphql", "application/json", 
                strings.NewReader(`{"query": "{ users { id name } }"}`))
            assert.NoError(t, err)
            assert.Equal(t, 200, resp.StatusCode)
            resp.Body.Close()
        }()
    }
    
    wg.Wait()
}
```

## Production Checklist

### ✅ Performance Optimizations
- [ ] Request caching enabled
- [ ] DoS protection configured
- [ ] Database connection pooling
- [ ] Prepared statements for frequent queries
- [ ] DataLoader pattern for N+1 problems
- [ ] Lazy loading for expensive fields

### ✅ Monitoring
- [ ] Request timing enabled
- [ ] Custom metrics collected
- [ ] Error rates monitored
- [ ] Memory usage tracked
- [ ] Database query performance monitored

### ✅ Security
- [ ] Query depth limits
- [ ] Field count limits
- [ ] Alias limits
- [ ] Rate limiting
- [ ] Authentication/authorization
- [ ] Input validation

### ✅ Scalability
- [ ] Horizontal scaling support
- [ ] Stateless request processing
- [ ] External cache integration (Redis)
- [ ] Load balancer configuration
- [ ] Circuit breaker patterns

## Next Steps

- **[Error Handling](ERROR_HANDLING.md)** - Robust error management
- **[Authentication](AUTH_PATTERNS.md)** - Security patterns
- **[Common Patterns](COMMON_PATTERNS.md)** - DataLoader, pagination, and more