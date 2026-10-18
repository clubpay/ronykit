---
name: typography
description: >-
  Apply professional web typography: font pairing by surface mode, modular/fluid
  vs. fixed type scales, unitless line-height by role, weight floors, line
  length, vertical rhythm, OpenType CSS properties, dark-mode optical
  adjustments, and typographic hygiene. Use when setting up CSS/Tailwind
  typography, designing article/editorial layouts, choosing fonts, or fixing UI
  text that feels cramped, generic, or hard to scan.
---

# Web Typography

Typography is 90% of web design. Good type makes an unadorned page feel
authored; sloppy type makes the most expensive component library look cheap.

## 1. Font selection & pairing by surface mode

Match your font stack to the surface mode (see `frontend-design` and
`dashboard-ui`):

- **Operate mode (dashboards, admin, SaaS tools):** **1 crisp sans + 1 mono.**
  Never use serif display faces on operational dashboards. Rely on weight
  (`400 / 500 / 600`) and 4-tier color/opacity for hierarchy.
- **Persuade & Read modes (marketing, editorial, docs):** Pair at most **2–3
  families** with distinct roles (`--font-display`, `--font-sans`, `--font-mono`).
  Pair fonts that share a similar **x-height and visual proportion** while
  contrasting in structure (e.g., high-contrast editorial serif display +
  neutral neo-grotesque body).
- **Avoid overused defaults as the sole personality** on Persuade surfaces:
  Inter, Roboto, Open Sans, Lato, Montserrat, Arial. Consider characterful
  variable families:
  - *Geometric / Neo-grotesque:* Plus Jakarta Sans, Geist, General Sans, Satoshi,
    Cabinet Grotesk,Space Grotesk, Instrument Sans, DM Sans, Outfit
  - *Editorial / Serif:* Instrument Serif, Fraunces, Newsreader, Playfair
    Display, Source Serif 4, News Cycle
  - *Monospace:* JetBrains Mono, Geist Mono, Commit Mono, Fira Code, Berkeley Mono
- **Load only the weights you use** (`400`, `500`, `600`, `700`) via `next/font`
  or `font-display: swap` + `size-adjust` fallback metrics so the browser never
  synthesizes blurry faux bold or faux italic.

## 2. Type scale & size floors (`rem`, never `px` for body)

- **Persuade / Read surfaces:** Use a **fluid `clamp()` scale** for display and
  section headings (`h1`–`h3`) so headlines scale smoothly from mobile to desktop.
- **Operate surfaces (dashboards/apps):** Use a **fixed `rem` scale** (ratio
  `1.125` Major Second or `1.200` Minor Third) so wider viewports show *more
  data*, not larger labels.
- **Hard size floors:**
  - **Body prose:** `1rem` (`16px`) default (`15–18px` for long-form reading).
  - **UI controls, tables, dense labels:** `0.875rem` (`14px`) standard,
    `0.8125rem` (`13px`) compact, **`0.75rem` (`12px`) absolute floor** (badges,
    captions, kbd hints only). Never ship `10px` or `11px` text.
  - **Mobile form inputs (`<input>`, `<textarea>`, `<select>`):** Must be
    **`≥ 16px` (`1rem`, `text-base sm:text-sm`)** on mobile viewports so iOS
    Safari never auto-zooms the page on focus.

## 3. Measure (line length) & deliberate wrapping

- **Body prose (`p`, `li`, `blockquote`):** `60–75ch` (`max-w-[65ch]`). Never let
  article or modal paragraphs stretch across a `1200px` container.
- **Lead paragraphs / intros:** `45–60ch` at `1.125–1.25rem`.
- **Card descriptions & UI help text:** `35–50ch`.
- **Headings:** `20–35ch` with `text-wrap: balance` (`text-balance`, effective up
  to ~6 lines — never apply `balance` to long paragraphs).
- **Body copy:** Apply `text-wrap: pretty` (`text-pretty`) to prevent lonely
  single-word orphans on the last line, plus `overflow-wrap: break-word` so long
  URLs or tokens never cause horizontal scroll.
- **Interactive labels:** Set `white-space: nowrap` on buttons, pills, badges,
  and nav links so clickable text never breaks across two lines.

## 4. Unitless line-height, tracking & weight floors

Line-height and letter-spacing must **tighten as font size grows**. Always use
**unitless** `line-height` so child elements inherit the ratio, not a fixed pixel
height:

| Role                                     | Size range    | `line-height` | `letter-spacing`  | Weight floor |
| ---------------------------------------- | ------------- | ------------- | ----------------- | ------------ |
| Display / Hero                           | `≥ 2.5rem`    | `1.05 – 1.12` | `-0.03em – -0.02em` | `300 – 700`  |
| Headings (`h1`–`h3`)                     | `1.25–2.25rem`| `1.15 – 1.30` | `-0.02em – -0.01em` | `500 – 700`  |
| Body prose                               | `1rem–1.125rem`| `1.50 – 1.65` | `0`               | `400`        |
| Card / UI text wrapping to **3+ lines**  | `0.8125–0.9375rem` | **`≥ 1.40`** | `0`               | `400`        |
| Single-line UI controls (buttons/inputs) | `0.8125–0.875rem` | `1.25 – 1.35` | `0`               | `400 – 500`  |
| Eyebrows / ALL CAPS labels (≤ 3 words)   | `0.75rem`     | `1.20 – 1.35` | `+0.06em – +0.10em` | `500 – 600`  |

- **Never apply `leading-none` (`1`) or `leading-tight` (`1.25`) to text that
  wraps to 3+ lines** — ascenders and descenders collide.
- **Weight floor below `18px`:** Use weight `400` or heavier for any text under
  `18px`. Thin/light weights (`100–300`) are only legible at `28px+`.
- **Never set lowercase text with wide positive tracking**, and never set body
  copy in `UPPERCASE`.

## 5. Vertical rhythm, optical trim & dark mode

- **Asymmetric heading margins:** A heading belongs to the section *below* it,
  not the one above (`margin-top: 2em; margin-bottom: 0.5em`, i.e., top ≥ `2–3×`
  bottom).
- **Paragraph spacing:** Use `margin-block-end: 1em` or `space-y-4` between
  paragraphs — never empty `<br><br>` tags.
- **Optical cap-height centering:** On tight buttons, badges, and pills where a
  font's descender box makes text look top-heavy, use progressive enhancement:
  `text-box: trim-both cap alphabetic`.
- **Dark mode optical compensation:** Light text on dark backgrounds Halates
  (appears slightly bolder). Set `-webkit-font-smoothing: antialiased` **once on
  `html`/`body`** (never scatter it across components), and step body text down
  slightly (`oklch(0.93 0.005 H)` instead of pure `#ffffff`).
- **Keep text selectable:** Only apply `user-select: none` to drag handles and
  interactive games — never to headings, tables, or UI text users copy.

## 6. OpenType via high-level CSS properties

Prefer standard CSS font properties over low-level `font-feature-settings` or
`font-variation-settings` so fallback fonts still render properly and rules
don't clobber each other:

- **Numbers in tables, KPIs, prices, timers, and axes:**
  `font-variant-numeric: tabular-nums` (`tabular-nums`).
- **Numbers in running prose:** `font-variant-numeric: proportional-nums`.
- **Slashed zero for IDs/codes/hashes:** `font-variant-numeric: tabular-nums slashed-zero`.
- **Variable fonts:** Set `font-weight: 550` instead of `font-variation-settings: "wght" 550`,
  and leave `font-optical-sizing: auto` enabled.
- **Real typographic punctuation:** Use curly quotes (`“ ”` `‘ ’`), real
  apostrophes (`’`), en-dashes for ranges (`10–20`), em-dashes (`—`), and real
  ellipsis (`…`) — see `references/html-entities.md`.

## Reference files

- `references/css-baseline.md` — Drop-in CSS custom properties, fluid `clamp()`
  scale, prose spacing, and OpenType classes.
- `references/html-entities.md` — Quick reference for proper typographic
  punctuation, dashes, quotes, and symbols.

## Attribution

Synthesized from [`jakubkrehel/skills`](https://github.com/jakubkrehel/skills)
(`better-typography`, MIT) and [`pbakaus/impeccable`](https://github.com/pbakaus/impeccable)
(Apache-2.0).
