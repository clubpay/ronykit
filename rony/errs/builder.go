package errs

import (
	"errors"
	"fmt"
)

// A Builder allows for gradual construction of an error.
// The zero value is ready for use.
//
// Use Err() to construct the error.
type Builder struct {
	code    ErrCode
	codeSet bool
	det     ErrDetails
	detSet  bool

	msg    string
	msgSet bool
	meta   []any
	err    error

	httpStatus    int
	httpStatusSet bool
}

// B is shorthand for creating a new Builder.
func B() *Builder { return &Builder{} }

// Code sets the error code.
func (b *Builder) Code(c ErrCode) *Builder {
	b.code = c
	b.codeSet = true

	return b
}

// Msg sets the error message if it is not already set
func (b *Builder) Msg(msg string) *Builder {
	if b.msgSet {
		return b
	}

	return b.MsgX(msg)
}

// MsgX sets the error msg even if it already set
func (b *Builder) MsgX(msg string) *Builder {
	b.msg = msg
	b.msgSet = true

	return b
}

// Msgf is like Msg but uses fmt.Sprintf to construct the message.
func (b *Builder) Msgf(format string, args ...any) *Builder {
	if b.msgSet {
		return b
	}

	return b.MsgfX(format, args...)
}

func (b *Builder) MsgfX(format string, args ...any) *Builder {
	b.msg = fmt.Sprintf(format, args...)
	b.msgSet = true

	return b
}

// Meta appends metadata key-value pairs.
func (b *Builder) Meta(metaPairs ...any) *Builder {
	b.meta = append(b.meta, metaPairs...)

	return b
}

// Details sets the details.
func (b *Builder) Details(det ErrDetails) *Builder {
	if b.detSet {
		return b
	}

	return b.DetailsX(det)
}

func (b *Builder) DetailsX(det ErrDetails) *Builder {
	b.det = det
	b.detSet = true

	return b
}

// HTTPStatus overrides the wire-level HTTP status code for this error,
// bypassing the default status derived from Code. Use this for
// compatibility with an external HTTP contract (e.g. matching a legacy
// service's exact status for a given error) when the fixed gRPC-style
// code set has no equivalent.
func (b *Builder) HTTPStatus(status int) *Builder {
	b.httpStatus = status
	b.httpStatusSet = true

	return b
}

// Cause sets the underlying error cause.
func (b *Builder) Cause(err error) *Builder {
	b.err = err
	err = Convert(err)

	e := &Error{}
	if errors.As(err, &e) {
		if !b.codeSet {
			b.code = e.Code
			b.codeSet = true
		}

		if !b.msgSet && e.Item != "" {
			b.msg = e.Item
			b.msgSet = true
		}

		if !b.detSet {
			b.det = e.Details
			b.detSet = true
		}

		if !b.httpStatusSet && e.httpStatus != 0 {
			b.httpStatus = e.httpStatus
			b.httpStatusSet = true
		}
	}

	return b
}

// Err returns the constructed error.
// It never returns nil.
//
// If Code has not been set or has been set to OK,
// the Code is set to Unknown.
//
// If Msg has not been set and Cause is nil,
// the Msg is set to "unknown error".
func (b *Builder) Err() error {
	code := b.code
	if code == OK {
		code = Unknown
	}

	msg := b.msg
	if msg == "" && b.err == nil {
		msg = "unknown error"
	}

	var errMeta Metadata

	e := &Error{
		Code:       code,
		Item:       msg,
		Meta:       mergeMeta(errMeta, b.meta),
		Details:    b.det,
		underlying: b.err,
	}
	if b.httpStatusSet {
		e.httpStatus = b.httpStatus
	}

	return e
}
