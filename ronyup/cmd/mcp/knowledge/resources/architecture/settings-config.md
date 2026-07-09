Define settings in `internal/settings/settings.go` as a typed struct with nested config blocks (`DBConfig`, `RedisConfig`, etc.) using `settings` struct tags.

Define `internal/settings/const.go` with:

- `ModuleName` (for example `"core/auth"`),
- `ModuleVersion`,
- and module-specific constants like `RedisPrefix`.

Expose package-level `ConfigName` and `ConfigPaths` variables (default to `"config.local"` and local paths).

Provide a constructor:

- `New(set settings.Settings) (*Settings, error)` that calls `set.SetFromFile(ConfigName, ConfigPaths...)`,
- then `set.Unmarshal(modSettings)`.

In the module root, expose:

- `LoadSettings(filename string, searchPaths ...string)` that sets `settings.ConfigName` and `settings.ConfigPaths`.

## Standalone vs bundled config

Two layouts exist:

- **Standalone** (running a feature module directly via `App()`): defaults in `internal/settings` apply — `ConfigName` is `config.local`, search paths are `.` and the module directory, and the YAML file lives at `internal/settings/config.local.yaml`.
- **Bundled** (all-in-one `cmd/service` or a custom bundle entrypoint): `x/di` calls `LoadSettings` before the fx graph starts. The config file name is `<service>.local` (last path segment of `ModuleName`, lowercased) and the search path is `<config-root>/<kind>/` (for example `config/service/auth.local` when `ModuleName` is `feature/auth` and `Kind` is `service`).

Override the bundled base directory with `di.SetConfigRoot(dir)` in the bundle entrypoint before `fx.New`, or pass `--config-dir` on the scaffolded `cmd/service` binary (default `./config`).

`ronyup setup feature` also writes the bundle-ready file at `config/<kind>/<service>.local.yaml` (for example `config/service/auth.local.yaml`) so the all-in-one entrypoint can load it without manual copying.
