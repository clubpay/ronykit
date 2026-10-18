package ee

import (
	"context"
	"fmt"
	"reflect"

	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/store"
)

// Aggregate is implemented (on the pointer) by aggregate structs.
// AggregateName must be a constant of the type. Load fills the aggregate with
// id using ctx.Entities[S](); what it loads is tracked so the aggregate can be
// saved later.
type Aggregate interface {
	AggregateName() string
	Load(ctx *LoadCtx, id string) error
}

// Saver optionally replaces the default diff-and-save of an aggregate.
type Saver interface {
	Save(ctx *SaveCtx) error
}

type memberShape int

const (
	shapeValue    memberShape = iota + 1 // S
	shapePtr                             // *S
	shapeSlice                           // []S
	shapePtrSlice                        // []*S
	shapeMap                             // map[string]S
	shapePtrMap                          // map[string]*S
)

type member struct {
	field  string
	index  []int
	shape  memberShape
	schema *entitySchema
}

type aggregateSchema struct {
	name    string
	typ     reflect.Type
	members []member
	root    int // index into members
	allow   map[reflect.Type]bool
}

func parseAggregateSchema(env *Env, name string, t reflect.Type) (*aggregateSchema, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: %s: empty AggregateName", ErrInvalidSchema, t)
	}

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: %s: aggregate must be a struct", ErrInvalidSchema, t)
	}

	a := &aggregateSchema{name: name, typ: t, root: -1, allow: map[reflect.Type]bool{}}
	for f := range t.Fields() {
		opts := tagOptions(f)
		if len(opts) == 1 && opts[0] == "-" || !f.IsExported() {
			continue
		}

		shape, st := memberShapeOf(f.Type)
		if shape == 0 {
			return nil, fmt.Errorf("%w: %s.%s: %s is not an entity member (use `ee:\"-\"` to skip it)",
				ErrInvalidSchema, t, f.Name, f.Type)
		}

		schema, err := env.entityOf(st)
		if err != nil {
			return nil, fmt.Errorf("%w: %s.%s: %w", ErrInvalidSchema, t, f.Name, err)
		}

		for _, o := range opts {
			if o != "root" {
				return nil, fmt.Errorf("%w: %s.%s: unknown tag option %q", ErrInvalidSchema, t, f.Name, o)
			}

			if shape != shapeValue && shape != shapePtr {
				return nil, fmt.Errorf("%w: %s.%s: root must be a single state", ErrInvalidSchema, t, f.Name)
			}

			if a.root >= 0 {
				return nil, fmt.Errorf("%w: %s: more than one root", ErrInvalidSchema, t)
			}

			a.root = len(a.members)
		}

		a.members = append(a.members, member{field: f.Name, index: f.Index, shape: shape, schema: schema})
		a.allow[st] = true
	}

	// Without an explicit root, the first single-entity member is the root.
	if a.root < 0 {
		for i, m := range a.members {
			if m.shape == shapeValue || m.shape == shapePtr {
				a.root = i

				break
			}
		}
	}

	if a.root < 0 {
		return nil, fmt.Errorf("%w: %s: aggregate needs a single-state root member", ErrInvalidSchema, t)
	}

	return a, nil
}

// memberShapeOf classifies t as one of the supported member shapes and
// returns the state type it holds.
func memberShapeOf(t reflect.Type) (memberShape, reflect.Type) {
	switch t.Kind() {
	case reflect.Struct:
		return shapeValue, t
	case reflect.Pointer:
		if t.Elem().Kind() == reflect.Struct {
			return shapePtr, t.Elem()
		}
	case reflect.Slice:
		switch e := t.Elem(); {
		case e.Kind() == reflect.Struct:
			return shapeSlice, e
		case e.Kind() == reflect.Pointer && e.Elem().Kind() == reflect.Struct:
			return shapePtrSlice, e.Elem()
		}
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return 0, nil
		}

		switch e := t.Elem(); {
		case e.Kind() == reflect.Struct:
			return shapeMap, e
		case e.Kind() == reflect.Pointer && e.Elem().Kind() == reflect.Struct:
			return shapePtrMap, e.Elem()
		}
	default:
	}

	return 0, nil
}

// AggregateController owns one aggregate type A.
type AggregateController[A any] struct {
	env     *Env
	schema  *aggregateSchema
	actions map[string]*action
	asAgg   func(*A) Aggregate
}

// NewAggregateController registers A. Every member state must already be
// registered with NewEntityController. Call it as
// ee.NewAggregateController[OrderAgg](env).
func NewAggregateController[A any, PA interface {
	*A
	Aggregate
}](env *Env) (*AggregateController[A], error) {
	schema, err := parseAggregateSchema(env, PA(new(A)).AggregateName(), reflect.TypeFor[A]())
	if err != nil {
		return nil, err
	}

	c := &AggregateController[A]{
		env:     env,
		schema:  schema,
		actions: map[string]*action{},
		asAgg:   func(a *A) Aggregate { return PA(a) },
	}

	err = env.register(func() error {
		return env.addController(c.owner(), c)
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *AggregateController[A]) Name() string { return c.schema.name }

// Get loads the aggregate through a read-only transaction, which the store may
// serve from a replica.
func (c *AggregateController[A]) Get(ctx context.Context, id string) (*A, error) {
	var out *A

	err := c.env.view(ctx, c.schema.allow, func(sc *scope) error {
		a, _, err := c.load(sc, id)
		out = a

		return err
	})

	return out, err
}

// Create inserts every member of a in one transaction.
func (c *AggregateController[A]) Create(ctx context.Context, a *A) error {
	return c.env.unit(ctx, c.schema.allow, false, func(sc *scope) error {
		_, err := c.save(sc, a, nil)

		return err
	})
}

// Update loads the aggregate, applies fn and saves it in one transaction,
// retrying on conflict.
func (c *AggregateController[A]) Update(ctx context.Context, id string, fn func(*A) error) error {
	return c.env.unit(ctx, c.schema.allow, true, func(sc *scope) error {
		a, members, err := c.load(sc, id)
		if err != nil {
			return err
		}

		if err := fn(a); err != nil {
			return err
		}

		_, err = c.save(sc, a, members)

		return err
	})
}

// Delete loads the aggregate and deletes every member state it holds.
func (c *AggregateController[A]) Delete(ctx context.Context, id string) error {
	return c.env.unit(ctx, c.schema.allow, true, func(sc *scope) error {
		_, members, err := c.load(sc, id)
		if err != nil {
			return err
		}

		for k := range members {
			if tr := sc.tracked[k]; tr != nil && !tr.deleted {
				if err := sc.remove(tr.schema, tr.ptr); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// Action makes the aggregate the consumer of event E. The event's Key is the
// aggregate id.
func (c *AggregateController[A]) Action[E KeyedEvent](
	h func(*AggregateCtx[A, E]) error,
	opts ...ActionOption,
) error {
	a, err := newAction[E](c.owner(), opts)
	if err != nil {
		return err
	}

	a.run = func(sc *scope, msg *bus.Message) error {
		ev, err := decodeEvent[E](c.env, msg)
		if err != nil {
			return err
		}

		actx := &AggregateCtx[A, E]{Context: sc.ctx, sc: sc, ctl: c, event: ev, meta: metaOf(&msg.Envelope)}
		if err := h(actx); err != nil {
			return err
		}

		if c.env.autoSave && actx.agg != nil && !actx.saved {
			return actx.Save(actx.agg)
		}

		return nil
	}

	return c.env.register(func() error {
		if err := c.env.claim(a.event, c.owner()); err != nil {
			return err
		}

		c.actions[a.event] = a

		return nil
	})
}

func (c *AggregateController[A]) owner() string { return "aggregate:" + c.schema.name }

func (c *AggregateController[A]) subscribedEvents() []string { return eventNames(c.actions) }

func (c *AggregateController[A]) handle(ctx context.Context, msg bus.Message) error {
	return c.env.dispatch(ctx, c.owner(), c.actions, c.schema.allow, msg)
}

// load runs the developer's Load and returns the member states the aggregate
// holds afterwards; a later save deletes those that disappear.
func (c *AggregateController[A]) load(sc *scope, id string) (*A, map[entityKey]bool, error) {
	a := new(A)
	if err := c.asAgg(a).Load(&LoadCtx{Context: sc.ctx, sc: sc}, id); err != nil {
		return nil, nil, err
	}

	refs, err := c.collect(a)
	if err != nil {
		return nil, nil, err
	}

	members := make(map[entityKey]bool, len(refs))
	for _, r := range refs {
		members[r.key] = true
	}

	return a, members, nil
}

// save writes a and returns its current member set. prev is the member set
// as of the last load/save (nil for a new aggregate).
func (c *AggregateController[A]) save(sc *scope, a *A, prev map[entityKey]bool) (map[entityKey]bool, error) {
	refs, err := c.collect(a)
	if err != nil {
		return nil, err
	}

	present := make(map[entityKey]bool, len(refs))
	for _, r := range refs {
		if present[r.key] {
			return nil, fmt.Errorf("ee: aggregate %s holds %s/%s twice", c.schema.name, r.key.kind, r.key.id)
		}

		present[r.key] = true
	}

	if s, ok := c.asAgg(a).(Saver); ok {
		return present, s.Save(&SaveCtx{Context: sc.ctx, sc: sc})
	}

	for _, r := range refs {
		if err := c.saveMember(sc, r); err != nil {
			return nil, err
		}
	}

	for k := range prev {
		if present[k] {
			continue
		}

		if tr := sc.tracked[k]; tr != nil && !tr.deleted {
			if err := sc.remove(tr.schema, tr.ptr); err != nil {
				return nil, err
			}
		}
	}

	return present, nil
}

func (c *AggregateController[A]) saveMember(sc *scope, r memberRef) error {
	tr, ok := sc.tracked[r.key]
	if !ok || tr.deleted {
		return sc.create(r.schema, r.ptr)
	}

	// The root is always written, so its version serializes concurrent saves
	// of the same aggregate even when only other members changed.
	if r.root {
		return sc.save(r.schema, r.ptr)
	}

	changed, err := sc.dirty(tr, r.ptr)
	if err != nil || !changed {
		return err
	}

	return sc.save(r.schema, r.ptr)
}

type memberRef struct {
	key    entityKey
	schema *entitySchema
	ptr    reflect.Value
	root   bool
}

// collect lists the member states a currently holds. Map values are copied,
// since Go map elements are not addressable.
func (c *AggregateController[A]) collect(a *A) ([]memberRef, error) {
	v := reflect.ValueOf(a).Elem()

	var refs []memberRef

	add := func(m member, root bool, ptr reflect.Value) error {
		id := m.schema.id(ptr.Elem())
		if id == "" {
			return fmt.Errorf(
				"ee: aggregate %s: %s has a %s with an empty id",
				c.schema.name,
				m.field,
				m.schema.name,
			)
		}

		refs = append(
			refs,
			memberRef{key: entityKey{m.schema.name, id}, schema: m.schema, ptr: ptr, root: root},
		)

		return nil
	}

	for i, m := range c.schema.members {
		root := i == c.schema.root
		fv := v.FieldByIndex(m.index)

		var ptrs []reflect.Value

		switch m.shape {
		case shapeValue:
			ptrs = append(ptrs, fv.Addr())
		case shapePtr:
			if !fv.IsNil() {
				ptrs = append(ptrs, fv)
			}
		case shapeSlice:
			for j := range fv.Len() {
				ptrs = append(ptrs, fv.Index(j).Addr())
			}
		case shapePtrSlice:
			for j := range fv.Len() {
				if e := fv.Index(j); !e.IsNil() {
					ptrs = append(ptrs, e)
				}
			}
		case shapeMap:
			for _, e := range fv.Seq2() {
				cp := reflect.New(e.Type())
				cp.Elem().Set(e)
				ptrs = append(ptrs, cp)
			}
		case shapePtrMap:
			for _, e := range fv.Seq2() {
				if !e.IsNil() {
					ptrs = append(ptrs, e)
				}
			}
		}

		if root && len(ptrs) == 0 {
			return nil, fmt.Errorf("ee: aggregate %s: root %s is nil", c.schema.name, m.field)
		}

		for _, p := range ptrs {
			if err := add(m, root, p); err != nil {
				return nil, err
			}
		}
	}

	return refs, nil
}

// LoadCtx is passed to Aggregate.Load. Only member state types are reachable.
type LoadCtx struct {
	context.Context //nolint:containedctx // handler contexts are request-scoped by design

	sc *scope
}

func (c *LoadCtx) Entities[S Entity]() *Repo[S] { return repoOf[S](c.sc) }

// Tx is the underlying store transaction, for backend-specific queries
// (e.g. postgres.PgxTx).
func (c *LoadCtx) Tx() store.Tx { return c.sc.tx }

// SaveCtx is passed to Saver.Save. Only member state types are reachable.
type SaveCtx struct {
	context.Context //nolint:containedctx // handler contexts are request-scoped by design

	sc *scope
}

func (c *SaveCtx) Entities[S Entity]() *Repo[S] { return repoOf[S](c.sc) }

// Tx is the underlying store transaction, for backend-specific queries.
func (c *SaveCtx) Tx() store.Tx { return c.sc.tx }

// AggregateCtx is what an AggregateController action handler receives.
type AggregateCtx[A any, E KeyedEvent] struct {
	context.Context //nolint:containedctx // handler contexts are request-scoped by design

	sc    *scope
	ctl   *AggregateController[A]
	event E
	meta  Meta

	agg     *A
	members map[entityKey]bool
	saved   bool
}

func (c *AggregateCtx[A, E]) Event() E   { return c.event }
func (c *AggregateCtx[A, E]) Meta() Meta { return c.meta }

// Aggregate loads the aggregate whose id is the event's Key. Later calls
// return the same pointer.
func (c *AggregateCtx[A, E]) Aggregate() (*A, error) {
	if c.agg != nil {
		return c.agg, nil
	}

	a, members, err := c.ctl.load(c.sc, c.event.Key())
	if err != nil {
		return nil, err
	}

	c.agg, c.members = a, members

	return a, nil
}

// Save writes a now: its Saver if it implements one, otherwise a diff against
// what Load returned (insert new members, update changed ones, delete removed
// ones). The root is always written, so concurrent saves of the same
// aggregate serialize.
func (c *AggregateCtx[A, E]) Save(a *A) error {
	members, err := c.ctl.save(c.sc, a, c.members)
	if err != nil {
		return err
	}

	c.members, c.saved = members, true

	return nil
}

// Tx is the underlying store transaction, for backend-specific queries.
func (c *AggregateCtx[A, E]) Tx() store.Tx { return c.sc.tx }

// Entities gives access to member entity types in the same transaction.
func (c *AggregateCtx[A, E]) Entities[S Entity]() *Repo[S] { return repoOf[S](c.sc) }

// Emit queues an event in the transactional outbox.
func (c *AggregateCtx[A, E]) Emit[T Event](ev T, opts ...SendOption) error {
	return c.sc.emit(ev, opts)
}
