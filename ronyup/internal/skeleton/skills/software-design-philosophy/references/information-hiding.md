# Information Hiding and Information Leakage

Information hiding is the most important technique for achieving deep modules. It was first articulated by David Parnas in 1971 and remains the foundation of good software design. Information leakage is its opposite -- and one of the most common sources of unnecessary complexity.


## Table of Contents
1. [The Information Hiding Principle](#the-information-hiding-principle)
2. [Information Leakage](#information-leakage)
3. [Reducing Information Leakage](#reducing-information-leakage)
4. [Case Study: HTTP Request Handling](#case-study-http-request-handling)
5. [Information Hiding Checklist](#information-hiding-checklist)
6. [Relationship to Other Principles](#relationship-to-other-principles)

---

## The Information Hiding Principle

**Each module should encapsulate a few design decisions, and its interface should reveal as little as possible about those decisions.**

The "information" being hidden includes:
- Data representations and storage formats
- Algorithms and implementation strategies
- Communication protocols and wire formats
- Caching strategies and performance optimizations
- Error handling details and recovery mechanisms
- Hardware and OS-specific details
- Concurrency and synchronization strategies
- Configuration and default values

### Why Information Hiding Reduces Complexity

1. **Reduces dependencies:** If callers don't know about an implementation detail, they can't depend on it. Changes to hidden information affect only the module that owns it.

2. **Reduces cognitive load:** Developers using the module need to understand only its interface, not its internals. The hidden information is complexity that is removed from their mental model.

3. **Eliminates unknown unknowns:** When information is properly hidden, there is nothing hidden that callers need to know. The interface is the complete contract.

4. **Enables independent evolution:** Hidden implementations can be changed, optimized, or replaced without affecting any caller.

## Information Leakage

**Information leakage occurs when a design decision is reflected in multiple modules.** It creates a dependency on that decision: if it changes, all modules that know about it must change too.

### Forms of Information Leakage

#### 1. Interface Leakage (Most Obvious)

The module's interface directly exposes implementation details.

```go
// Leaking: the interface exposes the file format and location.
type UserStoreLeaky interface {
	SaveAsJSON(ctx context.Context, u domain.User, path string) error
	LoadFromJSON(ctx context.Context, path string) (domain.User, error)
}

// Hiding: the interface abstracts the storage mechanism.
type UserRepository interface {
	Save(ctx context.Context, u domain.User) error
	Get(ctx context.Context, id string) (domain.User, error)
}
```

In the leaking version, every caller knows the storage format is JSON. Switching to a database requires changing every caller. In the hiding version, the storage mechanism is an internal decision.

**RonyKit note:** The same leak happens when a port returns sqlc row types (`db.User`, `pgtype.Text`) or when `internal/app` methods take the rony request context. `internal/app` should see only domain types and `context.Context`; `internal/repo/v0` maps rows to domain types, and handlers call `ctx.Context()` before calling the app.

#### 2. Back-Door Leakage (Most Subtle)

Two modules share knowledge that is not part of either interface, often through shared data formats, file conventions, or implicit protocols.

```go
// Package export writes:
func writeUser(w io.Writer, u User) error {
	_, err := io.WriteString(w, u.ID+","+u.Name+","+u.Email+"\n")
	return err
}

// Package report reads (far away in the codebase):
func readUsers(r io.Reader) ([]User, error) {
	var users []User
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		f := strings.Split(sc.Text(), ",")
		users = append(users, User{ID: f[0], Name: f[1], Email: f[2]})
	}
	return users, sc.Err()
}
```

Both modules know the CSV format (comma-separated, field order: id, name, email). This knowledge is not in either module's interface. If the format changes, both must change, but there is no compiler error or type check to guide you. This is a classic unknown unknown.

**Fix:** Create a single module that owns the data format:

```go
// Package usercsv owns the file layout; no other package knows the field
// order or the separator.
type Store struct{ path string }

func (s *Store) Write(ctx context.Context, u domain.User) error

func (s *Store) ReadAll(ctx context.Context) ([]domain.User, error)
```

**RonyKit note:** Two common back doors are JSON DTO shapes leaking into `internal/domain` (json tags on domain structs, or domain code that parses request JSON), and the same validation rule written in both the handler and the domain constructor. The DTO belongs in `api/`, and the rule belongs in `domain.NewX` alone.

#### 3. Temporal Leakage

Code is split based on when things happen rather than what knowledge they share.

```go
// Temporal decomposition: split by time.
type RequestReader struct{}

// ReadHeaders knows the HTTP header format.
func (RequestReader) ReadHeaders(r *bufio.Reader) (map[string]string, error)

type BodyParser struct{}

// ParseBody also knows the header format (Content-Length, Content-Type).
func (BodyParser) ParseBody(headers map[string]string, r *bufio.Reader) (Body, error)

type ResponseWriter struct{}

// WriteResponse also knows the HTTP format.
func (ResponseWriter) WriteResponse(w io.Writer, status int, headers map[string]string, body []byte) error
```

All three modules know the HTTP format, even though they are split into "read," "parse," and "write" phases. The temporal decomposition forces shared knowledge across module boundaries.

**Fix:** Organize by knowledge, not by time:

```go
// Conn owns all knowledge of the HTTP wire format.
type Conn struct{ rw *bufio.ReadWriter }

func (c *Conn) ReceiveRequest() (*Request, error)

func (c *Conn) SendResponse(resp *Response) error
```

The same trap shows up in feature code: `parse.go`, `validate.go`, and `persist.go` that each know the order's field rules. In RonyKit, organize by knowledge instead: `api/` owns the wire DTO, `domain.NewOrder` owns the invariants, and `repo/v0` owns the row mapping.

#### 4. Decorator Leakage

The Decorator pattern (in Go, a wrapper type that implements the same interface) is a frequent source of leakage because the wrapper must understand the full interface of the value it wraps.

```go
// loggingItemRepository knows every method of repo.ItemRepository.
type loggingItemRepository struct {
	next repo.ItemRepository
	l    *log.Logger
}

func (r loggingItemRepository) Create(ctx context.Context, item domain.Item) error {
	r.l.Info("repo create", log.String("id", item.ID()))
	return r.next.Create(ctx, item) // pass-through
}

func (r loggingItemRepository) Get(ctx context.Context, id string) (domain.Item, error) {
	r.l.Info("repo get", log.String("id", id))
	return r.next.Get(ctx, id) // pass-through
}

func (r loggingItemRepository) Delete(ctx context.Context, id string) error {
	r.l.Info("repo delete", log.String("id", id))
	return r.next.Delete(ctx, id) // pass-through
}

// ...and every other method on the port.
```

The wrapper is shallow: it adds minimal functionality (logging) but must duplicate the entire interface. Every change to `ItemRepository` propagates to every wrapper.

**Go note:** Wrapping a one-method interface like `io.Reader` is cheap, so the smell is specific to wide interfaces. Embedding `repo.ItemRepository` in the wrapper avoids restating methods, but then any new method silently skips the logging. That hides the leak without removing it.

**Better alternatives:**
- Add logging inside the original implementation (flag-controlled)
- Instrument at an existing narrow seam instead of duplicating the interface: handler middleware, or a tracing span (`x/telemetry/tracekit`) in the app method that owns the operation
- Add a hook/callback mechanism inside the deep module

### How to Detect Information Leakage

| Signal | What It Means |
|--------|--------------|
| Two modules that "always change together" | They share knowledge that should be in one place |
| A data format or protocol mentioned in multiple files | Format knowledge has leaked |
| Tests that break when internal implementation changes | Test code has leaked knowledge about internals |
| Comments like "must match format in module X" | Explicit acknowledgment of leakage |
| Global constants shared across modules | Shared knowledge that may indicate coupling |
| Similar parsing/formatting code in multiple modules | Format knowledge is duplicated |
| `json` tags on domain structs, or sqlc row types in `internal/app` | Wire or storage format has leaked into the core |
| The same validation in the handler and the domain constructor | Rule knowledge has leaked |

## Reducing Information Leakage

### Strategy 1: Merge Modules That Share Knowledge

If two modules share knowledge about a design decision, consider merging them. The result is one module that encapsulates the decision, with a single interface for the rest of the system.

**Before:**
```go
type ConfigReader struct{}

// Read knows the config file format.
func (ConfigReader) Read(path string) (map[string]any, error)

type ConfigApplier struct{}

// Apply also knows the config structure (key names, nesting, defaults).
func (ConfigApplier) Apply(cfg map[string]any) error
```

**After:**
```go
// LoadConfig reads the config file at path, applies defaults, and
// validates the result. All knowledge of the format and its keys lives
// here.
func LoadConfig(path string) (Config, error)
```

(In a RonyKit service, `x/settings` already plays this role; don't rebuild it per feature.)

### Strategy 2: Create a New Module for Shared Knowledge

If merging is not practical (the modules are genuinely different concerns), extract the shared knowledge into a new module that both depend on.

**Before:**
```go
// In api/order.go:
func errorBody(code, msg string, now time.Time) map[string]any {
	return map[string]any{"error": map[string]any{"code": code, "message": msg, "timestamp": now}}
}

// In webhook/stripe.go (a copy, maintained separately):
func errorBody(code, msg string, now time.Time) map[string]any {
	return map[string]any{"error": map[string]any{"code": code, "message": msg, "timestamp": now}}
}
```

**After:**
```go
// In internal/domain/errors.go, the one place the feature's error codes
// and messages are defined:
var (
	ErrOrderNotFound = errs.B().Code(errs.NotFound).Msg("ORDER_NOT_FOUND").Err()
	ErrOrderSave     = errs.GenWrap(errs.Internal, "ORDER_SAVE_FAILED")
)

// Both the API and webhook handlers return these rony/errs errors, and
// neither builds an error payload by hand.
```

### Strategy 3: Push Knowledge Downward

Move knowledge from callers into the module they call. This deepens the module and simplifies its interface.

**Before:**
```go
func submit(ctx context.Context, c *PaymentClient, req ChargeRequest) (Charge, error) {
	// The caller must know the retry strategy.
	var lastErr error
	for attempt := range 3 {
		charge, err := c.Charge(ctx, req)
		if err == nil {
			return charge, nil
		}
		if !errors.Is(err, ErrTransient) {
			return Charge{}, err
		}
		lastErr = err
		time.Sleep(time.Duration(1<<attempt) * time.Second)
	}
	return Charge{}, lastErr
}
```

**After:**
```go
func submit(ctx context.Context, c *PaymentClient, req ChargeRequest) (Charge, error) {
	// Retries, backoff, and error classification are hidden inside Charge.
	return c.Charge(ctx, req)
}
```

### Strategy 4: Separate Interface from Implementation Physically

Use language mechanisms to enforce information hiding:

| Language | Mechanism | Effect |
|----------|----------|--------|
| Python | Underscore prefix (`_private_method`) | Convention-based hiding |
| Java/C# | `private`/`protected` keywords | Compiler-enforced hiding |
| Go | Lowercase names (unexported); `internal/` directories | Package-level and module-subtree hiding (RonyKit's `feature/<name>/internal/`) |
| Rust | `pub` vs non-`pub` | Module-level hiding |
| TypeScript | Unexported module members, `#field` | Module-level hiding (prefer module scope over `private` on classes) |

### Strategy 5: Design Interfaces Around Abstractions

An interface should describe **what** the module does at an abstract level, not **how** it does it.

```go
// Leaking (how):
type LRUTTLCache interface {
	GetFromLRU(key string) ([]byte, bool)
	PutWithTTL(key string, value []byte, ttl time.Duration)
	EvictLRUEntries(n int)
}

// Hiding (what): LRU policy, TTL, and eviction are internal decisions.
type Cache interface {
	Get(key string) ([]byte, bool)
	Put(key string, value []byte)
}
```

## Case Study: HTTP Request Handling

A web server must read an HTTP request (headers and body), route it to a handler, process it, and send a response. Here is how temporal decomposition causes leakage versus how information-based decomposition avoids it.

### Temporal Decomposition (Problematic)

```
Phase 1: Read raw bytes from socket → knows HTTP header format
Phase 2: Parse headers → knows HTTP header format
Phase 3: Read body based on Content-Length → knows header meaning
Phase 4: Route to handler → knows URL format from headers
Phase 5: Build response → knows HTTP response format
Phase 6: Write response to socket → knows HTTP format
```

HTTP format knowledge is spread across 6 phases. Changing anything about the HTTP handling requires touching all of them.

### Information-Based Decomposition (Better)

```
HttpProtocol module:
  - Owns ALL knowledge of HTTP format (headers, body, status codes)
  - Reads socket → produces HttpRequest objects
  - Takes HttpResponse objects → writes to socket

Router module:
  - Owns URL pattern matching
  - Maps HttpRequest to handler function

Handler modules:
  - Work with high-level HttpRequest/HttpResponse objects
  - Know nothing about raw HTTP format
```

Now HTTP format knowledge lives in one place. The router knows only about URL patterns. Handlers know only about request/response objects. Each module hides its specific knowledge. Go's `net/http` is built this way, and so is RonyKit: the gateway owns the wire format, `rony.POST("/items", ...)` owns routing, and handlers receive typed request DTOs.

## Information Hiding Checklist

For each module in your system, ask:

| Question | Desired Answer |
|----------|---------------|
| What design decisions does this module hide? | At least one significant decision |
| Could the implementation be replaced without changing callers? | Yes |
| Does the interface mention implementation-specific concepts? | No |
| Do tests verify behavior or implementation? | Behavior |
| Are there other modules that share knowledge about the same implementation detail? | No |
| If this module's internal format changes, how many other modules must change? | Zero |

If any answer is unsatisfactory, information is leaking and the design should be reconsidered.

## Relationship to Other Principles

- **Deep modules** achieve depth primarily through information hiding -- the hidden information is what makes them deep
- **General-purpose interfaces** hide specific use cases, which is a form of information hiding
- **Comments** should describe the interface (what is visible) without revealing hidden implementation details
- **Strategic programming** is the mindset that makes developers willing to invest effort in proper information hiding rather than taking shortcuts that leak
