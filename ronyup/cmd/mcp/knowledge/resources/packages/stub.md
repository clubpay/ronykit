---
import_path: github.com/clubpay/ronykit/stub
short_name: stub
---

Generated typed clients for other RonyKIT services. Never hand-write HTTP/gRPC clients against a sibling feature.

## Usage Hint

In the **providing** feature, `gen/stub/gen.go` calls `rony.GenerateStub` with:

- `stubgen.NewGolangEngine(stubgen.GolangConfig{PkgName: name})`
- `stubgen.NewTypescriptEngine(stubgen.TypescriptConfig{GenerateSWR: true})`

Run `make gen-stub` after every contract change. Go output: `stub/<name>stub/`. TypeScript: `stub/<name>stub-typescript/`.

In the **consuming** feature `module.go`:

```go
di.StubProvider[settings.Settings, IFooStub, *foostub.Stub](
	settings.ModuleName,
	"FooHostPort",
	foostub.New,
)
```

Host/port comes from the consumer's typed settings (`settings.Services.FooHostPort`). There is no `di.ProvideXStub`.
