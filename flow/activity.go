package flow

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

type ActivityFunc[REQ, RES, STATE any] func(ctx *ActivityContext[REQ, RES, STATE], req REQ) (*RES, error)

func NewActivity[REQ, RES, STATE any](
	name string,
	fn ActivityFunc[REQ, RES, STATE],
) Activity[REQ, RES, STATE] {
	return Activity[REQ, RES, STATE]{
		Name: name,
		Fn:   fn,
	}
}

type Activity[REQ, RES, STATE any] struct {
	sdk  *SDK
	Name string
	Fn   ActivityFunc[REQ, RES, STATE]
}

func (a *Activity[REQ, RES, STATE]) Init(sdk *SDK) {
	a.sdk = sdk
	sdk.w.RegisterActivityWithOptions(
		func(ctx context.Context, req REQ) (*RES, error) {
			fCtx := &ActivityContext[REQ, RES, STATE]{
				ctx: ctx,
			}

			return a.Fn(fCtx, req)
		},
		activity.RegisterOptions{Name: a.Name, SkipInvalidStructFunctions: true},
	)
}

func (a *Activity[REQ, RES, STATE]) InitWithState(sdk *SDK, state STATE) {
	a.sdk = sdk
	sdk.w.RegisterActivityWithOptions(
		func(ctx context.Context, req REQ) (*RES, error) {
			fCtx := &ActivityContext[REQ, RES, STATE]{
				ctx: ctx,
				s:   state,
			}

			return a.Fn(fCtx, req)
		},
		activity.RegisterOptions{Name: a.Name, SkipInvalidStructFunctions: true},
	)
}

type ExecuteActivityOptions struct {
	// ScheduleToCloseTimeout - Total time that a workflow is willing to wait for an Activity to complete.
	// ScheduleToCloseTimeout limits the total time of an Activity's execution, including retries
	// 		(use StartToCloseTimeout to limit the time of a single attempt).
	// The zero value of this uses the default value.
	// Either this option or StartToCloseTimeout is required: Defaults to unlimited.
	ScheduleToCloseTimeout time.Duration

	// ScheduleToStartTimeout - Time that the Activity Task can stay in the Task Queue before it is picked up by
	// a Worker. Do not specify this timeout unless using host-specific Task Queues for Activity Tasks are being
	// used for routing. In almost all situations that don't involve routing activities to specific hosts, it is
	// better to rely on the default value.
	// ScheduleToStartTimeout is always non-retryable. Retrying after this timeout doesn't make sense, as it would
	// just put the Activity Task back into the same Task Queue.
	// Optional: Defaults to unlimited.
	ScheduleToStartTimeout time.Duration

	// StartToCloseTimeout - Maximum time of a single Activity execution attempt.
	// Note that the Temporal Server doesn't detect Worker process failures directly. It relies on this timeout
	// to detect that an Activity that didn't complete on time. So this timeout should be as short as the longest
	// possible execution of the Activity body. Potentially long-running Activities must specify HeartbeatTimeout
	// and call Activity.RecordHeartbeat(ctx, "my-heartbeat") periodically for timely failure detection.
	// Either this option or ScheduleToCloseTimeout is required: Defaults to the ScheduleToCloseTimeout value.
	StartToCloseTimeout time.Duration
	RetryPolicy         *RetryPolicy
}

func (a *Activity[REQ, RES, InitArg]) Execute(ctx Context, req REQ, opts ExecuteActivityOptions) Future[RES] {
	if opts.StartToCloseTimeout == 0 {
		opts.StartToCloseTimeout = time.Minute
	}
	if opts.ScheduleToCloseTimeout == 0 {
		opts.ScheduleToCloseTimeout = time.Hour * 24
	}
	ctx = workflow.WithActivityOptions(
		ctx,
		workflow.ActivityOptions{
			TaskQueue:              a.sdk.taskQ,
			ScheduleToCloseTimeout: opts.ScheduleToCloseTimeout,
			ScheduleToStartTimeout: opts.ScheduleToStartTimeout,
			StartToCloseTimeout:    opts.StartToCloseTimeout,
			RetryPolicy:            opts.RetryPolicy,
		},
	)

	return Future[RES]{
		f: workflow.ExecuteActivity(ctx, a.Name, req),
	}
}

func (a *Activity[REQ, RES, InitArg]) ExecuteLocal(ctx Context, req REQ, opts ExecuteActivityOptions) Future[RES] {
	if opts.StartToCloseTimeout == 0 {
		opts.StartToCloseTimeout = time.Minute
	}
	if opts.ScheduleToCloseTimeout == 0 {
		opts.ScheduleToCloseTimeout = time.Hour * 24
	}
	ctx = workflow.WithLocalActivityOptions(
		ctx,
		workflow.LocalActivityOptions{
			ScheduleToCloseTimeout: opts.ScheduleToCloseTimeout,
			StartToCloseTimeout:    opts.StartToCloseTimeout,
			RetryPolicy:            opts.RetryPolicy,
		},
	)

	return Future[RES]{
		f: workflow.ExecuteLocalActivity(ctx, a.Name, req),
	}
}

func ExecuteActivity[REQ, RES, IA any](
	ctx Context, act Activity[REQ, RES, IA], req REQ, opts ExecuteActivityOptions,
) Future[RES] {
	return act.Execute(ctx, req, opts)
}

func ExecuteActivityLocal[REQ, RES, IA any](
	ctx Context, act Activity[REQ, RES, IA], req REQ, opts ExecuteActivityOptions,
) Future[RES] {
	return act.ExecuteLocal(ctx, req, opts)
}
