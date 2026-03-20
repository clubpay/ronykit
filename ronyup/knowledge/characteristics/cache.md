---
keywords:
  - redis
  - cache
applies_to_files:
  - repo
  - settings
  - app
---
Use x/cache (Ristretto-backed) with key-prefix partitions and TTL for in-memory caching. Expose cache dependency via x/settings and keep cache logic in app/repo layers.

## File-Level Hint

Use x/cache (Ristretto) with key-prefix partitions and TTL. Expose cache settings via x/settings and keep logic in app/repo layers.
