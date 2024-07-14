package async

import (
	"context"
)

type Queue struct {
	e *Engine

	id       string
	priority int
	workers  int

	cancelFn context.CancelFunc
}

func (q *Queue) register(srv *Engine) error {
	q.e = srv

	ctx, cf := context.WithCancel(context.Background())
	q.cancelFn = cf

	env, err := srv.backend.SubscribeQueue(ctx, q.id)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case e := <-env:
				t, ok := q.e.tasks[e.TaskName]
				if !ok {
					q.e.captureErr(err)

					continue
				}

				go q.runTask(ctx, t, e)

			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (q *Queue) runTask(ctx context.Context, t task, e TaskEnvelope) {
	err := t.handle(ctx, e)
	if err != nil {
		// put in the retry list
		q.e.captureErr(err)
	}
}

func (q *Queue) unregister(_ *Engine) {
	q.cancelFn()
}

func (q *Queue) ID() string {
	return q.id
}

func (q *Queue) Priority() int {
	return q.priority
}

func (q *Queue) Workers() int {
	return q.workers
}

func (q *Queue) Pending(ctx context.Context) ([]TaskRef, error) {
	return q.e.backend.PendingTasks(ctx, q.id)
}

func (q *Queue) Completed(ctx context.Context) ([]TaskRef, error) {
	return q.e.backend.CompletedTasks(ctx, q.id)
}

func (q *Queue) Archived(ctx context.Context) ([]TaskRef, error) {
	return q.e.backend.ArchivedTasks(ctx, q.id)
}
