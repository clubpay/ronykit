// Package bustest is a conformance suite for bus.Bus backends. Every backend
// runs it from its own tests:
//
//	func TestConformance(t *testing.T) {
//		bustest.Run(t, func(t *testing.T) bus.Bus { return newIsolatedBus(t) })
//	}
//
// newBus is called once per subtest and must return a bus isolated from the
// others (fresh stream/topic prefix): groups start from the beginning of
// their events, so leftovers would leak between subtests. The suite closes
// the bus.
package bustest

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee/bus"
)

// Timeout bounds every wait; brokers with slow group joins (Kafka) need a
// generous one.
var Timeout = 60 * time.Second

func Run(t *testing.T, newBus func(t *testing.T) bus.Bus) {
	t.Helper()

	tests := []struct {
		name string
		fn   func(t *testing.T, b bus.Bus)
	}{
		{"EnvelopeRoundTrip", testRoundTrip},
		{"GroupsAndCompetingMembers", testGroups},
		{"PerKeyOrderAcrossMembers", testOrdering},
		{"PerKeyOrderAcrossEvents", testOrderingAcrossEvents},
		{"RedeliveryKeepsOrder", testRedelivery},
		{"OnlySubscribedEvents", testFiltering},
		{"CloseStopsDelivery", testClose},
		{"RequestReply", testRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := newBus(t)
			t.Cleanup(func() { _ = b.Close() })
			tc.fn(t, b)
		})
	}
}

// recorder collects deliveries and lets tests wait for a count.
type recorder struct {
	mu   sync.Mutex
	msgs []bus.Message
	by   map[string][]string // tag -> envelope IDs in delivery order
}

func newRecorder() *recorder { return &recorder{by: map[string][]string{}} }

func (r *recorder) handler(tag string) bus.Handler {
	return func(_ context.Context, m bus.Message) error {
		r.add(tag, m)

		return nil
	}
}

func (r *recorder) add(tag string, m bus.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.msgs = append(r.msgs, m)
	r.by[tag] = append(r.by[tag], m.ID)
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.msgs)
}

func (r *recorder) snapshot() map[string][]string {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make(map[string][]string, len(r.by))
	for k, v := range r.by {
		out[k] = slices.Clone(v)
	}

	return out
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()

	for deadline := time.Now().Add(Timeout); time.Now().Before(deadline); {
		if cond() {
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", what)
}

// settle gives a backend time to deliver anything it should not.
func settle() { time.Sleep(500 * time.Millisecond) }

func subscribe(t *testing.T, b bus.Bus, group string, events []string, h bus.Handler) bus.Subscription {
	t.Helper()

	s, err := b.Subscribe(t.Context(), group, events, h)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = s.Close() })

	return s
}

func publish(t *testing.T, b bus.Bus, envs ...bus.Envelope) {
	t.Helper()

	if err := b.Publish(t.Context(), envs...); err != nil {
		t.Fatal(err)
	}
}

func env(id, event, key string) bus.Envelope {
	return bus.Envelope{
		ID:           id,
		Event:        event,
		PartitionKey: key,
		Payload:      []byte(id),
		CreatedAt:    time.Now().UTC(),
	}
}

func testRoundTrip(t *testing.T, b bus.Bus) {
	t.Helper()

	r := newRecorder()
	subscribe(t, b, "g", []string{"order.paid"}, r.handler("g"))

	sent := bus.Envelope{
		ID:           "id-1",
		Event:        "order.paid",
		PartitionKey: "o1",
		Labels:       map[string]string{"merchant": "m1"},
		Headers:      map[string]string{"trace": "abc"},
		Payload:      []byte(`{"OrderID":"o1"}`),
		CreatedAt:    time.Date(2026, 10, 15, 12, 0, 0, 123, time.UTC),
		ExpiresAt:    time.Date(2026, 10, 15, 13, 0, 0, 0, time.UTC),
	}
	publish(t, b, sent)
	waitFor(t, "delivery", func() bool { return r.count() == 1 })

	got := r.msgs[0].Envelope
	if got.ID != sent.ID || got.Event != sent.Event || got.PartitionKey != sent.PartitionKey ||
		!maps.Equal(got.Labels, sent.Labels) || !maps.Equal(got.Headers, sent.Headers) ||
		string(got.Payload) != string(sent.Payload) ||
		!got.CreatedAt.Equal(sent.CreatedAt) || !got.ExpiresAt.Equal(sent.ExpiresAt) {
		t.Errorf("round trip:\n got %+v\nsent %+v", got, sent)
	}

	if r.msgs[0].Reply != nil {
		t.Error("Reply must be nil for published (non-request) messages")
	}
}

func testGroups(t *testing.T, b bus.Bus) {
	t.Helper()

	r := newRecorder()
	// Two members compete in group a; group b gets its own copy of every event.
	subscribe(t, b, "a", []string{"e"}, r.handler("a"))
	subscribe(t, b, "a", []string{"e"}, r.handler("a"))
	subscribe(t, b, "b", []string{"e"}, r.handler("b"))

	const n = 40
	for i := range n {
		publish(t, b, env(fmt.Sprint("m", i), "e", fmt.Sprint("k", i%8)))
	}

	waitFor(t, "2 copies of each event", func() bool { return r.count() >= 2*n })
	settle()

	got := r.snapshot()
	for _, g := range []string{"a", "b"} {
		ids := slices.Clone(got[g])
		slices.Sort(ids)

		if len(ids) != n || len(slices.Compact(ids)) != n {
			t.Errorf(
				"group %s: %d deliveries (%d distinct), want %d once each",
				g,
				len(got[g]),
				len(slices.Compact(ids)),
				n,
			)
		}
	}
}

func testOrdering(t *testing.T, b bus.Bus) {
	t.Helper()

	var (
		mu    sync.Mutex
		byKey = map[string][]int{}
		total int
	)

	h := func(_ context.Context, m bus.Message) error {
		var i int

		_, _ = fmt.Sscan(string(m.Payload), &i)

		mu.Lock()

		byKey[m.PartitionKey] = append(byKey[m.PartitionKey], i)
		total++
		mu.Unlock()

		return nil
	}

	subscribe(t, b, "g", []string{"e"}, h)
	subscribe(t, b, "g", []string{"e"}, h)

	const perKey = 30

	keys := []string{"k1", "k2", "k3", "k4"}
	envs := make([]bus.Envelope, 0, perKey*len(keys))

	for i := range perKey {
		for _, k := range keys {
			e := env(fmt.Sprint(k, "-", i), "e", k)
			e.Payload = fmt.Append(nil, i)
			envs = append(envs, e)
		}
	}

	publish(t, b, envs...)
	waitFor(t, "all deliveries", func() bool {
		mu.Lock()
		defer mu.Unlock()

		return total >= perKey*len(keys)
	})

	mu.Lock()
	defer mu.Unlock()

	for _, k := range keys {
		if !slices.IsSorted(byKey[k]) || len(byKey[k]) != perKey {
			t.Errorf("key %s delivered out of order or incomplete: %v", k, byKey[k])
		}
	}
}

// testOrderingAcrossEvents: events of different types with the same key
// (OrderCreated, then OrderPaid) must reach a group in publish order.
func testOrderingAcrossEvents(t *testing.T, b bus.Bus) {
	t.Helper()

	r := newRecorder()
	subscribe(t, b, "g", []string{"created", "paid", "shipped"}, r.handler("g"))

	const rounds = 10

	envs := make([]bus.Envelope, 0, 3*rounds)
	want := make([]string, 0, 3*rounds)

	for i := range rounds {
		for _, ev := range []string{"created", "paid", "shipped"} {
			id := fmt.Sprint(ev, "-", i)
			envs = append(envs, env(id, ev, "o1"))
			want = append(want, id)
		}
	}

	publish(t, b, envs...)
	waitFor(t, "all deliveries", func() bool { return r.count() >= len(want) })

	if got := r.snapshot()["g"]; !slices.Equal(got, want) {
		t.Errorf("per-key order across events broken:\n got %v\nwant %v", got, want)
	}
}

func testRedelivery(t *testing.T, b bus.Bus) {
	t.Helper()

	var (
		mu       sync.Mutex
		attempts = map[string]int{}
		order    []string
	)

	subscribe(t, b, "g", []string{"e"}, func(_ context.Context, m bus.Message) error {
		mu.Lock()
		defer mu.Unlock()

		attempts[m.ID]++
		if m.ID == "first" && attempts[m.ID] < 3 {
			return errors.New("try again")
		}

		order = append(order, m.ID)

		return nil
	})

	publish(t, b, env("first", "e", "k"), env("second", "e", "k"))
	waitFor(t, "both handled", func() bool {
		mu.Lock()
		defer mu.Unlock()

		return len(order) == 2
	})

	mu.Lock()
	defer mu.Unlock()

	if attempts["first"] != 3 || !slices.Equal(order, []string{"first", "second"}) {
		t.Errorf("attempts %v, order %v: a failed message must be redelivered before later ones of its key",
			attempts, order)
	}
}

func testFiltering(t *testing.T, b bus.Bus) {
	t.Helper()

	r := newRecorder()
	subscribe(t, b, "g", []string{"wanted"}, r.handler("g"))

	publish(t, b, env("1", "unwanted", "k"), env("2", "wanted", "k"))
	waitFor(t, "the subscribed event", func() bool { return r.count() >= 1 })
	settle()

	if got := r.snapshot()["g"]; !slices.Equal(got, []string{"2"}) {
		t.Errorf("deliveries = %v, want only the subscribed event", got)
	}
}

func testClose(t *testing.T, b bus.Bus) {
	t.Helper()

	r := newRecorder()
	s := subscribe(t, b, "g", []string{"e"}, r.handler("g"))

	publish(t, b, env("1", "e", "k"))
	waitFor(t, "first delivery", func() bool { return r.count() == 1 })

	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	publish(t, b, env("2", "e", "k"))
	settle()

	if n := r.count(); n != 1 {
		t.Errorf("%d deliveries after Close, want 1", n)
	}
}

func testRequest(t *testing.T, b bus.Bus) {
	t.Helper()

	req, ok := b.(bus.Requester)
	if !ok {
		t.Skip("bus does not implement bus.Requester")
	}

	subscribe(t, b, "g", []string{"ask"}, func(ctx context.Context, m bus.Message) error {
		if m.Reply == nil {
			return errors.New("request without Reply")
		}

		return m.Reply(ctx, bus.Envelope{ID: "re-" + m.ID, Payload: append([]byte("re:"), m.Payload...)})
	})

	ctx, cancel := context.WithTimeout(t.Context(), Timeout)
	defer cancel()

	r, err := req.Request(ctx, bus.Envelope{ID: "q1", Event: "ask", PartitionKey: "k", Payload: []byte("hi")})
	if err != nil || string(r.Payload) != "re:hi" {
		t.Errorf("reply %q, %v", r.Payload, err)
	}
}
