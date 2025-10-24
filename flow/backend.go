package flow

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/log"
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
	Namespace() string
	Group() string
	Start() error
	Stop()
	Client() client.Client
	ScheduleClient() client.ScheduleClient
	UpdateWorkflowRetentionPeriod(ctx context.Context, d time.Duration) error
	DataConverter() converter.DataConverter
}

type BackendConfig struct {
	HostPort      string
	Secure        bool
	Namespace     string
	Group         string
	TaskQueue     string
	DataConverter converter.DataConverter
	Credentials   client.Credentials
	Logger        log.Logger
	WorkerOptions worker.Options
}

var _ Backend = (*realBackend)(nil)

type realBackend struct {
	l     log.Logger
	cli   client.Client
	nsCli client.NamespaceClient
	creds client.Credentials
	dc    converter.DataConverter
	w     worker.Worker
	wOpts worker.Options

	ns       string
	group    string
	taskQ    string
	hostport string
	secure   bool
}

func NewBackend(cfg BackendConfig) (Backend, error) {
	// This worker option is forced to be true according to temporal-go-sdk.
	cfg.WorkerOptions.DisableRegistrationAliasing = true

	b := &realBackend{
		l:        cfg.Logger,
		dc:       cfg.DataConverter,
		creds:    cfg.Credentials,
		ns:       cfg.Namespace,
		group:    cfg.Group,
		taskQ:    cfg.TaskQueue,
		hostport: cfg.HostPort,
		secure:   cfg.Secure,
		wOpts:    cfg.WorkerOptions,
	}

	err := b.init()
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (r *realBackend) init() error {
	connOpt := client.ConnectionOptions{}
	if r.secure {
		connOpt.TLS = &tls.Config{}
	}

	var err error
	r.nsCli, err = client.NewNamespaceClient(client.Options{
		HostPort:          r.hostport,
		ConnectionOptions: connOpt,
	})
	if err != nil {
		return err
	}

	if _, err = r.nsCli.Describe(context.Background(), r.ns); err != nil {
		_ = r.nsCli.Register(
			context.Background(),
			&workflowservice.RegisterNamespaceRequest{
				Namespace:                        r.ns,
				WorkflowExecutionRetentionPeriod: &durationpb.Duration{Seconds: 72 * 3600},
			},
		)
	}

	clientOpt := client.Options{
		HostPort:          r.hostport,
		ConnectionOptions: connOpt,
		Namespace:         r.ns,
		DataConverter:     r.dc,
		Credentials:       r.creds,
		Logger:            r.l,
	}

	r.cli, err = client.NewLazyClient(clientOpt)
	if err != nil {
		return err
	}

	r.w = worker.New(
		r.cli,
		r.taskQ,
		r.wOpts,
	)

	return nil
}

func (r *realBackend) RegisterWorkflow(w any) {
	r.w.RegisterWorkflow(w)
}

func (r *realBackend) RegisterWorkflowWithOptions(w any, options workflow.RegisterOptions) {
	r.w.RegisterWorkflowWithOptions(w, options)
}

func (r *realBackend) RegisterDynamicWorkflow(w any, options workflow.DynamicRegisterOptions) {
	r.w.RegisterDynamicWorkflow(w, options)
}

func (r *realBackend) RegisterActivity(a any) {
	r.w.RegisterActivity(a)
}

func (r *realBackend) RegisterActivityWithOptions(a any, options activity.RegisterOptions) {
	r.w.RegisterActivityWithOptions(a, options)
}

func (r *realBackend) RegisterDynamicActivity(a any, options activity.DynamicRegisterOptions) {
	r.w.RegisterDynamicActivity(a, options)
}

func (r *realBackend) RegisterNexusService(service *nexus.Service) {
	r.w.RegisterNexusService(service)
}

func (r *realBackend) ExecuteWorkflow(
	ctx context.Context, options client.StartWorkflowOptions, workflow any, args ...any,
) (client.WorkflowRun, error) {
	return r.cli.ExecuteWorkflow(ctx, options, workflow, args...)
}

func (r *realBackend) TaskQueue() string {
	return r.taskQ
}

func (r *realBackend) Namespace() string {
	return r.ns
}

func (r *realBackend) Group() string {
	return r.group
}

func (r *realBackend) Start() error {
	return r.w.Start()
}

func (r *realBackend) Stop() {
	r.w.Stop()
}

func (r *realBackend) ScheduleClient() client.ScheduleClient {
	return r.cli.ScheduleClient()
}

func (r *realBackend) Client() client.Client {
	return r.cli
}

func (r *realBackend) DataConverter() converter.DataConverter {
	return r.dc
}

type UpdateNamespaceRequest struct {
	Description                      *string
	WorkflowExecutionRetentionPeriod *time.Duration
}

func (r *realBackend) UpdateWorkflowRetentionPeriod(ctx context.Context, d time.Duration) error {
	res, err := r.nsCli.Describe(ctx, r.ns)
	if err != nil {
		return err
	}

	if res.Config == nil {
		res.Config = &namespace.NamespaceConfig{}
	}
	res.Config.WorkflowExecutionRetentionTtl = durationpb.New(d)

	return r.nsCli.Update(
		ctx,
		&workflowservice.UpdateNamespaceRequest{
			Namespace: r.ns,
			Config:    res.Config,
		},
	)
}
