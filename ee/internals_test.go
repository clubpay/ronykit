package ee

import (
	"context"
	"errors"
	"io"
	"iter"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/store"
)

// State types with odd schemas, checked against the parser directly.

type badTagState struct {
	ID   string `ee:"id"`
	Junk string `ee:"nope"`
}

func (badTagState) StateName() string { return "bad-tag" }

type twoIDs struct {
	A string `ee:"id"`
	B string `ee:"id"`
}

func (twoIDs) StateName() string { return "two-ids" }

type hiddenID struct {
	name string `ee:"id"`
}

func (hiddenID) StateName() string { return "hidden-id" }

type hiddenField struct {
	ID     string `ee:"id"`
	secret []byte
}

func (hiddenField) StateName() string { return "hidden-field" }

type strCode string

type strKindID struct {
	ID strCode `ee:"id"`
}

func (strKindID) StateName() string { return "str-kind-id" }

// Aggregate types with odd schemas.

type aggNonStruct int

func (*aggNonStruct) AggregateName() string       { return "non-struct-agg" }
func (*aggNonStruct) Load(*LoadCtx, string) error { return nil }

type aggEmptyName struct {
	O *order
}

func (aggEmptyName) AggregateName() string        { return "" }
func (*aggEmptyName) Load(*LoadCtx, string) error { return nil }

type aggBadMember struct {
	N int
}

type aggBadPtr struct {
	P *int
}

type aggBadSlice struct {
	S []int
}

type aggBadMapKey struct {
	M map[int]orderItem
}

type aggBadOther struct {
	C chan order
}

type aggBadTag struct {
	O *order `ee:"junk"`
}

type aggRootSlice struct {
	Items []orderItem `ee:"root"`
}

func TestSchemaOddCases(t *testing.T) {
	if _, err := parseStateSchema("", typeOf[order](), JSONCodec); !errors.Is(err, ErrInvalidSchema) {
		t.Errorf("empty StateName: want ErrInvalidSchema, got %v", err)
	}

	// A state implemented on the pointer (or with no State method at all)
	// gives the parser a non-struct type.
	if _, err := parseStateSchema("x", typeOf[*order](), JSONCodec); !errors.Is(err, ErrInvalidSchema) {
		t.Errorf("non-struct: want ErrInvalidSchema, got %v", err)
	}

	if _, err := parseStateSchema("bad-tag", typeOf[badTagState](), JSONCodec); !errors.Is(err, ErrInvalidSchema) {
		t.Errorf("unknown tag: want ErrInvalidSchema, got %v", err)
	}

	if _, err := parseStateSchema("two-ids", typeOf[twoIDs](), JSONCodec); !errors.Is(err, ErrInvalidSchema) {
		t.Errorf("two id fields: want ErrInvalidSchema, got %v", err)
	}

	if _, err := parseStateSchema("hidden-id", typeOf[hiddenID](), JSONCodec); !errors.Is(err, ErrInvalidSchema) {
		t.Errorf("unexported id: want ErrInvalidSchema, got %v", err)
	}

	s, err := parseStateSchema("hidden-field", typeOf[hiddenField](), JSONCodec)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.fields["secret"]; ok {
		t.Error("unexported field must not be queryable")
	}

	// A named string kind satisfies the id check.
	s, err = parseStateSchema("str-kind-id", typeOf[strKindID](), JSONCodec)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.id(valueOf(strKindID{ID: "s1"})); got != "s1" {
		t.Errorf("named-string id = %q", got)
	}
}

func TestAggregateSchemaOddCases(t *testing.T) {
	env := newTestEnv(t)

	cases := []struct {
		name string
		typ  reflect.Type
	}{
		{"int member", typeOf[aggBadMember]()},
		{"pointer to non-struct", typeOf[aggBadPtr]()},
		{"slice of non-struct", typeOf[aggBadSlice]()},
		{"map with non-string key", typeOf[aggBadMapKey]()},
		{"channel member", typeOf[aggBadOther]()},
		{"unknown tag", typeOf[aggBadTag]()},
		{"root on a slice", typeOf[aggRootSlice]()},
	}
	for _, c := range cases {
		if _, err := parseAggregateSchema(env, "x", c.typ); !errors.Is(err, ErrInvalidSchema) {
			t.Errorf("%s: want ErrInvalidSchema, got %v", c.name, err)
		}
	}

	if _, err := NewAggregateController[aggEmptyName](env); !errors.Is(err, ErrInvalidSchema) {
		t.Errorf("empty AggregateName: want ErrInvalidSchema, got %v", err)
	}

	if _, err := NewAggregateController[aggNonStruct](env); !errors.Is(err, ErrInvalidSchema) {
		t.Errorf("non-struct aggregate: want ErrInvalidSchema, got %v", err)
	}
}

type stubController struct{}

func (stubController) subscribedEvents() []string                { return nil }
func (stubController) handle(context.Context, bus.Message) error { return nil }

func TestDuplicateControllerName(t *testing.T) {
	env := MustNew()

	if err := env.addController("state:x", stubController{}); err != nil {
		t.Fatal(err)
	}
	if err := env.addController("state:x", stubController{}); !errors.Is(err, ErrAlreadyRegistered) {
		t.Errorf("want ErrAlreadyRegistered, got %v", err)
	}
}

type dupNameState struct {
	ID string `ee:"id"`
}

func (dupNameState) StateName() string { return "dup-name" }

// A controller name can also be squatted by a non-controller stub; the state
// registration must still fail cleanly.
func TestStateControllerNameCollision(t *testing.T) {
	env := MustNew()

	if err := env.addController("state:dup-name", stubController{}); err != nil {
		t.Fatal(err)
	}

	if _, err := NewStateController[dupNameState](env); !errors.Is(err, ErrAlreadyRegistered) {
		t.Errorf("want ErrAlreadyRegistered, got %v", err)
	}
}

func TestControllerNames(t *testing.T) {
	env := MustNew()

	states, err := NewStateController[order](env)
	if err != nil {
		t.Fatal(err)
	}
	if got := states.Name(); got != "order" {
		t.Errorf("Name() = %q", got)
	}

	if _, err := NewStateController[orderItem](env); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStateController[payment](env); err != nil {
		t.Fatal(err)
	}

	aggs, err := NewAggregateController[orderAgg](env)
	if err != nil {
		t.Fatal(err)
	}
	if got := aggs.Name(); got != "order-agg" {
		t.Errorf("Name() = %q", got)
	}
}

// Send reports the encode failure of the event payload before touching the
// bus.
func TestSendEncodeError(t *testing.T) {
	env := MustNew()

	err := env.Send(context.Background(), badEvent{})
	if err == nil || !strings.Contains(err.Error(), "encode bad.evt") {
		t.Errorf("send of an unmarshalable event: %v", err)
	}
}

func TestDispatchIgnoresUnknownEvent(t *testing.T) {
	env := MustNew()

	// A subscription may carry events a controller has no action for (e.g.
	// after a hot upgrade); dispatch must acknowledge them.
	msg := bus.Message{Envelope: bus.Envelope{Event: "unknown", ID: "x"}}
	if err := env.dispatch(context.Background(), "state:x", map[string]*action{}, nil, msg); err != nil {
		t.Errorf("dispatch of an unknown event must not fail, got %v", err)
	}
}

func TestErrPermanentWrapsItsCause(t *testing.T) {
	cause := io.ErrUnexpectedEOF
	err := errPermanent{cause}

	if err.Error() != cause.Error() {
		t.Errorf("Error() = %q, want %q", err.Error(), cause.Error())
	}
	if !errors.Is(err, cause) {
		t.Error("errors.Is must look through errPermanent")
	}
}

func TestNewOptionValidation(t *testing.T) {
	if _, err := New(WithCodec(nil)); err == nil || !strings.Contains(err.Error(), "codec") {
		t.Errorf("nil codec: got %v", err)
	}

	if _, err := New(WithMaxRetries(-1)); err == nil || !strings.Contains(err.Error(), "max retries") {
		t.Errorf("negative max retries: got %v", err)
	}

	for _, d := range []time.Duration{0, -time.Second} {
		if _, err := New(WithOutboxPoll(d)); err == nil || !strings.Contains(err.Error(), "outbox poll") {
			t.Errorf("outbox poll %v: got %v", d, err)
		}
	}

	func() {
		defer func() {
			if recover() == nil {
				t.Error("MustNew must panic on an invalid option")
			}
		}()
		MustNew(WithMaxRetries(-1))
	}()
}

func TestSchemas(t *testing.T) {
	if got := MustNew().Schemas(); len(got) != 0 {
		t.Errorf("empty env returned %d schemas", len(got))
	}

	env := newTestEnv(t)
	got := env.Schemas()

	names := make([]string, 0, len(got))
	for _, si := range got {
		names = append(names, si.Name)
	}
	slices.Sort(names)

	if !slices.Equal(names, []string{"order", "order-item", "payment"}) {
		t.Errorf("schemas = %v", names)
	}

	for _, si := range got {
		if si.Type.Kind() != reflect.Struct {
			t.Errorf("%s: Type = %v", si.Name, si.Type)
		}
	}
}

func TestSelectorEdge(t *testing.T) {
	l := L{"merchant": "m1"}

	if AnyOf(Match("merchant", "m2")).Match(l) {
		t.Error("AnyOf with no matching selector must be false")
	}

	if MatchIn("merchant").Match(l) {
		t.Error("MatchIn without values must be false")
	}
}

func TestQueryCompileOddities(t *testing.T) {
	env := newTestEnv(t)
	s, _ := env.stateOf(typeOf[order]())

	if _, err := Where("Merchant", nil).compile(s); !errors.Is(err, ErrInvalidQuery) {
		t.Errorf("nil value: want ErrInvalidQuery, got %v", err)
	}

	if _, err := (Query{}).OrderBy("Nope", false).compile(s); !errors.Is(err, ErrInvalidQuery) {
		t.Errorf("unknown order field: want ErrInvalidQuery, got %v", err)
	}

	if _, err := (Query{}).And("Amount", In, []any{uint64(1), "nope"}).compile(s); !errors.Is(err, ErrInvalidQuery) {
		t.Errorf("bad In element: want ErrInvalidQuery, got %v", err)
	}

	// json tag names with options still resolve by their bare name.
	q, err := Where("Merchant", "m1").compile(s)
	if err != nil || q.Where[0].Field != "merchant" {
		t.Errorf("tagged field via Where: %v, %v", q, err)
	}
}

type metrics struct {
	ID    string `ee:"id"`
	Rate  float64
	Level float32
	Size  int64
}

func (metrics) StateName() string { return "metrics" }

type posT struct{ X, Y int }

type weird struct {
	ID  string `ee:"id"`
	Pos struct{ X, Y int }
}

func (weird) StateName() string { return "weird" }

func TestConvertValueOddities(t *testing.T) {
	env := MustNew()

	if _, err := NewStateController[metrics](env); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStateController[weird](env); err != nil {
		t.Fatal(err)
	}

	m, _ := env.stateOf(typeOf[metrics]())

	// Negative ints land in float fields without loss.
	if _, err := Where("Rate", -1).compile(m); err != nil {
		t.Errorf("negative int into float64: %v", err)
	}

	// A float64 value loses precision in a float32 field.
	if _, err := Where("Level", 0.1).compile(m); !errors.Is(err, ErrInvalidQuery) {
		t.Errorf("lossy float32: want ErrInvalidQuery, got %v", err)
	}

	// MaxInt64 fits int64.
	if _, err := Where("Size", int64(math.MaxInt64)).compile(m); err != nil {
		t.Errorf("max int64: %v", err)
	}

	// 1<<63 overflows int64 by wrapping negative.
	if _, err := Where("Size", uint64(1)<<63).compile(m); !errors.Is(err, ErrInvalidQuery) {
		t.Errorf("uint overflow into int64: want ErrInvalidQuery, got %v", err)
	}

	// A named type is assignable to the same unnamed struct type.
	w, _ := env.stateOf(typeOf[weird]())
	if _, err := Where("Pos", posT{X: 1, Y: 2}).compile(w); err != nil {
		t.Errorf("assignable struct value: %v", err)
	}
}

func TestEnvelopeOddities(t *testing.T) {
	env := newTestEnv(t)

	if _, err := env.envelope(badEvent{}, nil); err == nil || !strings.Contains(err.Error(), "encode bad.evt") {
		t.Errorf("unmarshalable event: got %v", err)
	}

	// An expiry in the past (or a negative TTL) expires immediately.
	e, err := env.envelope(orderPaid{}, []SendOption{ExpireAt(env.now().Add(-time.Second))})
	if err != nil || !e.Expired(env.now()) {
		t.Errorf("past ExpireAt: %v, expired=%v", err, e.Expired(env.now()))
	}

	e, err = env.envelope(orderPaid{}, []SendOption{ExpireIn(-time.Second)})
	if err != nil || !e.ExpiresAt.IsZero() || e.Expired(env.now()) {
		t.Errorf("non-positive ExpireIn must be ignored: %v, %v", err, e.ExpiresAt)
	}

	// Repeated WithHeader overwrites.
	e, err = env.envelope(orderNoted{}, []SendOption{WithHeader("k", "v1"), WithHeader("k", "v2")})
	if err != nil || e.Headers["k"] != "v2" || len(e.Headers) != 1 {
		t.Errorf("WithHeader overwrite: %v, %+v", err, e.Headers)
	}

	// A keyed event with an empty key does not claim a partition.
	e, err = env.envelope(orderPaid{}, nil)
	if err != nil || e.PartitionKey != "" {
		t.Errorf("empty key: %v, %q", err, e.PartitionKey)
	}
}

type badEvent struct {
	Fn func()
}

func (badEvent) EventName() string { return "bad.evt" }

// fakeFindTx serves a fixed record set for Get/Delete/Find, to drive the
// scope against a store that does not hide in-transaction deletes from Find.
// Only the overridden methods are called; the embedded interface stays nil.
type fakeFindTx struct {
	store.Tx

	recs []store.Record
}

func (f *fakeFindTx) Get(_ context.Context, kind, id string) (store.Record, error) {
	for _, r := range f.recs {
		if r.Kind == kind && r.ID == id {
			return r, nil
		}
	}

	return store.Record{}, store.ErrNotFound
}

func (*fakeFindTx) Delete(context.Context, string, string, uint64) error { return nil }

func (f *fakeFindTx) Find(_ context.Context, _ store.Query) iter.Seq2[store.Record, error] {
	return func(yield func(store.Record, error) bool) {
		for _, r := range f.recs {
			if !yield(r, nil) {
				return
			}
		}
	}
}

// TestFindSkipsDeletedTracking pins the identity map's guard against backends
// whose Find does not hide writes the unit itself deleted.
func TestFindSkipsDeletedTracking(t *testing.T) {
	env := MustNew()
	if _, err := NewStateController[order](env); err != nil {
		t.Fatal(err)
	}

	schema, err := env.stateOf(typeOf[order]())
	if err != nil {
		t.Fatal(err)
	}

	data, err := JSONCodec.Marshal(order{ID: "o1"})
	if err != nil {
		t.Fatal(err)
	}

	sc := newScope(context.Background(), env, &fakeFindTx{
		recs: []store.Record{{Kind: "order", ID: "o1", Version: 1, Data: data}},
	}, nil)

	v, err := sc.get(schema, "o1")
	if err != nil {
		t.Fatal(err)
	}

	if err := sc.remove(schema, v); err != nil {
		t.Fatal(err)
	}

	n := 0
	for _, err := range sc.find(schema, Query{}) {
		if err != nil {
			t.Fatal(err)
		}

		n++
	}
	if n != 0 {
		t.Errorf("find returned %d states the unit deleted", n)
	}
}
