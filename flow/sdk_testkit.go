package flow

import (
	"context"
	"reflect"
	"testing"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

var _ Backend = (*SDKTestKit)(nil)

type SDKTestKit struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env   *testsuite.TestWorkflowEnvironment
	group string
}

func NewSDKTestKit(group string) *SDKTestKit {
	kit := &SDKTestKit{
		group: group,
	}
	kit.env = kit.NewTestWorkflowEnvironment()

	return kit
}

func (sdk *SDKTestKit) RunTest(t *testing.T) {
	suite.Run(t, sdk)
}

func (sdk *SDKTestKit) ENV() *testsuite.TestWorkflowEnvironment {
	return sdk.env
}

func (sdk *SDKTestKit) InitWithState(state any) {
	for stateType, w := range registeredWorkflows {
		if stateType == reflect.TypeOf(state) {
			for _, t := range w {
				t.initWithStateAny(sdk, state)
			}
		}
	}
	for stateType, w := range registeredActivities {
		if stateType == reflect.TypeOf(state) {
			for _, t := range w {
				t.initWithStateAny(sdk, state)
			}
		}
	}
}

func (sdk *SDKTestKit) RegisterWorkflow(w interface{}) {
	sdk.RegisterWorkflow(w)

}

func (sdk *SDKTestKit) RegisterWorkflowWithOptions(w interface{}, options workflow.RegisterOptions) {
	sdk.env.RegisterWorkflowWithOptions(w, options)
}

func (sdk *SDKTestKit) RegisterDynamicWorkflow(w interface{}, options workflow.DynamicRegisterOptions) {
	sdk.env.RegisterDynamicWorkflow(w, options)
}

func (sdk *SDKTestKit) RegisterActivity(a interface{}) {
	sdk.env.RegisterActivity(a)
}

func (sdk *SDKTestKit) RegisterActivityWithOptions(a interface{}, options activity.RegisterOptions) {
	sdk.env.RegisterActivityWithOptions(a, options)
}

func (sdk *SDKTestKit) RegisterDynamicActivity(a interface{}, options activity.DynamicRegisterOptions) {
	sdk.env.RegisterDynamicActivity(a, options)
}

func (sdk *SDKTestKit) RegisterNexusService(service *nexus.Service) {
	sdk.env.RegisterNexusService(service)
}

func (sdk *SDKTestKit) ExecuteWorkflow(
	ctx context.Context, options client.StartWorkflowOptions, workflow any, args ...any,
) (client.WorkflowRun, error) {
	sdk.env.ExecuteWorkflow(workflow, args...)

	return nil, nil
}

func (sdk *SDKTestKit) TaskQueue() string {
	return ""
}

func (sdk *SDKTestKit) Namespace() string {
	return "test"
}

func (sdk *SDKTestKit) Group() string {
	return sdk.group
}
