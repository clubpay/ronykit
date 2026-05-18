---
name: scaffold_feature
---
Add a feature module to an existing RonyKIT workspace by delegating to `ronyup setup feature`.

## Extended Guidance

The tool runs `ronyup setup feature` inside `workspacePath` with the provided `name` as
both `--featureDir` and `--featureName` (template `service`). The result is a new module
at `feature/service/<name>/` containing `service.go`, `module.go`, `migration.go`,
`api/`, `internal/{app,domain,repo,settings}/`, and a `gen/stub/` generator.

A side-effect blank import is added to `cmd/service/features.go` so the new module's
`init()` registers it via `di.RegisterService`.

After scaffolding, run `make gen-stub` inside the feature whenever contracts change.
