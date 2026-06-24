---
name: design-tokens
description: >-
  Architect a scalable, themeable design-token system — primitive → semantic →
  component layers, CSS variables, a 4px spacing/type scale, and dark mode. Use
  when setting up or refactoring a project's tokens, naming color/spacing/radius
  values, wiring tokens into Tailwind/shadcn, enabling theme switching, or
  killing hard-coded hex/px scattered through components.
---

# Design Tokens

Tokens are the single source of truth for every visual value. The goal is that
a component never contains a raw `#2563EB` or `16px` — it references a named
token, so a theme change or rebrand is one edit, not a find-and-replace across
the codebase.

The discipline is **three layers**: raw values, what they *mean*, and how a
*component* consumes them. Skipping the middle layer is why most token systems
can't switch themes.

## When to use

- Standing up a new design system, or refactoring flat/ad-hoc tokens.
- Naming and organizing color, spacing, radius, shadow, type, motion values.
- Wiring tokens into Tailwind / shadcn, or enabling light/dark theming.
- Reviewing UI that hard-codes hex/px instead of referencing tokens.

For *consuming* shadcn/ui's generated variables and CLI, see `shadcn`; for
color discipline in data UIs, see `dashboard-ui`; for type scales and
hierarchy, see `typography`. This skill governs the token *architecture* those
build on.

## The three layers

```
Component   --button-bg, --card-padding      per-component overrides
   ↑
Semantic    --color-primary, --spacing-section   purpose-based aliases
   ↑
Primitive   --color-blue-600, --space-4          raw, meaningless values
```

| Layer     | Holds                          | Changes when…        |
| --------- | ------------------------------ | -------------------- |
| Primitive | Raw scales (the palette/ramp)  | Rarely — foundational|
| Semantic  | Meaning (primary, background)  | Theme / brand switch |
| Component | One component's knobs          | A component needs it |

**Rule:** primitives never appear in components, and semantics never hold raw
values. A component reads a component or semantic token; a semantic token reads
a primitive; a primitive is the only place a literal lives.

### 1. Primitive — raw values, no meaning

Full numeric ramps so the semantic layer always has a rung to point at. Use a
**4px-based spacing scale** and a consistent type/radius/shadow scale.

```css
:root {
  --color-gray-100: #f3f4f6;  --color-gray-900: #111827;
  --color-blue-600: #2563eb;  --color-blue-700: #1d4ed8;
  --color-red-600:  #dc2626;  --color-green-600: #16a34a;

  --space-1: 0.25rem;  --space-2: 0.5rem;  --space-4: 1rem;  --space-6: 1.5rem;
  --font-size-sm: 0.875rem; --font-size-base: 1rem; --font-size-lg: 1.125rem;
  --radius-sm: 0.25rem; --radius-lg: 0.5rem; --radius-full: 9999px;
  --duration-fast: 150ms; --duration-normal: 200ms;
}
```

### 2. Semantic — purpose-based aliases

The layer that makes theming possible. Names describe *role*, not appearance
(`--color-primary`, not `--color-blue`).

```css
:root {
  --color-background: var(--color-gray-50);
  --color-foreground: var(--color-gray-900);
  --color-primary: var(--color-blue-600);
  --color-primary-hover: var(--color-blue-700);
  --color-muted-foreground: var(--color-gray-500);
  --color-destructive: var(--color-red-600);
  --spacing-section: var(--space-6);
}
```

### 3. Component — one component's knobs

Only when a component needs a value that isn't a plain semantic read. Keeps
per-component tweaks out of the semantic layer.

```css
:root {
  --button-bg: var(--color-primary);
  --button-hover-bg: var(--color-primary-hover);
  --button-radius: var(--radius-lg);
  --card-padding: var(--space-4);
  --input-focus-ring: var(--color-primary);
}
```

## Naming convention

```
--{category}-{role}-{variant}-{state}
--color-primary            --color-primary-hover
--button-bg-hover          --space-section-sm
```

Categories: `color`, `space`, `font-size`, `radius`, `shadow`, `duration`,
`z` (layered scale: dropdown/sticky/modal/popover/tooltip).

## Dark mode — override semantics, never primitives

Dark mode re-points the *semantic* layer; primitives and components are
untouched. Use desaturated/lighter tonal variants, not literal inversion, and
verify contrast independently from light mode.

```css
.dark {
  --color-background: var(--color-gray-900);
  --color-foreground: var(--color-gray-50);
  --color-muted-foreground: var(--color-gray-400);
}
```

## Tailwind / shadcn integration

shadcn and Tailwind expect the **same** structure — define semantics as
`hsl()` channel triples so opacity modifiers (`bg-primary/50`) work, then map
them in the theme. This stays compatible with `npx shadcn@latest add`.

```css
:root { --primary: 217 91% 60%; --primary-foreground: 0 0% 100%; --radius: 0.5rem; }
.dark { --primary: 217 91% 60%; }
```

```ts
// tailwind.config.ts
colors: {
  primary: { DEFAULT: "hsl(var(--primary))", foreground: "hsl(var(--primary-foreground))" },
}
```

Store channels (`217 91% 60%`), not full `hsl(...)`, so `/opacity` modifiers
compose. See `shadcn` for the consumption side.

## Migration from flat tokens

Pull every literal up into a primitive, alias its meaning, and point the
component at the alias:

```css
/* before */ --button-primary-bg: #2563eb;
/* after  */ --color-blue-600: #2563eb;          /* primitive */
            --color-primary: var(--color-blue-600); /* semantic */
            --button-bg: var(--color-primary);    /* component */
```

For tool interchange (Figma, Style Dictionary), the W3C DTCG JSON format uses
`{ "$value": "#2563EB", "$type": "color" }` — keep one source and generate CSS
from it rather than maintaining two by hand.

## Common mistakes to avoid

1. Components referencing primitives directly (`var(--color-blue-600)`) — breaks
   theming; point them at a semantic token.
2. Semantic tokens holding raw hex/px instead of a `var()` to a primitive.
3. Dark mode overriding primitives (or re-inventing colors) instead of
   re-pointing semantics.
4. Color names that describe appearance (`--color-blue`) rather than role
   (`--color-primary`) — renames cascade everywhere on a rebrand.
5. Full `hsl(...)` strings where Tailwind/shadcn expect bare channels, killing
   `/opacity` modifiers.
6. Two sources of truth (a JSON file *and* hand-edited CSS) drifting apart.

## Checklist before "done"

- Three layers present; components read component/semantic tokens only,
  primitives hold the only literals.
- Spacing on a 4px scale; type/radius/shadow/duration each a consistent scale.
- Semantic names describe role, not appearance.
- Dark mode overrides semantics only; contrast verified separately.
- Tailwind/shadcn map uses `hsl(var(--token))` over bare channels; no stray hex
  in components.
- One source of truth (CSS or DTCG JSON), not both.

## Attribution

Adapted (web-focused, distilled) from the `design-system` token references of
[`ui-ux-pro-max`](https://github.com/nextlevelbuilder/ui-ux-pro-max-skill),
MIT License. The three-layer model aligns with the W3C Design Tokens Community
Group format.
