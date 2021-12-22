package json

import (
	"encoding/json"
	"github.com/ronaksoft/ronykit"
	"sync"
)

type Envelope struct {
	sync.Mutex
	HDR     map[string]string `json:"hdr"`
	Subject string            `json:"subject"`
	Body    json.RawMessage   `json:"body"`
}

func NewEnvelope() *Envelope {
	return &Envelope{
		HDR: map[string]string{},
	}
}

func (e *Envelope) Get(key string) (string, bool) {
	v, ok := e.HDR[key]

	return v, ok
}

func (e *Envelope) Set(key, val string) {
	e.Lock()
	e.HDR[key] = val
	e.Unlock()
}

func (e *Envelope) Header() ronykit.EnvelopeHeader {
	return e
}

func (e *Envelope) SetSubject(s string) {
	e.Subject = s
}

func (e *Envelope) GetSubject() string {
	return e.Subject
}

func (e *Envelope) SetBody(bytes []byte) {
	e.Body = bytes
}

func (e *Envelope) GetBody() []byte {
	return e.Body
}
