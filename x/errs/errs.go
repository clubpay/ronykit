package errs

import (
	"errors"
	"fmt"

	core "github.com/clubpay/ronykit/rony/errs"
)

// ErrCode is re-exported so callers can stay in this package.
type ErrCode = core.ErrCode

// ErrDetails is re-exported so callers can stay in this package.
type ErrDetails = core.ErrDetails

// Re-exported gRPC-style codes for convenience.
const (
	OK                 = core.OK
	Canceled           = core.Canceled
	Unknown            = core.Unknown
	InvalidArgument    = core.InvalidArgument
	DeadlineExceeded   = core.DeadlineExceeded
	NotFound           = core.NotFound
	AlreadyExists      = core.AlreadyExists
	PermissionDenied   = core.PermissionDenied
	ResourceExhausted  = core.ResourceExhausted
	FailedPrecondition = core.FailedPrecondition
	Aborted            = core.Aborted
	OutOfRange         = core.OutOfRange
	Unimplemented      = core.Unimplemented
	Internal           = core.Internal
	Unavailable        = core.Unavailable
	DataLoss           = core.DataLoss
	Unauthenticated    = core.Unauthenticated
)

// DefaultFallbackItem is the item Boundary uses for errors that carry no
// *core.Error identity. Callers should pass their own namespaced fallback
// (e.g. "AUTOMATION_INTERNAL") instead.
const DefaultFallbackItem = "INTERNAL"

// Sentinel declares a shared domain sentinel: a *core.Error with a stable
// item that callers match with errors.Is. Sentinels stay on the cause chain
// of every error derived from them.
func Sentinel(code ErrCode, item string) error {
	return core.B().Code(code).Msg(item).Err()
}

// New builds a *core.Error with the given code and stable item.
func New(code ErrCode, item string) error {
	return core.B().Code(code).Msg(item).Err()
}

// Newf is New with a formatted item. Prefer stable tokens; use it for
// tokens built from a closed set of values, not free-form sentences.
func Newf(code ErrCode, format string, args ...any) error {
	return core.B().Code(code).Msgf(format, args...).Err()
}

// Specify returns an error that keeps sentinel on the cause chain (so
// errors.Is(err, sentinel) keeps working) but carries a more specific stable
// item — and the sentinel's code — for clients.
func Specify(sentinel error, item string) error {
	if sentinel == nil {
		return New(Unknown, item)
	}

	return core.B().
		Code(core.Code(sentinel)).
		Msg(item).
		Cause(sentinel).
		Err()
}

// Specifyf is Specify with a formatted item.
func Specifyf(sentinel error, format string, args ...any) error {
	return Specify(sentinel, fmt.Sprintf(format, args...))
}

// WithDetails attaches client-visible structured details to err while
// preserving its code and item.
func WithDetails(err error, det ErrDetails) error {
	if err == nil {
		return nil
	}

	e := &core.Error{}
	if !errors.As(err, &e) {
		return core.B().Details(det).Cause(err).Err()
	}

	b := core.B().
		Code(e.Code).
		MsgX(e.Item).
		Details(det)

	if len(e.Meta) > 0 {
		pairs := make([]any, 0, len(e.Meta)*2)
		for k, v := range e.Meta {
			pairs = append(pairs, k, v)
		}

		b.Meta(pairs...)
	}

	return b.Cause(err).Err()
}

// Annotate marks err with the internal operation that failed (e.g.
// "automation.Create"). It changes nothing on the wire: code, item and
// details are preserved; the marker rides in Meta, which never leaves the
// process. Use it at app-layer boundaries instead of rebuilding the error
// with a generic message.
func Annotate(err error, op string) error {
	if err == nil {
		return nil
	}

	return core.B().Meta("op", op).Cause(err).Err()
}

// Boundary normalizes err at an API boundary into a wire-safe *core.Error.
//
//   - If the chain already carries a *core.Error, its identity (code,
//     codeName, item, details) is preserved.
//   - Anything else becomes Internal with fallbackItem (or
//     DefaultFallbackItem when empty); the original error stays on the
//     cause chain for logs, and Meta records where it escaped.
func Boundary(err error, fallbackItem string) error {
	if err == nil {
		return nil
	}

	e := &core.Error{}
	if errors.As(err, &e) {
		if e.CodeName == "" {
			e.CodeName = e.Code.String()
		}

		return e
	}

	if fallbackItem == "" {
		fallbackItem = DefaultFallbackItem
	}

	return core.B().
		Code(Internal).
		Msg(fallbackItem).
		Meta("escaped", true).
		Cause(err).
		Err()
}
