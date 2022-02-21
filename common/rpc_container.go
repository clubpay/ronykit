package common

import (
	"github.com/clubpay/ronykit"
	"github.com/goccy/go-json"
)

// SimpleIncomingJSONRPC implements ronykit.IncomingRPCContainer
type SimpleIncomingJSONRPC struct {
	Header  map[string]string `json:"hdr"`
	Payload json.RawMessage   `json:"payload"`
}

func (e *SimpleIncomingJSONRPC) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *SimpleIncomingJSONRPC) Fill(m ronykit.Message) (err error) {
	return json.Unmarshal(e.Payload, m)
}

func (e *SimpleIncomingJSONRPC) GetHdr(key string) string {
	return e.Header[key]
}

func (e *SimpleIncomingJSONRPC) GetHdrMap() map[string]string {
	return e.Header
}

// SimpleOutgoingJSONRPC implements ronykit.OutgoingRPCContainer
type SimpleOutgoingJSONRPC struct {
	Header  map[string]string `json:"hdr"`
	Payload ronykit.Message   `json:"payload"`
}

func (e *SimpleOutgoingJSONRPC) SetHdr(k, v string) {
	e.Header[k] = v
}

func (e *SimpleOutgoingJSONRPC) SetPayload(m ronykit.Message) {
	e.Payload = m
}

func (e *SimpleOutgoingJSONRPC) Reset() {
	for k := range e.Header {
		delete(e.Header, k)
	}
	e.Payload = nil
}

func (e *SimpleOutgoingJSONRPC) Marshal() ([]byte, error) {
	return json.Marshal(e)
}
