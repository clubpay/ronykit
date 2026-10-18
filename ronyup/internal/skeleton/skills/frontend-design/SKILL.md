---
name: frontend-design
description: >-
  Create distinctive, production-grade frontend interfaces with high design
  quality and zero "AI slop". Use when the user asks to build or restyle web
  components, pages, landing pages, or applications (React, Next.js, Vue,
  Svelte, or plain HTML/CSS) where visual identity, typography, color, motion,
  and composition matter.
---

# Frontend Design

This skill guides creation of distinctive, production-grade frontend interfaces
that avoid generic "AI slop" aesthetics. Implement real working code with
exceptional attention to aesthetic direction, surface mode, and craft details.

Pair with sibling skills by concern:

- **Operational dashboards / admin / data-dense SaaS** → `dashboard-ui`
- **Accessibility, keyboard/focus, forms, UX copy, layout resilience** → `ux-quality`
- **Exact surface, icon, microinteraction, and animation polish** → `ui-polish`
- **Color ramps, semantic CSS custom properties, dark mode** → `design-tokens`
- **Font pairing, fluid vs. fixed scales, OpenType, rhythm** → `typography`

## 1. Pre-flight scan & reference extraction (before writing code)

Before proposing a direction or generating components, inspect the workspace:

1. **`docs/design/<app>-frontend-design.md`** (if present) — the authoritative
   design system, palette, typography, and screen specs. Never drift from it.
2. **`globals.css` / `tailwind.config.*` / `components.json`** — existing CSS
   variables, Tailwind version (v3 vs. v4 `@theme`), and `shadcn` primitives.
3. **`package.json` & root layout** — loaded fonts (`next/font`), icon library
   (`lucide-react`, `@phosphor-icons/react`), and animation runtime (`motion`).
4. **Reference extraction (when the user supplies a URL, screenshot, or Figma
   frame):** Do not guess from adjectives. Systematically extract:
   - **Surface & text tokens:** Canvas background, elevated card/well step,
     4-tier text contrast levels, hairline border opacity, and primary accent.
   - **Typography:** Display vs. body vs. mono font families, heading-to-body
     scale ratio, weight distribution (`400/500/600`), and `letter-spacing`.
   - **Spatial rhythm:** Container max-width, grid column count/gap, card
     padding, and concentric border radii (`outer = inner + padding`).
   - **Signature detail:** The 1–2 craft elements that define the reference's
     character (translate the *system*, never copy brand logos or proprietary
     copy).

**Preserve what exists** unless asked for a redesign. Match the scope to the
request:

- **Component scope** (single card, modal, form, table): inherit the project's
  tokens, font scale, and radius; ship all 8 interactive states (`default`,
  `hover`, `:focus-visible`, `:active`, `disabled`, `loading`, `error`,
  `success`) without inventing page-level hero layouts.
- **Page / app scope** (new landing page, marketing site, or greenfield app):
  commit to a full aesthetic direction, macrostructure, and signature moment.

## 2. Match the surface mode

Different surfaces reward different design postures. Identify the mode first so
you don't apply marketing theatrics to a workbench or dashboard restraint to a
brand launch:

| Mode           | Surfaces                                     | Posture                                                                                                                               |
| -------------- | -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| **Persuade**   | Landing pages, marketing, pricing, launches  | Bold point of view, expressive display + body pairing, fluid `clamp()` headlines, asymmetric macrostructure, one signature moment.    |
| **Operate**    | Dashboards, admin, SaaS workbenches, tables  | **Route to `dashboard-ui`.** Earned familiarity, 1 sans + 1 mono (`tabular-nums`), fixed `rem` scale, weight/color over size, zero page-load choreography. |
| **Read**       | Docs, articles, changelogs, legal            | **Route to `typography`.** 60–70ch measure, generous line-height (`1.6`), clear heading cadence, quiet chrome.                        |
| **Experience** | Portfolios, interactive canvases, showcases  | The artifact or canvas leads; UI chrome recedes to whisper-quiet edges.                                                               |

## 3. Design thinking (Persuade & greenfield apps)

Commit to a BOLD aesthetic direction before coding:

- **Purpose & human:** What problem does this interface solve? Who uses it, and
  what should they notice in the first 2 seconds?
- **Tone:** Pick a clear conceptual anchor — brutally minimal, Swiss grid
  precision, editorial magazine, warm craft/organic, retro-futuristic, industrial
  utilitarian, atmospheric minimalism, or kinetic typography. Draw inspiration
  from real software and print craft (Linear, Raycast, Stripe, Vercel, Teenage
  Engineering, Criterion, Monocle), never generic SaaS templates.
- **Differentiation:** What is the single unforgettable detail — a live
  interactive product preview, a typographic focal point, a tactile surface
  layering, or a choreographed reveal?

**CRITICAL:** Intentional minimalism and maximalism both work. Minimalism
achieves impact through restraint, negative space, and surgical typography;
maximalism achieves impact through rich texture, editorial rhythm, and layered
composition. The enemy is **undecided middle-ground** where every section gets
the same safe card grid.

## 4. Aesthetics & structural guidelines

- **Typography:** Choose distinctive, characterful fonts (`next/font/google` or
  local variable fonts). Pair an expressive display/editorial face with a crisp
  neutral body face, plus a monospace face for metrics, timestamps, and code.
  Never default to Inter, Roboto, Arial, or Open Sans as the sole personality on
  a Persuade surface.
- **Color & theme:** Commit to a cohesive OKLCH palette via CSS custom
  properties (see `design-tokens`). One dominant surface (60–70%), one supporting
  elevated/sunken surface (20–25%), and one sharp accent (5–10% — primary CTA,
  active indicator, key focus). Tint neutrals subtly toward the brand hue
  (`chroma 0.005–0.015`).
- **Macrostructure & spatial composition:**
  - Vary section structure across a page — mix split-screen 50/50 layouts,
    asymmetric 2-column feature showcases, editorial typographic bands, bento
    grids (`2:1` / `1:2` spans), and full-bleed product stages.
  - Group related items with tight spacing first, then separate groups with at
    least `2×` that gap before reaching for borders or cards.
  - Keep a generous 25%+ viewport budget of open background so focal points
    breathe.
- **Authentic product proof over fake chrome:** Show real interactive component
  states, live typography, or actual screenshots framed with a `1px` neutral
  inset outline (`outline-black/10 dark:outline-white/10`) — never re-draw fake
  browser traffic-light bars, fake phone notches, or cartoon IDE frames in HTML.
- **Motion & surface craft:** Follow `ui-polish` for concentric border radii,
  layered shadows vs. dark-mode lightness steps, custom `--ease-out` curves, and
  stagger budgets (max 3 animation primitives per page).

## 5. Explicit anti-slop bans

Never ship these recognizable AI-generated patterns:

- **Template skeleton:** `Centered 100vh Hero → 3 equal icon-cards → Logo strip → CTA banner → 4-column footer`
  with identical vertical padding on every section.
- **Hanging section headers:** Never repeat the `tag on the left / big heading on the right`
  split header or decorative `01 / 02 / 03` eyebrow numbering on non-sequential
  sections.
- **Card-in-card & stripe bloat:** Never nest a bordered card inside a bordered
  card, never put a `3–4px` colored left/top border stripe on cards or alerts,
  and never box 3–4 items in a row with identical icon-in-tinted-circle + bold
  title + 2-line blurb.
- **Color & effect clichés:** No purple/indigo-to-cyan hero gradients, no glowing
  aurora blobs behind every section, no `background-clip: text` gradient
  headlines, and no neon outer `box-shadow` glows in dark mode.
- **Emoji as UI icons:** Never use `✨`, `🚀`, `⚡`, `🔒`, or `🎯` as feature
  or navigation icons — use a single SVG icon family (`currentColor`).
- **Invented metrics & filler copy:** Never fabricate statistics (`"10× faster"`,
  `"99.99% uptime"`, `"Trusted by 50,000+ teams"`), fake testimonials (`"John Doe"`,
  `"Jane Smith"`, `"Acme Corp"`), or AI buzzwords (`"Seamless"`, `"Unleash"`,
  `"Elevate"`, `"Next-Gen"`, `"Supercharge"`). If a real metric isn't provided,
  use a qualitative truth or `—` marked `/* metric to confirm */`.
- **Wrapped clickable text:** Buttons, nav links, pills, and CTAs must never wrap
  to two lines (`white-space: nowrap`).

## Attribution

Synthesized from [Anthropic `skills/frontend-design`](https://github.com/anthropics/skills/tree/main/skills/frontend-design)
(Apache-2.0), [`pbakaus/impeccable`](https://github.com/pbakaus/impeccable)
(Apache-2.0), and [`nexu-io/open-design` (`hallmark`)](https://github.com/nexu-io/open-design)
(MIT).
