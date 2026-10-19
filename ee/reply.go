package ee

import (
	"context"
	"fmt"

	"github.com/clubpay/ronykit/ee/bus"
)

// Request publishes ev and waits for the consuming action to call Reply.
// R is decoded from the reply payload with the env codec; use a value type
// the codec can unmarshal (a struct, not a nil pointer).
//
// The bus must implement bus.Requester. Kafka does not. ErrNotSupported
// otherwise. The call blocks until the handler replies or ctx ends.
func Request[R any, E Event](ctx context.Context, env *Env, ev E, opts ...SendOption) (R, error) {
	var zero R

	if env.bus == nil {
		return zero, fmt.Errorf("%w: request", ErrNotSupported)
	}

	rq, ok := env.bus.(bus.Requester)
	if !ok {
		return zero, fmt.Errorf("%w: bus has no request/reply", ErrNotSupported)
	}

	envelope, err := env.envelope(ev, opts)
	if err != nil {
		return zero, err
	}

	reply, err := rq.Request(ctx, envelope)
	if err != nil {
		return zero, err
	}

	if len(reply.Payload) == 0 {
		return zero, nil
	}

	if err := env.codec.Unmarshal(reply.Payload, &zero); err != nil {
		return zero, fmt.Errorf("ee: decode reply: %w", err)
	}

	return zero, nil
}

func replyValue(env *Env, reply func(context.Context, bus.Envelope) error, ctx context.Context, v any) error {
	if reply == nil {
		return fmt.Errorf("%w: not a request", ErrNotSupported)
	}

	payload, err := env.codec.Marshal(v)
	if err != nil {
		return fmt.Errorf("ee: encode reply: %w", err)
	}

	return reply(ctx, bus.Envelope{Payload: payload, CreatedAt: env.now()})
}
