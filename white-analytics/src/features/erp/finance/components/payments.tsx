"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { Ban, Loader2, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { formatDate } from "@/lib/format";
import { toast } from "@/lib/toast";
import { useErpAction } from "@/features/erp/shared/client";
import { ReasonAction } from "@/features/erp/shared/confirm-action";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { FormField } from "@/features/erp/shared/form-field";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { formatRupiah, parseMoney, sumMoney, toMoneyString } from "@/features/erp/shared/money";
import { DatePicker, toIsoDate } from "@/features/erp/shared/period-picker";
import { ErpEmpty } from "@/features/erp/shared/states";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { SearchFilter } from "@/features/erp/shared/url-filters";
import { createPaymentAction, voidPaymentAction } from "../actions";
import type { Aging, Invoice, LedgerKind, Payment, PostableAccount } from "../types";
import type { FinancePerms } from "../perms";
import { fs } from "../strings";
import { AccountPicker } from "./account-picker";
import { PaymentStatusBadge } from "./invoices";

const iv = fs.invoices;

/** Records a payment (AP) or receipt (AR) allocated across open invoices. */
export function PaymentDialog({ kind, openInvoices, accounts }: { kind: LedgerKind; openInvoices: Invoice[]; accounts: PostableAccount[] }) {
  const k = iv[kind];
  const [open, setOpen] = React.useState(false);
  const [party, setParty] = React.useState("");
  const [form, setForm] = React.useState({ paymentDate: toIsoDate(new Date()), method: "TRANSFER", cashAccountId: "", referenceNumber: "", notes: "" });
  const [alloc, setAlloc] = React.useState<Record<string, string>>({});
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const { pending, run } = useErpAction();
  const parties = [...new Set(openInvoices.map((i) => i.partyName))].sort();
  const visible = party ? openInvoices.filter((i) => i.partyName === party) : openInvoices;
  const total = sumMoney(Object.values(alloc));
  const payFull = (inv: Invoice) => (
    <button
      type="button"
      className="tabular underline decoration-dotted underline-offset-4"
      title={iv.payFull}
      onClick={() => setAlloc((a) => ({ ...a, [inv.id]: toMoneyString(parseMoney(inv.outstandingAmount)) }))}
    >
      {formatRupiah(inv.outstandingAmount)}
    </button>
  );
  const allocInput = (inv: Invoice) => (
    <MoneyInput value={alloc[inv.id] ?? ""} onChange={(v) => setAlloc((a) => ({ ...a, [inv.id]: v }))} />
  );

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const allocations = Object.entries(alloc)
      .filter(([, v]) => parseMoney(v) > 0)
      .map(([invoiceId, amount]) => ({ invoiceId, amount }));
    const over = allocations.find((a) => parseMoney(a.amount) > parseMoney(openInvoices.find((i) => i.id === a.invoiceId)?.outstandingAmount));
    if (!allocations.length) return toast.error(iv.allocationRequired);
    if (over) return toast.error(iv.allocationTooHigh);
    void run(() => createPaymentAction(kind, { ...form, allocations }), {
      success: fs.saved,
      onOk: (res) => {
        if (res.note) toast.info(res.note);
        setOpen(false);
      },
      onFail: (r) => setErrors(r.fieldErrors ?? {}),
    });
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (o) {
          setAlloc({});
          setErrors({});
          setParty("");
        }
      }}
    >
      <DialogTrigger asChild>
        <Button size="sm">
          <Plus className="size-4" /> {k.newPayment}
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-3xl">
        <form onSubmit={submit} className="grid gap-4">
          <DialogHeader>
            <DialogTitle>{k.newPayment}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 sm:grid-cols-3">
            <FormField label={fs.date} error={errors.paymentDate} required>
              <DatePicker value={form.paymentDate} onChange={(v) => setForm({ ...form, paymentDate: v })} />
            </FormField>
            <FormField label={iv.method}>
              <Select value={form.method} onValueChange={(v) => setForm({ ...form, method: v })}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(iv.methods).map(([v, label]) => (
                    <SelectItem key={v} value={v}>
                      {label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>
            <FormField label={iv.referenceNumber} htmlFor="pay-ref">
              <Input id="pay-ref" value={form.referenceNumber} onChange={(e) => setForm({ ...form, referenceNumber: e.target.value })} />
            </FormField>
          </div>
          <FormField label={fs.cash.cashAccount} hint={fs.mappingDefault}>
            <AccountPicker accounts={accounts} filter={(a) => a.isCashBank} value={form.cashAccountId} onChange={(v) => setForm({ ...form, cashAccountId: v })} clearable />
          </FormField>

          <div className="flex flex-wrap items-center justify-between gap-2">
            <h3 className="text-sm font-semibold">{iv.allocations}</h3>
            <Select value={party || "__all__"} onValueChange={(v) => setParty(v === "__all__" ? "" : v)}>
              <SelectTrigger size="sm" className="min-w-48" aria-label={k.party}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">{iv.allParties(k.party)}</SelectItem>
                {parties.map((p) => (
                  <SelectItem key={p} value={p}>
                    {p}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {visible.length === 0 ? (
            <ErpEmpty compact title={iv.noOpenInvoices} />
          ) : (
            <>
              {/* Phones: one block per invoice so the allocation input stays in view. */}
              <ul className="max-h-80 divide-y overflow-y-auto rounded-lg border md:hidden">
                {visible.map((inv) => (
                  <li key={inv.id} className="grid gap-2 px-3 py-2.5 text-sm">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <p className="truncate font-mono text-xs">{inv.invoiceNo}</p>
                        <p className="truncate" title={inv.partyName}>
                          {inv.partyName}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {iv.dueDate} {formatDate(inv.dueDate)}
                        </p>
                      </div>
                      <div className="shrink-0 text-right text-xs text-muted-foreground">
                        {iv.outstanding}
                        <div className="text-sm text-foreground">{payFull(inv)}</div>
                      </div>
                    </div>
                    {allocInput(inv)}
                  </li>
                ))}
              </ul>
              <div className="hidden max-h-72 overflow-auto rounded-lg border scrollbar-thin md:block">
                <table className="w-full min-w-[640px] text-sm">
                  <thead className="sticky top-0 z-10 bg-card text-xs text-muted-foreground">
                    <tr>
                      <th className="px-3 py-2 text-left font-medium">{iv.invoiceNo}</th>
                      <th className="px-3 py-2 text-left font-medium">{k.party}</th>
                      <th className="px-3 py-2 text-left font-medium">{iv.dueDate}</th>
                      <th className="px-3 py-2 text-right font-medium">{iv.outstanding}</th>
                      <th className="w-48 px-3 py-2 text-right font-medium">{iv.allocate}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {visible.map((inv) => (
                      <tr key={inv.id} className="border-t">
                        <td className="px-3 py-1.5 font-mono text-xs">{inv.invoiceNo}</td>
                        <td className="px-3 py-1.5">{inv.partyName}</td>
                        <td className="px-3 py-1.5">{formatDate(inv.dueDate)}</td>
                        <td className="px-3 py-1.5 text-right">{payFull(inv)}</td>
                        <td className="px-3 py-1.5">{allocInput(inv)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </>
          )}
          <div className="flex items-center justify-between rounded-lg bg-muted/40 px-4 py-2.5 text-sm">
            <span className="text-muted-foreground">{iv.paymentTotal}</span>
            <span className="text-base font-semibold tabular">{formatRupiah(total)}</span>
          </div>
          <FormField label={iv.notes} htmlFor="pay-notes">
            <Textarea id="pay-notes" rows={2} value={form.notes} onChange={(e) => setForm({ ...form, notes: e.target.value })} />
          </FormField>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
              {fs.cancel}
            </Button>
            <Button type="submit" disabled={pending || total <= 0}>
              {pending ? <Loader2 className="size-4 animate-spin" /> : null}
              {fs.save}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export function PaymentsTable({ kind, payments, page, perms }: { kind: LedgerKind; payments: Payment[]; page: ServerPage; perms: FinancePerms }) {
  const k = iv[kind];
  const { run } = useErpAction();
  const columns: ColumnDef<Payment, unknown>[] = [
    { accessorKey: "paymentNo", header: k.paymentNo, meta: { mono: true } },
    { accessorKey: "paymentDate", header: fs.date, cell: ({ row }) => formatDate(row.original.paymentDate) },
    { accessorKey: "partyName", header: k.party },
    {
      id: "allocations",
      header: iv.allocations,
      accessorFn: (r) => r.allocations.map((a) => a.invoiceNo).join(", "),
      cell: ({ row }) => (
        <ul className="text-xs">
          {row.original.allocations.map((a) => (
            <li key={a.invoiceId}>
              <span className="font-mono">{a.invoiceNo}</span> · {formatRupiah(a.amount)}
            </li>
          ))}
        </ul>
      ),
    },
    {
      id: "method",
      header: iv.method,
      accessorFn: (r) => r.method,
      cell: ({ row }) => (
        <span className="text-xs">
          {iv.methods[row.original.method] ?? row.original.method}
          {row.original.cashAccountName ? <span className="block text-muted-foreground">{row.original.cashAccountName}</span> : null}
        </span>
      ),
    },
    { accessorKey: "amount", header: fs.amount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.amount} /> },
    {
      accessorKey: "status",
      header: fs.status,
      cell: ({ row }) => (
        <span>
          <ErpStatusBadge status={row.original.status} />
          {row.original.voidReason ? <span className="mt-0.5 block max-w-48 text-xs text-muted-foreground">{row.original.voidReason}</span> : null}
        </span>
      ),
    },
    {
      id: "actions",
      header: fs.actions,
      enableSorting: false,
      meta: { align: "right" },
      cell: ({ row }) => {
        const p = row.original;
        return (
          <span className="flex justify-end gap-0.5">
            {p.journalEntryId ? (
              <Button asChild size="xs" variant="ghost">
                <Link href={`/erp/finance/journals/${p.journalEntryId}`}>{iv.journal}</Link>
              </Button>
            ) : null}
            {perms.canApprove && p.status === "POSTED" ? (
              <ReasonAction
                trigger={
                  <Button size="xs" variant="ghost" className="text-destructive">
                    <Ban className="size-3.5" /> {iv.void}
                  </Button>
                }
                title={iv.voidTitle}
                description={iv.voidDesc}
                reasonLabel={iv.voidReason}
                confirmLabel={iv.void}
                onConfirm={async (reason) => (await run(() => voidPaymentAction(kind, p.id, reason), { success: iv.voided })).ok}
              />
            ) : null}
          </span>
        );
      },
    },
  ];
  return <ErpDataTable columns={columns} data={payments} pagination={page} emptyMessage={iv.emptyPayments} toolbar={<SearchFilter placeholder={iv.searchPh} />} />;
}

const BUCKET_ORDER = ["current", "1_30", "31_60", "61_90", "90_plus"];

export function AgingView({ kind, aging }: { kind: LedgerKind; aging: Aging }) {
  const k = iv[kind];
  const buckets = [...aging.buckets].sort((a, b) => BUCKET_ORDER.indexOf(a.bucket) - BUCKET_ORDER.indexOf(b.bucket));
  const columns: ColumnDef<Invoice, unknown>[] = [
    { accessorKey: "invoiceNo", header: iv.invoiceNo, meta: { mono: true } },
    { accessorKey: "partyName", header: k.party },
    { accessorKey: "invoiceDate", header: iv.invoiceDate, cell: ({ row }) => formatDate(row.original.invoiceDate) },
    { accessorKey: "dueDate", header: iv.dueDate, cell: ({ row }) => formatDate(row.original.dueDate) },
    { accessorKey: "totalAmount", header: iv.totalAmount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.totalAmount} /> },
    { accessorKey: "outstandingAmount", header: iv.outstanding, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.outstandingAmount} /> },
    { accessorKey: "paymentStatus", header: iv.paymentStatus, cell: ({ row }) => <PaymentStatusBadge status={row.original.paymentStatus} /> },
  ];
  return (
    <div className="flex flex-col gap-4">
      <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-6">
        {buckets.map((b) => (
          <Card key={b.bucket} className="gap-1 px-4 py-3">
            <span className="label-mono text-muted-foreground">{iv.buckets[b.bucket] ?? b.bucket}</span>
            <Money value={b.amount} className="text-lg font-semibold" />
            <span className="text-xs text-muted-foreground">{iv.count(b.invoiceCount)}</span>
          </Card>
        ))}
        <Card className="gap-1 bg-brand/5 px-4 py-3">
          <span className="label-mono text-muted-foreground">{iv.totalOutstanding}</span>
          <Money value={aging.totalOutstanding} className="text-lg font-semibold" />
        </Card>
      </div>
      <ErpDataTable columns={columns} data={aging.invoices} emptyMessage={iv.noOpenInvoices} exportName={`umur-${kind.toLowerCase()}-${aging.asOf}`} />
    </div>
  );
}
