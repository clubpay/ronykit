package flow

import "context"

type chainExecutor[DATA, STATE any] interface {
	Execute(ctx Context, req DATA, opts ExecuteActivityOptions) Future[DATA]
}

// ChainFunc transforms DATA in a chain step using a standard context.Context.
type ChainFunc[DATA any] func(ctx context.Context, data DATA) (DATA, error)

// ChainActivityFunc transforms DATA in a chain step with access to activity state.
type ChainActivityFunc[DATA, STATE any] func(ctx *ActivityContext[DATA, DATA, STATE], data DATA) (DATA, error)

// ChainStep is one step in a chain workflow. Each step runs an activity that
// accepts DATA, mutates or replaces it, and passes the result to the next step.
type ChainStep[DATA, STATE any] struct {
	executor chainExecutor[DATA, STATE]
	options  ExecuteActivityOptions
}

// ChainStepOf binds an activity or activity factory to execution options.
func ChainStepOf[DATA, STATE any, E chainExecutor[DATA, STATE]](
	executor E,
	opts ExecuteActivityOptions,
) ChainStep[DATA, STATE] {
	return ChainStep[DATA, STATE]{
		executor: executor,
		options:  opts,
	}
}

// ChainStepFunc registers a chain step from a raw function via ToActivity.
func ChainStepFunc[DATA, STATE any](
	stepName, group string,
	fn ChainFunc[DATA],
	opts ExecuteActivityOptions,
	activityOpts ...ActivityOption,
) ChainStep[DATA, STATE] {
	act := ToActivity[STATE, DATA, DATA](
		stepName, group,
		func(ctx context.Context, data DATA) (*DATA, error) {
			res, err := fn(ctx, data)
			if err != nil {
				return nil, err
			}

			return &res, nil
		},
		activityOpts...,
	)

	return ChainStep[DATA, STATE]{
		executor: act,
		options:  opts,
	}
}

// ChainStepActivityFunc registers a chain step from an activity function.
func ChainStepActivityFunc[DATA, STATE any](
	stepName, group string,
	fn ChainActivityFunc[DATA, STATE],
	opts ExecuteActivityOptions,
	activityOpts ...ActivityOption,
) ChainStep[DATA, STATE] {
	act := NewActivity[DATA, DATA, STATE](
		stepName, group,
		func(ctx *ActivityContext[DATA, DATA, STATE], data DATA) (*DATA, error) {
			res, err := fn(ctx, data)
			if err != nil {
				return nil, err
			}

			return &res, nil
		},
		activityOpts...,
	)

	return ChainStep[DATA, STATE]{
		executor: act,
		options:  opts,
	}
}

// NewChainWorkflow creates a workflow that runs activities in sequence, passing
// the same DATA value through each step. Each activity receives the output of
// the previous one and returns an updated DATA for the next — similar to a
// visitor pipeline over a shared payload, but with durable activity execution
// between steps.
func NewChainWorkflow[DATA, STATE any](
	name, group string,
	steps ...ChainStep[DATA, STATE],
) *Workflow[DATA, DATA, STATE] {
	return NewWorkflow(
		name, group,
		func(ctx *WorkflowContext[DATA, DATA, STATE], req DATA) (*DATA, error) {
			data := req
			for _, step := range steps {
				res, err := step.executor.Execute(ctx.Context(), data, step.options).Get(ctx.Context())
				if err != nil {
					return nil, err
				}

				data = *res
			}

			return &data, nil
		},
	)
}

// NewChainWorkflowWithOptions is like NewChainWorkflow but applies the same
// execution options to every step.
func NewChainWorkflowWithOptions[DATA, STATE any](
	name, group string,
	opts ExecuteActivityOptions,
	executors ...chainExecutor[DATA, STATE],
) *Workflow[DATA, DATA, STATE] {
	steps := make([]ChainStep[DATA, STATE], len(executors))
	for i, executor := range executors {
		steps[i] = ChainStep[DATA, STATE]{
			executor: executor,
			options:  opts,
		}
	}

	return NewChainWorkflow(name, group, steps...)
}
