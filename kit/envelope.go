package kit

import (
	"sync"

	"github.com/clubpay/ronykit/kit/utils"
)

var envelopePool = &sync.Pool{}

type Walker interface {
	Walk(f func(k, v string) bool)
}

type EnvelopeHdr map[string]string

// ModifierFunc is a function that can modify the outgoing Envelope before sending it to the
// client. ModifierFunc only applies to outgoing envelopes.
type ModifierFunc func(envelope *Envelope)

// Envelope is an envelope around Message in RonyKIT. Envelopes are created internally
// by the RonyKIT framework and provide the abstraction which Bundle implementations could
// take advantage of. For example, in std/fasthttp Envelope headers translate from/to http
// request/response headers.
type Envelope struct {
	id       []byte
	ctx      *Context
	conn     Conn
	kvl      utils.SpinLock
	kv       EnvelopeHdr
	m        Message
	sizeHint int

	// outgoing identity of the Envelope if it is able to send
	outgoing bool

	// reuse identifies if the Envelope is going to be reused by pushing to the pool.
	reuse bool
	p     *sync.Pool
}

func newEnvelope(ctx *Context, conn Conn, outgoing bool) *Envelope {
	e, ok := envelopePool.Get().(*Envelope)
	if !ok {
		e = &Envelope{
			kv:    EnvelopeHdr{},
			p:     envelopePool,
			reuse: true,
		}
	}

	for k, v := range ctx.hdr {
		e.kv[k] = v
	}

	e.ctx = ctx
	e.conn = conn
	e.outgoing = outgoing

	return e
}

func (e *Envelope) release() {
	if !e.reuse {
		return
	}

	for k := range e.kv {
		delete(e.kv, k)
	}
	e.m = nil
	e.ctx = nil
	e.conn = nil
	e.id = e.id[:0]

	e.p.Put(e)
}

func (e *Envelope) dontReuse() {
	e.reuse = false
}

func (e *Envelope) GetID() string {
	return string(e.id)
}

func (e *Envelope) SetID(id string) *Envelope {
	e.id = append(e.id[:0], id...)

	return e
}

func (e *Envelope) SetHdr(key, value string) *Envelope {
	e.kvl.Lock()
	e.kv[key] = value
	e.kvl.Unlock()

	return e
}

func (e *Envelope) SetHdrWalker(walker Walker) *Envelope {
	e.kvl.Lock()
	walker.Walk(e.walkFunc)
	e.kvl.Unlock()

	return e
}

func (e *Envelope) walkFunc(k, v string) bool {
	e.kv[k] = v

	return true
}

func (e *Envelope) SetHdrMap(kv map[string]string) *Envelope {
	if kv == nil {
		return e
	}

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

func (e *Envelope) WalkHdr(f func(key string, val string) bool) *Envelope {
	e.kvl.Lock()
	for k, v := range e.kv {
		if !f(k, v) {
			break
		}
	}
	e.kvl.Unlock()

	return e
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

// Send writes the envelope to the connection based on the Gateway specification.
// You **MUST NOT** use the Envelope after calling this method.
// You **MUST NOT** call this function more than once.
// You **MUST NOT** call this method on incoming envelopes.
// You **MUST NOT** call this on the Envelope that you get from Context.In
func (e *Envelope) Send() {
	if e.conn == nil {
		panic("BUG!! do not call Send on nil conn, maybe called multiple times ?!")
	}

	if !e.outgoing {
		panic("BUG!! do not call Send on incoming envelope")
	}

	// run the modifiers in LIFO order
	modifiersCount := len(e.ctx.modifiers) - 1
	for idx := range e.ctx.modifiers {
		e.ctx.modifiers[modifiersCount-idx](e)
	}

	// Use WriteFunc to write the Envelope into the connection
	e.ctx.Error(e.conn.WriteEnvelope(e))

	// Release the envelope
	e.release()
}

// Reply creates a new envelope which it's id is
func (e *Envelope) Reply() *Envelope {
	return newEnvelope(e.ctx, e.conn, true).
		SetID(utils.B2S(e.id))
}

// IsOutgoing returns `true` if this Envelope is sending from Server to Client.
func (e *Envelope) IsOutgoing() bool {
	return e.outgoing
}

func (e *Envelope) SetSizeHint(size int) *Envelope {
	e.sizeHint = size

	return e
}

func (e *Envelope) SizeHint() int {
	if e.sizeHint == 0 {
		return CodecDefaultBufferSize
	}

	return e.sizeHint
}
