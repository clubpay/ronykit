---
name: write-workflow
description: Guide an AI agent through writing durable workflow jobs using the RonyKIT flow module (Temporal-based).
arguments:
  - name: workflow_name
    description: The name of the workflow to implement (e.g. "OrderFulfillment", "DataCollector").
    required: true
  - name: description
    description: A brief description of what the workflow orchestrates.
    required: true
  - name: service_name
    description: The service module that owns this workflow (without "mod" suffix).
    required: false
---
You are writing a durable workflow called "{{workflow_name}}" using the RonyKIT `flow` package.

Workflow description: {{description}}

{{#if service_name}}
This workflow belongs to the "{{service_name}}mod" service.
{{/if}}

## Core Concepts

The `flow` package (`github.com/clubpay/ronykit/flow`) provides type-safe generics wrappers over Temporal:

- **Workflow[REQ, RES, STATE]** — orchestration logic (deterministic, no side effects)
- **Activity[REQ, RES, STATE]** — side-effect execution (DB writes, HTTP calls, external APIs)
- **WorkflowContext** — provides timers, selectors, signals, child workflows, continue-as-new
- **ActivityContext** — provides logger, heartbeat, state access via `ctx.S()`

## Strict Split Rule

- **Workflow files**: orchestrate only — sequence activities, timers, selectors, child workflows, continue-as-new. NO direct I/O.
- **Activity files**: execute side effects — DB writes, HTTP/gRPC calls, stub calls, Kafka publishes. Access dependencies via `ctx.S()`.

## Defining a Workflow

```go
var {{workflow_name}} = flow.NewWorkflow[MyRequest, MyResponse, *app.App](
    "{{service_name}}/{{workflow_name}}", "{{service_name}}",
    func(ctx *flow.WorkflowContext[MyRequest, MyResponse, *app.App], req MyRequest) (*MyResponse, error) {
        // Execute activities in sequence
        result1, err := DoStepOne.Execute(ctx.Context(), req.Input, flow.ExecuteActivityOptions{
            StartToCloseTimeout: 30 * time.Second,
        }).Get(ctx.Context())
        if err != nil {
            return nil, err
        }

        // Use timers for delays
        _ = ctx.Sleep(5 * time.Minute)

        // Execute more activities
        result2, err := DoStepTwo.Execute(ctx.Context(), result1.Output, flow.ExecuteActivityOptions{
            StartToCloseTimeout: 60 * time.Second,
            RetryPolicy: &flow.RetryPolicy{
                MaximumAttempts: 3,
            },
        }).Get(ctx.Context())
        if err != nil {
            return nil, err
        }

        return &MyResponse{Result: result2.Value}, nil
    },
)
```

## Defining Activities

```go
var DoStepOne = flow.NewActivity[StepOneReq, StepOneRes, *app.App](
    "{{service_name}}/DoStepOne", "{{service_name}}",
    func(ctx *flow.ActivityContext[StepOneReq, StepOneRes, *app.App], req StepOneReq) (*StepOneRes, error) {
        // Access service dependencies via typed state
        db := ctx.S().AccountRepo
        result, err := db.GetAccount(ctx.Context(), req.AccountID)
        if err != nil {
            return nil, err
        }
        return &StepOneRes{Account: result}, nil
    },
)
```

## SDK Wiring (in datasource/module wiring)

```go
backend, err := flow.NewBackend(flow.BackendConfig{
    HostPort:  cfg.TemporalHostPort,
    Namespace: cfg.TemporalNamespace,
    TaskQueue: cfg.TemporalTaskQueue,
})

sdk := flow.NewSDK(flow.SDKConfig{
    DefaultBackend: backend,
    Logger:         logger,
})

// Register workflows/activities and inject shared state
sdk.InitWithState(app)

// Start worker in service lifecycle Start()
sdk.Start()

// Stop in Shutdown()
sdk.Stop()
```

## Advanced Patterns

### Signals (external wakeups)

```go
var ApprovalSignal = flow.NewSignal[ApprovalPayload]("approval")

// In workflow: wait for signal
ch := ApprovalSignal.GetChannel(ctx.Context())
sel := ctx.Selector()
flow.SelectorAddReceive(sel, ch, func(payload ApprovalPayload) {
    // handle approval
})
sel.Select(ctx.Context())
```

### Timer-or-Signal races

```go
sel := ctx.Selector()
timerFuture := ctx.Timer(24 * time.Hour)
flow.SelectorAddFuture(sel, timerFuture, func() { /* timeout path */ })
flow.SelectorAddReceive(sel, signalCh, func(val T) { /* signal path */ })
sel.Select(ctx.Context())
```

### Continue-as-new (for long-running loops)

```go
if ctx.GetVersion("v1", workflow.DefaultVersion, 1) == 1 {
    return nil, ctx.ContinueAsNewError(updatedReq)
}
```

### Child workflows

```go
run := ChildWorkflow.ExecuteAsChild(ctx.Context(), childReq, flow.ExecuteChildWorkflowOptions{
    WorkflowID: fmt.Sprintf("{{service_name}}/Child/%s", id),
})
result, err := run.Get(ctx.Context())
```

## Naming Conventions

- Workflow/activity names: `Feature/Action` (e.g. `"ledger/ProcessTransfer"`)
- Workflow IDs: deterministic from business keys (e.g. `"ledger/ProcessTransfer/<transferID>"`)
- Signal names: descriptive lowercase (e.g. `"approval"`, `"cancellation"`)
- Groups: service name without "mod" suffix

## Checklist

1. Define request/response structs for the workflow and each activity.
2. Define activities as package-level vars with `flow.NewActivity`.
3. Define the workflow as a package-level var with `flow.NewWorkflow`.
4. Keep workflows deterministic — no direct I/O, no time.Now(), no random.
5. Access dependencies in activities via `ctx.S()`.
6. Wire the SDK in datasource module, call `InitWithState(app)`.
7. Add retry policies to activities that call external services.
8. Use stable, business-key-based workflow IDs.
