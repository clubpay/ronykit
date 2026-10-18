package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"reflect"
	"strings"
	"time"

	"github.com/clubpay/ronykit/ee/store"
	"github.com/clubpay/ronykit/x/rkit"

	"github.com/jackc/pgx/v5"
)

// FindFunc replaces the generated Find query for one kind, e.g. with a
// hand-written or sqlc-generated query tuned to that kind's indexes. It runs
// in the unit of work's transaction and must yield records of that kind.
type FindFunc func(ctx context.Context, tx pgx.Tx, q store.Query) iter.Seq2[store.Record, error]

// WithFind installs fn as the Find implementation for kind.
func WithFind(kind string, fn FindFunc) Option {
	return func(c *config) { c.finders[kind] = fn }
}

// find streams matching records through a server-side cursor, fetching
// fetchSize rows at a time. Between fetches the connection is free, so the
// caller may write in the same transaction while iterating.
func (t *tx) find(ctx context.Context, q store.Query) iter.Seq2[store.Record, error] {
	return func(yield func(store.Record, error) bool) {
		query, args, err := buildFind(q)
		if err != nil {
			yield(store.Record{}, err)

			return
		}

		t.cursors++
		cursor := "ee_find_" + rkit.IntToStr(t.cursors)

		if _, err := t.ptx.Exec(ctx, "DECLARE "+cursor+" NO SCROLL CURSOR FOR "+query, args...); err != nil {
			yield(store.Record{}, mapError(err))

			return
		}

		defer func() { _, _ = t.ptx.Exec(ctx, "CLOSE "+cursor) }()

		fetch := "FETCH FORWARD " + rkit.IntToStr(t.s.fetchSize) + " FROM " + cursor

		for {
			batch, err := t.fetch(ctx, fetch)
			if err != nil {
				yield(store.Record{}, err)

				return
			}

			for _, r := range batch {
				if !yield(r, nil) {
					return
				}
			}

			if len(batch) < t.s.fetchSize {
				return
			}
		}
	}
}

func (t *tx) fetch(ctx context.Context, sql string) ([]store.Record, error) {
	rows, err := t.ptx.Query(ctx, sql)
	if err != nil {
		return nil, mapError(err)
	}

	recs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (store.Record, error) {
		var (
			r       store.Record
			version int64
		)

		err := row.Scan(&r.Kind, &r.ID, &version, &r.Data)
		r.Version = uint64(version)

		return r, err
	})

	return recs, mapError(err)
}

// buildFind renders q as SQL. Kind and field names are inlined as escaped
// literals so Postgres can match partial and expression indexes (see
// GINIndex and FieldIndex); values are always bind parameters.
func buildFind(q store.Query) (string, []any, error) {
	var (
		b    strings.Builder
		args []any
	)

	b.WriteString("SELECT kind, id, version, data FROM ee_states WHERE kind = ")
	b.WriteString(literal(q.Kind))

	for _, c := range q.Where {
		expr, err := condition(c, &args)
		if err != nil {
			return "", nil, err
		}

		b.WriteString(" AND ")
		b.WriteString(expr)
	}

	b.WriteString(" ORDER BY ")

	for _, o := range q.OrderBy {
		b.WriteString("data->")
		b.WriteString(literal(o.Field))

		if o.Desc {
			b.WriteString(" DESC")
		}

		b.WriteString(", ")
	}

	b.WriteString("id")

	if q.Limit > 0 {
		b.WriteString(" LIMIT ")
		b.WriteString(rkit.IntToStr(q.Limit))
	}

	return b.String(), args, nil
}

// valueKind groups Go values by how they compare in SQL.
type valueKind int

const (
	kindOther valueKind = iota
	kindText
	kindNumeric
	kindBool
	kindTime
)

func kindOf(v any) valueKind {
	if _, ok := v.(time.Time); ok {
		return kindTime
	}

	switch rv := reflect.ValueOf(v); {
	case rv.Kind() == reflect.String:
		return kindText
	case rv.CanInt(), rv.CanUint(), rv.CanFloat():
		return kindNumeric
	case rv.Kind() == reflect.Bool:
		return kindBool
	default:
		return kindOther
	}
}

var sqlOps = map[store.Op]string{
	store.Eq: "=", store.Neq: "<>", store.Gt: ">", store.Gte: ">=", store.Lt: "<", store.Lte: "<=",
}

func condition(c store.Cond, args *[]any) (string, error) {
	if c.Op == store.In {
		return inCondition(c, args)
	}

	op, ok := sqlOps[c.Op]
	if !ok {
		return "", fmt.Errorf("postgres: unsupported operator %q", c.Op)
	}

	k := kindOf(c.Value)

	// Equality on anything but time uses JSONB containment, which a GIN
	// index (GINIndex) serves.
	if c.Op == store.Eq && k != kindTime {
		doc, err := json.Marshal(map[string]any{c.Field: c.Value})
		if err != nil {
			return "", fmt.Errorf("postgres: %s: %w", c.Field, err)
		}

		return "data @> " + param(args, string(doc)) + "::jsonb", nil
	}

	if k == kindOther {
		return "", fmt.Errorf(
			"postgres: %s: operator %s needs a string, number, bool or time, got %T",
			c.Field,
			c.Op,
			c.Value,
		)
	}

	return fieldExpr(c.Field, k) + " " + op + " " + param(args, sqlValue(k, c.Value)) + castOf(k), nil
}

func inCondition(c store.Cond, args *[]any) (string, error) {
	vals, ok := c.Value.([]any)
	if !ok {
		return "", fmt.Errorf("postgres: %s: In expects []any, got %T", c.Field, c.Value)
	}

	if len(vals) == 0 {
		return "false", nil
	}

	k := kindOf(vals[0])
	if k == kindOther {
		return "", fmt.Errorf(
			"postgres: %s: In needs strings, numbers, bools or times, got %T",
			c.Field,
			vals[0],
		)
	}

	list := make([]any, len(vals))
	for i, v := range vals {
		if kindOf(v) != k {
			return "", fmt.Errorf("postgres: %s: In mixes %T and %T", c.Field, vals[0], v)
		}

		list[i] = sqlValue(k, v)
	}

	return fieldExpr(c.Field, k) + " = ANY(" + param(args, list) + castOf(k) + "[])", nil
}

// fieldExpr extracts field as a value of kind k. Non-text casts are guarded
// by jsonb_typeof so a row whose field has another JSON type yields NULL
// (no match) instead of a cast error.
func fieldExpr(field string, k valueKind) string {
	text := "(data->>" + literal(field) + ")"
	typeIs := func(t string) string { return "jsonb_typeof(data->" + literal(field) + ") = '" + t + "'" }

	switch k {
	case kindNumeric:
		return "(CASE WHEN " + typeIs("number") + " THEN " + text + "::numeric END)"
	case kindBool:
		return "(CASE WHEN " + typeIs("boolean") + " THEN " + text + "::boolean END)"
	case kindTime:
		return "(CASE WHEN " + typeIs("string") + " THEN " + text + "::timestamptz END)"
	default:
		return text
	}
}

func castOf(k valueKind) string {
	switch k {
	case kindNumeric:
		return "::numeric"
	case kindBool:
		return "::boolean"
	case kindTime:
		return "::timestamptz"
	default:
		return "::text"
	}
}

// sqlValue converts v to what pgx binds for kind k. Numbers travel as text
// to keep uint64 and float precision through ::numeric.
func sqlValue(k valueKind, v any) any {
	switch k {
	case kindNumeric:
		return fmt.Sprint(v)
	case kindText:
		return reflect.ValueOf(v).String()
	case kindBool:
		return reflect.ValueOf(v).Bool()
	default:
		return v
	}
}

func param(args *[]any, v any) string {
	*args = append(*args, v)

	return "$" + rkit.IntToStr(len(*args))
}

// literal quotes s as a SQL string literal.
func literal(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// FieldKind selects the expression FieldIndex indexes; it must match the Go
// type of the field the queries compare.
type FieldKind int

const (
	Text    FieldKind = FieldKind(kindText)
	Numeric FieldKind = FieldKind(kindNumeric)
)

// GINIndex returns DDL for a GIN index serving equality conditions
// (ee.Where(field, value), any value type but time) on every field of kind.
func GINIndex(kind string) string {
	return "CREATE INDEX IF NOT EXISTS " + indexName("gin", kind, "") +
		" ON ee_states USING gin (data jsonb_path_ops) WHERE kind = " + literal(kind)
}

// FieldIndex returns DDL for a B-tree index serving Gt/Gte/Lt/Lte/In
// conditions on one text or numeric field of kind. Field is the encoded
// (JSON) name. Equality is served by GINIndex instead; time fields cannot be
// indexed this way because their cast is not immutable.
func FieldIndex(kind, field string, k FieldKind) string {
	return "CREATE INDEX IF NOT EXISTS " + indexName("f", kind, field) +
		" ON ee_states (" + fieldExpr(field, valueKind(k)) + ") WHERE kind = " + literal(kind)
}

func indexName(prefix, kind, field string) string {
	clean := func(s string) string {
		return strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
				return r
			}

			return '_'
		}, strings.ToLower(s))
	}

	name := "ee_" + prefix + "_" + clean(kind)
	if field != "" {
		name += "_" + clean(field)
	}

	if len(name) > 63 { // Postgres identifier limit
		name = name[:63]
	}

	return name
}
