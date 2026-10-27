"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { ArrowDownLeft, ArrowLeftRight, ArrowRight, ArrowUpRight, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { formatDate } from "@/lib/format";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { FormField } from "@/features/erp/shared/form-field";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker, DateRangeFilter, toIsoDate } from "@/features/erp/shared/period-picker";
import { ErpEmpty } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { SearchFilter } from "@/features/erp/shared/url-filters";
import { createCashMovementAction, createCashTransferAction } from "../actions";
import type { CashBankAccount, CashMovement, CashTransfer, PostableAccount } from "../types";
import { fs } from "../strings";
import { AccountPicker } from "./account-picker";

const c = fs.cash;

export function CashAccountCards({ accounts, total }: { accounts: CashBankAccount[]; total: string }) {
  if (accounts.length === 0) return <ErpEmpty title={c.noAccounts} />;
  return (
    <div className="page-stack">
      <Card className="flex-row items-center justify-between px-(--card-pad)">
        <span className="label-mono text-muted-foreground">{c.totalBalance}</span>
        <Money value={total} signed className="text-2xl font-bold" />
      </Card>
      <div className="grid gap-(--gap) sm:grid-cols-2 xl:grid-cols-3">
        {accounts.map((a) => (
          <Link
            key={a.id}
            href={`/erp/finance/cash-bank/${a.id}`}
            className="group lift flex flex-col gap-3 rounded-[1.5rem] bg-card p-(--card-pad) shadow-(--card-shadow) outline-none focus-visible:ring-2 focus-visible:ring-ring"
          >
            <div className="flex items-start justify-between gap-2">
              <div className="min-w-0">
                <p className="truncate font-medium">{a.name}</p>
                <p className="font-mono text-xs text-muted-foreground">{a.codeDisplay || a.code}</p>
              </div>
              <ArrowRight className="size-4 text-muted-foreground transition-transform group-hover:translate-x-0.5" />
            </div>
            <Money value={a.balance} signed className="text-xl font-semibold" />
            <span className="text-xs text-muted-foreground">{c.movements(a.movementCount)}</span>
          </Link>
        ))}
      </div>
    </div>
  );
}

function useFormState<T extends object>(initial: T) {
  const [form, setForm] = React.useState(initial);
  const set = <K extends keyof T>(k: K, v: T[K]) => setForm((f) => ({ ...f, [k]: v }));
  return [form, set, setForm] as const;
}

export function CashMovementDialog({ kind, accounts }: { kind: "CASH_IN" | "CASH_OUT"; accounts: PostableAccount[] }) {
  const [open, setOpen] = React.useState(false);
  const blank = { entryDate: toIsoDate(new Date()), amount: "", cashAccountId: "", offsetAccountId: "", description: "", memo: "" };
  const [form, set, setForm] = useFormState(blank);
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const { pending, run } = useErpAction();
  const isIn = kind === "CASH_IN";
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (o) {
          setForm(blank);
          setErrors({});
        }
      }}
    >
      <DialogTrigger asChild>
        <Button size="sm" variant={isIn ? "default" : "outline"}>
          {isIn ? <ArrowDownLeft className="size-4" /> : <ArrowUpRight className="size-4" />}
          {isIn ? c.newCashIn : c.newCashOut}
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <form
          className="grid gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            void run(() => createCashMovementAction({ kind, ...form }), {
              success: c.created,
              onOk: () => setOpen(false),
              onFail: (r) => setErrors(r.fieldErrors ?? {}),
            });
          }}
        >
          <DialogHeader>
            <DialogTitle>{isIn ? c.newCashIn : c.newCashOut}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField label={fs.date} error={errors.entryDate} required>
              <DatePicker value={form.entryDate} onChange={(v) => set("entryDate", v)} />
            </FormField>
            <FormField label={fs.amount} error={errors.amount} required>
              <MoneyInput value={form.amount} onChange={(v) => set("amount", v)} invalid={!!errors.amount} />
            </FormField>
          </div>
          <FormField label={c.cashAccount} error={errors.cashAccountId} required>
            <AccountPicker accounts={accounts} filter={(a) => a.isCashBank} value={form.cashAccountId} onChange={(v) => set("cashAccountId", v)} />
          </FormField>
          <FormField label={isIn ? c.offsetIn : c.offsetOut} error={errors.offsetAccountId} required>
            <AccountPicker accounts={accounts} filter={(a) => !a.isCashBank} value={form.offsetAccountId} onChange={(v) => set("offsetAccountId", v)} />
          </FormField>
          <FormField label={fs.description} htmlFor="cm-desc" error={errors.description} required>
            <Textarea id="cm-desc" rows={2} value={form.description} onChange={(e) => set("description", e.target.value)} />
          </FormField>
          <FormField label={fs.memo} htmlFor="cm-memo">
            <Input id="cm-memo" value={form.memo} onChange={(e) => set("memo", e.target.value)} />
          </FormField>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              {fs.cancel}
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? <Loader2 className="size-4 animate-spin" /> : null}
              {fs.save}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export function CashTransferDialog({ accounts }: { accounts: PostableAccount[] }) {
  const [open, setOpen] = React.useState(false);
  const blank = { entryDate: toIsoDate(new Date()), amount: "", fromAccountId: "", toAccountId: "", description: "", memo: "" };
  const [form, set, setForm] = useFormState(blank);
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const { pending, run } = useErpAction();
  const cashOnly = (a: PostableAccount) => a.isCashBank;
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (o) {
          setForm(blank);
          setErrors({});
        }
      }}
    >
      <DialogTrigger asChild>
        <Button size="sm" variant="outline">
          <ArrowLeftRight className="size-4" /> {c.newTransfer}
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <form
          className="grid gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            void run(() => createCashTransferAction(form), {
              success: c.created,
              onOk: () => setOpen(false),
              onFail: (r) => setErrors(r.fieldErrors ?? {}),
            });
          }}
        >
          <DialogHeader>
            <DialogTitle>{c.transfer}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField label={fs.date} error={errors.entryDate} required>
              <DatePicker value={form.entryDate} onChange={(v) => set("entryDate", v)} />
            </FormField>
            <FormField label={fs.amount} error={errors.amount} required>
              <MoneyInput value={form.amount} onChange={(v) => set("amount", v)} invalid={!!errors.amount} />
            </FormField>
          </div>
          <FormField label={c.from} error={errors.fromAccountId} required>
            <AccountPicker accounts={accounts} filter={cashOnly} value={form.fromAccountId} onChange={(v) => set("fromAccountId", v)} />
          </FormField>
          <FormField label={c.to} error={errors.toAccountId} required>
            <AccountPicker accounts={accounts} filter={cashOnly} value={form.toAccountId} onChange={(v) => set("toAccountId", v)} />
          </FormField>
          <FormField label={fs.description} htmlFor="ct-desc" error={errors.description} required>
            <Textarea id="ct-desc" rows={2} value={form.description} onChange={(e) => set("description", e.target.value)} />
          </FormField>
          <FormField label={fs.memo} htmlFor="ct-memo">
            <Input id="ct-memo" value={form.memo} onChange={(e) => set("memo", e.target.value)} />
          </FormField>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              {fs.cancel}
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? <Loader2 className="size-4 animate-spin" /> : null}
              {fs.save}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

const entryLink = (id: string, no: string) => (
  <Link href={`/erp/finance/journals/${id}`} className="font-mono text-xs font-medium hover:underline">
    {no}
  </Link>
);
const accountLabel = (a: PostableAccount | null) => (a ? `${a.codeDisplay || a.code} ${a.name}` : "–");
const toolbar = (
  <>
    <SearchFilter placeholder={c.searchPh} />
    <DateRangeFilter />
  </>
);

export function MovementsTable({ rows, page, kind }: { rows: CashMovement[]; page: ServerPage; kind: "CASH_IN" | "CASH_OUT" }) {
  const columns: ColumnDef<CashMovement, unknown>[] = [
    { accessorKey: "entryNo", header: fs.journals.entryNo, cell: ({ row }) => entryLink(row.original.id, row.original.entryNo) },
    { accessorKey: "entryDate", header: fs.date, cell: ({ row }) => formatDate(row.original.entryDate) },
    { accessorKey: "description", header: fs.description },
    { id: "cash", header: c.cashAccount, accessorFn: (r) => accountLabel(r.cashAccount) },
    { id: "offset", header: kind === "CASH_IN" ? c.offsetIn : c.offsetOut, accessorFn: (r) => accountLabel(r.offsetAccount) },
    { accessorKey: "amount", header: fs.amount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.amount} /> },
    { accessorKey: "status", header: fs.status, cell: ({ row }) => <ErpStatusBadge status={row.original.status} /> },
  ];
  return <ErpDataTable columns={columns} data={rows} pagination={page} emptyMessage={c.empty} toolbar={toolbar} />;
}

export function TransfersTable({ rows, page }: { rows: CashTransfer[]; page: ServerPage }) {
  const columns: ColumnDef<CashTransfer, unknown>[] = [
    { accessorKey: "entryNo", header: fs.journals.entryNo, cell: ({ row }) => entryLink(row.original.id, row.original.entryNo) },
    { accessorKey: "entryDate", header: fs.date, cell: ({ row }) => formatDate(row.original.entryDate) },
    { accessorKey: "description", header: fs.description },
    { id: "from", header: c.from, accessorFn: (r) => accountLabel(r.fromAccount) },
    { id: "to", header: c.to, accessorFn: (r) => accountLabel(r.toAccount) },
    { accessorKey: "amount", header: fs.amount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.amount} /> },
    { accessorKey: "status", header: fs.status, cell: ({ row }) => <ErpStatusBadge status={row.original.status} /> },
  ];
  return <ErpDataTable columns={columns} data={rows} pagination={page} emptyMessage={c.empty} toolbar={toolbar} />;
}
