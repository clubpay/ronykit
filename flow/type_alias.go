package flow

import (
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type (
	// Context is the deterministic context passed through workflow code.
	Context = workflow.Context
	// CancelFunc cancels a workflow context from WithCancel or DisconnectedContext.
	CancelFunc = workflow.CancelFunc

	// RetryPolicy controls retries for activities, child workflows, and schedules.
	RetryPolicy = temporal.RetryPolicy

	// Logger is accepted by BackendConfig and SDKConfig. NewZapAdapter returns one.
	Logger = log.Logger

	// Credentials authenticates the Temporal client. Build a value with
	// NewAPIKeyStaticCredentials, NewAPIKeyDynamicCredentials, or NewMTLSCredentials.
	Credentials = client.Credentials

	// DataConverter serializes workflow and activity payloads.
	DataConverter = converter.DataConverter
	// FailureConverter maps Go errors to Temporal failures and back.
	FailureConverter = converter.FailureConverter
	// PayloadCodec transforms payloads inside a DataConverter.
	PayloadCodec = converter.PayloadCodec

	// WorkerOptions configures worker concurrency and polling. Set it on BackendConfig.
	WorkerOptions = worker.Options

	// Client is the Temporal service client. Prefer SDK and Workflow helpers for
	// the operations flow already wraps.
	Client = client.Client

	// StartWorkflowOptions is the low-level start request accepted by Backend.
	// Application code should use ExecuteWorkflowOptions.
	StartWorkflowOptions = client.StartWorkflowOptions
	// WithStartWorkflowOperation starts a workflow as part of an update-with-start.
	WithStartWorkflowOperation = client.WithStartWorkflowOperation

	// ApplicationError is a user-defined failure that crosses activity and workflow boundaries.
	ApplicationError = temporal.ApplicationError
	// ApplicationErrorOptions controls NewApplicationErrorWithOptions.
	ApplicationErrorOptions = temporal.ApplicationErrorOptions
	// CanceledError is returned when an activity or workflow is canceled.
	CanceledError = temporal.CanceledError
	// TimeoutError is returned when an activity or workflow times out.
	TimeoutError = temporal.TimeoutError
	// TerminatedError is returned when a workflow is terminated.
	TerminatedError = temporal.TerminatedError
	// ActivityError wraps a failure that originated in an activity.
	ActivityError = temporal.ActivityError
	// WorkflowExecutionError wraps a failure of a workflow execution.
	WorkflowExecutionError = temporal.WorkflowExecutionError
	// ChildWorkflowExecutionError wraps a failure of a child workflow.
	ChildWorkflowExecutionError = temporal.ChildWorkflowExecutionError
)

// EMPTY is the state type for workflows and activities that carry no shared state.
type EMPTY struct{}
