package flow

import (
	"context"
	"crypto/tls"
	"reflect"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/types/known/durationpb"
)

type Backend interface {
	worker.Registry
	ExecuteWorkflow(
		ctx context.Context,
		options client.StartWorkflowOptions,
		workflow any,
		args ...any,
	) (client.WorkflowRun, error)
	TaskQueue() string
	Group() string
}

type Config struct {
	HostPort      string
	Secure        bool
	Namespace     string
	Group         string
	TaskQueue     string
	DataConverter converter.DataConverter
	Credentials   client.Credentials
}

type realBackend struct {
	cli   client.Client
	w     worker.Worker
	ns    string
	group string
	taskQ string
}

func (r realBackend) RegisterWorkflow(w any) {
	r.w.RegisterWorkflow(w)
}

func (r realBackend) RegisterWorkflowWithOptions(w any, options workflow.RegisterOptions) {
	r.w.RegisterWorkflowWithOptions(w, options)
}

func (r realBackend) RegisterActivity(a any) {
	r.w.RegisterActivity(a)
}

func (r realBackend) RegisterActivityWithOptions(a any, options activity.RegisterOptions) {
	r.w.RegisterActivityWithOptions(a, options)
}

func (r realBackend) RegisterNexusService(service *nexus.Service) {
	r.w.RegisterNexusService(service)
}

func (r realBackend) ExecuteWorkflow(
	ctx context.Context, options client.StartWorkflowOptions, workflow any, args ...any,
) (client.WorkflowRun, error) {
	return r.cli.ExecuteWorkflow(ctx, options, workflow, args...)
}

func (r realBackend) TaskQueue() string {
	return r.taskQ
}

func (r realBackend) Namespace() string {
	return r.ns
}

func (r realBackend) Group() string {
	return r.group
}

type SDK struct {
	nsCli client.NamespaceClient
	dc    converter.DataConverter
	creds client.Credentials
	b     realBackend

	namespace string
	hostport  string
	secure    bool
}

func NewSDK(cfg Config) (*SDK, error) {
	sdk := &SDK{
		b: realBackend{
			taskQ: cfg.TaskQueue,
			ns:    cfg.Namespace,
			group: cfg.Group,
		},
		dc:        cfg.DataConverter,
		creds:     cfg.Credentials,
		namespace: cfg.Namespace,
		hostport:  cfg.HostPort,
		secure:    cfg.Secure,
	}

	err := sdk.invoke()
	if err != nil {
		return nil, err
	}

	return sdk, nil
}

func (sdk *SDK) invoke() error {
	connOpt := client.ConnectionOptions{}
	if sdk.secure {
		connOpt.TLS = &tls.Config{}
	}

	var err error
	sdk.nsCli, err = client.NewNamespaceClient(client.Options{
		HostPort:          sdk.hostport,
		ConnectionOptions: connOpt,
	})
	if err != nil {
		return err
	}

	if _, err = sdk.nsCli.Describe(context.Background(), sdk.namespace); err != nil {
		_ = sdk.nsCli.Register(
			context.Background(),
			&workflowservice.RegisterNamespaceRequest{
				Namespace:                        sdk.namespace,
				WorkflowExecutionRetentionPeriod: &durationpb.Duration{Seconds: 72 * 3600},
			},
		)
	}

	clientOpt := client.Options{
		HostPort:          sdk.hostport,
		ConnectionOptions: connOpt,
		Namespace:         sdk.namespace,
		DataConverter:     sdk.dc,
		Credentials:       sdk.creds,
	}

	sdk.b.cli, err = client.NewLazyClient(clientOpt)
	if err != nil {
		return err
	}

	sdk.b.w = worker.New(
		sdk.b.cli,
		sdk.b.taskQ,
		worker.Options{
			DisableRegistrationAliasing: true,
		},
	)

	return nil
}

func (sdk *SDK) Start() error {
	return sdk.b.w.Start()
}

func (sdk *SDK) Stop() {
	sdk.b.w.Stop()
}

func (sdk *SDK) TaskQueue() string {
	return sdk.b.taskQ
}

func (sdk *SDK) Init() {
	sdk.InitWithState(EMPTY{})
}

func (sdk *SDK) InitWithState(state any) {
	for stateType, w := range registeredWorkflows {
		if stateType == reflect.TypeOf(state) {
			for _, t := range w {
				t.initWithStateAny(&sdk.b, state)
			}
		}
	}
	for stateType, w := range registeredActivities {
		if stateType == reflect.TypeOf(state) {
			for _, t := range w {
				t.initWithStateAny(sdk.b, state)
			}
		}
	}
}

type UpdateNamespaceRequest struct {
	Description                      *string
	WorkflowExecutionRetentionPeriod *time.Duration
}

func (sdk *SDK) UpdateWorkflowRetentionPeriod(ctx context.Context, d time.Duration) error {
	res, err := sdk.nsCli.Describe(ctx, sdk.namespace)
	if err != nil {
		return err
	}

	if res.Config == nil {
		res.Config = &namespace.NamespaceConfig{}
	}
	res.Config.WorkflowExecutionRetentionTtl = durationpb.New(d)

	return sdk.nsCli.Update(
		ctx,
		&workflowservice.UpdateNamespaceRequest{
			Namespace: sdk.namespace,
			Config:    res.Config,
		},
	)
}

var _StateCtxKey = struct{}{}

func GetState[STATE any](ctx Context) STATE {
	return ctx.Value(_StateCtxKey).(STATE)
}

type temporalEntityT interface {
	initWithStateAny(sdk Backend, state any)
}

var (
	registeredWorkflows  = make(map[reflect.Type][]temporalEntityT)
	registeredActivities = make(map[reflect.Type][]temporalEntityT)
)
