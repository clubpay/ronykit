package flow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/clubpay/ronykit/x/rkit"

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
	// TaskQueue - Task queue to use for this activity. If not, set default task queue of SDK will be used.
	TaskQueue string
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
	DaysOfWeek []time.Weekday
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
	calSpec := rkit.Map(
		sc.Calendars,
		func(src ScheduleCalendarSpec) client.ScheduleCalendarSpec {
			cal := client.ScheduleCalendarSpec{
				Second: []client.ScheduleRange{{Start: src.Second}},
				Minute: []client.ScheduleRange{{Start: src.Minute}},
				Hour:   []client.ScheduleRange{{Start: src.Hour}},
			}

			if src.Month != 0 {
				cal.Month = []client.ScheduleRange{{Start: src.Month}}
			}

			if src.Year != 0 {
				cal.Year = []client.ScheduleRange{{Start: src.Year}}
			}

			switch {
			case len(src.DaysOfWeek) > 0:
				cal.DayOfWeek = rkit.Map(src.DaysOfWeek, func(d time.Weekday) client.ScheduleRange {
					return client.ScheduleRange{Start: int(d)}
				})
			case src.DayOfWeek != 0:
				cal.DayOfWeek = []client.ScheduleRange{{Start: int(src.DayOfWeek)}}
			}

			if src.DayOfMonth != 0 {
				cal.DayOfMonth = []client.ScheduleRange{{Start: src.DayOfMonth}}
			}

			return cal
		},
	)
	intervalSpec := rkit.Map(
		sc.Intervals,
		func(src ScheduleIntervalSpec) client.ScheduleIntervalSpec {
			return client.ScheduleIntervalSpec{
				Every:  src.Period,
				Offset: src.Offset,
			}
		},
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
	spec := req.Spec.toScheduleSpec()
	if spec.TimeZoneName == "" {
		spec.TimeZoneName = req.TimezoneName
	}

	opt := client.ScheduleOptions{
		ID:   req.ID,
		Spec: spec,
		Action: &client.ScheduleWorkflowAction{
			ID:                       req.Action.WorkflowIDPrefix,
			Workflow:                 req.Action.WorkflowName,
			Args:                     []any{req.Action.WorkflowArg},
			TaskQueue:                rkit.Coalesce(req.TaskQueue, sdk.b.TaskQueue()),
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

	return sdk.b.ScheduleClient().Create(ctx, opt)
}

func (sdk *SDK) GetSchedule(ctx context.Context, id string) ScheduleHandle {
	return sdk.b.ScheduleClient().GetHandle(ctx, id)
}

func (sdk *SDK) ListSchedules(ctx context.Context, query string, pageSize int) (ScheduleListIterator, error) {
	iter, err := sdk.b.ScheduleClient().List(
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
	return sdk.b.ScheduleClient().GetHandle(ctx, id).Delete(ctx)
}

func (sdk *SDK) TogglePause(ctx context.Context, id string, pause bool) error {
	if pause {
		return sdk.b.ScheduleClient().GetHandle(ctx, id).Pause(ctx, client.SchedulePauseOptions{})
	}

	return sdk.b.ScheduleClient().GetHandle(ctx, id).Unpause(ctx, client.ScheduleUnpauseOptions{})
}

func (sdk *SDK) Trigger(ctx context.Context, id string) error {
	return sdk.b.ScheduleClient().
		GetHandle(ctx, id).
		Trigger(
			ctx,
			client.ScheduleTriggerOptions{},
		)
}

const (
	defaultMigrateRetryInterval = time.Minute
	defaultMigrateMaxRounds     = 10
)

type SchedulerMigrator struct {
	from Backend
	to   Backend

	retryInterval time.Duration
	maxRounds     int
}

func NewSchedulerMigrator(from, to Backend) *SchedulerMigrator {
	return &SchedulerMigrator{
		from:          from,
		to:            to,
		retryInterval: defaultMigrateRetryInterval,
		maxRounds:     defaultMigrateMaxRounds,
	}
}

// WithRetry configures how Migrate re-visits schedules that the check function skipped.
// Between rounds it waits interval; it gives up after rounds attempts and returns an error
// naming the schedules that were left on the source cluster.
func (s *SchedulerMigrator) WithRetry(interval time.Duration, rounds int) *SchedulerMigrator {
	s.retryInterval = interval
	s.maxRounds = rounds

	return s
}

type (
	MigrateCheckResult struct {
		// Ignore is TRUE, then do not copy the scheduler
		Ignore bool
	}
	MigrateCheckFunc func(ctx context.Context, sch *client.ScheduleListEntry) MigrateCheckResult
)

// Migrate copies every schedule from the source cluster to the destination cluster,
// optionally deleting it from the source afterwards. Schedules that checkFn marks as
// Ignore are retried on later rounds, so a schedule that is skipped because it is about
// to fire still migrates once it settles. Migrate stops when nothing is left to retry,
// when ctx is canceled, or when it runs out of rounds.
func (s *SchedulerMigrator) Migrate(
	ctx context.Context,
	deleteSource bool,
	checkFn MigrateCheckFunc,
) error {
	rounds := max(s.maxRounds, 1)

	for round := range rounds {
		ignored, err := s.migrateOnce(ctx, deleteSource, checkFn)
		if err != nil {
			return err
		}

		if len(ignored) == 0 {
			return nil
		}

		if round == rounds-1 {
			return fmt.Errorf(
				"flow: %d schedule(s) left on the source cluster after %d rounds: %s",
				len(ignored), rounds, strings.Join(ignored, ", "),
			)
		}

		if err = sleepCtx(ctx, s.retryInterval); err != nil {
			return err
		}
	}

	return nil
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// migrateOnce makes a single pass over the source schedules and returns the IDs that
// checkFn asked to skip.
func (s *SchedulerMigrator) migrateOnce(
	ctx context.Context,
	deleteSource bool,
	checkFn MigrateCheckFunc,
) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	var ignored []string

	it, err := s.from.ScheduleClient().List(
		ctx,
		client.ScheduleListOptions{
			PageSize: 100,
			Query:    "",
		},
	)
	if err != nil {
		return nil, err
	}

	schToCli := s.to.ScheduleClient()
	schFromCli := s.from.ScheduleClient()

	for it.HasNext() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		ent, err := it.Next()
		if err != nil {
			return nil, err
		}

		_, err = schToCli.GetHandle(ctx, ent.ID).Describe(ctx)
		if err == nil {
			if deleteSource {
				err = schFromCli.GetHandle(ctx, ent.ID).Delete(ctx)
				if err != nil {
					return nil, err
				}
			}

			continue
		}

		if checkFn != nil {
			res := checkFn(ctx, ent)
			if res.Ignore {
				ignored = append(ignored, ent.ID)

				continue
			}
		}

		fromSchDesc, err := schFromCli.GetHandle(ctx, ent.ID).Describe(ctx)
		if err != nil {
			return nil, err
		}

		opt := client.ScheduleOptions{
			ID:                    ent.ID,
			Action:                fromSchDesc.Schedule.Action,
			TypedSearchAttributes: fromSchDesc.TypedSearchAttributes,
		}
		if fromSchDesc.Schedule.Spec != nil {
			opt.Spec = *fromSchDesc.Schedule.Spec
		}

		if fromSchDesc.Schedule.Policy != nil {
			opt.Overlap = fromSchDesc.Schedule.Policy.Overlap
			opt.CatchupWindow = fromSchDesc.Schedule.Policy.CatchupWindow
			opt.PauseOnFailure = fromSchDesc.Schedule.Policy.PauseOnFailure
		}

		if fromSchDesc.Schedule.State != nil {
			opt.Note = fromSchDesc.Schedule.State.Note
			opt.Paused = fromSchDesc.Schedule.State.Paused
			opt.RemainingActions = fromSchDesc.Schedule.State.RemainingActions
		}

		_, err = schToCli.Create(ctx, opt)
		if err != nil {
			return nil, err
		}

		if deleteSource {
			err = schFromCli.GetHandle(ctx, ent.ID).Delete(ctx)
			if err != nil {
				return nil, err
			}
		}
	}

	return ignored, nil
}
