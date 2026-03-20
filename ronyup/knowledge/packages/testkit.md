---
import_path: github.com/clubpay/ronykit/x/testkit
short_name: testkit
---
Integration test harness with fx, settings wiring, and Gnomock containers for Postgres/Redis.

## Usage Hint

Use testkit.Run in _test.go; call testkit.InitDB/InitRedis for container-backed infra.
