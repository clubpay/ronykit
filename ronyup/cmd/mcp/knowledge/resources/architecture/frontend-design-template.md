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

| Field     | Decision                                      |
|-----------|-----------------------------------------------|
| Subject   | <!-- what it is -->                           |
| Audience  | <!-- who uses it -->                          |
| Single job| <!-- one sentence: what this UI must achieve --> |
| Topology  | <!-- single app at frontend/ or frontend/<app>/ --> |

## 2. Aesthetic direction

<!-- One paragraph: mood, references, what to avoid (e.g. no Inter + purple gradient) -->

**Signature element:** <!-- the one memorable visual/device for this product -->

## 3. Token plan

### Color (primitives)

| Token            | Hex       | Role              |
|------------------|-----------|-------------------|
| surface          | #         | page background   |
| surface-elevated | #         | cards, panels     |
| text             | #         | primary copy      |
| text-muted       | #         | secondary copy    |
| accent           | #         | primary CTA/links |
| destructive      | #         | errors/danger     |

### Semantic aliases

```css
:root {
  --color-background: var(--color-surface);
  --color-foreground: var(--color-text);
  --color-primary: var(--color-accent);
  /* dark mode: override semantics only, in .dark { … } */
}
```

### Typography

| Role    | Family | Weight | Size scale notes |
|---------|--------|--------|------------------|
| Display |        |        |                  |
| Body    |        |        |                  |
| Mono    |        |        | optional         |

### Spacing & radius

- Base grid: 4px
- Section padding:
- Card radius:
- Button radius:

## 4. Layout

<!-- One-sentence layout concept + ASCII wireframe for main shell -->

- Max content width:
- Navigation: <!-- top / side / hybrid -->
- Mobile: <!-- how nav and hero adapt -->

## 5. Component rules

- UI kit: <!-- shadcn/ui + Tailwind -->
- Buttons: <!-- primary / secondary / ghost usage -->
- Forms: <!-- label placement, error display -->
- Tables/cards: <!-- dashboard vs marketing -->
- Empty & error states: <!-- voice and layout -->

## 6. Motion

- Load / enter:
- Hover / focus:
- Reduced motion: respect `prefers-reduced-motion`; disable decorative motion

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
