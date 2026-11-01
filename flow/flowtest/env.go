// Package flowtest runs flow workflows and activities without importing the Temporal SDK.
package flowtest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/clubpay/ronykit/flow"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

// Config selects which definitions a test environment registers.
// An empty Group registers every group. Set Group in tests so one case does
// not register definitions created by another.
type Config struct {
	Group            string
	TaskQueue        string
	DataConverter    flow.DataConverter
	FailureConverter flow.FailureConverter
	Logger           flow.Logger
}

// WorkflowEnv executes flow workflows in memory.
// Call Init or InitWithState before Execute, OnActivity, or OnWorkflow.
type WorkflowEnv struct {
	env     *testsuite.TestWorkflowEnvironment
	backend *testBackend
}

// ActivityEnv executes one activity outside a workflow.
type ActivityEnv struct {
	env     *testsuite.TestActivityEnvironment
	backend *testBackend
}

// NewWorkflowEnv returns an environment that runs workflows and their activities.
func NewWorkflowEnv(cfg Config) *WorkflowEnv {
	suite := newSuite(cfg)
	env := suite.NewTestWorkflowEnvironment()
	if cfg.DataConverter != nil {
		env.SetDataConverter(cfg.DataConverter)
	}

	env.SetFailureConverter(failureConverter(cfg))

	return &WorkflowEnv{
		env:     env,
		backend: newTestBackend(cfg, env),
	}
}

// NewActivityEnv returns an environment that runs activities directly.
func NewActivityEnv(cfg Config) *ActivityEnv {
	suite := newSuite(cfg)
	env := suite.NewTestActivityEnvironment()
	if cfg.DataConverter != nil {
		env.SetDataConverter(cfg.DataConverter)
	}

	env.SetFailureConverter(failureConverter(cfg))

	return &ActivityEnv{
		env:     env,
		backend: newTestBackend(cfg, nil),
	}
}

func (e *ActivityEnv) register(state any) {
	e.backend.activities = e.env
	flow.RegisterState(e.backend, state)
}

// SetHeartbeatDetails is returned by activity heartbeat reads during ExecuteActivity.
func (e *ActivityEnv) SetHeartbeatDetails(details any) {
	e.env.SetHeartbeatDetails(details)
}

// SetWorkerStopChannel is closed by the test to simulate worker shutdown.
func (e *ActivityEnv) SetWorkerStopChannel(stop chan struct{}) {
	e.env.SetWorkerStopChannel(stop)
}

// SetTimeout bounds a single activity execution.
func (e *ActivityEnv) SetTimeout(d time.Duration) {
	e.env.SetTestTimeout(d)
}

// ExecuteActivity registers the activity for state on the first call, then runs it.
// Later calls keep the state captured by that first registration.
func ExecuteActivity[REQ, RES, STATE any](
	env *ActivityEnv, act *flow.Activity[REQ, RES, STATE], state STATE, req REQ,
) (*RES, error) {
	env.register(state)

	val, err := env.env.ExecuteActivity(act.Name, req)
	if err != nil {
		return nil, err
	}

	var out RES
	if err = val.Get(&out); err != nil {
		return nil, err
	}

	return &out, nil
}

// Init registers workflows and activities that use flow.EMPTY.
func (e *WorkflowEnv) Init() {
	e.InitWithState(flow.EMPTY{})
}

// InitWithState registers workflows, activities, and factories for state.
func (e *WorkflowEnv) InitWithState(state any) {
	flow.RegisterState(e.backend, state)
}

// SetTimeout bounds the workflow run.
func (e *WorkflowEnv) SetTimeout(d time.Duration) {
	e.env.SetTestTimeout(d)
}

// After runs fn after delay on the workflow clock. Register it before Execute.
func (e *WorkflowEnv) After(delay time.Duration, fn func()) {
	e.env.RegisterDelayedCallback(fn, delay)
}

// Signal delivers a signal to the workflow under test.
func (e *WorkflowEnv) Signal(name string, arg any) {
	e.env.SignalWorkflow(name, arg)
}

// Signal delivers a typed signal to the workflow under test.
func Signal[T any](env *WorkflowEnv, sig flow.Signal[T], arg T) {
	env.Signal(sig.Name, arg)
}

// Cancel cancels the workflow under test.
func (e *WorkflowEnv) Cancel() {
	e.env.CancelWorkflow()
}

// IgnoreExternalSignals accepts every signal sent to a workflow outside this environment.
func (e *WorkflowEnv) IgnoreExternalSignals() {
	e.env.OnSignalExternalWorkflow(
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(nil).Maybe()
}

// OnExternalSignal stubs one signal sent to a workflow outside this environment.
func (e *WorkflowEnv) OnExternalSignal(workflowID, signalName string) *SignalMock {
	call := e.env.OnSignalExternalWorkflow(
		mock.Anything, workflowID, mock.Anything, signalName, mock.Anything,
	)

	return &SignalMock{call: call}
}

// AssertExpectations fails t when a mock was not called as declared.
func (e *WorkflowEnv) AssertExpectations(t *testing.T) bool {
	return e.env.AssertExpectations(t)
}

// Execute runs a registered workflow and returns its result.
// One environment can execute one workflow.
func Execute[REQ, RES, STATE any](
	env *WorkflowEnv, w *flow.Workflow[REQ, RES, STATE], req REQ,
) (*RES, error) {
	env.env.ExecuteWorkflow(w.Name, req)
	if !env.env.IsWorkflowCompleted() {
		return nil, errors.New("flowtest: workflow did not complete")
	}

	var out RES

	err := env.env.GetWorkflowResult(&out)
	if err != nil {
		return nil, err
	}

	return &out, nil
}

// OnActivity stubs an activity. Pass one request to match that value, or none to match any request.
func OnActivity[REQ, RES, STATE any](
	env *WorkflowEnv, act *flow.Activity[REQ, RES, STATE], match ...REQ,
) *Mock[RES] {
	return onActivity[RES](env, act.Name, requestArg(match))
}

// OnActivityFactory stubs an activity factory. The factory must already be bound with InitWithState.
func OnActivityFactory[REQ, RES, STATE any](
	env *WorkflowEnv, act *flow.ActivityFactory[REQ, RES, STATE], match ...REQ,
) *Mock[RES] {
	return onActivity[RES](env, act.Name(), requestArg(match))
}

// OnWorkflow stubs a child workflow. Pass one request to match that value, or none to match any request.
func OnWorkflow[REQ, RES, STATE any](
	env *WorkflowEnv, w *flow.Workflow[REQ, RES, STATE], match ...REQ,
) *Mock[RES] {
	arg := requestArg(match)
	call := env.env.OnWorkflow(w.Name, mock.Anything, arg)

	return &Mock[RES]{call: call}
}

func onActivity[RES any](env *WorkflowEnv, name string, arg any) *Mock[RES] {
	call := env.env.OnActivity(name, mock.Anything, arg)

	return &Mock[RES]{call: call}
}

func requestArg[REQ any](match []REQ) any {
	switch len(match) {
	case 0:
		return mock.Anything
	case 1:
		return match[0]
	default:
		panic("flowtest: activity and workflow mocks accept at most one request")
	}
}

// Mock is a typed expectation for an activity or child workflow.
type Mock[T any] struct {
	call *testsuite.MockCallWrapper
}

func (m *Mock[T]) Return(out *T, err error) *Mock[T] {
	m.call.Return(out, err)

	return m
}

func (m *Mock[T]) Once() *Mock[T] {
	m.call.Once()

	return m
}

func (m *Mock[T]) Times(n int) *Mock[T] {
	m.call.Times(n)

	return m
}

func (m *Mock[T]) Maybe() *Mock[T] {
	m.call.Maybe()

	return m
}

func (m *Mock[T]) Never() *Mock[T] {
	m.call.Never()

	return m
}

// After waits on the workflow clock before the mock returns.
func (m *Mock[T]) After(d time.Duration) *Mock[T] {
	m.call.After(d)

	return m
}

// SignalMock is an expectation for a signal sent outside the workflow under test.
type SignalMock struct {
	call *testsuite.MockCallWrapper
}

func (m *SignalMock) Return(err error) *SignalMock {
	m.call.Return(err)

	return m
}

func (m *SignalMock) Once() *SignalMock {
	m.call.Once()

	return m
}

func (m *SignalMock) Times(n int) *SignalMock {
	m.call.Times(n)

	return m
}

func (m *SignalMock) Maybe() *SignalMock {
	m.call.Maybe()

	return m
}

type activityRegister interface {
	RegisterActivity(any)
	RegisterActivityWithOptions(any, activity.RegisterOptions)
	RegisterDynamicActivity(any, activity.DynamicRegisterOptions)
}

type workflowRegister interface {
	RegisterWorkflow(any)
	RegisterWorkflowWithOptions(any, workflow.RegisterOptions)
	RegisterDynamicWorkflow(any, workflow.DynamicRegisterOptions)
	RegisterNexusService(*nexus.Service)
}

type testBackend struct {
	group      string
	taskQ      string
	dc         flow.DataConverter
	activities activityRegister
	workflows  workflowRegister
}

func newTestBackend(cfg Config, workflows workflowRegister) *testBackend {
	taskQ := cfg.TaskQueue
	if taskQ == "" {
		taskQ = "test"
	}

	b := &testBackend{
		group:     cfg.Group,
		taskQ:     taskQ,
		dc:        cfg.DataConverter,
		workflows: workflows,
	}
	if reg, ok := workflows.(activityRegister); ok {
		b.activities = reg
	}

	return b
}

func (b *testBackend) MatchesGroup(group string) bool {
	if b.group == "" {
		return true
	}

	return b.group == group
}

func (b *testBackend) RegisterWorkflow(w any) {
	if b.workflows != nil {
		b.workflows.RegisterWorkflow(w)
	}
}

func (b *testBackend) RegisterWorkflowWithOptions(w any, options workflow.RegisterOptions) {
	if b.workflows != nil {
		b.workflows.RegisterWorkflowWithOptions(w, options)
	}
}

func (b *testBackend) RegisterDynamicWorkflow(w any, options workflow.DynamicRegisterOptions) {
	if b.workflows != nil {
		b.workflows.RegisterDynamicWorkflow(w, options)
	}
}

func (b *testBackend) RegisterActivity(a any) {
	if b.activities != nil {
		b.activities.RegisterActivity(a)
	}
}

func (b *testBackend) RegisterActivityWithOptions(a any, options activity.RegisterOptions) {
	if b.activities != nil {
		b.activities.RegisterActivityWithOptions(a, options)
	}
}

func (b *testBackend) RegisterDynamicActivity(a any, options activity.DynamicRegisterOptions) {
	if b.activities != nil {
		b.activities.RegisterDynamicActivity(a, options)
	}
}

func (b *testBackend) RegisterNexusService(service *nexus.Service) {
	if b.workflows != nil {
		b.workflows.RegisterNexusService(service)
	}
}

func (b *testBackend) ExecuteWorkflow(
	context.Context, flow.StartWorkflowOptions, any, ...any,
) (client.WorkflowRun, error) {
	return nil, errors.New("flowtest: start workflows with flowtest.Execute")
}

func (b *testBackend) StartWorkflow(
	flow.StartWorkflowOptions, any, ...any,
) flow.WithStartWorkflowOperation {
	return nil
}

func (b *testBackend) TaskQueue() string { return b.taskQ }

func (b *testBackend) Namespace() string { return "test" }

func (b *testBackend) Group() string { return b.group }

func (b *testBackend) Start() error { return nil }

func (b *testBackend) Stop() {}

func (b *testBackend) Client() flow.Client { return nil }

func (b *testBackend) ScheduleClient() flow.ScheduleClient { return nil }

func (b *testBackend) UpdateWorkflowRetentionPeriod(context.Context, time.Duration) error {
	return errors.New("flowtest: namespace updates are not supported")
}

func (b *testBackend) DataConverter() flow.DataConverter {
	if b.dc == nil {
		return converter.GetDefaultDataConverter()
	}

	return b.dc
}

func newSuite(cfg Config) testsuite.WorkflowTestSuite {
	var suite testsuite.WorkflowTestSuite
	if cfg.Logger != nil {
		suite.SetLogger(cfg.Logger)
	}

	suite.SetDisableRegistrationAliasing(true)

	return suite
}

func failureConverter(cfg Config) flow.FailureConverter {
	if cfg.FailureConverter != nil {
		return cfg.FailureConverter
	}

	var opts []flow.FailureConverterOption
	if cfg.DataConverter != nil {
		opts = append(opts, flow.WithFailureConverterDataConverter(cfg.DataConverter))
	}

	return flow.DefaultFailureConverter(opts...)
}

var (
	_ flow.Backend      = (*testBackend)(nil)
	_ flow.GroupMatcher = (*testBackend)(nil)
)
