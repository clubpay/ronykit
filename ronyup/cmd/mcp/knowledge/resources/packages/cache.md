---
import_path: github.com/clubpay/ronykit/x/cache
short_name: cache
---

Process-local in-memory cache (Ristretto) with key-prefix partitions and TTL. Not a Redis client and not shared across instances.

## Usage Hint

```go
c, err := cache.New(cache.Config{}) // NumCounters/MaxCost/BufferItems default via rkit.Coalesce
part := c.Partition("items")
part.Set("id", item)
part.SetTTL("id", item, time.Minute)
got, ok := part.Get("id")
part.Purge("id")
```

- `cache.New(cfg Config) (*Cache, error)` — there is no `MustNew`.
- TTL is `SetTTL(key, val, ttl)`, not `Set(key, val, ttl)`.
- `Set` / `Get` / `Purge` are prefix-aware after `Partition`.

**When NOT:** multi-instance shared state, sessions, distributed locks, or anything that must survive process restart → Redis via
`datasource.InitRedis` (read `characteristics/redis`). Rate limits → `x/ratelimit`, not this cache.
