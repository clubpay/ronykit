package postgres

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/clubpay/ronykit/ee/store"
	"github.com/jackc/pgx/v5"
)

// TestIndexesMatchQueries guards the promise of GINIndex/FieldIndex: the
// planner must be able to use them for the SQL buildFind generates.
func TestIndexesMatchQueries(t *testing.T) {
	dsn := os.Getenv("EE_POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("EE_POSTGRES_TEST_DSN not set")
	}

	s := MustNew(WithPrimaryDSN(dsn))
	t.Cleanup(func() { _ = s.Close() })

	ctx := t.Context()
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		ddl  string
		cond store.Cond
	}{
		{GINIndex("idx-kind"), store.Cond{Field: "merchant", Op: store.Eq, Value: "m1"}},
		{FieldIndex("idx-kind", "merchant", Text), store.Cond{Field: "merchant", Op: store.Gt, Value: "m1"}},
		{FieldIndex("idx-kind", "amount", Numeric), store.Cond{Field: "amount", Op: store.Gte, Value: 10}},
		{FieldIndex("idx-kind", "amount", Numeric), store.Cond{Field: "amount", Op: store.In, Value: []any{1, 2}}},
		{GINIndex("idx-kind"), store.Cond{Field: "open", Op: store.Eq, Value: true}},
		{FieldIndex("idx-kind", "merchant", Text), store.Cond{Field: "merchant", Op: store.In, Value: []any{"m1", "m2"}}},
	}

	for _, c := range cases {
		err := pgx.BeginTxFunc(ctx, s.primary, pgx.TxOptions{}, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, c.ddl); err != nil {
				return err
			}

			// Many rows of one kind make the primary key (kind, id) useless
			// for selectivity, so only a matching index can win.
			_, err := tx.Exec(ctx, `
				INSERT INTO ee_states (kind, id, version, data)
				SELECT 'idx-kind', 'r' || i, 1,
				       jsonb_build_object('merchant', 'm' || i, 'amount', i, 'open', i % 100 = 0)
				FROM generate_series(1, 5000) AS i`)
			if err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, "ANALYZE ee_states"); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, "SET LOCAL enable_seqscan = off"); err != nil {
				return err
			}

			query, args, err := buildFind(store.Query{Kind: "idx-kind", Where: []store.Cond{c.cond}})
			if err != nil {
				return err
			}

			plan := explain(ctx, t, tx, query, args)
			name := c.ddl[len("CREATE INDEX IF NOT EXISTS "):strings.Index(c.ddl, " ON ")]

			if os.Getenv("EE_SHOW_PLANS") != "" {
				t.Logf("%s %s:\n%s", c.cond.Field, c.cond.Op, plan)
			}

			// The index must serve the condition itself, not just stand in for
			// "kind = ..." as a partial index.
			if !strings.Contains(plan, name) || !servesField(plan, c.cond.Field) {
				t.Errorf("%s %s is not served by %s:\n%s", c.cond.Field, c.cond.Op, name, plan)
			}

			return pgx.ErrTxClosed // roll back the index
		})
		if err != nil && err != pgx.ErrTxClosed {
			t.Fatal(err)
		}
	}
}

func explain(ctx context.Context, t *testing.T, tx pgx.Tx, query string, args []any) string {
	t.Helper()

	rows, err := tx.Query(ctx, "EXPLAIN "+query, args...)
	if err != nil {
		t.Fatal(err)
	}

	lines, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}

	return strings.Join(lines, "\n")
}

func servesField(plan, field string) bool {
	for line := range strings.Lines(plan) {
		if strings.Contains(line, "Index Cond:") && (strings.Contains(line, "'"+field+"'") || strings.Contains(line, `"`+field+`"`)) {
			return true
		}
	}

	return false
}
