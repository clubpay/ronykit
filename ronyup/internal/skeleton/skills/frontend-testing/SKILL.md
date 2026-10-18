---
name: frontend-testing
description: >-
  Test web frontends (React/Next.js) effectively. Use when writing or fixing
  unit/component tests with Vitest + Testing Library, end-to-end, a11y, and
  responsive layout tests with Playwright, or testing shared WebMCP/RonyKit
  stub handlers.
---

# Frontend Testing

Test what the user (and assistive tech) experiences, not implementation details.
Query the DOM the way a human or screen reader does.

## Close the loop yourself (do not hand back to the user)

Writing tests/code is not the finish line — a clean run is. Drive the
verify-and-fix loop autonomously; never ask the user to run the app, paste
errors back, or confirm a fix worked.

1. Run `bash frontend/verify.sh` (which runs `typecheck`, `lint`, `build`,
   `test`, `build-storybook`, UI polish static checks, and story coverage).
2. Read the full output, fix every error and meaningful warning, then re-run
   from the failed step.
3. Repeat until all steps pass. Only stop to ask the user when genuinely
   blocked (missing secret, ambiguous product decision) — not for routine
   type/lint/test/build failures.
4. Report completion only with fresh passing output as evidence.

## When to use

- Adding tests for React components, hooks, Server Actions, or Next.js routes.
- Writing end-to-end, accessibility, or responsive layout checks with Playwright.
- Testing RonyKit TypeScript stub wrappers or WebMCP tool handlers.
- Fixing flaky or implementation-coupled frontend tests.

## Levels

1. **Unit** — pure functions, hooks, state reducers, RonyKit stub error mappers,
   and WebMCP tool handlers. Fast, no DOM where possible.
2. **Component (Vitest + Testing Library)** — render a component; assert on
   accessible roles, states, form validation, and keyboard interaction.
3. **End-to-end & A11y (Playwright)** — drive a real browser through critical
   user journeys, WCAG scans (`@axe-core/playwright`), and mobile/desktop
   viewport checks.

Prefer the lowest level that proves the behavior; reserve E2E for critical paths.

## 1. Component tests (Vitest + Testing Library)

- **Query by role/label/text**, never class names or internal state:
  `getByRole('button', { name: /delete workspace/i })`, `getByLabelText(/email/i)`.
  Use `data-testid` only as a last resort when no semantic role exists.
- **Drive interactions with `@testing-library/user-event`**, including keyboard
  flows (`await user.tab()`, `await user.keyboard('{Enter}')`, `'{Escape}'`).
- **Test the interactive states that matter** (see `ux-quality`):
  - Form validation fires on blur/submit (not first keystroke), sets
    `aria-invalid="true"`, and exposes the error via `aria-describedby`.
  - Pending/loading state disables duplicate clicks.
  - Empty vs. filtered-empty state (`"Clear filters"` resets the query).
- **Mock network at the boundary** (MSW or typed RonyKit stub dependency
  injection), never by stubbing your own child components.

```tsx
test('validates email on submit and clears error when corrected', async () => {
  const onSubmit = vi.fn();
  const user = userEvent.setup();
  render(<InviteMemberForm onSubmit={onSubmit} />);

  await user.click(screen.getByRole('button', { name: /send invite/i }));
  const emailInput = screen.getByLabelText(/work email/i);
  expect(emailInput).toHaveAttribute('aria-invalid', 'true');

  await user.type(emailInput, 'ada@example.com');
  expect(emailInput).not.toHaveAttribute('aria-invalid', 'true');

  await user.click(screen.getByRole('button', { name: /send invite/i }));
  expect(onSubmit).toHaveBeenCalledWith({ email: 'ada@example.com' });
});
```

## 2. End-to-end, accessibility & layout resilience (Playwright)

- Use role/label locators and web-first assertions (`await expect(locator).toBeVisible()`)
  which auto-wait — **never use `page.waitForTimeout()`**.
- **Automated WCAG 2.2 AA scan (`@axe-core/playwright`):** Run an axe scan on
  every primary route in both light and dark themes (and with open modals/drawers):

```ts
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test('dashboard has no WCAG AA accessibility violations and no horizontal overflow', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 }); // mobile check first
  await page.goto('/dashboard');

  await expect(page.getByRole('heading', { level: 1 })).toBeVisible();

  // Assert zero unexpected horizontal page overflow at 375px mobile width
  const overflows = await page.evaluate(
    () => document.documentElement.scrollWidth > document.documentElement.clientWidth,
  );
  expect(overflows).toBe(false);

  // Assert zero WCAG 2.2 AA violations
  const a11y = await new AxeBuilder({ page })
    .withTags(['wcag2a', 'wcag2aa', 'wcag21aa', 'wcag22aa'])
    .analyze();
  expect(a11y.violations).toEqual([]);
});
```

## 3. Anti-patterns

- Asserting on implementation details (internal state, Tailwind class names,
  child component props).
- `data-testid` everywhere instead of accessible `getByRole` / `getByLabelText`
  queries (if a control is hard to query by role or label, its accessibility is
  broken — fix the markup!).
- Arbitrary sleeps (`waitForTimeout`, `setTimeout`) instead of web-first
  assertions or `findBy*`.
- Skipping error, empty, or keyboard-only paths.

## Commands

```bash
pnpm test                # Vitest unit/component
pnpm test:e2e            # Playwright E2E + a11y
bash frontend/verify.sh  # Full verification gate
```
