// Package kafka is a Kafka implementation of the ee bus contract
// (github.com/clubpay/ronykit/ee/bus), on franz-go.
//
// Layout: every event goes to one topic ("<prefix>.events" by default), keyed
// by the envelope's partition key, so all events of a key share a partition
// and keep their relative order even across event types. An ee group is a
// Kafka consumer group; members split the partitions. Each partition is
// processed sequentially: a failed record is retried in place, so nothing
// after it in that partition runs first. Offsets are committed only for
// processed records (at-least-once).
//
// Request/reply is not supported: Bus does not implement bus.Requester.
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/clubpay/ronykit/ee/bus"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
)

// EnvBrokers is used when WithBrokers is not given (comma separated).
const EnvBrokers = "EE_KAFKA_BROKERS"

type config struct {
	brokers     []string
	kgoOpts     []kgo.Opt
	prefix      string
	topicFor    func(event string) string
	partitions  int32
	replication int16
	autoCreate  bool
	maxAttempts int
	backoff     time.Duration
	onDead      func(bus.Envelope, error)
}

type Option func(*config)

func WithBrokers(brokers ...string) Option { return func(c *config) { c.brokers = brokers } }

// WithClientOptions adds franz-go options (TLS, SASL, ...) to every client.
func WithClientOptions(opts ...kgo.Opt) Option {
	return func(c *config) { c.kgoOpts = append(c.kgoOpts, opts...) }
}

// WithPrefix namespaces the topic ("<prefix>.events") and consumer groups
// ("<prefix>.<group>"). Default "ee".
func WithPrefix(p string) Option { return func(c *config) { c.prefix = p } }

// WithTopicRouter spreads events over several topics. Per-key order then only
// holds among events routed to the same topic.
func WithTopicRouter(fn func(event string) string) Option { return func(c *config) { c.topicFor = fn } }

// WithTopicDefaults sets partitions (default 16) and replication factor
// (default -1, the broker's default) for topics the bus creates.
func WithTopicDefaults(partitions int32, replication int16) Option {
	return func(c *config) { c.partitions, c.replication = partitions, replication }
}

// SkipTopicSetup leaves topic creation to operators.
func SkipTopicSetup() Option { return func(c *config) { c.autoCreate = false } }

// MaxAttempts caps deliveries of one record (default 10; 0 = unlimited).
func MaxAttempts(n int) Option { return func(c *config) { c.maxAttempts = n } }

// RetryBackoff is multiplied by the attempt number between retries (default
// 100ms). Retries block the record's partition.
func RetryBackoff(d time.Duration) Option { return func(c *config) { c.backoff = d } }

// OnDeadLetter receives envelopes that exhausted MaxAttempts; their offset is
// then committed so the partition moves on.
func OnDeadLetter(fn func(bus.Envelope, error)) Option { return func(c *config) { c.onDead = fn } }

// Bus implements bus.Bus.
type Bus struct {
	cfg      config
	producer *kgo.Client
	admin    *kadm.Client

	mu      sync.Mutex
	closed  bool
	created map[string]bool
	subs    map[*subscription]struct{}
}

var _ bus.Bus = (*Bus)(nil)

func New(opts ...Option) (*Bus, error) {
	cfg := config{
		prefix:      "ee",
		partitions:  16,
		replication: -1,
		autoCreate:  true,
		maxAttempts: 10,
		backoff:     100 * time.Millisecond,
		onDead:      func(bus.Envelope, error) {},
	}
	for _, o := range opts {
		o(&cfg)
	}

	if len(cfg.brokers) == 0 {
		for b := range strings.SplitSeq(os.Getenv(EnvBrokers), ",") {
			if b = strings.TrimSpace(b); b != "" {
				cfg.brokers = append(cfg.brokers, b)
			}
		}
	}

	if len(cfg.brokers) == 0 {
		return nil, errors.New("kafka: no brokers: use WithBrokers or " + EnvBrokers)
	}

	if cfg.topicFor == nil {
		topic := cfg.prefix + ".events"
		cfg.topicFor = func(string) string { return topic }
	}

	producer, err := kgo.NewClient(append([]kgo.Opt{
		kgo.SeedBrokers(cfg.brokers...),
		kgo.ProducerLinger(time.Millisecond),
	}, cfg.kgoOpts...)...)
	if err != nil {
		return nil, fmt.Errorf("kafka: %w", err)
	}

	return &Bus{
		cfg:      cfg,
		producer: producer,
		admin:    kadm.NewClient(producer),
		created:  map[string]bool{},
		subs:     map[*subscription]struct{}{},
	}, nil
}

func MustNew(opts ...Option) *Bus {
	b, err := New(opts...)
	if err != nil {
		panic(err)
	}

	return b
}

// Topics returns the topics events are routed to, e.g. for operational
// tooling.
func (b *Bus) Topics(events ...string) []string {
	var out []string

	for _, e := range events {
		if t := b.cfg.topicFor(e); !slices.Contains(out, t) {
			out = append(out, t)
		}
	}

	return out
}

// EnsureTopics creates the topics of events if they are missing.
func (b *Bus) EnsureTopics(ctx context.Context, events ...string) error {
	if !b.cfg.autoCreate {
		return nil
	}

	b.mu.Lock()

	var missing []string

	for _, t := range b.Topics(events...) {
		if !b.created[t] {
			missing = append(missing, t)
		}
	}
	b.mu.Unlock()

	if len(missing) == 0 {
		return nil
	}

	res, err := b.admin.CreateTopics(ctx, b.cfg.partitions, b.cfg.replication, nil, missing...)
	if err != nil {
		return fmt.Errorf("kafka: create topics: %w", err)
	}

	for _, r := range res {
		if r.Err != nil && !errors.Is(r.Err, kerr.TopicAlreadyExists) {
			return fmt.Errorf("kafka: create topic %s: %w", r.Topic, r.Err)
		}
	}

	b.mu.Lock()
	for _, t := range missing {
		b.created[t] = true
	}
	b.mu.Unlock()

	return nil
}

// Publish produces envelopes and waits for every ack. The producer is
// idempotent, so retries cannot reorder records of a partition.
func (b *Bus) Publish(ctx context.Context, envs ...bus.Envelope) error {
	events := make([]string, len(envs))
	for i, e := range envs {
		events[i] = e.Event
	}

	if err := b.EnsureTopics(ctx, events...); err != nil {
		return err
	}

	records := make([]*kgo.Record, len(envs))

	for i, e := range envs {
		data, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("kafka: encode %s: %w", e.Event, err)
		}

		records[i] = &kgo.Record{Topic: b.cfg.topicFor(e.Event), Value: data}
		if e.PartitionKey != "" {
			records[i].Key = []byte(e.PartitionKey)
		}
	}

	if err := b.producer.ProduceSync(ctx, records...).FirstErr(); err != nil {
		return fmt.Errorf("kafka: produce: %w", err)
	}

	return nil
}

func (b *Bus) Subscribe(
	ctx context.Context,
	group string,
	events []string,
	h bus.Handler,
) (bus.Subscription, error) {
	b.mu.Lock()
	closed := b.closed
	b.mu.Unlock()

	if closed {
		return nil, errors.New("kafka: bus closed")
	}

	if err := b.EnsureTopics(ctx, events...); err != nil {
		return nil, err
	}

	cl, err := kgo.NewClient(append([]kgo.Opt{
		kgo.SeedBrokers(b.cfg.brokers...),
		kgo.ConsumerGroup(b.cfg.prefix + "." + group),
		kgo.ConsumeTopics(b.Topics(events...)...),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
		// Processing and committing finish before partitions can move, so a
		// rebalance never hands a half-processed batch to another member.
		kgo.BlockRebalanceOnPoll(),
		kgo.FetchMaxWait(500 * time.Millisecond),
	}, b.cfg.kgoOpts...)...)
	if err != nil {
		return nil, fmt.Errorf("kafka: consumer %s: %w", group, err)
	}

	subCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	s := &subscription{
		b:      b,
		cl:     cl,
		h:      h,
		events: events,
		ctx:    subCtx,
		cancel: cancel,
		done:   make(chan struct{}),
	}

	b.mu.Lock()
	b.subs[s] = struct{}{}
	b.mu.Unlock()

	go s.run(subCtx)

	return s, nil
}

// Close stops every subscription (leaving their groups) and the producer.
func (b *Bus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()

		return nil
	}

	b.closed = true
	subs := make([]*subscription, 0, len(b.subs))

	for s := range b.subs {
		subs = append(subs, s)
	}
	b.mu.Unlock()

	for _, s := range subs {
		_ = s.Close()
	}

	b.producer.Close()

	return nil
}

type subscription struct {
	b      *Bus
	cl     *kgo.Client
	h      bus.Handler
	events []string
	ctx    context.Context //nolint:containedctx // lifetime of the poll loop
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

func (s *subscription) Close() error {
	s.once.Do(func() {
		s.cancel()
		<-s.done
		s.cl.Close() // commits nothing more; leaves the group

		s.b.mu.Lock()
		delete(s.b.subs, s)
		s.b.mu.Unlock()
	})

	return nil
}

func (s *subscription) run(ctx context.Context) {
	defer close(s.done)

	for {
		fetches := s.cl.PollFetches(ctx)
		if ctx.Err() != nil {
			s.cl.AllowRebalance()

			return
		}

		var (
			wg        sync.WaitGroup
			mu        sync.Mutex
			processed []*kgo.Record
		)

		// Partitions run in parallel; records of one partition in order.
		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			wg.Go(func() {
				if last := s.process(p.Records); last != nil {
					mu.Lock()

					processed = append(processed, last)
					mu.Unlock()
				}
			})
		})

		wg.Wait()

		if len(processed) > 0 {
			// Commit with a fresh context: a Close racing with this commit
			// must not drop work that is already done.
			commitCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			_ = s.cl.CommitRecords(commitCtx, processed...)

			cancel()
		}

		s.cl.AllowRebalance()
	}
}

// process handles records of one partition in order and returns the last one
// fully handled (whose offset may be committed), or nil.
func (s *subscription) process(records []*kgo.Record) *kgo.Record {
	var last *kgo.Record

	for _, r := range records {
		if !s.deliver(r) {
			return last // closing: leave the rest uncommitted
		}

		last = r
	}

	return last
}

// deliver retries r until it succeeds, is dead-lettered or the subscription
// closes (false).
func (s *subscription) deliver(r *kgo.Record) bool {
	var env bus.Envelope
	if err := json.Unmarshal(r.Value, &env); err != nil {
		s.b.cfg.onDead(
			bus.Envelope{},
			fmt.Errorf("kafka: undecodable record %s/%d@%d: %w", r.Topic, r.Partition, r.Offset, err),
		)

		return true
	}

	if !slices.Contains(s.events, env.Event) {
		return true // another group's event on a shared topic
	}

	for attempt := 1; ; attempt++ {
		err := s.h(s.ctx, bus.Message{Envelope: env})
		if err == nil {
			return true
		}

		if s.ctx.Err() != nil {
			return false
		}

		if s.b.cfg.maxAttempts > 0 && attempt >= s.b.cfg.maxAttempts {
			s.b.cfg.onDead(env, err)

			return true
		}

		select {
		case <-s.ctx.Done():
			return false
		case <-time.After(s.b.cfg.backoff * time.Duration(attempt)):
		}
	}
}
