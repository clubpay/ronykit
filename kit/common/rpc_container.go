package common

import (
	"sync"

	"github.com/clubpay/ronykit/kit"
	"github.com/clubpay/ronykit/kit/internal/json"
)

var inPool sync.Pool

// simpleIncomingJSONRPC implements kit.IncomingRPCContainer
type simpleIncomingJSONRPC struct {
	ID      string            `json:"id"`
	Header  map[string]string `json:"hdr"`
	Payload json.RawMessage   `json:"payload"`
}

func SimpleIncomingJSONRPC() kit.IncomingRPCContainer {
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
	return json.Unmarshal(data, e)
}

func (e *simpleIncomingJSONRPC) ExtractMessage(m kit.Message) error {
	switch m := m.(type) {
	case *kit.RawMessage:
		*m = kit.RawMessage(e.Payload)
	default:
		return json.Unmarshal(e.Payload, m)
	}

	return nil
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

// simpleOutgoingJSONRPC implements kit.OutgoingRPCContainer
type simpleOutgoingJSONRPC struct {
	ID      string            `json:"id"`
	Header  map[string]string `json:"hdr"`
	Payload kit.Message       `json:"payload"`
}

func SimpleOutgoingJSONRPC() kit.OutgoingRPCContainer {
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

func (e *simpleOutgoingJSONRPC) InjectMessage(m kit.Message) {
	e.Payload = m
}

func (e *simpleOutgoingJSONRPC) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e *simpleOutgoingJSONRPC) Release() {
	for k := range e.Header {
		delete(e.Header, k)
	}
	e.Payload = nil
	e.ID = e.ID[:0]

	outPool.Put(e)
}
