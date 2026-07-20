---
name: ux-quality
description: >-
  Make web UI actually usable and accessible — keyboard/focus, interaction and
  loading states, forms, navigation, and motion. Use when building or reviewing
  any interactive surface (forms, modals, menus, tables, flows) for usability,
  accessibility (WCAG), interaction states, or the "it looks fine but feels
  janky / fails a keyboard or screen-reader" class of problems.
---

# UX Quality

This is the usability layer *beneath* the visual layer. `frontend-design`
decides how a UI looks and `dashboard-ui` decides how data is laid out — this
skill decides whether the result is actually **operable**: reachable by
keyboard, legible to a screen reader, clear in every state, and forgiving of
mistakes. Most "this feels unprofessional / janky" feedback traces to a gap
here, not to the palette.

Pair with `shadcn` and `nextjs-modern` for implementation; cross-reference
`frontend-design` (look), `dashboard-ui` (data density), and `react-performance`
(runtime perf) instead of duplicating them.

## When to use

- Building or reviewing forms, modals, menus, tables, multi-step flows, or any
  interactive component.
- An audit pass: "is this accessible / keyboard-navigable / usable on mobile?"
- Debugging the "looks fine but feels off" class — missing states, no focus
  ring, layout-shifting press, errors with no recovery path.
- Designing empty / loading / error states for any data surface.

## Priority order

Fix top-down — the first two categories are correctness, not polish.

| # | Category                | Severity  |
| - | ----------------------- | --------- |
| 1 | Accessibility           | CRITICAL  |
| 2 | Interaction & touch     | CRITICAL  |
| 3 | Forms & feedback        | HIGH      |
| 4 | Navigation              | HIGH      |
| 5 | Motion                  | MEDIUM    |

## 1. Accessibility (CRITICAL)

The floor, not an enhancement. Verify with a keyboard and a screen reader, not
by eye.

- **Contrast** ≥ 4.5:1 for body text (3:1 for large text and meaningful UI
  glyphs); check light *and* dark independently — don't assume one transfers.
- **Visible focus** on every interactive element (2–4px ring); never remove the
  outline without replacing it. Tab order must match visual order.
- **Full keyboard operability** — everything clickable is reachable and
  triggerable by keyboard; no mouse-only or hover-only actions.
- **Name every control** — icon-only buttons get an `aria-label`; inputs get a
  real `<label>`; meaningful images get descriptive `alt` (decorative → `alt=""`).
- **Semantic HTML** — real `<button>`/`<a>`/`<nav>`/`<h1–h6>`; sequential
  headings, no level skips. Don't rebuild native controls from `<div>`s.
- **Color is never the only signal** — pair status color with an icon, label, or
  shape (red border *and* an error message).
- **`prefers-reduced-motion`** respected; support browser zoom / text scaling
  without clipping or overlap (never disable zoom in the viewport meta).
- **Live regions** — announce async results (`aria-live="polite"`, or
  `role="alert"` for errors); don't let toasts steal focus.

## 2. Interaction & touch (CRITICAL)

- **Hit targets** ≥ 44×44px; keep ≥ 8px between adjacent targets. If the icon is
  smaller, extend the hit area (padding / `hitSlop`) — don't shrink the target.
- **Immediate press feedback** (≤ ~100ms): opacity/background/elevation change.
  A control with no pressed state feels broken.
- **Don't shift layout on press/hover** — animate `transform`/`opacity`, not
  width/height/margins that nudge neighbors and cause jitter.
- **Tap, not hover, for primary actions** — hover-only menus/actions are
  invisible on touch; always give a tap/click path.
- **`cursor: pointer`** on clickable elements; `touch-action: manipulation` to
  drop the 300ms tap delay.
- **Disabled means disabled** — reduced opacity (~0.4–0.5) *and* a non-interactive
  semantic attribute, not just a faded look that still fires handlers.
- **Async work shows progress** — disable the trigger and show a spinner /
  skeleton when an operation exceeds ~300ms; never leave a dead-looking click.

## 3. Forms & feedback (HIGH)

Forms are where usability is won or lost. (Use `shadcn`'s `Field`/form
primitives for the wiring; this governs behavior.)

- **Visible persistent label** per field — placeholders are not labels; they
  vanish on input and fail screen readers.
- **Validate on blur, not on every keystroke**; show the error **below the
  field**, not only in a summary at the top.
- **Errors say cause + fix** ("Email is missing an @", not "Invalid input") and
  offer a recovery path (retry / edit / help).
- **On submit error, move focus to the first invalid field**; for many errors,
  show a summary with anchor links to each.
- **Mark required fields**; use semantic `type`/`inputmode` (email, tel, number)
  and `autocomplete` so mobile keyboards and autofill work.
- **Confirm before destructive or bulk actions**, or offer an **Undo** toast;
  destructive buttons use a danger color and sit apart from the primary action.
- **Auto-save / warn on unsaved changes** for long forms and dismissible sheets.
- **Submit feedback** — loading → explicit success or error state; confirm
  completed actions (checkmark / toast).

## 4. Navigation (HIGH)

- **Highlight the current location** (active nav item by color/weight/indicator)
  — users must always know where they are.
- **Predictable back** — restores prior scroll position, filters, and input;
  never silently resets the stack or jumps to home.
- **Everything important is URL-addressable** (deep-linkable) for sharing,
  bookmarks, and refresh — central to web UX.
- **Consistent placement** across pages; don't mix competing nav patterns
  (tabs + sidebar + bottom bar) at the same level.
- **Persistent sidebar on desktop**; bottom tab bar (≤ 5 items, icon + label) on
  mobile. Breadcrumbs for hierarchies 3+ levels deep.
- **Modals aren't navigation** — every modal/sheet has a clear close affordance
  (Esc, swipe-down on mobile); don't route primary flows through them.
- Keep **destructive actions** (logout, delete account) visually separated from
  routine nav items.

## 5. Motion (MEDIUM)

Motion should explain a change, never decorate. (`frontend-design` governs *how
much* personality; this governs *mechanics*.)

- **150–300ms** for micro-interactions, ≤ 400ms for larger transitions; avoid
  > 500ms.
- **`transform`/`opacity` only** — never animate `width`/`height`/`top`/`left`
  (causes reflow / CLS).
- **ease-out entering, ease-in exiting**; exit ~60–70% of enter duration so it
  feels responsive. Avoid linear easing for UI.
- **Interruptible & non-blocking** — a tap/gesture cancels an in-progress
  animation; never freeze input while animating.
- **Spatial continuity** — modals/sheets animate from their trigger; forward
  navigation moves one direction, back the opposite.
- Always gate non-essential motion behind `prefers-reduced-motion`.

## Quick mobile / touch checklist

- Respect safe areas (notch, Dynamic Island, gesture bar); fixed bars reserve
  padding so content isn't hidden behind them.
- ≥ 16px body inputs (smaller triggers iOS auto-zoom); no horizontal scroll;
  prefer `min-h-dvh` over `100vh`.
- Don't hijack system gestures (back-swipe, pull-to-refresh) or require
  pixel-perfect taps on thin edges.

## Common mistakes to avoid

1. Removed focus ring "because it's ugly" — keyboard users are now lost.
2. `<div onClick>` instead of `<button>` — no keyboard, no role, no focus.
3. Placeholder used as the only label.
4. Error shown only at the top of the form, far from the field that caused it.
5. Status conveyed by color alone (red/green with no icon or text).
6. Layout-shifting hover/press effects that make the page twitch.
7. No empty / loading / error state — the unhappy paths are ~80% of perceived
   polish; design them for every data surface.
8. Animating `width`/`height` (jank) and ignoring `prefers-reduced-motion`.

## Heuristic audit (Nielsen + Krug)

When doing a usability pass (not just accessibility), scan with these lenses:

- **Don't make me think** — labels and CTAs self-evident; no jargon or clever
  marketing copy on controls users must operate.
- **Clicks are fine if each is painless** — three obvious steps beat one
  ambiguous step; show progress on multi-step flows.
- **Nielsen's top heuristics** — system status visible (loading/saving/errors);
  speak the user's language; user control (undo, cancel, back); consistency;
  error prevention over error messages; recognition over recall (visible options,
  not memorized commands).
- **Severity triage** — catastrophic (blocks core task) before cosmetic; fix
  blockers before polish.

For a full heuristic catalog and severity scale, see the `ux-heuristics` skill
in [`wondelai/skills`](https://github.com/wondelai/skills) if installed; the
rules above cover the highest-frequency gaps in scaffolded apps.

## Checklist before "done"

- Operable by keyboard alone; visible focus everywhere; tab order = visual order.
- Contrast ≥ 4.5:1 verified in light and dark; color never the sole signal.
- Every control named (label / `aria-label` / `alt`); semantic HTML, ordered
  headings.
- Hit targets ≥ 44px with press feedback; no layout shift on hover/press;
  disabled is truly inert.
- Forms: visible labels, blur validation, field-level errors with cause + fix,
  focus to first invalid field, destructive actions confirmed.
- Active nav state shown; back is predictable; key views are deep-linkable.
- Motion 150–300ms, `transform`/`opacity` only, reduced-motion respected.
- Empty, loading, and error states designed for every data surface.

## Attribution

The rule catalog is distilled and adapted (web-focused) from
[`ui-ux-pro-max`](https://github.com/nextlevelbuilder/ui-ux-pro-max-skill)
(99 UX guidelines across accessibility, interaction, forms, navigation, motion;
sources include WCAG, Apple HIG, and Material Design), MIT License. The original
ships a Python/CSV search engine; only the guidance is adopted here, restructured
into RonyKit's self-contained skill format.
