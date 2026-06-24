---
name: nextjs-modern
description: >-
  Build modern Next.js (16, App Router) apps with React Server Components. Use
  when creating or reviewing Next.js routes, layouts, server/client components,
  data fetching, Server Actions, caching, or metadata in the frontend app.
---

# Modern Next.js (App Router)

Default to the server. Reach for the client only when you need interactivity.
Target Next.js 16 with the App Router and React Server Components.

## When to use

- Adding or reviewing routes, layouts, pages, or components in the frontend app.
- Choosing server vs client components, a data-fetching strategy, or caching.
- Wiring forms and mutations via Server Actions.

## Server vs Client Components

Components are **Server Components by default**. Add `'use client'` only when you
need state, effects, event handlers, or browser-only APIs.

- Keep `'use client'` at the leaves; pass server-fetched data down as props.
- Don't put `'use client'` on a layout/page just to use one interactive child —
  split the child out.
- Never import server-only code (DB clients, secrets) into a client component.
- **Minimize the serialization boundary:** props crossing server → client are
  serialized into the RSC payload. Pass only the minimal, serializable fields a
  client component needs — not whole ORM rows or large objects. Non-serializable
  values (functions, class instances, Dates in some setups) won't cross.

## Data fetching

- Fetch directly in async Server Components — no `useEffect` for initial data.
- Choose the cache explicitly: static by default, `revalidate` for ISR,
  `cache: 'no-store'` / `dynamic` for per-request data.
- Stream slow data with `<Suspense>` and `loading.tsx`; colocate `error.tsx`.
- Use `next/cache` (`revalidateTag`/`revalidatePath`) to invalidate after writes.

### Avoid request waterfalls

- Fetch independent data **in parallel**, not sequentially. Kick off the
  requests, then await together:

```tsx
const [user, posts] = await Promise.all([getUser(id), getPosts(id)]);
```

- Don't `await` a request only to pass its result into the next independent one
  — that serializes round-trips and inflates TTFB.
- Stream non-critical sections behind their own `<Suspense>` so the shell
  renders immediately.

### Deduplicate requests

- `fetch` is automatically deduped within a render pass — call it where the data
  is needed instead of prop-drilling.
- For non-`fetch` data sources (DB/ORM), wrap the loader in React `cache()` so
  repeated calls in one render hit the source once:

```tsx
import { cache } from 'react';
export const getUser = cache(async (id: string) => db.user.find(id));
```

## Mutations — Server Actions

- Use Server Actions for form submissions and mutations instead of hand-rolled
  API routes where possible.
- **Authenticate and authorize inside every action.** A Server Action is a
  public POST endpoint — hiding the button in the UI does not protect it. Re-check
  the session and the caller's permission to act on the target resource at the
  top of the action, before any mutation.
- Validate input on the server (e.g. Zod) — never trust the client.
- Revalidate or redirect after a successful mutation.

## Routing & structure

- Use the file conventions: `layout.tsx`, `page.tsx`, `loading.tsx`,
  `error.tsx`, `not-found.tsx`, route groups `(group)`, dynamic `[param]`.
- Co-locate components/tests with the route; share via a top-level module.
- `<Link>` for navigation; `next/image` and `next/font` for assets.

## Metadata & a11y

- Export `metadata` or `generateMetadata` for SEO instead of manual `<head>`.
- Semantic HTML, labelled controls, keyboard reachability.

## Performance

- Keep client bundles small; prefer Server Components and `next/dynamic` for
  heavy client-only widgets.
- Use `next/image` (sizing, lazy loading) and avoid layout shift.
- Pick the runtime deliberately: default Node.js runtime for full APIs; opt into
  the Edge runtime (`export const runtime = 'edge'`) only for light, latency-
  sensitive routes that don't need Node-only APIs.
- Profile before optimizing — use the bundle analyzer and a Lighthouse pass to
  confirm wins (TTFB, payload size) rather than guessing.

## Checklist

- Default to Server Components; `'use client'` only where required and at leaves.
- Independent data fetched in parallel; loaders deduped; no waterfalls.
- Caching strategy chosen deliberately per fetch.
- Mutations authorize the caller, validate input server-side, and revalidate
  affected data.
- `pnpm lint`, `pnpm typecheck`, and tests pass.
