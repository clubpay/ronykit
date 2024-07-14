package async

import (
	"context"
	"time"

	"github.com/clubpay/ronykit/kit/utils"
)

// Context represents the execution context of a task handler.
// It contains the underlying context.Context, a TaskEnvelope, and an enqueueFn function for task enqueueing.
type Context struct {
	ctx context.Context

	e  *Engine
	te TaskEnvelope
}

func newContext(
	ctx context.Context,
	e *Engine,
	te TaskEnvelope,
) *Context {
	return &Context{
		ctx: ctx,
		e:   e,
		te:  te,
	}
}

// LastTry shows if this is the last time this handler will be called.
func (ctx *Context) LastTry() bool {
	return ctx.te.MaxRetry == ctx.te.Retried
}

// Retries returns the number of times this task has been retried.
func (ctx *Context) Retries() int {
	return ctx.te.Retried
}

type RequeueParams struct {
	maxRetry  int
	uniqueKey string
	groupKey  string
	notBefore int64
	notAfter  int64
	notRetry  bool
}

func (x RequeueParams) MaxRetry(maxRetry int) RequeueParams {
	x.maxRetry = maxRetry

	return x
}

func (x RequeueParams) UniqueKey(key string) RequeueParams {
	x.uniqueKey = key

	return x
}

func (x RequeueParams) GroupKey(key string) RequeueParams {
	x.groupKey = key

	return x
}

// Delay set how long to wait before picking up by a worker.
// **NOTE**: this is just the minimum delay required, the actual delay until tasks are picked up can
// be different based on other factors such as server load.
// **NOTE**: the delay will be truncated to seconds.
func (x RequeueParams) Delay(d time.Duration) RequeueParams {
	x.notBefore = utils.TimeUnix() + int64(d/time.Second)

	return x
}

// NotAfter sets a time that if a task has not been started to be processed, then it will be
// dropped.
func (x RequeueParams) NotAfter(notAfter time.Time) RequeueParams {
	x.notAfter = notAfter.Unix()

	return x
}

// NotRetry makes sure that the Retries does not increase for this.
// It could be useful in some cases like rate-limit backoff handling, or recurring tasks.
func (x RequeueParams) NotRetry() RequeueParams {
	x.notRetry = true

	return x
}

// Requeue puts the task back into queue. It resets the internal counters.
func (ctx *Context) Requeue(p RequeueParams) error {
	retried := ctx.Retries()
	if !p.notRetry {
		retried += 1
	}
	err := ctx.e.backend.EnqueueTask(
		ctx.ctx,
		TaskEnvelope{
			ID:          ctx.te.ID,
			TaskName:    ctx.te.TaskName,
			QueueID:     ctx.te.QueueID,
			Payload:     ctx.te.Payload,
			Submitter:   ctx.e.id,
			MaxRetry:    utils.Coalesce(p.maxRetry, ctx.te.MaxRetry),
			Retried:     retried,
			UniqueKey:   utils.Coalesce(p.uniqueKey, ctx.te.UniqueKey),
			GroupKey:    utils.Coalesce(p.groupKey, ctx.te.GroupKey),
			SubmitAt:    utils.TimeUnix(),
			NotBeforeAt: utils.Coalesce(p.notBefore, utils.TimeUnix()),
			NotAfterAt:  p.notAfter,
		},
	)
	if err != nil {
		return err
	}

	return nil
}
