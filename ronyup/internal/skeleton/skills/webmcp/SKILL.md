---
name: webmcp
description: >-
  Build agent-friendly web apps with the W3C Web Model Context API (WebMCP).
  Use when exposing frontend features as structured tools for in-browser AI
  agents, adding document.modelContext tools to React/Next.js apps, designing
  tool schemas for dashboards or admin UIs, or making existing UI automatable
  without DOM scraping.
---

# WebMCP (agent-friendly frontend)

Expose your app's **real business logic** as structured tools agents can
discover and call — instead of clicking buttons or scraping the DOM.

Every agent-ready page ships two interfaces:

1. **Human UI** — the existing components, forms, and navigation.
2. **Agent tool surface** — named tools with JSON Schemas that call the same
   handlers, stores, and API clients the UI uses.

Keep one source of truth. If a human click and an agent tool call would
diverge, fix the architecture before adding more tools.

## When to use

- Adding WebMCP / `document.modelContext` support to a React or Next.js app.
- Designing tool names, schemas, and descriptions for dashboards, admin
  consoles, or operational UIs.
- Making filters, search, CRUD, navigation, or checkout flows agent-callable.
- Reviewing whether a frontend is safe and reliable for AI agents.

Combine with `nextjs-modern` (App Router boundaries), `dashboard-ui` (tool
granularity for admin screens), `ux-quality` (human + agent UI stay in sync),
and `frontend-testing` (verify tools, not just components).

## WebMCP vs related protocols

| Surface | What it is | When to use |
|---------|------------|-------------|
| **WebMCP** (`document.modelContext`) | W3C draft: page registers client-side tools for in-browser agents | Your **website** exposes what it can do while the user has the tab open |
| **MCP** (Model Context Protocol) | Server protocol: tools on a backend or desktop process | Backend services, DBs, CLIs — not a substitute for live page state |
| **mcp-dashboards** | MCP server that renders charts inside AI chat clients | Visualize data **in the conversation** — complementary, not WebMCP |
| **DOM automation** (Playwright, etc.) | Agent guesses selectors | Last resort when no tools exist; brittle and slow |

WebMCP is **not** a replacement for your REST/OpenAPI backend. Agents still
need auth, rate limits, and server validation. Tools are a typed façade over
logic you already run in the browser.

Spec: [webmachinelearning.github.io/webmcp](https://webmachinelearning.github.io/webmcp/)
· Chrome guide: [developer.chrome.com/docs/ai/webmcp](https://developer.chrome.com/docs/ai/webmcp)

## Default stack (use unless the project already chose otherwise)

| Package | Role |
|---------|------|
| `@mcp-b/global` | Polyfills `document.modelContext` in browsers without native WebMCP; steps aside when native support exists |
| `@mcp-b/react-webmcp` | `useWebMCP` hook — register/unregister tools with React lifecycle |
| `zod` | Input validation + JSON Schema generation for tool args |

Docs: [docs.mcp-b.ai](https://docs.mcp-b.ai) · Packages:
[WebMCP-org/npm-packages](https://github.com/WebMCP-org/npm-packages)

Do **not** default to `opentiny/webmcp-sdk` or ad-hoc `navigator.modelContext`
wrappers. The canonical API is `document.modelContext` per the W3C spec;
`@mcp-b/react-webmcp` aligns with it and handles cleanup via `AbortSignal`.

## Implementation workflow

Copy this checklist and track progress:

```
WebMCP integration:
- [ ] Inventory user journeys → candidate tools (verbs, not UI widgets)
- [ ] Map each tool to existing app logic (hook, store action, Server Action)
- [ ] Add WebMCP provider + polyfill (client-only)
- [ ] Register tools at the component/route that owns the state
- [ ] Set annotations (readOnlyHint, title) and tight schemas
- [ ] Mirror agent mutations in the UI (same state path as human clicks)
- [ ] Test with modelContextTesting API (see references/testing.md)
- [ ] Document tools in the frontend design doc (tool table)
```

### 1. Design tools from journeys, not DOM nodes

Bad: `click_save_button`, `fill_email_input`.
Good: `orders_filter`, `users_update_role`, `dashboard_set_date_range`.

One tool = one **user- or agent-meaningful operation**. Prefer coarse tools
that match how you would describe the task in plain language.

For dashboard/admin screens, read `references/dashboards.md` for KPI/filter/table
tool patterns. Pair with the `dashboard-ui` skill for layout.

### 2. Share logic with the UI

```tsx
// Pattern: extract the operation once, wire UI + tool to it
async function setDateRange(range: DateRange) {
  await persistRange(range);
  queryClient.invalidateQueries({ queryKey: ['metrics'] });
}

// Human: <DatePicker onChange={setDateRange} />
// Agent: useWebMCP({ name: 'dashboard_set_date_range', handler: setDateRange, ... })
```

Never duplicate business rules inside a tool handler. Call the same function
the button, form, or Server Action uses.

### 3. Bootstrap the runtime (React / Next.js)

1. Install: `pnpm add @mcp-b/global @mcp-b/react-webmcp zod`
2. Create a **client-only** `WebMCPProvider` that imports `@mcp-b/global` once.
3. Mount the provider inside the app tree — not in the root RSC layout.
4. Register tools in the leaf components or route modules that own the relevant
   state. Re-register when deps change (pass a deps array to `useWebMCP`).

Full Next.js App Router, Vite, and provider patterns:
`references/react-integration.md`.

### 4. Write schemas agents can rely on

- Every property needs a `description` (`.describe()` in Zod).
- Use `enum` / `z.enum()` for fixed sets — never free-text when choices are
  finite.
- Return structured JSON from handlers; include enough context for the agent
  to decide the next step (`items`, `total`, `appliedFilters`, `error`).
- Set `annotations.readOnlyHint: true` for getters, list, and search tools.
- Set `annotations.title` for human-facing labels in browser agent UIs.

Detailed naming, granularity, and anti-patterns: `references/tool-design.md`.

### 5. Keep the UI in sync

Agent tool calls run on the page's event loop. After a mutating tool:

- Update React state / TanStack Query cache the same way the UI would.
- Navigate with the router if the human flow would change pages.
- Surface errors in the UI, not only in the tool return value.

If `SubmitEvent.agentInvoked` is available (declarative forms, future API),
branch only for **telemetry or consent** — not for alternate business logic.

### 6. Security (non-negotiable)

WebMCP is gated by **origin isolation** and the **`tools` permissions
policy** (default `self`). Do not disable these to "make it work."

Inside every **mutating** tool handler, re-check what your UI already enforces:

- Session / auth (user is logged in).
- Authorization (user may act on this resource).
- Input validation (Zod on the server for Server Actions too).

Treat tools like public API endpoints that happen to run in the browser:

- Never expose admin-only operations without the same guards as the UI.
- Mark destructive flows clearly in `description`; prefer confirmation in the
  human UI when side effects are irreversible.
- Use `readOnlyHint: true` only when the handler truly does not mutate state.

## Declarative forms (optional, limited today)

The W3C spec defines HTML form annotations (`toolname`, `tooldescription`) so
agents can submit structured forms. The declarative section is still incomplete
in the spec — prefer the **imperative** `registerTool` / `useWebMCP` API for
production apps until declarative support is stable in your target browsers.

When declarative forms land fully, use them for simple CRUD forms; keep
imperative tools for multi-step flows, dashboards, and stateful filters.

## Desktop agent development (optional)

To let **Cursor / Claude Desktop** call tools on `localhost` during
development, use `@mcp-b/webmcp-local-relay` (see MCP-B docs). This is a dev
ergonomic — production agents are in-browser (e.g. Gemini in Chrome).

## Progressive reference map

| Working on… | Read |
|-------------|------|
| React / Next.js setup, provider, `useWebMCP` | `references/react-integration.md` |
| Tool names, schemas, granularity, hints | `references/tool-design.md` |
| Local testing, Chrome flags, relay | `references/testing.md` |
| Dashboard / admin tool catalogs | `references/dashboards.md` |

## Anti-patterns

- One tool per button or DOM element.
- Tool handlers that call `fetch` with different validation than the UI.
- Registering all tools in a single global file far from the owning feature.
- `readOnlyHint: true` on tools that write to server or local storage.
- Skipping tests — tools regress silently when props or API shapes change.
- Building a parallel "agent API" on the backend when the page already has the
  state and logic client-side.

## Checklist before "done"

- [ ] Every tool maps to a real user journey; names are `snake_case`, ≤128 chars,
      alphanumeric + `_` `.` `-` only.
- [ ] Handlers call shared app logic; UI reflects agent-driven state changes.
- [ ] Inputs validated with Zod; descriptions on every field; enums for finite sets.
- [ ] `readOnlyHint` / `title` annotations set correctly.
- [ ] `@mcp-b/global` loads before any `useWebMCP` call; Next.js boundaries correct.
- [ ] Tools tested via `navigator.modelContextTesting` or MCP-B testing helpers.
- [ ] `pnpm typecheck`, `pnpm lint`, `pnpm build`, `pnpm test` pass for the app.
- [ ] Frontend design doc lists exposed tools (name, description, read/write).

## Attribution

Patterns and API usage align with the
[W3C Web Model Context API](https://webmachinelearning.github.io/webmcp/) and
[MCP-B](https://docs.mcp-b.ai) (`@mcp-b/global`, `@mcp-b/react-webmcp`).
Consumer-side agent skill for browsing WebMCP sites:
[nekuda-ai/webmcp](https://github.com/nekuda-ai/webmcp).
