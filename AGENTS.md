# AGENTS.md

Practical instructions for coding agents and contributors working in this repository.

- **Scope:** entire repository rooted at this directory.
- **Default approach:** prefer minimal, targeted changes over broad refactors.

## AI assistants

- **MCP:** `ronyup mcp` (see `.cursor/mcp.json`) — knowledge resources and scaffold tools.
- **Skill:** `.agents/skills/ronykit-framework/` ([Agent Skills](https://agentskills.io/specification) layout; Cursor discovers it automatically) — invoke `/ronykit-framework` for orchestration; conventions live in MCP, not in the skill body. MCP index: `references/mcp-map.md` under that directory.

For scaffolded application workspaces (outside this monorepo), MCP knowledge and tools are the source of truth for service layout and handler conventions.

---

## Project overview

RonyKit is a Go toolkit for building high-performance network services. It is organized as a **Go workspace** (`go.work`, Go 1.25+) containing **40+ independent modules**.

Two abstraction levels exist:

- **`kit/`** — low-level core (EdgeServer, Gateway, Cluster, Contract, Context)
- **`rony/`** — high-level, batteries-included framework built on `kit`

Additional top-level modules:

- **`intent/`** — goal-driven agent framework (LLM pools, tools, skills, sessions, knowledge); wraps `rony.Server`
- **`flow/`** — durable workflow helpers (Temporal SDK integration)
- **`stub/`** — client stub generation (Go / TypeScript)
- **`ronyup/`** — project scaffolding CLI and MCP server
- **`testenv/`** — testing environment utilities

Implementations for gateways, clusters, LLMs, memory, knowledge, embedders, and MCP clients live under **`std/<kind>/<name>`** as separate `go.mod` modules (same pattern as `std/gateways/fasthttp`). The `intent` core stays dependency-light and defines interfaces only.

Extended utilities live under **`x/`** (di, telemetry, apidoc, cache, datasource, i18n, ratelimit, batch, settings, testkit, rkit, p).

---

## Repository layout

```
kit/              Core building blocks (EdgeServer, contracts, context, codecs)
rony/             High-level framework (server, typed context, state management)
intent/           Agent runtime (LLM pool, tools, skills, sessions, knowledge)
flow/             Workflow helpers (Temporal SDK integration)
stub/             Client stub generation (Go / TypeScript)
ronyup/           Project scaffolding CLI
testenv/          Testing environment utilities
std/
  gateways/       fasthttp, silverhttp, fastws, mcp
  clusters/       rediscluster, p2pcluster
  llms/           langchaingo, ollama
  embedders/      langchaingo
  knowledge/      static, chromem, milvus
  memories/       inmem, postgres, sqlite, sqlstore
  mcpclients/     gosdk
x/                Extended utilities (di, telemetry, apidoc, cache, …)
example/          Runnable examples (ex-01 through ex-12)
scripts/          Build & maintenance scripts
docs/             Diagrams and extra documentation
.agents/skills/   Agent skill definitions (ronykit-framework)
```

**Examples:** `ex-01`–`ex-04` use `kit` directly; later examples use `rony`. Notable entries:

- `ex-11-mcp` — MCP gateway
- `ex-12-agent` — intent agent with static knowledge, tools, and multi-model LLM pool

---

## Build & development commands

```sh
make setup       # Install tools (gotestsum, golangci-lint, markdownfmt)
make format-md   # Format all Markdown (*.md); preserves YAML frontmatter (see scripts/format-markdown.sh)
make test        # Run tests across all modules (excludes example/, ronyup/)
make lint        # Lint all modules (excludes example/)
make vet         # go vet all modules (excludes example/)
make tidy        # go mod tidy all modules (excludes example/)
```

- Test a single module: `cd <module> && go test ./...`
- Test `ronyup` specifically: `cd ronyup && go test ./...`

---

## Testing

- Framework: standard **`testing`** with **`github.com/stretchr/testify`** (`assert` / `require`). Prefer table-driven tests and `t.Run` subtests.
- Test runner: `gotestsum` with `--format pkgname-and-test-fails`.
- Coverage: `covermode=atomic`, generates `coverage.out` per module.
- For logic changes, run targeted tests for affected modules first; use `make test` for broader validation when feasible.
- Report any unrun checks and why they were skipped.
- Always run `make lint` after a task is done.

---

## Architecture quick reference

**Request flow:**

```
Client -> Gateway -> northBridge -> EdgeServer -> Contract lookup -> Handler chain -> Response
```

**Key abstractions:**

- **EdgeServer** — orchestrator that binds Gateways, Clusters, and Services.
- **Gateway** — inbound traffic (fasthttp, silverhttp, fastws, mcp).
- **Cluster** — multi-instance coordination (Redis, libp2p P2P).
- **Service** — logical grouping of Contracts.
- **Contract** — single API operation (input/output types, route selectors, handlers).
- **Context** — request-scoped state with four storage layers: per-request, per-connection, per-service (local), cluster-wide.

**Encoding:** JSON, Protobuf, MessagePack, multipart/form-data, or custom.

**Routing:** `RESTRouteSelector` (HTTP method + path) and `RPCRouteSelector` (predicate).

**Agent stack (`intent`):**

```
Client -> rony endpoint -> intent.Agent.RunTurn -> LLM pool -> tools / skills / memory
```

- **Agent** — composes knowledge, LLM pool, memory, tools, skills, sessions, and rony services.
- **LLM pool** — multiple `intent.LLM` backends with pluggable selection (`first`, `random`, `priority`, `enforced`).
- **std adapters** — e.g. `std/llms/ollama`, `std/llms/langchaingo`, `std/knowledge/static`, `std/memories/inmem`.

See `intent/README.md` and `intent/DESIGN.md` before changing agent behavior.

---

## std module conventions

- Each module under `std/` and `x/` has its own `go.mod`; respect module boundaries.
- Prefer RonyKit packages over third-party/stdlib substitutes where equivalents exist (`x/rkit`, `x/di`, `x/settings`, `x/telemetry/*`, `rony/errs`, etc.).
- **Workflows:** use `flow` only — do not import `go.temporal.io/sdk` directly (denied by workspace `.golangci.yml` in scaffolded apps).
- **Constructors:** std modules follow the gateway pattern — `New(opts ...Option) (T, error)` plus `MustNew(opts ...Option) T` that panics on error. Unset config fields may be filled from environment variables (see each package).
- **LLM adapters:** `std/llms/ollama` and `std/llms/langchaingo` expose functional options and env-backed defaults (`OLLAMA_*`, `OPENAI_*`). Wire multiple backends into `intent.NewLLMPool`; do not add a separate pool orchestration module.

---

## Developing `ronyup` (scaffolder + MCP server)

`ronyup/` produces scaffolded app workspaces and serves the `ronyup mcp` knowledge/tools that drive agents. When changing how agents build apps, edit these — not this file.

- **MCP knowledge** lives in `ronyup/cmd/mcp/knowledge/` and is embedded + auto-loaded by `loader.go` (no registration needed — dropping a `.md` file is enough):
  - `server/instructions.md` — always injected on MCP connect (the portable, cross-agent backbone).
  - `resources/{architecture,packages,characteristics,tools}/*.md` — read on demand (`knowledge://ronyup/<category>/<name>`).
  - `prompts/*.md` — workflow prompts (e.g. `design-new-service`, `design-frontend`).
- **Bundled skills** live in `ronyup/internal/skeleton/skills/<id>/SKILL.md` and must be registered in `skillCatalog` (`ronyup/cmd/setup/skills.go`); authoring heuristic (single file vs. `rules/` tree) is in `internal/skeleton/skills/README.md`. `copySkills` ships the whole directory.
- **Enforcement is gates and hooks, not prose** (small models ignore advisory text):
  - `ronyup/cmd/mcp/tools/scaffold/gate.go` — the SRS/SDD design gate that `scaffold_feature` enforces (portable across MCP clients).
  - `ronyup/internal/skeleton/workspace/verify.sh` (backend) and `internal/skeleton/frontend/verify.sh` — the `make verify` quality gates (test coverage, design-doc, lint/build/stories).
  - `ronyup/internal/skeleton/workspace/.cursor/hooks/*.sh` + `hooks.jsontmpl` — Cursor `stop` hooks that auto-loop the agent until a gate passes.
- **Keep the layers consistent.** A behavior change usually touches several surfaces at once: `server/instructions.md`, the relevant `resources/`/`prompts/` doc, `AGENTS.mdtmpl`, the `ronykit-framework` skill (both `.agents/skills/.../SKILL.md` and the skeleton copy under `ronyup/internal/skeleton/workspace/.agents/skills/`), and `references/mcp-map.md`.
- **Verify with** `cd ronyup && go test ./...` (the scaffold integration tests assert the generated gate/hook files exist).

---

## Code standards

- Preserve existing architecture and naming conventions.
- Add comments only when logic is non-obvious.
- Do not introduce unrelated formatting churn.
- Keep edits scoped to the specific task.
- Read the relevant package/module README before changing behavior.
- Update docs when behavior or developer workflow changes.

---

## Git hygiene

- Keep commits atomic with clear intent.
- Do not revert unrelated local changes owned by the user.
- Avoid destructive git operations unless explicitly requested.
- Main branch: `main`.
