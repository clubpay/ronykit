package postgres_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/clubpay/ronykit/ee"
	"github.com/clubpay/ronykit/ee/mem"
	"github.com/clubpay/ronykit/ee/store"
	"github.com/clubpay/ronykit/ee/store/storetest"
	"github.com/clubpay/ronykit/std/ee/store/postgres"
	"github.com/jackc/pgx/v5"
)

// The tests need a disposable database; they truncate the ee tables.
//
//	EE_POSTGRES_TEST_DSN=postgres://ee:ee@localhost:55432/ee?sslmode=disable
//	EE_POSTGRES_TEST_REPLICA_DSN=postgres://ee:ee@localhost:55432/ee_replica?sslmode=disable
//
// The "replica" may be any other database: the routing test only needs reads
// to land somewhere the primary's writes are not.
const (
	envDSN        = "EE_POSTGRES_TEST_DSN"
	envReplicaDSN = "EE_POSTGRES_TEST_REPLICA_DSN"
)

func newStore(t *testing.T, opts ...postgres.Option) *postgres.Store {
	t.Helper()

	dsn := os.Getenv(envDSN)
	if dsn == "" {
		t.Skip(envDSN + " not set")
	}

	s, err := postgres.New(append([]postgres.Option{postgres.WithPrimaryDSN(dsn)}, opts...)...)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = s.Close() })

	if err := s.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Primary().Exec(t.Context(), "TRUNCATE ee_states, ee_outbox"); err != nil {
		t.Fatal(err)
	}

	return s
}

func TestConformance(t *testing.T) {
	storetest.Run(t, func(t *testing.T) store.Store { return newStore(t, postgres.WithFetchSize(7)) })
}

type Order struct {
	ID       string `ee:"id" json:"id"`
	Merchant string `json:"merchant"`
	Amount   int    `json:"amount"`
	Status   string `json:"status"`
}

func (Order) EntityName() string { return "order" }

type OrderPaid struct{ OrderID string }

func (OrderPaid) EventName() string { return "order.paid" }
func (e OrderPaid) Key() string     { return e.OrderID }

type Receipt struct {
	ID string `ee:"id" json:"id"`
}

func (Receipt) EntityName() string { return "receipt" }

type ReceiptRequested struct{ OrderID string }

func (ReceiptRequested) EventName() string { return "receipt.requested" }
func (e ReceiptRequested) Key() string     { return e.OrderID }

func TestEndToEnd(t *testing.T) {
	s := newStore(t)
	b := mem.NewBus(mem.RetryBackoff(time.Millisecond))
	t.Cleanup(func() { _ = b.Close() })

	env := ee.MustNew(ee.WithStore(s), ee.WithBus(b), ee.WithMaxRetries(100), ee.WithMiddleware(ee.Dedup))
	orders, _ := ee.NewEntityController[Order](env)
	receipts, _ := ee.NewEntityController[Receipt](env)

	_ = orders.Action(func(ctx *ee.ActionCtx[Order, OrderPaid]) error {
		o, err := ctx.Entity()
		if err != nil {
			return err
		}

		o.Status = "paid"
		if err := ctx.Save(o); err != nil {
			return err
		}

		// sqlc/pgx code joins the same transaction and sees the pending write.
		ptx, ok := postgres.PgxTx(ctx.Tx())
		if !ok {
			return errors.New("not a postgres tx")
		}

		var status string
		if err := ptx.QueryRow(ctx, `SELECT data->>'status' FROM ee_states WHERE kind = 'order' AND id = $1`, o.ID).Scan(&status); err != nil {
			return err
		}

		if status != "paid" {
			return errors.New("raw query does not see the action's write: " + status)
		}

		return ctx.Emit(ReceiptRequested{OrderID: o.ID})
	})
	_ = receipts.Action(func(ctx *ee.ActionCtx[Receipt, ReceiptRequested]) error {
		return ctx.Create(&Receipt{ID: "r-" + ctx.Event().OrderID})
	})

	if err := env.Start(t.Context()); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = env.Close() })

	ctx := t.Context()
	if err := orders.Create(ctx, Order{ID: "o1", Merchant: "m1"}); err != nil {
		t.Fatal(err)
	}

	_ = env.Send(ctx, OrderPaid{OrderID: "o1"})

	eventually(t, "receipt r-o1", func() bool {
		_, err := receipts.Get(ctx, "r-o1")

		return err == nil
	})

	// Concurrent read-modify-writes on one row: no lost updates.
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if err := orders.Update(ctx, "o1", func(o *Order) error { o.Amount++; return nil }); err != nil {
				t.Error(err)
			}
		})
	}

	wg.Wait()

	if o, _ := orders.Get(ctx, "o1"); o.Amount != 20 || o.Status != "paid" {
		t.Errorf("order = %+v", o)
	}

	n := 0
	for o, err := range orders.Find(ctx, ee.Where("Merchant", "m1").And("Amount", ee.Gte, 20)) {
		if err != nil {
			t.Fatal(err)
		}

		n++

		_ = o
	}

	if n != 1 {
		t.Errorf("find: %d results", n)
	}
}

func TestReplicaRouting(t *testing.T) {
	replica := os.Getenv(envReplicaDSN)
	if replica == "" {
		t.Skip(envReplicaDSN + " not set")
	}

	s := newStore(t, postgres.WithReplicaDSNs(replica))

	// Give the "replica" the schema, but none of the primary's data.
	rs := postgres.MustNew(postgres.WithPrimaryDSN(replica))
	t.Cleanup(func() { _ = rs.Close() })

	if err := rs.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}

	_, _ = rs.Primary().Exec(t.Context(), "TRUNCATE ee_states, ee_outbox")

	if err := s.Check(t.Context()); err != nil {
		t.Fatal(err)
	}

	env := ee.MustNew(ee.WithStore(s))
	orders, _ := ee.NewEntityController[Order](env)
	ctx := t.Context()

	if err := orders.Create(ctx, Order{ID: "o1"}); err != nil {
		t.Fatal(err)
	}

	if _, err := orders.Get(ctx, "o1"); !errors.Is(err, ee.ErrNotFound) {
		t.Errorf("Get must read from the replica, got %v", err)
	}

	if _, err := orders.Get(postgres.ForcePrimary(ctx), "o1"); err != nil {
		t.Errorf("ForcePrimary must read the primary: %v", err)
	}

	// Writes always go to the primary, even through Update's read.
	if err := orders.Update(ctx, "o1", func(o *Order) error { o.Amount = 1; return nil }); err != nil {
		t.Errorf("Update must use the primary: %v", err)
	}
}

func TestMigrationsAndIndexes(t *testing.T) {
	applied := 0
	s := newStore(t, postgres.WithMigration("001-order-indexes", func(ctx context.Context, tx pgx.Tx) error {
		applied++

		for _, ddl := range []string{
			postgres.GINIndex("order"),
			postgres.FieldIndex("order", "merchant", postgres.Text),
			postgres.FieldIndex("order", "amount", postgres.Numeric),
		} {
			if _, err := tx.Exec(ctx, ddl); err != nil {
				return err
			}
		}

		return nil
	}))

	// newStore migrated once; a second run must not re-apply.
	if err := s.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}

	if err := s.Check(t.Context()); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := s.Primary().QueryRow(t.Context(),
		`SELECT count(*) FROM pg_indexes WHERE tablename = 'ee_states' AND indexname LIKE 'ee\_%\_order%'`).Scan(&n); err != nil {
		t.Fatal(err)
	}

	if applied > 1 || n != 3 {
		t.Errorf("applied %d times, %d indexes", applied, n)
	}

	unmigrated := postgres.MustNew(
		postgres.WithPrimary(s.Primary()),
		postgres.WithMigration("002-never-run", func(context.Context, pgx.Tx) error { return nil }),
	)
	if err := unmigrated.Check(t.Context()); err == nil {
		t.Error("Check must report pending migrations")
	}
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()

	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		if cond() {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", what)
}
