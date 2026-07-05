Creates or refreshes executable bundles under `cmd/<name>/` that compile in only selected feature modules.

## Usage

```bash
ronyup setup bundle --name auth-api --services feature/auth,feature/session
ronyup setup bundle --gen
ronyup setup bundle --remove auth-api
```

## Flags

- `--name` / `-n` — bundle name (`cmd/<name>/` directory)
- `--services` / `-s` — feature module paths (`settings.ModuleName`), comma-separated, or `*` for all features in `cmd/service/features.go`
- `--description` — optional text stored in `bundles.yaml`
- `--gen` — regenerate every bundle's `features.go` from `bundles.yaml`
- `--remove` — delete a bundle from `bundles.yaml` and remove `cmd/<name>/`

## Side effects

- Updates `bundles.yaml` at the Go workspace root.
- Creates `cmd/<name>/main.go` (delegates to `internal/runner`) and selective `features.go`.
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
