---
name: composition-patterns
description: >-
  Design reusable React component APIs that scale — compound components, lifted
  state, generic context, explicit variants. Use when a component is growing
  boolean props (isThread, isEditing…), when building a component library, or
  when reviewing component architecture. Includes React 19 API changes.
---

# React Composition Patterns

Build components by **composition, not configuration**. As a component grows, the
temptation is to add another boolean prop. Each one doubles the possible states
and breeds conditional logic until the component is unmaintainable. Compose small
pieces instead — explicit, self-documenting, and easy for both humans and agents
to extend. Adapted from Vercel Engineering's `composition-patterns` (MIT).

## When to use

- A component is accumulating boolean/mode props (`isThread`, `isEditing`,
  `isDM`, `showFooter`…) or deeply nested conditional rendering.
- Designing a reusable component or a component library API.
- Reviewing component architecture for flexibility and maintainability.

This governs how you *design your own* components; `shadcn` governs *using* its
components, and `react-performance` governs runtime speed. They don't overlap.

## 1. Avoid boolean-prop proliferation (the core rule)

Don't customize behavior with boolean flags — each flag multiplies states and
creates impossible combinations. Split into composed pieces instead.

```tsx
// Avoid: every new mode adds a flag and a branch
<Composer isThread isEditing={false} channelId="abc" showFormatting />

// Prefer: explicit composition, no conditionals to reason about
<Composer.Frame>
  <Composer.Input />
  <AlsoSendToChannelField id={channelId} />
  <Composer.Footer>
    <Composer.Formatting />
    <Composer.Submit />
  </Composer.Footer>
</Composer.Frame>
```

## 2. Compound components with a shared context

Structure a complex component as a set of subcomponents that read shared state
from context — not from props drilled through a monolithic parent. Consumers
compose only the pieces they need.

```tsx
const ComposerContext = createContext<ComposerContextValue | null>(null);

function ComposerInput() {
  const { state, actions, meta } = use(ComposerContext);
  return (
    <TextInput
      ref={meta.inputRef}
      value={state.input}
      onChangeText={(t) => actions.update((s) => ({ ...s, input: t }))}
    />
  );
}

// Export the pieces as one namespaced object
const Composer = {
  Provider: ComposerProvider,
  Frame: ComposerFrame,
  Input: ComposerInput,
  Footer: ComposerFooter,
  Submit: ComposerSubmit,
};
```

## 3. Lift state into a provider

Move state into a dedicated provider so siblings *outside* the main UI can read
and act on it — no prop drilling, no `useEffect` to sync upward, no reading from
a ref on submit.

```tsx
function ForwardMessageProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState(initialState);
  const forwardMessage = useForwardMessage();
  return (
    <Composer.Provider state={state}
      actions={{ update: setState, submit: forwardMessage }}>
      {children}
    </Composer.Provider>
  );
}

// A button visually OUTSIDE the composer can still submit — it just needs to be
// inside the provider.
function ForwardButton() {
  const { actions } = use(ComposerContext);
  return <Button onPress={actions.submit}>Forward</Button>;
}
```

Components that share state don't need to be visually nested — only within the
same provider.

## 4. Define a generic context interface (`state` / `actions` / `meta`)

Make the context a contract any provider can implement. UI components depend on
the *interface*, never a specific hook — so the same UI works with different
state backends (local, global, server-synced).

```tsx
interface ComposerState { input: string; attachments: Attachment[] }
interface ComposerActions {
  update: (fn: (s: ComposerState) => ComposerState) => void;
  submit: () => void;
}
interface ComposerMeta { inputRef: React.RefObject<TextInput> }

interface ComposerContextValue {
  state: ComposerState;
  actions: ComposerActions;
  meta: ComposerMeta;
}
```

## 5. Decouple state management from UI

The provider is the **only** place that knows *how* state is managed. The same
`Composer.Input` works whether state comes from `useState`, Zustand, or a server
sync — swap the provider, keep the UI.

```tsx
// Local, ephemeral
function ForwardProvider({ children }) {
  const [state, setState] = useState(initial);
  return <Composer.Provider state={state}
    actions={{ update: setState, submit: forward }}>{children}</Composer.Provider>;
}

// Global, synced — same interface, same UI
function ChannelProvider({ channelId, children }) {
  const { state, update, submit } = useGlobalChannel(channelId);
  return <Composer.Provider state={state}
    actions={{ update, submit }}>{children}</Composer.Provider>;
}
```

## 6. Explicit variants over modes

Prefer named variant components over one component driven by flags. Each variant
composes the pieces it needs and documents itself.

```tsx
// Avoid: <Composer isThread channelId="abc" showFormatting={false} />
<ThreadComposer channelId="abc" />
<EditMessageComposer messageId="xyz" />
<ForwardMessageComposer messageId="123" />
```

Each variant is explicit about its provider/state, its UI pieces, and its
actions — no impossible states to reason about.

## 7. Prefer children over render props

Use `children` for composing static structure; it's more readable and composes
naturally. Reserve `renderX` props for when the parent must pass data *back* to
the child.

```tsx
// Avoid: renderHeader / renderFooter / renderActions callbacks
// Prefer: children
<Composer.Frame>
  <CustomHeader />
  <Composer.Input />
  <Composer.Footer>
    <Composer.Formatting />
    <SubmitButton />
  </Composer.Footer>
</Composer.Frame>

// Render props still right when the child needs parent data:
<List data={items} renderItem={({ item }) => <Item item={item} />} />
```

## 8. React 19 APIs (React 19+ only — skip on 18)

- `ref` is a regular prop; drop the `forwardRef` wrapper.
- `use(Context)` replaces `useContext(Context)` and can be called conditionally.

```tsx
function ComposerInput({ ref, ...props }: Props & { ref?: React.Ref<TextInput> }) {
  return <TextInput ref={ref} {...props} />;
}

const value = use(ComposerContext); // not useContext(ComposerContext)
```

## Common mistakes to avoid

1. Adding "just one more" boolean prop instead of splitting into composed pieces.
2. A monolithic parent with `renderHeader`/`renderFooter` render-prop slots.
3. `useEffect` to sync child state up to a parent — lift it into a provider.
4. UI components importing a concrete state hook instead of the context interface.
5. Compound-component ceremony on a genuinely trivial component (it's overkill —
   reach for these patterns when the component is actually growing).
6. `forwardRef`/`useContext` on a React 19 codebase.

## Checklist before "done"

- No behavior-customizing boolean/mode props; variants are explicit components.
- Complex components are compound (subcomponents read a shared context).
- State lifted into a provider; the provider owns the implementation; UI depends
  only on the `state`/`actions`/`meta` interface.
- `children` used for static composition; render props only for passing data back.
- React 19: `ref` as a prop and `use()` over `useContext()` (if on 19+).
- `pnpm lint` and `pnpm typecheck` pass.

## Attribution

Adapted from
[`vercel-labs/agent-skills`](https://github.com/vercel-labs/agent-skills)
(`skills/composition-patterns`), © Vercel Engineering, MIT License.
