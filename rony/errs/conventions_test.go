package errs_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/clubpay/ronykit/rony/errs"
	"github.com/clubpay/ronykit/rony/errs/errmarshalling"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	err := errs.New(errs.PermissionDenied, "ACCESS_DENIED")

	require.Equal(t, errs.PermissionDenied, errs.Code(err))
	require.Equal(t, "ACCESS_DENIED", errs.Item(err))
}

func TestNewf(t *testing.T) {
	t.Parallel()

	err := errs.Newf(errs.InvalidArgument, "SESSION_OPTION_INVALID_%s", "MODE")

	require.Equal(t, errs.InvalidArgument, errs.Code(err))
	require.Equal(t, "SESSION_OPTION_INVALID_MODE", errs.Item(err))
}

func TestSentinel(t *testing.T) {
	t.Parallel()

	s := errs.Sentinel(errs.NotFound, "AUTOMATION_NOT_FOUND")

	require.ErrorIs(t, s, s)
	require.Equal(t, errs.NotFound, errs.Code(s))
	require.Equal(t, "AUTOMATION_NOT_FOUND", errs.Item(s))
}

func TestSpecifyKeepsSentinelAndSpecificItem(t *testing.T) {
	t.Parallel()

	sentinel := errs.Sentinel(errs.InvalidArgument, "INVALID_ARGUMENT")

	err := errs.Specify(sentinel, "TRIGGER_GITHUB_ORG_REQUIRED")

	require.ErrorIs(t, err, sentinel)
	require.Equal(t, errs.InvalidArgument, errs.Code(err))
	require.Equal(t, "TRIGGER_GITHUB_ORG_REQUIRED", errs.Item(err))
}

func TestSpecifyNilSentinel(t *testing.T) {
	t.Parallel()

	err := errs.Specify(nil, "SOMETHING")

	require.Equal(t, errs.Unknown, errs.Code(err))
	require.Equal(t, "SOMETHING", errs.Item(err))
}

func TestSpecifyf(t *testing.T) {
	t.Parallel()

	sentinel := errs.Sentinel(errs.InvalidArgument, "INVALID_ARGUMENT")

	err := errs.Specifyf(sentinel, "SESSION_OPTION_INVALID_%s", "MODE")

	require.ErrorIs(t, err, sentinel)
	require.Equal(t, "SESSION_OPTION_INVALID_MODE", errs.Item(err))
}

func TestAnnotatePreservesWireIdentity(t *testing.T) {
	t.Parallel()

	sentinel := errs.Sentinel(errs.AlreadyExists, "AUTOMATION_NAME_CONFLICT")

	err := errs.Annotate(errs.Specify(sentinel, "AUTOMATION_NAME_CONFLICT"), "automation.Create")

	require.ErrorIs(t, err, sentinel)
	require.Equal(t, errs.AlreadyExists, errs.Code(err))
	require.Equal(t, "AUTOMATION_NAME_CONFLICT", errs.Item(err))
	require.Equal(t, "automation.Create", errs.Meta(err)["op"])
}

func TestAnnotateNil(t *testing.T) {
	t.Parallel()

	require.NoError(t, errs.Annotate(nil, "op"))
}

func TestWithDetailsPreservesIdentity(t *testing.T) {
	t.Parallel()

	sentinel := errs.Sentinel(errs.FailedPrecondition, "FORMULA_NOT_IN_PACK")

	err := errs.WithDetails(sentinel, errs.Reason{Message: "formula \"x\" not attached"})

	require.ErrorIs(t, err, sentinel)
	require.Equal(t, errs.FailedPrecondition, errs.Code(err))
	require.Equal(t, "FORMULA_NOT_IN_PACK", errs.Item(err))

	det, ok := errs.DetailsAs[errs.Reason](err)
	require.True(t, ok)
	require.Equal(t, "formula \"x\" not attached", det.Message)
}

func TestWithDetailsPreservesMetaFromAnnotate(t *testing.T) {
	t.Parallel()

	err := errs.Annotate(errs.Sentinel(errs.Internal, "AUTOMATION_INTERNAL"), "automation.Create")
	err = errs.WithDetails(err, errs.Reason{Message: "boom"})

	require.Equal(t, "automation.Create", errs.Meta(err)["op"])
	require.Equal(t, "AUTOMATION_INTERNAL", errs.Item(err))
}

func TestWithDetailsPlainError(t *testing.T) {
	t.Parallel()

	err := errs.WithDetails(errors.New("plain"), errs.Reason{Message: "ctx"})

	require.Equal(t, errs.Unknown, errs.Code(err))
	det, ok := errs.DetailsAs[errs.Reason](err)
	require.True(t, ok)
	require.Equal(t, "ctx", det.Message)
}

func TestBoundaryPreservesErrorIdentity(t *testing.T) {
	t.Parallel()

	sentinel := errs.Sentinel(errs.NotFound, "AUTOMATION_NOT_FOUND")

	out := errs.Boundary(
		errs.Annotate(errs.Specify(sentinel, "AUTOMATION_NOT_FOUND"), "automation.Get"),
		"AUTOMATION_INTERNAL",
	)

	e := &errs.Error{}
	require.ErrorAs(t, out, &e)
	require.Equal(t, errs.NotFound, e.Code)
	require.Equal(t, "not_found", e.CodeName)
	require.Equal(t, "AUTOMATION_NOT_FOUND", e.Item)
	require.ErrorIs(t, out, sentinel)
}

func TestBoundaryFillsCodeName(t *testing.T) {
	t.Parallel()

	sentinel := errs.Sentinel(errs.NotFound, "AUTOMATION_NOT_FOUND")

	out := errs.Boundary(sentinel, "")
	out = errs.Boundary(out, "")

	e := &errs.Error{}
	require.ErrorAs(t, out, &e)
	require.Equal(t, "not_found", e.CodeName)
}

func TestBoundaryEscapesPlainError(t *testing.T) {
	t.Parallel()

	plain := fmt.Errorf("repo blew up: %w", errors.New("conn reset"))

	out := errs.Boundary(plain, "AUTOMATION_INTERNAL")

	e := &errs.Error{}
	require.ErrorAs(t, out, &e)
	require.Equal(t, errs.Internal, e.Code)
	require.Equal(t, "AUTOMATION_INTERNAL", e.Item)
	require.True(t, errs.Meta(out)["escaped"].(bool))
	require.ErrorContains(t, out, "conn reset")
}

func TestBoundaryEscapesStructuredShellsOfPlainErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		wrap        func(error) error
		wantDetails bool
	}{
		{
			name: "annotated",
			wrap: func(err error) error {
				return errs.Annotate(err, "item.Create")
			},
		},
		{
			name: "with details",
			wrap: func(err error) error {
				return errs.WithDetails(err, errs.Reason{Message: "database unavailable"})
			},
			wantDetails: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			out := errs.Boundary(tc.wrap(errors.New("db down")), "SAFE_INTERNAL")

			e := &errs.Error{}
			require.ErrorAs(t, out, &e)
			require.Equal(t, errs.Internal, e.Code)
			require.Equal(t, "SAFE_INTERNAL", e.Item)
			require.Equal(t, true, errs.Meta(out)["escaped"])
			require.ErrorContains(t, out, "db down")

			if tc.wantDetails {
				det, ok := errs.DetailsAs[errs.Reason](out)
				require.True(t, ok)
				require.Equal(t, "database unavailable", det.Message)
			}
		})
	}
}

func TestBoundaryPreservesExplicitUnknownItem(t *testing.T) {
	t.Parallel()

	out := errs.Boundary(errs.New(errs.Unknown, "EXPLICIT_UNKNOWN"), "SAFE_INTERNAL")

	require.Equal(t, errs.Unknown, errs.Code(out))
	require.Equal(t, "EXPLICIT_UNKNOWN", errs.Item(out))
}

func TestBoundaryEscapesWithDefaultFallback(t *testing.T) {
	t.Parallel()

	out := errs.Boundary(errors.New("plain"), "")

	require.Equal(t, errs.DefaultFallbackItem, errs.Item(out))
}

func TestBoundaryNil(t *testing.T) {
	t.Parallel()

	require.NoError(t, errs.Boundary(nil, "X"))
}

func TestIsCode(t *testing.T) {
	t.Parallel()

	require.True(t, errs.IsCode(errs.Sentinel(errs.InvalidArgument, "X"), errs.InvalidArgument))
	require.False(t, errs.IsCode(errs.Sentinel(errs.InvalidArgument, "X"), errs.NotFound))
	require.True(t, errs.IsCode(errors.New("plain"), errs.Unknown))
}

func TestItemNonError(t *testing.T) {
	t.Parallel()

	require.Equal(t, "", errs.Item(errors.New("plain")))
	require.Equal(t, "", errs.Item(nil))
}

func TestWithDetailsNil(t *testing.T) {
	t.Parallel()

	require.NoError(t, errs.WithDetails(nil, errs.Reason{Message: "unused"}))
}

type sessionLimitDetails struct {
	Current int `json:"current"`
	Limit   int `json:"limit"`
}

func (sessionLimitDetails) ErrDetails() {}

func TestDetailsAsCustomType(t *testing.T) {
	t.Parallel()

	err := errs.WithDetails(
		errs.Sentinel(errs.ResourceExhausted, "SESSION_LIMIT_EXCEEDED"),
		sessionLimitDetails{Current: 12, Limit: 10},
	)

	det, ok := errs.DetailsAs[sessionLimitDetails](err)
	require.True(t, ok)
	require.Equal(t, sessionLimitDetails{Current: 12, Limit: 10}, det)

	_, ok = errs.DetailsAs[errs.Reason](err)
	require.False(t, ok)
}

func TestDetailsAsReturnsFalseWithoutStructuredErrorDetails(t *testing.T) {
	t.Parallel()

	_, ok := errs.DetailsAs[errs.Reason](errs.New(errs.NotFound, "NOT_FOUND"))
	require.False(t, ok)

	_, ok = errs.DetailsAs[errs.Reason](errors.New("plain"))
	require.False(t, ok)
}

func TestConventionsPreserveErrorWireType(t *testing.T) {
	t.Parallel()

	data := errmarshalling.Marshal(errs.New(errs.NotFound, "SESSION_NOT_FOUND"))

	var payload struct {
		Type string `json:"@type"`
	}
	require.NoError(t, json.Unmarshal(data, &payload))
	require.Equal(t, "github.com/clubpay/ronykit/rony/errs.Error", payload.Type)

	restored, err := errmarshalling.Unmarshal(data)
	require.NoError(t, err)
	require.Equal(t, errs.NotFound, errs.Code(restored))
	require.Equal(t, "SESSION_NOT_FOUND", errs.Item(restored))
}
