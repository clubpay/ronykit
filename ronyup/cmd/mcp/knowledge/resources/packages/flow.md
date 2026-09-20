---
import_path: github.com/clubpay/ronykit/flow
short_name: flow
---

Type-safe Temporal orchestration. The **only** allowed way to write durable workflows in service code. Never import `go.temporal.io/sdk`
(denied by depguard).

## Usage Hint

```go
backend, err := flow.NewBackend(flow.BackendConfig{
	HostPort: cfg.TemporalHostPort, Namespace: cfg.TemporalNamespace, TaskQueue: cfg.TemporalTaskQueue,
})
sdk := flow.NewSDK(flow.SDKConfig{DefaultBackend: backend, Logger: logger})
sdk.InitWithState(app)
sdk.Start() // service Start(); sdk.Stop() on Shutdown
```

- Workflows: `flow.NewWorkflow[REQ, RES, STATE](name, group, fn)` — orchestration only (timers, selectors, activities). No I/O.
- Activities: `flow.NewActivity[REQ, RES, STATE](name, group, fn)` — side effects via `ctx.S()`.
- Versioning: `ctx.GetVersion("v1", flow.DefaultVersion, 1)` — use `flow.DefaultVersion`, not `workflow.DefaultVersion`.
- Long loops: `return nil, ctx.ContinueAsNewError(updatedReq)`.

**When NOT:** request/response HTTP handlers, one-shot background jobs that are not durable, or anything that belongs in `internal/app`.
Use the `write-workflow` prompt for full snippets. Read `architecture/flow-workflows`.
