package flow

import (
	"context"
	"time"

	"github.com/clubpay/ronykit/x/rkit"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

// ActivityIDReusePolicy controls reuse of an activity ID from a closed activity.
type ActivityIDReusePolicy = enumspb.ActivityIdReusePolicy

const (
	ActivityIDReusePolicyUnspecified = enumspb.ACTIVITY_ID_REUSE_POLICY_UNSPECIFIED
	// ActivityIDReusePolicyAllowDuplicate allows a new activity with an ID from a closed activity.
	ActivityIDReusePolicyAllowDuplicate = enumspb.ACTIVITY_ID_REUSE_POLICY_ALLOW_DUPLICATE
	// ActivityIDReusePolicyAllowDuplicateFailedOnly allows reuse when the last activity
	// failed, was canceled, was terminated, or timed out.
	ActivityIDReusePolicyAllowDuplicateFailedOnly = enumspb.ACTIVITY_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY
	// ActivityIDReusePolicyRejectDuplicate keeps the activity ID closed for future starts.
	ActivityIDReusePolicyRejectDuplicate = enumspb.ACTIVITY_ID_REUSE_POLICY_REJECT_DUPLICATE
)

// ActivityIDConflictPolicy controls a start that uses the ID of a running activity.
type ActivityIDConflictPolicy = enumspb.ActivityIdConflictPolicy

const (
	ActivityIDConflictPolicyUnspecified = enumspb.ACTIVITY_ID_CONFLICT_POLICY_UNSPECIFIED
	// ActivityIDConflictPolicyFail rejects the start when that ID is still running.
	ActivityIDConflictPolicyFail = enumspb.ACTIVITY_ID_CONFLICT_POLICY_FAIL
	// ActivityIDConflictPolicyUseExisting returns a handle for the running activity.
	ActivityIDConflictPolicyUseExisting = enumspb.ACTIVITY_ID_CONFLICT_POLICY_USE_EXISTING
)

// StartActivityOptions starts an activity without a parent workflow.
// ID is generated when empty. TaskQueue defaults to the activity backend.
type StartActivityOptions struct {
	ID                       string
	TaskQueue                string
	ScheduleToCloseTimeout   time.Duration
	ScheduleToStartTimeout   time.Duration
	StartToCloseTimeout      time.Duration
	HeartbeatTimeout         time.Duration
	StartDelay               time.Duration
	RetryPolicy              *RetryPolicy
	ActivityIDReusePolicy    ActivityIDReusePolicy
	ActivityIDConflictPolicy ActivityIDConflictPolicy
	SearchAttributes         SearchAttributes
	Summary                  string
	StaticDetails            string
}

// ActivityRun is a handle to a standalone activity execution.
type ActivityRun[RES any] struct {
	handle client.ActivityHandle

	ID    string
	RunID string
}

// Get waits for the activity to finish and returns its result.
func (r ActivityRun[RES]) Get(ctx context.Context) (*RES, error) {
	var out RES

	err := r.handle.Get(ctx, &out)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

// Cancel requests cancellation of the activity.
func (r ActivityRun[RES]) Cancel(ctx context.Context, reason string) error {
	return r.handle.Cancel(ctx, client.CancelActivityOptions{Reason: reason})
}

// Terminate stops the activity without running its cancellation path.
func (r ActivityRun[RES]) Terminate(ctx context.Context, reason string) error {
	return r.handle.Terminate(ctx, client.TerminateActivityOptions{Reason: reason})
}

// ExecuteStandalone schedules the activity on a task queue outside a workflow.
func (a *Activity[REQ, RES, STATE]) ExecuteStandalone(
	ctx context.Context, req REQ, opts StartActivityOptions,
) (*ActivityRun[RES], error) {
	backend := mustBackend(a.backend, "activity", a.Name)

	handle, err := backend.Client().ExecuteActivity(
		ctx,
		toClientStartActivityOptions(opts, backend.TaskQueue()),
		a.Name,
		req,
	)
	if err != nil {
		return nil, err
	}

	return &ActivityRun[RES]{
		handle: handle,
		ID:     handle.GetID(),
		RunID:  handle.GetRunID(),
	}, nil
}

// Handle points at an existing standalone activity. An empty runID selects the latest run.
func (a *Activity[REQ, RES, STATE]) Handle(id, runID string) ActivityRun[RES] {
	backend := mustBackend(a.backend, "activity", a.Name)
	handle := backend.Client().GetActivityHandle(client.GetActivityHandleOptions{
		ActivityID: id,
		RunID:      runID,
	})

	return ActivityRun[RES]{
		handle: handle,
		ID:     handle.GetID(),
		RunID:  handle.GetRunID(),
	}
}

func (a *ActivityFactory[REQ, RES, STATE]) ExecuteStandalone(
	ctx context.Context, req REQ, opts StartActivityOptions,
) (*ActivityRun[RES], error) {
	return a.mustAct().ExecuteStandalone(ctx, req, opts)
}

func (a *ActivityFactory[REQ, RES, STATE]) Handle(id, runID string) ActivityRun[RES] {
	return a.mustAct().Handle(id, runID)
}

func toClientStartActivityOptions(opts StartActivityOptions, taskQueue string) client.StartActivityOptions {
	timeouts := applyActivityTimeouts(ExecuteActivityOptions{
		ScheduleToCloseTimeout: opts.ScheduleToCloseTimeout,
		ScheduleToStartTimeout: opts.ScheduleToStartTimeout,
		StartToCloseTimeout:    opts.StartToCloseTimeout,
		HeartbeatTimeout:       opts.HeartbeatTimeout,
		RetryPolicy:            opts.RetryPolicy,
		TaskQueue:              opts.TaskQueue,
	})

	return client.StartActivityOptions{
		ID:                       rkit.Coalesce(opts.ID, rkit.RandomID(16)),
		TaskQueue:                rkit.Coalesce(timeouts.TaskQueue, taskQueue),
		ScheduleToCloseTimeout:   timeouts.ScheduleToCloseTimeout,
		ScheduleToStartTimeout:   timeouts.ScheduleToStartTimeout,
		StartToCloseTimeout:      timeouts.StartToCloseTimeout,
		HeartbeatTimeout:         timeouts.HeartbeatTimeout,
		StartDelay:               opts.StartDelay,
		RetryPolicy:              opts.RetryPolicy,
		ActivityIDReusePolicy:    opts.ActivityIDReusePolicy,
		ActivityIDConflictPolicy: opts.ActivityIDConflictPolicy,
		TypedSearchAttributes:    opts.SearchAttributes,
		Summary:                  opts.Summary,
		StaticDetails:            opts.StaticDetails,
	}
}
