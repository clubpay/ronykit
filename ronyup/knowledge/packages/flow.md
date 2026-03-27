---
import_path: github.com/clubpay/ronykit/flow
short_name: flow
---
Type-safe workflow orchestration over Temporal using generics for workflows, activities, signals, channels, futures, and shared state.

## Usage Hint

Use a single `*flow.SDK` per service.

Initialize it through `flow.NewBackend` + `flow.NewSDK` in datasource wiring,
then call `InitWithState(app)` and `Start()` from service lifecycle hooks.

For implementation patterns:
- Define workflows/activities as package-level vars with explicit names (`Feature/Action`) and typed request/response structs.
- Keep workflow functions orchestration-only: sequence activities, timers, selectors, and signals; keep business I/O in activities.
- Access dependencies in activities via typed state (`ctx.S()`) instead of globals/singletons.
- Apply retries on remote/network activities with `flow.ExecuteActivityOptions{RetryPolicy: ...}` and keep activity inputs idempotent.
- Use deterministic waiting with `ctx.Selector()` + `flow.SelectorAddReceive`/`flow.SelectorAddFuture` for timer-or-signal race patterns.
- For long-running loops, periodically rotate history with `ctx.ContinueAsNewError(req)`.
- Use stable workflow IDs that encode the business key (for example `Project/DataCollector/<token>`) and expose a helper for signal names.
