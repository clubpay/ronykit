---
name: nextjs-modern
description: >-
  Build and review Next.js App Router applications (Next.js 15/16+) — Server vs
  Client Component boundaries, async request APIs, data fetching and caching,
  Server Actions, streaming Suspense, URL state, LCP/image optimization, and
  deterministic metadata/SEO. Use when creating or modifying `app/` routes,
  layouts, loading/error boundaries, metadata, or data mutations.
---

# Modern Next.js (App Router)

Apply these conventions when building or reviewing Next.js App Router (`app/`)
codebases.

## 1. Server Components by default; Client Components at the leaves

- Every file in `app/` is a **React Server Component (RSC)** unless marked with
  `'use client'`. Keep layouts, pages, and data-fetching wrappers on the server.
- Push `'use client'` **as far down the component tree as possible** — only on
  interactive leaves that use `useState`, `useEffect`, event handlers, or
  browser APIs.
- **Compose Server inside Client via `children`:** A Client Component (e.g., an
  animated modal or collapsible sidebar) can render Server Components without
  turning them into client bundles when passed as `children` or slots.
- **Serialization boundary:** Props passed from Server → Client must be
  serializable (plain objects, primitives, Promises, Server Actions — never
  class instances or unserializable functions).

## 2. Async request APIs (Next.js 15+)

In Next.js 15+, `params`, `searchParams`, `cookies()`, `headers()`, and
`draftMode()` are **asynchronous**. Always `await` them:

```tsx
export default async function OrderPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<{ tab?: string }>;
}) {
  const { id } = await params;
  const { tab = 'overview' } = await searchParams;
  // ...
}
```

## 3. Data fetching & parallel streaming

- **Colocate reads where needed** and wrap shared server reads in `React.cache()`
  so layout, `generateMetadata`, and page deduplicate within a single request.
- **Eliminate waterfalls:** Start independent fetches in parallel with
  `Promise.all([getUser(id), getOrders(id)])` or split slow secondary sections
  into separate async Server Components wrapped in `<Suspense fallback={<Skeleton />}>`.
- **Never call your own Route Handlers (`app/api/...`) from a Server Component**
  over HTTP — call the underlying service/SDK or typed RonyKit stub directly.
- **Pass Promises to Client Components** and unwrap them with React 19's `use(promise)`
  when the shell should render immediately.

## 4. RonyKit TypeScript stub integration (`stub/<name>stub-typescript`)

In RonyKit fullstack workspaces, `make gen-stub` inside a backend feature
generates a zero-dependency TypeScript client (`stub.ts` + `swr.hooks.ts`):

- **Generated surface:** `<Service>Stub` (with typed `<method>(req, headers?, init?)`
  calls), `<Service>StubError` (`status`, `body`, `payload`, plus typed getters
  `code`, `item`, `serverMessage`, `details`), DTO interfaces, and `'use client'`
  SWR hooks (`use<Method>(stub, req | null | undefined, headers?, options?, init?)`
  typed with `<Service>StubError`).
- **Server Components & Server Actions:** Instantiate `<Service>Stub` on the
  server using `process.env.BACKEND_URL` and forward auth headers from
  `await cookies()` / `await headers()`. Pass Next.js `fetch` cache/revalidation
  options via the 3rd `init` parameter (`{ next: { tags: ['orders'] } }`).
- **Client reads:** Use the generated `use<Method>` SWR hooks from
  `swr.hooks.ts` when live client polling/revalidation is needed; pass `null` as
  `req` to conditionally pause fetching until parameters are ready.
- **Map `rony/errs` payloads cleanly:** `<Service>StubError` automatically parses
  RonyKit's `rony/errs` JSON (`{ code, item, message, details }`) into
  `err.code`, `err.item`, and `err.serverMessage`. Map `err.item` to
  `fieldErrors[err.item]` and surface `err.serverMessage` inline (never raw
  stack traces or generic `"Something went wrong"`):

```ts
export function toFormError(err: unknown): {
  fieldErrors?: Record<string, string[]>;
  message: string;
} {
  if (err && typeof err === 'object' && 'serverMessage' in err) {
    const stubErr = err as { item?: string; serverMessage: string };
    return {
      fieldErrors: stubErr.item ? { [stubErr.item]: [stubErr.serverMessage] } : undefined,
      message: stubErr.serverMessage,
    };
  }
  return { message: 'Unable to complete request. Please try again.' };
}
```

## 5. Mutations & Server Actions

- Validate every Server Action input at the boundary with **Zod** (`safeParse`)
  and verify **authentication + authorization inside the action** — Server
  Actions are public HTTP endpoints.
- Return structured results (`{ ok: true, data } | { ok: false, fieldErrors, message }`)
  for expected validation/domain errors instead of throwing raw exceptions.
- Pair form actions with `useActionState` (React 19) for pending state + field
  errors, and `useOptimistic` for instant UI updates.
- Call `revalidatePath` / `revalidateTag` after mutations before redirecting.

## 6. URL state, routing & boundary files

- **Deep-linkable UI state:** Store filter facets, search queries, sort order,
  pagination, and active tabs in URL search params (`useSearchParams` or `nuqs`)
  so views survive refresh and can be shared.
- **Real `<Link>` navigation:** Always use `<Link href="...">` for internal
  navigation (never `router.push` inside a `<div onClick>`) so prefetching,
  Cmd/Ctrl-click, and middle-click work.
- **Route boundaries:** Provide `loading.tsx` (layout-matched structural
  skeletons with zero CLS), `error.tsx` (`'use client'` with a `reset()` retry
  button), and `not-found.tsx` at meaningful route segments.

## 7. Metadata, Open Graph & SEO discipline

- **Single source of truth per route:** Define metadata via the static
  `export const metadata: Metadata` object or `generateMetadata` in `layout.tsx`
  / `page.tsx` — never render duplicate raw `<title>` or `<meta>` tags inside
  component bodies.
- **Set `metadataBase` once in the root `app/layout.tsx`** so relative Open
  Graph and Twitter image paths resolve to **absolute URLs**.
- **URL & title agreement:** Keep `title` (with root `title.template: "%s · App"`),
  `description`, `alternates.canonical`, and `openGraph.url` consistent on
  every indexable page. Pair `openGraph.images` (`1200×630`) with
  `twitter.card = "summary_large_image"`.
- **Viewport & browser chrome:** Export `viewport: Viewport` in the root layout
  with light/dark `themeColor` matching your `--background` tokens:
  ```tsx
  export const viewport: Viewport = {
    themeColor: [
      { media: '(prefers-color-scheme: light)', color: '#fafaf9' },
      { media: '(prefers-color-scheme: dark)', color: '#0c0d10' },
    ],
  };
  ```
- **Honest structured data:** Only emit JSON-LD (`application/ld+json`) for
  entities actually rendered on the page, and Gate non-production environments
  with `robots: { index: false, follow: false }`.

## 8. Images, fonts, LCP & hydration safety

- **LCP image priority:** Use `next/image` with explicit `width`/`height` (or
  `fill` inside an `aspect-*` container). Add `priority` (`fetchpriority="high"`)
  to the above-the-fold hero/LCP image — **never set `loading="lazy"` on the LCP
  image**. Lazy-load below-the-fold media.
- **Zero-CLS fonts:** Load fonts in the root layout via `next/font/google` or
  `next/font/local` bound to CSS variables (`--font-sans`, `--font-display`,
  `--font-mono`).
- **Hydration safety:** Never read `window`, `localStorage`, `Math.random()`, or
  `new Date()` directly during render in a Client Component — defer client-only
  values to `useEffect` or format timestamps consistently with an explicit
  locale/timezone. Never mix `value` without `onChange` and `defaultValue` on
  the same input.

## Checklist

- [ ] `'use client'` only on leaf interactive components; `params`/`searchParams` awaited
- [ ] Independent server fetches run in parallel (`Promise.all` or `<Suspense>`)
- [ ] Server Actions validate inputs with Zod and check auth before mutating
- [ ] Filters, tabs, and pagination sync to URL search params
- [ ] `metadataBase`, `canonical`, `openGraph`, and `viewport.themeColor` configured; LCP image uses `priority`
