---
name: cache
keywords:
- cache
- ristretto
- in-memory cache
applies_to_files:
- repo
- settings
- app
---

Use x/cache (Ristretto-backed) with key-prefix partitions and TTL for **process-local** in-memory caching. Expose cache dependency via
x/settings and keep cache logic in app/repo layers.

Do **not** use x/cache for Redis or any shared/distributed cache — it is not visible to other instances. For Redis connections read
`characteristics/redis` and `packages/datasource`.

## File-Level Hint

Use x/cache (Ristretto) with key-prefix partitions and TTL. Expose cache settings via x/settings and keep logic in app/repo layers. Not a
Redis cache.
