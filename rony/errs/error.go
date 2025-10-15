// Package errs provides structured error handling for Encore applications.
//
// See https://encore.dev/docs/develop/errors for more information about how errors work within Encore applications.
package errs

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"

	"github.com/clubpay/ronykit/kit/utils"
	"github.com/clubpay/ronykit/stub"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.Config{
	EscapeHTML:             false,
	SortMapKeys:            false,
	ValidateJsonRawMessage: true,
}.Froze()

// An Error is an error that provides structured information
// about the error. It includes an error code, a message,
// optionally additional structured details about the error,
// and arbitrary key-value metadata.
//
// The Details field is returned to external clients.
// The Meta field is only exposed to internal calls within Encore.
//
// Internally it captures an underlying error for printing
// and for use with errors.Is/As and call stack information.
//
// To provide accurate stack information, users are expected
// to convert non-Error errors into *Error as close to the
// root cause as possible. This is made simple with Wrap.
type Error struct {
	// Code is the error code to return.
	Code     ErrCode `json:"code"`
	CodeName string  `json:"codeName"`
	// Item is a descriptive message of the error.
	Item string `json:"item"`
	// Details are user-defined additional details.
	Details ErrDetails `json:"details"`
	// Meta are arbitrary key-value pairs for use within
	// the Encore application. They are not exposed to external clients.
	Meta Metadata `json:"-"`

	// underlying is the underlying error,
	// for use with errors.Is and errors.As.
	// It is not propagated across RPC boundaries.
	underlying error
}

// Metadata represents structured key-value pairs for attaching arbitrary
// metadata to errors. The metadata is propagated between internal services
// but is not exposed to external clients.
type Metadata map[string]any

// Wrap wraps the err, adding additional error information.
// If err is nil, it returns.
//
// If err is already an *Error, its code, message, and details
// are copied over to the new error.
func Wrap(err error, msg string, metaPairs ...any) error {
	if err == nil {
		return nil
	}

	e := &Error{Code: Unknown, Item: msg, underlying: err}

	var ee *Error
	if errors.As(err, &ee) {
		e.Details = ee.Details
		e.Code = ee.Code
		e.Meta = mergeMeta(ee.Meta, metaPairs)
	} else {
		e.Meta = mergeMeta(nil, metaPairs)
	}

	return e
}

// WrapCode is like Wrap but also sets the error code.
// If code is OK it reports nil.
func WrapCode(err error, code ErrCode, msg string, metaPairs ...any) error {
	if err == nil || code == OK {
		return nil
	}

	e := &Error{Code: code, Item: msg, underlying: err}

	ee := &Error{}
	if errors.As(err, &ee) {
		e.Details = ee.Details
		e.Meta = mergeMeta(ee.Meta, metaPairs)
	} else {
		e.Meta = mergeMeta(nil, metaPairs)
	}

	return e
}

func GenWrap(code ErrCode, msg string, metaPairs ...any) func(error, ...any) error {
	return func(err error, a ...any) error {
		return WrapCode(err, code, msg, append(metaPairs, a...)...)
	}
}

// Convert converts an error to an *Error.
// If the error is already an *Error, it returns it unmodified.
// If 'err' is nil, it returns.
func Convert(err error) error {
	if err == nil {
		return nil
	}

	var e *Error
	if errors.As(err, &e) {
		return e
	}

	var se *stub.Error
	if errors.As(err, &se) {
		out := &Error{
			Code:       ErrCode(se.Code()),
			Item:       se.Item(),
			underlying: se,
		}

		var errMap map[string]any

		_ = json.Unmarshal(utils.S2B(se.Item()), &errMap) //nolint:errcheck
		if errMap != nil {
			if item := utils.TryCast[string](errMap["item"]); len(item) > 0 {
				out.Item = item
			}
		}

		return out
	}

	return &Error{
		Code:       Unknown,
		underlying: err,
	}
}

// Code reports the error code from an error.
// If 'err' is nil, it reports OK.
// Otherwise, if err is not an *Error, it reports Unknown.
func Code(err error) ErrCode {
	if err == nil {
		return OK
	}

	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}

	return Unknown
}

// Meta reports the metadata included in the error.
// If 'err' is nil or the error lacks metadata it reports nil.
func Meta(err error) Metadata {
	var e *Error
	if errors.As(err, &e) {
		return e.Meta
	}

	return nil
}

// Details reports the error details included in the error.
// If err is nil or the error lacks details it reports nil.
func Details(err error) ErrDetails {
	var e *Error
	if errors.As(err, &e) {
		return e.Details
	}

	return nil
}

func (e Error) GetCode() int {
	return codeStatus[e.Code]
}

func (e Error) GetItem() string {
	return e.Item
}

// Error reports the error code and message.
func (e Error) Error() string {
	if e.Code == Unknown {
		return "unknown code: " + e.ErrorMessage()
	}

	return e.Code.String() + ": " + e.ErrorMessage()
}

// ErrorMessage reports the error message, joining this
// error's message with the messages from any underlying errors.
func (e *Error) ErrorMessage() string {
	if e.underlying == nil {
		return e.Item
	}

	var b strings.Builder
	b.WriteString(e.Item)

	next := e.underlying
	for next != nil {
		var msg string

		e := &Error{}
		if errors.As(next, &e) {
			msg = e.Item
			next = e.underlying
		} else {
			msg = next.Error()
			next = nil
		}

		if b.Len() > 0 && msg != "" {
			b.WriteString(": ")
		}

		b.WriteString(msg)
	}

	return b.String()
}

// Unwrap returns the underlying error, if any.
func (e *Error) Unwrap() error {
	return e.underlying
}

func mergeMeta(md Metadata, pairs []any) Metadata {
	n := len(pairs)
	if n%2 != 0 {
		panic(fmt.Sprintf("got uneven number (%d) of metadata key-values", n))
	}

	if md == nil && n > 0 {
		md = make(Metadata, n/2)
	}

	for i := 0; i < n; i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			panic(fmt.Sprintf("metadata key-value pair #%d key is not a string (is %T)", i/2, pairs[i]))
		}

		md[key] = pairs[i+1]
	}

	return md
}

func init() {
	jsoniter.RegisterTypeEncoderFunc("errs.Error", func(ptr unsafe.Pointer, stream *jsoniter.Stream) {
		e := (*Error)(ptr)

		stream.WriteObjectStart()
		stream.WriteObjectField("code")
		stream.WriteString(e.Code.String())
		stream.WriteMore()
		stream.WriteObjectField("message")
		stream.WriteString(e.ErrorMessage())
		stream.WriteMore()
		stream.WriteObjectField("details")
		stream.WriteVal(e.Details)
		stream.WriteObjectEnd()
	}, nil)
}

type Text string

func (t Text) Error() string {
	return string(t)
}
