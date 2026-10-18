---
name: ui-polish
description: >-
  Polish UI surfaces, icons, microinteractions, and animations with exact
  design-engineering values — concentric border radius, optical alignment,
  layered shadows vs. borders, image outlines, icon strokes/cross-fades,
  frequency-based motion budgets, custom ease-out curves, and compositor
  performance. Use when building UI components, refining or deslopping an
  existing interface, implementing hover/press/enter/exit animations, or
  fixing UI that "feels off" or stutters.
---

# UI Polish & Motion Craft

This skill holds the design-engineering details that separate "assembled by an
AI" from "crafted by a design engineer" — with exact values, not vague advice.
Apply it when building components or running a refinement/deslop pass once the
underlying layout and accessibility are sound.

For aesthetic direction and page macrostructure, see `frontend-design`; for
data-dense layouts, see `dashboard-ui`; for WCAG accessibility, forms, and copy,
see `ux-quality`; for token architecture, see `design-tokens`; for text setting,
see `typography`.

## When to use

- Building or reviewing UI components (buttons, cards, modals, popovers, tabs,
  toasts, icons, skeletons).
- Refining or "deslopping" an existing screen that works functionally but feels
  unfinished, generic, or slightly off.
- Implementing or auditing hover/press states, enter/exit transitions, icon
  swaps, stagger reveals, or motion performance.

## 1. Surfaces, depth & optical alignment

- **Concentric border radius:** Where nested surfaces share an even inset (up to
  `24px` padding), set `outerRadius = innerRadius + padding` (plus border width).
  Never give a padded parent and child the exact same `rounded-*` class. Past
  `24px` padding, treat them as separate surfaces.
- **Optical alignment over geometry:** Give a button with a leading/trailing
  icon `2px` less padding on the icon side; nudge play triangles `1–2px` toward
  their point; use `text-box: trim-both cap alphabetic` as progressive
  enhancement to center text vertically in tight pills and badges.
- **Shadows for elevation, borders for structure:**
  - In **light mode**, prefer layered transparent `box-shadow` for elevated cards
    and buttons (`0 0 0 1px oklch(0 0 0 / 0.06), 0 1px 2px -1px oklch(0 0 0 / 0.06), 0 2px 4px oklch(0 0 0 / 0.04)`).
  - In **dark mode**, depth shadows disappear — elevate via whisper-quiet
    surface lightness steps (`base → +7% → +9% → +12%`) and a `1px` ring
    (`0 0 0 1px oklch(1 0 0 / 0.08)`). Never use colored glow shadows on dark.
  - Keep real `1px` borders for dividers, table cells, and form inputs (which
    need ≥ 3:1 contrast). Put `border: 1px solid transparent` under shadow-only
    rings so `forced-colors` mode still draws an edge.
- **Neutral image outlines:** Give content photos/screenshots a `1px` inset
  outline in pure black/white: `outline-1 -outline-offset-1 outline-black/10 dark:outline-white/10`
  (`oklch(0 0 0 / 0.1)` light, `oklch(1 0 0 / 0.1)` dark). Never use a tinted
  palette or accent border on images — it reads as a colored fringe.
- **Theme browser surfaces:** Style `::selection`, `caret-color`, focus rings,
  `meta[name="theme-color"]`, and `<html style="color-scheme: dark">` (plus
  explicit `background-color`/`color` on native `<option>` for Windows dark
  mode) from your tokens. Unthemed browser defaults are an instant assembly tell.

## 2. Icons

- **One icon library per surface** (Lucide, Phosphor, or Radix Icons). Never mix
  icon sets, and **never use emoji (`✨ 🚀 ⚡`) as UI or feature icons**.
- **Match icon stroke to text weight:** `1.5px` stroke beside weight `400` text,
  scaling to `2px` at `500–600` and `2.5px` at `700`.
- **Single SVG + `currentColor`:** Recolor hover, active, and disabled states via
  CSS `color`/`opacity` — never hardcode `fill="#..."` or `stroke="#..."` inside
  UI icons. Default to outline; use fill to mark the active/selected state.
  Mirror only directional icons under `[dir="rtl"]`.
- **Contextual icon cross-fade:** When an icon swaps on a low-frequency action
  (copy → check, play → pause), cross-fade over `300ms` with `scale(0.25 → 1)`,
  `opacity(0 → 1)`, and `blur(4px → 0px)` using `cubic-bezier(0.2, 0, 0, 1)` or
  `{ type: "spring", duration: 0.3, bounce: 0 }` (`bounce` is always `0`). Keep
  the `"✓ Copied"` state for `2.5s` inline — no toast needed.

## 3. Motion engineering & frequency budget

Every animation must earn its place by explaining a state change or spatial
origin. Frequency decides whether an element animates at all:

| Interaction frequency                                       | Motion budget                                                                 |
| ----------------------------------------------------------- | ----------------------------------------------------------------------------- |
| **100+×/day** (⌘K command palette, keyboard shortcuts)      | **0ms — no animation ever.** Instant open/close; only highlight slides `120ms`.|
| **Tens/day** (row hover, tab switch, keystrokes)            | `0ms` or `≤ 150ms` on `opacity`/`background-color` only. Never stagger.       |
| **Occasional** (tooltips, dropdowns, popovers, modals)      | Standard purposeful transition (`125–400ms` budget below).                    |
| **Rare / first-load** (hero entrance, onboarding, empty UI) | One orchestrated sequence (`100ms` stagger, max 3 primitives/page, one-shot). |

### Timing & easing canon

Never use `ease-in` on UI entrances or interactive controls — it starts slow and
feels sluggish. Tokenize strong curves instead of relying on weak browser defaults:

```css
:root {
  --ease-out: cubic-bezier(0.23, 1, 0.32, 1);     /* entrances & interactive feedback */
  --ease-in-out: cubic-bezier(0.77, 0, 0.175, 1); /* on-screen morphing / movement */
  --ease-drawer: cubic-bezier(0.32, 0.72, 0, 1);  /* sheets & drawers */
}
```

| Element                     | Enter duration | Exit duration (~60–70% of enter) | Notes                                             |
| --------------------------- | -------------- | -------------------------------- | ------------------------------------------------- |
| Button press (`:active`)    | `100–150ms`    | `150ms`                          | `scale(0.96)` or `scale(0.97)` on `:not(:disabled):active` |
| Tooltip / small popover     | `125–180ms`    | `100–120ms`                      | Hover delay `800ms` (then `0ms` warm); focus delay `0ms` |
| Dropdown / select / menu    | `150–220ms`    | `120–150ms`                      | Scale from `var(--transform-origin)`, never `center` |
| Modal / dialog / drawer     | `200–350ms`    | `150–220ms`                      | Modal scales `0.96 → 1` from `center`; drawer uses `--ease-drawer` |
| Staggered section entrance  | `300ms`        | `150ms` (`-12px` `translateY`)   | `100ms` stagger, `translateY(12px → 0)` + `blur(4px → 0)`, one-shot |

### Physicality & interruptibility

- **Never `scale(0)`:** Nothing appears from nothing. Start entrances at
  `scale(0.95)` (or `0.9–0.97`) + `opacity: 0`.
- **Transitions or springs, never `@keyframes`, for interactive state:**
  CSS transitions and springs retarget smoothly when reversed mid-flight;
  `@keyframes` restart from zero. Reserve keyframes for one-shot or continuous
  ambient loops.
- **Skip mount animation on state swaps:** Set `initial={false}` on
  `AnimatePresence` wrapping a state toggle so the default state doesn't animate
  on initial page render.
- **No bouncy UI:** Keep `bounce: 0` on UI state changes; reserve slight spring
  bounce (`0.1–0.2`) for physical drag/swipe release.
- **Instant focus rings (`0ms`):** Never transition `outline` or focus-ring
  `box-shadow` — keyboard users need the indicator on frame one.
- **No flashing spinners:** Delay showing spinners by `150ms` and hold for at
  least `300ms` once visible. Prefer structural skeletons matching content size.
- **Silent success:** Do not fire a celebratory `"Done!"` toast when the user
  can already see the mutated state inline. Reserve toasts for errors, undo
  windows, and background completions.

## 4. Compositor & animation performance

- **Animate `transform` and `opacity` only.** Animating `width`, `height`,
  `top`, `left`, `margin`, or `padding` triggers layout + paint every frame. For
  accordion/tab height changes, use `grid-template-rows: 0fr → 1fr`.
- **Never `transition: all` / `transition-all`.** Name the exact changing
  properties (`transition-property: transform, opacity` or `color, background-color`).
- **Gate hover effects behind `@media (hover: hover) and (pointer: fine)`** so
  touch taps don't latch a stuck hover state. Set
  `-webkit-tap-highlight-color: transparent` when a control has custom press
  feedback.
- **Suppress transitions on theme switch** (`disableTransitionOnChange` in
  `next-themes`) so every surface doesn't smear over `200ms` at once.
- **Contain overlay scroll:** Set `overscroll-behavior: contain` on scrollable
  dialogs, drawers, sheets, and menus so scrolling past the end never scrolls
  the page behind.
- **No scroll-event polling:** Never drive animations from `scrollTop`/`scrollY`
  listeners. Use `IntersectionObserver` (one-shot reveals, pausing off-screen
  loops) or CSS `animation-timeline: view()`.
- **Blur & layer discipline:** Keep one-shot transition `blur()` `≤ 8px`; never
  animate blur or `backdrop-filter` continuously or on large containers. Put
  grain/noise overlays on `fixed inset-0 pointer-events-none`. Never animate
  inherited CSS variables on a parent to drive child transforms. Use
  `will-change` surgically on the exact animated property (`will-change: transform`),
  never `will-change: all`.
- **Gate spatial motion behind `prefers-reduced-motion`:** Under
  `prefers-reduced-motion: reduce`, collapse movement, scale, and blur to a
  `≤ 150ms` opacity cross-fade rather than removing feedback completely.

## 5. Polish & deslop audit pass (existing UI)

Polish is refinement, not a concealed redesign — preserve the project's tokens
and component vocabulary. Fix in triage order:

1. **Broken interactions & accessibility** (keyboard traps, missing focus ring,
   keyframe toggle that cannot reverse, hover stuck on touch).
2. **Missing states** (every interactive control needs default, hover,
   `:focus-visible`, `:active`, disabled, loading, error, and success/empty).
3. **Design-system drift** (raw hex/px instead of semantic tokens, hand-rolled
   markup where a `shadcn` primitive exists).
4. **Surface, icon, and motion defects** (check the table below).

| Detection pattern                                                        | Fix                                                                 |
| ------------------------------------------------------------------------ | ------------------------------------------------------------------- |
| Padded parent and child with the same `rounded-*`                        | `outerRadius = innerRadius + padding`                               |
| `box-shadow: 0 0 0 1px` ring with no `border`                            | Add `border: 1px solid transparent` for `forced-colors`             |
| Tinted or accent outline on `<img>`                                      | `outline-1 -outline-offset-1 outline-black/10 dark:outline-white/10`|
| `transition-all` or `transition: all`                                    | List exact properties (`transition-[transform,opacity]`)            |
| `animation:` on `:hover` or a toggled open/active class                  | Replace with CSS `transition` or spring (`initial={false}`)         |
| `ease-in` on UI entrance, or bouncy overshoot on standard UI             | `--ease-out` (`cubic-bezier(0.23, 1, 0.32, 1)`), `bounce: 0`        |
| Entrance starting from `scale(0)` or `scale-0`                           | Start from `scale(0.95)` + `opacity: 0`                             |
| Popover/dropdown scaling from `center`                                   | `transform-origin: var(--transform-origin)`                         |
| `active:scale-90` or `:active` scale without disabled guard              | `enabled:active:scale-[0.97]` (`0.96–0.98`)                         |
| Movement/scale/blur without `prefers-reduced-motion` guard               | Gate with `motion-safe:` / `useReducedMotion()`, fallback to opacity|
| Theme toggle without transition suppression                              | `disableTransitionOnChange` on `ThemeProvider`                      |
| `will-change: all` or permanent `will-change` on static elements         | Name exact property (`will-change: transform`) or remove            |
| Hardcoded `fill="#..."`/`stroke="#..."` in UI icons, or emoji icons      | Single icon library using `currentColor`                            |
| Scrollable modal/drawer/menu without `overscroll-behavior`               | Add `overscroll-behavior: contain` (`overscroll-contain`)           |

## Checklist before "done"

- Nested rounded containers use concentric radii; icon buttons are optically
  balanced; images have `1px` neutral inset outlines.
- Elevation uses layered transparent shadows in light mode and surface lightness
  steps + hairline rings in dark mode; browser chrome (`::selection`,
  `color-scheme`, `theme-color`) is themed.
- Single icon library (`currentColor`, stroke matched to text weight); no emoji
  icons.
- High-frequency actions (`⌘K`, row hover, tab switch) are instant (`0–150ms`);
  entrances use `--ease-out` from `scale(0.95)` and trigger `transform-origin`;
  exits are `60–70%` of enter duration.
- Only `transform`/`opacity` animate; no `transition-all`; hover gated on
  `(hover: hover)`; `prefers-reduced-motion` falls back to opacity cross-fade.

## Attribution

Distilled and synthesized from [`jakubkrehel/skills`](https://github.com/jakubkrehel/skills)
(`better-ui`), [`emilkowalski/skills`](https://github.com/emilkowalski/skills)
(`improve-animations`, MIT), [`ibelick/ui-skills`](https://github.com/ibelick/ui-skills)
(`baseline-ui`, `fixing-motion-performance`), and [`pbakaus/impeccable`](https://github.com/pbakaus/impeccable)
(`craft-floor`, `polish`, Apache-2.0).
