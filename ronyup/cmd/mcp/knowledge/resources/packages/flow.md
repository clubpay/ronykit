---
import_path: github.com/clubpay/ronykit/flow
short_name: flow
---

Type-safe workflow orchestration over Temporal using generics for workflows, activities, signals, channels, futures, and shared state.

NEVER import `go.temporal.io/sdk/*` directly in service code. `flow` is the only sanctioned way to build durable/long-running orchestration
in RonyKIT. Going through `flow` (not the raw SDK) gives you: enforced workflow/activity split for determinism, typed request/response/state
generics, retry-policy ergonomics, and dependency injection via typed state (`ctx.S()`). Hand-wiring the Temporal SDK bypasses all of these
guarantees and breaks consistency across services. The scaffolded workspace's `.golangci.yml` denies `go.temporal.io/sdk` imports so
`make lint` fails on violations.

## Usage Hint

Use a single `*flow.SDK` per service.

Initialize it through `flow.NewBackend` + `flow.NewSDK` in datasource wiring, then call `InitWithState(app)` and `Start()` from service
lifecycle hooks.

For implementation patterns:

- Define workflows/activities as package-level vars with explicit names (`Feature/Action`) and typed request/response structs.
- Keep workflow functions orchestration-only: sequence activities, timers, selectors, and signals; keep business I/O in activities.
- Access dependencies in activities via typed state (`ctx.S()`) instead of globals/singletons.
- Apply retries on remote/network activities with `flow.ExecuteActivityOptions{RetryPolicy: ...}` and keep activity inputs idempotent.
- Use deterministic waiting with `ctx.Selector()` + `flow.SelectorAddReceive`/`flow.SelectorAddFuture` for timer-or-signal race patterns.
- For long-running loops, periodically rotate history with `ctx.ContinueAsNewError(req)`.
- Use stable workflow IDs that encode the business key (for example `Project/DataCollector/<token>`) and expose a helper for signal names.
