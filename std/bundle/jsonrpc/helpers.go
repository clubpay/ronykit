package jsonrpc

import (
	"github.com/ronaksoft/ronykit"
)

type EnvelopeOption func(*Envelope)

func WithHeader(k, v string) EnvelopeOption {
	return func(e *Envelope) {
		e.Header[k] = v
	}
}

func NewEnvelope(
	id int64, predicate string, m ronykit.Message,
) (*Envelope, error) {
	env := &Envelope{
		Header:    map[string]string{},
		Predicate: predicate,
		ID:        id,
	}
	if err := env.SetPayload(m); err != nil {
		return nil, err
	}

	return env, nil
}

func SendEnvelope(
	ctx *ronykit.Context, id int64, predicate string, m ronykit.Message, opts ...EnvelopeOption,
) {
	envelope, err := NewEnvelope(id, predicate, m)
	if ctx.Error(err) {
		return
	}

	for _, o := range opts {
		o(envelope)
	}

	ctx.Send(envelope)
}
