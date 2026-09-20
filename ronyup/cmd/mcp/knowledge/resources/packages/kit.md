---
import_path: github.com/clubpay/ronykit/kit
short_name: kit
---

Low-level core: EdgeServer, Gateway, Cluster, Contract, Context, codecs. `rony` is built on `kit` — keep the dependency one-way.

## Usage Hint

Use `kit` when you are writing a **gateway, cluster, codec, or EdgeServer** integration, or migrating a pre-`rony` service (read
`architecture/migrating-kit-to-rony`).

Typical types: `kit.Context`, `kit.Gateway`, `kit.Cluster`, `kit.Contract`, `kit.RESTRouteSelector`, `kit.RPCRouteSelector`, `kit.Relay`,
`kit.RelayConfig`, `kit.RawMessage`.

Do **not** use `kit` in a scaffolded feature's `api/` handlers. Register routes with `rony.WithUnary` / `WithStream` / `WithRelay` and return
`rony.SetupOptionGroup` from `Desc()`. Building `kit.Contract` or wrapping `kit.RawMessage` to fake a proxy is a design violation — use
`rony.WithRelay`.
