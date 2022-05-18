package dto

import (
	"fmt"

	"github.com/clubpay/ronykit"
	"github.com/goccy/go-json"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"msg"`
}

var _ ronykit.ErrorMessage = (*Error)(nil)

func (e *Error) Error() string {
	return fmt.Sprintf("%s:%s", e.Code, e.Message)
}

func (e *Error) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func Err(code, msg string) *Error {
	return &Error{
		Code:    code,
		Message: msg,
	}
}
