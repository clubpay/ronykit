# decide

The `decide` package provides generic structured decisions and data extraction without struct tags. It supports System 1 decision engines, including **TypeSafe Jev** and **Laya**, and OpenAI-compatible LLMs.

```go
import "github.com/clubpay/ronykit/x/decide"
```

## Overview

`x/decide` maps plain Go structs to System 1 question cards: `noul`, `choice`, and `score`. It evaluates these cards against an input `state`.

The package returns a typed `*decide.Result[T]`. The result contains the populated struct and probability or confidence distributions for each field.

Define a schema without struct tags in three composable ways:

1. **Inline field-pointer builder:** `decide.MustSchema(func(b *decide.Builder[T], t *T) { ... })`
2. **Self-describing structs:** `func (t *MyStruct) Describe(b *decide.Builder[MyStruct])` with `decide.MustSchemaFor[MyStruct]()`
3. **Self-contained enum and rubric types:** `Criteria() map[E]string`, `Options() []E`, and `Levels() []string`

## Quick start

### 1. Inline builder

```go
type Ticket struct {
	Department string
	Urgency    int
	Refund     bool
	ChurnProb  float64
}

schema := decide.MustSchema(func(b *decide.Builder[Ticket], t *Ticket) {
	b.Choice(&t.Department, "Which team should handle this request?").
		Option("billing", "Invoices, payments, refunds").
		Option("tech", "Bugs, outages, integration issues").
		Option("sales", "Pricing, new contracts")

	b.Score(&t.Urgency, "How urgent is this request?").
		Levels("Can wait", "Needs attention soon", "Critical blocker")

	b.Bool(&t.Refund, "Does the customer explicitly request a refund?")
	b.Prob(&t.ChurnProb, "The customer threatens to cancel their account.")
}, decide.WithResolver(resolver))

res, err := schema.Decide(ctx, "We were billed twice! Refund today or we cancel.")
if err != nil {
	return err
}

ticket := res.Value()
deptProbs := res.ProbabilitiesOf(&res.Ptr().Department)
deptConf := res.ConfidenceOf(&res.Ptr().Department)
```

### 2. Self-contained types and self-describing struct

```go
type Department string

const (
	DeptBilling Department = "billing"
	DeptTech    Department = "tech"
)

func (Department) Criteria() map[Department]string {
	return map[Department]string{
		DeptBilling: "Invoices, payments, refunds",
		DeptTech:    "Bugs, outages, integration issues",
	}
}

type Priority int

func (Priority) Levels() []string {
	return []string{"Low", "Medium", "High"}
}

func (Priority) MinLevel() int { return 1 }

type Ticket struct {
	Department Department
	Priority   Priority
	Refund     bool
}

func (t *Ticket) Describe(b *decide.Builder[Ticket]) {
	b.Choice(&t.Department, "Which team should handle this request?")
	b.Score(&t.Priority, "How urgent is this request?")
	b.Bool(&t.Refund, "Does the customer explicitly request a refund?")
}

schema := decide.MustSchemaFor[Ticket](decide.WithResolver(resolver))
```

### 3. Template-driven state generation

Use `decide.MustTemplate[IN]` to render structured Go input as controlled text state. `decide.TemplateState` performs the same operation in one call. The package uses `text/template` before it sends state to the resolver.

```go
type TicketContext struct {
	Customer string
	Plan     string
	Subject  string
	Message  string
	Tags     []string
}

stateTmpl := decide.MustTemplate[TicketContext](`
Customer: {{ upper .Customer }} (Plan: {{ default "free" .Plan }})
Subject:  {{ trim .Subject }}
Tags:     {{ join .Tags ", " }}
---
{{ trim .Message }}
`)

res, err := schema.Decide(ctx, stateTmpl.State(TicketContext{
	Customer: "Acme Corp",
	Plan:     "enterprise",
	Subject:  "Duplicate charge on March invoice",
	Message:  "We were billed twice! Please refund today.",
	Tags:     []string{"billing", "duplicate"},
}))

// Or for batch execution:
batchRes, err := schema.DecideBatch(ctx, stateTmpl.States(ticketContexts))
```

Built-in template functions are `json`, `trim`, `upper`, `lower`, `join`, `snake`, `camel`, `kebab`, and `default`. Add custom functions with `decide.WithTemplateFuncs(...)`.

## Resolvers

| Resolver | Constructor | Wire protocol |
|----------|-------------|---------------|
| **TypeSafe Jev** | `decide.NewJevResolver` or `decide.MustNewJevResolver` | `POST /v1/systemone` at `https://api.typesafe.ai/v1/systemone` |
| **Laya** | `decide.NewLayaResolver` or `decide.MustNewLayaResolver` | `POST /v1/systemone` and native batch `POST /v1/systemone/batch` |
| **LLM** | `decide.NewLLMResolver` or `decide.MustNewLLMResolver` | OpenAI-compatible `POST /chat/completions` with `json_object` |
| **Pass or Static** | `decide.NewPassResolver` or `decide.NewStaticResolver` | Deterministic in-memory resolvers for tests |
