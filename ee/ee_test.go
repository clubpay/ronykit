package ee

import (
	"errors"
	"testing"
	"time"
)

type order struct {
	ID       string `ee:"id"`
	Merchant string `json:"merchant"`
	Amount   uint64
	Secret   string `json:"-"`
}

func (order) EntityName() string { return "order" }

type orderItem struct {
	ID      string `ee:"id"`
	OrderID string
}

func (orderItem) EntityName() string { return "order-item" }

type payment struct {
	ID      string `ee:"id"`
	OrderID string
}

func (payment) EntityName() string { return "payment" }

type orderPaid struct {
	OrderID  string
	Merchant string
}

func (orderPaid) EventName() string { return "order.paid" }
func (e orderPaid) Key() string     { return e.OrderID }
func (e orderPaid) Labels() L       { return L{"merchant": e.Merchant} }

type orderNoted struct{ Text string }

func (orderNoted) EventName() string { return "order.noted" }

type orderAgg struct {
	Order    *order
	Items    []orderItem
	Payments map[string]*payment
	Cache    map[int]string `ee:"-"`
}

func (orderAgg) AggregateName() string { return "order-agg" }

func (a *orderAgg) Load(ctx *LoadCtx, id string) error {
	o, err := ctx.Entities[order]().Get(id)
	a.Order = o

	return err
}

func newTestEnv(t *testing.T) *Env {
	t.Helper()

	now := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	env := MustNew(WithClock(func() time.Time { return now }))

	if _, err := NewEntityController[order](env); err != nil {
		t.Fatal(err)
	}

	if _, err := NewEntityController[orderItem](env); err != nil {
		t.Fatal(err)
	}

	if _, err := NewEntityController[payment](env); err != nil {
		t.Fatal(err)
	}

	return env
}

type noID struct{ Name string }

func (noID) EntityName() string { return "no-id" }

type intID struct {
	ID int `ee:"id"`
}

func (intID) EntityName() string { return "int-id" }

type unnamed struct {
	ID string `ee:"id"`
}

func (unnamed) EntityName() string { return "" }

type orderClone struct {
	ID string `ee:"id"`
}

func (orderClone) EntityName() string { return "order" }

func TestEntitySchema(t *testing.T) {
	env := MustNew()
	if _, err := NewEntityController[noID](env); !errors.Is(err, ErrInvalidSchema) {
		t.Errorf("no id: want ErrInvalidSchema, got %v", err)
	}

	if _, err := NewEntityController[intID](env); !errors.Is(err, ErrInvalidSchema) {
		t.Errorf("int id: want ErrInvalidSchema, got %v", err)
	}

	if _, err := NewEntityController[unnamed](env); !errors.Is(err, ErrInvalidSchema) {
		t.Errorf("empty name: want ErrInvalidSchema, got %v", err)
	}

	env = newTestEnv(t)
	if _, err := NewEntityController[order](env); !errors.Is(err, ErrAlreadyRegistered) {
		t.Errorf("duplicate type: want ErrAlreadyRegistered, got %v", err)
	}

	if _, err := NewEntityController[orderClone](env); !errors.Is(err, ErrAlreadyRegistered) {
		t.Errorf("duplicate name: want ErrAlreadyRegistered, got %v", err)
	}

	s, _ := env.entityOf(typeOf[order]())
	if got := s.id(valueOf(order{ID: "o1"})); got != "o1" {
		t.Errorf("id = %q", got)
	}

	if s.fields["Merchant"].encoded != "merchant" || s.fields["Amount"].encoded != "Amount" {
		t.Errorf("encoded names = %+v", s.fields)
	}

	if _, ok := s.fields["Secret"]; ok {
		t.Error(`json:"-" field must not be queryable`)
	}
}

func TestEnvelope(t *testing.T) {
	env := newTestEnv(t)

	e, err := env.envelope(orderPaid{OrderID: "o1", Merchant: "m1"}, []SendOption{WithHeader("k", "v"), ExpireIn(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}

	if e.Event != "order.paid" || e.PartitionKey != "o1" || e.Labels["merchant"] != "m1" || e.Headers["k"] != "v" {
		t.Errorf("envelope = %+v", e)
	}

	if want := env.now().Add(time.Minute); !e.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", e.ExpiresAt, want)
	}

	if e.Expired(env.now()) || !e.Expired(env.now().Add(time.Minute)) {
		t.Error("Expired boundaries are wrong")
	}

	e, _ = env.envelope(orderNoted{Text: "x"}, nil)
	if e.PartitionKey != "" || e.Labels != nil || !e.ExpiresAt.IsZero() {
		t.Errorf("optional fields must be empty: %+v", e)
	}
}

func TestSelector(t *testing.T) {
	l := L{"merchant": "m1", "region": "eu"}
	cases := map[string]struct {
		sel  Selector
		want bool
	}{
		"match":     {Match("merchant", "m1"), true},
		"mismatch":  {Match("merchant", "m2"), false},
		"in":        {MatchIn("region", "us", "eu"), true},
		"has":       {HasLabel("region"), true},
		"missing":   {HasLabel("tenant"), false},
		"all":       {AllOf(Match("merchant", "m1"), HasLabel("tenant")), false},
		"any":       {AnyOf(Match("merchant", "m2"), HasLabel("region")), true},
		"not":       {Not(HasLabel("tenant")), true},
		"empty-all": {AllOf(), true},
	}

	for name, c := range cases {
		if got := c.sel.Match(l); got != c.want {
			t.Errorf("%s: got %v", name, got)
		}
	}
}

func TestQueryCompile(t *testing.T) {
	env := newTestEnv(t)
	s, _ := env.entityOf(typeOf[order]())

	q, err := Where("Merchant", "m1").And("Amount", In, []int{1, 2}).And("Amount", Gt, 0).Limit(5).compile(s)
	if err != nil {
		t.Fatal(err)
	}

	if q.Kind != "order" || q.Limit != 5 || len(q.Where) != 3 || q.Where[0].Field != "merchant" {
		t.Fatalf("query = %+v", q)
	}

	if in := q.Where[1].Value.([]any); in[0] != uint64(1) {
		t.Errorf("In values not converted: %#v", in)
	}

	if q.Where[2].Value != uint64(0) {
		t.Errorf("Gt value not converted: %#v", q.Where[2].Value)
	}

	for name, bad := range map[string]Query{
		"unknown field": Where("Nope", "x"),
		"skipped field": Where("Secret", "x"),
		"negative uint": Where("Amount", -1),
		"fraction":      Where("Amount", 1.5),
		"wrong type":    Where("Merchant", 1),
		"in non-slice":  Query{}.And("Amount", In, 1),
	} {
		if _, err := bad.compile(s); !errors.Is(err, ErrInvalidQuery) {
			t.Errorf("%s: want ErrInvalidQuery, got %v", name, err)
		}
	}

	// And must not alias the backing array of a shared base query.
	base := Where("Merchant", "m1")

	a, b := base.And("Amount", Eq, 1), base.And("Amount", Eq, 2)
	if a.where[1].Value == b.where[1].Value {
		t.Error("queries derived from the same base share conditions")
	}
}

func TestAggregateSchema(t *testing.T) {
	env := newTestEnv(t)

	c, err := NewAggregateController[orderAgg](env)
	if err != nil {
		t.Fatal(err)
	}

	s := c.schema
	if s.name != "order-agg" || len(s.members) != 3 || s.members[s.root].field != "Order" {
		t.Fatalf("schema = %+v", s)
	}

	if s.members[1].shape != shapeSlice || s.members[2].shape != shapePtrMap {
		t.Errorf("shapes = %v, %v", s.members[1].shape, s.members[2].shape)
	}

	if _, err := NewAggregateController[orderAgg](env); !errors.Is(err, ErrAlreadyRegistered) {
		t.Errorf("duplicate aggregate: want ErrAlreadyRegistered, got %v", err)
	}

	type twoRoots struct {
		A *order   `ee:"root"`
		B *payment `ee:"root"`
	}
	if _, err := parseAggregateSchema(env, "x", typeOf[twoRoots]()); !errors.Is(err, ErrInvalidSchema) {
		t.Errorf("two roots: want ErrInvalidSchema, got %v", err)
	}

	type noRoot struct{ Items []orderItem }
	if _, err := parseAggregateSchema(env, "x", typeOf[noRoot]()); !errors.Is(err, ErrInvalidSchema) {
		t.Errorf("no root: want ErrInvalidSchema, got %v", err)
	}

	type unregistered struct {
		O *order
		X *noID
	}
	if _, err := parseAggregateSchema(env, "x", typeOf[unregistered]()); !errors.Is(err, ErrNotRegistered) {
		t.Errorf("unregistered member: want ErrNotRegistered, got %v", err)
	}

	// Aggregate scopes only reach member state types.
	sc := &scope{env: env, allow: s.allow}
	if err := repoOf[orderItem](sc).err; err != nil {
		t.Errorf("orderItem is a member, got %v", err)
	}

	if err := repoOf[noID](sc).err; !errors.Is(err, ErrNotMember) {
		t.Errorf("want ErrNotMember, got %v", err)
	}
}

func TestAction(t *testing.T) {
	env := newTestEnv(t)
	orders, _ := NewEntityController[order](MustNew())
	items, _ := NewEntityController[orderItem](orders.env)

	if err := orders.Action(func(*ActionCtx[order, orderPaid]) error { return nil }, Filter(Match("merchant", "m1"))); err != nil {
		t.Fatal(err)
	}

	// One consumer per event, across controllers.
	if err := items.Action(func(*ActionCtx[orderItem, orderPaid]) error { return nil }); !errors.Is(err, ErrAlreadyRegistered) {
		t.Errorf("second consumer: want ErrAlreadyRegistered, got %v", err)
	}

	if err := items.Action(func(*ActionCtx[orderItem, orderNoted]) error { return nil }, Filter(HasLabel("x"))); err == nil {
		t.Error("Filter on an event without Labels must fail")
	}

	aggs, _ := NewAggregateController[orderAgg](env)
	if err := aggs.Action(func(*AggregateCtx[orderAgg, orderPaid]) error { return nil }); err != nil {
		t.Fatal(err)
	}

	// orderNoted has no Key, so this does not compile:
	//   aggs.Action(func(*AggregateCtx[orderAgg, orderNoted]) error { return nil })
}
