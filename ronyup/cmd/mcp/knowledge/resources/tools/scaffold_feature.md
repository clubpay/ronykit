---
name: scaffold_feature
---

Add a feature module to an existing RonyKIT workspace by delegating to `ronyup setup feature`.

This tool is **gated on approved design documents**: it refuses to run unless both `docs/design/<name>-srs.md` and `docs/design/<name>-sdd.md` exist with frontmatter `status: approved`. Produce and get approval for the SRS (`write-srs`) and SDD (`write-sdd`) first.

## Extended Guidance

The tool runs `ronyup setup feature` inside `workspacePath` with:

| Argument          | CLI flag                        | Default    |
|-------------------|---------------------------------|------------|
| `name`            | `--featureDir`, `--featureName` | (required) |
| `template`        | `--template`                    | `service`  |
| `featurePrefix`   | `--featurePrefix`               | `feature`  |
| `groupByTemplate` | `--groupByTemplate`             | `false`    |
| `skipDesignGate`  | (bypasses SRS/SDD gate)         | `false`    |

### Design gate

Before delegating to the CLI, the tool checks `workspacePath/docs/design/`:

- `<name>-srs.md` and `<name>-sdd.md` must both exist.
- Each must have YAML frontmatter `status: approved`.

If a document is missing or still `status: draft`, the tool returns an actionable error and does **not** scaffold. Agents must never set `status: approved` themselves — only the user approves. Set `skipDesignGate: true` only when the user explicitly asks to skip the design documents.

Default layout: `feature/<name>/`. With `groupByTemplate: true`: `feature/<template>/<name>/` (for example `feature/service/<name>/`).

The result contains `service.go`, `module.go`, `migration.go`, `api/`, `internal/{app,domain,repo,settings}/`, and a `gen/stub/` generator.

A side-effect blank import is added to `cmd/service/features.go` so the new module's `init()` registers it via `di.RegisterService`.

After scaffolding, run `make gen-stub` inside the feature whenever contracts change.
