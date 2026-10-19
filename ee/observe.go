package ee

import "time"

// Observation kinds passed to WithObserver.
const (
	// ObsConflict is one optimistic-concurrency retry of a unit of work.
	ObsConflict = "conflict"
	// ObsPanic is a handler panic. The unit of work rolls back and the bus
	// redelivers until its own attempt limit.
	ObsPanic = "panic"
	// ObsDropped is an event acknowledged without running its effects
	// (undecodable payload).
	ObsDropped = "dropped"
	// ObsDeliveryError is a handler error returned to the bus for redelivery.
	ObsDeliveryError = "delivery_error"
	// ObsOutboxPublished is a relay batch accepted by the bus.
	ObsOutboxPublished = "outbox_published"
	// ObsOutboxDropped is one outbox entry whose payload is not an envelope.
	ObsOutboxDropped = "outbox_dropped"
	// ObsOutboxError is a claim, publish or ack failure. The lease expires
	// and another relay retries the entries.
	ObsOutboxError = "outbox_error"
)

// Observation is one runtime signal. Unused fields stay zero.
type Observation struct {
	Kind    string
	Attempt int
	Count   int
	Err     error
	Event   string
	ID      string
	Owner   string
}

// Stats is what Env.Stats reports.
type Stats struct {
	Started       bool
	Controllers   int
	Subscriptions int
	// OutboxPending is the unpublished outbox backlog, or -1 if the store
	// cannot report it.
	OutboxPending int
	OutboxOldest  time.Time
}
