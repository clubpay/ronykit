package store

// Op is a comparison operator usable in queries.
type Op string

const (
	Eq  Op = "eq"
	Neq Op = "neq"
	Gt  Op = "gt"
	Gte Op = "gte"
	Lt  Op = "lt"
	Lte Op = "lte"
	// In expects Value to be a []any.
	In Op = "in"
)

// Cond compares one top-level field of the encoded state. Field is the
// encoded name (e.g. the JSON key), already resolved by ee, and Value has the
// Go type of that field.
type Cond struct {
	Field string
	Op    Op
	Value any
}

type Order struct {
	Field string
	Desc  bool
}

// Query selects records of one kind. Conditions are ANDed. How it executes
// (JSONB, secondary sets, full scan) is up to the backend.
type Query struct {
	Kind    string
	Where   []Cond
	OrderBy []Order
	// Limit caps the number of records; 0 means no limit.
	Limit int
}
