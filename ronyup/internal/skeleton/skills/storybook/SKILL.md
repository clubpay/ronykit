---
name: storybook
description: >-
  Ensure every UI component ships with a Storybook story. Use when adding,
  changing, or reviewing frontend components — author stories alongside the
  component, cover its states and variants, and keep stories in sync.
---

# Component Storybook

Every reusable UI component must have a Storybook story. A component without a
story is not done. Stories are living documentation and the contract for a
component's states.

## When to use

- Creating a new component, or adding a variant/prop to an existing one.
- Reviewing a frontend PR — a new component without a `*.stories.tsx` is a
  blocking gap.
- Setting up Storybook in a frontend app that doesn't have it yet.

## The rule

> New or changed component → add or update its story in the same change.

- Co-locate the story with the component: `Button.tsx` → `Button.stories.tsx`.
- Cover the meaningful states: default, loading, disabled, error, empty, and
  each significant variant/size.
- Drive variants with `args`/`argTypes` so they're interactive in the Controls
  panel — don't hard-code one-off props.

## Authoring stories (CSF 3)

```tsx
import type { Meta, StoryObj } from '@storybook/react'
import { Button } from './Button'

const meta: Meta<typeof Button> = {
  title: 'Components/Button',
  component: Button,
  args: { children: 'Click me' },
  argTypes: { variant: { control: 'select', options: ['primary', 'secondary'] } },
}
export default meta

type Story = StoryObj<typeof Button>

export const Primary: Story = { args: { variant: 'primary' } }
export const Secondary: Story = { args: { variant: 'secondary' } }
export const Loading: Story = { args: { loading: true } }
export const Disabled: Story = { args: { disabled: true } }
```

## Quality bar for stories

- **Title reflects hierarchy** (`Category/Component`), matching the app's
  taxonomy.
- **Interactions:** use the `play` function for stateful flows; assert with
  `@storybook/test` where behavior matters.
- **Accessibility:** keep the a11y addon clean — fix violations, don't disable.
- **Isolation:** a story renders without app context; provide needed
  providers/mocks via decorators, not real network.
- **No dead stories:** remove stories for deleted components/variants.

## Commands

```bash
pnpm storybook          # run Storybook locally
pnpm build-storybook    # static build (CI)
pnpm test-storybook     # run interaction/a11y tests (if configured)
```

## Checklist before "done"

- Every new/changed component has a co-located, up-to-date `*.stories.tsx`.
- Stories cover the component's states and variants via `args`.
- Storybook builds and the a11y/interaction checks pass.
