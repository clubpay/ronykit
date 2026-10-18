---
name: review-frontend
description: >-
  Review an existing frontend app against its approved frontend design document,
  UI polish and UX quality rules, Storybook and accessibility coverage, and
  frontend/verify.sh gate.
arguments:
- name: app_path
  description: Path to the frontend app directory (e.g. frontend or frontend/admin).
  required: true
- name: app_name
  description: Frontend app slug (default "web" for a single app at frontend/).
  required: false
---

Review the frontend application at `{{app_path}}`{{#if app_name}} (app slug: `{{app_name}}`){{/if}} for design-system, UI polish, UX quality, and verification compliance.

Read these skills before auditing:

- `.agents/skills/frontend-design/SKILL.md`
- `.agents/skills/design-tokens/SKILL.md`
- `.agents/skills/typography/SKILL.md`
- `.agents/skills/ui-polish/SKILL.md`
- `.agents/skills/ux-quality/SKILL.md`
- `.agents/skills/storybook/SKILL.md`
- `.agents/skills/frontend-testing/SKILL.md`

Check the following in order:

1. **Design gate (`check_frontend_design`):** Run the MCP tool `check_frontend_design` for the app. Confirm `docs/design/<app>-frontend-design.md` exists, is `status: approved`, has all required sections, and contains no leftover `<!-- ... -->` placeholder comments.
2. **Token & theme fidelity (`design-tokens`, `shadcn`):**
   - `globals.css` defines the 4-tier text hierarchy (`--foreground`, `--foreground-secondary`, `--muted-foreground`, `--foreground-subtle`), `--surface-sunken`, and dedicated control tokens (`--input-bg`, `--input-border`).
   - Dark mode elevates via surface lightness steps (`+7% / +9% / +12%` + `1px` translucent borders) rather than heavy drop shadows.
   - Components consume semantic tokens (`bg-background`, `text-foreground`, `text-muted-foreground`) with zero scattered raw hex/RGB values.
3. **Typography & numeric discipline (`typography`):**
   - Fonts are loaded via `next/font` (or equivalent) with CSS variables (`--font-sans`, `--font-display`, `--font-mono`).
   - Numeric columns, KPI cards, prices, timers, and counts apply `tabular-nums` and right-align in tables.
   - Prose measure is capped (`60–75ch`), and headings use `text-wrap: balance`.
4. **Surface, icon & motion craft (`ui-polish`):**
   - Concentric border radii (`outerRadius = innerRadius + padding`) on nested cards/wells; no card-in-card double borders.
   - Single SVG icon set (`currentColor`, `aria-hidden="true"` on decorative icons, `aria-label` + `≥44×44px` hit target on icon-only buttons, no emoji icons).
   - No `transition-all` / `transition: all`, no `100vh` / `h-screen` / `min-h-screen` (use `dvh`), no `w-screen`, and no `scale(0)` entrances (use `scale(0.95)` + custom cubic-bezier easing).
   - High-frequency actions (`⌘K`, tab/row switching) have `0ms` instant feedback; `prefers-reduced-motion` is respected.
5. **5 view states & form ergonomics (`ux-quality`, `nextjs-modern`):**
   - Every data-driven screen implements all 5 states: **Loading** (layout-matched skeletons), **Empty** (first-use vs filtered-zero with *Reset filters*), **Error** (actionable retry), **Partial/Truncated**, and **Populated**.
   - Forms have visible `<Label>` elements, validate on blur/submit (clear immediately on change), wire inline errors via `aria-invalid` + `aria-describedby`, and map RonyKit `<Service>StubError` (`err.item`, `err.serverMessage`) to field errors.
6. **Storybook, tests & verify gate (`storybook`, `frontend-testing`):**
   - Every reusable UI component has a co-located `*.stories.tsx` importing `globals.css` and covering interactive states (`Default`, `Disabled`, `Loading`, `Error`, `DarkSurface`, `LongContentStress`).
   - Run `bash frontend/verify.sh` (or `make -C frontend verify`) and report any failures with concrete file:line fixes.
