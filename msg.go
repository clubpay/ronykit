package ronykit

import "sync"

type EnvelopeFactory interface {
	Get() Envelope
	Put(envelope Envelope)
}

type Envelope interface {
	sync.Locker // To inform linter Envelope must not be copied.
	Header() EnvelopeHeader
	SetBody([]byte)
	GetBody() []byte
}

type EnvelopeHeader interface {
	Get(key string) string
	Set(key, val string)
}
