package rpc

import (
	"sync"

	"github.com/goccy/go-json"
	"github.com/ronaksoft/ronykit"
)

type MessageContainer struct {
	Header    map[string]string `json:"hdr"`
	Predicate string            `json:"predicate"`
	ID        int64             `json:"id"`
	Payload   json.RawMessage   `json:"payload"`
}

func (e *MessageContainer) SetPayload(m ronykit.Message) (err error) {
	e.Payload, err = m.Marshal()

	return
}

func (e *MessageContainer) SetHeader(k, v string) {
	e.Header[k] = v
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
	e.Predicate = e.Predicate[:0]
	e.ID = 0

	containerPool.Put(e)
}
