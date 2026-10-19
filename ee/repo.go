package ee

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"reflect"

	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/store"
)

// scope is one unit of work: a storage transaction, an identity map of the
// states loaded through it and the events emitted from it. Writes go to the
// transaction as they are made; on commit, emitted events are appended to the
// outbox (and, with WithAutoSave, changed tracked states are saved).
type scope struct {
	ctx     context.Context //nolint:containedctx // a scope lives for one unit of work
	env     *Env
	tx      store.Tx
	tracked map[entityKey]*trackedEntity
	emitted []bus.Envelope
	// allow restricts which state types are reachable (aggregate members);
	// nil allows every registered state.
	allow map[reflect.Type]bool
}

type entityKey struct{ kind, id string }

type trackedEntity struct {
	schema  *entitySchema
	ptr     reflect.Value // *S handed to user code
	version uint64
	// snapshot is the encoded value as last read or written, for dirty
	// checking (auto-save and aggregate diffs).
	snapshot []byte
	deleted  bool
}

func newScope(ctx context.Context, env *Env, tx store.Tx, allow map[reflect.Type]bool) *scope {
	return &scope{ctx: ctx, env: env, tx: tx, tracked: map[entityKey]*trackedEntity{}, allow: allow}
}

func (s *scope) emit(ev Event, opts []SendOption) error {
	env, err := s.env.envelope(ev, opts)
	if err != nil {
		return err
	}

	s.emitted = append(s.emitted, env)

	return nil
}

func (s *scope) get(schema *entitySchema, id string) (reflect.Value, error) {
	k := entityKey{schema.name, id}
	if tr, ok := s.tracked[k]; ok {
		if tr.deleted {
			return reflect.Value{}, fmt.Errorf("%w: %s/%s", ErrNotFound, schema.name, id)
		}

		return tr.ptr, nil
	}

	rec, err := s.tx.Get(s.ctx, schema.name, id)
	if err != nil {
		return reflect.Value{}, fmt.Errorf("%w: %s/%s", err, schema.name, id)
	}

	return s.track(schema, rec)
}

func (s *scope) find(schema *entitySchema, q Query) iter.Seq2[reflect.Value, error] {
	sq, err := q.compile(schema)

	return func(yield func(reflect.Value, error) bool) {
		if err != nil {
			yield(reflect.Value{}, err)

			return
		}

		for rec, err := range s.tx.Find(s.ctx, sq) {
			if err != nil {
				yield(reflect.Value{}, err)

				return
			}

			// The identity map wins: a state already loaded in this unit of
			// work is returned as-is, with its pending changes.
			if tr, ok := s.tracked[entityKey{schema.name, rec.ID}]; ok {
				if !tr.deleted && !yield(tr.ptr, nil) {
					return
				}

				continue
			}

			ptr, err := s.track(schema, rec)
			if !yield(ptr, err) || err != nil {
				return
			}
		}
	}
}

func (s *scope) track(schema *entitySchema, rec store.Record) (reflect.Value, error) {
	ptr := reflect.New(schema.typ)
	if err := s.env.codec.Unmarshal(rec.Data, ptr.Interface()); err != nil {
		return reflect.Value{}, fmt.Errorf("ee: decode %s/%s: %w", schema.name, rec.ID, err)
	}

	snapshot, err := s.env.codec.Marshal(ptr.Interface())
	if err != nil {
		return reflect.Value{}, fmt.Errorf("ee: encode %s/%s: %w", schema.name, rec.ID, err)
	}

	s.tracked[entityKey{schema.name, rec.ID}] = &trackedEntity{
		schema:   schema,
		ptr:      ptr,
		version:  rec.Version,
		snapshot: snapshot,
	}

	return ptr, nil
}

func (s *scope) create(schema *entitySchema, ptr reflect.Value) error {
	id := schema.id(ptr.Elem())
	if id == "" {
		return fmt.Errorf("ee: create %s: empty id", schema.name)
	}

	data, err := s.env.codec.Marshal(ptr.Interface())
	if err != nil {
		return fmt.Errorf("ee: encode %s/%s: %w", schema.name, id, err)
	}

	err = s.tx.Insert(s.ctx, store.Record{Kind: schema.name, ID: id, Version: 1, Data: data})
	if err != nil {
		return fmt.Errorf("%w: create %s/%s", err, schema.name, id)
	}

	s.tracked[entityKey{schema.name, id}] = &trackedEntity{
		schema:   schema,
		ptr:      ptr,
		version:  1,
		snapshot: data,
	}

	return nil
}

func (s *scope) tracking(schema *entitySchema, ptr reflect.Value) (*trackedEntity, string, error) {
	id := schema.id(ptr.Elem())

	tr, ok := s.tracked[entityKey{schema.name, id}]
	switch {
	case !ok:
		return nil, id, fmt.Errorf(
			"%w: %s/%s (Get it in this unit of work first)",
			ErrNotLoaded, schema.name, id,
		)
	case tr.deleted:
		return nil, id, fmt.Errorf("%w: %s/%s", ErrNotFound, schema.name, id)
	default:
		return tr, id, nil
	}
}

func (s *scope) save(schema *entitySchema, ptr reflect.Value) error {
	tr, id, err := s.tracking(schema, ptr)
	if err != nil {
		return err
	}

	data, err := s.env.codec.Marshal(ptr.Interface())
	if err != nil {
		return fmt.Errorf("ee: encode %s/%s: %w", schema.name, id, err)
	}

	err = s.tx.Update(s.ctx, store.Record{Kind: schema.name, ID: id, Version: tr.version, Data: data})
	if err != nil {
		return fmt.Errorf("%w: save %s/%s", err, schema.name, id)
	}

	tr.version++
	tr.snapshot = data
	tr.ptr = ptr

	return nil
}

func (s *scope) remove(schema *entitySchema, ptr reflect.Value) error {
	tr, id, err := s.tracking(schema, ptr)
	if err != nil {
		return err
	}

	if err := s.tx.Delete(s.ctx, schema.name, id, tr.version); err != nil {
		return fmt.Errorf("%w: delete %s/%s", err, schema.name, id)
	}

	tr.deleted = true

	return nil
}

// dirty reports whether ptr differs from what was last read or written.
func (s *scope) dirty(tr *trackedEntity, ptr reflect.Value) (bool, error) {
	data, err := s.env.codec.Marshal(ptr.Interface())
	if err != nil {
		return false, err
	}

	return !bytes.Equal(data, tr.snapshot), nil
}

// rejectUnsaved fails the unit of work when WithRequireSave is set and a
// tracked entity was mutated without Save.
func (s *scope) rejectUnsaved() error {
	if !s.env.requireSave {
		return nil
	}

	for k, tr := range s.tracked {
		if tr.deleted {
			continue
		}

		changed, err := s.dirty(tr, tr.ptr)
		if err != nil {
			return err
		}

		if changed {
			return fmt.Errorf("ee: %s/%s changed without Save; call Save or use WithAutoSave", k.kind, k.id)
		}
	}

	return nil
}

// saveDirty saves every tracked state that changed since it was last read or
// written (WithAutoSave).
func (s *scope) saveDirty() error {
	for _, tr := range s.tracked {
		if tr.deleted {
			continue
		}

		changed, err := s.dirty(tr, tr.ptr)
		if err != nil {
			return err
		}

		if !changed {
			continue
		}

		if err := s.save(tr.schema, tr.ptr); err != nil {
			return err
		}
	}

	return nil
}

// commit runs at the end of a successful unit of work, inside the
// transaction.
func (s *scope) commit() error {
	if err := s.rejectUnsaved(); err != nil {
		return err
	}

	if s.env.autoSave {
		if err := s.saveDirty(); err != nil {
			return err
		}
	}

	if len(s.emitted) == 0 {
		return nil
	}

	entries := make([]store.OutboxEntry, len(s.emitted))
	for i, env := range s.emitted {
		data, err := json.Marshal(env)
		if err != nil {
			return fmt.Errorf("ee: encode envelope %s: %w", env.Event, err)
		}

		entries[i] = store.OutboxEntry{
			ID:           env.ID,
			PartitionKey: env.PartitionKey,
			Data:         data,
			CreatedAt:    env.CreatedAt,
		}
	}

	return s.tx.AppendOutbox(s.ctx, entries...)
}

// Repo gives transactional access to states of type S. Get, Find and Create
// track the version of what they return; Save and Delete need that version
// for their optimistic check, so they only accept tracked states.
//
// Within one unit of work, Get and Find return the same pointer for the same
// state (identity map).
type Repo[S Entity] struct {
	sc     *scope
	schema *entitySchema
	err    error
}

func repoOf[S Entity](sc *scope) *Repo[S] {
	t := reflect.TypeFor[S]()
	if sc.allow != nil && !sc.allow[t] {
		return &Repo[S]{err: fmt.Errorf("%w: %s", ErrNotMember, t)}
	}

	schema, err := sc.env.entityOf(t)

	return &Repo[S]{sc: sc, schema: schema, err: err}
}

func (r *Repo[S]) Get(id string) (*S, error) {
	if r.err != nil {
		return nil, r.err
	}

	v, err := r.sc.get(r.schema, id)
	if err != nil {
		return nil, err
	}

	return v.Interface().(*S), nil
}

// Find streams the states matching q. Consume it before the handler returns.
func (r *Repo[S]) Find(q Query) iter.Seq2[*S, error] {
	return func(yield func(*S, error) bool) {
		if r.err != nil {
			yield(nil, r.err)

			return
		}

		for v, err := range r.sc.find(r.schema, q) {
			if err != nil {
				yield(nil, err)

				return
			}

			if !yield(v.Interface().(*S), nil) {
				return
			}
		}
	}
}

// Create inserts v and starts tracking it.
func (r *Repo[S]) Create(v *S) error {
	if r.err != nil {
		return r.err
	}

	return r.sc.create(r.schema, reflect.ValueOf(v))
}

// Save writes v now, inside the transaction. v must have been returned by
// Get/Find/Create in this unit of work (ErrNotLoaded otherwise).
func (r *Repo[S]) Save(v *S) error {
	if r.err != nil {
		return r.err
	}

	return r.sc.save(r.schema, reflect.ValueOf(v))
}

// Delete removes v. Like Save, v must be tracked.
func (r *Repo[S]) Delete(v *S) error {
	if r.err != nil {
		return r.err
	}

	return r.sc.remove(r.schema, reflect.ValueOf(v))
}
