package flow

import (
	"context"
	"time"

	"github.com/clubpay/ronykit/kit/utils"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// ScheduleOverlapPolicy controls what happens when a workflow would be started
// by a schedule and is already running.
type ScheduleOverlapPolicy int32

const (
	ScheduleOverlapPolicyUnspecified ScheduleOverlapPolicy = 0
	// ScheduleOverlapPolicySkip (default) means don't start anything. When the
	// workflow completes, the next scheduled event after that time will be considered.
	ScheduleOverlapPolicySkip ScheduleOverlapPolicy = 1
	// ScheduleOverlapPolicyBufferOne means start the workflow again as soon as the
	// current one completes, but only buffer one start in this way. If another start is
	// supposed to happen when the workflow is running, and one is already buffered, then
	// only the first one will be started after the running workflow finishes.
	ScheduleOverlapPolicyBufferOne ScheduleOverlapPolicy = 2
	// ScheduleOverlapPolicyBufferAll means buffer up any number of starts to all
	// happen sequentially, immediately after the running workflow completes.
	ScheduleOverlapPolicyBufferAll ScheduleOverlapPolicy = 3
	// ScheduleOverlapPolicyCancelOther means that if there is another workflow
	// running, cancel it, and start the new one after the old one completes cancellation.
	ScheduleOverlapPolicyCancelOther ScheduleOverlapPolicy = 4
	// ScheduleOverlapPolicyTerminateOther means that if there is another workflow
	// running, terminate it and start the new one immediately.
	ScheduleOverlapPolicyTerminateOther ScheduleOverlapPolicy = 5
	// ScheduleOverlapPolicyAllowAll means to start any number of concurrent workflows.
	// Note that with this policy, the last completion result and
	// last failure will not be available since workflows are not sequential.
	ScheduleOverlapPolicyAllowAll ScheduleOverlapPolicy = 6
)

type CreateScheduleRequest struct {
	ID               string
	Action           ScheduleAction
	Spec             ScheduleSpec
	CatchupWindow    time.Duration
	RemainingActions int
	OverlapPolicy    ScheduleOverlapPolicy

	ExecutionTimeout time.Duration
	RunTimeout       time.Duration
	SearchAttributes SearchAttributes
	TimezoneName     string
}

type ScheduleAction struct {
	WorkflowIDPrefix string
	WorkflowName     string
	WorkflowArg      any
	SearchAttributes SearchAttributes
	RetryPolicy      *RetryPolicy
}

type ScheduleCalendarSpec struct {
	Second     int
	Minute     int
	Hour       int
	Month      int
	Year       int
	DayOfWeek  time.Weekday
	DayOfMonth int // between 1 and 31 inclusive
}

type ScheduleIntervalSpec struct {
	Period time.Duration
	Offset time.Duration
}

type ScheduleSpec struct {
	Calendars []ScheduleCalendarSpec
	Intervals []ScheduleIntervalSpec
	StartTime time.Time
	EndTime   time.Time
	Jitter    time.Duration
	Timezone  string
}

type (
	ScheduleEntry        = client.ScheduleListEntry
	ScheduleActionResult = client.ScheduleActionResult
	ScheduleHandle       = client.ScheduleHandle
	ScheduleListIterator = client.ScheduleListIterator
)

func (sc ScheduleSpec) toScheduleSpec() client.ScheduleSpec {
	calSpec := utils.Map(
		func(src ScheduleCalendarSpec) client.ScheduleCalendarSpec {
			cal := client.ScheduleCalendarSpec{}
			if src.Second != 0 {
				cal.Second = []client.ScheduleRange{{Start: src.Second}}
			}
			if src.Minute != 0 {
				cal.Minute = []client.ScheduleRange{{Start: src.Minute}}
			}
			if src.Hour != 0 {
				cal.Hour = []client.ScheduleRange{{Start: src.Hour}}
			}
			if src.Month != 0 {
				cal.Month = []client.ScheduleRange{{Start: src.Month}}
			}
			if src.Year != 0 {
				cal.Year = []client.ScheduleRange{{Start: src.Year}}
			}
			if src.DayOfWeek != 0 {
				cal.DayOfWeek = []client.ScheduleRange{{Start: int(src.DayOfWeek)}}
			}
			if src.DayOfMonth != 0 {
				cal.DayOfMonth = []client.ScheduleRange{{Start: src.DayOfMonth}}
			}

			return cal
		},
		sc.Calendars,
	)
	intervalSpec := utils.Map(
		func(src ScheduleIntervalSpec) client.ScheduleIntervalSpec {
			return client.ScheduleIntervalSpec{
				Every:  src.Period,
				Offset: src.Offset,
			}
		},
		sc.Intervals,
	)

	out := client.ScheduleSpec{
		Calendars:    calSpec,
		Intervals:    intervalSpec,
		StartAt:      sc.StartTime,
		EndAt:        sc.EndTime,
		Jitter:       sc.Jitter,
		TimeZoneName: sc.Timezone,
	}

	return out
}

func (sdk *SDK) CreateSchedule(ctx context.Context, req CreateScheduleRequest) (ScheduleHandle, error) {
	opt := client.ScheduleOptions{
		ID:   req.ID,
		Spec: req.Spec.toScheduleSpec(),
		Action: &client.ScheduleWorkflowAction{
			Workflow:                 req.Action.WorkflowName,
			Args:                     []any{req.Action.WorkflowArg},
			TaskQueue:                sdk.b.taskQ,
			WorkflowExecutionTimeout: req.ExecutionTimeout,
			WorkflowRunTimeout:       req.RunTimeout,
			TypedSearchAttributes:    req.Action.SearchAttributes,
			RetryPolicy:              req.Action.RetryPolicy,
		},
		Overlap:               enumspb.ScheduleOverlapPolicy(req.OverlapPolicy),
		CatchupWindow:         req.CatchupWindow,
		RemainingActions:      req.RemainingActions,
		TypedSearchAttributes: req.SearchAttributes,
	}

	return sdk.b.cli.ScheduleClient().Create(ctx, opt)
}

func (sdk *SDK) GetSchedule(ctx context.Context, id string) ScheduleHandle {
	return sdk.b.cli.ScheduleClient().GetHandle(ctx, id)
}

func (sdk *SDK) ListSchedules(ctx context.Context, query string, pageSize int) (ScheduleListIterator, error) {
	iter, err := sdk.b.cli.ScheduleClient().List(
		ctx,
		client.ScheduleListOptions{
			PageSize: pageSize,
			Query:    query,
		},
	)
	if err != nil {
		return nil, err
	}

	return iter, nil
}

func (sdk *SDK) DeleteSchedule(ctx context.Context, id string) error {
	return sdk.b.cli.ScheduleClient().GetHandle(ctx, id).Delete(ctx)
}

func (sdk *SDK) TogglePause(ctx context.Context, id string, pause bool) error {
	if pause {
		return sdk.b.cli.ScheduleClient().GetHandle(ctx, id).Pause(ctx, client.SchedulePauseOptions{})
	}

	return sdk.b.cli.ScheduleClient().GetHandle(ctx, id).Unpause(ctx, client.ScheduleUnpauseOptions{})
}

func (sdk *SDK) Trigger(ctx context.Context, id string) error {
	return sdk.b.cli.ScheduleClient().
		GetHandle(ctx, id).
		Trigger(
			ctx,
			client.ScheduleTriggerOptions{},
		)
}
