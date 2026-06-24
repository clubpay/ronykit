package intent

import "context"

// TaskState is the lifecycle state of a task execution.
type TaskState string

const (
	TaskStatePending   TaskState = "pending"
	TaskStateRunning   TaskState = "running"
	TaskStateWaiting   TaskState = "waiting"
	TaskStateCompleted TaskState = "completed"
	TaskStateFailed    TaskState = "failed"
	TaskStateCancelled TaskState = "cancelled"
)

// TaskHandle is a running or completed task instance.
type TaskHandle interface {
	ID() string
	Name() string
	State() TaskState
}

// TaskExecutor runs task lifecycles.
// A flow-backed implementation will provide durable state transitions later.
type TaskExecutor interface {
	Start(ctx context.Context, name string, input any) (TaskHandle, error)
	Get(ctx context.Context, id string) (TaskHandle, error)
	Signal(ctx context.Context, id string, signal any) error
	Cancel(ctx context.Context, id string) error
}
