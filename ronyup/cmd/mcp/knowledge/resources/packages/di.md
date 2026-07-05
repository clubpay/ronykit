---
import_path: github.com/clubpay/ronykit/x/di
short_name: di
---

Dependency injection glue for fx-based service registration, stub providers, and infra param wiring.

## Usage Hint

Wire all service dependencies through `x/di` and `uber/fx`.

In `service.go` `init()`, call `di.RegisterService[Service]` with:

- `Kind` (`"service"` or `"gateway"`),
- `Name` (`settings.ModuleName`),
- `InitFn` (`LoadSettings`),
- `ModuleFn` (`Module`).

Use:

- `di.ProvideDBParams[settings.Settings](MigrationFS)`
- `di.ProvideRedisParams[settings.Settings]()`

inside the `appModule` (or a dedicated `diDatasource` `fx.Options` block) to extract connection params from settings.

For inter-service clients, use `di.StubProvider[Settings, IStub, Stub](moduleName, hostPortField, constructor)` to expose a typed stub with B3 trace propagation; the host/port is read from `settings.Services.<hostPortField>`.

Register global middlewares via `di.RegisterMiddleware` in `cmd/runner` (imported by every `cmd/<bundle>/` executable).

Services are discovered at runtime via:

- `di.AllServices()` — every registered service.
- `di.GetService(kind, name)` — a specific registered service.
- `di.GetServiceByKind(kind)` — all services of a given kind.
