package async

import (
	"context"
	"encoding"
	"errors"
	"time"

	"github.com/clubpay/ronykit/kit/utils"
)

const (
	defaultMaxRetry = 10
)

type task interface {
	handle(ctx context.Context, e TaskEnvelope) error
}

type TaskDataType interface {
	encoding.BinaryMarshaler
}

type TaskDataTypePtr[DT TaskDataType] interface {
	*DT

	encoding.BinaryUnmarshaler
}

type Handler[TD TaskDataType, TDP TaskDataTypePtr[TD]] func(ctx *Context, p TD) error

type Task[TD TaskDataType, TDP TaskDataTypePtr[TD]] struct {
	e *Engine

	name    string
	handler Handler[TD, TDP]
}

func (t *Task[TD, TDP]) register(srv *Engine) error {
	t.e = srv

	return nil
}

func (t *Task[TD, TDP]) unregister(_ *Engine) {}

func (t *Task[TD, TDP]) Name() string {
	return t.name
}

type Status int

func (s Status) String() string {
	return [4]string{
		"StatusPending",
		"StatusRunning",
		"StatusSuccess",
		"StatusFailed",
	}[s]
}

const (
	StatusPending Status = iota
	StatusRunning
	StatusSuccess
	StatusFailed
)

type EnqueueParams struct {
	// ID is the unique identifier for this task. If left empty, it will be auto-generated.
	id        string
	maxRetry  int
	uniqueKey string
	groupKey  string
	notBefore int64
	notAfter  int64
}

func (x EnqueueParams) ID(id string) EnqueueParams {
	x.id = id

	return x
}

func (x EnqueueParams) MaxRetry(max int) EnqueueParams {
	x.maxRetry = max

	return x
}

func (x EnqueueParams) UniqueKey(key string) EnqueueParams {
	x.uniqueKey = key

	return x
}

func (x EnqueueParams) GroupKey(key string) EnqueueParams {
	x.groupKey = key

	return x
}

// Delay set how long to wait before picking up by a worker.
// **NOTE**: this is just the minimum delay required, the actual delay until tasks are picked up can
// be different based on other factors such as server load.
// **NOTE**: the delay will be truncated to seconds.
func (x EnqueueParams) Delay(d time.Duration) EnqueueParams {
	x.notBefore = utils.TimeUnix() + int64(d/time.Second)

	return x
}

// NotAfter sets a time that if a task has not been started to be processed, then it will be
// dropped.
func (x EnqueueParams) NotAfter(notAfter time.Time) EnqueueParams {
	x.notAfter = notAfter.Unix()

	return x
}

// Enqueue put the task into the queue, and it will be picked up by one of the workers soon.
func (t *Task[TD, TDP]) Enqueue(
	ctx context.Context,
	td TD, queueID string,
	p EnqueueParams,
) (*TaskRef, error) {
	if t.e == nil {
		return nil, ErrTaskNotRegistered
	}
	payload, err := td.MarshalBinary()
	if err != nil {
		return nil, err
	}

	envelope := TaskEnvelope{
		ID:          utils.Coalesce(p.id, "T_"+utils.RandomID(12)),
		TaskName:    t.name,
		QueueID:     queueID,
		Payload:     payload,
		Submitter:   t.e.id,
		MaxRetry:    utils.Coalesce(p.maxRetry, defaultMaxRetry),
		Retried:     0,
		UniqueKey:   p.uniqueKey,
		GroupKey:    p.groupKey,
		SubmitAt:    utils.TimeUnix(),
		NotBeforeAt: utils.Coalesce(p.notBefore, utils.TimeUnix()),
		NotAfterAt:  p.notAfter,
	}

	return t.enqueue(ctx, envelope)
}

func (t *Task[TD, TDP]) enqueue(ctx context.Context, envelope TaskEnvelope) (*TaskRef, error) {
	err := t.e.backend.EnqueueTask(ctx, envelope)
	if err != nil {
		return nil, err
	}

	return &TaskRef{
		id:       "T_" + utils.RandomID(16),
		taskName: envelope.TaskName,
		queueID:  envelope.QueueID,
	}, nil
}

func (t *Task[TD, TDP]) handle(ctx context.Context, e TaskEnvelope) error {
	var td TD
	err := TDP(&td).UnmarshalBinary(e.Payload)
	if err != nil {
		return errors.Join(ErrUnmarshalTaskEnvelop, err)
	}

	tCtx := newContext(ctx, t.e, e)

	return t.handler(tCtx, td)
}

type TaskRef struct {
	srv *Engine

	id       string
	taskName string
	queueID  string
}

func (tr *TaskRef) ID() string {
	return tr.id
}

func (tr *TaskRef) TaskName() string {
	return tr.taskName
}

func (tr *TaskRef) Status(ctx context.Context) (Status, error) {
	return tr.srv.backend.CheckTask(ctx, tr.queueID, tr.id)
}

func (tr *TaskRef) Cancel(ctx context.Context) error {
	return tr.srv.backend.CancelTask(ctx, tr.queueID, tr.id)
}

// TaskEnvelope represents a task envelope containing task's data which is serialized and will be
// sent and received from Backend.
type TaskEnvelope struct {
	ID        string `json:"id"`
	TaskName  string `json:"taskName"`
	QueueID   string `json:"queueID"`
	Payload   []byte `json:"payload"`
	Submitter string `json:"submitter"`
	// MaxRetry is the max number of retries for this task.
	MaxRetry int `json:"maxRetry"`
	// Retried is the number of times we've retried this task so far.
	Retried int `json:"retried"`
	// ErrorMsg holds the error message from the last failure.
	ErrorMsg string `json:"errorMsg"`
	// UniqueKey holds the redis key used for uniqueness lock for this task.
	//
	// Empty string indicates that no uniqueness lock was used.
	UniqueKey string `json:"uniqueKey"`
	// GroupKey holds the group key used for task aggregation.
	//
	// Empty string indicates no aggregation is used for this task.
	GroupKey string `json:"groupKey"`
	// Retention specifies the number of seconds the task should be retained after completion.
	Retention int64 `json:"retention"`
	// SubmitAt the time of the task has been submitted to backend
	// the number of seconds elapsed since January 1, 1970 UTC.
	SubmitAt int64 `json:"submitAt"`
	// NotBeforeAt is the minimum Unix timestamp at which this task could begin being processed.
	NotBeforeAt int64 `json:"notBeforeAt"`
	// NotAfterAt is the maximum Unix timestamp at which this task could begin being processed.
	NotAfterAt int64 `json:"notAfterAt"`
	// LastFailedAt is the time of last failure in Unix time,
	// the number of seconds elapsed since January 1, 1970 UTC.
	LastFailedAt int64 `json:"lastFailedAt"`
	// CompletedAt the time the task was processed successfully in Unix time,
	// the number of seconds elapsed since January 1, 1970 UTC.
	CompletedAt int64 `json:"completedAt"`
}
