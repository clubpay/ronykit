package main

import "github.com/goccy/go-json"

type errorMessage struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *errorMessage) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *errorMessage) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

type echoRequest struct {
	RandomID int64 `json:"randomID"`
}

func (e *echoRequest) Marshal() ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

type echoResponse struct {
	RandomID int64 `json:"randomID"`
}

func (e *echoResponse) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

type sumRequest struct {
	Val1 int64
	Val2 int64
}

func (s *sumRequest) Marshal() ([]byte, error) {
	return json.Marshal(s)
}

type sumResponse struct {
	Val int64
}

func (s *sumResponse) Marshal() ([]byte, error) {
	return json.Marshal(s)
}
