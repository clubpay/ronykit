// Package bus defines the transport contract that ee event-bus backends
// implement (in-memory in ee/mem, NATS and Kafka under std/ee/bus).
//
// The contract is deliberately untyped: backends move Envelopes and never see
// Go event types. All typing lives in the ee package.
package bus

import (
	"context"
	"errors"
	"time"
)

// ErrNotSupported is returned by optional capabilities a backend cannot offer.
var ErrNotSupported = errors.New("bus: not supported")

// Envelope is the unit of transport. Payload is already encoded by the ee
// codec; backends must treat it as opaque bytes. The JSON form is the outbox
// wire format, shared by every replica, so field names are fixed by tags.
type Envelope struct {
	// ID is unique per logical event and stays the same across redeliveries,
	// so consumers can deduplicate on it.
	ID string `json:"id"`
	// Event is the event name (EventName), e.g. "order.paid".
	Event string `json:"event"`
	// PartitionKey selects the ordering lane. Envelopes with the same key are
	// delivered in publish order to a single consumer of a group.
	PartitionKey string `json:"partition_key,omitempty"`
	// Labels are the event's Labels(), matched by
	// action selectors on the consumer side.
	Labels    map[string]string `json:"labels,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Payload   []byte            `json:"payload"`
	CreatedAt time.Time         `json:"created_at"`
	// ExpiresAt is optional; the zero value means the envelope never expires.
	ExpiresAt time.Time `json:"expires_at"`
}

// Expired reports whether the envelope must be dropped instead of delivered.
func (e *Envelope) Expired(now time.Time) bool {
	return !e.ExpiresAt.IsZero() && !now.Before(e.ExpiresAt)
}

// Message is what a Handler receives.
type Message struct {
	Envelope
	// Reply is non-nil only when the publisher used Requester.Request and is
	// waiting for an answer.
	Reply func(ctx context.Context, reply Envelope) error
}

// Handler consumes a message. Returning an error asks the backend to
// redeliver it later (at-least-once); returning nil acknowledges it.
type Handler func(ctx context.Context, msg Message) error

// Subscription is an active consumer registration.
type Subscription interface {
	Close() error
}

// Bus is the contract every event-bus backend implements.
type Bus interface {
	// Publish sends envelopes. It returns only after the backend has durably
	// accepted all of them.
	Publish(ctx context.Context, envs ...Envelope) error
	// Subscribe consumes the given event names as part of group. Each
	// envelope is delivered to exactly one member of a group (and redelivered
	// on error); different groups each receive every envelope.
	Subscribe(ctx context.Context, group string, events []string, h Handler) (Subscription, error)
	Close() error
}

// Requester is an optional capability for synchronous request/reply.
// Backends that cannot provide it (e.g. Kafka) simply don't implement it.
type Requester interface {
	Request(ctx context.Context, env Envelope) (Envelope, error)
}
