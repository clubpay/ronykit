package common

import (
	"github.com/clubpay/ronykit"
	"github.com/goccy/go-json"
)

// simpleIncomingJSONRPC implements ronykit.IncomingRPCContainer
type simpleIncomingJSONRPC struct {
	Header  map[string]string `json:"hdr"`
	Payload json.RawMessage   `json:"payload"`
}

var SimpleIncomingJSONRPC = func() ronykit.IncomingRPCContainer {
	return &simpleIncomingJSONRPC{
		Header: map[string]string{},
	}
}

func (e *simpleIncomingJSONRPC) Unmarshal(data []byte) error {
	return json.UnmarshalNoEscape(data, e)
}

func (e *simpleIncomingJSONRPC) Fill(m ronykit.Message) error {
	return json.UnmarshalNoEscape(e.Payload, m)
}

func (e *simpleIncomingJSONRPC) GetHdr(key string) string {
	return e.Header[key]
}

func (e *simpleIncomingJSONRPC) GetHdrMap() map[string]string {
	return e.Header
}

// simpleOutgoingJSONRPC implements ronykit.OutgoingRPCContainer
type simpleOutgoingJSONRPC struct {
	Header  map[string]string `json:"hdr"`
	Payload ronykit.Message   `json:"payload"`
}

var SimpleOutgoingJSONRPC = func() ronykit.OutgoingRPCContainer {
	return &simpleOutgoingJSONRPC{
		Header: make(map[string]string, 4),
	}
}

func (e *simpleOutgoingJSONRPC) SetHdr(k, v string) {
	e.Header[k] = v
}

func (e *simpleOutgoingJSONRPC) SetPayload(m ronykit.Message) {
	e.Payload = m
}

func (e *simpleOutgoingJSONRPC) Reset() {
	for k := range e.Header {
		delete(e.Header, k)
	}
	e.Payload = nil
}

func (e *simpleOutgoingJSONRPC) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
