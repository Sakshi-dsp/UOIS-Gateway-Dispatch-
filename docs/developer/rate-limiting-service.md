# Rate Limiting Service - Developer Documentation

**Service:** `internal/services/auth/rate_limit_service.go`  
**Status:** ✅ Production-Ready  
**Last Updated:** January 2025

---

## Overview

Redis-based sliding window counter rate limiting service for per-client request throttling. Enforces configurable burst limits and steady-state limits per client, returning accurate reset times and proper error taxonomy.

---

## Architecture

### Design Pattern
- **Dependency Injection**: Uses `RedisClient` interface for Redis operations
- **Service Layer**: Stateless service, all state in Redis
- **Error Taxonomy**: Domain errors (65011, 65012) mapped to HTTP status codes

### Key Components

```go
type RateLimitService struct {
    redis  RedisClient  // Redis client interface
    config RateLimitConfig
    logger *zap.Logger
}
```

### RedisClient Interface

```go
type RedisClient interface {
    Incr(ctx context.Context, key string) *redis.IntCmd
    Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
    TTL(ctx context.Context, key string) *redis.DurationCmd
}
```

---

## API

### CheckRateLimit

```go
func (s *RateLimitService) CheckRateLimit(
    ctx context.Context, 
    clientID string,
) (allowed bool, remaining int64, resetAt time.Time, err error)
```

**Parameters:**
- `ctx`: Request context
- `clientID`: Client identifier for rate limiting

**Returns:**
- `allowed`: Whether request is allowed
- `remaining`: Remaining requests in current window (-1 if disabled)
- `resetAt`: Time when rate limit resets (zero if disabled)
- `err`: Error if Redis operation fails (domain error 65011)

**Behavior:**
- If disabled: Returns `allowed=true`, `remaining=-1`, `resetAt=zero`
- If Redis error: Returns `allowed=false`, domain error 65011 (HTTP 503)
- If limit exceeded: Returns `allowed=false`, `remaining=0`
- If within limits: Returns `allowed=true`, `remaining=limit-currentCount`

### GetRateLimitError

```go
func (s *RateLimitService) GetRateLimitError(
    ctx context.Context, 
    clientID string,
) error
```

**Returns:** Domain error 65012 (HTTP 429) for rate limit exceeded.

---

## Configuration

### RateLimitConfig

```go
type RateLimitConfig struct {
    Enabled           bool   // Enable/disable rate limiting
    RedisKeyPrefix    string // Redis key prefix (e.g., "rate_limit:uois")
    RequestsPerMinute int    // Steady-state limit per window (NOTE: consider renaming to RequestsPerWindow)
    Burst             int    // Burst limit (enforced separately)
    WindowSeconds     int    // Window duration in seconds
}
```

**Note**: `RequestsPerMinute` is used as "requests per window" (where window = `WindowSeconds`). The naming can be confusing if `WindowSeconds ≠ 60`. Consider renaming to `RequestsPerWindow` in a future refactoring.

### Environment Variables

- `RATE_LIMIT_ENABLED` (default: true)
- `RATE_LIMIT_REDIS_KEY_PREFIX` (default: "rate_limit:uois")
- `RATE_LIMIT_REQUESTS_PER_MINUTE` (default: 60)
- `RATE_LIMIT_BURST` (default: 10)
- `RATE_LIMIT_WINDOW_SECONDS` (default: 60)

---

## Algorithm

### Sliding Window Counter

1. **Increment Counter**: `INCR rate_limit:uois:{clientID}`
2. **Set Expiry (First Request Only)**: `EXPIRE key {windowSeconds}` (only when count == 1)
3. **Get TTL**: `TTL key` (for accurate resetAt calculation)
4. **Check Limits**:
   - If `currentCount > burstLimit`: Reject
   - If `currentCount > requestsPerMinute`: Reject
   - Otherwise: Allow

### Critical Implementation Details

#### ✅ Expiry Reset Prevention
- **Issue**: Setting expiry on every request resets the window
- **Fix**: Only call `Expire()` when `currentCount == 1`
- **Impact**: Ensures correct sliding window behavior under sustained load

#### ✅ Accurate Reset Time
- **Issue**: Using `now + windowDuration` ignores actual Redis TTL
- **Fix**: Use `TTL` command to get actual remaining time
- **Impact**: Provides accurate `Retry-After` headers for clients

#### ✅ Error Taxonomy
- **Redis Failure**: Returns domain error 65011 (HTTP 503 - Dependency Unavailable)
- **Rate Limit Exceeded**: Returns domain error 65012 (HTTP 429 - Too Many Requests)

---

## Error Handling

### Error Codes

| Code | Message | HTTP Status | Retryable | Description |
|------|---------|-------------|-----------|-------------|
| 65011 | rate limiting unavailable | 503 | Yes | Redis/dependency failure |
| 65012 | rate limit exceeded | 429 | Yes | Client exceeded rate limit |

### Error Usage

```go
// Check rate limit
allowed, remaining, resetAt, err := service.CheckRateLimit(ctx, clientID)
if err != nil {
    // err is domain error 65011 (Redis failure)
    return err
}

if !allowed {
    // Rate limit exceeded
    rateLimitErr := service.GetRateLimitError(ctx, clientID)
    // rateLimitErr is domain error 65012 (HTTP 429)
    return rateLimitErr
}
```

---

## Redis Key Format

```
{RedisKeyPrefix}:{clientID}
```

**Example:**
```
rate_limit:uois:client-123
```

**TTL:** Set to `WindowSeconds` on first increment, auto-expires after window.

**Future Extensibility:**
The key format can be extended for more granular rate limiting:
- `{prefix}:{clientID}:{endpoint}` - Per-endpoint limits
- `{prefix}:{clientID}:{ip}` - Per-IP limits
- `{prefix}:{clientID}:{endpoint}:{ip}` - Combined granularity

Implementation: Update key generation in `CheckRateLimit()` when needed.

---

## Usage Example

```go
// Initialize
redisClient := redis.NewClient(&redis.Options{...})
cfg := config.RateLimitConfig{
    Enabled:           true,
    RedisKeyPrefix:    "rate_limit:uois",
    RequestsPerMinute: 60,
    Burst:             10,
    WindowSeconds:     60,
}
logger := zap.NewProduction()
rateLimitService := auth.NewRateLimitService(redisClient, cfg, logger)

// Check rate limit
allowed, remaining, resetAt, err := rateLimitService.CheckRateLimit(ctx, "client-123")
if err != nil {
    // Handle Redis error (65011)
    return err
}

if !allowed {
    // Handle rate limit exceeded (65012)
    rateLimitErr := rateLimitService.GetRateLimitError(ctx, "client-123")
    return rateLimitErr
}

// Request allowed
// Use `remaining` and `resetAt` for response headers
```

---

## Testing

### Test Coverage

**7 test cases:**
1. `TestRateLimitService_CheckRateLimit_Success_FirstRequest` - First request sets expiry
2. `TestRateLimitService_CheckRateLimit_Success_SubsequentRequest` - Subsequent requests don't reset expiry
3. `TestRateLimitService_CheckRateLimit_Exceeded` - Rate limit exceeded scenario
4. `TestRateLimitService_CheckRateLimit_Disabled` - Disabled mode
5. `TestRateLimitService_CheckRateLimit_RedisError` - Redis error handling (65011)
6. `TestRateLimitService_CheckRateLimit_BurstLimit` - Burst limit enforcement
7. `TestRateLimitService_GetRateLimitError` - Domain error generation (65012)

### Mock Redis Client

Uses `MockRedisClient` implementing `RedisClient` interface for unit testing.

---

## Production Considerations

### ✅ Fixed Issues

1. **Expiry Reset Bug**: Fixed - expiry only set on first increment
2. **Error Taxonomy**: Fixed - Redis errors return 65011, rate limits return 65012
3. **Reset Time Accuracy**: Fixed - uses Redis TTL for accurate resetAt

### Performance

- **Redis Operations**: 2-3 operations per request (INCR, TTL, optional EXPIRE)
- **Latency**: Minimal (Redis is fast)
- **Scalability**: Redis handles high concurrency well

### Monitoring

- Log rate limit violations (when `allowed=false`)
- Monitor Redis errors (domain error 65011)
- Track rate limit hit rates per client

### Future Improvements (Optional)

#### 1. Config Naming Clarity
- **Current**: `RequestsPerMinute` + `WindowSeconds` (can be confusing if WindowSeconds ≠ 60)
- **Consideration**: Rename to `RequestsPerWindow` for clarity
- **Impact**: Breaking change, requires config migration
- **Status**: Documented for future refactoring

#### 2. Key Extensibility
- **Current**: `{prefix}:{clientID}`
- **Future Options**:
  - `{prefix}:{clientID}:{endpoint}` - Per-endpoint rate limiting
  - `{prefix}:{clientID}:{ip}` - Per-IP rate limiting
  - `{prefix}:{clientID}:{endpoint}:{ip}` - Combined granularity
- **Implementation**: Key format is already flexible, just update key generation logic
- **Status**: Ready for extension when needed

#### 3. Metrics Instrumentation
- **Recommended Metric**: `rate_limit_exceeded_total{reason="burst|steady_state", client_id="..."}`
- **Library**: Prometheus (suggested in FR, not yet integrated)
- **Implementation**: Add counter increment when rate limit is exceeded
- **Status**: TODO - pending metrics library integration

---

## Integration Points

### HTTP Middleware

```go
func RateLimitMiddleware(rateLimitService *auth.RateLimitService) gin.HandlerFunc {
    return func(c *gin.Context) {
        clientID := extractClientID(c)
        
        allowed, remaining, resetAt, err := rateLimitService.CheckRateLimit(c.Request.Context(), clientID)
        if err != nil {
            // Redis error - return 503
            c.JSON(503, gin.H{"error": err.Error()})
            c.Abort()
            return
        }
        
        if !allowed {
            // Rate limit exceeded - return 429
            rateLimitErr := rateLimitService.GetRateLimitError(c.Request.Context(), clientID)
            c.Header("Retry-After", formatRetryAfter(resetAt))
            c.JSON(429, gin.H{"error": rateLimitErr.Error()})
            c.Abort()
            return
        }
        
        // Set rate limit headers
        c.Header("X-RateLimit-Limit", strconv.FormatInt(limit, 10))
        c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
        c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
        
        c.Next()
    }
}
```

---

## Related Files

- **Implementation**: `internal/services/auth/rate_limit_service.go`
- **Tests**: `internal/services/auth/rate_limit_service_test.go`
- **Config**: `internal/config/config.go` (RateLimitConfig)
- **Errors**: `pkg/errors/errors.go` (DomainError, error codes)

---

## Summary

**What It Does:**
- Per-client rate limiting using Redis sliding window counter
- Enforces burst and steady-state limits separately
- Returns accurate reset times and remaining counts
- Proper error taxonomy for infrastructure failures vs rate limits

**Key Features:**
- ✅ Correct sliding window semantics (no expiry reset bug)
- ✅ Accurate resetAt calculation (uses Redis TTL)
- ✅ Proper error handling (65011 for Redis failures, 65012 for rate limits)
- ✅ Disabled mode support
- ✅ Dependency injection for testability

**Production Status:** ✅ Ready for use

