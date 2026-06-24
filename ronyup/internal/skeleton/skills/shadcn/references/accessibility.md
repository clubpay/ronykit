# shadcn/ui Accessibility — what Radix gives you, and what you still owe

shadcn components are built on **Radix UI primitives**, which implement the
WAI-ARIA patterns for you: keyboard navigation, focus management, ARIA wiring,
and screen-reader announcements. This file is the component-specific layer —
what each primitive handles automatically and the gaps you must still fill.

For the general, framework-agnostic rules (contrast, color-not-alone, motion,
forms, navigation), see `ux-quality`. This file is the shadcn/Radix specifics.

## What you get for free (don't reinvent it)

| Component        | Radix handles                                                        |
| ---------------- | -------------------------------------------------------------------- |
| `Dialog`/`Sheet` | Focus trap, initial focus, `Esc` to close, focus returns to trigger  |
| `DropdownMenu`   | `Space`/`Enter` open, arrow-key roving focus, `Esc` close, typeahead |
| `Select`         | Listbox semantics, arrow/typeahead nav, keyboard open/close          |
| `Tabs`           | `role=tablist`, arrow-key navigation, `aria-selected`, panel linking |
| `Accordion`      | `aria-expanded`/`aria-controls`, content hidden when collapsed       |
| `Command`        | Filter-as-you-type, arrow nav, `Enter` select, `Esc` close           |
| `Tooltip`/`Popover` | Hover+focus triggers, dismiss-on-`Esc`, ARIA description          |

So: don't add your own `onKeyDown` arrow handling or focus traps to these —
you'll fight the primitive. Compose the primitive instead.

## What you still owe

### Name icon-only triggers

Radix wires the relationships, but it can't invent a label for a bare icon.

```tsx
<Button variant="ghost" size="icon" aria-label="Delete item">
  <Trash className="h-4 w-4" />
</Button>
// or visually-hidden text:
<Button><Trash className="h-4 w-4" /><span className="sr-only">Delete item</span></Button>
```

`SelectTrigger`, `DialogTrigger`, etc. that contain only an icon need the same.

### Forms — use the Form primitives, wire the error

`Field`/`FormField` + `FormMessage` connect label, control, description, and
error (`aria-invalid`, `aria-describedby`, `id`) for you. Use them rather than
ad-hoc `<div>`s so errors are announced.

```tsx
<FormField control={form.control} name="email" render={({ field, fieldState }) => (
  <FormItem>
    <FormLabel>Email</FormLabel>
    <FormControl>
      <Input {...field} aria-invalid={!!fieldState.error} />
    </FormControl>
    <FormDescription>We only use it for receipts.</FormDescription>
    <FormMessage />   {/* announced; links via aria-describedby */}
  </FormItem>
)} />
```

Mark required fields visibly *and* for SR (`<span className="sr-only">(required)</span>`),
and group related fields in a `<fieldset>` with a `<legend>`.

### Keep the focus ring

shadcn ships `focus-visible:ring-2 focus-visible:ring-ring
focus-visible:ring-offset-2` — keep it. Never `focus:outline-none` without a
visible replacement.

### Announce your own async updates

Toasts (`sonner` / toast) include a polite live region. For other dynamic
status text you render yourself, add one:

```tsx
<div aria-live="polite" aria-atomic="true">{statusMessage}</div>
<div aria-live="assertive">{errorMessage}</div>
```

### Skip link + landmarks

```tsx
<a href="#main" className="sr-only focus:not-sr-only focus:absolute focus:top-4 focus:left-4 focus:z-50 focus:rounded focus:bg-background focus:px-4 focus:py-2">
  Skip to content
</a>
<main id="main">…</main>
```

### Respect reduced motion

```tsx
<div className="transition-all motion-reduce:transition-none" />
```

## Quick checklist (shadcn-specific)

- Icon-only `*Trigger`/`Button`s have `aria-label` or `sr-only` text.
- Forms use `FormField`/`FormMessage`; errors set `aria-invalid` and are
  announced; required marked for SR; related fields in `fieldset`/`legend`.
- Focus ring intact (`focus-visible:ring-*`); no `outline-none` without a
  replacement.
- Custom async status uses an `aria-live` region (toasts already do).
- Didn't hand-roll keyboard/focus logic that the Radix primitive already owns.
- Ran the general `ux-quality` checklist too (contrast, color-not-alone, motion).

## Attribution

Adapted from the `ui-styling` skill of
[`ui-ux-pro-max`](https://github.com/nextlevelbuilder/ui-ux-pro-max-skill),
Apache-2.0 License.
