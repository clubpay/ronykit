package flow

import "go.temporal.io/sdk/workflow"

type Future[M any] struct {
	f workflow.Future
}

func (f Future[M]) Get(ctx Context) (*M, error) {
	var out M
	err := f.f.Get(ctx, &out)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func (f Future[M]) IsReady() bool {
	return f.f.IsReady()
}

type WorkflowFuture[M any] struct {
	f workflow.ChildWorkflowFuture
}

func (f WorkflowFuture[M]) Get(ctx Context) (*M, error) {
	var out M
	err := f.f.Get(ctx, &out)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

func (f WorkflowFuture[M]) IsReady() bool {
	return f.f.IsReady()
}
