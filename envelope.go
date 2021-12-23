package ronykit

import "sync"

type EnvelopePool interface {
	Release(envelope Envelope)
}

type Envelope interface {
	sync.Locker // To inform linter Envelope must not be copied.
	GetHeader() EnvelopeHeader
	GetMessage() Message
}

type EnvelopeHeader interface {
	Get(key string) (interface{}, bool)
	Set(key string, val interface{})
}

type Message interface {
	Marshal() ([]byte, error)
	Unmarshal(data []byte) error
}
