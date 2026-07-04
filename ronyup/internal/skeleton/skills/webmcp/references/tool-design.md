# WebMCP tool design

Agents choose tools from **name + description + inputSchema** alone. Invest in
these fields more than you would in button labels.

## Naming

W3C constraints: 1–128 chars, `[A-Za-z0-9_.-]` only.

Convention for multi-feature apps:

```
<domain>_<verb>           orders_filter, users_update_role
<domain>_<noun>_<verb>    dashboard_date_range_set
```

Use a consistent domain prefix per feature area so agents can disambiguate when
dozens of tools are registered.

| Good | Bad |
|------|-----|
| `orders_filter` | `filter` (too vague) |
| `checkout_apply_coupon` | `applyCoupon` (camelCase) |
| `reports_export_csv` | `btn_export` (UI-coupled) |

## Descriptions

Write for an agent that has never seen your UI:

- Start with **what changes** and **when to use** the tool.
- Mention prerequisites ("Requires admin role", "User must be on checkout page").
- State side effects ("Creates a draft order", "Sends password reset email").

```text
Filter the orders table by status, date range, and search text.
Returns matching order summaries (id, status, total). Does not export data.
```

## Schema design

### Prefer enums and bounded types

```ts
status: z.enum(['pending', 'paid', 'shipped']).describe('Filter by order status'),
limit: z.number().int().min(1).max(100).default(20).describe('Max rows to return'),
```

### Return structured, actionable output

```ts
return {
  applied: { status: 'paid', from: '2026-01-01' },
  total: 42,
  items: [{ id: 'ord_1', status: 'paid', totalCents: 1999 }],
  hasMore: true,
};
```

Avoid bare `"OK"` or empty objects unless the tool is a true no-op.

### `outputSchema` (MCP-B extension)

`@mcp-b/react-webmcp` supports optional `outputSchema` for typing and MCP
clients. Native Chrome WebMCP does not enforce it yet — still document return
shapes in TypeScript for your own tests.

## Annotations

| Field | Use |
|-------|-----|
| `title` | Short human label in browser agent UI |
| `readOnlyHint: true` | List, get, search, export-preview — no server or client writes |
| `readOnlyHint: false` | Create, update, delete, navigate, submit |
| `untrustedContentHint: true` | Output includes user-generated or external HTML |

When in doubt, `readOnlyHint: false` — agents treat read tools as safe to call
liberally.

## Granularity

| Screen type | Tool strategy |
|-------------|---------------|
| Simple form | One `create_*` or `update_*` tool per form |
| Search + table | `*_search` or `*_filter` + optional `*_get` for row detail |
| Multi-step wizard | One tool per step + `*_get_progress` read tool |
| Dashboard | `*_set_date_range`, `*_set_segment`, `*_refresh_metrics` (see dashboards.md) |

Too many tools → agent confusion. Too few → agents over-parameterize one mega-tool.
Aim for **5–15 tools per major screen**, not per widget.

## Idempotency and errors

- Document whether repeated calls are safe (`description`: "Idempotent — safe to retry").
- Return `{ success: false, error: '...', code: 'NOT_FOUND' }` instead of throwing
  when the agent can recover.
- Throw only for unexpected failures (network down, 500).

## Tool catalog in design docs

Add a section to `docs/design/<app>-frontend-design.md`:

```markdown
## WebMCP tools

| Tool | R/W | Description |
|------|-----|-------------|
| orders_filter | write | Filter orders table; returns summaries |
| orders_get | read | Fetch one order by ID |
```

Keeps design review, QA, and agent discovery aligned.

## Anti-patterns

- Mirroring every REST endpoint 1:1 — expose **tasks**, not your OpenAPI map.
- Giant `input` objects with 20 optional fields — split tools or use enums.
- Descriptions that reference CSS selectors or pixel positions.
- Tools that only make sense when another tool was called first, without saying
  so in the description.
