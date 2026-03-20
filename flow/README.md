# flow

`flow` is a type-safe, generics-based Go wrapper around the [Temporal](https://temporal.io) SDK.
It eliminates raw `interface{}` usage by parameterizing workflows, activities, signals, channels,
futures, and selectors with concrete request, response, and state types.

```
go get github.com/clubpay/ronykit/flow
```

## Why flow?

The stock Temporal Go SDK relies heavily on `interface{}` for inputs, outputs, and context values.
`flow` replaces those with Go generics so that:

- Workflow and activity signatures are checked at compile time.
- Shared application state (database pools, config, clients) is injected through a typed `STATE`
  parameter instead of global variables.
- Channels, futures, and selectors carry their payload type, removing the need for manual type
  assertions.
- Encrypted payloads, search attributes, and scheduler migration are available out of the box.

## Core Concepts

### SDK

`SDK` is the top-level entry point. It owns one or two `Backend` instances, registers all
workflows and activities, and starts the Temporal worker.

```go
backend, err := flow.NewBackend(flow.BackendConfig{
    HostPort:  "localhost:7233",
    Namespace: "my-namespace",
    Group:     "my-group",
    TaskQueue: "my-task-queue",
    Logger:    flow.NewZapAdapter(zapLogger),
})

sdk := flow.NewSDK(flow.SDKConfig{
    Logger:         flow.NewZapAdapter(zapLogger),
    DefaultBackend: backend,
})

sdk.Init()          // register workflows/activities that use flow.EMPTY as state
// sdk.InitWithState(myState)  // register workflows/activities that use a custom state type

err = sdk.Start()   // start the worker
defer sdk.Stop()
```

### Backend

`Backend` wraps a Temporal client and worker for a single cluster. It is created via
`NewBackend` and handles namespace auto-creation, TLS, custom `DataConverter`, and
`worker.Options` passthrough.

```go
backend, err := flow.NewBackend(flow.BackendConfig{
    HostPort:      "temporal.example.com:7233",
    Secure:        true,                          // enables TLS
    Namespace:     "production",
    Group:         "payments",
    TaskQueue:     "payment-tasks",
    DataConverter: flow.EncryptedDataConverter("my-secret-key"),
    Logger:        flow.NewZapAdapter(zapLogger),
})
```

### Workflow

A workflow is defined with three type parameters: `REQ` (input), `RES` (output), and
`STATE` (shared application state). Workflows self-register when constructed.

```go
var MyWorkflow = flow.NewWorkflow[MyRequest, MyResponse, flow.EMPTY](
    "MyWorkflow", "my-group",
    func(ctx *flow.WorkflowContext[MyRequest, MyResponse, flow.EMPTY], req MyRequest) (*MyResponse, error) {
        // Execute activities, use timers, handle signals, etc.
        result, err := myActivity.Execute(ctx.Context(), req, flow.ExecuteActivityOptions{
            StartToCloseTimeout: 30 * time.Second,
        }).Get(ctx.Context())
        if err != nil {
            return nil, err
        }

        return result, nil
    },
)
```

For workflows that don't return a result, use `NewWorkflowNoResult`:

```go
var FireAndForget = flow.NewWorkflowNoResult[MyRequest, flow.EMPTY](
    "FireAndForget", "my-group",
    func(ctx *flow.WorkflowContext[MyRequest, flow.EMPTY, flow.EMPTY], req MyRequest) error {
        // ...
        return nil
    },
)
```

**Executing a workflow** (from application code, outside a workflow):

```go
run, err := MyWorkflow.Execute(ctx, req, flow.ExecuteWorkflowOptions{
    ID:                       "order-123",
    WorkflowExecutionTimeout: 24 * time.Hour,
})

// Wait for the result
result, err := run.Get(ctx)
```

**Executing as a child workflow** (from within another workflow):

```go
future := ChildWorkflow.ExecuteAsChild(ctx.Context(), childReq, flow.ExecuteChildWorkflowOptions{
    WorkflowID:               "child-order-123",
    WorkflowExecutionTimeout: time.Hour,
})
result, err := future.Get(ctx.Context())
```

### Activity

Activities follow the same three-type-parameter pattern. They also self-register on
construction.

```go
var ProcessPayment = flow.NewActivity[PaymentReq, PaymentRes, flow.EMPTY](
    "ProcessPayment", "my-group",
    func(ctx *flow.ActivityContext[PaymentReq, PaymentRes, flow.EMPTY], req PaymentReq) (*PaymentRes, error) {
        // Use ctx.Context() for the standard context.Context
        // Use ctx.State() to access the typed shared state
        return &PaymentRes{Success: true}, nil
    },
)
```

**Executing an activity** from a workflow:

```go
future := ProcessPayment.Execute(ctx.Context(), paymentReq, flow.ExecuteActivityOptions{
    StartToCloseTimeout:    30 * time.Second,
    ScheduleToCloseTimeout: time.Hour,
    RetryPolicy: &flow.RetryPolicy{
        MaximumAttempts: 3,
    },
})
result, err := future.Get(ctx.Context())
```

**Local activities** run in the same process as the workflow, skipping the task queue:

```go
future := ProcessPayment.ExecuteLocal(ctx.Context(), req, flow.ExecuteActivityOptions{
    StartToCloseTimeout: 5 * time.Second,
})
```

#### Activity Factories

When an activity's implementation depends on runtime state (e.g., a database connection),
use `NewActivityFactory` or `ToActivityFactory`:

```go
var SendEmail = flow.NewActivityFactory[EmailReq, EmailRes, *AppState](
    "SendEmail", "my-group",
    func(s *AppState) flow.ActivityFunc[EmailReq, EmailRes, *AppState] {
        return func(ctx *flow.ActivityContext[EmailReq, EmailRes, *AppState], req EmailReq) (*EmailRes, error) {
            err := s.Mailer.Send(req.To, req.Subject, req.Body)
            return &EmailRes{Sent: err == nil}, err
        }
    },
)
```

### Shared State

The `STATE` type parameter threads application dependencies (DB pools, configs, HTTP clients)
into workflows and activities without globals.

```go
type AppState struct {
    DB     *sql.DB
    Mailer *mailer.Client
}

var myWorkflow = flow.NewWorkflowWithState[Req, Res, *AppState](
    "MyWorkflow", "my-group", &AppState{},
    func(ctx *flow.WorkflowContext[Req, Res, *AppState], req Req) (*Res, error) {
        // ctx.State() returns *AppState
        return nil, nil
    },
)

// At startup:
sdk.InitWithState(&AppState{DB: db, Mailer: mailer})
```

You can also retrieve the state inside a raw `context.Context` using the generic helper:

```go
state := flow.GetState[*AppState](ctx)
```

## Workflow Primitives

### Future

`Future[T]` is a type-safe wrapper around `workflow.Future`. Activity and child-workflow
executions return a future that is resolved asynchronously.

```go
f := myActivity.Execute(ctx.Context(), req, opts)

if f.IsReady() {
    result, err := f.Get(ctx.Context())
}
```

### Channel

`Channel[T]` provides type-safe, deterministic communication within a workflow.

```go
ch := flow.NewChannel[string](ctx.Context())

// Send (blocks until received)
ch.Send(ctx.Context(), "hello")

// Receive (blocks until sent)
value, more := ch.Receive(ctx.Context())

// Non-blocking variants
value, ok := ch.ReceiveAsync()
ch.SendAsync("world")
```

Buffered and named variants are available:

```go
flow.NewBufferedChannel[int](ctx.Context(), 10)
flow.NewNamedChannel[int](ctx.Context(), "my-channel")
flow.NewNamedBufferedChannel[int](ctx.Context(), "my-channel", 10)
```

### Signal

`Signal[T]` models external events sent to a running workflow.

```go
var OrderCancelled = flow.Signal[CancelPayload]{Name: "order-cancelled"}

// Inside a workflow — receive signals
ch := OrderCancelled.GetChannel(ctx.Context())
payload, more := ch.Receive(ctx.Context())

// From outside or another workflow — send a signal
OrderCancelled.Send(ctx.Context(), targetWorkflowID, CancelPayload{Reason: "user-request"})

// From application code (outside workflow context)
sdk.Signal(ctx, "workflow-id", "order-cancelled", CancelPayload{Reason: "timeout"})
```

### SignalChannel

`SignalChannel[T]` is a receive-only channel obtained from a `Signal`. It supports the same
`Receive`, `ReceiveAsync`, `ReceiveWithTimeout`, and `ReceiveAsyncWithMoreFlag` methods as
`Channel[T]`.

### Selector

`Selector` multiplexes across channels and futures, similar to Go's `select` statement but
deterministic for replay.

```go
sel := ctx.Selector()

flow.SelectorAddReceive(sel, signalCh, func(ch flow.SignalChannel[Payload], more bool) {
    val, _ := ch.Receive(ctx.Context())
    // handle val
})

flow.SelectorAddFuture(sel, activityFuture, func(f flow.Future[Result]) {
    result, err := f.Get(ctx.Context())
    // handle result
})

sel.Select(ctx.Context())
```

### WaitGroup

`WaitGroup` (aliased from `workflow.WaitGroup`) coordinates concurrent workflow goroutines.

```go
wg := ctx.WaitGroup()
wg.Add(2)

ctx.Go(func(gCtx flow.Context) {
    defer wg.Done(gCtx)
    // work...
})

ctx.Go(func(gCtx flow.Context) {
    defer wg.Done(gCtx)
    // work...
})

wg.Wait(ctx.Context())
```

### Timers and Sleep

```go
// Deterministic sleep
ctx.Sleep(5 * time.Minute)

// Timer as a future (can be used with Selector or cancelled)
timerFuture := ctx.Timer(10 * time.Minute)
```

### SideEffect and MutableSideEffect

Execute non-deterministic logic (e.g., UUID generation) safely within a workflow:

```go
id, err := flow.SideEffect(ctx.Context(), func() string {
    return uuid.New().String()
})

val, err := flow.MutableSideEffect(ctx.Context(), "config-flag", func() bool {
    return fetchFeatureFlag()
})
```

### Versioning

Use `GetVersion` to safely introduce backward-compatible changes to running workflows:

```go
v := ctx.GetVersion("change-id", flow.DefaultVersion, 1)
if v == 1 {
    // new logic
} else {
    // old logic
}
```

### ContinueAsNew

Restart a workflow with a fresh history to avoid unbounded event growth:

```go
return nil, ctx.ContinueAsNewError(newReq)
```

## Scheduling

The SDK provides a full scheduling API on top of Temporal's schedule primitives.

```go
// Create a schedule that runs every hour
handle, err := sdk.CreateSchedule(ctx, flow.CreateScheduleRequest{
    ID: "hourly-cleanup",
    Action: flow.ScheduleAction{
        WorkflowName: "CleanupWorkflow",
        WorkflowArg:  CleanupReq{MaxAge: 24 * time.Hour},
    },
    Spec: flow.ScheduleSpec{
        Intervals: []flow.ScheduleIntervalSpec{
            {Period: time.Hour},
        },
    },
    OverlapPolicy: flow.ScheduleOverlapPolicySkip,
})

// Calendar-based scheduling
handle, err := sdk.CreateSchedule(ctx, flow.CreateScheduleRequest{
    ID: "daily-report",
    Action: flow.ScheduleAction{
        WorkflowName: "DailyReportWorkflow",
        WorkflowArg:  ReportReq{},
    },
    Spec: flow.ScheduleSpec{
        Calendars: []flow.ScheduleCalendarSpec{
            {Hour: 9, Minute: 0},
        },
    },
})

// Manage schedules
sdk.TogglePause(ctx, "hourly-cleanup", true)    // pause
sdk.TogglePause(ctx, "hourly-cleanup", false)   // unpause
sdk.Trigger(ctx, "hourly-cleanup")              // trigger immediately
sdk.DeleteSchedule(ctx, "hourly-cleanup")       // delete
sdk.GetSchedule(ctx, "hourly-cleanup")          // get handle
iter, err := sdk.ListSchedules(ctx, "", 100)    // list all
```

### Schedule Overlap Policies

| Policy | Behavior |
|---|---|
| `ScheduleOverlapPolicySkip` | Skip if already running (default) |
| `ScheduleOverlapPolicyBufferOne` | Buffer one start |
| `ScheduleOverlapPolicyBufferAll` | Buffer all starts |
| `ScheduleOverlapPolicyCancelOther` | Cancel the running workflow, start new after cancellation |
| `ScheduleOverlapPolicyTerminateOther` | Terminate running, start immediately |
| `ScheduleOverlapPolicyAllowAll` | Allow concurrent executions |

## Searching and Inspecting Workflows

```go
// Search workflows with a visibility query
res, err := sdk.SearchWorkflows(ctx, flow.SearchWorkflowRequest{
    Query: flow.AND(
        flow.EQ(flow.WorkflowFilterWorkflowType, "MyWorkflow"),
        flow.EQ(flow.WorkflowFilterExecutionStatus, flow.ExecutionStatusRunning),
    ),
})

// Count workflows grouped by status
counts, err := sdk.CountWorkflows(ctx, flow.CountWorkflowRequest{
    Query: flow.EQ(flow.WorkflowFilterWorkflowType, "MyWorkflow"),
})
// counts.Total, counts.Counts["Running"], counts.Counts["Completed"], ...

// Get workflow history
history, err := sdk.GetWorkflowHistory(ctx, flow.GetWorkflowHistoryRequest{
    WorkflowID: "order-123",
})

// Describe a workflow execution
desc, err := sdk.DescribeWorkflowExecution(ctx, flow.DescribeWorkflowExecutionRequest{
    WorkflowID: "order-123",
})

// Cancel a workflow
sdk.CancelWorkflow(ctx, flow.CancelWorkflowRequest{WorkflowID: "order-123"})
```

### Workflow Filter Query Builder

Build visibility queries with a composable builder:

```go
q := flow.AND(
    flow.EQ(flow.WorkflowFilterWorkflowType, "OrderWorkflow"),
    flow.OR(
        flow.EQ(flow.WorkflowFilterExecutionStatus, flow.ExecutionStatusRunning),
        flow.EQ(flow.WorkflowFilterExecutionStatus, flow.ExecutionStatusFailed),
    ),
)
// Result: WorkflowType = 'OrderWorkflow' AND (ExecutionStatus = 'Running' OR ExecutionStatus = 'Failed')

flow.StartsWith(flow.WorkflowFilterWorkflowId, "order-")
flow.IN(flow.WorkflowFilterExecutionStatus, flow.ExecutionStatusRunning, flow.ExecutionStatusFailed)
flow.GT(flow.WorkflowFilterWorkflowId, "order-100")
flow.LT(flow.WorkflowFilterWorkflowId, "order-200")
```

## Search Attributes

Helpers for creating typed search attributes to attach to workflow executions:

```go
attrs := flow.NewSearchAttributes(
    flow.AttrString("customer", "acme-corp"),
    flow.AttrInt64("priority", 1),
    flow.AttrBool("express", true),
    flow.AttrKeyword("region", "us-east-1"),
    flow.AttrKeywords("tags", []string{"urgent", "vip"}),
)

run, err := MyWorkflow.Execute(ctx, req, flow.ExecuteWorkflowOptions{
    ID:               "order-123",
    SearchAttributes: attrs,
})
```

## Payload Encryption

`flow` includes an AES-encrypted data converter with zlib compression. Payloads stored in
Temporal are encrypted at rest.

```go
backend, err := flow.NewBackend(flow.BackendConfig{
    // ...
    DataConverter: flow.EncryptedDataConverter("my-32-byte-secret-key"),
})
```

### Codec Server

The `codecserver` sub-package exposes an HTTP service compatible with Temporal's
[Codec Server protocol](https://docs.temporal.io/production-deployment/data-encryption).
Point the Temporal Web UI at this endpoint to decrypt payloads for display.

```go
import "github.com/clubpay/ronykit/flow/codecserver"

svc := codecserver.NewService("/temporal-codec", map[string]string{
    "production": "my-secret-key",
    "*":          "fallback-key",  // wildcard namespace
})

// Register svc.Desc() with a RonyKit EdgeServer
```

## Cluster Migration

When migrating between Temporal clusters, the SDK supports a deprecating backend to keep old
workflows running while moving schedulers to the new cluster.

```go
sdk := flow.NewSDK(flow.SDKConfig{
    DefaultBackend:     newBackend,
    DeprecatingBackend: oldBackend,
})
```

On `Start()`, the SDK automatically migrates schedulers from the old backend to the new one.
Old workflows continue running on the deprecating backend until completion. Use
`SchedulerMigrator` directly for manual control:

```go
m := flow.NewSchedulerMigrator(oldBackend, newBackend)
err := m.Migrate(ctx, true, func(ctx context.Context, sch *flow.ScheduleEntry) flow.MigrateCheckResult {
    // Return Ignore: true to skip specific schedules
    return flow.MigrateCheckResult{}
})
```

## Logging

`flow` ships a Zap adapter that implements Temporal's `log.Logger` interface:

```go
logger := flow.NewZapAdapter(zap.Must(zap.NewProduction()))
```

## Error Utilities

```go
// Wrap errors with the current OpenTelemetry trace ID for correlation
err = flow.WrapError(ctx, err)

// Check if an error is a Temporal ApplicationError
ok, appErr := flow.IsApplicationError(err)
```

## Workflow ID Policies

### Reuse Policy (for completed workflows)

| Policy | Behavior |
|---|---|
| `WorkflowIdReusePolicyAllowDuplicate` | Always allow reuse |
| `WorkflowIdReusePolicyAllowDuplicateFailedOnly` | Allow reuse only if last run failed/cancelled/timed-out |
| `WorkflowIdReusePolicyRejectDuplicate` | Never allow reuse |
| `WorkflowIdReusePolicyTerminateIfRunning` | Allow reuse and terminate if still running |

### Conflict Policy (for running workflows)

| Policy | Behavior |
|---|---|
| `WorkflowIdConflictPolicyFail` | Return error (default) |
| `WorkflowIdConflictPolicyUseExisting` | Return handle to existing run |
| `WorkflowIdConflictPolicyTerminateExisting` | Terminate existing, start new |

## Requirements

- Go 1.25+
- A running Temporal server
