package flow

import (
	"context"
	"time"

	"go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"google.golang.org/protobuf/types/known/durationpb"
)

type Config struct {
	HostPort  string
	Namespace string
	TaskQueue string
}

type SDK struct {
	nsCli  client.NamespaceClient
	cli    client.Client
	replay worker.WorkflowReplayer
	w      worker.Worker

	taskQ     string
	namespace string
	hostport  string
}

func NewSDK(cfg Config) (*SDK, error) {
	sdk := &SDK{
		taskQ:     cfg.TaskQueue,
		namespace: cfg.Namespace,
		hostport:  cfg.HostPort,
		replay:    worker.NewWorkflowReplayer(),
	}

	err := sdk.invoke()
	if err != nil {
		return nil, err
	}

	return sdk, nil
}

func (sdk *SDK) invoke() error {
	var err error
	sdk.nsCli, err = client.NewNamespaceClient(client.Options{
		HostPort: sdk.hostport,
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

	sdk.cli, err = client.NewLazyClient(
		client.Options{
			HostPort:  sdk.hostport,
			Namespace: sdk.namespace,
		},
	)
	if err != nil {
		return err
	}

	sdk.w = worker.New(
		sdk.cli,
		sdk.taskQ,
		worker.Options{
			DisableRegistrationAliasing: true,
		},
	)

	return nil
}

func (sdk *SDK) Start() error {
	return sdk.w.Start()
}

func (sdk *SDK) Stop() {
	sdk.w.Stop()
}

func (sdk *SDK) TaskQueue() string {
	return sdk.taskQ
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
