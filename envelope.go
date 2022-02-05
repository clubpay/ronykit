package ronykit

import (
	"sync"

	"github.com/ronaksoft/ronykit/utils"
)

var envelopePool = &sync.Pool{}

type Envelope struct {
	ctx  *Context
	conn Conn
	kvl  utils.SpinLock
	kv   map[string]string
	m    Message
	p    *sync.Pool
}

func newEnvelope(ctx *Context, conn Conn) *Envelope {
	e, ok := envelopePool.Get().(*Envelope)
	if !ok {
		e = &Envelope{
			kv: map[string]string{},
			p:  envelopePool,
		}
	}

	for k, v := range ctx.hdr {
		e.kv[k] = v
	}

	e.ctx = ctx
	e.conn = conn

	return e
}

func (e *Envelope) release() {
	for k := range e.kv {
		delete(e.kv, k)
	}
	e.m = nil
	e.ctx = nil
	e.conn = nil

	e.p.Put(e)
}

func (e *Envelope) SetHdr(key, value string) *Envelope {
	e.kvl.Lock()
	e.kv[key] = value
	e.kvl.Unlock()

	return e
}

func (e *Envelope) SetHdrMap(kv map[string]string) *Envelope {
	e.kvl.Lock()
	for k, v := range kv {
		e.kv[k] = v
	}
	e.kvl.Unlock()

	return e
}

func (e *Envelope) GetHdr(key string) string {
	e.kvl.Lock()
	v := e.kv[key]
	e.kvl.Unlock()

	return v
}

func (e *Envelope) WalkHdr(f func(key string, val string) bool) {
	e.kvl.Lock()
	for k, v := range e.kv {
		if !f(k, v) {
			break
		}
	}
	e.kvl.Unlock()
}

func (e *Envelope) SetMsg(msg Message) *Envelope {
	e.m = msg

	return e
}

func (e *Envelope) GetMsg() Message {
	if e.m == nil {
		return nil
	}

	return e.m
}

// Send writes the envelope to the connection based on the Bundle specification.
// You **MUST NOT** use the Envelope after calling this method.
// You **MUST NOT** call this function more than once.
func (e *Envelope) Send() {
	if e.conn == nil {
		panic("BUG!! do not call Send multiple times")
	}

	// Use WriteFunc to write the Envelope into the connection
	e.ctx.Error(e.ctx.wf(e.conn, e))

	// Release the envelope
	e.release()
}

type MessageFactory func() Message

type Message interface {
	Marshal() ([]byte, error)
}

// RawMessage is a bytes slice which could be used as Message. This is helpful for
// raw data messages.
type RawMessage []byte

func (rm RawMessage) Marshal() ([]byte, error) {
	return rm, nil
}

// ErrorMessage is a special kind of Message which is also an error.
type ErrorMessage interface {
	Message
	error
}
