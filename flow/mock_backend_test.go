package flow

import (
	"context"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type mockBackend struct {
	group   string
	taskQ   string
	ns      string
	wfRegs  []string
	actRegs []string
	sched   client.ScheduleClient
}

func (m *mockBackend) RegisterWorkflow(any) {}

func (m *mockBackend) RegisterWorkflowWithOptions(_ any, options workflow.RegisterOptions) {
	m.wfRegs = append(m.wfRegs, options.Name)
}

func (m *mockBackend) RegisterDynamicWorkflow(any, workflow.DynamicRegisterOptions) {}

func (m *mockBackend) RegisterActivity(any) {}

func (m *mockBackend) RegisterActivityWithOptions(_ any, options activity.RegisterOptions) {
	m.actRegs = append(m.actRegs, options.Name)
}

func (m *mockBackend) RegisterDynamicActivity(any, activity.DynamicRegisterOptions) {}

func (m *mockBackend) RegisterNexusService(*nexus.Service) {}

func (m *mockBackend) ExecuteWorkflow(
	context.Context, client.StartWorkflowOptions, any, ...any,
) (client.WorkflowRun, error) {
	return nil, nil
}

func (m *mockBackend) StartWorkflow(client.StartWorkflowOptions, any, ...any) client.WithStartWorkflowOperation {
	return nil
}

func (m *mockBackend) TaskQueue() string { return m.taskQ }

func (m *mockBackend) Namespace() string { return m.ns }

func (m *mockBackend) Group() string { return m.group }

func (m *mockBackend) Start() error { return nil }

func (m *mockBackend) Stop() {}

func (m *mockBackend) Client() client.Client { return nil }

func (m *mockBackend) ScheduleClient() client.ScheduleClient { return m.sched }

func (m *mockBackend) UpdateWorkflowRetentionPeriod(context.Context, time.Duration) error {
	return nil
}

func (m *mockBackend) DataConverter() converter.DataConverter {
	return converter.GetDefaultDataConverter()
}

var (
	_ Backend         = (*mockBackend)(nil)
	_ worker.Registry = (*mockBackend)(nil)
)
