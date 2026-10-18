---
name: storybook
description: >-
  Author and maintain Storybook stories with CSF 3.0 best practices — args,
  decorators, parameters, token/theme wiring, 8-state coverage, and a11y
  verification. Use when creating or editing *.stories.* files, configuring
  .storybook/, or ensuring every UI component ships with stories.
---

# Storybook

Every reusable UI component must have a Storybook story. A component without a
story is not done. Stories are living documentation and the contract for a
component's states.

## When to use

- Creating a new component, or adding a variant/prop to an existing one.
- Writing or editing `*.stories.tsx` / `*.stories.ts` files.
- Configuring args, decorators, parameters, or `.storybook/` setup.
- Reviewing a frontend PR — a new component without a co-located story is a
  blocking gap.

## The rule

> New or changed component → add or update its story in the same change.

This is **automatic and unprompted**: create the story as part of building the
component, without waiting for the user to ask. After authoring or updating
stories, run `pnpm build-storybook` and fix any failures yourself before
reporting done.

## Best practices

### 1. Use CSF 3.0 with `satisfies Meta<typeof Component>`

Prefer Component Story Format 3.0 with `satisfies` for full type checking and
inference, and **omit `title`** so Storybook infers the sidebar hierarchy from
the file path.

```tsx
import type { Meta, StoryObj } from '@storybook/react';
import { Button } from './Button';

const meta = {
  component: Button,
  tags: ['autodocs'],
  parameters: { layout: 'centered' },
  args: {
    variant: 'primary',
    size: 'md',
    children: 'Save changes',
  },
} satisfies Meta<typeof Button>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Primary: Story = {};
export const Secondary: Story = { args: { variant: 'secondary' } };
export const Disabled: Story = { args: { disabled: true } };
```

### 2. Args-based stories & minimal `argTypes`

- **Declare defaults in Meta `args`** — stories override only what differs.
- **Let TypeScript infer `argTypes`:** Only specify `argTypes` when
  auto-inference is inadequate (`ReactNode` edited as `control: 'text'`, or
  locking a prop with `control: false`).

### 3. Cover interactive states, dark mode & layout resilience

A single happy-path story is not enough. Cover the states that apply to the
component (see `ux-quality` and `ui-polish`):

- **Variants & states:** `Default`, `Secondary`/`Destructive`, `Disabled`,
  `Loading`/`Pending`, `Empty` (first-use vs. filtered), and `Error` (`aria-invalid`).
- **Dark mode:** Verify the component on both `:root` (light) and `.dark`
  surfaces so token contrast and surface lightness steps hold up.
- **Text-growth & truncation stress story:** Include a story with long
  localized strings (+40% length) or unbreakable IDs/URLs to prove `min-w-0`,
  `truncate`, and `nowrap` buttons never blow out the layout.

```tsx
export const LongContentStress: Story = {
  args: {
    children: ' Einstellungen speichern und Workspace синхронизировать',
  },
};

export const DarkSurface: Story = {
  globals: { theme: 'dark' },
};
```

### 4. Wire `.storybook/preview.ts` to your real design tokens & fonts

Never hardcode `fontFamily: 'Arial'` or arbitrary `#ffffff` / `#333333`
backgrounds in `.storybook/preview.ts`. Import your app's `globals.css` so
Storybook renders the exact OKLCH tokens, `--font-sans` / `--font-mono`
variables, and `.dark` class used in production:

```tsx
// .storybook/preview.tsx
import type { Preview } from '@storybook/react';
import '../src/app/globals.css'; // real design tokens & typography baseline

const preview: Preview = {
  parameters: {
    actions: { argTypesRegex: '^on[A-Z].*' },
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    a11y: {
      // Fail or surface WCAG AA violations in the a11y addon
      test: 'error',
    },
  },
  globalTypes: {
    theme: {
      description: 'Global theme for components',
      toolbar: {
        title: 'Theme',
        icon: 'mirror',
        items: ['light', 'dark'],
        dynamicTitle: true,
      },
    },
  },
  decorators: [
    (Story, context) => {
      const isDark = context.globals.theme === 'dark';
      return (
        <div
          className={
            isDark
              ? 'dark bg-background text-foreground p-6 font-sans antialiased'
              : 'bg-background text-foreground p-6 font-sans antialiased'
          }
        >
          <Story />
        </div>
      );
    },
  ],
};

export default preview;
```

### 5. Storybook configuration (`main.ts`)

Include `@storybook/addon-a11y` alongside essentials and interactions:

```ts
// .storybook/main.ts
import type { StorybookConfig } from '@storybook/nextjs'; // or '@storybook/react-vite'

const config: StorybookConfig = {
  stories: ['../src/**/*.stories.@(ts|tsx)'],
  addons: [
    '@storybook/addon-essentials',
    '@storybook/addon-interactions',
    '@storybook/addon-a11y',
  ],
  framework: {
    name: '@storybook/nextjs',
    options: {},
  },
};

export default config;
```

### 6. Interaction (`play`) & accessibility checks

For stateful components (dialogs, dropdowns, forms, tabs), add a `play`
function using `@storybook/test` (`userEvent`, `within`, `expect`) to verify
keyboard/pointer flows and ARIA states:

```tsx
import { expect, userEvent, within } from '@storybook/test';

export const OpensAndValidates: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(canvas.getByRole('button', { name: /create key/i }));
    await expect(canvas.getByRole('alert')).toBeVisible();
  },
};
```

## File naming

```
Component.tsx           # component implementation
Component.stories.tsx   # story file (same directory)
Component.test.tsx      # test file
```

## Quality bar & commands

```bash
pnpm storybook          # run Storybook locally
pnpm build-storybook    # static build (enforced by frontend/verify.sh)
pnpm test-storybook     # interaction/a11y tests (if configured)
```

## Checklist before "done"

- Every new/changed component has a co-located, up-to-date `*.stories.tsx`.
- Stories use CSF 3.0 with `satisfies Meta<typeof Component>`, shared Meta
  `args`, and path-based titles.
- `.storybook/preview.tsx` imports `globals.css` tokens and supports light/dark
  switching (no hardcoded `Arial` or `#fff`/`#333`).
- States (`Default`, `Disabled`, `Loading`, `Error`, `DarkSurface`, long text)
  are represented and `pnpm build-storybook` passes.

---

_Adapted from the `storybook` skill in
[DaleStudy/skills](https://github.com/dalestudy/skills) (MIT)._
