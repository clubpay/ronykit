# ee Kafka bus

Kafka implementation of the [`ee`](../../../../ee) bus contract, on
[franz-go](https://github.com/twmb/franz-go). Request/reply is not supported:
the bus does not implement `bus.Requester`.

```go
b, err := kafka.New(kafka.WithBrokers("localhost:9092")) // or EE_KAFKA_BROKERS
env := ee.MustNew(ee.WithBus(b), ee.WithStore(s))
```

TLS, SASL and other client settings: `WithClientOptions(kgo.Opt...)`.

## Layout

- All events go to one topic, `<prefix>.events` (prefix default `ee`), keyed
  by the envelope's partition key. A key's events therefore share a partition
  and keep their order across event types (e.g. `OrderCreated` before
  `OrderPaid`). Groups skip events they did not subscribe to.
- `WithTopicRouter` can spread events over several topics; per-key order then
  only holds among events routed to the same topic.
- An ee group is the consumer group `<prefix>.<group>`; members split the
  partitions.
- Topics are created on first use with `WithTopicDefaults` (16 partitions,
  broker-default replication) unless `SkipTopicSetup()`; `EnsureTopics` does
  it up front.

## Delivery

- Partitions are processed in parallel, records of a partition sequentially.
  A failed record is retried in place (`RetryBackoff × attempt`), blocking its
  partition; after `MaxAttempts` (default 10) it goes to `OnDeadLetter` and
  its offset is committed.
- Offsets are committed only for handled records, after each poll. Rebalances
  wait for the poll's processing and commit to finish, so a partition never
  moves with half-processed records.
- The producer is idempotent, so retries cannot reorder a partition.

At-least-once still means duplicates are possible (e.g. a crash between
handling and committing); use `ee.Dedup` where effects must apply once.

## Tests

Skipped unless `EE_KAFKA_TEST_BROKERS` is set; each test uses its own topic.

```sh
docker run -d --rm --name ee-kafka -p 59092:59092 \
  -e KAFKA_NODE_ID=1 -e KAFKA_PROCESS_ROLES=broker,controller \
  -e KAFKA_LISTENERS=PLAINTEXT://:59092,CONTROLLER://:9093 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:59092 \
  -e KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER \
  -e KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT \
  -e KAFKA_CONTROLLER_QUORUM_VOTERS=1@localhost:9093 \
  -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
  -e KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=0 \
  apache/kafka:3.9.1
EE_KAFKA_TEST_BROKERS=localhost:59092 go test ./...
```
