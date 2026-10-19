// Package storetest is a conformance suite for store.Store backends. Every
// backend runs it from its own tests:
//
//	func TestConformance(t *testing.T) {
//		storetest.Run(t, func(t *testing.T) store.Store { return newEmptyStore(t) })
//	}
//
// newStore must return an empty store; it is called once per subtest. Record
// data is JSON, as produced by ee.JSONCodec.
package storetest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee/store"
	"github.com/clubpay/ronykit/x/rkit"
)

// nameField is the JSON field most Find cases filter on.
const nameField = "name"

func Run(t *testing.T, newStore func(t *testing.T) store.Store) {
	t.Helper()

	tests := []struct {
		name string
		fn   func(t *testing.T, s store.Store)
	}{
		{"CRUD", testCRUD},
		{"ConcurrentUpdateConflicts", testConcurrentUpdate},
		{"DuplicateInsertKeepsTxUsable", testDuplicateInsert},
		{"Rollback", testRollback},
		{"ReadOwnWrites", testReadOwnWrites},
		{"Find", testFind},
		{"FindWhileWriting", testFindWhileWriting},
		{"FindEarlyStop", testFindEarlyStop},
		{"FindWithTxWrites", testFindWithTxWrites},
		{"DeleteMissing", testDeleteMissing},
		{"StaleOpWithinTx", testStaleWithinTx},
		{"InsertDeleteReinsert", testInsertDeleteReinsert},
		{"View", testView},
		{"ViewWritesFail", testViewWrites},
		{"OutboxClaimAck", testOutbox},
		{"OutboxPerKeyOrder", testOutboxPerKeyOrder},
		{"OutboxLeaseExpiry", testOutboxLeaseExpiry},
		{"OutboxStats", testOutboxStats},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) { tc.fn(t, newStore(t)) })
	}
}

func rec(kind, id, data string) store.Record {
	return store.Record{Kind: kind, ID: id, Version: 1, Data: []byte(data)}
}

func tx(t *testing.T, s store.Store, fn func(ctx context.Context, tx store.Tx) error) error {
	t.Helper()

	return s.Tx(t.Context(), fn)
}

func mustTx(t *testing.T, s store.Store, fn func(ctx context.Context, tx store.Tx) error) {
	t.Helper()

	if err := tx(t, s, fn); err != nil {
		t.Fatal(err)
	}
}

func get(t *testing.T, s store.Store, id string) (store.Record, error) {
	t.Helper()

	var r store.Record

	err := tx(t, s, func(ctx context.Context, tx store.Tx) error {
		var err error

		r, err = tx.Get(ctx, "k", id)

		return err
	})

	return r, err
}

func testCRUD(t *testing.T, s store.Store) {
	t.Helper()

	mustTx(t, s, func(ctx context.Context, tx store.Tx) error {
		return tx.Insert(ctx, rec("k", "1", `{"v":1}`))
	})

	r, err := get(t, s, "1")
	if err != nil || r.Version != 1 || r.Kind != "k" || r.ID != "1" {
		t.Fatalf("get after insert: %+v, %v", r, err)
	}

	err = tx(t, s, func(ctx context.Context, tx store.Tx) error {
		return tx.Update(ctx, store.Record{Kind: "k", ID: "1", Version: 1, Data: []byte(`{"v":2}`)})
	})
	if err != nil {
		t.Fatal(err)
	}

	if r, _ := get(t, s, "1"); r.Version != 2 || !jsonEqual(r.Data, `{"v":2}`) {
		t.Errorf("after update: version %d, data %s", r.Version, r.Data)
	}

	err = tx(t, s, func(ctx context.Context, tx store.Tx) error {
		return tx.Update(ctx, store.Record{Kind: "k", ID: "1", Version: 1, Data: []byte(`{}`)})
	})
	if !errors.Is(err, store.ErrConflict) {
		t.Errorf("stale update: want ErrConflict, got %v", err)
	}

	err = tx(t, s, func(ctx context.Context, tx store.Tx) error { return tx.Delete(ctx, "k", "1", 1) })
	if !errors.Is(err, store.ErrConflict) {
		t.Errorf("stale delete: want ErrConflict, got %v", err)
	}

	mustTx(t, s, func(ctx context.Context, tx store.Tx) error { return tx.Delete(ctx, "k", "1", 2) })

	if _, err := get(t, s, "1"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("after delete: want ErrNotFound, got %v", err)
	}

	err = tx(t, s, func(ctx context.Context, tx store.Tx) error {
		return tx.Update(ctx, store.Record{Kind: "k", ID: "1", Version: 2, Data: []byte(`{}`)})
	})
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("update of missing record: want ErrNotFound, got %v", err)
	}
}

func testConcurrentUpdate(t *testing.T, s store.Store) {
	t.Helper()

	mustTx(t, s, func(ctx context.Context, tx store.Tx) error { return tx.Insert(ctx, rec("k", "1", `{}`)) })

	// Both transactions read version 1 before either writes.
	var (
		wg   sync.WaitGroup
		read sync.WaitGroup
		errs = make([]error, 2)
	)

	read.Add(2)

	for i := range 2 {
		wg.Go(func() {
			errs[i] = s.Tx(t.Context(), func(ctx context.Context, tx store.Tx) error {
				r, err := tx.Get(ctx, "k", "1")

				read.Done()

				if err != nil {
					return err
				}

				read.Wait()

				r.Data = fmt.Appendf(nil, `{"by":%d}`, i)

				return tx.Update(ctx, r)
			})
		})
	}

	wg.Wait()

	if (errs[0] == nil) == (errs[1] == nil) || !errors.Is(errors.Join(errs...), store.ErrConflict) {
		t.Errorf("want exactly one ErrConflict, got %v", errs)
	}

	if r, _ := get(t, s, "1"); r.Version != 2 {
		t.Errorf("version = %d, want 2", r.Version)
	}
}

func testDuplicateInsert(t *testing.T, s store.Store) {
	t.Helper()

	mustTx(t, s, func(ctx context.Context, tx store.Tx) error { return tx.Insert(ctx, rec("k", "1", `{}`)) })

	mustTx(t, s, func(ctx context.Context, tx store.Tx) error {
		if err := tx.Insert(ctx, rec("k", "1", `{}`)); !errors.Is(err, store.ErrConflict) {
			return fmt.Errorf("duplicate insert: want ErrConflict, got %w", err)
		}

		// The transaction must still be usable (ee.Dedup relies on it).
		return tx.Insert(ctx, rec("k", "2", `{}`))
	})

	if _, err := get(t, s, "2"); err != nil {
		t.Errorf("insert after a conflict was lost: %v", err)
	}
}

func testRollback(t *testing.T, s store.Store) {
	t.Helper()

	boom := errors.New("boom")

	err := tx(t, s, func(ctx context.Context, tx store.Tx) error {
		if err := tx.Insert(ctx, rec("k", "1", `{}`)); err != nil {
			return err
		}

		if err := tx.AppendOutbox(
			ctx,
			store.OutboxEntry{ID: "e1", Data: []byte("x"), CreatedAt: time.Now()},
		); err != nil {
			return err
		}

		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("want the fn error back, got %v", err)
	}

	if _, err := get(t, s, "1"); !errors.Is(err, store.ErrNotFound) {
		t.Errorf("insert of a rolled back tx is visible: %v", err)
	}

	if got, _ := s.Outbox().Claim(t.Context(), 10, time.Minute); len(got) != 0 {
		t.Errorf("outbox entries of a rolled back tx are visible: %v", got)
	}
}

func testReadOwnWrites(t *testing.T, s store.Store) {
	t.Helper()

	mustTx(t, s, func(ctx context.Context, tx store.Tx) error {
		if err := tx.Insert(ctx, rec("k", "1", `{"name":"a"}`)); err != nil {
			return err
		}

		r, err := tx.Get(ctx, "k", "1")
		if err != nil {
			return fmt.Errorf("get own insert: %w", err)
		}

		r.Data = []byte(`{"name":"b"}`)
		if err := tx.Update(ctx, r); err != nil {
			return fmt.Errorf("update own insert: %w", err)
		}

		var names []string

		q := store.Query{Kind: "k", Where: []store.Cond{{Field: nameField, Op: store.Eq, Value: "b"}}}

		for r, err := range tx.Find(ctx, q) {
			if err != nil {
				return err
			}

			names = append(names, r.ID)
		}

		if !slices.Equal(names, []string{"1"}) {
			return fmt.Errorf("find own writes: %v", names)
		}

		return nil
	})
}

type findRow struct {
	ID   string
	Name string
	N    int
	F    float64
	OK   bool
	At   time.Time
}

func testFind(t *testing.T, s store.Store) {
	t.Helper()

	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rows := []findRow{
		{"1", "a", 1, 0.5, true, base},
		{"2", "b", 2, 1.5, false, base.Add(time.Hour)},
		{"3", "c", 3, 2.5, true, base.Add(2 * time.Hour)},
		{"4", "b", 4, 3.5, false, base.Add(3 * time.Hour)},
	}

	mustTx(t, s, func(ctx context.Context, tx store.Tx) error {
		for _, r := range rows {
			data := fmt.Sprintf(`{"id":%q,"name":%q,"n":%d,"f":%v,"ok":%t,"at":%q}`,
				r.ID, r.Name, r.N, r.F, r.OK, r.At.Format(time.RFC3339Nano))
			if err := tx.Insert(ctx, rec("row", r.ID, data)); err != nil {
				return err
			}
		}

		// Another kind must never leak into results.
		return tx.Insert(ctx, rec("other", "1", `{"name":"a"}`))
	})

	cases := []struct {
		name string
		q    store.Query
		want []string
	}{
		{"all", store.Query{}, []string{"1", "2", "3", "4"}},
		{"eq string", where(nameField, store.Eq, "b"), []string{"2", "4"}},
		{"neq string", where(nameField, store.Neq, "b"), []string{"1", "3"}},
		{"gt string", where(nameField, store.Gt, "a"), []string{"2", "3", "4"}},
		{"eq int", where("n", store.Eq, 3), []string{"3"}},
		{"gt int", where("n", store.Gt, 2), []string{"3", "4"}},
		{"gte uint", where("n", store.Gte, uint64(2)), []string{"2", "3", "4"}},
		{"lt float", where("f", store.Lt, 1.5), []string{"1"}},
		{"lte float", where("f", store.Lte, 1.5), []string{"1", "2"}},
		{"eq bool", where("ok", store.Eq, true), []string{"1", "3"}},
		{"gte time", where("at", store.Gte, base.Add(2*time.Hour)), []string{"3", "4"}},
		{
			"eq time other zone",
			where("at", store.Eq, base.Add(time.Hour).In(time.FixedZone("x", 3600))),
			[]string{"2"},
		},
		{"in string", where(nameField, store.In, []any{"a", "c"}), []string{"1", "3"}},
		{"in int", where("n", store.In, []any{1, 4}), []string{"1", "4"}},
		{"and", store.Query{Where: []store.Cond{
			{Field: nameField, Op: store.Eq, Value: "b"},
			{Field: "n", Op: store.Gt, Value: 2},
		}}, []string{"4"}},
		{"missing field", where("nope", store.Eq, "x"), nil},
		{
			"order desc",
			store.Query{OrderBy: []store.Order{{Field: "n", Desc: true}}},
			[]string{"4", "3", "2", "1"},
		},
		{
			"order then id",
			store.Query{OrderBy: []store.Order{{Field: nameField}}},
			[]string{"1", "2", "4", "3"},
		},
		{"limit", store.Query{OrderBy: []store.Order{{Field: "n"}}, Limit: 2}, []string{"1", "2"}},
	}

	for _, c := range cases {
		c.q.Kind = "row"
		ordered := len(c.q.OrderBy) > 0

		var got []string

		err := s.View(t.Context(), func(ctx context.Context, tx store.Tx) error {
			for r, err := range tx.Find(ctx, c.q) {
				if err != nil {
					return err
				}

				got = append(got, r.ID)
			}

			return nil
		})
		if err != nil {
			t.Errorf("%s: %v", c.name, err)

			continue
		}

		if !ordered {
			slices.Sort(got)
		}

		if !slices.Equal(got, c.want) {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

func where(field string, op store.Op, v any) store.Query {
	return store.Query{Where: []store.Cond{{Field: field, Op: op, Value: v}}}
}

// testFindWhileWriting checks that a handler may write while iterating Find
// results in the same transaction.
func testFindWhileWriting(t *testing.T, s store.Store) {
	t.Helper()

	mustTx(t, s, func(ctx context.Context, tx store.Tx) error {
		for i := range 250 {
			if err := tx.Insert(ctx, rec("k", fmt.Sprintf("%03d", i), `{"seen":false}`)); err != nil {
				return err
			}
		}

		return nil
	})

	n := 0

	mustTx(t, s, func(ctx context.Context, tx store.Tx) error {
		for r, err := range tx.Find(ctx, store.Query{Kind: "k"}) {
			if err != nil {
				return err
			}

			r.Data = []byte(`{"seen":true}`)
			if err := tx.Update(ctx, r); err != nil {
				return err
			}

			n++
		}

		return nil
	})

	if n != 250 {
		t.Errorf("iterated %d records, want 250", n)
	}
}

func testView(t *testing.T, s store.Store) {
	t.Helper()

	mustTx(t, s, func(ctx context.Context, tx store.Tx) error { return tx.Insert(ctx, rec("k", "1", `{}`)) })

	err := s.View(t.Context(), func(ctx context.Context, tx store.Tx) error {
		if _, err := tx.Get(ctx, "k", "1"); err != nil {
			return err
		}

		return tx.Insert(ctx, rec("k", "2", `{}`))
	})
	if !errors.Is(err, store.ErrReadOnly) {
		t.Errorf("write in View: want ErrReadOnly, got %v", err)
	}
}

// testViewWrites extends testView to every write operation.
func testViewWrites(t *testing.T, s store.Store) {
	t.Helper()

	err := s.View(t.Context(), func(ctx context.Context, tx store.Tx) error {
		if err := tx.Update(
			ctx,
			store.Record{Kind: "k", ID: "1", Version: 1},
		); !errors.Is(
			err,
			store.ErrReadOnly,
		) {
			t.Errorf("update in View: want ErrReadOnly, got %v", err)
		}

		if err := tx.Delete(ctx, "k", "1", 1); !errors.Is(err, store.ErrReadOnly) {
			t.Errorf("delete in View: want ErrReadOnly, got %v", err)
		}

		if err := tx.AppendOutbox(ctx, store.OutboxEntry{ID: "e"}); !errors.Is(err, store.ErrReadOnly) {
			t.Errorf("append in View: want ErrReadOnly, got %v", err)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// testDeleteMissing pins that deleting an absent record reports ErrNotFound,
// like updating one.
func testDeleteMissing(t *testing.T, s store.Store) {
	t.Helper()

	err := tx(t, s, func(ctx context.Context, tx store.Tx) error {
		return tx.Delete(ctx, "k", "missing", 1)
	})
	if !errors.Is(err, store.ErrNotFound) {
		t.Errorf("delete of a missing record: want ErrNotFound, got %v", err)
	}
}

// testStaleWithinTx pins in-transaction version checking: a second stale
// update inside the same transaction conflicts, and the transaction stays
// usable afterwards (ee.Dedup relies on both properties).
func testStaleWithinTx(t *testing.T, s store.Store) {
	t.Helper()

	err := tx(t, s, func(ctx context.Context, tx store.Tx) error {
		if err := tx.Insert(ctx, rec("k", "1", `{"v":1}`)); err != nil {
			return err
		}

		if err := tx.Update(
			ctx,
			store.Record{Kind: "k", ID: "1", Version: 1, Data: []byte(`{"v":2}`)},
		); err != nil {
			return err
		}

		if err := tx.Update(
			ctx,
			store.Record{Kind: "k", ID: "1", Version: 1, Data: []byte(`{"v":3}`)},
		); !errors.Is(
			err,
			store.ErrConflict,
		) {
			t.Errorf("stale update within the tx: want ErrConflict, got %v", err)
		}

		// The transaction is still usable.
		return tx.Insert(ctx, rec("k", "2", `{}`))
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := get(t, s, "2"); err != nil {
		t.Errorf("work after an in-tx conflict was lost: %v", err)
	}
}

// testInsertDeleteReinsert allows a record deleted and re-inserted inside one
// transaction: the delete shadows the insert conflict.
func testInsertDeleteReinsert(t *testing.T, s store.Store) {
	t.Helper()

	err := tx(t, s, func(ctx context.Context, tx store.Tx) error {
		if err := tx.Insert(ctx, rec("k", "1", `{}`)); err != nil {
			return err
		}

		if err := tx.Delete(ctx, "k", "1", 1); err != nil {
			return err
		}

		return tx.Insert(ctx, rec("k", "1", `{"re":true}`))
	})
	if err != nil {
		t.Fatal(err)
	}

	r, err := get(t, s, "1")
	if err != nil || r.Version != 1 || !jsonEqual(r.Data, `{"re":true}`) {
		t.Errorf("after delete+reinsert: %+v, %v", r, err)
	}
}

// testFindWithTxWrites checks that Find merges the transaction's own writes:
// inserts and updates are visible, deletes hide records, and writes of other
// kinds never leak in (or other kinds' records out).
func testFindWithTxWrites(t *testing.T, s store.Store) {
	t.Helper()

	err := tx(t, s, func(ctx context.Context, tx store.Tx) error {
		if err := tx.Insert(ctx, rec("k", "a", `{"name":"x"}`)); err != nil {
			return err
		}

		if err := tx.Insert(ctx, rec("k", "b", `{"name":"y"}`)); err != nil {
			return err
		}

		if err := tx.Delete(ctx, "k", "a", 1); err != nil {
			return err
		}

		if err := tx.Update(
			ctx,
			store.Record{Kind: "k", ID: "b", Version: 1, Data: []byte(`{"name":"z"}`)},
		); err != nil {
			return err
		}

		// A write in another kind must not leak into this kind's results.
		if err := tx.Insert(ctx, rec("other", "o1", `{"name":"z"}`)); err != nil {
			return err
		}

		ids, err := findIDs(ctx, tx, store.Query{Kind: "k"})
		if err != nil {
			return err
		}

		slices.Sort(ids)

		if !slices.Equal(ids, []string{"b"}) {
			return fmt.Errorf("find after own writes: %v, want [b]", ids)
		}

		q := store.Query{Kind: "k", Where: []store.Cond{{Field: nameField, Op: store.Eq, Value: "z"}}}
		if id := firstID(ctx, tx, q); id != "b" {
			return fmt.Errorf("eq z after update: %q, want b", id)
		}

		// The other-kind write exists where asked for it.
		if id := firstID(ctx, tx, store.Query{Kind: "other"}); id != "o1" {
			return fmt.Errorf("other kind: %q, want o1", id)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func findIDs(ctx context.Context, tx store.Tx, q store.Query) ([]string, error) {
	var out []string

	for r, err := range tx.Find(ctx, q) {
		if err != nil {
			return nil, err
		}

		out = append(out, r.ID)
	}

	return out, nil
}

func firstID(ctx context.Context, tx store.Tx, q store.Query) string {
	for r, err := range tx.Find(ctx, q) {
		if err != nil {
			return "err: " + err.Error()
		}

		return r.ID
	}

	return ""
}

// testFindEarlyStop checks that stopping the iteration early is allowed.
func testFindEarlyStop(t *testing.T, s store.Store) {
	t.Helper()

	mustTx(t, s, func(ctx context.Context, tx store.Tx) error {
		for i := range 3 {
			if err := tx.Insert(ctx, rec("k", rkit.IntToStr(i), `{}`)); err != nil {
				return err
			}
		}

		return nil
	})

	err := s.View(t.Context(), func(ctx context.Context, tx store.Tx) error {
		n := 0

		for _, err := range tx.Find(ctx, store.Query{Kind: "k"}) {
			if err != nil {
				return err
			}

			n++

			break
		}

		if n != 1 {
			t.Errorf("visited %d records after break", n)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func appendOutbox(t *testing.T, s store.Store, entries ...store.OutboxEntry) {
	t.Helper()

	for i := range entries {
		entries[i].Data = []byte("payload-" + entries[i].ID)
		entries[i].CreatedAt = time.Now()
	}

	mustTx(t, s, func(ctx context.Context, tx store.Tx) error { return tx.AppendOutbox(ctx, entries...) })
}

func ids(es []store.OutboxEntry) []string {
	return rkit.Map(es, func(e store.OutboxEntry) string { return e.ID })
}

func testOutbox(t *testing.T, s store.Store) {
	t.Helper()

	ctx := t.Context()
	ob := s.Outbox()

	appendOutbox(t, s, store.OutboxEntry{ID: "a"}, store.OutboxEntry{ID: "b"})

	got, err := ob.Claim(ctx, 10, time.Minute)
	if err != nil || !slices.Equal(ids(got), []string{"a", "b"}) {
		t.Fatalf("claim: %v, %v", ids(got), err)
	}

	if string(got[0].Data) != "payload-a" {
		t.Errorf("data = %q", got[0].Data)
	}

	if again, _ := ob.Claim(ctx, 10, time.Minute); len(again) != 0 {
		t.Errorf("leased entries claimed twice: %v", ids(again))
	}

	if err := ob.Ack(ctx, "a", "b"); err != nil {
		t.Fatal(err)
	}
}

func testOutboxPerKeyOrder(t *testing.T, s store.Store) {
	t.Helper()

	ctx := t.Context()
	ob := s.Outbox()

	appendOutbox(t, s,
		store.OutboxEntry{ID: "a", PartitionKey: "k1"},
		store.OutboxEntry{ID: "b", PartitionKey: "k1"},
		store.OutboxEntry{ID: "c", PartitionKey: "k2"},
		store.OutboxEntry{ID: "d"},
	)

	// Relay 1 takes only "a"; relay 2 must not get "b" while "a" is leased.
	r1 := rkit.Ok(ob.Claim(ctx, 1, time.Minute))
	r2 := rkit.Ok(ob.Claim(ctx, 10, time.Minute))

	if !slices.Equal(ids(r1), []string{"a"}) || !slices.Equal(ids(r2), []string{"c", "d"}) {
		t.Fatalf("relay1 %v, relay2 %v", ids(r1), ids(r2))
	}

	_ = ob.Ack(ctx, "a")

	if r3, _ := ob.Claim(ctx, 10, time.Minute); !slices.Equal(ids(r3), []string{"b"}) {
		t.Errorf("after ack: %v", ids(r3))
	}
}

func testOutboxLeaseExpiry(t *testing.T, s store.Store) {
	t.Helper()

	ctx := t.Context()
	ob := s.Outbox()

	appendOutbox(
		t,
		s,
		store.OutboxEntry{ID: "a", PartitionKey: "k"},
		store.OutboxEntry{ID: "b", PartitionKey: "k"},
	)

	if got, _ := ob.Claim(ctx, 1, 300*time.Millisecond); !slices.Equal(ids(got), []string{"a"}) {
		t.Fatalf("claim: %v", ids(got))
	}

	time.Sleep(400 * time.Millisecond)

	// The relay that held "a" died: "a" is claimable again, still before "b".
	if got, _ := ob.Claim(ctx, 10, time.Minute); !slices.Equal(ids(got), []string{"a", "b"}) {
		t.Errorf("after lease expiry: %v", ids(got))
	}
}

func testOutboxStats(t *testing.T, s store.Store) {
	t.Helper()

	rep, ok := s.(store.StatsReporter)
	if !ok {
		t.Skip("store does not report outbox stats")
	}

	ctx := t.Context()

	st, err := rep.OutboxStats(ctx)
	if err != nil || st.Pending != 0 || !st.Oldest.IsZero() {
		t.Fatalf("empty stats: %+v, %v", st, err)
	}

	created := time.Now().Add(-2 * time.Minute).UTC().Truncate(time.Millisecond)

	mustTx(t, s, func(ctx context.Context, tx store.Tx) error {
		return tx.AppendOutbox(ctx, store.OutboxEntry{
			ID:        "a",
			Data:      []byte("x"),
			CreatedAt: created,
		})
	})

	st, err = rep.OutboxStats(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if st.Pending != 1 || st.Oldest.Sub(created).Abs() > time.Second {
		t.Fatalf("stats %+v, created %s", st, created)
	}
}

// jsonEqual compares JSON semantically; backends such as Postgres JSONB do
// not preserve formatting.
func jsonEqual(data []byte, want string) bool {
	var a, b any

	return json.Unmarshal(data, &a) == nil && json.Unmarshal([]byte(want), &b) == nil &&
		reflect.DeepEqual(a, b)
}
