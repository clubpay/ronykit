# decide

Generic, tag-free structured decision and extraction wrapper over System 1 decision engines (**TypeSafe Jev**, **Laya**) and **OpenAI-compatible LLMs**.

```go
import "github.com/clubpay/ronykit/x/decide"
```

## Overview

`x/decide` maps plain Go structs into System 1 question cards (`noul`, `choice`, `score`), evaluates them against an input `state`, and hydrates the answers back into a strongly-typed `*decide.Result[T]` carrying both the populated struct and per-field probability/confidence distributions.

It supports three composable ways to define a schema without struct tags:

1. **Inline field-pointer builder** (`decide.MustSchema(func(b *decide.Builder[T], t *T) { ... })`)
2. **Self-describing structs** (`func (t *MyStruct) Describe(b *decide.Builder[MyStruct])` + `decide.MustSchemaFor[MyStruct]()`)
3. **Self-contained enum and rubric types** (`Criteria() map[E]string`, `Options() []E`, `Levels() []string`)

## Quick Start

### 1. Inline Builder

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

### 2. Self-Contained Types & Self-Describing Struct

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

### 3. Template-Driven State Generation

Use `decide.MustTemplate[IN]` (or one-shot `decide.TemplateState`) to render structured Go input into a controlled text state using `text/template` before sending it to the resolver:

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

Built-in template functions: `json`, `trim`, `upper`, `lower`, `join`, `snake`, `camel`, `kebab`, and `default`. Custom functions can be added via `decide.WithTemplateFuncs(...)`.

## Resolvers

| Resolver | Constructor | Wire Protocol |
|---|---|---|
| **TypeSafe Jev** | `decide.NewJevResolver` / `decide.MustNewJevResolver` | `POST /v1/systemone` (`https://api.typesafe.ai/v1/systemone`) |
| **Laya** | `decide.NewLayaResolver` / `decide.MustNewLayaResolver` | `POST /v1/systemone` + native batch `POST /v1/systemone/batch` |
| **LLM** | `decide.NewLLMResolver` / `decide.MustNewLLMResolver` | OpenAI-compatible `POST /chat/completions` (`json_object`) |
| **Pass / Static** | `decide.NewPassResolver` / `decide.NewStaticResolver` | Deterministic in-memory resolvers for tests |
