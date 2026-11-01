package flowtest_test

import (
	"errors"
	"testing"
	"time"

	"github.com/clubpay/ronykit/flow"
	"github.com/clubpay/ronykit/flow/flowtest"
	"github.com/clubpay/ronykit/rony/errs"
)

type addState struct {
	Base int
}

func TestWorkflowRunsActivityWithState(t *testing.T) {
	const group = "flowtest-state"

	act := flow.NewActivity[int, int, addState](
		"FlowtestAdd", group,
		func(ctx *flow.ActivityContext[int, int, addState], req int) (*int, error) {
			v := req + ctx.State().Base

			return &v, nil
		},
	)
	wf := flow.NewWorkflow[int, int, addState](
		"FlowtestAddWorkflow", group,
		func(ctx *flow.WorkflowContext[int, int, addState], req int) (*int, error) {
			return act.Execute(ctx.Context(), req, flow.ExecuteActivityOptions{
				StartToCloseTimeout: time.Minute,
			}).Get(ctx.Context())
		},
	)

	env := flowtest.NewWorkflowEnv(flowtest.Config{Group: group})
	env.InitWithState(addState{Base: 10})

	got, err := flowtest.Execute(env, wf, 5)
	if err != nil {
		t.Fatal(err)
	}
	if *got != 15 {
		t.Fatalf("got %d", *got)
	}
}

func TestOnActivity(t *testing.T) {
	const group = "flowtest-mock"

	act := flow.NewActivity[int, int, flow.EMPTY](
		"FlowtestMocked", group,
		func(_ *flow.ActivityContext[int, int, flow.EMPTY], req int) (*int, error) {
			return &req, nil
		},
	)
	wf := flow.NewWorkflow[int, int, flow.EMPTY](
		"FlowtestMockedWorkflow", group,
		func(ctx *flow.WorkflowContext[int, int, flow.EMPTY], req int) (*int, error) {
			return act.Execute(ctx.Context(), req, flow.ExecuteActivityOptions{
				StartToCloseTimeout: time.Minute,
			}).Get(ctx.Context())
		},
	)

	env := flowtest.NewWorkflowEnv(flowtest.Config{Group: group})
	env.Init()

	mocked := 9
	flowtest.OnActivity(env, act, 4).Return(&mocked, nil).Once()

	got, err := flowtest.Execute(env, wf, 4)
	if err != nil {
		t.Fatal(err)
	}
	if *got != 9 {
		t.Fatalf("got %d", *got)
	}

	env.AssertExpectations(t)
}

func TestSignalAndExternalSignal(t *testing.T) {
	const group = "flowtest-signal"

	sig := flow.Signal[string]{Name: "flowtest-ping"}
	wf := flow.NewWorkflow[string, string, flow.EMPTY](
		"FlowtestSignalWorkflow", group,
		func(ctx *flow.WorkflowContext[string, string, flow.EMPTY], req string) (*string, error) {
			sig.Send(ctx.Context(), "other-workflow", req)
			got, _ := sig.GetChannel(ctx.Context()).Receive(ctx.Context())
			out := req + got

			return &out, nil
		},
	)

	env := flowtest.NewWorkflowEnv(flowtest.Config{Group: group})
	env.Init()
	env.IgnoreExternalSignals()
	env.After(time.Minute, func() {
		flowtest.Signal(env, sig, "!")
	})

	got, err := flowtest.Execute(env, wf, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if *got != "hi!" {
		t.Fatalf("got %q", *got)
	}
}

func TestOnWorkflow(t *testing.T) {
	const group = "flowtest-child"

	child := flow.NewWorkflow[int, int, flow.EMPTY](
		"FlowtestChild", group,
		func(_ *flow.WorkflowContext[int, int, flow.EMPTY], req int) (*int, error) {
			return &req, nil
		},
	)
	parent := flow.NewWorkflow[int, int, flow.EMPTY](
		"FlowtestParent", group,
		func(ctx *flow.WorkflowContext[int, int, flow.EMPTY], req int) (*int, error) {
			return child.ExecuteAsChild(ctx.Context(), req, flow.ExecuteChildWorkflowOptions{}).Get(ctx.Context())
		},
	)

	env := flowtest.NewWorkflowEnv(flowtest.Config{Group: group})
	env.Init()

	mocked := 7
	flowtest.OnWorkflow(env, child).Return(&mocked, nil)

	got, err := flowtest.Execute(env, parent, 1)
	if err != nil {
		t.Fatal(err)
	}
	if *got != 7 {
		t.Fatalf("got %d", *got)
	}
}

func TestActivityFactoryAndBusinessError(t *testing.T) {
	const group = "flowtest-factory"

	fac := flow.NewActivityFactory[int, int, addState](
		"FlowtestFactory", group,
		func(s addState) flow.ActivityFunc[int, int, addState] {
			return func(_ *flow.ActivityContext[int, int, addState], req int) (*int, error) {
				if req < 0 {
					return nil, &errs.Error{Code: errs.NotFound, Item: "missing"}
				}

				v := req + s.Base

				return &v, nil
			}
		},
	)
	okFlow := flow.NewWorkflow[int, int, addState](
		"FlowtestFactoryWorkflow", group,
		func(ctx *flow.WorkflowContext[int, int, addState], req int) (*int, error) {
			return fac.Execute(ctx.Context(), req, flow.ExecuteActivityOptions{
				StartToCloseTimeout: time.Minute,
			}).Get(ctx.Context())
		},
	)

	env := flowtest.NewWorkflowEnv(flowtest.Config{Group: group})
	env.InitWithState(addState{Base: 2})

	got, err := flowtest.Execute(env, okFlow, 3)
	if err != nil {
		t.Fatal(err)
	}
	if *got != 5 {
		t.Fatalf("got %d", *got)
	}

	fail := flow.NewWorkflow[int, flow.EMPTY, addState](
		"FlowtestFactoryFail", group,
		func(ctx *flow.WorkflowContext[int, flow.EMPTY, addState], req int) (*flow.EMPTY, error) {
			_, err := fac.Execute(ctx.Context(), req, flow.ExecuteActivityOptions{
				StartToCloseTimeout: time.Minute,
			}).Get(ctx.Context())

			return nil, err
		},
	)
	failEnv := flowtest.NewWorkflowEnv(flowtest.Config{Group: group})
	failEnv.InitWithState(addState{Base: 2})

	_, err = flowtest.Execute(failEnv, fail, -1)
	var domainErr *errs.Error
	if !errors.As(err, &domainErr) || errs.Code(err) != errs.NotFound {
		t.Fatalf("got %v", err)
	}
}

func TestExecuteActivityHeartbeat(t *testing.T) {
	const group = "flowtest-heartbeat"

	act := flow.NewActivity[struct{}, int, flow.EMPTY](
		"FlowtestHeartbeat", group,
		func(ctx *flow.ActivityContext[struct{}, int, flow.EMPTY], _ struct{}) (*int, error) {
			var details int
			if err := ctx.HeartbeatDetails(&details); err != nil {
				return nil, err
			}

			return &details, nil
		},
	)

	env := flowtest.NewActivityEnv(flowtest.Config{Group: group})
	env.SetHeartbeatDetails(42)

	got, err := flowtest.ExecuteActivity(env, act, flow.EMPTY{}, struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	if *got != 42 {
		t.Fatalf("got %d", *got)
	}
}
