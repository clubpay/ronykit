---
name: frontend-design
description: >-
  Design distinctive, production-grade frontend interfaces that don't read as
  templated AI defaults. Use when bootstrapping a frontend app, building or
  restyling a landing page, marketing site, hero, brand-forward product surface,
  or any UI where visual identity matters — choosing palette, typography,
  layout, motion, and copy. Required before running create-next-app or writing UI.
---

# Frontend Design

## Bootstrap gate (read this first — enforced)

**Do not** run `create-next-app`, `create vite`, or write UI until:

1. You have **asked** the user design questions (aesthetic, audience, light/dark, brand constraints, v1 screens).
2. You have written `docs/design/<app>-frontend-design.md` with a token plan and design-system rules (MCP `design-frontend`, template
   `architecture/frontend-design-template`).
3. The user has set `status: approved` in the doc frontmatter.

Also read `design-tokens` and `typography` skills before implementing. `frontend/verify.sh` enforces the approved design doc.

Approach this as the design lead at a small studio: every client gets a visual
identity that could not be mistaken for anyone else's. Make deliberate,
opinionated choices about palette, type, and layout that are specific to *this*
brief, and take one real aesthetic risk you can justify.

The failure mode to avoid is **AI slop** — the statistical center of every UI
the model has seen (Inter, a purple gradient, evenly-spaced rounded cards). It
looks generated, not designed. This skill pulls you off that center.

## When to use

- Building or restyling a landing page, marketing site, hero, pricing page,
  docs home, or brand-forward product surface.
- Choosing aesthetic direction, palette, typography, layout, or motion.
- Reviewing UI that feels generic, templated, or "AI-generated".

For **data-dense internal/operational UI** (admin panels, KPI strips, data
tables, ops consoles) use `dashboard-ui` instead — that optimizes for scanning
and decisions, this optimizes for identity. Pair both with `shadcn` and
`nextjs-modern` for implementation; this skill governs the *look*, those govern
the *components and wiring*.

## 1. Ground it in the subject

Before designing, name the **subject** (what it is), the **audience**, and the
page's **single job** — and state your choice if the brief left it open.

- Distinctive choices come from the subject's own world — its materials,
  instruments, artifacts, and vernacular. A vinyl marketplace and a tax tool
  should not share a mood board.
- Use any known context about the user's product or prior designs as a hint.
- Build with the brief's real content throughout — not lorem ipsum that you
  swap later. Placeholder content hides the design's real problems.

## 2. Two-pass process — plan, critique, then build

Do the planning in your thinking; only surface ideas once you're confident
they'll land. **Never start coding before the plan passes the critique.**

**Pass 1 — a compact token plan:**

| Token     | What to decide                                                       |
| --------- | ------------------------------------------------------------------- |
| Color     | 4–6 named hex values (surfaces, text, one accent, status if needed) |
| Type      | 2+ roles: a characterful display face + complementary body (+ data) |
| Layout    | One-sentence concept + a quick ASCII wireframe to compare options   |
| Signature | The one element this page is remembered by, embodying the brief     |

**Pass 2 — critique against the brief:** for each token, ask *"would I produce
this for almost any similar prompt?"* If yes, it's a default, not a choice —
revise it and say what changed and why. Only then write code, deriving every
color and type value from the revised plan.

## 3. Calibrate away from the known defaults

Current AI design clusters around three looks. They're legitimate *only* when
the brief actually asks for them — otherwise they're tells:

1. Warm cream background (~`#F4F1EA`) + high-contrast serif + terracotta accent.
2. Near-black background + one acid-green or vermilion accent.
3. Broadsheet layout: hairline rules, zero radius, dense newspaper columns.

If the brief pins a direction, follow it exactly — the brief's words always win.
Where it leaves an axis free, don't spend that freedom on one of these.

## 4. Typography carries the personality

- Pair display and body deliberately — not the same families you'd reach for on
  any project. Set an intentional scale with chosen weights, widths, spacing.
- Make the type treatment itself memorable, not a neutral delivery vehicle.
- Avoid the reflexive system stack (Inter/Roboto/Arial) unless the brief calls
  for that neutrality; load real faces via `next/font`.

## 5. Structure is information, not decoration

- Structural devices — numbering, eyebrows, dividers, labels — must encode
  something *true* about the content.
- `01 / 02 / 03` markers are only right when the content is genuinely a sequence
  (a real process, a typed timeline). Question them before adding them.
- The hero is a thesis: open with the most characteristic thing in the subject's
  world (a headline, image, demo, or interactive moment) — not a reflexive big
  number + label + gradient.

## 6. Motion deliberately, restraint everywhere else

- Decide *where and if* animation serves the subject: a load sequence, a
  scroll-triggered reveal, a hover micro-interaction, ambient atmosphere.
- One orchestrated moment beats scattered effects. Excess motion is itself a
  signal the design is AI-generated — when in doubt, cut it.
- **Spend boldness in one place** (the signature); keep everything around it
  quiet. Chanel's rule: before shipping, remove one accessory.
- Match complexity to the vision: maximalism needs elaborate execution;
  minimalism needs precise spacing, type, and detail. Elegance is executing the
  chosen vision well.

## 7. Copy is design material

Words exist to make the interface easier to understand. Bring the same
intentionality to copy as to spacing and color.

- Write from the user's side of the screen — name things by what people control
  ("notifications", not "webhook config"). Specific beats clever.
- Active voice; a control says what it does ("Save changes", not "Submit"). Keep
  one verb through a flow — a "Publish" button yields a "Published" toast.
- Treat errors and empty states as direction: say what happened and what to do,
  in the interface's voice. An empty screen is an invitation to act.
- Sentence case, plain verbs, no filler; each element does exactly one job.

## Quality floor (non-negotiable, don't announce it)

- Responsive down to mobile; nothing overflows or collapses below 360px.
- Visible keyboard focus; semantic HTML; labelled controls.
- `prefers-reduced-motion` respected.
- Use semantic tokens (`bg-background`, `text-foreground`, `border`, `ring`) per
  `shadcn` rather than scattering raw hex once the plan's tokens are wired in.
- Watch CSS specificity — type-based (`.section`) and element-based (`.cta`)
  selectors that fight over padding/margins are a common self-inflicted bug.

## Common mistakes to avoid

1. AI-slop defaults — Inter + purple gradient + evenly-spaced rounded cards.
2. Coding before the plan passes critique; tokens chosen ad hoc in the markup.
3. Decorative structure — numbered markers on non-sequential content, dividers
   that mean nothing.
4. Boldness spread everywhere — three "signature" elements means none of them is.
5. Motion for its own sake; ignoring `prefers-reduced-motion`.
6. Lorem ipsum that hides real layout problems until the end.
7. Skipping the quality floor (mobile, focus, a11y) in pursuit of the look.

## Checklist before "done"

- Subject, audience, and single job named; design grounded in the subject.
- Token plan (color/type/layout/signature) written, then critiqued and revised
  off the defaults; code derives from it.
- Display/body pairing is intentional; type is a feature, not a neutral default.
- One signature element; everything around it is quiet; motion is deliberate.
- Copy is specific, active-voice, consistent; empty/error states give direction.
- Quality floor met: responsive, keyboard-focusable, reduced-motion, semantic
  tokens; `pnpm lint` and `pnpm typecheck` pass.
