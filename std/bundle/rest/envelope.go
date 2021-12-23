package rest

import (
	"sync"

	"github.com/goccy/go-json"
	"github.com/ronaksoft/ronykit"
)

var _ ronykit.Envelope = &Envelope{}

type Envelope struct {
	sync.Mutex
	HDR     map[string]interface{}
	Message ronykit.Message
}

func NewEnvelope() *Envelope {
	return &Envelope{
		HDR: map[string]interface{}{},
	}
}

func (e *Envelope) Get(key string) (interface{}, bool) {
	v, ok := e.HDR[key]

	return v, ok
}

func (e *Envelope) Set(key string, val interface{}) {
	e.Lock()
	e.HDR[key] = val
	e.Unlock()
}

func (e *Envelope) Method() string {
	return e.HDR[Method].(string)
}

func (e *Envelope) Path() string {
	return e.HDR[Path].(string)
}

func (e *Envelope) GetHeader() ronykit.EnvelopeHeader {
	return e
}

func (e *Envelope) GetMessage() ronykit.Message {
	return e.Message
}

type Message struct {
	Status string          `json:"status"`
	Body   json.RawMessage `json:"body"`
}

func (m *Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *Message) Unmarshal(data []byte) error {
	return json.Unmarshal(data, m)
}

type Container struct {
	Messages []*Message `json:"messages"`
}

func (c *Container) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

func (c *Container) Unmarshal(data []byte) error {
	return json.Unmarshal(data, c)
}
