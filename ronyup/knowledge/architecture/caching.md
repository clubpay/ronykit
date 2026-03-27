Use `x/cache` (Ristretto-backed) for in-memory caching with key-prefix
partitions and TTL.

Avoid hand-rolled `sync.Map` or custom expiring maps.
