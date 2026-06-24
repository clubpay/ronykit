# shadcn/ui Theming & Dark Mode

Deeper theming patterns for when the live shadcn MCP isn't available. **First
choice is always `shadcn info --json` + the MCP** — it reports the project's
Tailwind version and token setup, which decides which variant below applies.
For token *architecture* (primitive→semantic→component) see `design-tokens`.

## Know your Tailwind version first — the patterns differ

shadcn generates different token plumbing per Tailwind major. Detect it
(`shadcn info --json`, or check `globals.css`) before editing.

- **Tailwind v4 (current):** tokens live in `globals.css` via `@theme inline`,
  colors are **OKLCH**, and there is **no `tailwind.config.ts`** color block.
  Customize by editing the `:root` / `.dark` variables and the `@theme` map.
- **Tailwind v3 (legacy):** tokens are **HSL channels** in `:root`/`.dark` and
  mapped in `tailwind.config.ts` under `theme.extend.colors`.

Don't paste a v3 `tailwind.config.ts` color block into a v4 project (or vice
versa) — match what the project already has.

## Dark mode (Next.js App Router, `next-themes`)

```bash
npm install next-themes
```

```tsx
// app/layout.tsx
import { ThemeProvider } from "next-themes"

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body>
        <ThemeProvider attribute="class" defaultTheme="system" enableSystem
          disableTransitionOnChange>
          {children}
        </ThemeProvider>
      </body>
    </html>
  )
}
```

```tsx
// theme toggle
"use client"
import { Moon, Sun } from "lucide-react"
import { useTheme } from "next-themes"
import { Button } from "@/components/ui/button"

export function ThemeToggle() {
  const { setTheme, theme } = useTheme()
  return (
    <Button variant="ghost" size="icon"
      onClick={() => setTheme(theme === "light" ? "dark" : "light")}>
      <Sun className="h-5 w-5 scale-100 dark:scale-0 transition-transform" />
      <Moon className="absolute h-5 w-5 scale-0 dark:scale-100 transition-transform" />
      <span className="sr-only">Toggle theme</span>
    </Button>
  )
}
```

`suppressHydrationWarning` on `<html>` is required — the theme class is set
before React hydrates. Without `next-themes`, toggle `documentElement.classList`
and persist to `localStorage`, reading `prefers-color-scheme` as the default.

## CSS variable system

Always pair each color with its `*-foreground`, and store **channels** (not a
full `hsl(...)`/`oklch(...)`) so `/opacity` modifiers compose (`bg-primary/50`).

```css
/* Tailwind v4 — globals.css */
:root {
  --background: oklch(1 0 0);
  --foreground: oklch(0.145 0 0);
  --primary: oklch(0.205 0 0);
  --primary-foreground: oklch(0.985 0 0);
  --radius: 0.625rem;
}
.dark {
  --background: oklch(0.145 0 0);
  --foreground: oklch(0.985 0 0);
  --primary: oklch(0.985 0 0);
  --primary-foreground: oklch(0.205 0 0);
}
@theme inline {
  --color-background: var(--background);
  --color-primary: var(--color-primary, var(--primary));
}
```

```css
/* Tailwind v3 — globals.css (HSL channels) + tailwind.config.ts mapping */
:root { --primary: 222.2 47.4% 11.2%; --primary-foreground: 210 40% 98%; --radius: 0.5rem; }
.dark { --primary: 210 40% 98%; --primary-foreground: 222.2 47.4% 11.2%; }
```

```ts
// v3 only
colors: { primary: { DEFAULT: "hsl(var(--primary))", foreground: "hsl(var(--primary-foreground))" } }
```

## Customizing colors

1. **Edit the variables** in `globals.css` (the source of truth) — not
   per-component hex. Re-point `:root` *and* `.dark` together.
2. **Theme generator:** <https://ui.shadcn.com/themes> — pick a base, copy the
   variables it emits in your Tailwind version's format.
3. **Multiple themes:** scope overrides under a data attribute.

```css
[data-theme="violet"] { --primary: oklch(0.55 0.24 292); --primary-foreground: oklch(0.99 0 0); }
```

## Customizing components (they're in your repo)

Components are copied into `components/ui/*` — edit them directly. Add variants
through `cva` rather than scattering `className` overrides:

```tsx
// components/ui/button.tsx
const buttonVariants = cva("inline-flex items-center justify-center rounded-md …", {
  variants: {
    variant: {
      default: "bg-primary text-primary-foreground hover:bg-primary/90",
      // new variant — reuse semantic tokens, don't hard-code hex
      brand: "bg-brand text-brand-foreground hover:bg-brand/90",
    },
    size: { default: "h-10 px-4 py-2", xl: "h-14 px-10 text-lg" },
  },
  defaultVariants: { variant: "default", size: "default" },
})
```

For a one-off, pass `className` (merged via `cn`); for a repeated change, add a
`cva` variant so it stays consistent and themeable.

## Radius, base color, style

- `--radius` controls rounding globally; component `rounded-*` derive from it.
- `init` picks a base gray (slate/zinc/neutral/stone) and a style (`default` vs
  `new-york`); both are recorded in `components.json`.

## Checklist

- Confirmed Tailwind major (MCP/`info --json`); used the matching token format.
- Colors edited as variables in `globals.css`, paired with `*-foreground`,
  stored as channels for `/opacity`.
- Dark mode via `next-themes` (or class toggle) with `suppressHydrationWarning`;
  both themes tested.
- New variants added through `cva` with semantic tokens, no stray hex.

## Attribution

Patterns adapted (and updated for Tailwind v4) from the `ui-styling` skill of
[`ui-ux-pro-max`](https://github.com/nextlevelbuilder/ui-ux-pro-max-skill),
Apache-2.0 License.
