# intent

`intent` is a goal-driven agent framework for RonyKit. An **Agent** wraps a `rony.Server` and composes:

- **Knowledge** — static catalog (prompts, facts) plus dynamic RAG retrieval
- **Skills** — named capabilities loaded on demand via the `activate_skill` tool
- **LLM pool** — pluggable model selection strategies
- **Memory** — session-scoped search/save backends
- **MCP servers** — external tool and context sources
- **Endpoints** — rony services via `ServiceDescriptor` and `EndpointMount`
- **Tasks** — task lifecycle via `TaskExecutor` (flow-backed durability later)

```
go get github.com/clubpay/ronykit/intent
```

## Layout

| Package / area | Purpose                                         |
|----------------|-------------------------------------------------|
| `intent`       | Agent wrapper, runtime loop, and all core types |
| `intent/errs`  | Shared error sentinels                          |

Core types live in the root `intent` package:

| Type group | Examples                                                |
|------------|---------------------------------------------------------|
| Embeddings | `Embedder`                                              |
| Knowledge  | `Knowledge`, `StaticStore`, `Retriever`, `Indexer`      |
| LLM        | `LLM`, `Pool`, `Selector`, `Message`                    |
| Memory     | `Memory`, `SessionMemory`, `MemoryRecord`               |
| MCP        | `MCPServer`, `MCPRegistry`, `MCPTool`                   |
| Tools      | `LocalTool`, `ToolRegistry`, `ToolExecutor`             |
| Skills     | `Skill`, `SkillCard`, `SkillRegistry`                   |
| Sessions   | `SessionManager`, `Session`, `LoadHistory`              |
| Endpoints  | `ServiceDescriptor`, `EndpointMount`, `ServiceRegistry` |
| Tasks      | `TaskExecutor`, `TaskHandle`, `TaskState`               |
| Agent      | `Agent`, `Option`, `RunTurn`, `TurnInput`, `TurnResult` |

## Skills

A **skill** is a named capability the agent loads on demand instead of carrying in
every prompt (progressive disclosure). Each turn the runtime advertises only a
catalog of `name: description` cards plus a synthetic `activate_skill` tool. When
the model calls `activate_skill` with a skill name, the runtime injects that
skill's full instructions and unlocks any tools scoped to it; those tools stay
hidden until activation.

```go
skills := intent.NewSkillRegistry()
_ = skills.Register(intent.Skill{
Name:         "billing",
Description:  "Handle refunds and billing questions.",
Instructions: "Confirm the order ID before issuing a refund...",
Tools:        []string{"issue_refund"}, // hidden until activated
})

agent := intent.New(
intent.WithLLMPool(pool),
intent.WithTools(tools),
intent.WithSkills(skills),
)
```

File-based skills load through `std/knowledge/static`; each `skills/*.md` file may
start with a YAML front-matter block (`name`, `description`, `tools`, `triggers`,
`examples`) followed by the instructions body. Use `store.Skills()` to obtain the registry.

The skill catalog is appended to the user message on the first LLM iteration so
routing hints stay adjacent to the request.

## Status

Scaffold and interfaces only. See [DESIGN.md](DESIGN.md) for architecture, planned `std/<kind>/<name>` implementations, and phased rollout.
