package flow

import (
	"context"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/log"
)

type ActivityInfo = activity.Info

type ActivityContext[REQ, RES any] struct {
	ctx context.Context
}

func (ctx ActivityContext[REQ, RES]) Context() context.Context {
	return ctx.ctx
}

func (ctx ActivityContext[REQ, RES]) Log() log.Logger {
	return activity.GetLogger(ctx.ctx)
}

func (ctx ActivityContext[REQ, RES]) Info() ActivityInfo {
	return activity.GetInfo(ctx.ctx)
}

func (ctx ActivityContext[REQ, RES]) WorkflowID() string {
	return activity.GetInfo(ctx.ctx).WorkflowExecution.ID
}
