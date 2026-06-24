---
name: shadcn
description: >-
  Build UI with shadcn/ui — find, install, compose, and customize components
  using the correct CLI and patterns. Use when adding or styling frontend
  components, building forms/dashboards, theming, or working with components.json
  and the shadcn registry/MCP.
---

# shadcn/ui

shadcn/ui components are copied into your project (not a dependency) and
composed from primitives. Always discover the real component API before
generating code — don't guess prop names.

## When to use

- Adding or composing UI components, forms, dialogs, tables, dashboards.
- Theming (CSS variables, dark mode, radius, colors) or adding from a registry.
- Reviewing frontend work that uses shadcn/ui.

## Get the live, project-aware skill (recommended)

The official skill reads your project and injects accurate context. Install it
and enable the shadcn MCP so the assistant uses real component data:

```bash
npx skills add shadcn/ui          # installs the official shadcn skill
```

When present, it activates on a `components.json` and runs `shadcn info --json`
to learn your framework, Tailwind version, aliases, base library (`radix` or
`base`), icon library, and installed components. Prefer this over memory.

## CLI workflow

Use the CLI (`npx shadcn@latest …`) — never hand-copy component source:

```bash
npx shadcn@latest init             # set up components.json + base styles
npx shadcn@latest search <query>   # find components in the registry
npx shadcn@latest view <name>      # inspect a component's files/API
npx shadcn@latest docs <name>      # read its documentation
npx shadcn@latest add <name>       # install a component (and its deps)
npx shadcn@latest info --json      # current project configuration
npx shadcn@latest diff             # see upstream changes to installed components
```

Discover with `search`/`view`/`docs` (or the MCP tools) **before** writing code,
so you use the correct, version-specific API on the first try.

## Composition rules

- Compose from shadcn primitives instead of raw HTML where one exists.
- **Forms:** use `Field`/`FieldGroup` (and the form primitives) for layout and
  labelling, not ad-hoc `div`s.
- **Option sets:** `ToggleGroup`/`RadioGroup` rather than hand-rolled buttons.
- Respect the base library's API (`radix` vs `base`) — they differ; check first.
- Keep generated component files editable in your repo; customize there.

## Theming

- Style via the CSS variables shadcn generates (semantic tokens like
  `bg-background`, `text-foreground`, `border`, `ring`) — avoid hard-coded hex.
- Colors use OKLCH; respect the Tailwind version (v3 vs v4) the project uses.
- Dark mode through the `.dark` class / theme tokens, not duplicated styles.
- Adjust radius/colors via the theme variables, not per-component overrides.

## Checklist

- Component API confirmed via `view`/`docs`/MCP before coding.
- Components installed with `add` (deps resolved), not pasted by hand.
- Semantic color tokens used; no stray hex values.
- Forms/option-sets use the proper shadcn composition primitives.

## Learn more

- CLI: <https://ui.shadcn.com/docs/cli>
- MCP server: <https://ui.shadcn.com/docs/mcp>
- Theming: <https://ui.shadcn.com/docs/theming>
- Registry: <https://ui.shadcn.com/docs/registry>
