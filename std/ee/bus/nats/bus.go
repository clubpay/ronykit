// Package nats is a NATS JetStream implementation of the ee bus contract
// (github.com/clubpay/ronykit/ee/bus).
//
// Layout: one stream captures "<prefix>.>". An envelope is published to
// "<prefix>.<event>.<partition>", where partition = hash(PartitionKey) mod
// Partitions. Each ee group gets one durable pull consumer per partition with
// MaxAckPending=1, so every partition is processed strictly in order while
// the group's members (replicas) compete for partitions. A failed delivery is
// NAKed and redelivered before anything after it in its partition.
package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/clubpay/ronykit/ee/bus"
	"github.com/clubpay/ronykit/x/rkit"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// EnvURL is used when neither WithConn nor WithURL is given.
const EnvURL = "EE_NATS_URL"

// headerReplyTo carries the core-NATS inbox of a Request.
const headerReplyTo = "Ee-Reply-To"

type config struct {
	conn        *nats.Conn
	url         string
	natsOpts    []nats.Option
	prefix      string
	stream      string
	partitions  int
	replicas    int
	maxAge      time.Duration
	dupWindow   time.Duration
	ackWait     time.Duration
	maxAttempts int
	backoff     time.Duration
	onDead      func(bus.Envelope, error)
	setup       bool
}

type Option func(*config)

// WithConn uses an existing connection. The caller keeps ownership.
func WithConn(nc *nats.Conn) Option { return func(c *config) { c.conn = nc } }

// WithURL connects (and on Close, disconnects) with the given options.
func WithURL(url string, opts ...nats.Option) Option {
	return func(c *config) { c.url, c.natsOpts = url, opts }
}

// WithPrefix sets the subject prefix (default "ee"); the stream captures
// "<prefix>.>" and is named after it unless WithStream is given.
func WithPrefix(p string) Option { return func(c *config) { c.prefix = p } }

func WithStream(name string) Option { return func(c *config) { c.stream = name } }

// WithPartitions sets the number of ordered lanes per group (default 16).
// Changing it on a live stream reshuffles keys: drain consumers first.
func WithPartitions(n int) Option { return func(c *config) { c.partitions = n } }

// WithStreamReplicas sets the stream replication factor (default 1).
func WithStreamReplicas(n int) Option { return func(c *config) { c.replicas = n } }

// WithMaxAge bounds how long the stream keeps messages (default 7 days).
func WithMaxAge(d time.Duration) Option { return func(c *config) { c.maxAge = d } }

// WithDuplicateWindow sets JetStream's dedup window on envelope IDs
// (default 2m), which absorbs outbox republishing after a relay crash.
func WithDuplicateWindow(d time.Duration) Option { return func(c *config) { c.dupWindow = d } }

// WithAckWait sets how long the server waits for an ack before redelivering
// (default 30s). Long handlers are kept alive with progress heartbeats.
func WithAckWait(d time.Duration) Option { return func(c *config) { c.ackWait = d } }

// MaxAttempts caps deliveries of one envelope (default 10; 0 = unlimited).
func MaxAttempts(n int) Option { return func(c *config) { c.maxAttempts = n } }

// RetryBackoff is multiplied by the attempt number for the NAK delay
// (default 100ms).
func RetryBackoff(d time.Duration) Option { return func(c *config) { c.backoff = d } }

// OnDeadLetter receives envelopes that exhausted MaxAttempts; they are then
// terminated so their partition moves on.
func OnDeadLetter(fn func(bus.Envelope, error)) Option { return func(c *config) { c.onDead = fn } }

// SkipStreamSetup leaves stream creation to operators; New then only checks
// that the stream exists.
func SkipStreamSetup() Option { return func(c *config) { c.setup = false } }

// Bus implements bus.Bus and bus.Requester.
type Bus struct {
	cfg   config
	nc    *nats.Conn
	owned bool
	js    jetstream.JetStream

	mu     sync.Mutex
	closed bool
	subs   map[*subscription]struct{}
}

var (
	_ bus.Bus       = (*Bus)(nil)
	_ bus.Requester = (*Bus)(nil)
)

func New(opts ...Option) (*Bus, error) {
	cfg := config{
		prefix:      "ee",
		partitions:  16,
		replicas:    1,
		maxAge:      7 * 24 * time.Hour,
		dupWindow:   2 * time.Minute,
		ackWait:     30 * time.Second,
		maxAttempts: 10,
		backoff:     100 * time.Millisecond,
		onDead:      func(bus.Envelope, error) {},
		setup:       true,
	}
	for _, o := range opts {
		o(&cfg)
	}

	if cfg.stream == "" {
		cfg.stream = streamName(cfg.prefix)
	}

	if err := validSubjectPart(cfg.prefix); err != nil || cfg.partitions < 1 {
		return nil, fmt.Errorf("nats: invalid prefix %q or partitions %d", cfg.prefix, cfg.partitions)
	}

	b := &Bus{cfg: cfg, nc: cfg.conn, subs: map[*subscription]struct{}{}}

	if b.nc == nil {
		url := cfg.url
		if url == "" {
			url = os.Getenv(EnvURL)
		}

		if url == "" {
			return nil, errors.New("nats: no connection: use WithConn, WithURL or " + EnvURL)
		}

		nc, err := nats.Connect(url, cfg.natsOpts...)
		if err != nil {
			return nil, fmt.Errorf("nats: connect: %w", err)
		}

		b.nc, b.owned = nc, true
	}

	js, err := jetstream.New(b.nc)
	if err != nil {
		b.closeConn()

		return nil, err
	}

	b.js = js

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := b.ensureStream(ctx); err != nil {
		b.closeConn()

		return nil, err
	}

	return b, nil
}

func MustNew(opts ...Option) *Bus {
	return rkit.Must(New(opts...))
}

func (b *Bus) ensureStream(ctx context.Context) error {
	if !b.cfg.setup {
		_, err := b.js.Stream(ctx, b.cfg.stream)
		if err != nil {
			return fmt.Errorf("nats: stream %s: %w", b.cfg.stream, err)
		}

		return nil
	}

	_, err := b.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:       b.cfg.stream,
		Subjects:   []string{b.cfg.prefix + ".>"},
		Storage:    jetstream.FileStorage,
		Replicas:   b.cfg.replicas,
		MaxAge:     b.cfg.maxAge,
		Duplicates: b.cfg.dupWindow,
	})
	if err != nil {
		return fmt.Errorf("nats: stream %s: %w", b.cfg.stream, err)
	}

	return nil
}

// Stream is the JetStream stream name, e.g. for operational tooling.
func (b *Bus) Stream() string { return b.cfg.stream }

func (b *Bus) subject(event, key, id string) string {
	key = rkit.Coalesce(key, id)

	h := fnv.New32a()
	_, _ = h.Write([]byte(key))

	return b.cfg.prefix + "." + event + "." + rkit.IntToStr(int(h.Sum32()%uint32(b.cfg.partitions)))
}

// Publish sends envelopes asynchronously and waits for every ack. Envelopes
// to the same subject keep their order. The envelope ID is the JetStream
// message ID, so republishing within the duplicate window is a no-op.
func (b *Bus) Publish(ctx context.Context, envs ...bus.Envelope) error {
	futures := make([]jetstream.PubAckFuture, 0, len(envs))

	for _, e := range envs {
		msg, err := b.msg(e)
		if err != nil {
			return err
		}

		f, err := b.js.PublishMsgAsync(msg)
		if err != nil {
			return fmt.Errorf("nats: publish %s: %w", e.Event, err)
		}

		futures = append(futures, f)
	}

	for _, f := range futures {
		select {
		case <-f.Ok():
		case err := <-f.Err():
			return fmt.Errorf("nats: publish: %w", err)
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (b *Bus) msg(e bus.Envelope) (*nats.Msg, error) {
	if err := validSubjectPart(e.Event); err != nil {
		return nil, err
	}

	data, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("nats: encode %s: %w", e.Event, err)
	}

	msg := nats.NewMsg(b.subject(e.Event, e.PartitionKey, e.ID))
	msg.Data = data
	msg.Header.Set(jetstream.MsgIDHeader, e.ID)

	return msg, nil
}

// Request publishes env through the stream, like Publish, and waits for the
// consuming handler's reply on a core-NATS inbox.
func (b *Bus) Request(ctx context.Context, env bus.Envelope) (bus.Envelope, error) {
	msg, err := b.msg(env)
	if err != nil {
		return bus.Envelope{}, err
	}

	inbox := b.nc.NewRespInbox()

	sub, err := b.nc.SubscribeSync(inbox)
	if err != nil {
		return bus.Envelope{}, err
	}

	defer func() { _ = sub.Unsubscribe() }()

	msg.Header.Set(headerReplyTo, inbox)

	if _, err := b.js.PublishMsg(ctx, msg); err != nil {
		return bus.Envelope{}, fmt.Errorf("nats: request %s: %w", env.Event, err)
	}

	reply, err := sub.NextMsgWithContext(ctx)
	if err != nil {
		return bus.Envelope{}, err
	}

	var out bus.Envelope

	return out, json.Unmarshal(reply.Data, &out)
}

func (b *Bus) Subscribe(
	ctx context.Context,
	group string,
	events []string,
	h bus.Handler,
) (bus.Subscription, error) {
	for _, e := range events {
		if err := validSubjectPart(e); err != nil {
			return nil, err
		}
	}

	b.mu.Lock()
	closed := b.closed
	b.mu.Unlock()

	if closed {
		return nil, errors.New("nats: bus closed")
	}

	subCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	s := &subscription{b: b, h: h, ctx: subCtx, cancel: cancel}

	for p := range b.cfg.partitions {
		filters := rkit.Map(
			events,
			func(e string) string { return b.cfg.prefix + "." + e + "." + rkit.IntToStr(p) },
		)

		cons, err := b.js.CreateOrUpdateConsumer(ctx, b.cfg.stream, jetstream.ConsumerConfig{
			Durable:        consumerName(group, p),
			FilterSubjects: filters,
			AckPolicy:      jetstream.AckExplicitPolicy,
			AckWait:        b.cfg.ackWait,
			MaxAckPending:  1, // strict order within the partition
			MaxDeliver:     -1,
			DeliverPolicy:  jetstream.DeliverAllPolicy,
		})
		if err != nil {
			_ = s.Close()

			return nil, fmt.Errorf("nats: consumer %s/%d: %w", group, p, err)
		}

		cc, err := cons.Consume(s.handle, jetstream.PullMaxMessages(1))
		if err != nil {
			_ = s.Close()

			return nil, fmt.Errorf("nats: consume %s/%d: %w", group, p, err)
		}

		s.consumers = append(s.consumers, cc)
	}

	b.mu.Lock()
	b.subs[s] = struct{}{}
	b.mu.Unlock()

	return s, nil
}

// Close stops every subscription and, if the bus opened the connection,
// drains and closes it.
func (b *Bus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()

		return nil
	}

	b.closed = true
	subs := rkit.MapKeysToArray(b.subs)
	b.mu.Unlock()

	for _, s := range subs {
		_ = s.Close()
	}

	b.closeConn()

	return nil
}

func (b *Bus) closeConn() {
	if b.owned {
		_ = b.nc.Drain()
	}
}

type subscription struct {
	b         *Bus
	h         bus.Handler
	ctx       context.Context //nolint:containedctx // lifetime of the subscription's handlers
	cancel    context.CancelFunc
	consumers []jetstream.ConsumeContext
	once      sync.Once
}

func (s *subscription) Close() error {
	s.once.Do(func() {
		for _, cc := range s.consumers {
			cc.Stop()
		}

		for _, cc := range s.consumers {
			<-cc.Closed()
		}

		s.cancel()

		s.b.mu.Lock()
		delete(s.b.subs, s)
		s.b.mu.Unlock()
	})

	return nil
}

func (s *subscription) handle(msg jetstream.Msg) {
	var env bus.Envelope
	if err := json.Unmarshal(msg.Data(), &env); err != nil {
		s.b.cfg.onDead(bus.Envelope{}, fmt.Errorf("nats: undecodable message on %s: %w", msg.Subject(), err))
		_ = msg.Term()

		return
	}

	m := bus.Message{Envelope: env}
	if inbox := msg.Headers().Get(headerReplyTo); inbox != "" {
		m.Reply = func(_ context.Context, r bus.Envelope) error {
			data, err := json.Marshal(r)
			if err != nil {
				return err
			}

			return s.b.nc.Publish(inbox, data)
		}
	}

	stop := s.heartbeat(msg)
	err := s.h(s.ctx, m)

	stop()

	if err == nil {
		_ = msg.Ack()

		return
	}

	attempt := 1
	if md, mdErr := msg.Metadata(); mdErr == nil {
		attempt = int(md.NumDelivered)
	}

	if s.b.cfg.maxAttempts > 0 && attempt >= s.b.cfg.maxAttempts {
		s.b.cfg.onDead(env, err)

		_ = msg.Term()

		return
	}

	_ = msg.NakWithDelay(s.b.cfg.backoff * time.Duration(attempt))
}

// heartbeat tells the server the message is still being worked on, so a slow
// handler is not redelivered to another member mid-flight.
func (s *subscription) heartbeat(msg jetstream.Msg) (stop func()) {
	done := make(chan struct{})
	t := time.NewTicker(max(s.b.cfg.ackWait/3, 10*time.Millisecond))

	go func() {
		defer t.Stop()

		for {
			select {
			case <-done:
				return
			case <-t.C:
				_ = msg.InProgress()
			}
		}
	}()

	return func() { close(done) }
}

// consumerName derives a durable name from an ee group, which may contain
// characters NATS rejects (e.g. "state:order"); a hash of the original keeps
// sanitized names distinct.
func consumerName(group string, partition int) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(group))

	clean := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}

		return '_'
	}, group)

	return fmt.Sprintf("%s_%08x_p%d", clean, h.Sum32(), partition)
}

func streamName(prefix string) string {
	return strings.ToUpper(strings.NewReplacer(".", "_", "-", "_").Replace(prefix))
}

// validSubjectPart rejects names that would break subject tokenization.
func validSubjectPart(s string) error {
	if s == "" || strings.ContainsAny(s, " \t\r\n*>") || strings.HasPrefix(s, ".") ||
		strings.HasSuffix(s, ".") || strings.Contains(s, "..") {
		return fmt.Errorf("nats: %q is not usable in a subject", s)
	}

	return nil
}
