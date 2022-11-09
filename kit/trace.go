package kit

import "context"

type Tracer interface {
	Handler() HandlerFunc
	Propagator() TracePropagator
}

// TracePropagator propagates cross-cutting concerns as key-value text
// pairs within a carrier that travels in-band across process boundaries.
type TracePropagator interface {
	// Inject set cross-cutting concerns from the Context into the carrier.
	Inject(ctx context.Context, carrier TraceCarrier)

	// Extract reads cross-cutting concerns from the carrier into a Context.
	Extract(ctx context.Context, carrier TraceCarrier) context.Context
}

// TraceCarrier is the storage medium used by a TracePropagator.
type TraceCarrier interface {
	// Get returns the value associated with the passed key.
	Get(key string) string

	// Set stores the key-value pair.
	Set(key string, value string)
}
