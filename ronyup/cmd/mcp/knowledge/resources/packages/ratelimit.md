---
import_path: github.com/clubpay/ronykit/x/ratelimit
short_name: ratelimit
---

Distributed, Redis-backed rate limiting using an atomic Lua token-bucket. Use this instead of hand-rolling counters or importing a third-party limiter, so limits are consistent and shared across all service instances.

## Usage Hint

Construct once with a `*redis.Client` (provided by `x/datasource`), then check per key:

```go
limiter := ratelimit.NewLimiter(rdb)

res, err := limiter.Allow(ctx, "login:"+userID, ratelimit.PerMinute(10))
if err != nil {
    return err
}
if res.Allowed == 0 {
    // over limit — res.RetryAfter / res.ResetAfter tell the client when to retry
    return errs.B().Code(errs.RateLimited).Msg("RATE_LIMITED").Err()
}
```

- Build limits with `ratelimit.PerSecond(n)` / `PerMinute(n)` / `PerHour(n)`, or a custom `ratelimit.Limit{Rate, Burst, Period}`.
- `Allow` / `AllowN` / `AllowAtMost` report how many events are permitted; `Result` carries `Allowed`, `Remaining`, `RetryAfter`, `ResetAfter`.
- `Reset(ctx, key)` clears a key's usage.
