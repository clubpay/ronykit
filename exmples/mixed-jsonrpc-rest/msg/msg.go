package msg

import "github.com/goccy/go-json"

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

type EchoRequest struct {
	RandomID int64 `json:"randomId" paramName:"randomID"`
}

func (e *EchoRequest) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

type EchoResponse struct {
	RandomID int64 `json:"randomId"`
}

func (e *EchoResponse) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

type SumRequest struct {
	Val1 int64
	Val2 int64
}

func (s *SumRequest) Marshal() ([]byte, error) {
	return json.Marshal(s)
}

type SumResponse struct {
	Val int64
}

func (s *SumResponse) Marshal() ([]byte, error) {
	return json.Marshal(s)
}
