package fasthttp

import (
	"sync"

	"github.com/goccy/go-json"
	"github.com/ronaksoft/ronykit"
)

type incomingMessage struct {
	Header  map[string]string `json:"hdr"`
	Payload json.RawMessage   `json:"payload"`
}

func (e *incomingMessage) Unmarshal(m ronykit.Message) (err error) {
	return json.Unmarshal(e.Payload, m)
}

type outgoingMessage struct {
	Header  map[string]string `json:"hdr"`
	Payload ronykit.Message   `json:"payload"`
}

func (e *outgoingMessage) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

var outgoingMessagePool sync.Pool

func acquireOutgoingMessage() *outgoingMessage {
	e, ok := outgoingMessagePool.Get().(*outgoingMessage)
	if !ok {
		e = &outgoingMessage{
			Header: map[string]string{},
		}
	}

	return e
}

func releaseOutgoingMessage(e *outgoingMessage) {
	for k := range e.Header {
		delete(e.Header, k)
	}

	outgoingMessagePool.Put(e)
}
