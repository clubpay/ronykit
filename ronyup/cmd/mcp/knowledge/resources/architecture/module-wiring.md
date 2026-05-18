The `module.go` file defines the fx dependency graph for the service.

Declare a package-level:

- `var appModule = fx.Module(settings.ModuleName, ...)`

that includes:

- `v0repo.Init` (the repo adapter `fx.Options`),
- any `pkg` `Init` modules,
- `fx.Provide(settings.New, app.New, api.New)`,
- the datasource param providers (`di.ProvideDBParams`, `di.ProvideRedisParams`),
- and the concrete initializers (`datasource.InitDB("", "")`,
  `datasource.InitRedis("", "")`).

Use the datasource param helpers exported by `x/di`:

- `di.ProvideDBParams[settings.Settings](MigrationFS)`,
- `di.ProvideRedisParams[settings.Settings]()`.

For inter-service dependencies, use `di.StubProvider[Settings, IStub, Stub](moduleName,
hostPortField, constructor)` to provide a typed stub client with trace propagation.

In `service.go`, define `LoadSettings(filename string, searchPaths ...string)` that
sets the `settings.ConfigName` and `settings.ConfigPaths` package-level variables;
`di.RegisterService` will call it with the runtime config path before `Module()` is
executed.
