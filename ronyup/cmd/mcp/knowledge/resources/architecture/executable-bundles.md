# Executable bundles

Scaffolded workspaces compile feature modules into one or more executables.

## Layout

| Path | Purpose |
|------|---------|
| `bundles.yaml` | Build-time manifest: which features each executable includes |
| `internal/runner/` | Shared runtime bootstrap (cobra/fx/rony, middleware, health check) |
| `cmd/service/` | Default all-in-one **dev** binary (`services: ["*"]`) |
| `cmd/<name>/` | Optional production bundles (compile-time mix-and-match) |

## Runtime vs compile-time

| Layer | Mechanism | Example |
|-------|-----------|---------|
| **Compile-time** | `bundles.yaml` + selective `features.go` | `cmd/auth-api` includes only `feature/auth` |
| **Runtime** | `--service` flag or `SERVICES` env | Same binary, start subset at launch |

Service names match `settings.ModuleName` (typically `feature/<name>`).

## Commands

```bash
# Default dev binary (all compiled-in features)
make run
cd cmd/service && go run .

# Create a production bundle
ronyup setup bundle --name auth-api --services feature/auth,feature/session

# Regenerate every bundle's features.go from bundles.yaml
ronyup setup bundle --gen

# Runtime subset within a binary
go run . --service feature/auth
SERVICES=feature/auth,feature/user go run .

# Build a specific bundle
make build-bundle BUNDLE=auth-api
make run-bundle BUNDLE=auth-api
```

## Upgrading older workspaces

Workspaces created before bundles were introduced need a one-time migration:

```bash
ronyup setup migrate bundles
ronyup setup sync --only backend   # optional: Makefile bundle targets
```

See `knowledge://ronyup/tools/migrate_bundles`.

## Feature registration

- `ronyup setup feature` adds a blank import to `cmd/service/features.go`.
- Bundles in `bundles.yaml` that list the feature (or `"*"`) get their `cmd/<bundle>/features.go` refreshed automatically.
- Global middleware registers in `internal/runner` via `di.RegisterMiddleware`.
