package bustest_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/bus/bustest"
)

// syncBus drives the suite synchronously: Publish returns after every group
// has consumed the envelope, retrying a failing delivery in place to keep
// per-key order. It does not implement bus.Requester, so the suite exercises
// its skip branch for that case.
type syncBus struct {
	mu     sync.Mutex
	closed bool
	groups map[string]*syncGroup
}

type syncGroup struct {
	events map[string]bool
	subs   []bus.Handler
	closed []bool
	next   int
}

func newSyncBus() *syncBus { return &syncBus{groups: map[string]*syncGroup{}} }

func (b *syncBus) Subscribe(_ context.Context, name string, events []string, h bus.Handler) (bus.Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, errors.New("sync: bus closed")
	}

	g, ok := b.groups[name]
	if !ok {
		g = &syncGroup{events: map[string]bool{}}
		b.groups[name] = g
	}

	start := len(g.subs)

	for _, e := range events {
		g.events[e] = true
	}

	g.subs = append(g.subs, h)
	g.closed = append(g.closed, false)

	i := start

	return syncSub{b: b, g: g, i: &i}, nil
}

type syncSub struct {
	b *syncBus
	g *syncGroup
	i *int
}

func (s syncSub) Close() error {
	s.b.mu.Lock()
	defer s.b.mu.Unlock()

	s.g.closed[*s.i] = true

	return nil
}

func (b *syncBus) Publish(_ context.Context, envs ...bus.Envelope) error {
	for _, env := range envs {
		b.mu.Lock()

		if b.closed {
			b.mu.Unlock()

			return errors.New("sync: bus closed")
		}

		var targets []bus.Handler

		for _, g := range b.groups {
			if !g.events[env.Event] {
				continue
			}
			// Competing consumers: exactly one live member per group.
			h := g.next
			g.next++

			for range len(g.subs) {
				if i := h % len(g.subs); !g.closed[i] {
					targets = append(targets, g.subs[i])

					break
				}

				h++
			}
		}

		b.mu.Unlock()

		for _, h := range targets {
			// In-place retries keep a failing envelope ahead of its key.
			for attempt := range 8 {
				if err := h(context.Background(), bus.Message{Envelope: env}); err == nil {
					break
				} else if attempt == 7 {
					// exhausted: drop
				}
			}
		}
	}

	return nil
}

func (b *syncBus) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true

	return nil
}

func TestSuiteAgainstSyncBus(t *testing.T) {
	bustest.Run(t, func(*testing.T) bus.Bus { return newSyncBus() })
}
