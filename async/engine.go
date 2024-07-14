package async

import (
	"context"
)

type Engine struct {
	id      string
	backend Backend
	tasks   map[string]task

	errFunc func(err error)
}

func NewEngine(backend Backend, opt ...Option) (*Engine, error) {
	srv := &Engine{
		backend: backend,
	}
	for _, o := range opt {
		err := o(srv)
		if err != nil {
			return nil, err
		}
	}

	return srv, nil
}

func (srv *Engine) captureErr(err error) {
	if srv.errFunc != nil {
		srv.errFunc(err)
	}
}

func (srv *Engine) Shutdown(ctx context.Context) error {
	return nil
}

type taskSetupParams struct{}

type TaskSetupOption func(*taskSetupParams)

func SetupTask[TD TaskDataType, TDP TaskDataTypePtr[TD]](
	name string,
	h Handler[TD, TDP],
	opts ...TaskSetupOption,
) *Task[TD, TDP] {
	p := &taskSetupParams{}
	for _, o := range opts {
		o(p)
	}

	t := &Task[TD, TDP]{
		name:    name,
		handler: h,
	}

	return t
}

type queueSetupParams struct {
	workers  int
	priority int
	maxLen   int
}

type QueueSetupOption func(*queueSetupParams)

func WithWorkers(workers int) QueueSetupOption {
	return func(q *queueSetupParams) {
		q.workers = workers
	}
}

func WithPriority(priority int) QueueSetupOption {
	return func(q *queueSetupParams) {
		q.priority = priority
	}
}

// WithMaxLen indicates the maximum number of waiting tasks should not exceed `maxLen` and
// the Enqueue method of the task should return ErrQueueIsFull error.
func WithMaxLen(maxLen int) QueueSetupOption {
	return func(q *queueSetupParams) {
		q.maxLen = maxLen
	}
}

func SetupQueue(
	name string,
	opt ...QueueSetupOption,
) *Queue {
	p := &queueSetupParams{}
	for _, o := range opt {
		o(p)
	}

	return &Queue{
		id:       name,
		priority: p.priority,
		workers:  p.workers,
	}
}
