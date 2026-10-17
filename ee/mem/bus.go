package mem

import (
	"context"
	"errors"
	"hash/fnv"
	"sync"
	"time"

	"github.com/clubpay/ronykit/ee/bus"
)

// Bus is an in-memory bus.Bus with consumer groups, per-key ordering and
// redelivery. Envelopes published to an event nobody subscribes to are
// dropped.
//
// Each group has a fixed number of lanes; envelopes with the same partition
// key always go to the same lane and are delivered one at a time, in order.
// A failing delivery is retried in place (blocking its lane) up to
// MaxAttempts, then handed to the dead-letter callback.
type Bus struct {
	lanes       int
	maxAttempts int
	backoff     time.Duration
	onDead      func(bus.Envelope, error)

	ctx    context.Context //nolint:containedctx // lifetime of the bus workers
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu     sync.Mutex
	closed bool
	groups map[string]*group
}

type BusOption func(*Bus)

// Lanes sets the number of ordered lanes per group (default 16).
func Lanes(n int) BusOption { return func(b *Bus) { b.lanes = n } }

// MaxAttempts caps deliveries of one envelope (default 10; 0 = unlimited).
func MaxAttempts(n int) BusOption { return func(b *Bus) { b.maxAttempts = n } }

// RetryBackoff is multiplied by the attempt number between redeliveries
// (default 10ms).
func RetryBackoff(d time.Duration) BusOption { return func(b *Bus) { b.backoff = d } }

// OnDeadLetter receives envelopes that exhausted MaxAttempts.
func OnDeadLetter(fn func(bus.Envelope, error)) BusOption {
	return func(b *Bus) { b.onDead = fn }
}

func NewBus(opts ...BusOption) *Bus {
	b := &Bus{
		lanes:       16,
		maxAttempts: 10,
		backoff:     10 * time.Millisecond,
		onDead:      func(bus.Envelope, error) {},
		groups:      map[string]*group{},
	}
	for _, o := range opts {
		o(b)
	}

	if b.lanes < 1 {
		b.lanes = 1
	}

	b.ctx, b.cancel = context.WithCancel(context.Background())

	return b
}

var (
	_ bus.Bus       = (*Bus)(nil)
	_ bus.Requester = (*Bus)(nil)
)

var errClosed = errors.New("mem: bus closed")

type group struct {
	events map[string]bool
	subs   []*subscription
	next   int
	lanes  []*queue
}

type subscription struct {
	b      *Bus
	g      *group
	h      bus.Handler
	closed bool
}

func (s *subscription) Close() error {
	s.b.mu.Lock()
	defer s.b.mu.Unlock()

	s.closed = true

	return nil
}

func (b *Bus) Subscribe(
	_ context.Context,
	name string,
	events []string,
	h bus.Handler,
) (bus.Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil, errClosed
	}

	g, ok := b.groups[name]
	if !ok {
		g = &group{events: map[string]bool{}}

		for range b.lanes {
			q := newQueue()
			g.lanes = append(g.lanes, q)

			b.wg.Go(func() { b.work(g, q) })
		}

		b.groups[name] = g
	}

	for _, e := range events {
		g.events[e] = true
	}

	s := &subscription{b: b, g: g, h: h}
	g.subs = append(g.subs, s)

	return s, nil
}

func (b *Bus) Publish(_ context.Context, envs ...bus.Envelope) error {
	for _, e := range envs {
		if _, err := b.route(bus.Message{Envelope: e}); err != nil {
			return err
		}
	}

	return nil
}

// Request delivers env to its (single) consuming group and waits for the
// handler to call msg.Reply.
func (b *Bus) Request(ctx context.Context, env bus.Envelope) (bus.Envelope, error) {
	replies := make(chan bus.Envelope, 1)
	msg := bus.Message{
		Envelope: env,
		Reply: func(_ context.Context, r bus.Envelope) error {
			select {
			case replies <- r:
			default: // a redelivery already replied
			}

			return nil
		},
	}

	n, err := b.route(msg)
	if err != nil {
		return bus.Envelope{}, err
	}

	if n == 0 {
		return bus.Envelope{}, errors.New("mem: no subscriber for " + env.Event)
	}

	select {
	case r := <-replies:
		return r, nil
	case <-ctx.Done():
		return bus.Envelope{}, ctx.Err()
	}
}

func (b *Bus) route(msg bus.Message) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return 0, errClosed
	}

	n := 0

	for _, g := range b.groups {
		if !g.events[msg.Event] {
			continue
		}

		key := msg.PartitionKey
		if key == "" {
			key = msg.ID
		}

		h := fnv.New32a()
		_, _ = h.Write([]byte(key))
		g.lanes[h.Sum32()%uint32(len(g.lanes))].push(msg)

		n++
	}

	return n, nil
}

// pick returns the next live subscriber of g, round-robin.
func (b *Bus) pick(g *group) bus.Handler {
	b.mu.Lock()
	defer b.mu.Unlock()

	for range g.subs {
		s := g.subs[g.next%len(g.subs)]
		g.next++

		if !s.closed {
			return s.h
		}
	}

	return nil
}

func (b *Bus) work(g *group, q *queue) {
	for {
		msg, ok := q.pop()
		if !ok {
			return
		}

		b.deliver(g, msg)
	}
}

func (b *Bus) deliver(g *group, msg bus.Message) {
	for attempt := 1; ; attempt++ {
		h := b.pick(g)
		if h == nil {
			return // every member left the group
		}

		err := h(b.ctx, msg)
		if err == nil {
			return
		}

		if b.maxAttempts > 0 && attempt >= b.maxAttempts {
			b.onDead(msg.Envelope, err)

			return
		}

		select {
		case <-b.ctx.Done():
			return
		case <-time.After(b.backoff * time.Duration(attempt)):
		}
	}
}

// Close stops every worker. In-flight deliveries finish; queued ones are
// dropped.
func (b *Bus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()

		return nil
	}

	b.closed = true
	for _, g := range b.groups {
		for _, q := range g.lanes {
			q.close()
		}
	}
	b.mu.Unlock()

	b.cancel()
	b.wg.Wait()

	return nil
}

// queue is an unbounded FIFO, so Publish never blocks (a handler may publish
// to its own lane).
type queue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	items  []bus.Message
	closed bool
}

func newQueue() *queue {
	q := &queue{}
	q.cond = sync.NewCond(&q.mu)

	return q
}

func (q *queue) push(m bus.Message) {
	q.mu.Lock()
	q.items = append(q.items, m)
	q.mu.Unlock()
	q.cond.Signal()
}

func (q *queue) pop() (bus.Message, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.items) == 0 && !q.closed {
		q.cond.Wait()
	}

	if q.closed {
		return bus.Message{}, false
	}

	m := q.items[0]
	q.items[0] = bus.Message{}
	q.items = q.items[1:]

	return m, true
}

func (q *queue) close() {
	q.mu.Lock()
	q.closed = true
	q.mu.Unlock()
	q.cond.Broadcast()
}
