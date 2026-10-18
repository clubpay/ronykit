# Frontend design document template

Write to `docs/design/<app>-frontend-design.md`. Replace placeholders; delete instructional comments before sharing with the user.

```markdown
---
status: draft
app: <app-slug>
stack: <e.g. Next.js 16 App Router + shadcn/ui + Tailwind>
updated: <YYYY-MM-DD>
---

# <Product / app name> — frontend design

## 1. Context

| Field        | Decision                                      |
|--------------|-----------------------------------------------|
| Subject      | <!-- what it is -->                           |
| Audience     | <!-- who uses it -->                          |
| Single job   | <!-- one sentence: what this UI must achieve --> |
| Surface mode | <!-- Persuade | Operate | Read | Experience --> |
| Topology     | <!-- single app at frontend/ or frontend/<app>/ --> |

## 2. Aesthetic direction

<!-- One paragraph: mood, references, what to avoid (e.g. no Inter + purple gradient, no 3-equal-icon-card grids, no hanging section headers, no fake browser chrome) -->

**Signature element:** <!-- the one memorable visual/device for this product -->

## 3. Token plan

### Color (OKLCH / primitives & roles)

| Token                  | Light     | Dark      | Role                                         |
|------------------------|-----------|-----------|----------------------------------------------|
| `--background`         |           |           | page canvas (60%)                            |
| `--surface-sunken`     |           |           | recessed wells, table headers, inset inputs  |
| `--card` / `--popover` |           |           | elevated surfaces (lightness steps in dark)  |
| `--foreground`         |           |           | Tier 1: headings & primary values            |
| `--foreground-secondary`|          |           | Tier 2: body copy & table cells              |
| `--muted-foreground`   |           |           | Tier 3: labels, column headers, timestamps   |
| `--foreground-subtle`  |           |           | Tier 4: placeholders, disabled, empty dashes |
| `--primary`            |           |           | primary CTA/links (10%)                      |
| `--input-bg` / `--input-border` |  |           | dedicated control tokens (≥3:1 border)       |
| `--border` / `--ring`  |           |           | hairline dividers (`1px`) & instant focus    |
| `--destructive` / status|          |           | ≥15° from primary; paired `-subtle`/`-fg`/`-border` |

### Semantic aliases

```css
:root {
  color-scheme: light;
  --color-background: var(--background);
  --color-foreground: var(--foreground);
  --color-primary: var(--primary);
  /* dark mode: override semantics only, in .dark { color-scheme: dark; … } */
}
```

### Typography

| Role    | Family | Weight | Size scale notes |
|---------|--------|--------|------------------|
| Display |        |        | omit on `Operate` dashboards; fluid `clamp()` on `Persuade` |
| Body    |        |        | `16px` (`1rem`) prose, `14px` UI, `≥16px` mobile inputs |
| Mono    |        |        | `tabular-nums` for numbers, metrics, IDs, timestamps |

### Spacing, concentric radius & elevation

- Base grid: 4px; group with space first (gap between groups ≥ `2×` inner gap)
- Section padding:
- Card & inner radius: concentric (`outerRadius = innerRadius + padding`)
- Elevation: layered transparent shadows in light mode; surface lightness steps (`+7% / +9% / +12%`) + `1px` translucent borders in dark mode

## 4. Layout

<!-- One-sentence layout concept + ASCII wireframe for main shell -->

- Max content width:
- Navigation: <!-- top / side / hybrid; sidebar shares canvas background on Operate UIs -->
- Mobile: <!-- how nav and hero adapt; 100dvh, ≥44x44px hit targets -->

## 5. Component rules

- UI kit: <!-- shadcn/ui + Tailwind; single SVG icon library (currentColor, no emoji icons) -->
- Buttons: <!-- verb + object labels, nowrap, 2px optical icon-side padding, enabled:active:scale-[0.97] -->
- Forms: <!-- visible labels, validate on blur/submit, aria-describedby errors, never block paste -->
- Tables/cards: <!-- right-aligned tabular-nums numbers, 5-8 columns max, no card-in-card nesting -->
- Empty & error states: <!-- first-use empty vs filtered empty with Reset filters; Undo toast for reversible actions -->

## 6. Motion

- Frequency budget: <!-- 0ms for 100+/day actions (⌘K, row hover, tab switch); 125-350ms for overlays -->
- Load / enter: <!-- --ease-out: cubic-bezier(0.23, 1, 0.32, 1) from scale(0.95) + trigger transform-origin; never ease-in or scale(0) -->
- Hover / focus: <!-- hover gated on @media (hover: hover); focus-visible ring instant 0ms -->
- Reduced motion: respect `prefers-reduced-motion` (collapse spatial motion to `≤150ms` opacity cross-fade); suppress transitions on theme switch (`disableTransitionOnChange`)

## 7. v1 surfaces

| Route / screen | Purpose | Notes |
|--------------|---------|-------|
|              |         |       |

## 8. WebMCP tools (agent-friendly surface)

<!-- Skip this section only when the app explicitly does not need in-browser agent automation. -->

**Agent goal:** <!-- one sentence: what agents should accomplish on this app -->

| Tool | R/W | Description | Maps to |
|------|-----|-------------|---------|
| <!-- e.g. orders_filter --> | read / write | <!-- plain-language: inputs, side effects, return shape --> | <!-- hook, Server Action, or shared function --> |

Rules for the tool catalog:

- Name tools after **journeys** (`dashboard_set_date_range`), not DOM (`click_save`).
- W3C name constraints: 1–128 chars, `[A-Za-z0-9_.-]` only, `snake_case`.
- **Read** tools: `readOnlyHint: true` — list, search, get KPIs.
- **Write** tools: same auth/validation as the human UI; state description mentions side effects.
- Handlers call the **same logic** as buttons/forms — no duplicated business rules.
- After agent mutations, the visible UI must update the same way as human clicks.

Implementation: read skill `webmcp` (`.agents/skills/webmcp/SKILL.md`). Default stack:
`@mcp-b/global`, `@mcp-b/react-webmcp`, `zod`. Reference example:
`example/ex-13-webmcp/` in the RonyKit monorepo.

## 9. Quality floor (non-negotiable)

- Responsive ≥360px; no horizontal overflow
- Visible keyboard focus; semantic HTML; labelled controls
- Semantic design tokens in components (no scattered raw hex)
- Reusable components ship co-located Storybook stories
- `pnpm lint`, `pnpm typecheck`, `pnpm build` pass via `frontend/verify.sh`
- WebMCP tools (when listed in §8) tested via shared handler unit tests; optional
  `modelContextTesting` smoke in dev (see `webmcp` skill)

## 10. Revision history

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 0.1     |      |        | Initial draft |
```
