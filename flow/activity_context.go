package flow

import (
	"context"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/log"
)

type ActivityInfo = activity.Info

type ActivityContext[REQ, RES, STATE any] struct {
	ctx context.Context //nolint:containedctx
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

func (ctx *ActivityContext[REQ, RES, STATE]) HasHearBeat() bool {
	return activity.HasHeartbeatDetails(ctx.ctx)
}

func (ctx *ActivityContext[REQ, RES, STATE]) GetHeartBeat() any {
	return activity.GetHeartbeatDetails(ctx.ctx)
}

func (ctx *ActivityContext[REQ, RES, STATE]) SetHeartBeat(details any) {
	activity.RecordHeartbeat(ctx.ctx, details)
}

type Client = client.Client

func (ctx *ActivityContext[REQ, RES, STATE]) Client() Client {
	return activity.GetClient(ctx.ctx)
}

func SetHearBeat[D, REQ, RES, STATE any](
	ctx ActivityContext[REQ, RES, STATE], details D,
) {
	activity.RecordHeartbeat(ctx.ctx, details)
}

func GetHeartBeat[D, REQ, RES, STATE any](ctx *ActivityContext[REQ, RES, STATE]) D {
	return activity.GetHeartbeatDetails(ctx.ctx).(D)
}
