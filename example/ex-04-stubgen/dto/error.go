package dto

import "fmt"

type ErrorMessage struct {
	Code int    `json:"code"`
	Item string `json:"item"`
}

func (e ErrorMessage) GetCode() int {
	return e.Code
}

func (e ErrorMessage) GetItem() string {
	return e.Item
}

func (e ErrorMessage) Error() string {
	return fmt.Sprintf("%d:%s", e.Code, e.Item)
}

func Err(code int, item string) ErrorMessage {
	return ErrorMessage{
		Code: code,
		Item: item,
	}
}
