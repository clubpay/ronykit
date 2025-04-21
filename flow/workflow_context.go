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

func (ctx WorkflowContext[REQ, RES, STATE]) NamedSelector(name string) Selector {
	return workflow.NewNamedSelector(ctx.ctx, name)
}

func (ctx WorkflowContext[REQ, RES, STATE]) Selector() Selector {
	return workflow.NewSelector(ctx.ctx)
}

func (ctx WorkflowContext[REQ, RES, STATE]) WaitGroup() WaitGroup {
	return workflow.NewWaitGroup(ctx.ctx)
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

func (ctx WorkflowContext[REQ, RES, STATE]) State() STATE {
	return ctx.s
}

func (ctx *WorkflowContext[REQ, RES, STATE]) SetState(s STATE) {
	ctx.s = s
}

func (ctx WorkflowContext[REQ, RES, STATE]) S() STATE {
	return ctx.s
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
