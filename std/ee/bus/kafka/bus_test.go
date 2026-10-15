package kafka_test

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
	eekafka "github.com/clubpay/ronykit/std/ee/bus/kafka"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

// EE_KAFKA_TEST_BROKERS enables the tests, e.g. localhost:59092.
const envBrokers = "EE_KAFKA_TEST_BROKERS"

// newBus returns a bus on its own topic prefix (and so its own groups); the
// topic is deleted after the test.
func newBus(t *testing.T, opts ...eekafka.Option) *eekafka.Bus {
	t.Helper()

	brokers := os.Getenv(envBrokers)
	if brokers == "" {
		t.Skip(envBrokers + " not set")
	}

	prefix := "t" + strings.ToLower(rand.Text()[:10])
	b, err := eekafka.New(append([]eekafka.Option{
		eekafka.WithBrokers(strings.Split(brokers, ",")...),
		eekafka.WithPrefix(prefix),
		eekafka.WithTopicDefaults(4, 1),
		eekafka.RetryBackoff(10 * time.Millisecond),
	}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cl, err := kgo.NewClient(kgo.SeedBrokers(strings.Split(brokers, ",")...))
		if err != nil {
			return
		}
		defer cl.Close()

		_, _ = kadm.NewClient(cl).DeleteTopics(context.Background(), prefix+".events")
	})

	return b
}

func TestConformance(t *testing.T) {
	bustest.Run(t, func(t *testing.T) bus.Bus { return newBus(t) })
}

func TestNotARequester(t *testing.T) {
	var b bus.Bus = newBus(t)
	t.Cleanup(func() { _ = b.Close() })

	if _, ok := b.(bus.Requester); ok {
		t.Error("kafka bus must not implement bus.Requester")
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()

	for deadline := time.Now().Add(bustest.Timeout); time.Now().Before(deadline); {
		if cond() {
			return
		}

		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", what)
}

func TestDeadLetter(t *testing.T) {
	dead := make(chan bus.Envelope, 1)
	b := newBus(t, eekafka.MaxAttempts(3), eekafka.OnDeadLetter(func(e bus.Envelope, _ error) { dead <- e }))
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
	case <-time.After(bustest.Timeout):
		t.Fatal("no dead letter")
	}

	waitFor(t, "the next record", func() bool { return after.Load() == 1 })

	if n := attempts.Load(); n != 3 {
		t.Errorf("attempts = %d, want 3", n)
	}
}

// TestResumeAfterRestart: a member that stops and a new one in the same group
// continue from committed offsets, without replaying handled records.
func TestResumeAfterRestart(t *testing.T) {
	b := newBus(t)
	t.Cleanup(func() { _ = b.Close() })

	var seen atomic.Int32

	h := func(context.Context, bus.Message) error {
		seen.Add(1)

		return nil
	}

	s, _ := b.Subscribe(t.Context(), "g", []string{"e"}, h)
	_ = b.Publish(t.Context(), bus.Envelope{ID: "1", Event: "e", PartitionKey: "k"})
	waitFor(t, "first", func() bool { return seen.Load() == 1 })
	_ = s.Close()

	_ = b.Publish(t.Context(), bus.Envelope{ID: "2", Event: "e", PartitionKey: "k"})
	_, _ = b.Subscribe(t.Context(), "g", []string{"e"}, h)
	waitFor(t, "second", func() bool { return seen.Load() >= 2 })
	time.Sleep(time.Second)

	if n := seen.Load(); n != 2 {
		t.Errorf("handled %d records, want 2 (no replay)", n)
	}
}

type Order struct {
	ID     string `ee:"id"`
	Status string
}

func (Order) StateName() string { return "order" }

type OrderPaid struct{ OrderID string }

func (OrderPaid) EventName() string { return "order.paid" }
func (e OrderPaid) Key() string     { return e.OrderID }

func TestWithEE(t *testing.T) {
	b := newBus(t)
	t.Cleanup(func() { _ = b.Close() })

	env := ee.MustNew(ee.WithBus(b), ee.WithStore(mem.NewStore()))
	orders, _ := ee.NewStateController[Order](env)
	_ = orders.Action(func(ctx *ee.ActionCtx[Order, OrderPaid]) error {
		o, err := ctx.State()
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
