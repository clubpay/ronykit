The `module.go` file defines the fx dependency graph for the service.

Declare a package-level:

- `var appModule = fx.Module(settings.ModuleName, ...)`

that includes:

- `v0repo.Init` (the repo adapter `fx.Options`),
- any `pkg` `Init` modules,
- a `diDatasource` `fx.Options` block,
- and `fx.Provide(settings.New, app.New, api.New)`.

The `diDatasource` block calls:

- `di.ProvideDBParams[settings.Settings](MigrationFS)`,
- `di.ProvideRedisParams[settings.Settings]()`,
- optionally `di.ProvideNatsParams`, `di.ProvideDruidParams`, etc.

Follow it with `datasource.InitDB("", "")`, `datasource.InitRedis("", "")`,
etc. to initialize concrete data sources.

For inter-service dependencies, use
`di.ProvideStorageStub[settings.Settings](settings.ModuleName)` or similar stub
providers.

Define `LoadSettings(filename string, searchPaths ...string)` that sets
`settings.ConfigName` and `settings.ConfigPaths` package-level variables.
