package rest

import (
	"sync"

	"github.com/ronaksoft/ronykit"
)

type pool struct {
	p sync.Pool
}

var envelopePool = &pool{}

func (p *pool) Get() *Envelope {
	e, ok := p.p.Get().(*Envelope)
	if !ok {
		e = NewEnvelope()
	}

	return e
}

func (p *pool) Release(envelope ronykit.Envelope) {
	e, ok := envelope.(*Envelope)
	if !ok {
		return
	}

	p.p.Put(e)
}
