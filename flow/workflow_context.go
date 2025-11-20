package flow

import (
	"time"

	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type WorkflowContext[REQ, RES, STATE any] struct {
	ctx workflow.Context
	s   STATE
}

type WorkflowInfo = workflow.Info

func (ctx WorkflowContext[REQ, RES, STATE]) Info() *WorkflowInfo {
	return workflow.GetInfo(ctx.ctx)
}

func (ctx WorkflowContext[REQ, RES, STATE]) Context() workflow.Context {
	return ctx.ctx
}

func (ctx WorkflowContext[REQ, RES, STATE]) WithCancel() (workflow.Context, workflow.CancelFunc) {
	return workflow.WithCancel(ctx.ctx)
}

func (ctx WorkflowContext[REQ, RES, STATE]) NamedSelector(name string) Selector {
	return workflow.NewNamedSelector(ctx.ctx, name)
}

func (ctx WorkflowContext[REQ, RES, STATE]) Selector() Selector {
	return workflow.NewSelector(ctx.ctx)
}

func (ctx WorkflowContext[REQ, RES, STATE]) WaitGroup() WaitGroup {
	return workflow.NewWaitGroup(ctx.ctx)
}

func (ctx WorkflowContext[REQ, RES, STATE]) Go(fn func(ctx Context)) {
	workflow.Go(ctx.ctx, fn)
}

func (ctx WorkflowContext[REQ, RES, STATE]) Log() log.Logger {
	return workflow.GetLogger(ctx.ctx)
}

func (ctx WorkflowContext[REQ, RES, STATE]) Sleep(d time.Duration) error {
	return workflow.Sleep(ctx.ctx, d)
}

func (ctx WorkflowContext[REQ, RES, STATE]) Timer(d time.Duration) Future[temporal.CanceledError] {
	return Future[temporal.CanceledError]{
		f: workflow.NewTimer(ctx.ctx, d),
	}
}

func (ctx WorkflowContext[REQ, RES, STATE]) TimerCtx(
	wCtx Context, d time.Duration,
) Future[temporal.CanceledError] {
	return Future[temporal.CanceledError]{
		f: workflow.NewTimer(wCtx, d),
	}
}

func (ctx WorkflowContext[REQ, RES, STATE]) State() STATE {
	return ctx.s
}

func (ctx *WorkflowContext[REQ, RES, STATE]) SetState(s STATE) {
	ctx.s = s
}

func (ctx WorkflowContext[REQ, RES, STATE]) S() STATE {
	return ctx.s
}

func (ctx WorkflowContext[REQ, RES, STATE]) WorkflowID() string {
	return workflow.GetInfo(ctx.ctx).WorkflowExecution.ID
}

func (ctx WorkflowContext[REQ, RES, STATE]) WorkflowRunID() string {
	return workflow.GetInfo(ctx.ctx).WorkflowExecution.RunID
}

func (ctx WorkflowContext[REQ, RES, STATE]) WorkflowType() string {
	return workflow.GetInfo(ctx.ctx).WorkflowType.Name
}

func (ctx WorkflowContext[REQ, RES, STATE]) ContinueAsNewError(req REQ) error {
	return workflow.NewContinueAsNewError(ctx.ctx, ctx.WorkflowType(), req)
}

type Version = workflow.Version

const DefaultVersion = workflow.DefaultVersion

func (ctx WorkflowContext[REQ, RES, STATE]) GetVersion(changeID string, minVer, maxVer Version) Version {
	return workflow.GetVersion(ctx.ctx, changeID, minVer, maxVer)
}

func SideEffect[T any](ctx Context, fn func() T) (T, error) {
	reqEncoded := workflow.SideEffect(
		ctx,
		func(wctx workflow.Context) any {
			return fn()
		},
	)

	var out T

	err := reqEncoded.Get(&out)

	return out, err
}

func MutableSideEffect[T comparable](ctx Context, id string, fn func() T) (T, error) {
	reqEncoded := workflow.MutableSideEffect(
		ctx, id,
		func(wctx workflow.Context) any {
			return fn()
		},
		func(a, b any) bool {
			return a.(T) == b.(T) //nolint:forcetypeassert
		},
	)

	var out T

	err := reqEncoded.Get(&out)

	return out, err
}
