package ee

import (
	"crypto/rand"
	"fmt"
	"maps"
	"time"

	"github.com/clubpay/ronykit/ee/bus"
)

// Event is implemented by every event type. EventName must be a constant of
// the type (it is called on the zero value) and must use a value receiver.
// Versioning is done with a new type and a new name, e.g. "order.paid.v2".
type Event interface {
	EventName() string
}

// Keyed is optionally implemented by events about one entity or aggregate.
// The key is both the partition key (events with the same key are processed
// in order) and the id ctx.Entity()/ctx.Aggregate() load.
type Keyed interface {
	Key() string
}

// KeyedEvent is an Event that implements Keyed. Aggregate actions require it.
type KeyedEvent interface {
	Event
	Keyed
}

// Labeled is optionally implemented by events that carry labels for action
// Filters.
type Labeled interface {
	Labels() L
}

// L is a label set.
type L = map[string]string

// Meta is the envelope metadata of a received event.
type Meta struct {
	ID           string
	Event        string
	PartitionKey string
	Labels       L
	Headers      map[string]string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

func metaOf(env *bus.Envelope) Meta {
	return Meta{
		ID:           env.ID,
		Event:        env.Event,
		PartitionKey: env.PartitionKey,
		Labels:       env.Labels,
		Headers:      env.Headers,
		CreatedAt:    env.CreatedAt,
		ExpiresAt:    env.ExpiresAt,
	}
}

type sendConfig struct {
	expiresAt time.Time
	ttl       time.Duration
	headers   map[string]string
}

type SendOption func(*sendConfig)

func ExpireIn(d time.Duration) SendOption { return func(c *sendConfig) { c.ttl = d } }
func ExpireAt(t time.Time) SendOption     { return func(c *sendConfig) { c.expiresAt = t } }

func WithHeader(k, v string) SendOption {
	return func(c *sendConfig) {
		if c.headers == nil {
			c.headers = map[string]string{}
		}

		c.headers[k] = v
	}
}

// envelope builds the transport envelope of ev.
func (e *Env) envelope(ev Event, opts []SendOption) (bus.Envelope, error) {
	name := ev.EventName()

	payload, err := e.codec.Marshal(ev)
	if err != nil {
		return bus.Envelope{}, fmt.Errorf("ee: encode %s: %w", name, err)
	}

	cfg := sendConfig{}
	for _, o := range opts {
		o(&cfg)
	}

	now := e.now()
	env := bus.Envelope{
		ID:        rand.Text(),
		Event:     name,
		Headers:   maps.Clone(cfg.headers),
		Payload:   payload,
		CreatedAt: now,
		ExpiresAt: cfg.expiresAt,
	}

	if env.ExpiresAt.IsZero() && cfg.ttl > 0 {
		env.ExpiresAt = now.Add(cfg.ttl)
	}

	if k, ok := ev.(Keyed); ok {
		env.PartitionKey = k.Key()
	}

	if l, ok := ev.(Labeled); ok {
		env.Labels = l.Labels()
	}

	return env, nil
}
