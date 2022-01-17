package rpc

import (
	"sync"

	"github.com/goccy/go-json"
	"github.com/ronaksoft/ronykit"
)

type Envelope struct {
	Header    map[string]string `json:"hdr"`
	Predicate string            `json:"predicate"`
	ID        int64             `json:"id"`
	Payload   json.RawMessage   `json:"payload"`
}

func (e *Envelope) SetPayload(m ronykit.Message) (err error) {
	e.Payload, err = m.Marshal()

	return
}

func (e *Envelope) SetHeader(k, v string) {
	e.Header[k] = v
}

func (e *Envelope) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e *Envelope) Unmarshal(m ronykit.Message) (err error) {
	return json.Unmarshal(e.Payload, m)
}

var envelopePool sync.Pool

func acquireEnvelope() *Envelope {
	e, ok := envelopePool.Get().(*Envelope)
	if !ok {
		e = &Envelope{
			Header: map[string]string{},
		}
	}

	return e
}

func releaseEnvelope(e *Envelope) {
	for k := range e.Header {
		delete(e.Header, k)
	}
	e.Predicate = e.Predicate[:0]
	e.ID = 0

	envelopePool.Put(e)
}
