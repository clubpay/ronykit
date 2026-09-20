Default HTTP for scaffolded apps is already wired inside `rony` (`std/gateways/fasthttp`). Do not add a second HTTP server or import
`net/http` to serve feature APIs.

| Need | Package |
|------|---------|
| Default REST / JSON APIs | already in `rony` (fasthttp) — no extra gateway import |
| Alternate HTTP engine | `std/gateways/silverhttp` — only if the user asks |
| Native WebSocket gateway | `std/gateways/fastws` — only if the user asks |
| MCP tool/resource server | `std/gateways/mcp` — only if the user asks |
| Redis-backed cluster | `std/clusters/rediscluster` — only if the user asks |
| Peer-to-peer cluster | `std/clusters/p2pcluster` — only if the user asks |

Std constructors follow `New(opts ...Option) (T, error)` plus `MustNew` that panics.

Do **not** invent extra gateways, raw `http.ListenAndServe`, or a custom `kit.EdgeServer` bootstrap in a scaffolded workspace. Server
construction lives in `pkg/runner` via `rony.NewServer`. Feature modules only export `Desc()` options.

For handler-initiated proxies read `architecture/handler-relay`. For the `kit` vs `rony` split read `packages/kit` and `packages/rony`.
