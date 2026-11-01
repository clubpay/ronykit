# ee NATS bus

This module implements the [`ee`](../../../../ee) bus contract with NATS JetStream. It includes request/reply.

```go
b, err := nats.New(nats.WithURL("nats://localhost:4222")) // or WithConn(nc), or EE_NATS_URL
env := ee.MustNew(ee.WithBus(b), ee.WithStore(s))
```

## Layout

- One stream captures `<prefix>.>`. The default prefix is `ee`, and the stream name is `EE`. `New` creates or updates the stream unless `SkipStreamSetup()` is set.
- An envelope is published to `<prefix>.<event>.<partition>`. The partition is `fnv32a(PartitionKey) mod Partitions`, with default 16. Envelopes without a key are distributed by ID.
- Each ee group has one durable pull consumer for each partition. The consumer filters the group's events and uses `MaxAckPending=1`. One member processes a partition in strict order. Members compete for partitions. Events with the same key share a partition, so order is preserved across event types.

`WithPartitions` limits a group's parallelism. Changing this value on a live stream redistributes keys. Drain consumers before you change it.

## Delivery

- A failed handler sends a negative acknowledgment with delay `RetryBackoff × attempt`. The message is redelivered before later messages in the partition. After `MaxAttempts`, default 10, it goes to `OnDeadLetter` and is terminated. The partition then continues.
- Progress heartbeats run while a handler runs. They prevent the server from redelivering the message to another member after `WithAckWait`, default 30s.
- The envelope ID is the JetStream message ID. Republication within `WithDuplicateWindow`, default 2m, has no effect. This behavior handles outbox relay retries after a crash.

## Request/reply

`ee.Request` publishes through the stream like other events and uses the same ordering. A header contains a core NATS reply inbox. The handler's `ctx.Reply` answers on this inbox. The bus-level `Request` uses the same path.

## Tests

Set `EE_NATS_TEST_URL` to enable the tests. Each test uses its own stream.

```sh
docker run -d --rm --name ee-nats -p 54222:4222 nats:2.11-alpine -js
EE_NATS_TEST_URL=nats://localhost:54222 go test ./...
```
