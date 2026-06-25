# intent — Design Notes

Status: **Phase 1 implemented** — core runtime, std adapters, and tests. See phases below for remaining work.

## Summary

`intent` is RonyKit's goal-driven agent framework. An **Agent** wraps a `rony.Server` and composes knowledge, LLM pools, session memory, MCP
clients, tools, and rony endpoints.

Implementations live in **`std/<kind>/<name>`** sub-modules (separate `go.mod` per module, like `std/gateways/mcp`). The `intent` core
module stays dependency-light and defines interfaces only.

---

## Position in RonyKit

```
kit        → wire protocol & edge primitives
rony       → services, handlers, server
flow       → durable workflows (Temporal)
intent     → autonomous agents (interfaces + agent runtime)
std/*      → concrete implementations (gateways, LLM adapters, memory backends, …)
```

| Concern                            | Module                       | Role                                     |
|------------------------------------|------------------------------|------------------------------------------|
| Agent runtime                      | `intent`                     | Compose capabilities; wrap `rony.Server` |
| Task durability                    | `flow`                       | Long-running task state machines         |
| Agent exposes API                  | `rony` + `ServiceDescriptor` | HTTP/RPC endpoints via `EndpointMount`   |
| Agent exposes tools (host)         | `std/gateways/mcp`           | MCP server gateway (exists)              |
| Agent uses external tools (client) | `std/mcpclients/gosdk`       | MCP client adapter (planned)             |
| LLM providers                      | `std/llms/langchaingo`       | langchaingo adapter (planned)            |

---

## Core Concepts

### Agent

Configuration container + `rony.Server` wrapper. Holds references to Knowledge, LLM pool, Memory, MCP registry, tool registry, task
registry, and executor. Does **not** embed heavy SDK dependencies.

```go
agent := intent.New(
intent.WithName("support"),
intent.WithKnowledge(knowledge),
intent.WithLLMPool(pool),
intent.WithMemory(mem),
intent.WithMCPServers(mcpReg),
intent.WithService(serviceDesc),
intent.WithServerOption(rony.Listen(":8080")),
)
```

### Session

A **session** is one conversation or task run. It owns:

- Session ID
- `memory.SessionMemory` handle
- Conversation message history (stored via memory)
- Optional enforced LLM model (`llm.StrategyEnforced`)
- Metadata (user, task ID)

Session is **not yet scaffolded** — planned as `intent/session`.

```
Agent (long-lived)
  └── Session (per conversation / task run)
        ├── SessionMemory
        ├── History → []llm.Message
        └── Selection override (enforced model)
```

### Knowledge vs Memory

|                             | Scope             | Mutability    | Example                               |
|-----------------------------|-------------------|---------------|---------------------------------------|
| **Knowledge (static)**      | Agent-wide        | Setup time    | System prompt, facts                  |
| **Knowledge (dynamic/RAG)** | Agent-wide corpus | Indexed async | Product docs in vector DB             |
| **Skills**                  | Agent-wide        | Setup time    | "billing" capability loaded on demand |
| **Memory**                  | Per session       | Runtime       | "User asked about order #42"          |

- `knowledge.StaticStore` — `List`, `Get`
- `knowledge.Retriever` — `Retrieve` (RAG at request time)
- `knowledge.Indexer` — `Index`, `DeleteSource` (ingestion pipeline)
- `memory.Memory.ForSession(id)` — session-scoped `Save`, `Search`, `Delete`, `Clear`

### LLM Pool

Multiple LLMs per agent; selection is pluggable:

| Strategy   | Behavior                             |
|------------|--------------------------------------|
| `random`   | Pick any available model             |
| `first`    | First in pool                        |
| `priority` | Ordered fallback by `Model.Priority` |
| `enforced` | Session/task specifies `ModelID`     |

Adapter target: `github.com/tmc/langchaingo/llms.Model`.

### MCP

Two distinct roles — do not conflate:

| Direction             | Module                 | Purpose                                                  |
|-----------------------|------------------------|----------------------------------------------------------|
| **Server** (host)     | `std/gateways/mcp`     | Rony service exposes tools to external MCP clients       |
| **Client** (consumer) | `std/mcpclients/gosdk` | Agent connects to external MCP servers for tools/context |

Agent-side interface: `intent/mcp.Server`, `Registry`, `ClientFactory`.  
Adapter target: `github.com/modelcontextprotocol/go-sdk/mcp.ClientSession`.

### Tools

**Gap (not yet scaffolded):** no bridge between `llm.ToolCall` (model output) and tool execution (MCP or local).

Planned `intent/tool` package:

```
ToolRegistry
  ├── local tools   (Go handlers)
  └── MCP tools     (via mcp.Server.CallTool)

Flow:
  1. Registry.List() → []llm.ToolDefinition   (for LLM context)
  2. LLM returns llm.ToolCall
  3. Registry.Execute(ctx, call) → llm.Message (RoleTool)
  4. Append to session history → next LLM turn
```

### Skills

A **skill** is a named capability loaded on demand rather than carried in every
prompt (progressive disclosure). Skills are first-class, not just `KindSkill`
knowledge entries.

```
Skill { Name, Description, Instructions, Tools, Meta }
SkillRegistry { List → []SkillCard (cheap), Get(name) → Skill (full body) }
```

Runtime flow per turn:

```
1. List skill cards → inject a compact catalog (name: description) as a system message
2. Advertise a synthetic activate_skill tool (single `name` enum of skill names)
3. Skill-scoped Tools stay hidden until their skill is active
4. Model calls activate_skill(name):
     → inject skill.Instructions as the tool result
     → unlock that skill's Tools for the rest of the turn
5. Calls to gated-but-inactive tools are rejected with a hint to activate first
```

`intent.DefaultSkillRegistry` is the in-memory implementation; `std/knowledge/static`
loads skills from `skills/*.md` files with optional YAML front-matter
(`name`, `description`, `tools`) and exposes them via `Store.Skills()`.

### Endpoints (rony services)

Agents define HTTP/RPC handlers through **ServiceDescriptor** and **EndpointMount** instead of receiving `*rony.Server` directly:

```
ServiceDescriptor.Mount(EndpointMount)
  → rony.Setup(...) on internal server
```

`EndpointMount.Setup` mirrors `rony.Setup` and keeps handler registration abstracted at the intent layer.

### Tasks

Protocol-level work units are separate from rony endpoint registration:

```
TaskExecutor → Start / Signal / Cancel (flow-backed later)
Task.State   → pending → running → waiting → completed | failed | cancelled
```

Durable execution uses `flow` workflows; task handlers remain thin rony unary/stream handlers mounted via `ServiceDescriptor`.

### Embedder

Vector memory and RAG retrieval require embeddings. Planned interface:

```go
// intent/embed/embedder.go (planned)
type Embedder interface {
Embed(ctx context.Context, texts []string) ([][]float32, error)
Dimensions() int
}
```

Adapter target: langchaingo `embeddings.Embedder`.  
Implementations: `std/embedders/langchaingo`.

---

## Agent Loop

One turn inside a session:

```
1. Load static knowledge     → knowledge.StaticStore.List/Get (prompts, facts)
2. RAG retrieve              → knowledge.Retriever.Retrieve(query)
3. Load session history      → memory.SessionMemory.Search / stored messages
4. List available tools      → tool.Registry.List → llm.ToolDefinition
5. Advertise skills          → SkillRegistry.List → catalog + activate_skill tool
6. Select LLM                → llm.Pool.Select(session selection)
7. Generate                  → llm.LLM.Generate(request)
8. If tool calls:
     a. activate_skill → inject Instructions, unlock skill tools
     b. otherwise tool.Registry.Execute(each call) (gated tools rejected if inactive)
     c. append RoleTool messages
     d. goto 6 (tools recomposed from active skills)
9. Save turn to session memory
10. Return response
```

Streaming variant uses `llm.LLM.Stream` with incremental chunks.

---

## Module Layout

### `intent/` (core — interfaces + agent runtime)

```
intent/
├── DESIGN.md           ← this document
├── README.md
├── agent.go            ← Agent wraps rony.Server
├── agent_options.go
├── endpoint.go         ← ServiceDescriptor, EndpointMount
├── task.go             ← TaskExecutor, TaskHandle
```

**Dependency rule:** `intent` core imports only `rony`, `kit`, and stdlib. No langchaingo, MCP SDK, chromem-go, or Milvus in core.

### `std/<kind>/<name>` (implementations)

Each implementation is an independent Go module with its own `go.mod`, registered in workspace `go.work`.

| Kind          | Name          | Implements                              | Key dependency                       |
|---------------|---------------|-----------------------------------------|--------------------------------------|
| `llms`        | `langchaingo` | `intent/llm.LLM`, `Pool`                | `github.com/tmc/langchaingo`         |
| `embedders`   | `langchaingo` | `intent/embed.Embedder`                 | `github.com/tmc/langchaingo`         |
| `memories`    | `inmem`       | `intent/memory.Memory`                  | none                                 |
| `memories`    | `postgres`    | `intent/memory.Memory`                  | Postgres                             |
| `memories`    | `chromem`     | `intent/memory.Memory`                  | `github.com/philippgille/chromem-go` |
| `memories`    | `milvus`      | `intent/memory.Memory`                  | Milvus client                        |
| `knowledge`   | `static`      | `intent/knowledge.StaticStore`          | filesystem / embed                   |
| `knowledge`   | `chromem`     | `intent/knowledge.Retriever`, `Indexer` | chromem-go                           |
| `knowledge`   | `milvus`      | `intent/knowledge.Retriever`, `Indexer` | Milvus client                        |
| `mcpclients`  | `gosdk`       | `intent/mcp.Server`, `ClientFactory`    | `modelcontextprotocol/go-sdk`        |
| `tools`       | `mcp`         | `intent/tool` MCP-backed executor       | `intent/mcp` + go-sdk                |
| `tasks`       | `flow`        | `intent/task.Executor`                  | `flow` (Temporal)                    |

Naming follows existing convention: `github.com/clubpay/ronykit/std/<kind>/<name>`.

---

## Current Scaffold (done)

| Package               | Status | Notes                             |
|-----------------------|--------|-----------------------------------|
| `intent`              | ✅      | Agent + options                   |
| `intent/knowledge`    | ✅      | Static + RAG interfaces           |
| `intent/memory`       | ✅      | Session-scoped                    |
| `intent/llm`          | ✅      | SDK-aligned types                 |
| `intent/llm/selector` | ✅      | Strategy constants + Selector     |
| `intent/mcp`          | ✅      | Client-side; SDK-aligned          |
| `intent/task`         | ✅      | TaskExecutor                      |
| `intent/endpoint`     | ✅      | ServiceDescriptor + EndpointMount |
| `intent/session`      | ❌      | Planned                           |
| `intent/tool`         | ❌      | Planned                           |
| `intent/embed`        | ❌      | Planned                           |
| `intent/errors`       | ❌      | Planned (via `rony/errs`)         |

---

## Planned Interface Additions

### `intent/session`

```go
type Session struct {
ID       string
Memory   memory.SessionMemory
Metadata map[string]string
}

type Manager interface {
Create(ctx context.Context, opts ...Option) (*Session, error)
Get(ctx context.Context, id string) (*Session, error)
Close(ctx context.Context, id string) error
}
```

Session manager lives on Agent; tasks receive a session ID or handle.

### `intent/tool`

```go
type Tool interface {
Definition() llm.ToolDefinition
Execute(ctx context.Context, args json.RawMessage) (llm.Message, error)
}

type Registry interface {
Register(t Tool) error
RegisterMCP(server mcp.Server) error // bridge MCP tools into registry
Definitions(ctx context.Context) ([]llm.ToolDefinition, error)
Execute(ctx context.Context, call llm.ToolCall) (llm.Message, error)
}
```

### `intent/embed`

```go
type Embedder interface {
Embed(ctx context.Context, texts []string) ([][]float32, error)
Dimensions() int
}
```

### `intent/endpoint`

```go
type ServiceDescriptor struct {
Name  string
Mount func (m EndpointMount) error
}

type EndpointMount struct { /* wraps internal *rony.Server */ }

func Setup[S State[A], A Action](m EndpointMount, serviceName string, init InitState[S, A], opts ...SetupOption[S, A])
```

### `intent/errors`

Typed errors using `rony/errs` patterns:

- `ErrSessionNotFound`
- `ErrModelNotFound`
- `ErrToolNotFound`
- `ErrKnowledgeNotSupported` (partial implementation)
- `ErrMCPServerUnavailable`

---

## Task + flow Integration

```
Client request
  → rony task handler (thin)
    → task.Executor.Start(ctx, "summarize", input)
      → flow workflow (durable)
        → agent loop (session, knowledge, LLM, tools, memory)
        → state transitions (pending → running → waiting → done)
```

- **rony handler** — HTTP/RPC entry point; validates input, delegates to executor
- **task.Executor** — abstracts local vs flow-backed execution
- **flow workflow** — durable state; survives restarts; signals for human-in-the-loop (`StateWaiting`)

`std/tasks/flow` implements `task.Executor` using `flow.SDK`.

---

## Phased Implementation

### Phase 1 — Foundation (interfaces + first adapters)

1. Add `intent/session`, `intent/tool`, `intent/embed`, `intent/errors`
2. `std/llms/langchaingo` — LLM adapter
3. `std/memories/inmem` — session memory for dev/test
4. `std/knowledge/static` — filesystem prompts/skills/facts
5. `std/mcpclients/gosdk` — MCP client adapter

**Exit criteria:** agent loop works end-to-end with in-memory session, static knowledge, langchaingo LLM, MCP tools.

### Phase 2 — RAG + vector backends

1. `std/embedders/langchaingo`
2. `std/knowledge/chromem` — RAG retriever + indexer
3. `std/memories/chromem` — vector session memory (optional)

**Exit criteria:** agent retrieves relevant documents at request time.

### Phase 3 — Production backends

1. `std/memories/postgres`
2. `std/knowledge/milvus`
3. `std/memories/milvus`
4. Built-in LLM selectors (`random`, `first`, `priority`, `enforced`)

**Exit criteria:** production-grade persistence and retrieval.

### Phase 4 — Tasks + durability

1. `std/tasks/flow` — flow-backed `task.Executor`
2. Example task as rony service with state machine
3. Session enforced model from task context

**Exit criteria:** long-running tasks survive restarts.

---

## Open Questions

| # | Question                                                                                  | Current lean                                          |
|---|-------------------------------------------------------------------------------------------|-------------------------------------------------------|
| 1 | Single `Knowledge` interface vs separate `WithStaticKnowledge` + `WithRetriever` options? | Separate options on Agent; composite optional         |
| 2 | Generic tasks (`task.Executor[IN, OUT]`) vs `any`?                                        | Start with `any`; revisit when flow integration lands |
| 3 | Session ID generation — UUID (`x/rkit`) vs caller-provided?                               | Default UUID; caller can override                     |
| 4 | Tool naming — prefix MCP tools with server name (`server/tool`)?                          | Yes, to avoid collisions across MCP servers           |
| 5 | Agent loop — explicit runtime type vs embedded in Agent?                                  | `Agent.RunTurn(ctx, TurnInput)`; runtime is internal  |
| 6 | Streaming — expose on Agent/Session or task-only?                                         | Session-level `RunTurnStream`                         |

---

## Testing Strategy

- **Interface compile tests** — `var _ Interface = (*Impl)(nil)` in each `std/*` module
- **In-memory integration tests** — Phase 1 loop with `std/memories/inmem` + mock LLM
- **Framework:** standard `testing` + testify (`assert` / `require`) (repo convention)
- **No SDK calls in `intent` core tests** — mocks only

---

## References

- [langchaingo](https://github.com/tmc/langchaingo/) — LLM + embeddings adapter target
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) — MCP client/server
- [chromem-go](https://github.com/philippgille/chromem-go) — embedded vector store
- `std/gateways/mcp` — existing MCP **server** gateway in RonyKit
- `flow` — durable workflow module
- `docs/architecture.md` — RonyKit request flow
