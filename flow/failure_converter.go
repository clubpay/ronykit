package flow

import (
	"errors"
	"fmt"

	"github.com/clubpay/ronykit/rony/errs"
	"github.com/clubpay/ronykit/rony/errs/errmarshalling"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/protobuf/proto"
)

const errsErrorFailureType = "github.com/clubpay/ronykit/rony/errs.Error"

// FailureConverterConfig configures a Temporal FailureConverter for *errs.Error.
type FailureConverterConfig struct {
	DataConverter converter.DataConverter
	Fallback      converter.FailureConverter
}

// FailureConverterOption configures DefaultFailureConverter.
type FailureConverterOption func(*FailureConverterConfig)

// WithFailureConverterDataConverter sets the DataConverter used to encode failure details.
func WithFailureConverterDataConverter(dc converter.DataConverter) FailureConverterOption {
	return func(cfg *FailureConverterConfig) {
		cfg.DataConverter = dc
	}
}

// WithFailureConverterFallback sets the converter used for non-*errs.Error failures.
func WithFailureConverterFallback(fc converter.FailureConverter) FailureConverterOption {
	return func(cfg *FailureConverterConfig) {
		cfg.Fallback = fc
	}
}

// DefaultFailureConverter returns a FailureConverter that round-trips *errs.Error values
// across Temporal activity, workflow, and client boundaries using errmarshalling.
// Business-domain errs codes are marked non-retryable in Temporal failures.
func DefaultFailureConverter(opts ...FailureConverterOption) converter.FailureConverter {
	cfg := FailureConverterConfig{
		DataConverter: converter.GetDefaultDataConverter(),
		Fallback:      temporal.GetDefaultFailureConverter(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &errsFailureConverter{
		dc:       cfg.DataConverter,
		fallback: cfg.Fallback,
	}
}

type errsFailureConverter struct {
	dc       converter.DataConverter
	fallback converter.FailureConverter
}

func (c *errsFailureConverter) ErrorToFailure(err error) *failurepb.Failure {
	if err == nil {
		return nil
	}

	var e *errs.Error
	if errors.As(err, &e) {
		return c.errsErrorToFailure(err, e)
	}

	return c.fallback.ErrorToFailure(err)
}

func (c *errsFailureConverter) FailureToError(failure *failurepb.Failure) error {
	if failure == nil {
		return nil
	}

	if info := failure.GetApplicationFailureInfo(); info != nil && info.GetType() == errsErrorFailureType {
		return c.failureToErrsError(info)
	}

	if cause := failure.GetCause(); cause != nil {
		causeErr := c.FailureToError(cause)
		if causeErr != nil && containsErrsError(causeErr) {
			shell := proto.Clone(failure).(*failurepb.Failure)
			shell.Cause = nil

			shellErr := c.fallback.FailureToError(shell)
			if shellErr == nil {
				return causeErr
			}

			return &chainedFailureError{
				outer: shellErr,
				cause: causeErr,
			}
		}
	}

	return c.fallback.FailureToError(failure)
}

func (c *errsFailureConverter) errsErrorToFailure(err error, e *errs.Error) *failurepb.Failure {
	payload, convErr := c.dc.ToPayload(errmarshalling.Marshal(err))
	if convErr != nil {
		return c.fallback.ErrorToFailure(err)
	}

	return &failurepb.Failure{
		Message: err.Error(),
		Source:  "GoSDK",
		FailureInfo: &failurepb.Failure_ApplicationFailureInfo{
			ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{
				Type:         errsErrorFailureType,
				NonRetryable: isBusinessErrCode(e.Code),
				Details: &commonpb.Payloads{
					Payloads: []*commonpb.Payload{payload},
				},
			},
		},
	}
}

func (c *errsFailureConverter) failureToErrsError(info *failurepb.ApplicationFailureInfo) error {
	data, err := failureDetailsPayload(info.GetDetails(), c.dc)
	if err != nil {
		return c.fallback.FailureToError(&failurepb.Failure{
			FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: info},
		})
	}

	unmarshaled, err := errmarshalling.Unmarshal(data)
	if err != nil {
		return c.fallback.FailureToError(&failurepb.Failure{
			FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: info},
		})
	}

	return unmarshaled
}

func failureDetailsPayload(details *commonpb.Payloads, dc converter.DataConverter) ([]byte, error) {
	if details == nil || len(details.GetPayloads()) == 0 {
		return nil, fmt.Errorf("flow: missing failure details payloads")
	}

	var data []byte
	if err := dc.FromPayload(details.GetPayloads()[0], &data); err != nil {
		return nil, fmt.Errorf("flow: decode failure details: %w", err)
	}

	return data, nil
}

// isBusinessErrCode reports whether code represents a domain/business error that
// should not trigger automatic Temporal activity retries.
func isBusinessErrCode(code errs.ErrCode) bool {
	switch code {
	case errs.InvalidArgument, errs.NotFound, errs.AlreadyExists,
		errs.PermissionDenied, errs.FailedPrecondition, errs.Aborted,
		errs.OutOfRange, errs.Unimplemented, errs.Unauthenticated,
		errs.Canceled:
		return true
	default:
		return false
	}
}

func containsErrsError(err error) bool {
	var e *errs.Error

	return errors.As(err, &e)
}

// chainedFailureError preserves Temporal wrapper errors such as ActivityError
// while exposing a converted *errs.Error through the unwrap chain.
type chainedFailureError struct {
	outer error
	cause error
}

func (e *chainedFailureError) Error() string {
	return fmt.Sprintf("%v: %v", e.outer, e.cause)
}

func (e *chainedFailureError) Unwrap() error {
	return e.cause
}

func (e *chainedFailureError) As(target any) bool {
	if errors.As(e.outer, target) {
		return true
	}

	return errors.As(e.cause, target)
}
