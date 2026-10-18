---
name: dashboard-ui
description: >-
  Design and build data-dense dashboards, admin panels, analytics views, and
  operational SaaS interfaces (Operate mode) — information hierarchy, KPI cards,
  tables, filter bars, charts, empty/loading states, and density controls. Use
  when building back-office tools, observability UIs, financial/ops consoles,
  CRM views, or any screen where users scan, compare, and act on structured data.
---

# Dashboard & Data-Dense UI (Operate Mode)

Dashboards and operational tools are **workbenches, not billboards**. Users open
them dozens of times a day to answer a question or complete a task in seconds.
Every pixel of decoration, oversized header, or page-load animation steals
attention from the data.

Pair with `design-tokens` (surfaces & status tokens), `typography` (tabular
numerals & fixed scales), `shadcn` (DataTable, Sheet, Command, Dialog),
`ux-quality` (keyboard flows & form states), and `ui-polish` (surface elevation &
instant high-frequency interactions).

## When to use

- Analytics dashboards, observability/monitoring views, billing/finance consoles
- Admin panels, CRUD back-offices, CRM/support queues, inventory tables
- Any screen combining KPIs, charts, filter bars, and data tables

## 1. Domain grounding & the 5-second hierarchy

Before laying out a dashboard, state three things: **who** the operator is, the
**primary verb** (triage, reconcile, monitor, approve, debug), and the **target
density** (`compact`, `default`, or `comfortable`).

Organize the viewport top-to-bottom, left-to-right by question priority:

1. **Status & anomalies (top):** Is anything broken or needing action right now?
2. **Summary KPIs (row 1, max 3–4 cards):** What are the headline numbers and
   how did they change vs. the comparison period?
3. **Trends & distribution (row 2, 1–2 charts):** Why did the numbers move?
4. **Actionable records (row 3+, table/list):** Which specific items need
   inspection or action?

**Squint test:** Blur your eyes — the primary metric or anomaly must stand out
first, section structure second, and chrome/gridlines last.

**Anti-patterns:**

- Never open a dashboard with a welcome banner (`"Good morning, Alex! 👋"`).
- Never place more than **4 KPI cards** in a single row.
- Never show a raw number without **context** (delta `%`, comparison period,
  target, or sparkline).
- Never animate dashboard widgets on page load — data should render immediately.

## 2. Operate-mode typography & 4-tier hierarchy

Operational UIs rely on **weight and color/opacity** more than font size jumps:

- **Font families:** One crisp sans-serif for UI + one monospace face for IDs,
  hashes, shortcuts, and timestamps. **Never use serif display fonts** on
  operational dashboards.
- **Fixed `rem` scale (not fluid `clamp()`):** Keep UI text at fixed `rem` steps
  (ratio `1.125–1.25`) so resizing the browser shows *more data columns*, not
  bigger labels.
- **Tabular numerals everywhere:** Enable `font-variant-numeric: tabular-nums`
  (`tabular-nums`) on every KPI value, table column, delta badge, axis label,
  timer, and pagination counter.
- **4-tier text hierarchy at a single size (`14px`):**
  1. **Primary (`--foreground`, weight `500–600`):** Record names, KPI values,
     active labels.
  2. **Secondary (`--foreground-secondary`, weight `400–500`):** Supporting cell
     values, descriptions.
  3. **Tertiary (`--muted-foreground`, weight `400`):** Column headers, field
     labels, timestamps, units.
  4. **Muted / disabled:** Placeholders, empty-cell dashes (`—`), shortcut hints.

## 3. Subtle surface layering & density

- **Whisper-quiet elevation:** Separate regions with `1px` hairline borders and
  subtle surface steps rather than heavy drop shadows. In dark mode, elevate via
  lightness steps (`canvas → +7% card → +9% sticky bar → +12% popover/modal`) +
  `1px` border (`rgba(255, 255, 255, 0.06–0.12)`).
- **Unified sidebar & canvas:** Keep the sidebar on the **same background token
  as the canvas** separated by a `1px` border — two competing dark/light
  backgrounds fracture the workbench.
- **Inset controls:** Form inputs, search boxes, and filter pills sit slightly
  *darker/recessed* than the card surface in dark mode (and crisp white with a
  ≥ 3:1 border in light mode).
- **Avoid card-in-card bloat:** At medium/high density, do not wrap every metric
  or sub-section in its own bordered card. Use a single outer surface with
  `divide-y` / `divide-x` hairline dividers or whitespace (`gap` between groups
  ≥ `2×` inner gap).

| Density tier    | Row height | Cell padding | Font size | Best for                                 |
| --------------- | ---------- | ------------ | --------- | ---------------------------------------- |
| **Compact**     | `32–36px`  | `6px 10px`   | `12–13px` | Logs, trading/ops, power-user queues     |
| **Default**     | `40–44px`  | `8px 12px`   | `13–14px` | Standard SaaS tables, CRM, billing       |
| **Comfortable** | `48–56px`  | `12px 16px`  | `14px`    | Media rows with avatars, two-line cells  |

## 4. KPI cards that carry meaning

Every KPI card needs at most **3 data points**:

1. **Label** (`12–13px`, medium, `text-muted-foreground`, top-left, no ALL CAPS)
2. **Primary value** (`24–32px`, semibold, `tabular-nums`, tight tracking `-0.02em`)
3. **Comparison / delta** (`+12.4% vs last 30d`) with semantic direction + icon
   (`↑` / `↓`) — never rely on red/green color alone (add arrow + `+/-` sign;
   remember higher error/latency is **bad**, not green).

## 5. Data tables (the workhorse)

- **Alignment rules (non-negotiable):**
  - **Left-align** text, names, emails, statuses, and tags.
  - **Right-align** numbers, currency, percentages, durations, and counts (with
    `tabular-nums`), matching the column header alignment to the cells.
  - **Center-align** only single-character icons or checkboxes.
- **Column discipline:** Keep visible columns to **5–8** at default width. Push
  secondary metadata into a row-click **detail `Sheet` / drawer** or a column
  visibility toggle.
- **Truncation & layout resilience:** Set `min-w-0` on flex children and use
  `truncate` with a way to inspect the full value (tooltip or detail drawer).
  Never let a long ID or email blow out table width.
- **Sticky chrome:** Keep the table header (`thead`) and bulk-action bar sticky
  during vertical scroll; keep the primary identifier column sticky during
  horizontal scroll on wide tables.
- **Row actions:** Put 1–2 primary actions or a `⋯` (`DropdownMenu`) in the
  rightmost column, always reachable via keyboard focus (never `display: none`
  until hover).
- **Bulk selection:** Header checkbox supports checked, unchecked, and
  `indeterminate` states; selecting rows reveals a contextual action bar (`"3
  selected · Export · Archive · Delete"`).

## 6. Filter bars, time-range & URL state

- **Single horizontal toolbar above the data:**
  `[Search input (⌘K / /)] [Status facet] [Owner facet] [Date range picker] ··· [Saved views] [Density] [Export] [Primary CTA]`
- **URL-synced state:** Persist active filters, sort column/direction, page
  size, and date range in URL search params so every filtered view is shareable,
  bookmarkable, and survives refresh.
- **Active filter visibility:** Show applied filter count and a one-click
  `"Reset filters"` action whenever any filter is non-default.

## 7. Charts & visualizations

- **Pick the simplest chart that answers the question:**
  - Trend over time → **Line** or **subtle area** (`0.08–0.15` fill opacity)
  - Ranked comparison → **Horizontal bar** (labels never rotate diagonally)
  - Part-to-whole (≤ 5 slices) → **Stacked bar** (avoid pie charts with > 3 slices)
- **Restrained chrome:** Dashed or `4–6%` opacity horizontal gridlines only;
  remove vertical gridlines and chart borders; format axes compactly (`$1.2M`,
  `45ms`, `99.9%`); limit a single chart to **≤ 4 series** using your `--chart-1`
  .. `--chart-5` tokens.
- **Crosshair tooltip:** Show exact timestamp + all series values in
  `tabular-nums`, sorted by value.

## 8. Loading, empty, error & freshness states

Every widget and table handles 4 states independently (one failing widget must
never crash the whole page):

1. **Loading:** Structural skeletons matching exact card/table heights (zero
   CLS); delay spinners by `150ms`.
2. **Zero-data empty (first use):** Explain what populates this view + primary
   CTA (`"Connect data source"`).
3. **Zero-results empty (filtered):** Echo the active filter (`"No invoices
   match 'status:overdue'"`) + `"Reset filters"` button.
4. **Error & staleness:** Inline retry on the failed card/table + timestamp or
   `"Updated 12s ago · Refresh"` indicator for live views.

## Checklist before "done"

- 1 sans + 1 mono font on a fixed `rem` scale; `tabular-nums` on every numeric
  element; numbers right-aligned, text left-aligned.
- Sidebar shares canvas background; surfaces layer via `1px` borders and subtle
  lightness steps without card-in-card nesting.
- Filters, sort, pagination, and date range persist in URL search params.
- Every KPI card has a comparison/delta; every table and chart handles loading,
  empty, filtered-empty, and error states independently.

## Attribution

Synthesized from [`Dammyjay93/interface-design`](https://github.com/Dammyjay93/interface-design)
(MIT) and [`pbakaus/impeccable`](https://github.com/pbakaus/impeccable)
(`operate.md`, Apache-2.0).
