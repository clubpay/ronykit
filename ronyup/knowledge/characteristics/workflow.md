---
keywords:
  - workflow
  - temporal
  - orchestration
  - long running
  - saga
  - async
applies_to_files:
  - app
  - settings
---
When a feature requires durable orchestration, model it with `flow`: keep workflows deterministic and move external I/O into typed activities. Register workflows/activities at package level and wire dependencies through typed app state passed via `InitWithState`.

## File-Level Hint

In `internal/app`, separate workflow orchestration files (`*_workflow.go`) from activity implementation files (`workflow_activities.go`). Use `Selector` + timers/signals for wait conditions, retries for remote activities, and `ContinueAsNew` for perpetual collectors.
