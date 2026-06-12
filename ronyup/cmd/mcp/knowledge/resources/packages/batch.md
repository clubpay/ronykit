---
import_path: github.com/clubpay/ronykit/x/batch
short_name: batch
---

Generic micro-batcher that coalesces many concurrent single-item calls into bounded batches processed by one function. Use this instead of
hand-rolling channels, tickers, and worker pools when you need to amortize per-call overhead (bulk DB writes, bulk external API calls,
fan-in aggregation).

## Usage Hint

Define a batch function over typed input/output, then enter items concurrently:

```go
b := batch.NewMulti[InItem, OutItem](
func (tagID string, entries []batch.Entry[InItem, OutItem]) {
// process up to batchSize entries at once
for _, e := range entries {
in := e.Value() // typed input
e.Callback(result) // deliver the typed result for this entry
}
},
batch.WithBatchSize(100),
batch.WithMaxWorkers(int32(runtime.NumCPU()*10)),
batch.WithMinWaitTime(5*time.Millisecond),
)

entry := batch.NewEntry[InItem, OutItem](in, func (out OutItem) { /* receive result */ })
b.EnterAndWait(tagID, entry) // blocks until this entry's batch is processed
```

- `NewMulti` keys batches by `tagID` (e.g. per-tenant); use a single tag for a global batcher.
- Build items with `batch.NewEntry[IN, OUT](value, callback)`; read input via `Entry.Value()` and return output via `Entry.Callback(out)`
  inside the batch func.
- Tune with `WithBatchSize`, `WithMaxWorkers`, `WithMinWaitTime`.
- Use `EnterAndWait(tagID, entry)` for request/response flows; `Enter(tagID, entry)` for fire-and-forget.
