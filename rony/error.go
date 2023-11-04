package rony

import "github.com/clubpay/ronykit/kit"

type Error struct {
	code int
	item string
	err  error
}

var _ kit.ErrorMessage = (*Error)(nil)

func NewError(err error) Error {
	return Error{
		err: err,
	}
}

func (e Error) SetCode(code int) Error {
	e.code = code

	return e
}

func (e Error) GetCode() int {
	return e.code
}

func (e Error) SetItem(item string) Error {
	e.item = item

	return e
}

func (e Error) GetItem() string {
	return e.item
}

func (e Error) Error() string {
	return e.err.Error()
}
