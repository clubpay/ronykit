package rpc

import (
	"sync"

	"github.com/goccy/go-json"
	"github.com/ronaksoft/ronykit"
)

type MessageContainer struct {
	Header  map[string]string `json:"hdr"`
	Payload json.RawMessage   `json:"payload"`
}

func (e *MessageContainer) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e *MessageContainer) Unmarshal(m ronykit.Message) (err error) {
	return json.Unmarshal(e.Payload, m)
}

var containerPool sync.Pool

func acquireMsgContainer() *MessageContainer {
	e, ok := containerPool.Get().(*MessageContainer)
	if !ok {
		e = &MessageContainer{
			Header: map[string]string{},
		}
	}

	return e
}

func releaseEnvelope(e *MessageContainer) {
	for k := range e.Header {
		delete(e.Header, k)
	}

	containerPool.Put(e)
}
