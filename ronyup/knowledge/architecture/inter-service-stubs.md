For cross-service communication, consume generated stubs instead of duplicating
contracts.

Each service generates stubs via `make gen-stub` (or `make gen-go-stub` /
`make gen-ts-stub`), which runs `gen/stub/gen.go` (a main package using cobra)
that calls `rony.GenerateStub` with `stubgen.NewGolangEngine` or
`stubgen.NewTypescriptEngine`.

Generated outputs:

- Go stubs: `stub/<servicename>stub/`
- TypeScript stubs: `stub/<servicename>stub-typescript/`

In the consuming service's `module.go`, provide the stub via
`di.ProvideStorageStub[settings.Settings](settings.ModuleName)` or equivalent
for other services.

The stub host/port is read from the consuming service's settings (for example
`settings.ServicesConfig.StorageHostPort`).

Always regenerate stubs after changing contracts in the source service.
