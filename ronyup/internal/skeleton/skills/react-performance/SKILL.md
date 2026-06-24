---
name: react-performance
description: >-
  Optimize React/Next.js runtime performance — eliminate waterfalls, shrink
  bundles, cut unnecessary re-renders, and speed up rendering. Use when writing,
  reviewing, or refactoring components for speed, or chasing slow renders, large
  bundles, janky interactions, or high TTFB/LCP/INP.
---

# React Performance

70 performance rules for React and Next.js, vendored from Vercel Engineering's
`react-best-practices` (MIT). The full rule set lives in `rules/` — each file is
self-contained (impact metric + incorrect→correct code). This `SKILL.md` is the
**index and router**: it carries the highest-impact patterns inline so you can
act without opening anything, and points you to the exact rule file for depth.

For **Server Components, data fetching, caching, Server Actions, and route-level
code splitting** at a higher level, see `nextjs-modern`. The `async-*` and
`server-*` rules here are the detailed reference for that same ground.

## When to use

- Writing or reviewing React components, hooks, or Next.js pages for speed.
- Chasing slow renders, janky interactions, large bundles, or poor Core Web
  Vitals (LCP, INP, TTFB).
- Refactoring code that re-renders too often or ships too much JS.

## How to use the rules (read this first)

1. **Measure before optimizing.** Profile the proven bottleneck with the React
   DevTools Profiler, the Next.js bundle analyzer, and a Lighthouse/Web Vitals
   pass. Don't memoize on a hunch — most "just to be safe" memoization only
   adds cost.
2. **Apply in priority order** (table below) — the top categories move the
   needle far more than the bottom ones.
3. **Open the specific rule file before applying it**, using the routing table
   and index below. Each `rules/<name>.md` has the exact incorrect→correct code
   and the impact metric. Match the example, don't paraphrase from memory.

### Priority order

| # | Category               | Impact       | Prefix       |
| - | ---------------------- | ------------ | ------------ |
| 1 | Eliminate waterfalls   | CRITICAL     | `async-`     |
| 2 | Bundle size            | CRITICAL     | `bundle-`    |
| 3 | Server-side perf       | HIGH         | `server-`    |
| 4 | Client data fetching   | MEDIUM-HIGH  | `client-`    |
| 5 | Re-render optimization | MEDIUM       | `rerender-`  |
| 6 | Rendering performance  | MEDIUM       | `rendering-` |
| 7 | JavaScript hot paths   | LOW-MEDIUM   | `js-`        |
| 8 | Advanced patterns      | LOW          | `advanced-`  |

### Routing — what to read for the task at hand

| If you are working on…                          | Read these rules            |
| ----------------------------------------------- | --------------------------- |
| Server data fetching / API routes / actions     | `async-*`, `server-*`       |
| Bundle size, slow initial load, code splitting  | `bundle-*`                  |
| Client-side fetching, global listeners, storage | `client-*`                  |
| Too many re-renders, janky typing/interaction   | `rerender-*`                |
| Paint, hydration, long lists, SVG, scripts      | `rendering-*`               |
| Hot loops over large data                       | `js-*`                      |
| Stable refs, init-once, effect-event edge cases | `advanced-*`                |

## Common path (act on these without opening a file)

The CRITICAL/HIGH patterns, inline. For the rest, open the rule file.

- **Parallelize independent awaits** (`async-parallel`, 2–10×). Sequential
  `await`s each cost a full round-trip.

```tsx
const [user, posts] = await Promise.all([fetchUser(), fetchPosts()]);
```

- **Check cheap sync conditions before awaiting**, and move an `await` into the
  branch that uses it (`async-cheap-condition-before-await`, `async-defer-await`).
- **No barrel imports on the hot path** (`bundle-barrel-imports`) — import from
  the direct path; keep import paths statically analyzable (`bundle-analyzable-paths`).
- **`next/dynamic` for heavy/below-the-fold components**; defer third-party
  scripts until after hydration (`bundle-dynamic-imports`, `bundle-defer-third-party`).
- **Authenticate Server Actions** like API routes — they're public endpoints
  (`server-auth-actions`). Dedupe per-request reads with `React.cache()`
  (`server-cache-react`); minimize props serialized to client components
  (`server-serialization`).
- **Derive state during render, not in an effect** (`rerender-derived-state-no-effect`):

```tsx
const fullName = `${first} ${last}`; // not useState + useEffect
```

- **Never define a component inside a component** (`rerender-no-inline-components`)
  — it remounts the subtree every render.
- **Wrap non-urgent updates in a transition** (`rerender-transitions`,
  `rerender-use-deferred-value`) so input stays responsive.

## Full rule index

Open `rules/<name>.md` for the code example and impact metric.

### 1. Eliminate waterfalls — CRITICAL (`async-`)

- `async-parallel` — `Promise.all()` for independent operations.
- `async-cheap-condition-before-await` — check sync conditions before awaiting.
- `async-defer-await` — move `await` into the branch that uses it.
- `async-dependencies` — `better-all` for partial dependencies.
- `async-api-routes` — start promises early, await late in API routes.
- `async-suspense-boundaries` — `<Suspense>` to stream content.

### 2. Bundle size — CRITICAL (`bundle-`)

- `bundle-barrel-imports` — import directly, avoid barrel files.
- `bundle-analyzable-paths` — keep import/file-system paths statically analyzable.
- `bundle-dynamic-imports` — `next/dynamic` for heavy components.
- `bundle-defer-third-party` — load analytics/logging after hydration.
- `bundle-conditional` — load modules only when a feature activates.
- `bundle-preload` — preload on hover/focus for perceived speed.

### 3. Server-side performance — HIGH (`server-`)

- `server-auth-actions` — authenticate server actions like API routes.
- `server-cache-react` — `React.cache()` for per-request dedup.
- `server-cache-lru` — LRU cache for cross-request caching.
- `server-dedup-props` — avoid duplicate serialization in RSC props.
- `server-hoist-static-io` — hoist static I/O (fonts, logos) to module level.
- `server-no-shared-module-state` — no module-level mutable request state.
- `server-serialization` — minimize data passed to client components.
- `server-parallel-fetching` — restructure components to parallelize fetches.
- `server-parallel-nested-fetching` — chain nested fetches in `Promise.all`.
- `server-after-nonblocking` — `after()` for non-blocking work.

### 4. Client-side data fetching — MEDIUM-HIGH (`client-`)

- `client-swr-dedup` — SWR for automatic request deduplication.
- `client-event-listeners` — deduplicate global event listeners.
- `client-passive-event-listeners` — passive listeners for scroll/touch.
- `client-localstorage-schema` — version and minimize localStorage data.

### 5. Re-render optimization — MEDIUM (`rerender-`)

- `rerender-defer-reads` — don't subscribe to state only used in callbacks.
- `rerender-memo` — extract expensive work into memoized components.
- `rerender-memo-with-default-value` — hoist default non-primitive props.
- `rerender-dependencies` — use primitive dependencies in effects.
- `rerender-derived-state` — subscribe to derived booleans, not raw values.
- `rerender-derived-state-no-effect` — derive during render, not in effects.
- `rerender-functional-setstate` — functional `setState` for stable callbacks.
- `rerender-lazy-state-init` — pass a function to `useState` for expensive values.
- `rerender-simple-expression-in-memo` — don't memo trivial primitives.
- `rerender-split-combined-hooks` — split hooks with independent dependencies.
- `rerender-move-effect-to-event` — put interaction logic in event handlers.
- `rerender-transitions` — `startTransition` for non-urgent updates.
- `rerender-use-deferred-value` — defer expensive renders to keep input responsive.
- `rerender-use-ref-transient-values` — refs for transient frequent values.
- `rerender-no-inline-components` — don't define components inside components.

### 6. Rendering performance — MEDIUM (`rendering-`)

- `rendering-animate-svg-wrapper` — animate the wrapper, not the SVG element.
- `rendering-content-visibility` — `content-visibility` for long lists.
- `rendering-hoist-jsx` — extract static JSX outside components.
- `rendering-svg-precision` — reduce SVG coordinate precision.
- `rendering-hydration-no-flicker` — inline script for client-only data.
- `rendering-hydration-suppress-warning` — suppress expected mismatches.
- `rendering-activity` — `Activity` component for show/hide.
- `rendering-conditional-render` — ternary, not `&&`, for conditionals.
- `rendering-usetransition-loading` — prefer `useTransition` for loading state.
- `rendering-resource-hints` — React DOM resource hints for preloading.
- `rendering-script-defer-async` — `defer`/`async` on script tags.

### 7. JavaScript hot paths — LOW-MEDIUM (`js-`)

Only inside measured loops or large datasets — don't litter normal code.

- `js-index-maps` — build a `Map` for repeated lookups.
- `js-set-map-lookups` — `Set`/`Map` for O(1) lookups.
- `js-combine-iterations` — combine multiple filter/map into one loop.
- `js-flatmap-filter` — `flatMap` to map and filter in one pass.
- `js-cache-property-access` — cache object properties in loops.
- `js-cache-function-results` — cache results in a module-level `Map`.
- `js-cache-storage` — cache localStorage/sessionStorage reads.
- `js-batch-dom-css` — group CSS changes via classes or `cssText`.
- `js-length-check-first` — check array length before expensive comparison.
- `js-early-exit` — return early from functions.
- `js-hoist-regexp` — hoist `RegExp` creation outside loops.
- `js-min-max-loop` — loop for min/max instead of sort.
- `js-tosorted-immutable` — `toSorted()` for immutability.
- `js-request-idle-callback` — defer non-critical work to idle time.

### 8. Advanced patterns — LOW (`advanced-`)

- `advanced-effect-event-deps` — don't put `useEffectEvent` results in deps.
- `advanced-event-handler-refs` — store event handlers in refs.
- `advanced-init-once` — initialize app once per app load.
- `advanced-use-latest` — `useLatest` for stable callback refs.

## Checklist before "done"

- Bottleneck profiled (Profiler / bundle analyzer / Lighthouse), not guessed.
- Relevant `rules/*.md` opened and the code matched, not paraphrased.
- No sequential await waterfalls; heavy/rare code split via `next/dynamic`.
- No state derived in effects; no components defined inside components;
  transient values in refs.
- Non-urgent updates wrapped in a transition; memoization justified by a
  measurement.
- `pnpm lint` and `pnpm typecheck` pass.

## Attribution

Rules in `rules/` are vendored from
[`vercel-labs/agent-skills`](https://github.com/vercel-labs/agent-skills)
(`skills/react-best-practices`), © Vercel Engineering, MIT License. See
`rules/metadata.json` for version and references.
