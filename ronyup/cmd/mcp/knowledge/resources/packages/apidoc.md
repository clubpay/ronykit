---
import_path: github.com/clubpay/ronykit/x/apidoc
short_name: apidoc
---

OpenAPI/Swagger 2.0 and Postman collection generation from service descriptors, with embedded UI.

## Usage Hint

```go
gen := apidoc.New("My API", "v1", "Service description")
ui, err := gen.SwaggerUI(descs...)
// also: ReDocUI / ScalarUI — serve the returned fs.FS at /docs
```

- Constructor requires title, version, and description strings: `apidoc.New(title, ver, desc) *Generator`.
- Prefer `rony.WithAPIDocs(path)` on the server in scaffolded apps; use this package when you need a standalone generator.
- Input DTOs need `json` (and `swag` when documenting enums) tags or stub/doc generation fails.

Read `architecture/apidoc-generation`.
