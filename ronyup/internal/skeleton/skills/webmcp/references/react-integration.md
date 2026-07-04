# React / Next.js WebMCP integration

## Install

```bash
pnpm add @mcp-b/global @mcp-b/react-webmcp zod
```

Optional dev typing for raw `registerTool` calls:

```bash
pnpm add -D @mcp-b/webmcp-types
```

## Provider (load once, client-only)

`@mcp-b/global` must run before any tool registration. In Next.js App Router,
use a dedicated client component — do **not** mark the root layout `'use client'`.

```tsx
// components/webmcp-provider.tsx
'use client';

import '@mcp-b/global';

export function WebMCPProvider({ children }: { children: React.ReactNode }) {
  return <>{children}</>;
}
```

```tsx
// app/layout.tsx — Server Component
import { WebMCPProvider } from '@/components/webmcp-provider';

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <WebMCPProvider>{children}</WebMCPProvider>
      </body>
    </html>
  );
}
```

### Vite / CRA

Import `@mcp-b/global` at the top of `main.tsx` before `createRoot`, or use the
same provider wrapper pattern.

## Register tools at the owning component

Register where the state lives so schemas stay accurate and cleanup is automatic.

```tsx
'use client';

import { useWebMCP } from '@mcp-b/react-webmcp';
import { z } from 'zod';
import { useOrdersFilters } from '@/hooks/use-orders-filters';

export function OrdersToolbar() {
  const { filters, setFilters, search } = useOrdersFilters();

  useWebMCP(
    {
      name: 'orders_filter',
      description:
        'Filter the orders table by status, date range, and free-text search.',
      inputSchema: {
        status: z
          .enum(['all', 'pending', 'paid', 'shipped', 'cancelled'])
          .optional()
          .describe('Order status filter'),
        query: z.string().optional().describe('Search order ID or customer email'),
        from: z.string().datetime().optional().describe('ISO start date'),
        to: z.string().datetime().optional().describe('ISO end date'),
      },
      annotations: {
        title: 'Filter orders',
        readOnlyHint: false,
      },
      handler: async (input) => {
        const next = { ...filters, ...input };
        setFilters(next);
        const result = await search(next);
        return {
          total: result.total,
          items: result.items.map((o) => ({
            id: o.id,
            status: o.status,
            total: o.totalCents,
          })),
        };
      },
    },
    [filters],
  );

  return (/* existing toolbar UI */);
}
```

### `useWebMCPContext` for read-only snapshots

Expose current page context without mutation:

```tsx
useWebMCPContext(
  'orders_list_state',
  'Current orders table filters and result count',
  () => ({ filters, total, page }),
);
```

## Next.js App Router rules

| Rule | Why |
|------|-----|
| Provider + hooks in `'use client'` files only | `document` is browser-only |
| Do not import server-only modules into tool files | Secrets and DB clients stay on server |
| Prefer calling Server Actions from tool handlers for mutations | One validation path on the server |
| Pass serializable data into client tool components | RSC boundary limits |

### Server Actions from tools

```tsx
'use client';

import { updateUserRole } from '@/app/actions/users'; // Server Action

useWebMCP({
  name: 'users_update_role',
  description: 'Change a user role. Requires admin session.',
  inputSchema: {
    userId: z.string().uuid(),
    role: z.enum(['member', 'admin', 'viewer']),
  },
  handler: async (input) => {
    const result = await updateUserRole(input);
    if (!result.ok) return { success: false, error: result.error };
    return { success: true, userId: input.userId, role: input.role };
  },
});
```

The Server Action must perform the same auth checks as the admin UI form.

## File layout (recommended)

```
frontend/<app>/src/
  components/webmcp-provider.tsx    # polyfill bootstrap
  features/orders/
    orders-toolbar.tsx              # UI + orders_filter tool
    orders-table.tsx                # UI + orders_list_state context
  features/users/
    user-role-form.tsx              # UI + users_update_role tool
```

Avoid a monolithic `tools/register-all.ts` unless the app is tiny.

## Execution state in the UI

`useWebMCP` returns `state` (`isExecuting`, `error`, `lastResult`). Use it for
agent-invoked loading indicators — especially when humans and agents share a
screen:

```tsx
const filterTool = useWebMCP({ /* ... */ });

{filterTool.state.isExecuting && <span className="text-muted">Agent updating…</span>}
```

## Permissions policy

If tools fail with `NotAllowedError`, ensure the page is **origin-isolated**
and the `tools` policy allows your origin. Do not set `Origin-Agent-Cluster: ?0`
or use `document.domain` — both disable WebMCP.

For cross-origin iframes embedding your app, you may need `exposedTo` on
`registerTool` (advanced; see W3C spec). Default same-origin apps do not need
this.

## Further reading

- [@mcp-b/react-webmcp reference](https://docs.mcp-b.ai/packages/react-webmcp/reference)
- [First React tool tutorial](https://docs.mcp-b.ai/tutorials/first-react-tool)
- [Framework guides](https://docs.mcp-b.ai/how-to/frameworks)
