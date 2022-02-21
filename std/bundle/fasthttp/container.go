package fasthttp

import (
	"github.com/clubpay/ronykit"
	"github.com/goccy/go-json"
)

type IncomingRPCFactory func() IncomingRPC

type IncomingRPC interface {
	Unmarshal(data []byte) error
	Fill(m ronykit.Message) error
	GetHdr(key string) string
	GetHdrMap() map[string]string
}

type OutgoingRPCFactory func() OutgoingRPC

type OutgoingRPC interface {
	Marshal() ([]byte, error)
	SetHdr(k, v string)
	SetPayload(m ronykit.Message)
	Reset()
}

type jsonIncomingRPC struct {
	Header  map[string]string `json:"hdr"`
	Payload json.RawMessage   `json:"payload"`
}

func (e *jsonIncomingRPC) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *jsonIncomingRPC) Fill(m ronykit.Message) (err error) {
	return json.Unmarshal(e.Payload, m)
}

func (e *jsonIncomingRPC) GetHdr(key string) string {
	return e.Header[key]
}

func (e *jsonIncomingRPC) GetHdrMap() map[string]string {
	return e.Header
}

type jsonOutgoingRPC struct {
	Header  map[string]string `json:"hdr"`
	Payload ronykit.Message   `json:"payload"`
}

func (e *jsonOutgoingRPC) SetHdr(k, v string) {
	e.Header[k] = v
}

func (e *jsonOutgoingRPC) SetPayload(m ronykit.Message) {
	e.Payload = m
}

func (e *jsonOutgoingRPC) Reset() {
	for k := range e.Header {
		delete(e.Header, k)
	}
	e.Payload = nil
}

func (e *jsonOutgoingRPC) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

//var outgoingMessagePool sync.Pool
//
//func acquireOutgoingMessage() OutgoingRPC {
//	e, ok := outgoingMessagePool.Get().(OutgoingRPC)
//	if !ok {
//		e = &jsonOutgoingRPC{
//			Header: map[string]string{},
//		}
//	}
//
//	return e
//}
//
//func releaseOutgoingMessage(e OutgoingRPC) {
//	for k := range e.Header {
//		delete(e.Header, k)
//	}
//
//	outgoingMessagePool.Put(e)
//}
