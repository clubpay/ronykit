Frontend design is **document-first**, like backend SRS and SDD. Do not initialize a frontend stack or write UI code until the user approves a frontend design document.

## Paths

| Document | File pattern |
|----------|--------------|
| Frontend design | `docs/design/<app>-frontend-design.md` |

`<app>` is the frontend app slug:

- For one app at `frontend/`, use `web`. The path is `docs/design/web-frontend-design.md`.
- For multiple apps, use the directory name. For `frontend/admin/`, the path is `docs/design/admin-frontend-design.md`.

## Approval gate

Each document starts with YAML frontmatter `status: draft`. Only the **user** sets `status: approved`. Never approve on the user's behalf.

Do not run `create-next-app`, `create vite`, or scaffold components until the design document has `status: approved`.

Run MCP tool `check_frontend_design` before stack initialization. It requires the matching `*-frontend-design.md` with `status: approved` and all required sections.

After a `package.json` exists under `frontend/`, `frontend/verify.sh` enforces the same requirements.

## Bootstrap workflow

Complete this workflow before any UI code:

1. **Clarify topology.** Ask whether there is one app or multiple apps. Confirm the app slug and stack, such as Next.js or Vite.
2. **Ask design questions.** Do not guess these answers:
   - Subject, audience, and the page or product's single job
   - Aesthetic direction, such as minimal, editorial, brutalist, playful, or enterprise
   - Existing brand palette, logo, and fonts, or freedom to propose them
   - Light mode, dark mode, or both
   - Marketing surface or dashboard and data-dense surface
   - Required v1 surfaces, including landing, authentication, dashboard, and settings
3. **Read guidance.** Read `knowledge://ronyup/architecture/frontend-design-template`. Read `.agents/skills/frontend-design/SKILL.md`, `design-tokens/SKILL.md`, and `typography/SKILL.md` under `.agents/skills/`.
4. **Write the document.** Create `docs/design/<app>-frontend-design.md` with `status: draft`. Include the token plan and design-system rules from the template.
5. **Run EGM design review.** Read `knowledge://ronyup/architecture/egm-design-review` and `.agents/skills/ronykit-framework/references/egm.md`. Run zoom out, comprehension, design critique, and intermediate zoom out with fresh read-only Goldfish subagents. Write findings under `docs/design/reviews/<app>-frontend-design/`. Resolve them with the user. If fresh subagents are unavailable, mark the review `not run`. Ask whether to continue with human-only review. Never simulate Goldfish in the main context.
6. **Present a summary and wait.** Get user approval and `status: approved` before initializing the frontend or writing UI.

## Required design-system rules

The approved document is the source of truth for all UI work on that app. It must include:

- **Alternatives and risks:** Viable rejected directions with concrete reasons. Include usability, accessibility, responsive, performance, and maintenance risks with mitigations.
- **Token plan:** Four to six named hex colors, display and body type roles, spacing and radius scales, and one signature element.
- **Semantic tokens:** Map primitives to roles such as `--color-primary` and `--color-background`. Components must not use raw hex values.
- **Typography:** Font families with `next/font` or equivalent, scale, and weights.
- **Layout rules:** Grid, maximum width, section rhythm, and navigation pattern.
- **Component rules:** shadcn and Tailwind usage, button, card, and input variants, and status colors.
- **Motion:** Animated and static elements with the `prefers-reduced-motion` policy.
- **WebMCP tools:** Required for agent-friendly apps. Add a catalog with name, read/write behavior, description, and mapping to shared app logic. See template section 8 and skill `webmcp`.
- **Quality floor:** Responsive behavior at 360px or wider, keyboard focus, semantic HTML, and Storybook for reusable components.

After approval, add tokens to the app before building pages. Use `globals.css`, the Tailwind theme, and shadcn variables as applicable.

## MCP prompt

Use the `design-frontend` prompt for the full bootstrap workflow.

## Related skills

Read each applicable `SKILL.md` before coding:

| Requirement | Skills |
|-------------|--------|
| Bootstrap and aesthetic | `frontend-design`, `design-tokens`, `typography` |
| Surface, icon, and motion quality | `ui-polish`, `ux-quality` |
| Agent-friendly and WebMCP | `webmcp`, `nextjs-modern`, `ux-quality` |
| Dashboard and admin in Operate mode | `dashboard-ui`, `shadcn`, `ux-quality`, `ui-polish` |
| Implementation | `nextjs-modern`, `shadcn`, `react-performance` |
| Stories and tests | `storybook`, `frontend-testing` |
