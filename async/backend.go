package async

import (
	"context"
	"errors"
)

type Backend interface {
	CreateQueue(ctx context.Context, name string) error
	ListQueues(ctx context.Context) ([]string, error)
	SubscribeQueue(ctx context.Context, queueID string) (<-chan TaskEnvelope, error)

	EnqueueTask(ctx context.Context, e TaskEnvelope) error
	CheckTask(ctx context.Context, queueID string, refID string) (Status, error)
	CancelTask(ctx context.Context, queueID string, refID string) error
	PendingTasks(ctx context.Context, queueID string) ([]TaskRef, error)
	CompletedTasks(ctx context.Context, queueID string) ([]TaskRef, error)
	ArchivedTasks(ctx context.Context, queueID string) ([]TaskRef, error)
}

var ErrQueueAlreadyExists = errors.New("task queue already exists")
