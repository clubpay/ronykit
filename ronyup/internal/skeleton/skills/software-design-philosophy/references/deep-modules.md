# Deep vs Shallow Modules

The concept of module depth is one of the most powerful ideas in Ousterhout's philosophy. It provides a concrete way to evaluate whether a module is pulling its weight in the system.


## Table of Contents
1. [The Core Idea](#the-core-idea)
2. [Visualizing Module Depth](#visualizing-module-depth)
3. [Examples of Deep Modules](#examples-of-deep-modules)
4. [Examples of Shallow Modules](#examples-of-shallow-modules)
5. [The Disease of Classitis](#the-disease-of-classitis)
6. [When Shallow Is Acceptable](#when-shallow-is-acceptable)
7. [Designing for Depth](#designing-for-depth)
8. [Measuring Depth in Practice](#measuring-depth-in-practice)
9. [Common Objections](#common-objections)

---

## The Core Idea

Every module has two parts:
- **Interface:** The complexity it imposes on the rest of the system (the cost)
- **Implementation:** The functionality it provides (the benefit)

A module's value is determined by the ratio of functionality provided to interface complexity imposed.

```
Module Value = Functionality / Interface Complexity
```

**Deep modules** have high value: they provide a lot of functionality through a simple interface. **Shallow modules** have low value: their interface is nearly as complex as their implementation, so they add little net simplification to the system.

## Visualizing Module Depth

Think of a module as a rectangle:
- Width at the top = interface complexity
- Height = implementation depth (functionality hidden)

```
Deep Module:               Shallow Module:
┌──────┐                   ┌──────────────────────┐
│      │                   │                      │
│      │                   └──────────────────────┘
│      │
│      │
│      │
│      │
└──────┘
Narrow interface,          Wide interface,
deep implementation.       shallow implementation.
```

The goal is tall, narrow rectangles: modules that hide substantial complexity behind small interfaces.

## Examples of Deep Modules

### Unix File I/O

The Unix file I/O interface is one of the deepest abstractions in computing. Go's `os` package keeps its shape almost unchanged:

```go
// Package os
func Open(name string) (*File, error)
func (f *File) Read(b []byte) (n int, err error)
func (f *File) Write(b []byte) (n int, err error)
func (f *File) Seek(offset int64, whence int) (ret int64, err error)
func (f *File) Close() error

// Package io: the one-method interface every file, socket, buffer, and
// decompressor satisfies.
type Reader interface {
	Read(p []byte) (n int, err error)
}
```

Five functions. Behind this simple interface, the implementation handles:
- Disk block allocation and management
- Directory traversal and path resolution
- File permissions and access control
- Buffer caching and write-back strategies
- Device driver communication
- File system journal and crash recovery
- Network file system protocols (NFS)
- Memory-mapped file coordination
- Concurrent access and locking

The interface is measured in a few functions; the implementation is hundreds of thousands of lines of code. This is extreme depth.

### Garbage Collectors

A garbage collector's interface is essentially invisible:

```
Interface: (none -- just allocate objects normally)
```

Behind this zero-complexity interface, the implementation handles:
- Reference tracking and reachability analysis
- Generational collection strategies
- Compaction and memory defragmentation
- Concurrent collection without stopping the world
- Weak references and finalization
- Heap sizing and growth heuristics

The deepest modules are those whose interfaces are so simple that callers may not even realize they exist.

### TCP/IP Networking

```go
func ping(ctx context.Context, addr string) ([]byte, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("PING\n")); err != nil {
		return nil, err
	}
	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	return buf[:n], err
}
```

Behind `Write` and `Read` (the same `io.Writer`/`io.Reader` methods as a file):
- Packet segmentation and reassembly
- Retransmission and acknowledgment
- Flow control and congestion avoidance
- Routing across networks
- Checksum verification
- Connection state management
- Out-of-order packet handling

`net/http.Get(url)` stacks DNS, TLS, connection reuse, redirects, and HTTP/2 negotiation on top of this, and its interface is still one function.

### Hash Maps

```go
func mapOps(m map[string]int) {
	m["key"] = 1
	v, ok := m["key"]
	delete(m, "key")
	_, _ = v, ok
}
```

Behind this:
- Hash function computation
- Collision resolution and bucket layout
- Dynamic resizing and incremental rehashing
- Memory allocation strategies
- Load factor management
- Safe deletion during `range` iteration

## Examples of Shallow Modules

### Java I/O Classes (Classic Example)

To read a serialized object from a file in Java, the caller must stack three classes: `FileInputStream` (reads bytes, no buffering), `BufferedInputStream` (adds buffering -- why isn't this the default?), and `ObjectInputStream` (deserializes objects).

Each class is shallow: its interface is nearly as complex as its implementation. The total cognitive load of three interfaces is greater than what a single deep interface would impose, and forgetting the middle layer silently makes reads slow.

Go layers readers too, but the argument still holds, and the contrast is instructive:

```go
func loadSnapshot(path string) (Snapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return Snapshot{}, err
	}
	defer f.Close()

	var s Snapshot
	// No bufio.NewReader needed: gob.NewDecoder buffers internally when
	// its reader is not already an io.ByteReader.
	err = gob.NewDecoder(f).Decode(&s)
	return s, err
}
```

The layers are still explicit (`os.Open`, then a decoder), but each one is deep on its own: `*os.File` is usable without wrapping, they compose through the one-method `io.Reader` so the extra interface cost is close to zero, and the decoder absorbs the buffering decision instead of pushing it to every caller. For the most common case there is an even deeper shortcut:

```go
func loadRaw(path string) ([]byte, error) {
	return os.ReadFile(path) // opens, sizes the buffer, reads to EOF, closes
}
```

### Thin Wrapper Types

In Go, classitis shows up as one-method interfaces with a single implementation, wired together for no benefit:

```go
type UserValidator interface {
	Validate(u domain.User) error
}

type UserSaver interface {
	Save(ctx context.Context, u domain.User) error
}

type userValidator struct{}

func (userValidator) Validate(u domain.User) error {
	if u.Name() == "" || u.Email() == "" {
		return domain.ErrUserInvalid
	}
	return nil
}

type UserService struct {
	validator UserValidator
	saver     UserSaver
}

func (s *UserService) CreateUser(ctx context.Context, u domain.User) error {
	if err := s.validator.Validate(u); err != nil {
		return err
	}
	return s.saver.Save(ctx, u)
}
```

A constructor and one `App` method are enough:

```go
func NewUser(name, email string) (User, error) {
	if name == "" || email == "" {
		return User{}, ErrUserInvalid
	}
	return User{id: rkit.RandomID(16), name: name, email: email}, nil
}

func (a *App) CreateUser(ctx context.Context, name, email string) (domain.User, error) {
	u, err := domain.NewUser(name, email)
	if err != nil {
		return domain.User{}, err
	}
	if err := a.users.Create(ctx, u); err != nil {
		return domain.User{}, err
	}
	return u, nil
}
```

The first version creates two extra interfaces (and their fakes, files, and wiring) without providing meaningful abstraction. The validation and persistence logic is too simple to justify separate modules. The repository port `a.users` stays, because it hides a real decision: how and where users are stored.

### Pass-Through Methods

```go
func (svc Service) CreateOrder(ctx *RContext, in CreateOrderRequest) (*CreateOrderResponse, error) {
	if len(in.Lines) == 0 {
		return nil, errs.B().Code(errs.InvalidArgument).Msg("ORDER_EMPTY").Err()
	}
	total := sumLines(in.Lines)
	if in.CouponCode == "WELCOME10" {
		total = total * 90 / 100
	}
	id, err := svc.app.CreateOrder(ctx.Context(), in.CustomerID, total)
	if err != nil {
		return nil, errs.B().Cause(err).Msg("CREATE_ORDER_FAILED").Err()
	}
	return &CreateOrderResponse{ID: id}, nil
}

func (a *App) CreateOrder(ctx context.Context, customerID string, total int64) (string, error) {
	return a.orders.CreateOrder(ctx, customerID, total)
}

func (r *orderRepository) CreateOrder(ctx context.Context, customerID string, total int64) (string, error) {
	return r.q.InsertOrder(ctx, db.InsertOrderParams{CustomerID: customerID, Total: total})
}
```

The `CreateOrder` method appears three times, and the middle one just forwards its arguments. RonyKit requires handlers to go through `internal/app`, so a thin app method for plain CRUD is acceptable. The smell here is different: the business rules (an order needs lines, the coupon discount) sit in the handler, so the app layer has nothing left to hide. Move the rules into `domain.NewOrder` and `App.CreateOrder`. Then the handler only decodes and maps, and the app method becomes deep.

## The Disease of Classitis

**Classitis** is the misguided belief that "classes should be small" applied without judgment. It produces systems with hundreds of tiny classes, each doing very little, connected by a web of interfaces.

### Symptoms of Classitis

| Symptom | Example |
|---------|---------|
| Many tiny packages or types with 10-30 lines each | `stringhelper`, `dateformat`, `nullcheck` packages |
| Most methods are one-liners or delegates | `func (u *User) GetName() string { return u.name }` on plain data |
| Understanding a feature requires reading 8+ types | Handler, Service, Repository, Mapper, Validator, DTO, Entity, Factory |
| Names end in -Helper, -Util, -Manager; packages named `util`, `common` | `UserManager`, `OrderHelper`, `package util` |
| Many one-method interfaces with a single implementation | A `Validator` interface whose only implementation is `validator` |

**Go note:** Go has no classes, but classitis shows up in three forms: too many tiny packages, too many one-method interfaces with one implementation, and pass-through layers (handler, app, and repo methods that each just forward).

### Why Classitis Happens

1. **Misinterpreted "Single Responsibility Principle"**: SRP says "one reason to change," not "one thing it does." A module can do many things if they all change together.
2. **Cargo cult patterns**: Applying patterns (Strategy, Factory, Builder) reflexively without evaluating whether they add depth.
3. **Metrics worship**: Optimizing for "small file size" or "few methods per type" instead of depth.
4. **Test-driven granularity**: Creating types and interfaces just to make them independently testable, even when they have no independent meaning.

### The Cure

Ask for each type or package: **"Does this hide significant complexity behind its interface?"**

If the answer is no, it is a candidate for merging with its caller or neighbor. Fewer, deeper modules almost always produce simpler systems than many shallow ones.

## When Shallow Is Acceptable

Not every module needs to be deep. Shallow modules are acceptable when:

| Situation | Why It's OK | Example |
|-----------|------------|---------|
| **Dispatchers** | Routing logic is inherently shallow | A URL router that maps paths to handlers |
| **Interface adapters** | Translating between two deep modules | Converting between internal and external data formats |
| **Language/framework requirements** | The framework demands the layer | A RonyKit handler that must call `internal/app`; an `http.Handler` adapter |
| **Genuine one-liner utilities** | The abstraction is the name itself | `isEven(n)`, `clamp(value, lo, hi)` |
| **Entry points** | Top-level wiring that connects modules | `main()`, fx wiring in `module.go` / `service.go` |

The key is that these shallow modules should be **rare exceptions**, not the norm. If most of your modules are shallow, the design needs rethinking.

## Designing for Depth

### Strategy 1: Combine Related Functionality

Instead of:
```
RequestParser + RequestValidator + RequestAuthorizer + RequestHandler + ResponseBuilder
```

Consider:
```
RequestHandler (parses, validates, authorizes, handles, and builds response)
```

If these operations always happen together and share knowledge about the request format, combining them into one deep module eliminates four interfaces and produces a simpler system.

### Strategy 2: Hide Implementation Decisions

Ask: "What decisions does this module make that no one else needs to know about?"

Each hidden decision adds depth. Good examples:
- Buffer sizes and caching strategies
- Retry logic and backoff policies
- Connection pooling and lifecycle management
- Data format and serialization details
- Concurrency and locking strategies

### Strategy 3: Provide Defaults

Instead of requiring callers to specify everything:

```go
// Shallow: the caller must decide every option.
func Connect(host string, port int, timeout time.Duration, retries int, retryDelay time.Duration,
	tlsCert, tlsKey string, keepAlive time.Duration, bufferSize int) (*Client, error)

// Deep: sensible defaults hide the decisions; callers override only what
// they need, e.g. Open(ctx, dsn, WithTimeout(2*time.Second)).
func Open(ctx context.Context, dsn string, opts ...Option) (*Client, error)
```

### Strategy 4: Absorb Complexity

When two approaches exist -- one that is simpler for the module but pushes complexity to callers, and one that is harder to implement but simpler for callers -- choose the one that makes life easier for callers.

```go
// Pushes complexity to callers: they get raw bytes and must know the log
// format to parse them.
func (l *Log) ReadRaw(ctx context.Context) ([]byte, error)

// Absorbs complexity: callers get parsed, structured entries.
func (l *Log) Read(ctx context.Context) ([]Entry, error)
```

### Strategy 5: Question Every Interface Element

For each method, parameter, or return value in an interface, ask:
- "Do callers actually need this?"
- "Can the module decide this internally?"
- "Is there a simpler way to express this?"

Remove anything that does not earn its place. Every element in an interface is a cost that must be justified by the functionality it enables.

## Measuring Depth in Practice

### Quick Assessment

| Question | Deep | Shallow |
|----------|------|---------|
| How many methods in the interface? | Few (3-7) | Many (15+) |
| How many parameters per method? | Few (1-3) | Many (5+) |
| How long is the implementation? | Significantly larger than interface | About the same as interface |
| Can you describe the module in one sentence? | Yes | Need a paragraph |
| Does the module hide a non-trivial decision? | Yes, several | Not really |
| Would removing it require callers to duplicate code? | Lots of duplication | Minimal duplication |

### Depth Ratio

A rough heuristic: compare the lines of interface documentation to lines of implementation. If they are close to equal, the module is likely shallow. If the implementation is 5-10x larger than the interface description, the module is likely deep.

This is not about lines of code per se -- it is about the amount of hidden complexity relative to the exposed interface. A one-line interface like `runtime.GC()` that hides thousands of lines of garbage collection logic is extremely deep.

## Common Objections

### "But small classes are easier to test!"

Small types are easier to **unit test** in isolation, but the system is harder to **integration test** because you have more interfaces to fake, more interactions to verify, and more wiring to get right. Deeper modules that own more behavior are often easier to test at the level that matters: "does this feature work?"

### "But the Single Responsibility Principle says..."

SRP says a module should have "one reason to change," which is about **cohesion**, not about size. A module that handles all aspects of file I/O (opening, reading, writing, buffering, closing) changes for one reason: when file I/O requirements change. That is a single responsibility implemented deeply.

### "But what about separation of concerns?"

Separation of concerns is about keeping unrelated things apart, not about splitting related things into tiny pieces. If parsing, validating, and processing a request are all concerned with "handling a request," they can live in one module. Separate concerns that are genuinely independent (e.g., logging and business logic), not every step of a single workflow.
