Frontend design is **document-first**, like backend SRS/SDD. Do not initialize a frontend stack or write UI code until the user has
approved a frontend design document.

## Paths

| Document            | File pattern                          |
|---------------------|---------------------------------------|
| Frontend design     | `docs/design/<app>-frontend-design.md` |

`<app>` is the frontend app slug:

- Single app at `frontend/` → use `web` → `docs/design/web-frontend-design.md`
- Multiple apps → directory name, e.g. `frontend/admin/` → `docs/design/admin-frontend-design.md`

## Approval gate

Each document starts with YAML frontmatter `status: draft`. Only the **user** sets `status: approved`. Never approve on the user's behalf.

**Do not** run `create-next-app`, `create vite`, or scaffold components until the design doc is `status: approved`.

The frontend verify gate (`frontend/verify.sh`) enforces this: once a `package.json` exists under `frontend/`, the matching
`*-frontend-design.md` must exist with `status: approved`.

## Bootstrap workflow (before any UI code)

1. **Clarify topology** — one app or multiple? Which app slug and stack (Next.js, Vite, …)?
2. **Ask design questions** (do not guess):
   - Subject, audience, and the page/product's single job
   - Aesthetic direction (refer to examples: minimal, editorial, brutalist, playful, enterprise, …)
   - Brand constraints (existing palette/logo/fonts?) or freedom to propose
   - Light/dark/both; marketing vs dashboard/data-dense
   - Must-have surfaces for v1 (landing, auth, dashboard, settings, …)
3. **Read** `knowledge://ronyup/architecture/frontend-design-template` and invoke skills `frontend-design`, `design-tokens`, and `typography`
   (read `.agents/skills/<id>/SKILL.md`).
4. **Write** `docs/design/<app>-frontend-design.md` with `status: draft`, including the token plan and design-system rules (see template).
5. **Present a summary and wait** for user approval (`status: approved`) before initializing the frontend or writing UI.

## Design-system rules in the document

The approved doc is the source of truth for all UI work on that app. It must include:

- **Token plan** — named colors (4–6 hex), type roles (display + body), spacing/radius scale, one signature element
- **Semantic tokens** — map primitives → roles (`--color-primary`, `--color-background`, …); no raw hex in components
- **Typography** — font families (via `next/font` or equivalent), scale, weights
- **Layout rules** — grid/max-width, section rhythm, navigation pattern
- **Component rules** — shadcn/Tailwind usage, button/card/input variants, status colors
- **Motion** — what animates, what stays static; `prefers-reduced-motion` policy
- **WebMCP tools** (when the app should be agent-friendly) — tool catalog table: name,
  read/write, description, mapping to shared app logic; see template §8 and skill `webmcp`
- **Quality floor** — responsive ≥360px, keyboard focus, semantic HTML, Storybook for reusable components

After approval, wire tokens into the app (`globals.css`, Tailwind theme, shadcn variables) before building pages.

## MCP prompt

Use the `design-frontend` prompt for the full bootstrap workflow.

## Related skills (read SKILL.md before coding)

| Need                         | Skills                                              |
|------------------------------|-----------------------------------------------------|
| Bootstrap / aesthetic        | `frontend-design`, `design-tokens`, `typography`    |
| Agent-friendly / WebMCP      | `webmcp`, `nextjs-modern`, `ux-quality`             |
| Dashboard / admin            | `dashboard-ui`, `shadcn`, `ux-quality`            |
| Implementation               | `nextjs-modern`, `shadcn`, `react-performance`    |
| Stories & tests              | `storybook`, `frontend-testing`                     |
