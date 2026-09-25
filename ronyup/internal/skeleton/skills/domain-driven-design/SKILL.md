---
name: domain-driven-design
description: >-
  Model software around the business domain — bounded contexts, aggregates,
  entities vs value objects, domain events, and ubiquitous language. Use when
  writing an SRS/SDD, deciding how to split features into RonyKit service
  modules, designing `internal/domain` types, placing invariants, integrating
  with another service or external API (anti-corruption layer), or when "the
  code doesn't match the business". For layer placement (api/app/repo), see
  go-design. For module/interface depth, see
  software-design-philosophy. For storage and consistency trade-offs, see
  ddia-systems.
license: MIT
metadata:
  author: wondelai
  version: "1.4.0"
---

# Domain-Driven Design Framework

Framework for tackling software complexity by modeling code around the business domain. The greatest risk in software is not technical failure -- it is building a model that does not reflect how the business actually works.

## When to use

- Writing the SRS glossary (§1.3) or the SDD domain design (§4) for a new
  feature.
- Deciding whether a capability is a new feature module or belongs in an
  existing one.
- Designing `internal/domain` entities, value objects, enums, and errors.
- A handler, app method, or SQL query enforces a rule the domain type should
  own (anemic model).
- Calling another service or a third-party API and its model starts leaking
  into yours.

## RonyKit mapping (scaffolded apps)

| DDD concept | RonyKit home | Rule |
| ----------- | ------------ | ---- |
| Bounded context | One feature module (`feature/<name>`, package `<name>mod`) | Own contract, settings, schema, and ubiquitous language. Split modules by model boundary, not by table or team size. |
| Ubiquitous language | SRS glossary → SDD → Go identifiers, contract names, SQL names | A term in the SRS appears verbatim in `internal/domain` and the API contract. |
| Entity / aggregate root | `internal/domain` struct with guarded methods | Invariant-backing fields unexported; constructors return `(T, error)`. |
| Value object | `internal/domain` type with validating constructor | Immutable, compared by value (`Money`, `Email`, type-safe enum structs). |
| Repository | `internal/repo/port.go` (interface) + `internal/repo/v0` (sqlc) | Methods speak the domain (`ListOverdueInvoices`), return domain types — never sqlc rows. One interface per aggregate / domain concept. |
| Application service | `internal/app` | Orchestrates aggregates and ports; never re-implements invariants. |
| Domain errors | `internal/domain/errors.go` via `rony/errs` | Named after business outcomes (`ErrInsufficientFunds`). |
| Anti-corruption layer | Generated stub (`di.StubProvider`) wrapped by an app-side port | Translate the other service's DTOs into your domain types at the boundary. |
| Open host / published language | Rony contract + `make gen-stub` + `x/apidoc` | Your contract is the published language; version it deliberately. |
| Domain events / long processes | `flow` workflows (`characteristics/workflow`) | Cross-aggregate or cross-service processes; never import the Temporal SDK directly. |

Read MCP `architecture/domain-layer`, `architecture/repo-ports`, and
`architecture/inter-service-stubs` before implementing; record the model in the
SDD (`architecture/design-documents`).

## Core Principle

**The model is the code; the code is the model.** Software should embody a deep, shared understanding of the business domain. When domain experts and developers speak the same language and that language is directly expressed in the codebase, complexity becomes manageable and the system evolves gracefully as the business changes.

## Scoring

**Goal: 10/10.** Score a domain model by awarding **1 point per satisfied row of the Quick Diagnostic** (7 rows) plus up to 3 points for depth: +1 if the Core Domain has a genuinely rich model (not just CRUD), +1 if invariants live inside aggregates rather than in services, +1 if the ubiquitous language is consistent across conversation, code, and tests. Bands: **9-10** = expert-readable names, explicit context boundaries with ACLs, small aggregates, behavior-rich entities, events for cross-aggregate flow, an identified Core Domain; **5-6** = some domain language but leaky boundaries or anemic objects; **<=3** = technical naming, one model for everything, logic scattered in services. Report the score and the specific diagnostic rows failing.

## Framework

### 1. Ubiquitous Language

**Core concept:** A shared, rigorous language between developers and domain experts, used consistently in conversation, documentation, and code. When the language changes, the code changes -- and awkward naming in code feeds back into refining the language.

**Why it works:** Ambiguity is the root cause of most modeling failures. When a developer says "order" and an expert means "purchase request," bugs are inevitable; a ubiquitous language forces every name in code to map to a concept the business recognizes and validates.

**Key insights:**
- The language emerges from deep collaboration, not a glossary bolted on after the fact
- If a concept is hard to name, the model is likely wrong -- naming difficulty is a design signal
- Technical jargon (`DataProcessor` vs. `ClaimAdjudicator`) hides domain logic from the experts who could correct it
- Different bounded contexts may use the same word with different meanings -- and that is fine

**Code applications:**

| Context | Pattern | Example |
|---------|---------|---------|
| Type/method naming | Name after domain concepts and verbs | `domain.LoanApplication`, `policy.Underwrite(app)` -- not `RequestHandler`, `Process()` |
| Module structure | Organize by domain concept | `feature/shipping`, `feature/billing` -- not `controllers/`, `services/`, `utils/` |
| Code review | Reject technical-only names | Flag `Manager`, `Helper`, `Processor`, `Util`, and `common` packages as naming smells |

See: [references/ubiquitous-language.md](references/ubiquitous-language.md) when running modeling sessions or maintaining a glossary -- covers how the language evolves and feeds back into code.

### 2. Bounded Contexts and Context Mapping

**Core concept:** A bounded context is an explicit boundary within which a particular domain model applies. The same word ("Customer") can mean different things in different contexts; context maps define the relationships and translation strategies between them.

**Why it works:** Large systems that try to maintain a single unified model inevitably collapse into inconsistency. Bounded contexts accept that different parts of the business need different models; context maps manage the integration between them.

**Key insights:**
- A bounded context is not a microservice -- it is a linguistic and model boundary. RonyKit fits this well: each context is a feature module, and bundles (`cmd/all-in-one` or smaller) decide which modules run in one process
- Context boundaries often align with team boundaries (Conway's Law)
- The nine context mapping patterns describe political and technical relationships between teams
- Anti-Corruption Layer is the most important defensive pattern -- never let a foreign model leak into your core domain
- Shared Kernel couples two teams; keep it small and explicitly governed
- Start by mapping what exists (Big Ball of Mud), then define target boundaries

**Code applications:**

| Context | Pattern | Example |
|---------|---------|---------|
| Service integration | Anti-Corruption Layer | Wrap the other feature's generated stub (or a third-party SDK) behind an app-side interface that returns your `domain` types |
| Legacy migration | Conformist / ACL | Wrap the legacy system behind an adapter that speaks your domain language |
| API design | Open Host Service + Published Language | Your Rony contract, published as generated Go/TS stubs and `x/apidoc` Swagger |

See: [references/bounded-contexts.md](references/bounded-contexts.md) for the nine mapping patterns and integration strategies.

### 3. Entities, Value Objects, and Aggregates

**Core concept:** Entities have identity that persists across state changes. Value Objects are defined entirely by their attributes and are immutable. Aggregates are clusters of entities and value objects with a single root that enforces consistency boundaries.

**Why it works:** Without these distinctions, everything becomes a mutable, identity-bearing object -- tangled state, inconsistent updates, fragile concurrency. Aggregates draw the line: everything inside is guaranteed consistent; everything outside is eventually consistent.

**Key insights:**
- Entity test: "Am I the same thing even if all my attributes change?" (a person changes name and address -- still the same person)
- Value Object test: "Am I defined only by my attributes?" (any $10 bill is interchangeable with another)
- Most things should be Value Objects, not Entities -- prefer immutability
- Keep aggregates small (one root plus a minimal cluster); reference other aggregates by ID, not object reference
- Immediate consistency only within an aggregate; design for eventual consistency between aggregates

**Code applications:**

| Context | Pattern | Example |
|---------|---------|---------|
| Identity tracking | Entity with ID | `domain.Order` with unexported `id` (from `rkit.RandomID`) and an `ID()` getter; survives state changes |
| Immutable attributes | Value Object | `NewAddress(street, city, zip) (Address, error)`; value receiver methods, no setters -- replace, never mutate |
| Consistency boundary | Aggregate Root | `Order` is root; `OrderLine` changes go through `o.AddLine(...) error`; `Lines()` returns a copy |
| Concurrency control | Optimistic locking on root | `version` column on `orders`; `UPDATE ... WHERE id = $1 AND version = $2` returns a conflict error if two edits race |

See: [references/building-blocks.md](references/building-blocks.md) for aggregate design rules and consistency boundaries.

### 4. Domain Events

**Core concept:** A domain event captures something that happened in the domain that experts care about, named in past tense (`OrderPlaced`, `PaymentReceived`) -- a fact that has already occurred.

**Why it works:** Domain events decouple cause from effect. When `OrderPlaced` is published, shipping, billing, and notifications each react independently without the ordering context knowing about them -- less coupling, eventual consistency, a natural audit trail.

**Key insights:**
- Events are immutable facts -- once published, they cannot be changed or retracted
- Domain events are internal to a bounded context; integration events cross boundaries
- Events enable temporal decoupling: the producer does not wait for the consumer
- Event sourcing stores the full event history as the source of truth, deriving current state by replay
- Not every state change deserves an event -- only publish what the domain cares about

**Code applications:**

| Context | Pattern | Example |
|---------|---------|---------|
| State transitions | Record event on domain action | `o.Place(now) (OrderPlaced, error)` returns a past-tense value type; the `App` method acts on it in the same use case |
| Cross-context integration | Durable reaction in another context | A `flow` workflow started after `OrderPlaced` calls the shipping feature's stub to request a label |
| Eventual consistency | Async handlers | Stock reservation runs as a `flow` activity with retries; RonyKit has no built-in event bus, so don't invent one — add an outbox table if you introduce a broker |

See: [references/domain-events.md](references/domain-events.md) for event naming, event sourcing, and integration events.

### 5. Repositories and Factories

**Core concept:** Repositories provide the illusion of an in-memory collection of domain objects, hiding persistence. Factories encapsulate complex creation logic so aggregates are always born in a valid state.

**Why it works:** When persistence and assembly details leak into domain code, every storage change ripples through business rules and aggregates can be constructed in half-valid states. Repositories confine SQL concerns to the adapter so the domain stays testable in memory; factories make the only path to an aggregate one that enforces its invariants, so an invalid instance is unrepresentable.

**Key insights:**
- Evans puts the Repository interface in the domain layer. RonyKit puts it in `internal/repo/port.go`, a package that depends only on `internal/domain`, with the sqlc implementation in `internal/repo/v0`. The dependency still points inward, which is what the rule protects (layer placement: `go-design`)
- Repository methods speak the ubiquitous language: `ListPending(ctx)`, not `GetByStatusCode(ctx, 3)`; they take and return domain types, never sqlc rows
- Collection-oriented repositories mimic `Add`/`Remove`; persistence-oriented ones use `Save` — RonyKit ports are usually persistence-oriented (`Create`, `Update`, `Get`)
- Factories are warranted for complex rules or multi-part assembly; most of the time a validating `NewX(...) (X, error)` constructor is the factory
- The Specification pattern encapsulates query criteria as domain objects; in Go, a plain filter struct passed to one port method (`ListInvoices(ctx, InvoiceFilter{OverdueBy: 30 * 24 * time.Hour})`) is usually enough — don't build a generic specification framework

**Code applications:**

| Context | Pattern | Example |
|---------|---------|---------|
| Data access abstraction | Repository interface | `repo.OrderRepository.ListByCustomer(ctx, customerID)` in `port.go`; sqlc-backed `v0repo.orderRepository` implements it |
| Complex creation | Factory function | `domain.NewOrderFromQuote(q Quote, now time.Time) (Order, error)` validates and assembles from a `Quote` |
| Query encapsulation | Filter struct | `InvoiceFilter{Status: domain.InvoiceStatusOpen, DueBefore: cutoff}` passed to `ListInvoices` |

See: [references/repositories-factories.md](references/repositories-factories.md) for Repository, Factory, and Specification patterns.

### 6. Strategic Design and Distillation

**Core concept:** Not all parts of a system are equally important. Strategic design identifies the Core Domain -- where competitive advantage lives -- and distinguishes it from Supporting Subdomains (necessary, not differentiating) and Generic Subdomains (commodity).

**Why it works:** Applying the same rigor everywhere spreads your best talent thin and over-engineers commodity functionality. Identifying the Core Domain concentrates the best developers and deepest modeling where they matter most.

**Key insights:**
- Core Domain: invest your best people and deepest modeling; Supporting: build, but don't over-engineer; Generic (auth, email, payments): buy or use open-source
- Distillation extracts and highlights the Core Domain from surrounding complexity
- A Domain Vision Statement is a one-page description of the Core Domain's value proposition
- Revisit what is "core" as the business evolves -- today's differentiator may become tomorrow's commodity

**Code applications:**

| Context | Pattern | Example |
|---------|---------|---------|
| Build vs. buy | Classify subdomain type | Build the custom pricing engine (core); use a provider for email/SMS (generic). Payments are generic for most products — but core if payments are what you sell |
| Team allocation | Best developers on Core Domain | Seniors model underwriting rules; juniors integrate the email service |
| Code organization | Separate core from generic | A rich `feature/pricing` module with a deep `internal/domain` vs. a thin notification adapter; reuse RonyKit `x/*` packages (rate limiting, settings, i18n) instead of building generic subdomains |

See: [references/strategic-design.md](references/strategic-design.md) when deciding where to invest engineering effort -- subdomain classification and distillation techniques.

## Common Mistakes

| Mistake | Why It Fails | Fix |
|---------|-------------|-----|
| Technical names instead of domain language | Logic hidden behind `DataManager`; experts can't validate the model | Rename to domain terms (`ClaimAdjudicator`); if no domain term exists, the concept may be wrong |
| One model to rule them all | A single `Customer` type shared by billing, shipping, and marketing becomes bloated and contradictory | Bounded contexts: each feature module gets its own `domain.Customer` with only the fields it needs |
| Giant aggregates | Concurrency conflicts, slow loads, transactional bottlenecks | Keep aggregates small; reference by ID; eventual consistency between them |
| Anemic domain model | Structs are data bags with exported fields; rules scatter across handlers, `App`, and SQL | Unexported fields + guarded methods in `internal/domain`; `App` orchestrates only |
| No Anti-Corruption Layer | Foreign models leak in; code couples to another service's DTOs | Wrap every stub and third-party SDK behind an app-side interface returning domain types |
| Importing another feature's `internal/` or sharing its tables | Two contexts silently share one model | Call through generated stubs; each module owns its schema |
| Bounded context = microservice | Premature extraction; distributed complexity without benefit | A context is a model boundary, not a deployment unit; run modules together in one bundle until scaling says otherwise |
| Skipping domain experts | Developers invent a model that doesn't match reality; expensive rework | Regular modeling sessions until experts say "yes, that is how it works" |

## Quick Diagnostic

| Question | If No | Action |
|----------|-------|--------|
| Can a domain expert read your type, method, and route names and understand them? | Technical jargon hides the model | Rename types, methods, events, and contract routes to the SRS glossary terms |
| Are bounded context boundaries explicitly defined? | Models bleed; same term means different things | Draw a context map; define boundaries and translations |
| Are aggregates small (one root + minimal cluster)? | Slow loads, concurrency issues | Split aggregates; reference by ID; accept eventual consistency |
| Do domain objects contain behavior, not just data? | Anemic model; logic scattered in services | Move business rules into entities and value objects |
| Are domain events used for cross-aggregate communication? | Tight coupling, synchronous chains | Return past-tense event values from aggregate methods; drive durable reactions with `flow` workflows |
| Is there an Anti-Corruption Layer at every external integration? | Foreign models pollute your domain | Add a translation layer at each boundary |
| Have you identified which subdomain is core? | Best talent spread thin | Classify subdomains; focus deep modeling on the Core Domain |

## Further Reading

For the complete methodology, patterns, and deeper insights:

- [*"Domain-Driven Design: Tackling Complexity in the Heart of Software"*](https://www.amazon.com/Domain-Driven-Design-Tackling-Complexity-Software/dp/0321125215?tag=wondelai00-20) by Eric Evans

## About the Author

**Eric Evans** is a software design consultant and the originator of Domain-Driven Design, developed through work on large-scale systems in finance, insurance, and logistics. His 2003 book *Domain-Driven Design: Tackling Complexity in the Heart of Software* is one of the most influential software architecture books ever written, and he continues to evolve DDD through his consultancy, Domain Language.

## Attribution

Framework and reference material adapted from
[`wondelai/skills`](https://github.com/wondelai/skills)
(`domain-driven-design`, MIT License). RonyKit mapping added for scaffolded
workspaces.
