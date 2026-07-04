# Testing WebMCP tools

Verify tools the way you verify API handlers — with structured calls and
assertions on inputs/outputs, not by manually clicking the UI.

## Browser testing API

Chromium builds with WebMCP expose `navigator.modelContextTesting` (testing
only — not for production logic):

```javascript
// List registered tools
const tools = navigator.modelContextTesting.listTools();
console.log(tools.map((t) => t.name));

// Execute a tool
const raw = await navigator.modelContextTesting.executeTool(
  'orders_filter',
  JSON.stringify({ status: 'paid', limit: 5 }),
);
const result = JSON.parse(raw);
```

### Chrome setup (native WebMCP)

1. Chrome Canary/Beta 149+ (origin trial) or flag-enabled build.
2. Enable WebMCP testing:
   `chrome://flags/#enable-webmcp-testing`
3. Page must be **origin-isolated** (default for HTTPS localhost).

Chrome guide: [developer.chrome.com/docs/ai/webmcp](https://developer.chrome.com/docs/ai/webmcp)

Install the **Model Context Tool Inspector** extension (linked from Chrome docs)
to exercise tools with natural language during development.

## MCP-B polyfill testing

With `@mcp-b/global`, the same `document.modelContext` API works in dev
browsers without native support. After loading the app:

```javascript
const tools = await document.modelContext.getTools?.() ?? [];
// or modelContextTesting when available
```

Tutorial with console steps:
[docs.mcp-b.ai/tutorials/first-react-tool](https://docs.mcp-b.ai/tutorials/first-react-tool)

Runnable reference: `example/ex-13-webmcp/` in the RonyKit monorepo (task board with
five tools).

## Vitest component tests

Test the **handler logic** directly; mock `useWebMCP` registration in unit
tests if needed:

```tsx
import { describe, it, expect, vi } from 'vitest';
import { buildOrdersFilterResult } from './orders-logic';

describe('orders_filter handler logic', () => {
  it('returns summaries for paid orders', async () => {
    const result = await buildOrdersFilterResult({ status: 'paid', limit: 10 });
    expect(result.total).toBeGreaterThanOrEqual(0);
    expect(result.items[0]).toHaveProperty('id');
  });
});
```

Extract `buildOrdersFilterResult` (or similar) from the component so UI and
WebMCP share one testable function — same pattern as Server Actions.

## Playwright E2E (optional)

For full integration, drive `modelContextTesting` inside a WebMCP-capable
browser context. Reference consumer tooling:
[nekuda-ai/webmcp](https://github.com/nekuda-ai/webmcp) (`webmcp-playwright.mjs`).

Set `WEBMCP_CHROME` to Chrome Canary when Playwright's bundled Chromium lacks
WebMCP.

## Desktop agent relay (dev only)

`@mcp-b/webmcp-local-relay` forwards page tools to Claude Desktop / Cursor.
Use to validate schemas with real agents during development — not a substitute
for automated tests.

Docs: [docs.mcp-b.ai](https://docs.mcp-b.ai) → "Connect a Desktop Agent"

## What to test

| Case | Assert |
|------|--------|
| Tool registers on mount | Name appears in `listTools()` |
| Valid input | Structured return matches schema |
| Invalid input | Error object or rejected promise |
| Auth gate | Unauthenticated call returns error, no mutation |
| UI sync | After tool call, DOM shows updated state |
| Unmount cleanup | Tool removed from list after navigating away |

## CI practical note

Most CI runners lack WebMCP-native Chrome today. Default CI strategy:

1. Unit-test shared handler logic (always).
2. Run Playwright + WebMCP only in a optional job or locally before release.
3. Polyfill smoke test: verify app builds and `document.modelContext` exists
   after `@mcp-b/global` in a headless browser if you add a thin E2E check.

Document skipped WebMCP E2E in the PR when CI cannot run native tools yet.
