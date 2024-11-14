package dummybackend

import (
	"container/list"
	"context"

	"github.com/clubpay/ronykit/async"
	"github.com/clubpay/ronykit/kit/utils"
)

var _ async.Backend = (*Backend)(nil)

type Backend struct {
	queues map[string]list.List
	subs   map[string][]chan async.TaskEnvelope
}

func New() *Backend {
	return &Backend{
		queues: make(map[string]list.List),
		subs:   make(map[string][]chan async.TaskEnvelope),
	}
}

func (b *Backend) CreateQueue(_ context.Context, name string) error {
	b.queues[name] = list.List{}

	return nil
}

func (b *Backend) ListQueues(_ context.Context) ([]string, error) {
	var queues []string
	for name := range b.queues {
		queues = append(queues, name)
	}

	return queues, nil
}

func (b *Backend) SubscribeQueue(_ context.Context, queueID string) (<-chan async.TaskEnvelope, error) {
	teChan := make(chan async.TaskEnvelope, 10)
	b.subs[queueID] = append(b.subs[queueID], teChan)

	return teChan, nil
}

func (b *Backend) EnqueueTask(_ context.Context, e async.TaskEnvelope) error {
	s, ok := b.subs[e.QueueID]
	if !ok {
		idx := utils.RandomInt(len(s))
		s[idx] <- e
	}

	return nil
}

func (b *Backend) CheckTask(_ context.Context, queueID string, refID string) (async.Status, error) {
	return async.StatusSuccess, nil
}

func (b *Backend) CancelTask(_ context.Context, queueID string, refID string) error {
	return nil
}

func (b *Backend) PendingTasks(_ context.Context, queueID string) ([]async.TaskRef, error) {
	//TODO implement me
	panic("implement me")
}

func (b *Backend) CompletedTasks(_ context.Context, queueID string) ([]async.TaskRef, error) {
	//TODO implement me
	panic("implement me")
}

func (b *Backend) ArchivedTasks(_ context.Context, queueID string) ([]async.TaskRef, error) {
	//TODO implement me
	panic("implement me")
}
