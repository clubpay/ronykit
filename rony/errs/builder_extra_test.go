package errs_test

import (
	"errors"
	"testing"

	"github.com/clubpay/ronykit/rony/errs"
)

type detailInfo struct {
	Info string
}

func (detailInfo) ErrDetails() {}

func TestBuilderDefaultsAndMessages(t *testing.T) {
	err := errs.B().Err()
	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("expected errs.Error, got %T", err)
	}
	if e.Code != errs.Unknown || e.Item != "unknown error" {
		t.Fatalf("unexpected default error: %+v", e)
	}

	err = errs.B().Msg("first").Msg("second").Err()
	if !errors.As(err, &e) || e.Item != "first" {
		t.Fatalf("unexpected message selection: %+v", e)
	}

	err = errs.B().Msg("first").MsgX("override").Err()
	if !errors.As(err, &e) || e.Item != "override" {
		t.Fatalf("unexpected message override: %+v", e)
	}

	err = errs.B().Msgf("msg %d", 1).Err()
	if !errors.As(err, &e) || e.Item != "msg 1" {
		t.Fatalf("unexpected msgf: %+v", e)
	}

	err = errs.B().MsgfX("msg %d", 2).Err()
	if !errors.As(err, &e) || e.Item != "msg 2" {
		t.Fatalf("unexpected msgfx: %+v", e)
	}
}

func TestBuilderCauseAndDetails(t *testing.T) {
	cause := &errs.Error{
		Code:    errs.NotFound,
		Item:    "missing",
		Details: detailInfo{Info: "detail"},
	}

	err := errs.B().Cause(cause).Err()
	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("expected errs.Error, got %T", err)
	}
	if e.Code != errs.NotFound || e.Item != "missing" {
		t.Fatalf("unexpected cause propagation: %+v", e)
	}
	if _, ok := e.Details.(detailInfo); !ok {
		t.Fatalf("expected details to propagate, got %T", e.Details)
	}
}

func TestBuilderDetailsOverrides(t *testing.T) {
	d1 := detailInfo{Info: "first"}
	d2 := detailInfo{Info: "second"}

	err := errs.B().Details(d1).Details(d2).Err()
	if det, ok := errs.Details(err).(detailInfo); !ok || det.Info != "first" {
		t.Fatalf("unexpected details: %+v", errs.Details(err))
	}

	err = errs.B().Details(d1).DetailsX(d2).Err()
	if det, ok := errs.Details(err).(detailInfo); !ok || det.Info != "second" {
		t.Fatalf("unexpected details override: %+v", errs.Details(err))
	}
}

func TestBuilderMetaAndPanics(t *testing.T) {
	err := errs.B().Code(errs.InvalidArgument).Msg("bad").Meta("k", "v").Err()
	if meta := errs.Meta(err); meta["k"] != "v" {
		t.Fatalf("unexpected meta: %v", meta)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for odd meta pairs")
		}
	}()
	_ = errs.B().Meta("k").Err()
}

func TestBuilderMetaKeyPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for non-string meta key")
		}
	}()
	_ = errs.B().Meta(1, "v").Err()
}

func TestWrapsAndCodes(t *testing.T) {
	if errs.Wrap(nil, "msg") != nil {
		t.Fatal("expected nil wrap")
	}
	if errs.WrapCode(nil, errs.NotFound, "msg") != nil {
		t.Fatal("expected nil wrapcode with nil error")
	}
	if errs.WrapCode(errors.New("x"), errs.OK, "msg") != nil {
		t.Fatal("expected nil wrapcode with OK code")
	}

	base := errors.New("base")
	wrapped := errs.Wrap(base, "wrapped", "k", "v")
	if errs.Code(wrapped) != errs.Unknown {
		t.Fatalf("unexpected wrap code: %v", errs.Code(wrapped))
	}
	if meta := errs.Meta(wrapped); meta["k"] != "v" {
		t.Fatalf("unexpected wrap meta: %v", meta)
	}

	gen := errs.GenWrap(errs.NotFound, "missing", "a", "b")
	wrapped = gen(errors.New("boom"), "c", "d")
	if errs.Code(wrapped) != errs.NotFound {
		t.Fatalf("unexpected gen wrap code: %v", errs.Code(wrapped))
	}
	meta := errs.Meta(wrapped)
	if meta["a"] != "b" || meta["c"] != "d" {
		t.Fatalf("unexpected gen wrap meta: %v", meta)
	}

	converted := errs.Convert(errors.New("plain"))
	var convErr *errs.Error
	if !errors.As(converted, &convErr) || convErr.Code != errs.Unknown {
		t.Fatalf("unexpected convert error: %+v", converted)
	}
}

func TestErrorUnwrap(t *testing.T) {
	base := errors.New("base")
	err := errs.Wrap(base, "wrapped")
	if errors.Unwrap(err) != base {
		t.Fatalf("unexpected unwrap: %v", errors.Unwrap(err))
	}
	if errs.Details(nil) != nil {
		t.Fatal("expected nil details for nil error")
	}
}

func TestHTTPStatusAndText(t *testing.T) {
	if errs.HTTPStatusToCode(404) != errs.NotFound {
		t.Fatalf("unexpected status to code")
	}
	if errs.HTTPStatusToCode(999) != errs.Unknown {
		t.Fatalf("unexpected status to code for unknown")
	}

	err := errs.B().Code(errs.NotFound).Msg("missing").Err()
	if errs.HTTPStatus(err) != 404 {
		t.Fatalf("unexpected http status for not found: %d", errs.HTTPStatus(err))
	}
	if errs.HTTPStatus(errors.New("boom")) != 500 {
		t.Fatalf("unexpected http status for generic error: %d", errs.HTTPStatus(errors.New("boom")))
	}
	if errs.NotFound.String() != "not_found" {
		t.Fatalf("unexpected code string: %s", errs.NotFound.String())
	}
	if errs.NotFound.HTTPStatus() != 404 {
		t.Fatalf("unexpected code http status: %d", errs.NotFound.HTTPStatus())
	}

	if errs.Text("hi").Error() != "hi" {
		t.Fatalf("unexpected text error")
	}
}

func TestBuilderHTTPStatusOverride(t *testing.T) {
	err := errs.B().Code(errs.InvalidArgument).Msg("PHONE_IS_NOT_WHITELISTED").HTTPStatus(406).Err()

	var e *errs.Error
	if !errors.As(err, &e) {
		t.Fatalf("expected errs.Error, got %T", err)
	}
	if got := e.GetCode(); got != 406 {
		t.Fatalf("expected overridden GetCode 406, got %d", got)
	}
	if got := errs.HTTPStatus(err); got != 406 {
		t.Fatalf("expected overridden HTTPStatus 406, got %d", got)
	}
	// Code-derived classification (retryability, gRPC mapping, ...) is untouched.
	if e.Code != errs.InvalidArgument {
		t.Fatalf("expected Code to remain InvalidArgument, got %v", e.Code)
	}

	// No override set: falls back to the default Code-derived status.
	plain := errs.B().Code(errs.NotFound).Msg("missing").Err()
	if got := errs.HTTPStatus(plain); got != errs.NotFound.HTTPStatus() {
		t.Fatalf("expected default status %d, got %d", errs.NotFound.HTTPStatus(), got)
	}
}

func TestBuilderHTTPStatusPropagatesThroughCause(t *testing.T) {
	cause := errs.B().Code(errs.NotFound).Msg("OTP_EXPIRED").HTTPStatus(410).Err()

	wrapped := errs.B().Cause(cause).Err()
	if got := errs.HTTPStatus(wrapped); got != 410 {
		t.Fatalf("expected propagated override 410, got %d", got)
	}

	// An explicit override on the wrapping builder wins over the cause's.
	wrapped = errs.B().Cause(cause).HTTPStatus(404).Err()
	if got := errs.HTTPStatus(wrapped); got != 404 {
		t.Fatalf("expected explicit override 404 to win, got %d", got)
	}
}

func TestWrapPreservesHTTPStatusOverride(t *testing.T) {
	inner := errs.B().Code(errs.InvalidArgument).Msg("PHONE_IS_NOT_WHITELISTED").HTTPStatus(406).Err()

	wrapped := errs.Wrap(inner, "session")
	if got := errs.HTTPStatus(wrapped); got != 406 {
		t.Fatalf("Wrap should keep HTTP status override, got %d", got)
	}
	if errs.Code(wrapped) != errs.InvalidArgument {
		t.Fatalf("Wrap should keep code, got %v", errs.Code(wrapped))
	}

	// WrapCode replaces the code; the override is intentionally not copied
	// so the new code's default HTTP status applies.
	recoded := errs.WrapCode(inner, errs.Internal, "INTERNAL")
	if got := errs.HTTPStatus(recoded); got != errs.Internal.HTTPStatus() {
		t.Fatalf("WrapCode should use the new code's status, got %d", got)
	}
	if errs.Code(recoded) != errs.Internal {
		t.Fatalf("WrapCode should replace code, got %v", errs.Code(recoded))
	}

	plain := errs.Wrap(errors.New("x"), "wrapped")
	if got := errs.HTTPStatus(plain); got != errs.Unknown.HTTPStatus() {
		t.Fatalf("Wrap of non-Error should stay unknown/500, got %d", got)
	}
}

func TestErrCodeOutOfRangeIsSafe(t *testing.T) {
	cases := []errs.ErrCode{99, -1, 1000}
	for _, c := range cases {
		if got := c.String(); got != "unknown" {
			t.Fatalf("out-of-range String(%d) = %q, want unknown", c, got)
		}
		if got := c.HTTPStatus(); got != 500 {
			t.Fatalf("out-of-range HTTPStatus(%d) = %d, want 500", c, got)
		}

		e := &errs.Error{Code: c, Item: "x"}
		if got := e.GetCode(); got != 500 {
			t.Fatalf("GetCode for code %d = %d, want 500", c, got)
		}
		if got := errs.HTTPStatus(e); got != 500 {
			t.Fatalf("HTTPStatus for code %d = %d, want 500", c, got)
		}
	}

	// Valid codes are unchanged, including the zero value.
	if errs.OK.String() != "ok" || errs.OK.HTTPStatus() != 200 {
		t.Fatalf("OK mapping changed: %s %d", errs.OK.String(), errs.OK.HTTPStatus())
	}
	if errs.Unauthenticated.String() != "unauthenticated" || errs.Unauthenticated.HTTPStatus() != 401 {
		t.Fatalf("Unauthenticated mapping changed: %s %d", errs.Unauthenticated.String(), errs.Unauthenticated.HTTPStatus())
	}
}

func TestHTTPStatusCases(t *testing.T) {
	cases := []errs.ErrCode{
		errs.OK,
		errs.Canceled,
		errs.Unknown,
		errs.InvalidArgument,
		errs.DeadlineExceeded,
		errs.NotFound,
		errs.AlreadyExists,
		errs.PermissionDenied,
		errs.ResourceExhausted,
		errs.FailedPrecondition,
		errs.Aborted,
		errs.OutOfRange,
		errs.Unimplemented,
		errs.Internal,
		errs.Unavailable,
		errs.DataLoss,
		errs.Unauthenticated,
	}

	for _, c := range cases {
		err := &errs.Error{Code: c, Item: "x"}
		if errs.HTTPStatus(err) != c.HTTPStatus() {
			t.Fatalf("unexpected http status for %v: %d", c, errs.HTTPStatus(err))
		}
	}
}
