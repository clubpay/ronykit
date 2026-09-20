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

func (ctx *ActivityContext[REQ, RES, STATE]) HasHeartbeat() bool {
	return activity.HasHeartbeatDetails(ctx.ctx)
}

// HasHearBeat is a misspelled alias of HasHeartbeat.
//
// Deprecated: use HasHeartbeat.
func (ctx *ActivityContext[REQ, RES, STATE]) HasHearBeat() bool {
	return ctx.HasHeartbeat()
}

func (ctx *ActivityContext[REQ, RES, STATE]) HeartbeatDetails(dest any) error {
	return activity.GetHeartbeatDetails(ctx.ctx, dest)
}

func (ctx *ActivityContext[REQ, RES, STATE]) GetHeartbeat() (any, error) {
	var details any

	err := activity.GetHeartbeatDetails(ctx.ctx, &details)

	return details, err
}

// GetHeartBeat is a misspelled alias of GetHeartbeat.
//
// Deprecated: use GetHeartbeat.
func (ctx *ActivityContext[REQ, RES, STATE]) GetHeartBeat() (any, error) {
	return ctx.GetHeartbeat()
}

func (ctx *ActivityContext[REQ, RES, STATE]) SetHeartbeat(details any) {
	activity.RecordHeartbeat(ctx.ctx, details)
}

// SetHeartBeat is a misspelled alias of SetHeartbeat.
//
// Deprecated: use SetHeartbeat.
func (ctx *ActivityContext[REQ, RES, STATE]) SetHeartBeat(details any) {
	ctx.SetHeartbeat(details)
}

type Client = client.Client

func (ctx *ActivityContext[REQ, RES, STATE]) Client() Client {
	return activity.GetClient(ctx.ctx)
}

func SetHeartbeat[D, REQ, RES, STATE any](
	ctx ActivityContext[REQ, RES, STATE], details D,
) {
	activity.RecordHeartbeat(ctx.ctx, details)
}

// SetHearBeat is a misspelled alias of SetHeartbeat.
//
// Deprecated: use SetHeartbeat.
func SetHearBeat[D, REQ, RES, STATE any](
	ctx ActivityContext[REQ, RES, STATE], details D,
) {
	SetHeartbeat(ctx, details)
}

func GetHeartbeat[D, REQ, RES, STATE any](ctx *ActivityContext[REQ, RES, STATE]) (D, error) {
	var details D

	err := activity.GetHeartbeatDetails(ctx.ctx, &details)

	return details, err
}

// GetHeartBeat is a misspelled alias of GetHeartbeat.
//
// Deprecated: use GetHeartbeat.
func GetHeartBeat[D, REQ, RES, STATE any](ctx *ActivityContext[REQ, RES, STATE]) (D, error) {
	return GetHeartbeat[D, REQ, RES, STATE](ctx)
}
