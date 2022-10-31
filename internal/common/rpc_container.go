package common

import (
	"sync"

	"github.com/clubpay/ronykit"
	"github.com/goccy/go-json"
)

var inPool sync.Pool

// simpleIncomingJSONRPC implements ronykit.IncomingRPCContainer
type simpleIncomingJSONRPC struct {
	ID      string            `json:"id"`
	Header  map[string]string `json:"hdr"`
	Payload json.RawMessage   `json:"payload"`
}

func SimpleIncomingJSONRPC() ronykit.IncomingRPCContainer {
	v, ok := inPool.Get().(*simpleIncomingJSONRPC)
	if !ok {
		v = &simpleIncomingJSONRPC{
			Header: make(map[string]string, 4),
		}
	}

	return v
}

func (e *simpleIncomingJSONRPC) GetID() string {
	return e.ID
}

func (e *simpleIncomingJSONRPC) Unmarshal(data []byte) error {
	return json.UnmarshalNoEscape(data, e)
}

func (e *simpleIncomingJSONRPC) ExtractMessage(m ronykit.Message) error {
	return json.UnmarshalNoEscape(e.Payload, m)
}

func (e *simpleIncomingJSONRPC) GetHdr(key string) string {
	return e.Header[key]
}

func (e *simpleIncomingJSONRPC) GetHdrMap() map[string]string {
	return e.Header
}

func (e *simpleIncomingJSONRPC) Release() {
	for k := range e.Header {
		delete(e.Header, k)
	}
	e.Payload = e.Payload[:0]
	e.ID = e.ID[:0]

	inPool.Put(e)
}

var outPool = sync.Pool{}

// simpleOutgoingJSONRPC implements ronykit.OutgoingRPCContainer
type simpleOutgoingJSONRPC struct {
	ID      string            `json:"id"`
	Header  map[string]string `json:"hdr"`
	Payload ronykit.Message   `json:"payload"`
}

func SimpleOutgoingJSONRPC() ronykit.OutgoingRPCContainer {
	v, ok := outPool.Get().(*simpleOutgoingJSONRPC)
	if !ok {
		v = &simpleOutgoingJSONRPC{
			Header: make(map[string]string, 4),
		}
	}

	return v
}

func (e *simpleOutgoingJSONRPC) SetID(id string) {
	e.ID = id
}

func (e *simpleOutgoingJSONRPC) SetHdr(k, v string) {
	e.Header[k] = v
}

func (e *simpleOutgoingJSONRPC) InjectMessage(m ronykit.Message) {
	e.Payload = m
}

func (e *simpleOutgoingJSONRPC) Marshal() ([]byte, error) {
	return json.MarshalNoEscape(e)
}

func (e *simpleOutgoingJSONRPC) Release() {
	for k := range e.Header {
		delete(e.Header, k)
	}
	e.Payload = nil
	e.ID = e.ID[:0]

	outPool.Put(e)
}
