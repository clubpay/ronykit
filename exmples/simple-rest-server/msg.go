package main

import "github.com/goccy/go-json"

type echoRequest struct {
	RandomID int64 `json:"randomId" paramName:"randomID"`
}

func (e *echoRequest) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

type echoResponse struct {
	RandomID int64 `json:"randomId"`
}

func (e *echoResponse) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

type sumRequest struct {
	Val1 int64 `paramName:"val1"`
	Val2 int64 `paramName:"val2"`
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
