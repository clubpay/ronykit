Define settings in `internal/settings/settings.go` as a typed struct with
nested config blocks (`DBConfig`, `RedisConfig`, etc.) using `settings` struct
tags.

Define `internal/settings/const.go` with:

- `ModuleName` (for example `"core/auth"`),
- `ModuleVersion`,
- and module-specific constants like `RedisPrefix`.

Expose package-level `ConfigName` and `ConfigPaths` variables (default to
`"config.local"` and local paths).

Provide a constructor:

- `New(set settings.Settings) (*Settings, error)` that calls
  `set.SetFromFile(ConfigName, ConfigPaths...)`,
- then `set.Unmarshal(modSettings)`.

In the module root, expose:

- `LoadSettings(filename string, searchPaths ...string)` that sets
  `settings.ConfigName` and `settings.ConfigPaths`.

Config YAML files live in `internal/settings/config.local.yaml`.
