# shadcn/ui Forms & Data Tables

Production patterns for the two highest-density surfaces in shadcn/ui apps:
forms (`useActionState` / React Hook Form + Zod + RonyKit `<Service>StubError`)
and data tables (`@tanstack/react-table` + URL state + `tabular-nums`).

---

## 1. Forms (`Field` + Zod + RonyKit `<Service>StubError`)

### Non-negotiable form rules
1. **Visible `<Label>` on every input** — placeholders disappear on type and are not labels.
2. **Validate on blur or submit, clear immediately on change** — never show red errors on the first keystroke while the user is still typing.
3. **Wire `aria-invalid` and `aria-describedby`** — link both helper descriptions and error messages so screen readers announce them.
4. **Never block paste** (`onPaste` preventDefault is forbidden on all inputs, including OTP/password).
5. **Preserve submit button dimensions during pending state** — swap the label or prepend an inline `animate-spin` icon while keeping `min-w-[...]` so the button never shifts layout.

### Server Action + RonyKit `<Service>StubError` mapping

RonyKit's generated `<Service>StubError` exposes `err.item` (the offending field/domain item) and `err.serverMessage` (the human-readable message from `rony/errs`):

```ts
// app/orders/actions.ts
'use server';

import { z } from 'zod';
import { OrderServiceStub, OrderServiceStubError } from '@/stub/orderstub-typescript/stub';

const CreateOrderSchema = z.object({
  customerEmail: z.string().email('Enter a valid customer email address.'),
  amountCents: z.coerce.number().int().positive('Amount must be greater than zero.'),
});

export type CreateOrderState = {
  ok?: boolean;
  message?: string;
  fieldErrors?: Record<string, string[]>;
};

export async function createOrderAction(
  _prev: CreateOrderState,
  formData: FormData,
): Promise<CreateOrderState> {
  const parsed = CreateOrderSchema.safeParse(Object.fromEntries(formData));
  if (!parsed.success) {
    return {
      ok: false,
      fieldErrors: parsed.error.flatten().fieldErrors,
    };
  }

  const stub = new OrderServiceStub(process.env.BACKEND_URL ?? 'http://localhost:8080');
  try {
    await stub.createOrder(parsed.data);
    return { ok: true };
  } catch (err) {
    if (err instanceof OrderServiceStubError) {
      return {
        ok: false,
        fieldErrors: err.item ? { [err.item]: [err.serverMessage] } : undefined,
        message: err.serverMessage,
      };
    }
    return { ok: false, message: 'Unable to create order. Please try again.' };
  }
}
```

### Accessible Client Form (`useActionState` + blur/change error clearing)

```tsx
'use client';

import { useActionState, useState } from 'react';
import { Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { createOrderAction, type CreateOrderState } from './actions';

const initialState: CreateOrderState = {};

export function CreateOrderForm() {
  const [state, formAction, isPending] = useActionState(createOrderAction, initialState);
  const [clearedFields, setClearedFields] = useState<Record<string, boolean>>({});

  const emailError = !clearedFields.customerEmail ? state.fieldErrors?.customerEmail?.[0] : undefined;

  return (
    <form
      action={(fd) => {
        setClearedFields({});
        formAction(fd);
      }}
      className="space-y-4 max-w-md"
      noValidate
    >
      {state.message && !state.fieldErrors && (
        <div
          role="alert"
          className="rounded-md border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive"
        >
          {state.message}
        </div>
      )}

      <div className="space-y-1.5">
        <Label htmlFor="customerEmail">Customer email</Label>
        <Input
          id="customerEmail"
          name="customerEmail"
          type="email"
          autoComplete="email"
          required
          aria-invalid={Boolean(emailError)}
          aria-describedby={emailError ? 'customerEmail-error' : 'customerEmail-hint'}
          onChange={() =>
            setClearedFields((prev) => (prev.customerEmail ? prev : { ...prev, customerEmail: true }))
          }
        />
        <p id="customerEmail-hint" className="text-xs text-muted-foreground">
          Receipt and invoice PDF will be sent here.
        </p>
        {emailError && (
          <p id="customerEmail-error" role="alert" className="text-xs text-destructive">
            {emailError}
          </p>
        )}
      </div>

      <Button
        type="submit"
        disabled={isPending}
        className="min-w-[140px] enabled:active:scale-[0.97]"
      >
        {isPending && <Loader2 className="mr-2 size-4 animate-spin" aria-hidden="true" />}
        <span>{isPending ? 'Creating…' : 'Create order'}</span>
      </Button>
    </form>
  );
}
```

---

## 2. Data Tables (`@tanstack/react-table` + shadcn `Table`)

### Non-negotiable table rules
1. **Cap default visible columns at 5–8** — move secondary metadata to a detail drawer (`Sheet`) or toggleable column visibility menu.
2. **Right-align all numeric/currency/percentage columns (`text-right tabular-nums`)** — both `<TableHead>` and `<TableCell>` must match alignment.
3. **Never wrap primary identifiers or status badges across lines** — use `whitespace-nowrap` on IDs/dates/badges and `max-w-[240px] truncate` on long text columns.
4. **Sticky header + recessed surface** — use `bg-[var(--surface-sunken)]` on `<TableHeader>` so column headers remain distinct from rows.
5. **Distinguish first-use empty vs filtered-zero empty** — when active filters return 0 rows, render a **Reset filters** button directly inside the empty row.

### Table pattern with numeric alignment and filtered empty state

```tsx
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Button } from '@/components/ui/button';

export interface OrderRow {
  id: string;
  customerEmail: string;
  status: 'paid' | 'pending' | 'failed';
  amountFormatted: string;
  createdAt: string;
}

export function OrdersTable({
  rows,
  hasActiveFilters,
  onResetFilters,
}: {
  rows: OrderRow[];
  hasActiveFilters: boolean;
  onResetFilters: () => void;
}) {
  return (
    <div className="rounded-lg border border-border overflow-hidden bg-card">
      <Table>
        <TableHeader className="bg-[var(--surface-sunken)]">
          <TableRow>
            <TableHead className="w-[140px]">Order ID</TableHead>
            <TableHead>Customer</TableHead>
            <TableHead className="w-[120px]">Status</TableHead>
            <TableHead className="w-[140px] text-right">Amount</TableHead>
            <TableHead className="w-[160px] text-right">Created</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.length === 0 ? (
            <TableRow>
              <TableCell colSpan={5} className="h-32 text-center">
                {hasActiveFilters ? (
                  <div className="flex flex-col items-center gap-2">
                    <p className="text-sm text-muted-foreground">
                      No orders match the current filters.
                    </p>
                    <Button variant="outline" size="sm" onClick={onResetFilters}>
                      Reset filters
                    </Button>
                  </div>
                ) : (
                  <p className="text-sm text-muted-foreground">
                    No orders yet. Create your first order to get started.
                  </p>
                )}
              </TableCell>
            </TableRow>
          ) : (
            rows.map((row) => (
              <TableRow key={row.id} className="h-11">
                <TableCell className="font-mono text-xs whitespace-nowrap tabular-nums">
                  {row.id}
                </TableCell>
                <TableCell className="max-w-[260px] truncate">{row.customerEmail}</TableCell>
                <TableCell className="whitespace-nowrap">{row.status}</TableCell>
                <TableCell className="text-right font-medium tabular-nums whitespace-nowrap">
                  {row.amountFormatted}
                </TableCell>
                <TableCell className="text-right text-muted-foreground text-xs tabular-nums whitespace-nowrap">
                  {row.createdAt}
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}
```
