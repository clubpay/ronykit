package errs

import (
	"errors"
	"fmt"

	"github.com/clubpay/ronykit/x/rkit"
)

// DefaultFallbackItem is the item Boundary uses for errors without a
// non-empty *Error item. Callers should pass their own namespaced fallback
// (e.g. "AUTOMATION_INTERNAL") instead.
const DefaultFallbackItem = "INTERNAL"

// Sentinel declares a shared domain sentinel: an *Error with a stable
// item that callers match with errors.Is. Sentinels stay on the cause chain
// of every error derived from them.
func Sentinel(code ErrCode, item string) error {
	return B().Code(code).Msg(item).Err()
}

// New builds an *Error with the given code and stable item.
func New(code ErrCode, item string) error {
	return B().Code(code).Msg(item).Err()
}

// Newf is New with a formatted item. Prefer stable tokens; use it for
// tokens built from a closed set of values, not free-form sentences.
func Newf(code ErrCode, format string, args ...any) error {
	return B().Code(code).Msgf(format, args...).Err()
}

// Specify returns an error that keeps sentinel on the cause chain (so
// errors.Is(err, sentinel) keeps working) but carries a more specific stable
// item — and the sentinel's code — for clients.
func Specify(sentinel error, item string) error {
	if sentinel == nil {
		return New(Unknown, item)
	}

	return B().
		Code(Code(sentinel)).
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

	e := &Error{}
	if !errors.As(err, &e) {
		return B().Details(det).Cause(err).Err()
	}

	b := B().
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
// details are preserved; the marker rides in Meta, which is not exposed to
// external clients. Use it at app-layer boundaries instead of rebuilding the
// error with a generic message.
func Annotate(err error, op string) error {
	if err == nil {
		return nil
	}

	return B().Meta("op", op).Cause(err).Err()
}

// Boundary normalizes err at an API boundary into a wire-safe *Error.
//
//   - If the chain already carries an *Error with a non-empty item, its
//     identity (code, codeName, item, details) is preserved.
//   - Anything else becomes Internal with fallbackItem (or
//     DefaultFallbackItem when empty); the original error stays on the
//     cause chain for logs, and Meta records where it escaped.
func Boundary(err error, fallbackItem string) error {
	if err == nil {
		return nil
	}

	e := &Error{}
	if errors.As(err, &e) && e.Item != "" {
		if e.CodeName == "" {
			e.CodeName = e.Code.String()
		}

		return e
	}

	fallbackItem = rkit.Coalesce(fallbackItem, DefaultFallbackItem)

	return B().
		Code(Internal).
		Msg(fallbackItem).
		Meta("escaped", true).
		Cause(err).
		Err()
}
