package flow

import (
	"errors"

	"go.temporal.io/sdk/temporal"
)

// NewApplicationError returns a retryable application error with message, errType,
// and optional details. Put errType in RetryPolicy.NonRetryableErrorTypes to skip
// retries for that type.
func NewApplicationError(message, errType string, details ...any) error {
	return temporal.NewApplicationError(message, errType, details...)
}

// NewApplicationErrorWithCause returns a retryable application error with cause.
func NewApplicationErrorWithCause(message, errType string, cause error, details ...any) error {
	return temporal.NewApplicationErrorWithCause(message, errType, cause, details...)
}

// NewNonRetryableApplicationError returns an application error that Temporal will not retry.
func NewNonRetryableApplicationError(message, errType string, cause error, details ...any) error {
	return temporal.NewNonRetryableApplicationError(message, errType, cause, details...)
}

// NewApplicationErrorWithOptions returns an application error controlled by options.
func NewApplicationErrorWithOptions(message, errType string, options ApplicationErrorOptions) error {
	return temporal.NewApplicationErrorWithOptions(message, errType, options)
}

// NewCanceledError returns an error that marks an activity or child workflow as canceled.
func NewCanceledError(details ...any) error {
	return temporal.NewCanceledError(details...)
}

// IsApplicationError reports whether err is an ApplicationError and returns it.
func IsApplicationError(err error) (bool, *ApplicationError) {
	var applicationError *ApplicationError

	ok := errors.As(err, &applicationError)

	return ok, applicationError
}

// IsCanceledError reports whether err is a CanceledError.
func IsCanceledError(err error) bool {
	return temporal.IsCanceledError(err)
}

// IsTimeoutError reports whether err is a TimeoutError.
func IsTimeoutError(err error) bool {
	return temporal.IsTimeoutError(err)
}

// IsTerminatedError reports whether err is a TerminatedError.
func IsTerminatedError(err error) bool {
	return temporal.IsTerminatedError(err)
}

// IsWorkflowExecutionAlreadyStartedError reports whether err means a workflow
// with the same ID is already running, including the child-workflow form.
func IsWorkflowExecutionAlreadyStartedError(err error) bool {
	return temporal.IsWorkflowExecutionAlreadyStartedError(err)
}

// IsPanicError reports whether err is a workflow or activity panic.
func IsPanicError(err error) bool {
	return temporal.IsPanicError(err)
}
