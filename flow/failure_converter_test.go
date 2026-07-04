package flow

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clubpay/ronykit/rony/errs"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestDefaultFailureConverterRoundTripsErrsError(t *testing.T) {
	fc := DefaultFailureConverter()

	original := errs.B().
		Code(errs.NotFound).
		Msg("missing item").
		Meta("request_id", "abc").
		Cause(errors.New("db miss")).
		Err()

	restored := roundTripFailure(t, fc, original)

	var got *errs.Error
	if !errors.As(restored, &got) {
		t.Fatalf("expected *errs.Error, got %T", restored)
	}
	if got.Code != errs.NotFound {
		t.Fatalf("unexpected code: %v", got.Code)
	}
	if got.Item != "missing item" {
		t.Fatalf("unexpected item: %q", got.Item)
	}
	if got.Meta["request_id"] != "abc" {
		t.Fatalf("unexpected meta: %v", got.Meta)
	}
	if got.Unwrap() == nil || got.Unwrap().Error() != "db miss" {
		t.Fatalf("unexpected underlying: %v", got.Unwrap())
	}
	if errs.Code(restored) != errs.NotFound {
		t.Fatalf("unexpected errs.Code: %v", errs.Code(restored))
	}
}

func TestDefaultFailureConverterRoundTripsWrappedErrsError(t *testing.T) {
	fc := DefaultFailureConverter()

	original := fmt.Errorf("activity failed: %w", &errs.Error{
		Code: errs.InvalidArgument,
		Item: "bad input",
	})
	restored := roundTripFailure(t, fc, original)

	var got *errs.Error
	if !errors.As(restored, &got) {
		t.Fatalf("expected *errs.Error, got %T", restored)
	}
	if errs.Code(restored) != errs.InvalidArgument {
		t.Fatalf("unexpected code: %v", errs.Code(restored))
	}
	if restored.Error() != original.Error() {
		t.Fatalf("unexpected message: %q", restored.Error())
	}
}

func TestDefaultFailureConverterActivityFailureRoundTrip(t *testing.T) {
	fc := DefaultFailureConverter()

	original := &errs.Error{Code: errs.NotFound, Item: "missing"}
	inner := fc.ErrorToFailure(original)
	restored := fc.FailureToError(&failurepb.Failure{
		Message: "activity error",
		FailureInfo: &failurepb.Failure_ActivityFailureInfo{
			ActivityFailureInfo: &failurepb.ActivityFailureInfo{
				ActivityType: &commonpb.ActivityType{Name: "TestActivity"},
			},
		},
		Cause: inner,
	})

	var got *errs.Error
	if !errors.As(restored, &got) {
		t.Fatalf("expected *errs.Error, got %T (%v)", restored, restored)
	}
	if errs.Code(restored) != errs.NotFound {
		t.Fatalf("unexpected code: %v", errs.Code(restored))
	}
}

func TestDefaultFailureConverterMarksBusinessCodesNonRetryable(t *testing.T) {
	fc := DefaultFailureConverter()

	businessFailure := fc.ErrorToFailure(&errs.Error{Code: errs.NotFound, Item: "missing"})
	if !businessFailure.GetApplicationFailureInfo().GetNonRetryable() {
		t.Fatal("expected business errs code to be non-retryable")
	}

	retryableFailure := fc.ErrorToFailure(&errs.Error{Code: errs.Unavailable, Item: "down"})
	if retryableFailure.GetApplicationFailureInfo().GetNonRetryable() {
		t.Fatal("expected infrastructure errs code to remain retryable")
	}
}

func TestDefaultFailureConverterFallsBackForGenericErrors(t *testing.T) {
	fc := DefaultFailureConverter()
	fallback := temporal.GetDefaultFailureConverter()

	original := errors.New("boom")
	gotFailure := fc.ErrorToFailure(original)
	wantFailure := fallback.ErrorToFailure(original)
	if gotFailure.GetMessage() != wantFailure.GetMessage() {
		t.Fatalf("unexpected failure message: %q", gotFailure.GetMessage())
	}

	restored := fc.FailureToError(gotFailure)
	if restored == nil || restored.Error() != "boom" {
		t.Fatalf("unexpected restored error: %v", restored)
	}
}

func TestDefaultFailureConverterUsesConfiguredDataConverter(t *testing.T) {
	dc := converter.NewCompositeDataConverter(converter.NewJSONPayloadConverter())
	fc := DefaultFailureConverter(WithFailureConverterDataConverter(dc))

	original := &errs.Error{Code: errs.PermissionDenied, Item: "denied"}
	restored := roundTripFailure(t, fc, original)

	if errs.Code(restored) != errs.PermissionDenied {
		t.Fatalf("unexpected code: %v", errs.Code(restored))
	}
}

func TestDefaultFailureConverterActivityBoundary(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	env.SetFailureConverter(DefaultFailureConverter())

	var calls atomic.Int32
	activity := func(context.Context) error {
		calls.Add(1)
		return &errs.Error{Code: errs.NotFound, Item: "missing"}
	}
	env.RegisterActivity(activity)
	env.ExecuteWorkflow(func(ctx workflow.Context) error {
		ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Hour,
		})
		return workflow.ExecuteActivity(ctx, activity).Get(ctx, nil)
	})

	if !env.IsWorkflowCompleted() {
		t.Fatal("expected workflow to complete")
	}

	wfErr := env.GetWorkflowError()
	var got *errs.Error
	if !errors.As(wfErr, &got) {
		t.Fatalf("expected *errs.Error from workflow error, got %T (%v)", wfErr, wfErr)
	}
	if errs.Code(wfErr) != errs.NotFound {
		t.Fatalf("unexpected code: %v", errs.Code(wfErr))
	}
	if calls.Load() != 1 {
		t.Fatalf("expected single activity attempt for non-retryable business error, got %d", calls.Load())
	}
}

func TestIsBusinessErrCode(t *testing.T) {
	cases := []struct {
		code errs.ErrCode
		want bool
	}{
		{errs.NotFound, true},
		{errs.InvalidArgument, true},
		{errs.Unavailable, false},
		{errs.Internal, false},
		{errs.Unknown, false},
		{errs.ResourceExhausted, false},
	}

	for _, tc := range cases {
		if got := isBusinessErrCode(tc.code); got != tc.want {
			t.Fatalf("isBusinessErrCode(%v) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

func roundTripFailure(t *testing.T, fc converter.FailureConverter, err error) error {
	t.Helper()

	failure := fc.ErrorToFailure(err)
	if failure == nil {
		t.Fatal("expected failure")
	}

	restored := fc.FailureToError(failure)
	if restored == nil {
		t.Fatal("expected restored error")
	}

	return restored
}
