# Bundled runtime config

When features are scaffolded with `ronyup setup feature`, a bundle-ready config
file is created here:

```
config/<kind>/<service>.local.yaml
```

For example, scaffolding `auth` with `--template service` creates
`config/service/auth.local.yaml`.

The all-in-one `cmd/service` entrypoint loads these files via `x/di` (override
the base directory with `--config-dir`, default `./config`). Standalone feature
modules still use `internal/settings/config.local.yaml` when run via `App()`.
