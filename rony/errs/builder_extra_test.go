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
