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
}

// B is shorthand for creating a new Builder.
func B() *Builder { return &Builder{} }

// Code sets the error code.
func (b *Builder) Code(c ErrCode) *Builder {
	b.code = c
	b.codeSet = true

	return b
}

// Msg sets the error message.
func (b *Builder) Msg(msg string) *Builder {
	b.msg = msg
	b.msgSet = true

	return b
}

// Msgf is like Msg but uses fmt.Sprintf to construct the message.
func (b *Builder) Msgf(format string, args ...any) *Builder {
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
	b.det = det
	b.detSet = true

	return b
}

// Cause sets the underlying error cause.
func (b *Builder) Cause(err error) *Builder {
	b.err = err
	e := &Error{}
	if errors.As(err, &e) {
		if !b.codeSet {
			b.code = e.Code
		}
		if !b.msgSet {
			b.msg = e.Item
		}
		if !b.detSet {
			b.det = e.Details
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

	return &Error{
		Code:       code,
		Item:       msg,
		Meta:       mergeMeta(errMeta, b.meta),
		Details:    b.det,
		underlying: b.err,
	}
}
