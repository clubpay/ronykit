# ee Kafka bus

This module implements the [`ee`](../../../../ee) bus contract with Kafka and [franz-go](https://github.com/twmb/franz-go).

The bus does not implement `bus.Requester`, so `ee.Request` is not supported. Use NATS when a handler must answer the publisher.

```go
b, err := kafka.New(kafka.WithBrokers("localhost:9092")) // or EE_KAFKA_BROKERS
env := ee.MustNew(ee.WithBus(b), ee.WithStore(s))
```

Use `WithClientOptions(kgo.Opt...)` for TLS, SASL, and other client settings.

## Layout

- All events use one topic, `<prefix>.events`. The default prefix is `ee`. The envelope's partition key is the record key. Events with the same key share a partition and keep their order across event types. For example, `OrderCreated` stays before `OrderPaid`. Groups skip events without a matching subscription.
- `WithTopicRouter` can distribute events over several topics. Per-key order then applies only to events routed to the same topic.
- An ee group uses the consumer group `<prefix>.<group>`. Members divide the partitions.
- Topics are created on first use. `WithTopicDefaults` sets 16 partitions and broker-default replication. `SkipTopicSetup()` prevents automatic creation. `EnsureTopics` creates topics in advance.

## Delivery

- Partitions are processed in parallel. Records within a partition are processed in sequence. A failed record is retried in place with delay `RetryBackoff × attempt`. This retry blocks its partition. After `MaxAttempts`, default 10, the record goes to `OnDeadLetter` and its offset is committed.
- Offsets are committed only for handled records after each poll. Rebalances wait for poll processing and commit to finish. A partition therefore never moves with partially processed records.
- The producer is idempotent. Retries cannot reorder a partition.

At-least-once delivery can still produce duplicates. For example, a crash can occur between handling and commit. Use `ee.Dedup` when effects must apply once.

## Tests

Set `EE_KAFKA_TEST_BROKERS` to enable the tests. Each test uses its own topic.

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
