package common

import (
	"sync"

	"github.com/clubpay/ronykit/kit"
	"github.com/goccy/go-json"
)

var inSimplePool sync.Pool

// simpleIncomingJSONRPC implements kit.IncomingRPCContainer
type simpleIncomingJSONRPC struct {
	ID      string            `json:"id"`
	Header  map[string]string `json:"hdr"`
	Payload json.RawMessage   `json:"payload"`
}

func SimpleIncomingJSONRPC() kit.IncomingRPCContainer {
	v, ok := inSimplePool.Get().(*simpleIncomingJSONRPC)
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
	return json.Unmarshal(e.Payload, m)
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

	inSimplePool.Put(e)
}

var outSimplePool = sync.Pool{}

// simpleOutgoingJSONRPC implements kit.OutgoingRPCContainer
type simpleOutgoingJSONRPC struct {
	ID      string            `json:"id"`
	Header  map[string]string `json:"hdr"`
	Payload kit.Message       `json:"payload"`
}

func SimpleOutgoingJSONRPC() kit.OutgoingRPCContainer {
	v, ok := outSimplePool.Get().(*simpleOutgoingJSONRPC)
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

	outSimplePool.Put(e)
}
