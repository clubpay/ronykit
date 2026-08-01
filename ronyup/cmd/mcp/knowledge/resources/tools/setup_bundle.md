---
name: setup_bundle
---

Creates or refreshes executable bundles under `cmd/<name>/` that compile in only selected feature modules.

## Extended Guidance

Available as the MCP tool `setup_bundle` (in-process; equivalent to `ronyup setup bundle`):

| Argument        | CLI flag        | Default    |
|-----------------|-----------------|------------|
| `workspacePath` | (cwd)           | (required) |
| `name`          | `--name`        | required unless `gen` |
| `services`      | `--services`    | required when creating |
| `description`   | `--description` |            |
| `gen`           | `--gen`         | `false`    |
| `remove`        | `--remove`      | `false`    |

CLI equivalent:

```bash
ronyup setup bundle --name auth-api --services feature/auth,feature/session
ronyup setup bundle --gen
ronyup setup bundle --remove auth-api
```

## Flags

- `--name` / `-n` — bundle name (`cmd/<name>/` directory)
- `--services` / `-s` — feature module paths (`settings.ModuleName`), comma-separated, or `*` for all features in `cmd/all-in-one/features.go`
- `--description` — optional text stored in `bundles.yaml`
- `--gen` — regenerate every bundle's `features.go` from `bundles.yaml`
- `--remove` — delete a bundle from `bundles.yaml` and remove `cmd/<name>/`

## Side effects

- Updates `bundles.yaml` at the Go workspace root.
- Creates `cmd/<name>/main.go` (delegates to `pkg/runner`) and selective `features.go`.
- Initializes a Go module and runs `go work use ./cmd/<name>`.
- `ronyup setup feature` refreshes bundles that list the new feature (or use `"*"`).

## Runtime subsetting

Within any compiled binary, start a subset at runtime:

```bash
go run . --service feature/auth,feature/user
SERVICES=feature/auth go run .
```

## Related

- `knowledge://ronyup/architecture/executable-bundles` — bundle layout overview.
- `knowledge://ronyup/tools/migrate_bundles` — upgrade legacy workspaces before creating bundles.
