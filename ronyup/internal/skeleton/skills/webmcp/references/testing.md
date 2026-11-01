# Testing WebMCP tools

Verify tools the way you verify API handlers — with structured calls and
assertions on inputs/outputs, not by manually clicking the UI.

## Browser testing API

The current W3C API and `@mcp-b/global` expose `getTools()` and `executeTool()`
on `document.modelContext`. Execution takes the discovered tool descriptor and
an input object, not a tool name and JSON string:

```javascript
const tools = await document.modelContext.getTools();
const filterTool = tools.find((tool) => tool.name === 'orders_filter');
if (!filterTool) throw new Error('orders_filter is not registered');

const raw = await document.modelContext.executeTool(filterTool, {
  status: 'paid',
  limit: 5,
});
const result = JSON.parse(raw);
```

Some Chromium builds also expose `navigator.modelContextTesting` for test-only
introspection. Treat it as an optional browser helper, not the portable API or
production application logic.

### Chrome setup (native WebMCP)

1. Join the WebMCP origin trial (available from Chrome 149), or use a current
   flag-enabled Chromium build.
2. For local flag-based development, enable
   `chrome://flags/#enable-webmcp-testing` and relaunch.
3. Use a secure context and leave the `tools` Permissions Policy enabled.

Chrome guide: [developer.chrome.com/docs/ai/webmcp](https://developer.chrome.com/docs/ai/webmcp)

Install the **Model Context Tool Inspector** extension (linked from Chrome docs)
to exercise tools with natural language during development.

## MCP-B polyfill testing

With `@mcp-b/global`, the same `document.modelContext.getTools()` and
`executeTool()` calls work in development browsers without native support.
Tutorial with console steps:
[docs.mcp-b.ai/tutorials/first-react-tool](https://docs.mcp-b.ai/tutorials/first-react-tool)

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

For full integration, evaluate the standard `document.modelContext.getTools()`
and `executeTool()` flow inside a WebMCP-capable browser context. Point
Playwright at a compatible Chrome/Chromium executable when its bundled browser
lacks WebMCP, and use the Chrome guide to enable the origin trial or testing
flag.

## Desktop agent relay (dev only)

`@mcp-b/webmcp-local-relay` forwards page tools to Claude Desktop / Cursor.
Use to validate schemas with real agents during development — not a substitute
for automated tests.

Docs: [docs.mcp-b.ai](https://docs.mcp-b.ai) → "Connect a Desktop Agent"

## What to test

| Case | Assert |
|------|--------|
| Tool registers on mount | Name appears in `getTools()` |
| Valid input | Structured return matches schema |
| Invalid input | Error object or rejected promise |
| Auth gate | Unauthenticated call returns error, no mutation |
| UI sync | After tool call, DOM shows updated state |
| Unmount cleanup | Tool removed from list after navigating away |

## CI practical note

Most CI runners lack WebMCP-native Chrome today. Default CI strategy:

1. Unit-test shared handler logic (always).
2. Run Playwright + WebMCP only in an optional job or locally before release.
3. Polyfill smoke test: verify app builds and `document.modelContext` exists
   after `@mcp-b/global` in a headless browser if you add a thin E2E check.

Document skipped WebMCP E2E in the PR when CI cannot run native tools yet.
