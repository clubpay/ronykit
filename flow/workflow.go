package flow

import (
	"context"
	"strings"
	"time"

	"github.com/clubpay/ronykit/kit/utils"
	enumspb "go.temporal.io/api/enums/v1"
	v112 "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

type WorkflowFunc[REQ, RES any] func(ctx *WorkflowContext[REQ, RES], req REQ) (*RES, error)

func NewWorkflow[REQ, RES, InitArg any](
	name string,
	factory func(initArg InitArg) WorkflowFunc[REQ, RES],
) Workflow[REQ, RES, InitArg] {
	return Workflow[REQ, RES, InitArg]{
		Name:    name,
		Factory: factory,
	}
}

type Workflow[REQ, RES, InitArg any] struct {
	sdk     *SDK
	Name    string
	Factory func(InitArg) WorkflowFunc[REQ, RES]
}

func (w *Workflow[REQ, RES, InitArg]) Init(sdk *SDK, initArg InitArg) {
	sdk.w.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, req REQ) (*RES, error) {
			return w.Factory(initArg)(
				&WorkflowContext[REQ, RES]{
					ctx: ctx,
				}, req,
			)
		},
		workflow.RegisterOptions{Name: w.Name},
	)

	sdk.replay.RegisterWorkflowWithOptions(
		func(ctx workflow.Context, req REQ) (*RES, error) {
			return w.Factory(initArg)(
				&WorkflowContext[REQ, RES]{
					ctx: ctx,
				}, req,
			)
		},
		workflow.RegisterOptions{Name: w.Name},
	)

	w.sdk = sdk
}

// WorkflowIdReusePolicy
// Defines whether to allow re-using a workflow id from a previously *closed* workflow.
// If the request is denied, a `WorkflowExecutionAlreadyStartedFailure` is returned.
//
// See `WorkflowIdConflictPolicy` for handling workflow id duplication with a *running* workflow.
type WorkflowIdReusePolicy int32

const (
	WorkflowIdReusePolicyUnspecified WorkflowIdReusePolicy = 0
	// WorkflowIdReusePolicyAllowDuplicate
	// Allow starting a workflow execution using the same workflow id.
	WorkflowIdReusePolicyAllowDuplicate WorkflowIdReusePolicy = 1
	// WorkflowIdReusePolicyAllowDuplicateFailedOnly
	// Allow starting a workflow execution using the same workflow id, only when the last
	// execution's final state is one of [terminated, cancelled, timed out, failed].
	WorkflowIdReusePolicyAllowDuplicateFailedOnly WorkflowIdReusePolicy = 2
	// WorkflowIdReusePolicyRejectDuplicate
	// Do not permit re-use of the workflow id for this workflow. Future start workflow requests
	// could potentially change the policy, allowing re-use of the workflow id.
	WorkflowIdReusePolicyRejectDuplicate WorkflowIdReusePolicy = 3
	// WorkflowIdReusePolicyTerminateIfRunning
	// This option belongs in WorkflowIdConflictPolicy but is here for backwards compatibility.
	// If specified, it acts like ALLOW_DUPLICATE, but also the WorkflowId*Conflict*Policy on
	// the request is treated as WorkflowIdConflictPolicyTerminateExisting.
	// If no running workflow, then the behavior is the same as ALLOW_DUPLICATE.
	WorkflowIdReusePolicyTerminateIfRunning WorkflowIdReusePolicy = 4
)

// WorkflowIdConflictPolicy
// Defines what to do when trying to start a workflow with the same workflow id as a *running* workflow.
// Note that it is *never* valid to have two actively running instances of the same workflow id.
//
// See `WorkflowIdReusePolicy` for handling workflow id duplication with a *closed* workflow.
type WorkflowIdConflictPolicy int32

const (
	WorkflowIdConflictPolicyUnspecified WorkflowIdConflictPolicy = 0
	// WorkflowIdConflictPolicyFail
	// Don't start a new workflow; instead return `WorkflowExecutionAlreadyStartedFailure`.
	WorkflowIdConflictPolicyFail WorkflowIdConflictPolicy = 1
	// WorkflowIdConflictPolicyUseExisting
	// Don't start a new workflow; instead return a workflow handle for the running workflow.
	WorkflowIdConflictPolicyUseExisting WorkflowIdConflictPolicy = 2
	// WorkflowIdConflictPolicyTerminateExisting
	// Terminate the running workflow before starting a new one.
	WorkflowIdConflictPolicyTerminateExisting WorkflowIdConflictPolicy = 3
)

type ExecuteWorkflowOptions struct {
	// ID – The business identifier of the workflow execution.
	// Optional: defaulted to an uuid.
	ID string
	// WorkflowExecutionTimeout – The timeout for the duration of workflow execution.
	// It includes retries and continues as new. Use WorkflowRunTimeout to limit the execution time
	// of a single workflow run.
	// The resolution is seconds.
	// Optional: defaulted to unlimited.
	WorkflowExecutionTimeout time.Duration

	// WorkflowRunTimeout – The timeout for the duration of a single workflow run.
	// The resolution is seconds.
	// Optional: defaulted to WorkflowExecutionTimeout.
	WorkflowRunTimeout time.Duration

	// WorkflowTaskTimeout – The timeout for processing a workflow task from the time the worker
	// pulled this task. If a workflow task is lost, it is retried after this timeout.
	// The resolution is seconds.
	// Optional: defaulted to 10 secs.
	WorkflowTaskTimeout time.Duration
	// StartDelay – Time to wait before dispatching the first workflow task.
	// If the workflow gets a signal before the delay, a workflow task will be dispatched and the rest
	// of the delay will be ignored. A signal from signal with start will not trigger a workflow task.
	// Cannot be set at the same time as a CronSchedule.
	StartDelay time.Duration
	// WorkflowIDReusePolicy
	// Specifies server behavior if a *completed* workflow with the same id exists.
	// This can be useful for dedupe logic if set to RejectDuplicate
	// Optional: defaulted to AllowDuplicate.
	WorkflowIDReusePolicy WorkflowIdReusePolicy

	// WorkflowIDConflictPolicy
	// Specifies server behavior if a *running* workflow with the same id exists.
	// This cannot be set if WorkflowIDReusePolicy is set to TerminateIfRunning.
	// Optional: defaulted to Fail.
	WorkflowIDConflictPolicy WorkflowIdConflictPolicy
}

type WorkflowRun[T any] struct {
	internal client.WorkflowRun
	ID       string
	RunID    string
}

func (x WorkflowRun[T]) Get(ctx context.Context) (*T, error) {
	var result T
	err := x.internal.Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (w *Workflow[REQ, RES, InitArg]) Execute(
	ctx context.Context, req REQ, opts ExecuteWorkflowOptions,
) (*WorkflowRun[RES], error) {
	run, err := w.sdk.cli.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                       opts.ID,
			TaskQueue:                w.sdk.taskQ,
			WorkflowExecutionTimeout: opts.WorkflowExecutionTimeout,
			WorkflowRunTimeout:       opts.WorkflowRunTimeout,
			WorkflowTaskTimeout:      opts.WorkflowTaskTimeout,
			WorkflowIDReusePolicy:    enumspb.WorkflowIdReusePolicy(opts.WorkflowIDReusePolicy),
			WorkflowIDConflictPolicy: enumspb.WorkflowIdConflictPolicy(opts.WorkflowIDConflictPolicy),
			StartDelay:               opts.StartDelay,
		},
		w.Name, req,
	)
	if err != nil {
		return nil, err
	}

	return &WorkflowRun[RES]{
		internal: run,
		ID:       run.GetID(),
		RunID:    run.GetRunID(),
	}, nil
}

type ExecuteChildWorkflowOptions struct {
	// WorkflowID of the child workflow to be scheduled.
	// Optional: an auto generated workflowID will be used if this is not provided.
	WorkflowID string

	// TaskQueue that the child workflow needs to be scheduled on.
	// Optional: the parent workflow task queue will be used if this is not provided.
	TaskQueue string

	// WorkflowExecutionTimeout - The end-to-end timeout for the child workflow execution including retries
	// and continue as new.
	// Optional: defaults to unlimited.
	WorkflowExecutionTimeout time.Duration

	// WorkflowRunTimeout - The timeout for a single run of the child workflow execution. Each retry or
	// continue as new should obey this timeout. Use WorkflowExecutionTimeout to specify how long the parent
	// is willing to wait for the child completion.
	// Optional: defaults to WorkflowExecutionTimeout
	WorkflowRunTimeout time.Duration

	// WorkflowTaskTimeout - Maximum execution time of a single Workflow Task. In the majority of cases there is
	// no need to change this timeout. Note that this timeout is not related to the overall Workflow duration in
	// any way. It defines for how long the Workflow can get blocked in the case of a Workflow Worker crash.
	// Default is 10 seconds. The Maximum value allowed by the Temporal Server is 1 minute.
	WorkflowTaskTimeout time.Duration

	// WaitForCancellation - Whether to wait for a canceled child workflow to be ended (child workflow can be ended
	// as: completed/failed/timeout/terminated/canceled)
	// Optional: default false
	WaitForCancellation bool

	// WorkflowIDReusePolicy - Whether server allow reuse of workflow ID, can be useful
	// for dedupe logic if set to WorkflowIdReusePolicyRejectDuplicate
	WorkflowIDReusePolicy enumspb.WorkflowIdReusePolicy

	// RetryPolicy specify how to retry child workflow if error happens.
	// Optional: default is no retry
	RetryPolicy *RetryPolicy

	// ParentClosePolicy specify how the retry child workflow get terminated.
	// default is Terminate
	ParentClosePolicy enumspb.ParentClosePolicy
}

func (w *Workflow[REQ, RES, InitArg]) ExecuteAsChild(
	ctx Context,
	req REQ,
	opts ExecuteChildWorkflowOptions,
) WorkflowFuture[RES] {
	return WorkflowFuture[RES]{
		f: workflow.ExecuteChildWorkflow(
			workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				WorkflowID:               opts.WorkflowID,
				TaskQueue:                opts.TaskQueue,
				WorkflowExecutionTimeout: opts.WorkflowExecutionTimeout,
				WorkflowRunTimeout:       opts.WorkflowRunTimeout,
				WorkflowIDReusePolicy:    opts.WorkflowIDReusePolicy,
				RetryPolicy:              opts.RetryPolicy,
			}),
			w.Name, req,
		),
	}
}

type WorkflowExecution struct {
	Name        string
	WorkflowID  string
	RunID       string
	HistorySize int64
	Memo        string
	StartTime   time.Time
	CloseTime   time.Time
	Duration    time.Duration
	Status      string
}

type SearchWorkflowRequest struct {
	NextPageToken []byte
	Query         string
}

type SearchWorkflowResponse struct {
	Executions    []WorkflowExecution
	NextPageToken []byte
}

func (sdk *SDK) SearchWorkflows(ctx context.Context, req SearchWorkflowRequest) (*SearchWorkflowResponse, error) {
	cliReq := &workflowservice.ListWorkflowExecutionsRequest{
		Namespace:     sdk.namespace,
		PageSize:      100,
		NextPageToken: req.NextPageToken,
		Query:         req.Query,
	}

	cliRes, err := sdk.cli.ListWorkflow(ctx, cliReq)
	if err != nil {
		return nil, err
	}

	res := &SearchWorkflowResponse{
		Executions:    utils.Map(toWorkflowExecution, cliRes.Executions),
		NextPageToken: cliRes.NextPageToken,
	}

	return res, nil
}

type CountWorkflowRequest struct {
	Query string
}

type CountWorkflowResponse struct {
	Total  int64
	Counts map[string]int64
}

func (sdk *SDK) CountWorkflows(ctx context.Context, req CountWorkflowRequest) (*CountWorkflowResponse, error) {
	res, err := sdk.cli.CountWorkflow(
		ctx,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: sdk.namespace,
			Query:     req.Query + " GROUP BY ExecutionStatus",
		},
	)
	if err != nil {
		return nil, err
	}

	out := &CountWorkflowResponse{
		Total:  res.Count,
		Counts: make(map[string]int64),
	}
	for _, c := range res.Groups {
		out.Counts[strings.Trim(string(c.GetGroupValues()[0].GetData()), "\"")] = c.GetCount()
	}

	return out, nil
}

type GetWorkflowHistoryRequest struct {
	WorkflowID  string
	RunID       string
	Skip        int
	Limit       int
	OnlyLastOne bool
}

type GetWorkflowHistoryResponse struct {
	Events []HistoryEvent
}

func (sdk *SDK) GetWorkflowHistory(
	ctx context.Context, req GetWorkflowHistoryRequest,
) (*GetWorkflowHistoryResponse, error) {
	if req.Limit <= 0 {
		req.Limit = 100
	}

	filterType := enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT
	if req.OnlyLastOne {
		filterType = enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT
	}
	iter := sdk.cli.GetWorkflowHistory(
		ctx,
		req.WorkflowID,
		req.RunID,
		false,
		filterType,
	)

	events := make([]HistoryEvent, 0, 100)
	offset := req.Skip
	limit := req.Limit
	for iter.HasNext() {
		e, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if offset--; offset >= 0 {
			continue
		}
		events = append(
			events,
			HistoryEvent{
				ID:      e.GetEventId(),
				Time:    e.GetEventTime().AsTime().Unix(),
				Type:    e.GetEventType().String(),
				Payload: utils.ToMap(e.GetAttributes()),
			},
		)

		if limit--; limit <= 0 {
			break
		}
	}

	return &GetWorkflowHistoryResponse{
		Events: events,
	}, nil
}

func toWorkflowExecution(src *v112.WorkflowExecutionInfo) WorkflowExecution {
	if src == nil {
		return WorkflowExecution{}
	}

	return WorkflowExecution{
		Name:        src.GetType().GetName(),
		WorkflowID:  src.GetExecution().GetWorkflowId(),
		RunID:       src.GetExecution().GetRunId(),
		HistorySize: src.GetHistoryLength(),
		StartTime:   src.GetStartTime().AsTime(),
		CloseTime:   src.CloseTime.AsTime(),
		Duration:    src.GetExecutionDuration().AsDuration(),
		Status:      src.GetStatus().String(),
	}
}

type CancelWorkflowRequest struct {
	WorkflowID string
	RunID      string
}

type CancelWorkflowResponse struct {
	Success bool
}

func (sdk *SDK) CancelWorkflow(ctx context.Context, req CancelWorkflowRequest) (*CancelWorkflowResponse, error) {
	err := sdk.cli.CancelWorkflow(ctx, req.WorkflowID, req.RunID)
	if err != nil {
		return nil, err
	}

	res := &CancelWorkflowResponse{
		Success: true,
	}

	return res, nil
}
