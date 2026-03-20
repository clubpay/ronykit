---
import_path: github.com/clubpay/ronykit/x/cache
short_name: cache
---
In-memory Ristretto cache with key-prefix partitions and TTL support.

## Usage Hint

Create cache.New(cfg), use cache.Partition for namespace isolation, Set/Get with TTL.
