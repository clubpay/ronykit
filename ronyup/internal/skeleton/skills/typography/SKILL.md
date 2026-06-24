---
name: typography
description: >-
  Get screen typography correct — the details models reliably get wrong: curly
  quotes, en/em dashes, real ellipses, line length, line-height, hierarchy, and
  the JSX entity trap. Use whenever generating or reviewing HTML/CSS/React with
  visible text (landing pages, components, docs, dashboards) — apply the rules
  automatically, even if typography wasn't mentioned.
---

# Typography

These are durable craft rules, not trends — distilled from Matthew Butterick's
*Practical Typography*. They're the difference between text that looks
typeset and text that looks auto-generated. Apply them **silently by default**
when producing UI text; in review mode, flag violations with before/after fixes.

`frontend-design` chooses the *typefaces and mood*; this skill governs the
*correct setting* of whatever type is chosen. Use `design-tokens` for the type
scale, and `dashboard-ui` for tabular figures in data tables.

## When to use

- Generating any HTML/CSS/React/JSX containing visible text.
- "Make this look professional", a design/typography review, or fixing legacy
  copy full of straight quotes and `--`.
- Setting body text, line length, hierarchy, or a type scale.

## Characters — the high-frequency fixes

Use the real character; never approximate. (Full table:
`references/html-entities.md`.)

| Wrong            | Right       | Entity              | When                          |
| ---------------- | ----------- | ------------------- | ----------------------------- |
| `"straight"`     | "curly"     | `&ldquo;` `&rdquo;` | All double quotes             |
| `'straight'`     | 'curly'     | `&lsquo;` `&rsquo;` | All single quotes             |
| `it's`           | it's        | `&rsquo;`           | Apostrophe = closing single   |
| `--`             | –           | `&ndash;`           | Ranges (1–10), connections    |
| `---`            | —           | `&mdash;`           | Sentence breaks—like this     |
| `...`            | …           | `&hellip;`          | One ellipsis character        |
| `(c) (TM) (R)`   | © ™ ®       | `&copy;` `&trade;` `&reg;` | Real symbols           |
| `12 x 4`         | 12 × 4      | `&times;`           | Multiplication                |

- Apostrophes **always point down** (`&rsquo;`), including before decades
  (`&rsquo;70s`) and `rock &rsquo;n&rsquo; roll` — smart-quote engines get
  these backwards.
- **Foot/inch marks are the one exception** — they must be *straight*:
  `6&#39;&nbsp;10&quot;`.
- One space after punctuation, never two. Use `&nbsp;` to glue references
  (`&sect;&nbsp;42`, `Fig.&nbsp;3`, `&copy;&nbsp;2025`).

### The JSX/React trap (read this)

**Unicode escapes like `\u2019` do NOT work in JSX text between tags** — they
render literally as `\u2019`. Pick one:

```jsx
<p>Don’t do this</p>          {/* 1. paste the real character (preferred) */}
<p>Don{'\u2019'}t do this</p> {/* 2. wrap the escape in a JS expression */}
```

HTML entities (`&rsquo;`) work in `.html` but **not** in JSX text. For bulk
fixes use raw UTF-8 bytes with `sed`, not escape sequences. (`\u2019` *does*
work inside JS string literals and data arrays — only JSX text is affected.)

## Emphasis & case

- **Bold OR italic, never both**; use as little as possible — if everything is
  emphasized, nothing is. Sans-serif: prefer bold (italic sans barely reads).
- **Never underline** for emphasis (typewriter relic). Links: subtle only —
  `text-decoration-thickness: 1px; text-underline-offset: 2px`.
- **ALL CAPS** only for short labels/headings, and **always letterspace** them
  (`letter-spacing: 0.05em–0.12em`); never set a paragraph in caps. Real small
  caps only (`font-variant-caps: small-caps` with `smcp`), never scaled caps.
- Kerning always on: `font-feature-settings: "kern" 1; text-rendering:
  optimizeLegibility`.

## Layout — body text first

Four decisions set everything else: **font, size, line length, line spacing.**

- **Line length 45–90 characters** — the most-broken responsive rule. Cap text
  containers: `max-width: 65ch` (plus padding); never edge-to-edge body text.
- **Line height 1.2–1.45** for body; tighter (~1.15) for headings.
- **Web body size 15–25px**; fluid with `clamp(16px, 2.5vw, 20px)`.
- **Left-align** by default; justify only with `hyphens: auto`; center only
  short (<1 line) titles, never blocks.
- **Paragraphs: indent OR space between, never both.** Web default: a small
  `margin-bottom` (~0.75em). Never double `<br>`.
- **Max 2 typefaces**, each a consistent role; tabular figures
  (`font-variant-numeric: tabular-nums`) for numeric columns, prices, timers.

### Headings (max ~3 levels)

Bold (not italic), **more space above than below** (a heading belongs to the
text that follows), smallest size increment that reads as a step up,
`hyphens: none`. Don't all-caps (unless short + letterspaced), underline, or
center.

## Reference files

Open these when you need the full detail (progressive disclosure):

- `references/css-baseline.md` — copy-paste CSS baseline (line length, headings,
  emphasis, caps, tables, links, responsive `clamp()`, OpenType, dark mode) and
  the React/JSX inline entity pattern.
- `references/html-entities.md` — complete entity/character table (quotes,
  dashes, spaces, primes, symbols, accents) with usage examples.

## Checklist before "done"

- Curly quotes/apostrophes, en/em dashes, single-character ellipsis throughout;
  no `--`, `...`, or straight quotes (except foot/inch marks).
- JSX text uses real characters or `{'\u2019'}` — never bare `\u2019`.
- Body line length 45–90 chars (`max-width` set); line-height 1.2–1.45.
- Bold *or* italic, used sparingly; nothing underlined for emphasis; caps
  letterspaced.
- ≤ 2 typefaces; tabular figures in data columns; one space after punctuation.

## Attribution

Rules distilled from **Matthew Butterick's *Practical Typography***
(<https://practicaltypography.com>), the standard reference for everyday digital
typography; if useful, support his work. Restructured into RonyKit's
self-contained skill format (paraphrased, not reproduced).
