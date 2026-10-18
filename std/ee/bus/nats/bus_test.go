package nats_test

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/bus/bustest"
	"github.com/clubpay/ronykit/ee/mem"
	eenats "github.com/clubpay/ronykit/std/ee/bus/nats"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// EE_NATS_TEST_URL enables the tests, e.g. nats://localhost:54222 against
// `docker run -p 54222:4222 nats:2.11-alpine -js`.
const envURL = "EE_NATS_TEST_URL"

// newBus returns a bus on its own prefix and stream, deleted after the test.
func newBus(t *testing.T, opts ...eenats.Option) *eenats.Bus {
	t.Helper()

	url := os.Getenv(envURL)
	if url == "" {
		t.Skip(envURL + " not set")
	}

	prefix := "t" + strings.ToLower(rand.Text()[:10])
	b, err := eenats.New(append([]eenats.Option{
		eenats.WithURL(url),
		eenats.WithPrefix(prefix),
		eenats.WithPartitions(4),
		eenats.RetryBackoff(10 * time.Millisecond),
	}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		nc, err := nats.Connect(url)
		if err != nil {
			return
		}
		defer nc.Close()

		js, _ := jetstream.New(nc)
		_ = js.DeleteStream(t.Context(), b.Stream())
	})

	return b
}

func TestConformance(t *testing.T) {
	bustest.Run(t, func(t *testing.T) bus.Bus { return newBus(t) })
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()

	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); {
		if cond() {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", what)
}

func TestDeadLetter(t *testing.T) {
	dead := make(chan bus.Envelope, 1)
	b := newBus(t, eenats.MaxAttempts(3), eenats.OnDeadLetter(func(e bus.Envelope, _ error) { dead <- e }))
	t.Cleanup(func() { _ = b.Close() })

	var attempts, after atomic.Int32

	_, _ = b.Subscribe(t.Context(), "g", []string{"e"}, func(_ context.Context, m bus.Message) error {
		if m.ID == "poison" {
			attempts.Add(1)

			return errors.New("never")
		}

		after.Add(1)

		return nil
	})

	_ = b.Publish(t.Context(),
		bus.Envelope{ID: "poison", Event: "e", PartitionKey: "k"},
		bus.Envelope{ID: "next", Event: "e", PartitionKey: "k"})

	select {
	case e := <-dead:
		if e.ID != "poison" {
			t.Errorf("dead letter %q", e.ID)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("no dead letter")
	}

	// The partition must move on once the poison message is terminated.
	waitFor(t, "the next message", func() bool { return after.Load() == 1 })

	if n := attempts.Load(); n != 3 {
		t.Errorf("attempts = %d, want 3", n)
	}
}

func TestDuplicateEnvelopeIsDropped(t *testing.T) {
	b := newBus(t)
	t.Cleanup(func() { _ = b.Close() })

	var n atomic.Int32

	_, _ = b.Subscribe(t.Context(), "g", []string{"e"}, func(context.Context, bus.Message) error {
		n.Add(1)

		return nil
	})

	// The outbox relay republishes after a crash: same envelope ID.
	e := bus.Envelope{ID: "same", Event: "e", PartitionKey: "k"}
	_ = b.Publish(t.Context(), e)
	_ = b.Publish(t.Context(), e)

	waitFor(t, "delivery", func() bool { return n.Load() >= 1 })
	time.Sleep(500 * time.Millisecond)

	if got := n.Load(); got != 1 {
		t.Errorf("delivered %d times, want 1", got)
	}
}

func TestSlowHandlerIsNotRedelivered(t *testing.T) {
	b := newBus(t, eenats.WithAckWait(time.Second))
	t.Cleanup(func() { _ = b.Close() })

	var n atomic.Int32

	h := func(context.Context, bus.Message) error {
		n.Add(1)
		time.Sleep(2500 * time.Millisecond) // 2.5x AckWait

		return nil
	}

	// Two members: without heartbeats the other one would get a redelivery.
	_, _ = b.Subscribe(t.Context(), "g", []string{"e"}, h)
	_, _ = b.Subscribe(t.Context(), "g", []string{"e"}, h)

	_ = b.Publish(t.Context(), bus.Envelope{ID: "slow", Event: "e", PartitionKey: "k"})

	waitFor(t, "delivery", func() bool { return n.Load() >= 1 })
	time.Sleep(4 * time.Second)

	if got := n.Load(); got != 1 {
		t.Errorf("delivered %d times, want 1", got)
	}
}

type Order struct {
	ID     string `ee:"id"`
	Status string
}

func (Order) EntityName() string { return "order" }

type OrderPaid struct{ OrderID string }

func (OrderPaid) EventName() string { return "order.paid" }
func (e OrderPaid) Key() string     { return e.OrderID }

func TestWithEE(t *testing.T) {
	b := newBus(t)
	t.Cleanup(func() { _ = b.Close() })

	env := ee.MustNew(ee.WithBus(b), ee.WithStore(mem.NewStore()))
	orders, _ := ee.NewEntityController[Order](env)
	_ = orders.Action(func(ctx *ee.ActionCtx[Order, OrderPaid]) error {
		o, err := ctx.Entity()
		if err != nil {
			return err
		}

		o.Status = "paid"

		return ctx.Save(o)
	})

	if err := env.Start(t.Context()); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = env.Close() })

	_ = orders.Create(t.Context(), Order{ID: "o1"})
	_ = env.Send(t.Context(), OrderPaid{OrderID: "o1"})

	waitFor(t, "order paid", func() bool {
		o, _ := orders.Get(t.Context(), "o1")

		return o.Status == "paid"
	})
}
