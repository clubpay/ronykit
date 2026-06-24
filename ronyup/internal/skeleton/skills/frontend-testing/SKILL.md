---
name: frontend-testing
description: >-
  Test web frontends (React/Next.js) effectively. Use when writing or fixing
  unit/component tests with Vitest + Testing Library, end-to-end tests with
  Playwright, or deciding what to test in a frontend app.
---

# Frontend Testing

Test what the user experiences, not implementation details. Query the DOM the
way a user (or assistive tech) does.

## When to use

- Adding tests for React components, hooks, or Next.js routes.
- Writing end-to-end flows with Playwright.
- Fixing flaky or implementation-coupled frontend tests.

## Levels

1. **Unit** — pure functions, hooks, utilities. Fast, no DOM where possible.
2. **Component** — render a component with Testing Library; assert on visible
   output and behavior.
3. **End-to-end** — Playwright drives a real browser through whole user flows.

Prefer the lowest level that proves the behavior; reserve E2E for critical paths.

## Component tests (Vitest + Testing Library)

- Query by **role/label/text**, not test IDs or class names:
  `getByRole('button', { name: /save/i })`. Use `data-testid` only as a last
  resort.
- Drive interactions with `@testing-library/user-event`, not raw `fireEvent`.
- Assert on what the user sees (text, roles, ARIA state), never on component
  internals or props.
- Use `findBy*` / `waitFor` for async UI instead of fixed timeouts.
- Mock network at the boundary (e.g. MSW), not by stubbing your own components.

```tsx
test('submits the form', async () => {
  const user = userEvent.setup()
  render(<LoginForm onSubmit={onSubmit} />)
  await user.type(screen.getByLabelText(/email/i), 'a@b.com')
  await user.click(screen.getByRole('button', { name: /sign in/i }))
  expect(onSubmit).toHaveBeenCalledWith({ email: 'a@b.com' })
})
```

## End-to-end (Playwright)

- Use role/text locators and web-first assertions (`await expect(locator)…`)
  which auto-wait — avoid `waitForTimeout`.
- Make tests independent and idempotent; reset state via API/fixtures, not the UI.
- Keep the suite small and focused on critical journeys; let component tests
  cover the variations.

## Anti-patterns

- Asserting on implementation (state, class names, child component props).
- `data-testid` everywhere instead of accessible queries.
- Arbitrary sleeps to "fix" flakiness instead of awaiting a condition.
- E2E tests for logic that a component or unit test could cover.

## Commands

```bash
pnpm test            # Vitest unit/component
pnpm test:e2e        # Playwright
```
