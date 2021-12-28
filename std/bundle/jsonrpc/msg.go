package jsonrpc

import (
	"github.com/goccy/go-json"
	"github.com/ronaksoft/ronykit"
)

type Envelope struct {
	Header    map[string]string `json:"hdr"`
	Predicate string            `json:"predicate"`
	ID        int64             `json:"id"`
	Payload   json.RawMessage   `json:"payload"`
}

func (e *Envelope) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func (e *Envelope) SetPayload(m ronykit.Message) (err error) {
	e.Payload, err = m.Marshal()

	return
}

func (e *Envelope) Unmarshal(m ronykit.Message) (err error) {
	return json.Unmarshal(e.Payload, m)
}
