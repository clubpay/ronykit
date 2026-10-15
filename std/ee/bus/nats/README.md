# ee NATS bus

NATS JetStream implementation of the [`ee`](../../../../ee) bus contract,
including request/reply.

```go
b, err := nats.New(nats.WithURL("nats://localhost:4222")) // or WithConn(nc), or EE_NATS_URL
env := ee.MustNew(ee.WithBus(b), ee.WithStore(s))
```

## Layout

- One stream captures `<prefix>.>` (prefix default `ee`, stream `EE`). `New`
  creates or updates it unless `SkipStreamSetup()`.
- An envelope is published to `<prefix>.<event>.<partition>`, with
  `partition = fnv32a(PartitionKey) mod Partitions` (default 16). Envelopes
  without a key are spread by ID.
- Each ee group gets one durable pull consumer per partition, filtered to the
  group's events, with `MaxAckPending=1`. A partition is processed strictly in
  order, by one member at a time, and members (replicas) compete for
  partitions. Since a key's events share a partition, order holds across
  event types too.

`WithPartitions` caps a group's parallelism; changing it on a live stream
reshuffles keys, so drain consumers first.

## Delivery

- A failed handler NAKs with a delay of `RetryBackoff × attempt`; the message
  is redelivered before anything after it in the partition. After
  `MaxAttempts` (default 10) it goes to `OnDeadLetter` and is terminated so
  the partition moves on.
- While a handler runs, progress heartbeats keep the server from redelivering
  it to another member after `WithAckWait` (default 30s).
- The envelope ID is the JetStream message ID, so republishing within
  `WithDuplicateWindow` (default 2m), as the outbox relay does after a crash,
  is a no-op.

## Request/reply

`Request` publishes through the stream like any event (same ordering) with a
core-NATS reply inbox in a header; the handler's `msg.Reply` answers on it.

## Tests

Skipped unless `EE_NATS_TEST_URL` is set; each test uses its own stream.

```sh
docker run -d --rm --name ee-nats -p 54222:4222 nats:2.11-alpine -js
EE_NATS_TEST_URL=nats://localhost:54222 go test ./...
```
