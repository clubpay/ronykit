package ee

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/ee/store"
)

// Env wires schemas, controllers and backends together. Register everything,
// then call Start; registration after Start fails with ErrStarted.
//
// All replicas of a service build the same Env, so every check done here at
// registration holds cluster-wide.
type Env struct {
	bus      bus.Bus
	store    store.Store
	codec    Codec
	now      func() time.Time
	log      *slog.Logger
	autoSave bool
	// maxRetries is how many times a unit of work is re-run on ErrConflict.
	maxRetries int
	outboxPoll time.Duration

	middlewares []Middleware

	mu           sync.RWMutex
	started      bool
	states       map[reflect.Type]*stateSchema
	statesByName map[string]*stateSchema
	// consumers maps an event name to the controller that owns it. An event
	// has exactly one consuming action.
	consumers   map[string]string
	controllers map[string]controller

	kick   chan struct{}
	subs   []bus.Subscription
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// controller is the untyped view the Env has of state and aggregate
// controllers, used to subscribe them on Start.
type controller interface {
	subscribedEvents() []string
	handle(ctx context.Context, msg bus.Message) error
}

type Option func(*Env)

func WithBus(b bus.Bus) Option              { return func(e *Env) { e.bus = b } }
func WithStore(s store.Store) Option        { return func(e *Env) { e.store = s } }
func WithCodec(c Codec) Option              { return func(e *Env) { e.codec = c } }
func WithClock(now func() time.Time) Option { return func(e *Env) { e.now = now } }
func WithLogger(l *slog.Logger) Option      { return func(e *Env) { e.log = l } }

// WithMaxRetries sets how many times a unit of work (an action, Update, ...)
// is re-run after a version conflict before the error is returned. Default 5.
func WithMaxRetries(n int) Option { return func(e *Env) { e.maxRetries = n } }

// WithOutboxPoll sets how often the relay polls the outbox when no local
// commit woke it up (e.g. entries left by a crashed replica). Default 1s.
func WithOutboxPoll(d time.Duration) Option { return func(e *Env) { e.outboxPoll = d } }

// WithAutoSave makes every unit of work save, at commit, the tracked states
// that changed since they were loaded or last saved. Without it (the
// default), nothing is written unless the handler calls Save.
func WithAutoSave() Option { return func(e *Env) { e.autoSave = true } }

func New(opts ...Option) (*Env, error) {
	e := &Env{
		codec:        JSONCodec,
		now:          time.Now,
		log:          slog.Default(),
		maxRetries:   5,
		outboxPoll:   time.Second,
		kick:         make(chan struct{}, 1),
		states:       map[reflect.Type]*stateSchema{},
		statesByName: map[string]*stateSchema{},
		consumers:    map[string]string{},
		controllers:  map[string]controller{},
	}
	for _, o := range opts {
		o(e)
	}

	if e.codec == nil {
		return nil, errors.New("ee: codec must not be nil")
	}

	return e, nil
}

func MustNew(opts ...Option) *Env {
	e, err := New(opts...)
	if err != nil {
		panic(err)
	}

	return e
}

// Start subscribes every controller to its events and starts the outbox
// relay.
func (e *Env) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.started {
		return ErrStarted
	}

	if e.bus == nil || e.store == nil {
		return errors.New("ee: both a bus and a store are required to start")
	}

	for name, c := range e.controllers {
		events := c.subscribedEvents()
		if len(events) == 0 {
			continue
		}

		sub, err := e.bus.Subscribe(ctx, name, events, c.handle)
		if err != nil {
			e.closeSubs()

			return fmt.Errorf("ee: subscribe %s: %w", name, err)
		}

		e.subs = append(e.subs, sub)
	}

	relayCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	e.cancel = cancel
	e.wg.Go(func() { e.relay(relayCtx) })
	e.started = true

	return nil
}

// Close unsubscribes every controller and stops the outbox relay. The bus
// and store are owned by the caller and stay open.
func (e *Env) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cancel == nil {
		return nil
	}

	err := e.closeSubs()

	e.cancel()
	e.cancel = nil
	e.wg.Wait()

	return err
}

func (e *Env) closeSubs() error {
	errs := make([]error, 0, len(e.subs))
	for _, s := range e.subs {
		errs = append(errs, s.Close())
	}

	e.subs = nil

	return errors.Join(errs...)
}

// Send publishes an event from outside any action. Inside actions use
// ctx.Emit, which goes through the transactional outbox instead.
func (e *Env) Send[E Event](ctx context.Context, ev E, opts ...SendOption) error {
	env, err := e.envelope(ev, opts)
	if err != nil {
		return err
	}

	return e.bus.Publish(ctx, env)
}

// Schemas describes every registered state schema. Storage backends may use
// it in their bootstrap/migration helpers.
func (e *Env) Schemas() []StateSchemaInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()

	out := make([]StateSchemaInfo, 0, len(e.states))
	for _, s := range e.states {
		out = append(out, s.info())
	}

	return out
}

func (e *Env) register(fn func() error) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.started {
		return ErrStarted
	}

	return fn()
}

// addController must be called inside register.
func (e *Env) addController(name string, c controller) error {
	if _, ok := e.controllers[name]; ok {
		return fmt.Errorf("%w: controller %q", ErrAlreadyRegistered, name)
	}

	e.controllers[name] = c

	return nil
}

// claim makes owner the only consumer of event. Must be called inside
// register.
func (e *Env) claim(event, owner string) error {
	if prev, ok := e.consumers[event]; ok {
		return fmt.Errorf("%w: event %q is already consumed by %s", ErrAlreadyRegistered, event, prev)
	}

	e.consumers[event] = owner

	return nil
}

func (e *Env) stateOf(t reflect.Type) (*stateSchema, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	s, ok := e.states[t]
	if !ok {
		return nil, fmt.Errorf("%w: state %s", ErrNotRegistered, t)
	}

	return s, nil
}
