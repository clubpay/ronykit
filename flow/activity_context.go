package flow

import (
	"context"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/log"
)

type ActivityInfo = activity.Info

type ActivityContext[REQ, RES, STATE any] struct {
	ctx context.Context
	s   STATE
}

func (ctx ActivityContext[REQ, RES, STATE]) Context() context.Context {
	return ctx.ctx
}

func (ctx ActivityContext[REQ, RES, STATE]) Log() log.Logger {
	return activity.GetLogger(ctx.ctx)
}

func (ctx ActivityContext[REQ, RES, STATE]) Info() ActivityInfo {
	return activity.GetInfo(ctx.ctx)
}

func (ctx ActivityContext[REQ, RES, STATE]) WorkflowID() string {
	return activity.GetInfo(ctx.ctx).WorkflowExecution.ID
}

func (ctx ActivityContext[REQ, RES, STATE]) State() STATE {
	return ctx.s
}

func (ctx ActivityContext[REQ, RES, STATE]) S() STATE {
	return ctx.s
}

func (ctx *ActivityContext[REQ, RES, STATE]) SetState(state STATE) {
	ctx.s = state
}
