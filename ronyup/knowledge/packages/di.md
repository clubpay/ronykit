---
import_path: github.com/clubpay/ronykit/x/di
short_name: di
---
Dependency injection glue for fx-based service registration, stub providers, and infra param wiring.

## Usage Hint

Use di.RegisterService in module.go; di.ProvideDBParams/ProvideRedisParams for infra; di.StubProvider for inter-service stubs.
