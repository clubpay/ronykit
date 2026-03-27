Wire all service dependencies through `x/di` and `uber/fx`.

In `service.go` `init()`, call `di.RegisterService[Service]` with:

- `Kind` (`"service"` or `"gateway"`),
- `Name` (last segment of `settings.ModuleName`),
- `InitFn` (`LoadSettings`),
- `ModuleFn` (`Module`).

Use:

- `di.ProvideDBParams[settings.Settings](MigrationFS)`
- `di.ProvideRedisParams[settings.Settings]()`

in the `diDatasource` `fx.Options` block to extract connection params from
settings.

Use `di.ProvideStorageStub` or equivalent for inter-service stub clients with
trace propagation.

Register global middlewares via `di.RegisterMiddleware` in the `cmd/` entrypoint
`init()`.

Services are discovered at runtime via:

- `di.GetServiceByKind("service")`
- `di.GetServiceByKind("gateway")`
