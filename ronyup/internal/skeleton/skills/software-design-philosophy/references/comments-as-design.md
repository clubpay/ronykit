# Comments as Design Documentation

Comments are one of the most debated topics in software engineering. Ousterhout argues that comments are not merely helpful -- they are essential design documentation that captures information that cannot be expressed in code. The belief that "good code is self-documenting" is partially true for implementation details, but dangerously wrong for abstractions, design decisions, and cross-cutting concerns.


## Table of Contents
1. [Why Comments Matter](#why-comments-matter)
2. [The Four Types of Comments](#the-four-types-of-comments)
3. [Comment-Driven Design](#comment-driven-design)
4. [The "Self-Documenting Code" Myth](#the-self-documenting-code-myth)
5. [Maintaining Comments](#maintaining-comments)
6. [Comments Anti-Patterns](#comments-anti-patterns)
7. [Summary](#summary)

---

## Why Comments Matter

Code tells you **what** the program does. Comments tell you:
- **Why** it does it that way
- **What** the abstraction promises (the contract)
- **What** assumptions the code makes
- **What** alternatives were considered and rejected
- **What** constraints link this code to other modules
- **What** is not obvious from reading the code

Without comments, this information exists only in the original developer's head. When that developer moves on, the information is lost. Future developers must reverse-engineer intent from implementation -- an error-prone process that leads to incorrect changes and accumulated complexity.

## The Four Types of Comments

### 1. Interface Comments

**Purpose:** Define the abstraction that a package, type, or function presents to its users.

**This is the most important type of comment.** Interface comments form the contract between a module and its callers. They should describe:
- What the function/method does (at an abstract level)
- What each parameter means and its constraints
- What the return value represents
- What side effects occur
- What errors can be returned and under what conditions
- What the caller must ensure before calling (preconditions)
- What the caller can assume after the call (postconditions)

**Examples:**

```go
// FindNearest returns the candidate closest to target by Euclidean
// distance, ignoring candidates farther than maxDistance. Pass
// math.Inf(1) to consider every candidate.
//
// The bool result is false if no candidate is within maxDistance. If
// several candidates are equidistant, the one that appears first in
// candidates wins. FindNearest returns ErrNoCandidates if candidates is
// empty. It never modifies candidates.
func FindNearest(target Point, candidates []Point, maxDistance float64) (Point, bool, error)
```

```go
// Acquire returns a connection from the pool, blocking until one is
// available or ctx is done.
//
// The returned connection has been validated with a lightweight query
// and is ready to use. The caller must call Release exactly once when
// done; that returns the connection to the pool. Acquire is safe for
// concurrent use.
//
// Acquire returns ctx.Err() if ctx is canceled or its deadline passes
// first, and ErrPoolClosed if Close has been called.
func (p *Pool) Acquire(ctx context.Context) (*Conn, error)
```

In a RonyKit feature, the repository port in `internal/repo/port.go` is where interface comments pay off most: `internal/app` only ever sees this contract, never the sqlc implementation behind it.

```go
// ItemRepository persists items. Implementations must be safe for
// concurrent use.
type ItemRepository interface {
	// Create stores a new item. It returns domain.ErrItemExists if an item
	// with the same ID is already stored; the stored item is left unchanged.
	Create(ctx context.Context, item domain.Item) error

	// Get returns the item with the given ID, or domain.ErrItemNotFound.
	Get(ctx context.Context, id string) (domain.Item, error)

	// Delete removes the item. Deleting an item that does not exist is not
	// an error, so callers may retry safely.
	Delete(ctx context.Context, id string) error
}
```

**Go note:** Go doc comments start with the name of the identifier (`// Acquire returns ...`) and are plain prose, with no `@param`/`@throws` tags. Name parameters inline, and name the sentinel errors or `errs` codes a caller can match with `errors.Is`.

**Key rules for interface comments:**
- Describe the abstraction, not the implementation
- If the comment mentions implementation details (algorithms, data structures, internal variables), it is too detailed
- A developer should be able to use the module correctly by reading only the interface comment, without reading any implementation code
- If you cannot write a clear interface comment, the interface may be poorly designed

### 2. Data Structure Member Comments

**Purpose:** Explain the meaning, constraints, and invariants of fields in a struct or other data structure.

Field names alone rarely convey all the information a developer needs. Comments should clarify:
- What the field represents (especially if the name is ambiguous)
- Units and encoding (milliseconds? seconds? UTC? local time?)
- Valid ranges and boundary conditions
- Relationships with other fields
- When the field is set and when it may be null/zero

**Examples:**

```go
type RetryConfig struct {
	// MaxRetries is the number of retries after the initial attempt, so
	// total attempts = MaxRetries + 1. Zero disables retries.
	MaxRetries int

	// BaseDelay is the delay before the first retry. Later retries use
	// exponential backoff (BaseDelay * 2^attempt) with +/-20% jitter to
	// prevent a thundering herd.
	BaseDelay time.Duration

	// MaxDelay caps the backoff regardless of attempt number.
	// Must be >= BaseDelay.
	MaxDelay time.Duration
}
```

Using `time.Duration` instead of an `int` of milliseconds lets the type carry the unit, so the comment can spend its words on what the type cannot say.

```go
type PageCache struct {
	mu sync.Mutex // guards all fields below

	// pages maps page ID to cached content. Entries are evicted in LRU
	// order when the cache would exceed maxEntries. A page present here
	// matches the on-disk version as of lastSync.
	pages map[int64]Page

	// lastSync is when the cache was last synchronized with disk (UTC).
	// All entries are valid as of this time; writes after it may not be
	// reflected.
	lastSync time.Time

	// maxEntries bounds the cache. When it is reached, the least recently
	// accessed entry is evicted before a new one is inserted.
	// Invariant: len(pages) <= maxEntries.
	maxEntries int
}
```

**TypeScript note:** Component props are data members too. TSDoc on each prop shows up in editor hovers wherever the component is used:

```tsx
type PriceTagProps = {
  /** Amount in minor units (cents). Never a float. */
  readonly amountMinor: number;
  /** ISO 4217 code such as "EUR"; decides decimal places and symbol. */
  readonly currency: string;
  /** Previous price in minor units, shown struck through. Must exceed `amountMinor`. */
  readonly compareAtMinor?: number;
};

/** Formats a price for the user's locale. Uses no hooks, so it can render in Server Components. */
export function PriceTag({ amountMinor, currency, compareAtMinor }: PriceTagProps) {
  const nf = new Intl.NumberFormat(undefined, { style: 'currency', currency });
  const digits = nf.resolvedOptions().maximumFractionDigits ?? 2;
  const format = (minor: number) => nf.format(minor / 10 ** digits);
  return (
    <span>
      {compareAtMinor !== undefined && <s className="text-muted-foreground">{format(compareAtMinor)}</s>}{' '}
      {format(amountMinor)}
    </span>
  );
}
```

### 3. Implementation Comments

**Purpose:** Explain **why** the code does something a particular way, or clarify non-obvious logic.

Implementation comments should not describe **what** the code does -- that should be clear from reading the code itself. They should explain:
- Why this approach was chosen over alternatives
- What non-obvious constraint or edge case the code handles
- What would go wrong if the code were changed in an obvious-seeming way
- Performance considerations that drove the implementation choice

**Good implementation comments:**

```go
func findEntry(sortedEntries []Entry, key string) (int, bool) {
	// Binary search instead of a linear scan: the slice is sorted and can
	// hold 100k+ entries. The linear scan caused 200ms latency in
	// production (see incident #4521).
	return slices.BinarySearchFunc(sortedEntries, key, func(e Entry, k string) int {
		return strings.Compare(e.Key, k)
	})
}
```

```go
func removeExpired(items []Item, now time.Time) []Item {
	// Walk backwards so that deleting an element does not shift the
	// indexes still to be visited; a forward loop would skip the element
	// after each deletion.
	for i := len(items) - 1; i >= 0; i-- {
		if items[i].ExpiredAt(now) {
			items = slices.Delete(items, i, i+1)
		}
	}
	return items
}
```

(`slices.DeleteFunc` removes the need for both the loop and the comment. Sometimes the best fix for a hard-to-explain comment is a better API.)

```go
func (a *App) processBatch(ctx context.Context, records []Record) []Result {
	results := make([]Result, 0, len(records))
	for _, rec := range records {
		res, err := a.vendor.Process(ctx, rec.Data)
		if err != nil {
			// Deliberately log and continue instead of returning: the vendor
			// SDK returns undocumented error types (timeouts, validation and
			// I/O errors seen in production), and one bad record must not
			// abort the whole batch. The record stays PENDING, so the next run
			// retries it.
			a.l.Warn("vendor processing failed", log.String("record_id", rec.ID), log.Error(err))
			res = defaultResult()
		}
		results = append(results, res)
	}
	return results
}
```

**Bad implementation comments (just repeat the code):**

```go
func countActive(users []User) int {
	// Initialize counter
	counter := 0

	// Loop through users
	for _, u := range users {
		// Check if user is active
		if u.Active {
			// Increment counter
			counter++
		}
	}

	// Return the result
	return counter
}
```

These comments add no information. The code already says what it does. Remove them.

### 4. Cross-Module Comments

**Purpose:** Document dependencies and design decisions that span multiple modules.

These are the hardest comments to maintain but often the most critical, because cross-module relationships are the biggest source of unknown unknowns.

**Examples:**

```go
// requestTimeout must be longer than the payment client's total retry
// budget (currently 3 retries x 30s = 90s). If it is shorter, the caller
// gives up before the retries complete.
// See feature/payment/internal/app/retry.go: maxRetryDuration.
const requestTimeout = 120 * time.Second
```

```go
// serverMessage field order must match the binary protocol defined in
// docs/protocol-v3.md section 4.2. The TypeScript client parser
// (web/src/lib/protocol/parser.ts) reads fields in exactly this order.
// Changing the order here requires updating both the doc and the parser.
type serverMessage struct {
	Version    uint16 // big-endian
	Type       uint8
	PayloadLen uint32 // big-endian
	Payload    []byte // PayloadLen bytes
}
```

```go
// OnOrderCompleted handles OrderCompleted events. The event bus calls it
// on its own consumer goroutine, not on a request, so it must not rely on
// request-scoped values in ctx. Hand long-running work to a flow
// workflow instead of blocking the consumer.
//
// The bus delivers at-least-once, so this handler must be idempotent: it
// keys fulfillment by event.OrderID. See eventbus.Subscribe for the
// delivery guarantees.
func (a *App) OnOrderCompleted(ctx context.Context, event OrderCompletedEvent) error
```

**Best practices for cross-module comments:**
- Place the comment in the most likely place a developer would look
- Reference the other module explicitly (file path, type or function name)
- Explain what would go wrong if the relationship were violated
- Consider using a shared constants file for values that must stay in sync

## Comment-Driven Design

**Write the comments before writing the code.**

This is one of Ousterhout's most practical recommendations. The process:

1. **Write the interface comment first:** Before writing any implementation, write the comment that describes what the function/type/package does, what its parameters mean, and what it returns.

2. **Evaluate the design:** If the interface comment is hard to write, unclear, or requires mentioning implementation details, the interface design is probably wrong. Redesign the interface until the comment is clean and simple.

3. **Write the implementation:** With a clear interface comment as your guide, the implementation has a clear target.

4. **Add implementation comments:** As you write code, add comments for any non-obvious decisions.

### Why Comment-Driven Design Works

| Benefit | Explanation |
|---------|-------------|
| Forces clear thinking | Writing what something does before how reveals confusion early |
| Catches bad abstractions | If you can't describe the interface simply, it's too complex |
| Produces better interfaces | The act of writing clarifies what callers actually need |
| Comments stay accurate | Written alongside the design, not retrofitted later |
| Saves time | Avoids implementing a design that turns out to be wrong |

### Example

**Step 1:** Write the interface comment.

```go
// MergeSorted merges sorted sequences into a single sorted sequence.
//
// Each input must already be sorted in ascending order according to cmp.
// The result yields every element of every input in globally sorted
// order. Memory use is O(len(seqs)), regardless of sequence length.
//
// Equal elements are yielded in the order their source sequences appear
// in seqs (stable merge).
func MergeSorted[T any](cmp func(a, b T) int, seqs ...iter.Seq[T]) iter.Seq[T]
```

**Step 2:** Evaluate. Is this clear? Can a caller use this without reading the implementation? What about edge cases -- empty streams, single stream, duplicate elements? Add those details if needed.

**Step 3:** Implement. The comment now serves as the specification.

## The "Self-Documenting Code" Myth

The claim that "good code doesn't need comments" contains a kernel of truth but is dangerously incomplete.

### Where Self-Documenting Code Works

Code **can** document itself for low-level implementation details:

```go
// Self-documenting -- no implementation comments needed:
func (c Cart) Total() Money {
	var total Money
	for _, item := range c.Items {
		total += item.Price
	}
	return total
}

func isEligible(u User) bool { return u.Age >= 18 && u.HasValidID }

func activeAbove(records []Record, threshold int) []Record {
	return rkit.Filter(records, func(r Record) bool { return r.Active && r.Score > threshold })
}
```

Good variable names, clear control flow, and simple expressions make the **what** obvious. Comments that restate this are noise.

### Where Self-Documenting Code Fails

Code **cannot** document:

| Information | Why Code Can't Express It | Example |
|------------|--------------------------|---------|
| **Abstractions** | Code shows implementation, not the promise | An interface's contract and guarantees |
| **Why** | Code shows what happens, not why this approach | Why binary search instead of hash lookup |
| **Constraints** | Code enforces constraints but doesn't explain them | Why a timeout is set to 120 seconds |
| **Design alternatives** | Code shows the choice made, not choices rejected | Why we chose polling over webhooks |
| **Cross-module relationships** | Code in one module can't describe its relationship to another | This timeout must match the retry config |
| **Performance rationale** | Optimized code is often less readable | Why we denormalized this data structure |
| **Assumptions** | Code operates on assumptions it cannot state | "This list is always sorted by the caller" |

### The Practical Rule

**Use self-documenting code for the "what" (implementation). Use comments for the "why" (design decisions), the "what" at a higher level (abstractions/interfaces), and the "beware" (non-obvious constraints and relationships).**

## Maintaining Comments

Comments that are wrong are worse than no comments. Here are strategies for keeping them accurate:

### 1. Place Comments Near the Code

The closer a comment is to the code it describes, the more likely it will be updated when the code changes. Doc comments directly above the declaration are better than comments in a separate documentation file.

### 2. Avoid Duplicating Information

If the same information is stated in a comment and enforced in code, one will eventually become stale. State each fact once.

```go
// Bad: the comment restates the declaration.
type retryPolicyBefore struct {
	// MaxRetries is an int holding the maximum number of retries.
	MaxRetries int
}

// Good: the comment adds what the declaration cannot say.
type retryPolicyAfter struct {
	// MaxRetries of 0 disables retries. Values above 10 are capped at 10
	// to avoid overloading the downstream service during outages.
	MaxRetries int
}
```

### 3. Update Comments in the Same Commit

Make it a code review norm: if you change a function's behavior, you must update its interface comment in the same commit. Stale comments are a code review finding.

### 4. Use Comments as a Design Smell Detector

If a comment is hard to write, the code may be too complex. If a comment needs to be very long, the interface may be doing too much. If a comment keeps going out of date, the module's boundaries may be wrong. Difficult comments are a signal, not just a chore.

### 5. Treat Comment Quality as a Review Criterion

In code reviews, evaluate comments alongside code:
- Are interface comments complete and accurate?
- Do implementation comments explain why, not what?
- Are cross-module comments present where needed?
- Are there missing comments on non-obvious code?

## Comments Anti-Patterns

| Anti-Pattern | Problem | Fix |
|-------------|---------|-----|
| **Comment repeats the code** | Adds noise, no information | Delete it; let the code speak for implementation details |
| **Comment describes what, not why** | Misses the valuable information | Rewrite to explain the reasoning or design decision |
| **Comment on every line** | Obscures code, hard to maintain | Comment only non-obvious sections; trust clear code |
| **TODO without context** | "TODO: fix this" is useless months later | Include the issue number, the problem, and the fix direction |
| **Commented-out code** | Dead code that confuses readers | Delete it; version control preserves history |
| **Banner comments** | `/////// SECTION ///////` adds structure without information | Use meaningful function/type/package boundaries instead |
| **Apology comments** | "Sorry, this is a hack" acknowledges but doesn't fix | Fix the hack or add context on why it is necessary and when it can be fixed |
| **Stale comments** | Describe behavior that no longer exists | Update or remove in the same commit as the code change |

## Summary

Comments are not a sign of bad code. They are design documentation that captures the most valuable and perishable information in a system: the designer's intent, the abstraction's contract, and the non-obvious relationships between components. Write interface comments first, maintain them alongside code, and use them as a tool for thinking clearly about design.
