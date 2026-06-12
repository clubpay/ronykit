---
import_path: github.com/clubpay/ronykit/x/p
short_name: p
---

Low-level pooling primitives for hot paths: reusable timers, wait groups, and byte buffers backed by `sync.Pool`. Use these in
performance-sensitive code instead of creating your own pools or allocating per call.

## Usage Hint

- `p.AcquireTimer(d)` / `p.ReleaseTimer(t)` — reuse `*time.Timer` instead of `time.NewTimer` per call.
- `p.AcquireWaitGroup()` / `p.ReleaseWaitGroup(wg)` — pooled `*sync.WaitGroup` for fan-out/fan-in.
- `p.Bytes` / `BytesPool` — pooled, growable byte buffers (implement `io.Reader`/`io.Writer`) for serialization and I/O staging.

Always release back to the pool when done (typically via `defer`), and never retain a pooled value after release. This package is for
internal/hot-path use; for everyday slice/map work prefer `x/rkit`.
