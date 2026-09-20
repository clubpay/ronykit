---
name: redis
keywords:
- redis
- dragonfly
- valkey
applies_to_files:
- module
- settings
- repo
---

Redis (or Dragonfly/Valkey) is infrastructure, not `x/cache`. Open the client with `x/datasource.InitRedis` and
`di.ProvideRedisParams[settings.Settings]()` in `module.go`. Read connection fields from typed `x/settings`.

Use Redis-backed packages on that client: `x/ratelimit` for distributed limits. Keep repository/app logic behind ports.

Do **not** substitute `x/cache` (in-memory Ristretto, process-local) for Redis.

## File-Level Hint

Wire Redis via `datasource.InitRedis` + `di.ProvideRedisParams`. Do not use `x/cache` for shared state.
