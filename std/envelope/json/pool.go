package json

import (
	"sync"

	"github.com/ronaksoft/ronykit"
)

type Pool struct {
	p sync.Pool
}

func NewPool() *Pool {
	return &Pool{}
}

func (p *Pool) Get() ronykit.Envelope {
	e, ok := p.p.Get().(*Envelope)
	if !ok {
		e = NewEnvelope()
	}

	return e
}

func (p *Pool) Release(envelope ronykit.Envelope) {
	e, ok := envelope.(*Envelope)
	if !ok {
		return
	}

	p.p.Put(e)
}
