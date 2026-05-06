# Rate Limiter Package

Shared rate limiting primitives for GC operations, extracted from zen-gc and zen-watcher.

## Usage

```go
import "github.com/zenmesh/zen-sdk/pkg/gc/ratelimiter"

// Create rate limiter (10 operations per second)
rl := ratelimiter.NewRateLimiter(10)

// Wait for next operation (blocks until allowed)
if err := rl.Wait(ctx); err != nil {
    return err
}

// Or check without waiting
if !rl.Allow() {
    return errors.New("rate limit exceeded")
}

// Update rate dynamically
rl.SetRate(20) // 20 ops/sec
```

## Implementation

Uses `golang.org/x/time/rate` (token bucket algorithm) for efficient rate limiting.

## Migration

**From zen-gc**:
```go
// Before
import "github.com/zenmesh/zen-gc/pkg/controller"
rl := controller.NewRateLimiter(maxPerSecond)

// After
import "github.com/zenmesh/zen-sdk/pkg/gc/ratelimiter"
rl := ratelimiter.NewRateLimiter(maxPerSecond)
```

**From zen-watcher**:
```go
// Before
import "github.com/zenmesh/zen-watcher/pkg/server"
rl := server.NewRateLimiter(maxTokens, refillInterval)

// After
import "github.com/zenmesh/zen-sdk/pkg/gc/ratelimiter"
rl := ratelimiter.NewRateLimiter(maxPerSecond)
```

