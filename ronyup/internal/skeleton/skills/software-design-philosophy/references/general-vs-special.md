# General-Purpose vs Special-Purpose Modules

One of the most important design decisions is how general-purpose or special-purpose a module's interface should be. Ousterhout advocates for a "somewhat general-purpose" approach: general enough to avoid special cases, specific enough to avoid over-engineering.


## Table of Contents
1. [The Spectrum](#the-spectrum)
2. [The Key Question](#the-key-question)
3. [Push Complexity Downward](#push-complexity-downward)
4. [Configuration Parameters: Complexity Amplifiers](#configuration-parameters-complexity-amplifiers)
5. [When Specialization Is Justified](#when-specialization-is-justified)
6. [Practical Guidelines](#practical-guidelines)
7. [The Relationship to Information Hiding](#the-relationship-to-information-hiding)
8. [Summary](#summary)

---

## The Spectrum

```
Too Special ←————————————————————————→ Too General
  (bloated     (sweet spot:              (wasted effort,
  with         "somewhat                  unnecessary
  special      general-purpose")          abstraction)
  cases)
```

### Too Special-Purpose

A module designed for one specific use case. Its interface includes details that tie it to a particular caller or scenario.

```go
// Too special: one type per email use case.
type WelcomeEmailSender interface {
	SendWelcomeEmail(ctx context.Context, userName, userEmail, planName string) error
}

type PasswordResetEmailSender interface {
	SendResetEmail(ctx context.Context, userEmail, resetToken string, expiry time.Duration) error
}

type InvoiceEmailSender interface {
	SendInvoiceEmail(ctx context.Context, userEmail, invoiceID string, amount int64, due time.Time) error
}
```

Three types doing essentially the same thing (sending email) with interfaces tied to specific use cases. Adding a fourth email type requires creating another type.

### Too General-Purpose

A module designed for every conceivable use case, including ones that may never arise.

```go
// Too general: anticipates every possible need.
type UniversalMessageDispatcher interface {
	Dispatch(ctx context.Context, channel Channel, template Template, recipients []Recipient,
		variables map[string]string, priority Priority, schedule Schedule, retry RetryPolicy,
		attachments []Attachment, tracking TrackingConfig, abTest ABTestConfig,
		localization LocalizationConfig, rateLimit RateLimitConfig, webhooks []Webhook) error
}
```

The interface is so general that using it requires understanding 13 parameters. Most callers will use only a fraction of them.

### Somewhat General-Purpose (The Sweet Spot)

```go
// Somewhat general: covers current needs with a simple interface.
type Mailer interface {
	Send(ctx context.Context, msg Email) error
}

type Email struct {
	To          string
	Subject     string
	Body        string
	Attachments []Attachment // optional; nil means none
}
```

This covers welcome emails, password resets, invoices, and any future email type with a single, simple interface. It is general enough to handle all current use cases without special-case methods, but it does not try to handle SMS, push notifications, or A/B testing.

## The Key Question

> **"What is the simplest interface that will cover all my current needs?"**

This question is the practical tool for finding the sweet spot. It has three important parts:

1. **Simplest interface:** Minimize the number of methods, parameters, and concepts
2. **All current needs:** Do not design for hypothetical future requirements
3. **Cover:** The interface must actually work for every current use case without workarounds

### Applying the Question

**Step 1:** List all current use cases for the module.

**Step 2:** For each use case, identify what the caller needs from the module.

**Step 3:** Find the minimal set of methods and parameters that satisfies all callers.

**Step 4:** Check that no use case requires awkward workarounds.

**Example:**

A text editor needs to support:
- Inserting text at a position
- Deleting a range of text
- Replacing a range of text

Special-purpose approach:
```go
type Buffer interface {
	InsertText(pos Position, text string)
	DeleteRange(start, end Position)
	ReplaceRange(start, end Position, text string)
	InsertHeading(pos Position, level int, text string)
	InsertBulletPoint(pos Position, text string)
	DeleteWord(pos Position)
	ReplaceWord(pos Position, word string)
	Backspace()
	DeleteSelection()
}
```

Somewhat general-purpose approach:
```go
type Buffer interface {
	Insert(pos Position, text string)
	Delete(start, end Position)
}
```

The general-purpose approach covers all cases with two methods. `ReplaceRange` is just `Delete` followed by `Insert`. `Backspace` and `DeleteSelection` are `Delete` over a range the UI layer computes. Headings and bullet points are just text with formatting characters. The interface is simpler and covers all current needs.

**Go note (RonyKit repository ports):** The same trade-off shows up in `internal/repo/port.go`. One port method per screen or query is the special-purpose version. A domain-level method that takes a small filter struct is the somewhat general one:

```go
// Special-purpose: a new port method (and sqlc query) for every caller.
type OrderRepositorySpecial interface {
	ListOpenOrdersForCustomer(ctx context.Context, customerID string) ([]domain.Order, error)
	ListOrdersCreatedToday(ctx context.Context) ([]domain.Order, error)
	ListOverdueOrdersForCustomer(ctx context.Context, customerID string, now time.Time) ([]domain.Order, error)
}

// Somewhat general: one method; the zero value of each filter field means
// "don't filter on this".
type OrderRepository interface {
	List(ctx context.Context, f OrderFilter) ([]domain.Order, error)
}

type OrderFilter struct {
	CustomerID    string
	Status        domain.OrderStatus
	CreatedAfter  time.Time
	DueBefore     time.Time
	Limit, Offset int
}
```

Stop before the filter turns into a query language. Add a field only when a current use case needs it.

## Push Complexity Downward

**Principle:** It is more important for a module to have a simple interface than a simple implementation.

When complexity must exist somewhere in the system, it is better to put it inside a module (deepening it) than in the module's interface (burdening all callers).

### Why Downward, Not Upward?

| Complexity Location | Who Bears the Cost | Multiplier Effect |
|--------------------|-------------------|-------------------|
| Inside the module | The module's developer, once | 1x |
| In the interface | Every caller, every time they use it | Nx (where N = number of callers) |

A module with a complex implementation but simple interface imposes complexity on one developer (the module author). A module with a simple implementation but complex interface imposes complexity on every developer who uses it.

### Example: Connection Pooling

**Complexity pushed up (to callers):**
```go
func countOrders(ctx context.Context, pool *ConnPool) (int64, error) {
	// Every caller manages the pool lifecycle.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return 0, err
	}
	defer pool.Release(conn)
	// ...and must also handle exhaustion, stale connections, reconnection.
	return conn.QueryInt64(ctx, "SELECT count(*) FROM orders")
}
```

**Complexity pushed down (into the module):**
```go
func countOrders(ctx context.Context, db *sql.DB) (int64, error) {
	// database/sql pools, validates, and reconnects internally; callers
	// just query.
	var n int64
	err := db.QueryRowContext(ctx, "SELECT count(*) FROM orders").Scan(&n)
	return n, err
}
```

### Example: Error Handling

**Complexity pushed up:**
```go
func handle(src string) error {
	res := parser.Parse(src)
	switch {
	case res.SyntaxErr != nil:
		return handleSyntaxError(res.SyntaxErr)
	case res.SemanticErr != nil:
		return handleSemanticError(res.SemanticErr)
	case res.Ambiguity != nil:
		return handleAmbiguity(res.Ambiguity)
	}
	return process(res.Value)
}
```

**Complexity pushed down:**
```go
func handle(src string) error {
	// Parse classifies every failure into one *ParseError that carries a
	// clear message and a location.
	v, err := parser.Parse(src)
	if err != nil {
		var pe *ParseError
		if errors.As(err, &pe) {
			showError(pe.Msg, pe.Pos)
		}
		return err
	}
	return process(v)
}
```

## Configuration Parameters: Complexity Amplifiers

Configuration parameters are one of the most common ways modules push complexity upward to callers. Each parameter represents a decision the module is refusing to make.

### The Problem

```go
func newSessionCache() (*Cache, error) {
	// 11 decisions pushed to the caller.
	return NewCache(CacheConfig{
		MaxSize:          1000,
		EvictionPolicy:   "lru",
		TTL:              time.Hour,
		CleanupInterval:  5 * time.Minute,
		MaxMemoryMB:      256,
		Serializer:       "json",
		Compression:      true,
		CompressionLevel: 6,
		StatsEnabled:     true,
		StatsInterval:    time.Minute,
		ThreadSafe:       true,
	})
}
```

Every parameter is a question the caller must answer. Most callers don't know the right answer and will either copy values from examples or guess. Wrong values cause subtle performance problems or bugs that are hard to diagnose.

### Better Approaches

| Strategy | How It Helps | Example |
|----------|-------------|---------|
| **Sensible defaults** | Module makes the decision unless overridden | `NewCache()` works with reasonable defaults; override only what you need |
| **Auto-detection** | Module determines the right value at runtime | Auto-size based on available memory; auto-select compression based on data characteristics |
| **Progressive disclosure** | Simple API for simple use; options for advanced use | `NewCache()` for basic use; `NewCache(WithTTL(time.Hour))` functional options for custom |
| **Convention over configuration** | Follow well-known patterns | The datasource reads its DSN from `x/settings` (e.g. a `DATABASE_URL` key); no parameter needed |
| **Elimination** | Remove the parameter entirely | Instead of a `ThreadSafe` field, always be safe for concurrent use (the cost is usually negligible) |

### When Configuration Is Justified

Configuration parameters are justified when:
1. **Different callers genuinely need different values** (not just "might someday need")
2. **The module cannot determine the right value** (it lacks the information)
3. **The wrong default would cause real harm** (not just suboptimal performance)
4. **The decision changes between deployments** (environment-specific settings)

### The Test

For each configuration parameter, ask:
- "Can the module figure this out on its own?" If yes, remove the parameter.
- "Do most callers use the same value?" If yes, make it the default.
- "Will the caller know the right value?" If no, the parameter is shifting complexity, not simplifying.

## When Specialization Is Justified

Despite the general preference for general-purpose design, specialization is appropriate in certain situations:

### 1. Domain-Specific Modules

When a module embodies domain-specific knowledge that does not generalize.

```go
// Justified specialization: tax rules are inherently domain-specific.
type USTaxCalculator struct{ rates RateTable }

func (c USTaxCalculator) FederalTax(income Money, status FilingStatus, deductions []Deduction) Money

func (c USTaxCalculator) StateTax(income Money, state State) Money
```

A "general-purpose tax calculator" would need to know about every country's tax system. Specialization to US taxes hides substantial domain complexity behind a focused interface.

### 2. Performance-Critical Paths

When general-purpose abstractions introduce unacceptable overhead.

```go
// General-purpose: flexible but slow for the hot path.
func Transform(img Image, pipeline []Transformer) Image {
	for _, t := range pipeline {
		img = t.Apply(img) // dynamic dispatch and a new Image per step
	}
	return img
}

// Specialized: optimized for one hot path. Converts in place, with no
// interface calls and no allocation.
func RGBToHSVInPlace(pixels []Pixel) {
	for i := range pixels {
		pixels[i] = rgbToHSV(pixels[i])
	}
}
```

### 3. User-Facing Interfaces

When the interface is used by end users (not developers), specialized vocabulary improves usability.

```go
func scheduleReport(ctx context.Context, s *Scheduler, now time.Time) error {
	// General-purpose API: flexible but requires domain knowledge.
	if err := s.CreateRecurringTask(ctx, RecurringTask{
		Interval: 7 * 24 * time.Hour,
		Start:    nextMonday(now),
		Run:      sendReport,
	}); err != nil {
		return err
	}

	// Specialized API: matches the user's mental model.
	return s.SendWeeklyReport(ctx, time.Monday, "09:00")
}
```

### 4. Adapters and Bridges

When connecting two systems with incompatible interfaces, the adapter is inherently specific to both.

```go
// stripeEvents translates Stripe webhook payloads into domain events; it
// is the only code that knows Stripe's event shapes.
type stripeEvents struct{}

func (stripeEvents) ToPaymentEvent(evt stripe.Event) (domain.PaymentEvent, error)
```

## Practical Guidelines

### When Designing a New Module

1. List all current use cases
2. Ask: "What is the simplest interface that covers all of these?"
3. Resist adding methods for hypothetical future use cases
4. Push complexity into the implementation, away from the interface
5. Default to slightly more general than you think you need -- it is usually simpler

### When Reviewing an Existing Module

| Signal | Problem | Action |
|--------|---------|--------|
| Many methods that differ only in parameters | Over-specialization | Merge into fewer, more general methods |
| Methods named after specific callers | Coupling to use cases | Rename around the concept, not the caller |
| Long parameter lists | Complexity pushed upward | Add defaults, auto-detect, or absorb decisions |
| Multiple modules with similar functionality | Opportunity for generalization | Extract a shared general-purpose module |
| Configuration that "nobody touches" | Parameters that should be defaults | Make them defaults or remove them |

### When Adding a Feature

Before adding a new method or parameter:
1. Can an existing method handle this with its current interface?
2. Can a slight generalization of an existing method handle this?
3. Does the new method introduce a special case that could be avoided?

The best features are those that require no interface changes because the existing abstraction already supports them.

## The Relationship to Information Hiding

General-purpose interfaces hide **use-case-specific knowledge**. When an interface is general, callers don't need to know about other callers' use cases. This is a form of information hiding that reduces dependencies between callers.

A special-purpose interface like `SendWelcomeEmail()` creates a dependency: every developer who sees it must understand the welcome email use case. A general-purpose interface like `Send(ctx, Email{To, Subject, Body})` hides all specific use cases, reducing the information each developer must hold in mind.

## Summary

The goal is not the most general-purpose design possible. It is the **simplest interface that covers all current needs**. This sweet spot produces modules that are:
- Simple to use (few methods, few parameters)
- Flexible enough for current needs (no workarounds required)
- Future-friendly (new use cases often fit the existing abstraction)
- Deep (general interfaces tend to hide more implementation complexity)

When in doubt, lean slightly toward more general -- it is usually simpler. But stop well before building a framework for every conceivable future need.
