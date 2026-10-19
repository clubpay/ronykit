// Package decide provides a generic, schema-driven decision and structured extraction
// wrapper over System 1 decision models (TypeSafe Jev, Laya) and OpenAI-compatible LLMs.
//
// Instead of cluttering Go structs with long struct tags, callers bind plain Go struct
// fields using field pointers in a builder callback (Option 1), implement Describer[T]
// on the struct (Option 2), or attach Criteria / Levels methods to reusable field types
// (Option 3).
//
// Inline Builder Example:
//
//	type Ticket struct {
//	    Department string
//	    Urgency    int
//	    Refund     bool
//	}
//
//	schema := decide.MustSchema(func(b *decide.Builder[Ticket], t *Ticket) {
//	    b.Choice(&t.Department, "Which team should handle this request?").
//	        Option("billing", "Invoices, payments, refunds").
//	        Option("tech", "Bugs, outages, integration issues")
//	    b.Score(&t.Urgency, "How urgent is this request?").
//	        Levels("Can wait", "Needs attention soon", "Critical blocker")
//	    b.Bool(&t.Refund, "Does the customer explicitly ask for a refund?")
//	}, decide.WithResolver(resolver))
//
//	res, err := schema.Decide(ctx, "We were billed twice! Please refund today.")
//	if err != nil {
//	    return err
//	}
//	ticket := res.Value()
//	deptProbs := res.FieldProbabilities(&res.ValuePtr().Department)
package decide
