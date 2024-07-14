package async

import "errors"

var (
	ErrUnmarshalTaskEnvelop = errors.New("failed to unmarshal task envelope")
	ErrQueueIsFull          = errors.New("queue is full")
	ErrTaskNotRegistered    = errors.New("task not registered")
	ErrTaskDeadlineExceeded = errors.New("task deadline exceeded")
)
