package flow

import "go.temporal.io/sdk/workflow"

type Signal[T any] struct {
	Name string
}

func (s Signal[T]) GetChannel(ctx Context) SignalChannel[T] {
	return SignalChannel[T]{
		ch: workflow.GetSignalChannel(ctx, s.Name),
	}
}

func (s Signal[T]) Send(ctx Context, workflowID string, arg T) {
	workflow.SignalExternalWorkflow(ctx, workflowID, "", s.Name, arg)
}
