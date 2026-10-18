---
name: ux-quality
description: >-
  Review and build web UIs against WCAG 2.2 AA accessibility, keyboard/focus
  discipline, 8-state interactive completeness, form ergonomics, layout
  resilience, and clear UX copy. Use when building or auditing forms, dialogs,
  navigation, interactive controls, responsive/RTL layouts, or running a UX
  and accessibility review.
---

# UX Quality, Accessibility & Layout Resilience

Great visuals fail if a keyboard user gets trapped, a mobile user can't tap a
control, a long translation blows out a flex row, or a destructive button deletes
data with no recovery. Apply these rules to every screen and component.

For exact surface, icon, and animation curves/timings, see `ui-polish`.

## 1. Semantic HTML & keyboard/focus discipline

- **Native elements first:** `<button type="button">` for actions, `<a href>`
  for navigation (preserves Cmd/Ctrl-click and middle-click), `<label>` for form
  controls, `<fieldset>`/`<legend>` for radio/checkbox groups, `<dialog>` for
  modals. Never attach `onClick` to a `<div>` or `<span>` without `role`,
  `tabIndex={0}`, and `onKeyDown` (`Enter` + `Space`).
- **Landmarks & heading hierarchy:** Every page has `<header>`, `<nav>`,
  `<main id="main-content">`, and a skip-to-content link as the first focusable
  element. Headings increment by one level (`h1 → h2 → h3`) by document outline,
  not visual size.
- **Visible, instant focus ring:** Never `outline: none` without a replacement.
  Use `:focus-visible` (`2px solid var(--ring)`, `outline-offset: 2px`, ≥ 3:1
  contrast) and `:focus-within` for compound inputs. Focus rings must appear in
  **`0ms`** — never transition `outline`.
- **Sticky header clearance:** Set `scroll-padding-block-start` on the scroll
  container (or `scroll-margin-top` on targets) matching any sticky header
  height so anchored links and keyboard-focused elements never hide under chrome.
- **Overlays & focus traps:** Modals, sheets, and drawers trap `Tab`/`Shift+Tab`,
  close on `Escape`, set `overscroll-behavior: contain`, lock background scroll,
  and **restore focus to the trigger** when closed.
- **ARIA wiring:**
  - Icon-only buttons require `aria-label`; decorative SVGs require
    `aria-hidden="true"`.
  - Disclosures/accordions/menus set `aria-expanded` and `aria-controls`.
  - Async status updates use `aria-live="polite"` (`role="status"`); urgent
    errors use `role="alert"`.

## 2. Complete interaction states (all 8)

Design and implement all 8 states for every interactive control or data view:

1. **Default** — clear affordance (buttons look clickable; inputs have ≥ 3:1
   border contrast; links are distinguishable beyond color alone).
2. **Hover** — gated behind `@media (hover: hover) and (pointer: fine)`.
3. **Focus (`:focus-visible`)** — high-contrast ring, instant `0ms`.
4. **Active / pressed (`:active`)** — tactile press (`enabled:active:scale-[0.97]`).
5. **Disabled (`disabled` / `aria-disabled`)** — reduced contrast, `cursor: not-allowed`,
   press/hover animations suppressed. Explain *why* if non-obvious.
6. **Loading / pending** — disable duplicate submission while preserving button
   width (swap label smoothly or show inline spinner; never jump width).
7. **Empty** — distinguish first-use empty (teach + primary CTA) from
   filtered-zero empty (echo active query + `"Clear filters"` button).
8. **Error** — inline, specific, and recoverable without wiping user input.

## 3. Touch targets & input ergonomics

- **Hit area ≥ `44 × 44px` on touch (`32 × 32px` desktop pointer minimum):**
  Expand small visual icons (`16–20px`) with padding or an invisible `::after`
  hit area (`min-h-11 min-w-11` on mobile) and keep ≥ `8px` gap between targets.
- **Single hit target for labels + controls:** Wrap checkboxes/radios and their
  `<label>` (or connect via `htmlFor` with adjacent padding) so there is zero
  dead gap between text and control.
- **Mobile keyboard & zoom safety:** Set `type` (`email`, `tel`, `url`, `search`),
  `inputMode` (`numeric`, `decimal`), `autoComplete` (`email`, `current-password`,
  `one-time-code`), and `enterKeyHint`. Keep input `font-size ≥ 16px` on mobile
  (`text-base sm:text-sm`) so iOS Safari never auto-zooms on focus.

## 4. Form UX that respects the user

- **Visible labels always:** Placeholders vanish on typing and fail contrast —
  never use them as the only label. Placeholders show an example (`name@example.com`)
  or end with `…` (`Search…`), never instructions.
- **Never block paste:** Never attach `onPaste` + `preventDefault()` to password,
  code, or confirmation fields — it breaks password managers and accessibility.
- **Disable spellcheck on identifiers:** Set `spellCheck={false}` and
  `autoCapitalize="none"` on emails, usernames, API keys, URLs, and codes.
- **Validate late, clear early:** Validate on **blur** or **submit** (never fire
  `"Invalid email"` on the first keystroke), then clear the error **immediately
  on change** once valid.
- **Programmatic error linking:** Set `aria-invalid="true"` on the invalid field
  and link its error message via `aria-describedby`. On submit failure, focus the
  first invalid input and show a summary if the form scrolls.
- **Dirty-form protection:** Warn before discarding unsaved edits when closing a
  drawer/modal or navigating away.

## 5. Layout resilience, responsive & i18n safety

- **Space first, borders second:** Group related elements with tight spacing and
  separate groups with ≥ `2×` that gap before adding divider lines or card boxes.
- **Prevent flex/grid blowout:** Add `min-w-0` on flex children and use
  `minmax(0, 1fr)` (never bare `1fr`) on CSS Grid tracks containing text or
  tables so `truncate` and `overflow-x: auto` work instead of expanding the page.
- **Viewport units that work:** Use `min-h-dvh` (`100dvh`, never `100vh`) so
  mobile browser toolbars don't clip bottom CTAs, and `w-full` (`width: 100%`,
  never `100vw`, which includes scrollbar width and causes horizontal overflow).
- **Never wrap clickable labels:** Buttons, tabs, nav links, and badges must stay
  on a single line (`white-space: nowrap` / `shrink-0`).
- **Text-growth & zoom resilience:** Never set a fixed `height` on elements that
  hold text — use `min-height` and vertical padding so 200% browser zoom and
  German/French translations (+30–40% length) wrap cleanly.
- **Logical CSS properties for RTL:** Prefer `margin-inline-start` (`ms-*`),
  `padding-inline` (`px-*`), `inset-inline-end` (`end-*`), and `text-align: start`
  (`text-start`) over physical `left`/`right`.
- **Safe areas:** Pad fixed bottom bars and sheets with `env(safe-area-inset-bottom)`.

## 6. UX copy, confirmation & i18n formatting

- **Verb + object button labels:** Label actions with what they do (`"Create API key"`,
  `"Delete workspace"`, `"Keep editing"`), never `"OK"`, `"Yes"` / `"No"`, or
  generic `"Submit"`.
- **Descriptive links:** Write `"Read the rate-limit docs"`, never `"Click here"`
  or repeated bare `"Learn more"` links.
- **Positive toggle labels:** Phrase switches/checkboxes as what happens when
  **ON** (`"Send weekly digest"`), never double negatives (`"Don't send emails"`).
- **Undo over confirmation modals:**
  - **Reversible actions** (archive, move, mark read, soft-delete): execute
    optimistically and show a `5–10s` toast with an **`Undo`** button.
  - **Irreversible destructive actions** (permanent delete, revoke key, drop
    data): require a blocking `AlertDialog` naming the exact resource +
    consequence, with a danger-styled verb button (`"Delete 'prod-db'"`).
- **Actionable errors:** State *what failed*, *why*, and *what to do next*
  (`"Payment declined by card issuer — update your card or try another method"`),
  never `"Something went wrong"` or raw error codes.
- **Locale-aware formatting:** Format dates, times, numbers, and currencies via
  `Intl.DateTimeFormat` and `Intl.NumberFormat`. Keep translatable strings whole
  (ICU plurals, never `"You have " + count + " items"`), and mark brand names,
  code identifiers, and keyboard shortcuts with `translate="no"`.

## UX audit checklist

- [ ] Full flow completable with `Tab`, `Shift+Tab`, `Enter`, `Space`, arrows, `Escape`
- [ ] Instant (`0ms`) `:focus-visible` ring on every interactive control; sticky headers use `scroll-padding-block-start`
- [ ] All 8 states designed (default, hover, focus, active, disabled, loading, empty, error)
- [ ] Every input has a visible `<label>`, `aria-describedby` error wiring, no blocked paste, and `≥ 16px` mobile font
- [ ] Touch targets ≥ `44 × 44px`; flex/grid tracks use `min-w-0` / `minmax(0, 1fr)`; `100dvh` & `w-full` used
- [ ] Buttons use verb + object labels (`nowrap`); reversible actions offer Undo; irreversible actions use `AlertDialog`

## Attribution

Synthesized from [`vercel-labs/web-interface-guidelines`](https://github.com/vercel-labs/web-interface-guidelines),
[`ibelick/ui-skills`](https://github.com/ibelick/ui-skills) (`fixing-accessibility`),
and [`jakubkrehel/skills`](https://github.com/jakubkrehel/skills) (`better-layout`,
`better-writing`, `better-accessibility`, MIT).
