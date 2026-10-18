package ee_test

import (
	"context"
	"errors"
	"iter"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/mem"
	"github.com/clubpay/ronykit/ee/store"
)

// faultStore fails exactly the operations its hooks decide to fail. The hooks
// own their once-only bookkeeping, so tests control the fault schedule.
type faultStore struct {
	*mem.Store
	txOp  func(op, kind, id string) error
	claim func() error
	ack   func() error
}

func newFaultStore(hooks func(st *faultStore)) *faultStore {
	st := &faultStore{Store: mem.NewStore()}
	if hooks != nil {
		hooks(st)
	}

	return st
}

func (s *faultStore) Tx(ctx context.Context, fn func(ctx context.Context, tx store.Tx) error) error {
	return s.Store.Tx(ctx, func(ctx context.Context, tx store.Tx) error {
		return fn(ctx, &faultTx{tx: tx, s: s})
	})
}

func (s *faultStore) Outbox() store.Outbox { return &faultOutbox{ob: s.Store.Outbox(), s: s} }

type faultTx struct {
	tx store.Tx
	s  *faultStore
}

func (t *faultTx) fault(op, kind, id string) error {
	if t.s.txOp == nil {
		return nil
	}

	return t.s.txOp(op, kind, id)
}

func (t *faultTx) Get(ctx context.Context, kind, id string) (store.Record, error) {
	if err := t.fault("get", kind, id); err != nil {
		return store.Record{}, err
	}

	return t.tx.Get(ctx, kind, id)
}

func (t *faultTx) Insert(ctx context.Context, rec store.Record) error {
	if err := t.fault("insert", rec.Kind, rec.ID); err != nil {
		return err
	}

	return t.tx.Insert(ctx, rec)
}

func (t *faultTx) Update(ctx context.Context, rec store.Record) error {
	if err := t.fault("update", rec.Kind, rec.ID); err != nil {
		return err
	}

	return t.tx.Update(ctx, rec)
}

func (t *faultTx) Delete(ctx context.Context, kind, id string, version uint64) error {
	if err := t.fault("delete", kind, id); err != nil {
		return err
	}

	return t.tx.Delete(ctx, kind, id, version)
}

func (t *faultTx) Find(ctx context.Context, q store.Query) iter.Seq2[store.Record, error] {
	return t.tx.Find(ctx, q)
}

func (t *faultTx) AppendOutbox(ctx context.Context, entries ...store.OutboxEntry) error {
	var id string
	if len(entries) > 0 {
		id = entries[0].ID
	}

	if err := t.fault("outbox", "", id); err != nil {
		return err
	}

	return t.tx.AppendOutbox(ctx, entries...)
}

type faultOutbox struct {
	ob store.Outbox
	s  *faultStore
}

func (o *faultOutbox) Claim(ctx context.Context, limit int, leaseFor time.Duration) ([]store.OutboxEntry, error) {
	if o.s.claim != nil {
		if err := o.s.claim(); err != nil {
			return nil, err
		}
	}

	return o.ob.Claim(ctx, limit, leaseFor)
}

func (o *faultOutbox) Ack(ctx context.Context, ids ...string) error {
	if o.s.ack != nil {
		if err := o.s.ack(); err != nil {
			return err
		}
	}

	return o.ob.Ack(ctx, ids...)
}

// faultBus fails the calls its hooks decide to fail. Hook fields are guarded,
// since tests may arm or clear them while the bus runs.
type faultBus struct {
	*mem.Bus

	mu     sync.Mutex
	subErr func(group string) error
	pubErr func(n int) error
}

func newFaultBus(hooks func(b *faultBus)) *faultBus {
	b := &faultBus{Bus: mem.NewBus(mem.RetryBackoff(time.Millisecond), mem.MaxAttempts(4))}
	if hooks != nil {
		hooks(b)
	}

	return b
}

// setSubErr arms or clears the Subscribe fault.
func (b *faultBus) setSubErr(f func(group string) error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subErr = f
}

// setPubErr arms or clears the Publish fault.
func (b *faultBus) setPubErr(f func(n int) error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.pubErr = f
}

func (b *faultBus) Subscribe(
	ctx context.Context,
	group string,
	events []string,
	h bus.Handler,
) (bus.Subscription, error) {
	b.mu.Lock()
	hook := b.subErr
	b.mu.Unlock()

	if hook != nil {
		if err := hook(group); err != nil {
			return nil, err
		}
	}

	return b.Bus.Subscribe(ctx, group, events, h)
}

func (b *faultBus) Publish(ctx context.Context, envs ...bus.Envelope) error {
	b.mu.Lock()
	hook := b.pubErr
	b.mu.Unlock()

	if hook != nil {
		if err := hook(len(envs)); err != nil {
			return err
		}
	}

	return b.Bus.Publish(ctx, envs...)
}

// plainNoted is an event with neither Key nor Labels.
type plainNoted struct {
	Text string
}

func (plainNoted) EventName() string { return "plain.noted" }

// badEvent cannot be encoded by any codec: its Fn field is a func.
type badEvent struct {
	OrderID string
	Fn      func()
}

func (badEvent) EventName() string { return "bad.evt" }

// newFaultFixture is newFixture for a caller-supplied store and bus.
func newFaultFixture(t *testing.T, st store.Store, bs bus.Bus, opts ...ee.Option) *fixture {
	t.Helper()

	f := &fixture{}
	if fs, ok := st.(*faultStore); ok {
		f.store = fs.Store
	} else if ms, ok := st.(*mem.Store); ok {
		f.store = ms
	}

	if mb, ok := bs.(*mem.Bus); ok {
		f.bus = mb
	}

	if fb, ok := bs.(*faultBus); ok {
		f.bus = fb.Bus
	}

	f.env = ee.MustNew(append([]ee.Option{
		ee.WithStore(st), ee.WithBus(bs), ee.WithOutboxPoll(5 * time.Millisecond),
	}, opts...)...)
	f.orders = must(ee.NewEntityController[Order](f.env))
	f.receipts = must(ee.NewEntityController[Receipt](f.env))
	f.items = must(ee.NewEntityController[Item](f.env))
	f.payments = must(ee.NewEntityController[Payment](f.env))
	f.aggs = must(ee.NewAggregateController[OrderAgg](f.env))

	t.Cleanup(func() {
		_ = f.env.Close()
		_ = f.bus.Close()
	})

	return f
}

// faultSet queues one-shot faults, keyed by "op/kind/id".
type faultSet struct {
	mu     sync.Mutex
	faults map[string]bool
}

func newFaultSet() *faultSet {
	return &faultSet{faults: map[string]bool{}}
}

func (f *faultSet) arm(op, kind, id string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.faults[op+"/"+kind+"/"+id] = true
}

// armAny arms a fault for any id of op/kind.
func (f *faultSet) armAny(op, kind string) {
	f.arm(op, kind, "")
}

func (f *faultSet) clear() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.faults = map[string]bool{}
}

// hook returns the txOp the faultStore consults before each operation. An
// armed fault with an empty id matches every id of op/kind.
func (f *faultSet) hook() func(op, kind, id string) error {
	return func(op, kind, id string) error {
		f.mu.Lock()
		defer f.mu.Unlock()

		if f.faults[op+"/"+kind+"/"+id] || f.faults[op+"/"+kind+"/"] {
			delete(f.faults, op+"/"+kind+"/"+id)
			delete(f.faults, op+"/"+kind+"/")

			return store.ErrConflict
		}

		return nil
	}
}

// ---- poison codec ---------------------------------------------------------

// poisonCodec is JSONCodec plus one value it can decode but will never
// re-encode: record data that decodes must still fail the snapshot re-encode,
// modelling codecs that reject values they did not produce.
type poisonCodec struct {
	codec ee.Codec
}

var errPoison = errors.New("poison codec refused to encode evil data")

func newPoisonCodec() poisonCodec { return poisonCodec{codec: ee.JSONCodec} }

func (c poisonCodec) Marshal(v any) ([]byte, error) {
	if p, ok := v.(*poisonOrder); ok && p != nil && p.Evil != nil {
		return nil, errPoison
	}

	return c.codec.Marshal(v)
}

func (c poisonCodec) Unmarshal(data []byte, v any) error { return c.codec.Unmarshal(data, v) }

func (c poisonCodec) FieldName(f reflect.StructField) (string, bool) {
	return c.codec.FieldName(f)
}

// newPoisonFixture is a fixture whose codec refuses Evil-bearing poison
// states. It returns the fixture and the poison controller.
func newPoisonFixture(t *testing.T, opts ...ee.Option) (*fixture, *ee.EntityController[poisonOrder]) {
	t.Helper()

	f := newFaultFixture(t, mem.NewStore(), mem.NewBus(
		mem.RetryBackoff(time.Millisecond), mem.MaxAttempts(3),
	), append([]ee.Option{ee.WithCodec(newPoisonCodec())}, opts...)...)

	c, err := ee.NewEntityController[poisonOrder](f.env)
	if err != nil {
		t.Fatal(err)
	}

	return f, c
}
