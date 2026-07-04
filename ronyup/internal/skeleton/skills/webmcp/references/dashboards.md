# WebMCP for dashboards and admin UIs

Dashboards are ideal WebMCP surfaces: agents need to **set filters**, **read
KPIs**, and **drill into rows** — exactly the operations your ops users perform.

Pair this reference with the `dashboard-ui` skill for layout and the `webmcp`
tool-design rules for schemas.

## Mental model

```
Human: scans KPI strip → adjusts date range → opens table row
Agent: calls dashboard_set_date_range → dashboard_get_kpis → orders_get
```

Expose **decision-level** tools, not chart-rendering internals.

## Recommended tool set (per dashboard)

| Tool | R/W | Purpose |
|------|-----|---------|
| `dashboard_get_kpis` | read | Hero metrics for current filters |
| `dashboard_set_date_range` | write | Change global time window |
| `dashboard_set_segment` | write | Product/region/team filter |
| `dashboard_refresh` | read | Force refetch; return stale hint |
| `<entity>_search` | read | Table search with pagination |
| `<entity>_get` | read | Single record detail |
| `<entity>_update_*` | write | Allowed admin mutations only |

Adjust domain prefix (`orders_`, `users_`, `infra_`) per app.

## KPI tool example

```tsx
useWebMCP({
  name: 'dashboard_get_kpis',
  description:
    'Return current KPI values (revenue, orders, conversion, AOV) for the active date range and segment filters.',
  inputSchema: {},
  annotations: { title: 'Get KPIs', readOnlyHint: true },
  handler: async () => ({
    range: filters.dateRange,
    segment: filters.segment,
    kpis: {
      revenue: metrics.revenue,
      orders: metrics.orders,
      conversionRate: metrics.conversionRate,
      aov: metrics.aov,
    },
    asOf: new Date().toISOString(),
  }),
});
```

Return numbers agents can compare across calls (`asOf`, `range`, `segment`).

## Filter tools vs table tools

**Global filters** (date, segment) should invalidate the same query keys the
UI uses so KPIs and tables stay consistent.

**Table tools** accept `cursor` / `page` / `limit` and return `{ items, total,
hasMore }` — mirror your API pagination, do not return unbounded lists.

## Charts and visualization

- WebMCP tools return **data**, not pixels. Agents reason over JSON.
- For rich inline charts inside **chat clients**, use a separate MCP server like
  [mcp-dashboards](https://github.com/KyuRish/mcp-dashboards) — that is not
  WebMCP and does not replace on-page tools.
- If the dashboard exports CSV/PDF, expose `reports_export_csv` with explicit
  side-effect description.

## Admin mutation discipline

Back-office apps often bundle dangerous actions. For each write tool:

1. Same authorization as the UI button.
2. Description states irreversibility.
3. Return the **new state** (`previousRole`, `role`) for agent audit trails.

Avoid a single `admin_execute` tool with a string `action` parameter — agents
will hallucinate action names.

## Empty, loading, and error states

Read tools should encode UI state:

```ts
return {
  status: 'loading' | 'ready' | 'error',
  kpis: status === 'ready' ? { ... } : null,
  error: status === 'error' ? 'Upstream metrics timeout' : undefined,
};
```

Agents can then wait or call `dashboard_refresh` instead of guessing.

## Storybook and WebMCP

Register tools in the same components that Storybook stories render so designers
and agents see consistent behavior. See `storybook` skill for story coverage;
add a "WebMCP tools" subsection to the story description when tools are present.
