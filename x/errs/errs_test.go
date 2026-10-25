package errs_test

import (
	"errors"
	"fmt"
	"testing"

	core "github.com/clubpay/ronykit/rony/errs"
	"github.com/clubpay/ronykit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestSentinel(t *testing.T) {
	t.Parallel()

	s := errs.Sentinel(errs.NotFound, "AUTOMATION_NOT_FOUND")

	require.ErrorIs(t, s, s)
	require.Equal(t, errs.NotFound, core.Code(s))
	require.Equal(t, "AUTOMATION_NOT_FOUND", errs.Item(s))
}

func TestSpecifyKeepsSentinelAndSpecificItem(t *testing.T) {
	t.Parallel()

	sentinel := errs.Sentinel(errs.InvalidArgument, "INVALID_ARGUMENT")

	err := errs.Specify(sentinel, "TRIGGER_GITHUB_ORG_REQUIRED")

	require.ErrorIs(t, err, sentinel)
	require.Equal(t, errs.InvalidArgument, core.Code(err))
	require.Equal(t, "TRIGGER_GITHUB_ORG_REQUIRED", errs.Item(err))
}

func TestSpecifyNilSentinel(t *testing.T) {
	t.Parallel()

	err := errs.Specify(nil, "SOMETHING")

	require.Equal(t, errs.Unknown, core.Code(err))
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
	require.Equal(t, errs.AlreadyExists, core.Code(err))
	require.Equal(t, "AUTOMATION_NAME_CONFLICT", errs.Item(err))
	require.Equal(t, "automation.Create", core.Meta(err)["op"])
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
	require.Equal(t, errs.FailedPrecondition, core.Code(err))
	require.Equal(t, "FORMULA_NOT_IN_PACK", errs.Item(err))

	det, ok := errs.DetailsAs[errs.Reason](err)
	require.True(t, ok)
	require.Equal(t, "formula \"x\" not attached", det.Message)
}

func TestWithDetailsPreservesMetaFromAnnotate(t *testing.T) {
	t.Parallel()

	err := errs.Annotate(errs.Sentinel(errs.Internal, "AUTOMATION_INTERNAL"), "automation.Create")
	err = errs.WithDetails(err, errs.Reason{Message: "boom"})

	require.Equal(t, "automation.Create", core.Meta(err)["op"])
	require.Equal(t, "AUTOMATION_INTERNAL", errs.Item(err))
}

func TestWithDetailsPlainError(t *testing.T) {
	t.Parallel()

	err := errs.WithDetails(errors.New("plain"), errs.Reason{Message: "ctx"})

	require.Equal(t, errs.Unknown, core.Code(err))
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

	e := &core.Error{}
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

	e := &core.Error{}
	require.ErrorAs(t, out, &e)
	require.Equal(t, "not_found", e.CodeName)
}

func TestBoundaryEscapesPlainError(t *testing.T) {
	t.Parallel()

	plain := fmt.Errorf("repo blew up: %w", errors.New("conn reset"))

	out := errs.Boundary(plain, "AUTOMATION_INTERNAL")

	e := &core.Error{}
	require.ErrorAs(t, out, &e)
	require.Equal(t, errs.Internal, e.Code)
	require.Equal(t, "AUTOMATION_INTERNAL", e.Item)
	require.True(t, core.Meta(out)["escaped"].(bool))
	require.ErrorContains(t, out, "conn reset")
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
	// Plain errors convert to Unknown at the edge.
	require.True(t, errs.IsCode(errors.New("plain"), errs.Unknown))
}

func TestItemNonError(t *testing.T) {
	t.Parallel()

	require.Equal(t, "", errs.Item(errors.New("plain")))
	require.Equal(t, "", errs.Item(nil))
}
