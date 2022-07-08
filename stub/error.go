package stub

import "fmt"

type Error struct {
	code int
	item string
	err  error
}

func NewError(code int, item string) *Error {
	return &Error{
		code: code,
		item: item,
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
