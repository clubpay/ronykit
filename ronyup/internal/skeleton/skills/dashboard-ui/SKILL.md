---
name: dashboard-ui
description: >-
  Design and build data-dense dashboards, back-office, and admin apps that
  surface the few metrics a decision needs. Use when laying out an admin/console
  screen, KPI strip, chart grid, data table, sidebar nav, or reviewing the
  visual design and information hierarchy of an internal/operational UI.
---

# Dashboard & Admin UI

A dashboard optimizes for **repeated decision-making under time pressure**, not
persuasion. The user returns daily to extract a number, compare it to a target,
and act in seconds. That means less hero typography, more data density, a
stricter color system, and zero decorative motion.

The job: surface the smallest set of metrics needed to make a decision, in the
order the eye scans them, with one accent color and no decorative noise.

## When to use

- Building or reviewing an admin panel, back-office console, analytics overview,
  ops dashboard, or settings/management screen.
- Laying out a screen with KPI cards, charts, data tables, or a sidebar nav.
- Choosing a chart type, color usage, or where an element belongs on the page.
- Designing empty/loading/error states for an operational UI.

Combine with the `shadcn` and `nextjs-modern` skills for implementation — this
skill governs layout and information hierarchy; those govern the components and
data wiring.

## 1. One job per dashboard

Before laying anything out, write one sentence: **who** looks at this, and
**what decision** do they make from it? ("The on-call engineer opens this during
an incident to find which service is degraded.")

- "An overview of the business" / "all the data in one place" is not a job —
  it's a database with bad lighting.
- Three audiences = three dashboards. Don't serve every stakeholder from one
  screen; make focused screens and link between them.

## 2. Lead with 3–5 hero metrics

Pick the smallest set of numbers that, glanced at together, tell you whether the
job is going well today. These become the top KPI strip.

- Each KPI card shows three things: the **current value** (large number), a
  **delta vs. the previous comparable period** (small % with a directional
  arrow), and an inline **sparkline / trend**.
- Use absolute numbers, not normalized ratios, for the hero metric when stakes
  are real (`$1,248,000` reads faster than "104% of target").
- Make the delta sign explicit — a green up-arrow next to "+12.5% vs last month"
  beats a bare "12.5%".
- A KPI card without a sparkline is just a number; the trend earns its 60px.
- Twelve cards is a wall, not a strip — walls don't get scanned. Cut to 4–6.

## 3. Lay it out on the F-pattern

Users scan in an F-shape: a horizontal sweep across the top, a shorter sweep
lower, then a vertical drift down the left edge. Place by importance:

1. Most important metric **top-left** — where the eye lands first.
2. Secondary metrics extend **right** along the KPI strip.
3. The chart that explains the hero metric sits **directly below**.
4. Detail (tables, drilldowns) flows down from there.
5. Filters and time-range pickers go **top-right** — reachable, but not
   competing with metrics for attention.

Keep the primary view to **7–8 elements max**. Push secondary detail behind
tabs, drilldowns, or a "View all" link (progressive disclosure).

## 4. 12-column grid, card-based content

| Region       | Width                  | Purpose                                   |
| ------------ | ---------------------- | ----------------------------------------- |
| Left sidebar | 240–280px              | Nav, workspace switcher, account menu     |
| Top bar      | Full width, 56–64px    | Page title, search, date range, notifs    |
| Content area | 12-column CSS grid     | KPI cards, charts, tables, detail panels  |

- KPI strip of 4 cards → each spans **3 columns**. Paired charts → **6 + 6** or
  **8 + 4** (wide chart + narrow context panel). Full-width table → **12**.
- Every content element lives in a **card**: consistent radius (8–12px),
  consistent padding (24px default), consistent border.
- **Prefer a 1px border at ~8% opacity over a drop shadow** — it reads cleaner
  and ages better (Linear/Vercel ship borders, not shadows).
- **Whitespace (16–24px) does the grouping** — no divider lines, color blocks,
  or section headings needed when spacing is right.
- A **persistent sidebar on desktop**, never a hamburger menu — it's navigation,
  not wasted space.

## 5. Color discipline — one accent, the rest for status

The fastest way to look professional is to use almost no color.

- **Two surface colors** — off-white/white cards in light mode; near-black in
  dark (`#0a0a0a`/`#111`, never pure `#000`).
- **Two text colors** — near-black primary, 60–70% gray secondary.
- **One accent** — interactive elements, links, the active sidebar item, the
  hero chart line. That's it.
- **Three semantic status colors** — green (on track / up), red (at risk /
  down), amber (caution). **Reserved**: they appear nowhere else.

Heuristic: if you removed every color except one accent + three status colors,
does the dashboard still work? If yes, your discipline is right. The moment red
becomes a decorative icon background, you've taught the user that red means
nothing — and broken the most useful signal a dashboard has.

> Never use colored card backgrounds. Use neutral cards with a colored accent on
> the metric or a status pill.

Use semantic tokens (`bg-background`, `text-foreground`, `border`, `ring`) per
the `shadcn` skill — don't hard-code hex.

## 6. Charts follow data shape

Pick the chart from the data, not because it looks interesting.

| Data shape                       | Use                                   | Don't use                  |
| -------------------------------- | ------------------------------------- | -------------------------- |
| Trend over time                  | Line, area, sparkline                 | Bar-per-day                |
| Categorical compare (≤7)         | Horizontal bar                        | Pie, 3D anything           |
| Categorical compare (>7)         | Sorted table, treemap                 | Bar chart with 25 bars     |
| Composition (parts of a whole)   | Stacked bar / single-row stacked bar  | Pie with >3 slices         |
| High-density patterns            | Heatmap                               | Line chart with 50 series  |
| Detail to sort or scan           | Table with sortable columns           | Any chart                  |
| Relationship between 2 variables | Scatter plot                          | "Balanced scorecard"       |

- **Tables are charts.** First-class: sortable columns, sticky headers, status
  pills, pagination. Most "this feels janky" complaints trace to a bad table.
- **Don't use color as a category.** Distinguish series by line style, weight,
  or order — five-line charts stop being readable around the third line.
- React charting: reach for Recharts, Tremor, or Apache ECharts.

## Mobile dashboards — the rules change

A desktop dashboard squeezed onto a 430px phone is worse than one designed for
mobile. When targeting mobile:

- **One dominant KPI** at the top, not four.
- Secondary metrics in a **2×2 grid** (two columns max).
- **Bottom tab bar** (≤5 tabs) instead of a sidebar — thumb-reachable.
- **Cut the table** → a list of the top 3–5 items + "View all".
- Simpler charts: drop axis labels/legends where possible, never >2 series.

## Common mistakes to avoid

1. The 14-card overview — cut to 4–6; the rest belong on sub-pages.
2. Colored card backgrounds — neutral cards + accent/status pill instead.
3. Pie charts with 7 slices — use a sorted bar chart.
4. Hamburger menu on desktop — keep the persistent sidebar.
5. Decorative motion — ticking counters, self-drawing charts, hover wiggles.
   None help a decision; cut them.
6. Design system defined in two places — one source of truth for tokens.
7. **No empty states.** What does it look like with no data / for a new user?
   Empty states are ~80% of perceived polish — design them, plus loading and
   error states, for every data surface.

## Checklist before "done"

- One stated job; primary view ≤ 7–8 elements; 3–5 hero metrics top-left.
- KPI cards show value + signed delta + trend; layout follows the F-pattern.
- 12-column grid; consistent cards (radius/padding/border, not shadows);
  spacing does the grouping.
- One accent + 3 reserved status colors only; no decorative/background color;
  semantic tokens, no stray hex.
- Chart type matches data shape; tables are sortable/sticky/paginated.
- Empty, loading, and error states designed for every data surface.
