package stub

import (
	"fmt"

	"github.com/clubpay/ronykit"
)

type Error struct {
	code int
	item string
	msg  ronykit.Message
	err  error
}

func NewError(code int, item string) *Error {
	return &Error{
		code: code,
		item: item,
	}
}

func NewErrorWithMsg(code int, item string, msg ronykit.Message) *Error {
	return &Error{
		code: code,
		item: item,
		msg:  msg,
	}
}

func WrapError(err error) *Error {
	if err == nil {
		return nil
	}

	return &Error{err: err}
}

func (err Error) Error() string {
	if err.err != nil {
		return err.err.Error()
	}

	return fmt.Sprintf("ERR(%d): %s", err.code, err.item)
}

func (err Error) Msg() ronykit.Message {
	return err.msg
}

func (err Error) Code() int {
	return err.code
}

func (err Error) Item() string {
	return err.item
}

func (err Error) Is(target error) bool {
	var cond bool
	//nolint:errorlint
	switch e := target.(type) {
	case Error:
		cond = e.err == nil && e.code == err.code && e.item == err.item

	case *Error:
		cond = e.err == nil && e.code == err.code && e.item == err.item
	}

	return cond
}

func (err Error) Unwrap() error {
	return err.err
}
