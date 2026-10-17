package mem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/store"
)

func TestStoreCloseLenAndFindCorrupt(t *testing.T) {
	s := NewStore()
	ctx := t.Context()

	mustTx := func(fn func(ctx context.Context, tx store.Tx) error) {
		t.Helper()

		if err := s.Tx(ctx, fn); err != nil {
			t.Fatal(err)
		}
	}

	mustTx(func(ctx context.Context, tx store.Tx) error {
		for i := range 2 {
			if err := tx.Insert(ctx, store.Record{Kind: "k", ID: fmt.Sprint(i), Version: 1, Data: []byte(`{}`)}); err != nil {
				return err
			}
		}

		return tx.Insert(ctx, store.Record{Kind: "kind-nope", ID: "x", Version: 1, Data: []byte(`{}`)})
	})

	if n := s.Len("k"); n != 2 {
		t.Errorf("Len(k) = %d, want 2", n)
	}
	if n := s.Len("kind-nope"); n != 1 {
		t.Errorf("Len(kind-nope) = %d, want 1", n)
	}
	if n := s.Len("missing"); n != 0 {
		t.Errorf("Len(missing) = %d, want 0", n)
	}

	// A record whose data is not an object fails the Find decode and yields an
	// error through the sequence.
	mustTx(func(ctx context.Context, tx store.Tx) error {
		return tx.Insert(ctx, store.Record{Kind: "bad", ID: "1", Version: 1, Data: []byte("[1,2]")})
	})

	err := s.View(ctx, func(ctx context.Context, tx store.Tx) error {
		for _, err := range tx.Find(ctx, store.Query{Kind: "bad"}) {
			if err == nil {
				t.Error("Find delivered a value for an undecodable record")

				continue
			}

			return nil
		}

		return fmt.Errorf("Find must fail on an undecodable record")
	})
	if err != nil {
		t.Error(err)
	}

	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestViewRejectsEveryWrite(t *testing.T) {
	s := NewStore()
	ctx := t.Context()

	err := s.View(ctx, func(ctx context.Context, tx store.Tx) error {
		if err := tx.Update(ctx, store.Record{Kind: "k", ID: "1", Version: 1}); !errors.Is(err, store.ErrReadOnly) {
			t.Errorf("update in View: %v", err)
		}
		if err := tx.Delete(ctx, "k", "1", 1); !errors.Is(err, store.ErrReadOnly) {
			t.Errorf("delete in View: %v", err)
		}
		if err := tx.AppendOutbox(ctx, store.OutboxEntry{ID: "e"}); !errors.Is(err, store.ErrReadOnly) {
			t.Errorf("append in View: %v", err)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestClaimEdgeCases(t *testing.T) {
	s := NewStore()
	ctx := t.Context()

	appendOutboxEntries := func(ids ...string) {
		t.Helper()

		var entries []store.OutboxEntry
		for _, id := range ids {
			entries = append(entries, store.OutboxEntry{ID: id})
		}

		if err := s.Tx(ctx, func(ctx context.Context, tx store.Tx) error { return tx.AppendOutbox(ctx, entries...) }); err != nil {
			t.Fatal(err)
		}
	}

	// A zero limit claims nothing.
	if got, _ := s.Outbox().Claim(ctx, 0, time.Minute); len(got) != 0 {
		t.Errorf("Claim(0) = %v", got)
	}

	appendOutboxEntries("a", "b")

	if got, _ := s.Outbox().Claim(ctx, 10, time.Minute); len(got) != 2 {
		t.Fatalf("Claim(10) = %v", got)
	}

	// Acking unknown ids is a no-op.
	if err := s.Outbox().Ack(ctx, "zzz"); err != nil {
		t.Errorf("Ack(unknown): %v", err)
	}
}

func TestBusClosedOperations(t *testing.T) {
	b := NewBus()
	ctx := t.Context()

	if err := b.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := b.Subscribe(ctx, "g", []string{"e"}, func(context.Context, bus.Message) error { return nil }); !errors.Is(err, errClosed) {
		t.Errorf("subscribe on closed bus: %v", err)
	}

	if err := b.Publish(ctx, bus.Envelope{ID: "1", Event: "e"}); !errors.Is(err, errClosed) {
		t.Errorf("publish on closed bus: %v", err)
	}

	if _, err := b.Request(ctx, bus.Envelope{ID: "2", Event: "e"}); !errors.Is(err, errClosed) {
		t.Errorf("request on closed bus: %v", err)
	}

	// Publishing nothing to a closed bus is still fine.
	if err := b.Close(); err != nil {
		t.Errorf("double close: %v", err)
	}
}

func TestBusLanesClamp(t *testing.T) {
	// Lanes(0) used to divide by zero while routing.
	b := NewBus(Lanes(0))
	defer b.Close()

	ctx := t.Context()

	if _, err := b.Subscribe(ctx, "g", []string{"e"}, func(_ context.Context, m bus.Message) error {
		if m.ID != "1" {
			t.Errorf("deliver = %s", m.ID)
		}

		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := b.Publish(ctx, bus.Envelope{ID: "1", Event: "e"}); err != nil {
		t.Fatal(err)
	}
}

func TestBusReplyTwiceKeepsFirst(t *testing.T) {
	b := NewBus(MaxAttempts(2), RetryBackoff(time.Millisecond))
	defer b.Close()

	ctx := t.Context()

	if _, err := b.Subscribe(ctx, "g", []string{"ask"}, func(c context.Context, m bus.Message) error {
		r := bus.Envelope{Payload: []byte("ok")}
		if err := m.Reply(c, r); err != nil {
			return err
		}
		// Already replied (a redelivery would race the first answer).
		if err := m.Reply(c, r); err != nil {
			t.Errorf("second reply: %v", err)
		}

		return errors.New("still fail")
	}); err != nil {
		t.Fatal(err)
	}

	r, err := b.Request(ctx, bus.Envelope{ID: "1", Event: "ask", PartitionKey: "k"})
	if err != nil || string(r.Payload) != "ok" {
		t.Errorf("reply %q, %v", r.Payload, err)
	}
}

func TestBusRequestCanceledContext(t *testing.T) {
	b := NewBus(MaxAttempts(1))
	defer b.Close()

	ctx := t.Context()

	if _, err := b.Subscribe(ctx, "g", []string{"silent"}, func(context.Context, bus.Message) error {
		return errors.New("no reply")
	}); err != nil {
		t.Fatal(err)
	}

	deadline, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := b.Request(deadline, bus.Envelope{ID: "1", Event: "silent"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("want DeadlineExceeded, got %v", err)
	}
}

func TestBusCloseDuringBackoff(t *testing.T) {
	b := NewBus(MaxAttempts(0), RetryBackoff(time.Second))
	ctx := t.Context()

	delivered := make(chan struct{}, 1)

	if _, err := b.Subscribe(ctx, "g", []string{"e"}, func(context.Context, bus.Message) error {
		delivered <- struct{}{}

		return errors.New("always fail")
	}); err != nil {
		t.Fatal(err)
	}

	if err := b.Publish(ctx, bus.Envelope{ID: "1", Event: "e"}); err != nil {
		t.Fatal(err)
	}

	<-delivered // first attempt failed; the worker is backing off

	if err := b.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case <-delivered:
		t.Error("a delivery happened after Close")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestBusSubscribersLeave(t *testing.T) {
	b := NewBus(RetryBackoff(time.Millisecond))
	defer b.Close()

	ctx := t.Context()

	stay := make(chan string, 4)

	// The first member leaves; deliveries continue through the second.
	left, err := b.Subscribe(ctx, "g", []string{"e"}, func(context.Context, bus.Message) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Subscribe(ctx, "g", []string{"e"}, func(_ context.Context, m bus.Message) error {
		stay <- m.ID

		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := left.Close(); err != nil {
		t.Fatal(err)
	}

	if err := b.Publish(ctx, bus.Envelope{ID: "1", Event: "e"}); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-stay:
		if got != "1" {
			t.Errorf("delivery = %s", got)
		}
	case <-time.After(time.Second):
		t.Error("no delivery after the first member left")
	}

	// Once every member is gone, queued envelopes are dropped silently.
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestMatchRejectsIncomparable seeds one record per JSON type and runs
// conditions whose value type cannot compare with the document value.
func TestMatchRejectsIncomparable(t *testing.T) {
	s := NewStore()
	ctx := t.Context()

	byID := map[string]any{}
	if err := s.Tx(ctx, func(ctx context.Context, tx store.Tx) error {
		insert := func(id, data string) error {
			return tx.Insert(ctx, store.Record{Kind: "m", ID: id, Version: 1, Data: []byte(data)})
		}
		for id, data := range map[string]string{
			"num":     `{"x":123}`,
			"str":     `{"x":"abc"}`,
			"bool":    `{"x":true}`,
			"float":   `{"x":2.5}`,
			"missing": `{}`,
			"time":    `{"x":"2026-01-01T00:00:00Z"}`,
			"badtime": `{"x":"not-a-time"}`,
		} {
			if err := insert(id, data); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatal(err)
	}
	_ = byID

	run := func(name string, q store.Query) []string {
		t.Helper()

		var got []string

		err := s.View(ctx, func(ctx context.Context, tx store.Tx) error {
			for r, err := range tx.Find(ctx, q) {
				if err != nil {
					return err
				}

				got = append(got, r.ID)
			}

			return nil
		})
		if err != nil {
			t.Errorf("%s: %v", name, err)
		}

		return got
	}

	// A time condition matches only the RFC 3339 document.
	if got := run("eq time", store.Query{Kind: "m", Where: []store.Cond{{Field: "x", Op: store.Eq, Value: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}}}); !equal(got, []string{"time"}) {
		t.Errorf("eq time: %v", got)
	}

	// String/bool/numeric conditions only match their own document type.
	if got := run("eq string", store.Query{Kind: "m", Where: []store.Cond{{Field: "x", Op: store.Eq, Value: "abc"}}}); !equal(got, []string{"str"}) {
		t.Errorf("eq string: %v", got)
	}
	if got := run("eq bool", store.Query{Kind: "m", Where: []store.Cond{{Field: "x", Op: store.Eq, Value: true}}}); !equal(got, []string{"bool"}) {
		t.Errorf("eq bool: %v", got)
	}

	// A fractional document compares numerically against an int condition.
	if got := run("gt int against float", store.Query{Kind: "m", Where: []store.Cond{{Field: "x", Op: store.Gt, Value: 2}}}); !equal(got, []string{"float", "num"}) {
		t.Errorf("gt int: %v", got)
	}

	// A fractional document also compares against a uint condition, and
	// non-numbers are never matches.
	if got := run("lte uint", store.Query{Kind: "m", Where: []store.Cond{{Field: "x", Op: store.Lte, Value: uint64(3)}}}); !equal(got, []string{"float"}) {
		t.Errorf("lte uint: %v", got)
	}

	// Float conditions only match numeric documents.
	if got := run("eq float", store.Query{Kind: "m", Where: []store.Cond{{Field: "x", Op: store.Eq, Value: 2.5}}}); !equal(got, []string{"float"}) {
		t.Errorf("eq float: %v", got)
	}

	// A condition value of an unsupported kind never matches.
	if got := run("slice value", store.Query{Kind: "m", Where: []store.Cond{{Field: "x", Op: store.Eq, Value: []any{"a"}}}}); len(got) != 0 {
		t.Errorf("slice value: %v", got)
	}

	// An unknown operator matches nothing.
	if got := run("unknown op", store.Query{Kind: "m", Where: []store.Cond{{Field: "x", Op: store.Op("zzz"), Value: "abc"}}}); len(got) != 0 {
		t.Errorf("unknown op: %v", got)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// TestOrderByTyped pins the deterministic orderings for a kind whose values
// are of one type: numbers and bools order by value, a missing document value
// sorts first, and ties fall back to the record id.
func TestOrderByTyped(t *testing.T) {
	s := NewStore()
	ctx := t.Context()

	if err := s.Tx(ctx, func(ctx context.Context, tx store.Tx) error {
		ins := func(id, data string) error {
			return tx.Insert(ctx, store.Record{Kind: "o", ID: id, Version: 1, Data: []byte(data)})
		}
		for id, data := range map[string]string{
			"low":    `{"n":1,"b":false}`,
			"high":   `{"n":9,"b":true}`,
			"mid":    `{"n":5}`,
			"absent": `{}`,
		} {
			if err := ins(id, data); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		t.Fatal(err)
	}

	run := func(q store.Query) []string {
		t.Helper()

		var got []string

		if err := s.View(ctx, func(ctx context.Context, tx store.Tx) error {
			for r, err := range tx.Find(ctx, q) {
				if err != nil {
					return err
				}

				got = append(got, r.ID)
			}

			return nil
		}); err != nil {
			t.Fatal(err)
		}

		return got
	}

	// Numbers order by value; the record without the field stays first.
	if got := run(store.Query{Kind: "o", OrderBy: []store.Order{{Field: "n"}}}); !equal(got, []string{"absent", "low", "mid", "high"}) {
		t.Errorf("order n: %v", got)
	}

	// Descending numbers.
	if got := run(store.Query{Kind: "o", OrderBy: []store.Order{{Field: "n", Desc: true}}}); !equal(got, []string{"high", "mid", "low", "absent"}) {
		t.Errorf("order n desc: %v", got)
	}

	// Bools compare false < true; the records without b tie and sort first by
	// id.
	if got := run(store.Query{Kind: "o", OrderBy: []store.Order{{Field: "b"}}}); !equal(got, []string{"absent", "mid", "low", "high"}) {
		t.Errorf("order b: %v", got)
	}
	if got := run(store.Query{Kind: "o", OrderBy: []store.Order{{Field: "b", Desc: true}}}); !equal(got, []string{"high", "low", "absent", "mid"}) {
		t.Errorf("order b desc: %v", got)
	}
}

// TestTypedDecoderMatches runs Find against a custom Decoder that hands the
// store native Go values instead of json.Number strings.
func TestTypedDecoderMatches(t *testing.T) {
	dec := func(data []byte) (map[string]any, error) {
		doc, err := JSONDecoder(data)
		if err != nil {
			return nil, err
		}

		if n, ok := doc["n"].(json.Number); ok {
			if i, err := n.Int64(); err == nil {
				doc["n"] = int(i)
			}
		}
		if s, ok := doc["at"].(string); ok {
			if ts, err := time.Parse(time.RFC3339Nano, s); err == nil {
				doc["at"] = ts
			}
		}

		return doc, nil
	}

	s := NewStore(WithDecoder(dec), WithStoreClock(func() time.Time { return time.Unix(1, 0) }))
	ctx := t.Context()

	if err := s.Tx(ctx, func(ctx context.Context, tx store.Tx) error {
		return tx.Insert(ctx, store.Record{
			Kind: "r", ID: "1", Version: 1,
			Data: []byte(`{"n":3,"at":"2026-06-01T10:00:00Z"}`),
		})
	}); err != nil {
		t.Fatal(err)
	}

	err := s.View(ctx, func(ctx context.Context, tx store.Tx) error {
		if err := tx.Insert(ctx, store.Record{Kind: "r", ID: "2", Version: 1}); !errors.Is(err, store.ErrReadOnly) {
			t.Error("a typed store still rejects View writes")
		}

		n := 0

		for _, err := range tx.Find(ctx, store.Query{
			Kind:    "r",
			Where:   []store.Cond{{Field: "n", Op: store.Eq, Value: 3}},
			OrderBy: []store.Order{{Field: "at"}},
		}) {
			if err != nil {
				return err
			}

			n++
		}

		if n != 1 {
			t.Errorf("typed decoder matches: %d", n)
		}

		// The missing at field never equals a time value.
		n = 0

		for _, err := range tx.Find(ctx, store.Query{
			Kind:  "r",
			Where: []store.Cond{{Field: "at", Op: store.Eq, Value: time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)}, {Field: "x", Op: store.Eq, Value: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}},
		}) {
			if err != nil {
				return err
			}

			n++
		}

		if n != 0 {
			t.Errorf("time against a missing field matched %d rows", n)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestUpdateConflictAtCommit isolates the commit-time version check: two
// transactions observe version 1, buffer the same write, and the loser fails
// at commit, not at write time.
func TestUpdateConflictAtCommit(t *testing.T) {
	s := NewStore()
	ctx := t.Context()

	mustTx := func(fn func(ctx context.Context, tx store.Tx) error) {
		t.Helper()

		if err := s.Tx(ctx, fn); err != nil {
			t.Fatal(err)
		}
	}

	mustTx(func(ctx context.Context, tx store.Tx) error {
		return tx.Insert(ctx, store.Record{Kind: "k", ID: "1", Version: 1, Data: []byte(`{"by":-1}`)})
	})

	var (
		raced  = [2]chan struct{}{make(chan struct{}, 1), make(chan struct{}, 1)}
		raced2 = [2]chan struct{}{make(chan struct{}, 1), make(chan struct{}, 1)}
		errs   = [2]error{}
		wg     sync.WaitGroup
	)

	for i := range 2 {
		wg.Go(func() {
			errs[i] = s.Tx(ctx, func(ctx context.Context, tx store.Tx) error {
				if _, err := tx.Get(ctx, "k", "1"); err != nil {
					return err
				}

				raced[i] <- struct{}{}
				<-raced[1-i] // both have observed version 1

				if err := tx.Update(ctx, store.Record{Kind: "k", ID: "1", Version: 1, Data: []byte(fmt.Sprintf(`{"by":%d}`, i))}); err != nil {
					return err
				}

				raced2[i] <- struct{}{}
				<-raced2[1-i] // both have buffered their write before any commit

				return nil
			})
		})
	}
	wg.Wait()

	joined := errors.Join(errs[0], errs[1])
	if (errs[0] == nil) == (errs[1] == nil) || !errors.Is(joined, store.ErrConflict) {
		t.Errorf("want exactly one commit conflict, got %v", errs)
	}

	var v uint64

	if err := s.View(ctx, func(ctx context.Context, tx store.Tx) error {
		r, err := tx.Get(ctx, "k", "1")
		v = r.Version

		return err
	}); err != nil {
		t.Fatal(err)
	}
	if v != 2 {
		t.Errorf("version = %d, want 2", v)
	}
}

func TestCompareBool(t *testing.T) {
	cases := []struct {
		a, b bool
		want int
	}{
		{true, true, 0},
		{false, false, 0},
		{false, true, -1},
		{true, false, 1},
	}

	for _, c := range cases {
		if got := compareBool(c.a, c.b); got != c.want {
			t.Errorf("compareBool(%v, %v) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
