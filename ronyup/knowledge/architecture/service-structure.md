Every service module follows a fixed `service.go` boilerplate.

- Package name is the service name suffixed with `mod`
  (for example `authmod`, `ledgermod`, `notifmod`).
- The `Service` struct holds `*api.Service` and `*fx.App`.

Expose:

- `App(opt ...fx.Option) *Service`, built via
  `fx.New(appModule, fx.Populate(&apiSvc), fx.Options(opt...))`.
- `Module(opt ...fx.Option) fx.Option`, which wraps `App()` in an `fx.Provide`
  with `fx.Lifecycle` hooks for start/shutdown.

Register the service in `init()` via `di.RegisterService[Service]` with:

- `Kind` (`"service"` or `"gateway"`),
- `Name` (derived from `settings.ModuleName`),
- `InitFn` (`LoadSettings`),
- `ModuleFn` (`Module`).

Delegate `Desc()` to `svc.api.Desc()`. Implement `Start(ctx)` and
`Shutdown(ctx)` by forwarding to `svc.fxApp.Start/Stop`.
