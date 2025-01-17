package flow

import (
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type (
	Context     = workflow.Context
	RetryPolicy = temporal.RetryPolicy
)

type EMPTY struct{}
