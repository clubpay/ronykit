---
name: storybook
description: >-
  Author and maintain Storybook stories with CSF 3.0 best practices — args,
  decorators, parameters, and config. Use when creating or editing *.stories.*
  files, configuring .storybook/, or ensuring every UI component ships with
  stories.
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

### 1. Use CSF 3.0

Prefer Component Story Format 3.0 — concise and type-safe.

```tsx
// ❌ CSF 2.0 (legacy)
export default {
  title: 'Components/Button',
  component: Button,
};
export const Primary = () => <Button variant="primary">Click me</Button>;

// ✅ CSF 3.0 (preferred)
import type { Meta, StoryObj } from '@storybook/react';
import { Button } from './Button';

const meta = {
  component: Button,
  tags: ['autodocs'],
  args: {
    variant: 'primary',
    children: 'Click me',
  },
} satisfies Meta<typeof Button>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Primary: Story = {};

export const Secondary: Story = {
  args: { variant: 'secondary' },
};
```

### 2. Args-based stories

Define component props as args so the Controls panel stays interactive.

- **Declare defaults in `args`** — not `argTypes.defaultValue`. Meta-level
  `args` become the default selection in Controls.
- **Share common args at the Meta level**; override only what differs per
  story.

```tsx
// ❌ Hard-coded props in render
export const Disabled: Story = {
  render: () => <Button disabled>Disabled</Button>,
};

// ❌ Duplicated args across stories
export const Primary: Story = {
  args: { children: 'Click me', variant: 'primary' },
};
export const Secondary: Story = {
  args: { children: 'Click me', variant: 'secondary' },
};

// ✅ Meta-level shared args; stories override only differences
const meta = {
  component: Button,
  args: {
    children: 'Click me',
    variant: 'primary',
  },
} satisfies Meta<typeof Button>;

export const Primary: Story = {};
export const Secondary: Story = { args: { variant: 'secondary' } };
export const Disabled: Story = { args: { disabled: true } };
```

### 3. Omit `title` — infer from file path

Avoid hard-coding `title` strings — they are not type-safe and drift when
components move. Storybook infers the sidebar hierarchy from the file path.

```tsx
// ❌ Manual title — sync breaks on renames/moves
const meta = {
  title: 'Components/Button',
  component: Button,
} satisfies Meta<typeof Button>;

// ✅ Omit title — path-based inference
const meta = {
  component: Button,
} satisfies Meta<typeof Button>;
```

### 4. Type-safe Meta with `satisfies`

Use `satisfies Meta<typeof Component>` for both type checking and inference.

```tsx
// ❌ Loses inference
const meta: Meta<typeof Button> = { component: Button };

// ✅ Check and infer
const meta = {
  component: Button,
  args: { size: 'md', variant: 'primary' },
} satisfies Meta<typeof Button>;

export default meta;
type Story = StoryObj<typeof meta>;
```

### 5. Decorators for context

Wrap stories with providers or layout via decorators.

```tsx
// Per-story decorator
export const WithTheme: Story = {
  decorators: [
    (Story) => (
      <ThemeProvider theme="dark">
        <Story />
      </ThemeProvider>
    ),
  ],
};

// All stories in the file
const meta = {
  component: Button,
  decorators: [
    (Story) => (
      <div style={{ padding: '3rem' }}>
        <Story />
      </div>
    ),
  ],
} satisfies Meta<typeof Button>;
```

### 6. Parameters for behavior

```tsx
const meta = {
  component: Button,
  parameters: {
    layout: 'centered',
    backgrounds: {
      default: 'light',
      values: [
        { name: 'light', value: '#ffffff' },
        { name: 'dark', value: '#000000' },
      ],
    },
  },
} satisfies Meta<typeof Button>;

export const OnDark: Story = {
  parameters: { backgrounds: { default: 'dark' } },
};
```

### 7. ArgTypes — prefer auto-inference

Storybook infers argTypes from TypeScript prop types. Manual overrides drift
when props change — only specify argTypes when auto-inference is inadequate.

**Valid reasons to override:**

- `ReactNode` prop that should be edited as text in Controls → `control: 'text'`
- Compound component pattern (multiple exports) → explicit argTypes
- Prop fixed for a specific story → `control: false`

```tsx
// ❌ Unnecessary manual argTypes
const meta = {
  component: Button,
  argTypes: {
    variant: { control: 'select', options: ['primary', 'secondary'] },
    disabled: { control: 'boolean' },
  },
} satisfies Meta<typeof Button>;

// ✅ Let inference work; override only when needed
const meta = {
  component: Button,
  argTypes: {
    children: { control: 'text' },
  },
} satisfies Meta<typeof Button>;

export const Horizontal: Story = {
  args: { orientation: 'horizontal' },
  argTypes: { orientation: { control: false } },
};
```

## Recommended story structure

```tsx
import type { Meta, StoryObj } from '@storybook/react';
import { Button } from './Button';

const meta = {
  component: Button,
  tags: ['autodocs'],
  parameters: { layout: 'centered' },
  args: {
    children: 'Button',
    size: 'md',
    variant: 'primary',
  },
} satisfies Meta<typeof Button>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Primary: Story = {};
export const Secondary: Story = { args: { variant: 'secondary' } };
export const Disabled: Story = { args: { disabled: true } };

export const WithCustomTheme: Story = {
  decorators: [
    (Story) => (
      <ThemeProvider theme="custom">
        <Story />
      </ThemeProvider>
    ),
  ],
};
```

## ArgTypes reference (when manual override is needed)

```tsx
argTypes: {
  children: { control: 'text' },
  opacity: {
    control: { type: 'range', min: 0, max: 1, step: 0.1 },
  },
  onClick: { action: 'clicked' },
  orientation: { control: false },
}
```

## Common parameters

```tsx
parameters: {
  layout: 'centered' | 'fullscreen' | 'padded',
  backgrounds: {
    default: 'light',
    values: [
      { name: 'light', value: '#ffffff' },
      { name: 'dark', value: '#333333' },
    ],
  },
  actions: { argTypesRegex: '^on[A-Z].*' },
  docs: {
    description: { component: 'Button component description' },
  },
}
```

## Decorator patterns

```tsx
// Style wrapper
(Story) => <div style={{ padding: '3rem' }}><Story /></div>

// Theme provider
(Story) => <ThemeProvider theme="dark"><Story /></ThemeProvider>

// React Router
(Story) => <MemoryRouter initialEntries={['/']}><Story /></MemoryRouter>

// i18n
(Story) => <I18nProvider locale="en"><Story /></I18nProvider>

// Global state
(Story) => <Provider store={mockStore}><Story /></Provider>
```

## File naming

```
Component.tsx           # component implementation
Component.stories.tsx   # story file (same directory)
Component.test.tsx      # test file
```

Co-locate: `Button.tsx` → `Button.stories.tsx`.

## Storybook configuration

```typescript
// .storybook/main.ts
import type { StorybookConfig } from '@storybook/react-vite';

const config: StorybookConfig = {
  stories: ['../src/**/*.stories.@(ts|tsx)'],
  addons: [
    '@storybook/addon-essentials',
    '@storybook/addon-interactions',
  ],
  framework: {
    name: '@storybook/react-vite',
    options: {},
  },
};

export default config;
```

```typescript
// .storybook/preview.ts
import type { Preview } from '@storybook/react';

const preview: Preview = {
  parameters: {
    actions: { argTypesRegex: '^on[A-Z].*' },
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/,
      },
    },
  },
  decorators: [
    (Story) => (
      <div style={{ fontFamily: 'Arial, sans-serif' }}>
        <Story />
      </div>
    ),
  ],
};

export default preview;
```

## Quality bar

- **Interactions:** use `play` functions for stateful flows; assert with
  `@storybook/test` where behavior matters.
- **Accessibility:** keep the a11y addon clean — fix violations, don't disable.
- **Isolation:** stories render without app context; provide providers/mocks via
  decorators, not real network calls.
- **No dead stories:** remove stories for deleted components/variants.

## Commands

```bash
pnpm storybook          # run Storybook locally
pnpm build-storybook    # static build (CI)
pnpm test-storybook     # interaction/a11y tests (if configured)
```

## Checklist before "done"

- Every new/changed component has a co-located, up-to-date `*.stories.tsx`.
- Stories use CSF 3.0 with shared Meta `args` and path-based titles.
- Storybook builds and a11y/interaction checks pass.

---

_Adapted from the `storybook` skill in
[DaleStudy/skills](https://github.com/dalestudy/skills) (MIT)._
