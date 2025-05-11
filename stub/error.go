package stub

import (
	"fmt"

	"github.com/clubpay/ronykit/kit"
)

type Error struct {
	code int
	item string
	msg  kit.Message
	err  error
}

func NewError(code int, item string) *Error {
	return &Error{
		code: code,
		item: item,
	}
}

func NewErrorWithMsg(msg kit.Message) *Error {
	wErr := &Error{
		msg: msg,
	}
	if e, ok := msg.(interface{ GetCode() int }); ok {
		wErr.code = e.GetCode()
	}
	if e, ok := msg.(interface{ GetItem() string }); ok {
		wErr.item = e.GetItem()
	}

	return wErr
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

func (err *Error) SetMsg(msg kit.Message) *Error {
	err.msg = msg

	return err
}

func (err Error) Msg() kit.Message {
	return err.msg
}

func (err *Error) SetCode(code int) *Error {
	err.code = code

	return err
}

func (err Error) Code() int {
	return err.code
}

func (err Error) GetCode() int {
	return err.code
}

func (err *Error) SetItem(item string) *Error {
	err.item = item

	return err
}

func (err Error) Item() string {
	return err.item
}

func (err Error) GetItem() string {
	return err.item
}

func (err Error) Is(target error) bool {
	var cond bool
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
