---
name: design-tokens
description: >-
  Architect CSS custom-property design token systems (3-tier primitive →
  semantic → component tokens) using OKLCH color ramps, 4-tier text hierarchy,
  dedicated control tokens, spacing/radius/elevation scales, and first-class
  light/dark theming in Tailwind CSS v3/v4 and shadcn/ui. Use when setting up
  `globals.css`, defining a palette, adding dark mode, or eliminating hardcoded
  hex/px values.
---

# Design Tokens & Theming Architecture

Hardcoded hex codes (`#3b82f6`), arbitrary pixel spacing, and ad-hoc shadows make
interfaces brittle and dark mode impossible to maintain. A **3-tier token
architecture** gives every component a single semantic source of truth.

## 1. Three-tier token hierarchy

| Tier              | Purpose                                | Example                                                     | Used in components?                |
| ----------------- | -------------------------------------- | ----------------------------------------------------------- | ---------------------------------- |
| **1. Primitive**  | Raw palette ramps & numeric scales     | `--neutral-100`, `--brand-600`, `--space-4`                 | **Never directly**                 |
| **2. Semantic**   | Role-based tokens that flip per theme  | `--background`, `--foreground-secondary`, `--border`, `--primary` | **Yes — 95% of styling**     |
| **3. Component**  | Scoped overrides when a widget needs it| `--sidebar-bg`, `--input-bg`, `--chart-1`                   | Only inside that component         |

**Rule:** Never borrow a token across roles (for example, never use a border
token for text color or a card surface token for an input background) — even if
they share the same value in light mode, they diverge in dark mode.

## 2. OKLCH color ramps & hue discipline

Prefer **`oklch(L C H)`** over hex or HSL: equal lightness `L` steps look equally
bright to the human eye across every hue.

- **Keep the palette minimal:** One neutral ramp, one primary accent ramp, and
  only the semantic status ramps (`success`, `warning`, `destructive`, `info`)
  your UI actually renders.
- **Constant hue per ramp:** Hold hue `H` constant across a ramp (`50 → 950`) and
  tighten lightness steps at the light end (`50, 100, 200`) where subtle surface
  differences matter most. When adjusting contrast, change **`L`**, not `H`.
- **Tint your neutrals:** Add a subtle hint of the brand hue (`C = 0.005–0.015`)
  to neutral surfaces, text, and borders so the UI feels cohesive rather than
  sterile `#000`/`#fff`.
- **Distinct status hues:** Keep `success`, `warning`, and `destructive` hues at
  least **`15°` away** from your primary accent hue so users never confuse a
  brand badge with an error or warning.
- **Paired status tokens:** Every status role needs **3 tokens** — subtle
  surface (`--status-*-subtle`), readable foreground (`--status-*-fg` at ≥ 4.5:1
  contrast on that surface), and hairline border (`--status-*-border`).

## 3. Four-tier text hierarchy & dedicated control tokens

Two text tokens (`foreground` and `muted-foreground`) are rarely enough for rich
data UIs. Define **4 text levels** plus **dedicated control tokens**:

```css
:root {
  color-scheme: light;

  /* Surfaces (60 / 30 / 10 distribution) */
  --background: oklch(0.985 0.004 255);          /* Base canvas */
  --surface-sunken: oklch(0.962 0.006 255);      /* Wells, code blocks, table headers */
  --card: oklch(1 0 0);                          /* Elevated cards */
  --popover: oklch(1 0 0);                       /* Dropdowns, dialogs */

  /* 4-tier text hierarchy */
  --foreground: oklch(0.145 0.012 255);          /* Tier 1: headings, primary values */
  --foreground-secondary: oklch(0.32 0.012 255); /* Tier 2: body copy, table cells */
  --muted-foreground: oklch(0.48 0.012 255);     /* Tier 3: labels, column headers, timestamps */
  --foreground-subtle: oklch(0.62 0.010 255);    /* Tier 4: placeholders, disabled, empty dashes */

  /* Dedicated interactive & control tokens */
  --primary: oklch(0.50 0.19 262);
  --primary-foreground: oklch(0.99 0.005 262);
  --input-bg: oklch(1 0 0);
  --input-border: oklch(0.80 0.010 255);         /* ≥ 3:1 contrast for input boundaries */
  --border: oklch(0.90 0.008 255);               /* Structural hairline dividers */
  --ring: oklch(0.55 0.18 262);

  /* Concentric radius & motion tokens */
  --radius-sm: 0.375rem;  /* 6px — inner badges, inputs */
  --radius-md: 0.5rem;    /* 8px — buttons, menu items */
  --radius-lg: 0.75rem;   /* 12px — cards (inner 8px + 4px/8px pad) */
  --radius-xl: 1rem;      /* 16px — modals, panels */
  --ease-out: cubic-bezier(0.23, 1, 0.32, 1);
  --ease-in-out: cubic-bezier(0.77, 0, 0.175, 1);
}

.dark {
  color-scheme: dark;

  /* Dark mode elevates via lightness steps (+7% → +9% → +12%), not drop shadows */
  --background: oklch(0.135 0.010 255);          /* Base canvas */
  --surface-sunken: oklch(0.11 0.010 255);       /* Recessed wells & inset inputs */
  --card: oklch(0.175 0.012 255);                /* Step 1 elevation */
  --popover: oklch(0.21 0.014 255);              /* Step 2 elevation (menus/modals) */

  /* Softened off-white text & desaturated accents to prevent halation */
  --foreground: oklch(0.94 0.006 255);
  --foreground-secondary: oklch(0.78 0.008 255);
  --muted-foreground: oklch(0.64 0.010 255);
  --foreground-subtle: oklch(0.46 0.008 255);

  /* Lower chroma slightly on dark surfaces so accents don't vibrate */
  --primary: oklch(0.66 0.16 262);
  --primary-foreground: oklch(0.13 0.015 262);
  --input-bg: oklch(0.12 0.010 255);             /* Inset darker than card */
  --input-border: oklch(1 0 0 / 16%);
  --border: oklch(1 0 0 / 9%);                   /* Translucent white hairline */
  --ring: oklch(0.68 0.16 262);
}
```

## 4. Dark-mode engineering rules

- **Never invert the light ramp mechanically:** Dark mode uses **surface
  lightness steps** (`--background < --card < --popover`) + translucent white
  hairlines (`oklch(1 0 0 / 8–14%)`) instead of dark drop shadows.
- **Inputs are inset in dark mode:** Set `--input-bg` slightly *darker* than
  `--card` so text fields read as recessed wells.
- **Lower chroma on dark:** Reduce accent and status saturation (`C`) by ~10–15%
  in `.dark` so colors don't vibrate against dark surfaces.
- **Set `color-scheme: dark`** on `.dark` / `<html>` so native scrollbars, date
  pickers, and `<select>` menus match the active theme, and style `::selection`
  + `meta[name="theme-color"]` from your tokens.
- **Single switching mechanism:** Drive dark mode with **one** strategy across
  the whole app (typically `.dark` class via `next-themes` with
  `disableTransitionOnChange`), never a mix of `.dark` classes and raw
  `@media (prefers-color-scheme: dark)` overrides that fight each other.

## 5. Tailwind v4 (`@theme inline`) vs. v3 bridge

In **Tailwind v4**, expose semantic tokens in `globals.css`:

```css
@import "tailwindcss";

@theme inline {
  --color-background: var(--background);
  --color-surface-sunken: var(--surface-sunken);
  --color-card: var(--card);
  --color-popover: var(--popover);
  --color-foreground: var(--foreground);
  --color-foreground-secondary: var(--foreground-secondary);
  --color-muted-foreground: var(--muted-foreground);
  --color-primary: var(--primary);
  --color-primary-foreground: var(--primary-foreground);
  --color-border: var(--border);
  --color-input: var(--input-border);
  --color-ring: var(--ring);
  --radius-sm: var(--radius-sm);
  --radius-md: var(--radius-md);
  --radius-lg: var(--radius-lg);
  --radius-xl: var(--radius-xl);
}
```

## Token audit checklist

- [ ] Zero raw `#hex`, `rgb()`, or Tailwind palette literals (`bg-slate-800`) inside feature components
- [ ] Every status role has paired `-subtle`, `-fg` (≥ 4.5:1), and `-border` tokens at ≥ `15°` hue separation from `--primary`
- [ ] 4-tier text tokens and dedicated `--input-bg` / `--input-border` tokens defined
- [ ] Dark mode uses surface lightness steps, translucent borders, `color-scheme: dark`, and `disableTransitionOnChange`
- [ ] Nested border radii follow concentric math (`outer = inner + padding`)

## Attribution

Synthesized from [`jakubkrehel/skills`](https://github.com/jakubkrehel/skills)
(`better-colors`, MIT) and [`Dammyjay93/interface-design`](https://github.com/Dammyjay93/interface-design)
(MIT).
