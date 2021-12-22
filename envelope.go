package ronykit

import "sync"

type EnvelopePool interface {
	Release(envelope Envelope)
}

type Envelope interface {
	sync.Locker // To inform linter Envelope must not be copied.
	Header() EnvelopeHeader
	SetSubject(string)
	GetSubject() string
	SetBody([]byte)
	GetBody() []byte
}

type EnvelopeHeader interface {
	Get(key string) (string, bool)
	Set(key, val string)
}
