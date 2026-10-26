"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { CheckCheck, Loader2, Pencil, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { formatDate } from "@/lib/format";
import { useErpAction } from "@/features/erp/shared/client";
import { ConfirmAction } from "@/features/erp/shared/confirm-action";
import { ErpDataTable, type ServerPage } from "@/features/erp/shared/data-table";
import { FormField } from "@/features/erp/shared/form-field";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { formatRupiah, parseMoney } from "@/features/erp/shared/money";
import { DatePicker, toIsoDate } from "@/features/erp/shared/period-picker";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { DateRangeFilter } from "@/features/erp/shared/period-picker";
import { ResetFilters, SearchFilter, SearchSelectFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { toast } from "@/lib/toast";
import { deleteInvoiceAction, postInvoiceAction, saveInvoiceAction } from "../actions";
import type { Invoice, LedgerKind, PostableAccount } from "../types";
import type { FinancePerms } from "../perms";
import { fs } from "../strings";
import { AccountPicker } from "./account-picker";

const iv = fs.invoices;

export function PaymentStatusBadge({ status }: { status: string }) {
  if (!status) return null;
  return (
    <ErpStatusBadge
      status={status.toUpperCase()}
      labels={Object.fromEntries(Object.entries(iv.payStatus).map(([k, v]) => [k.toUpperCase(), v]))}
      kinds={{ PAID: "good", PARTIAL: "warning", OVERDUE: "critical", UNPAID: "neutral" }}
    />
  );
}

function addDays(iso: string, n: number) {
  const d = new Date(`${iso}T00:00:00`);
  d.setDate(d.getDate() + n);
  return toIsoDate(d);
}

export function InvoiceDialog({
  kind,
  invoice,
  accounts,
  onClose,
}: {
  kind: LedgerKind;
  invoice?: Invoice;
  accounts: PostableAccount[];
  onClose: () => void;
}) {
  const k = iv[kind];
  const today = toIsoDate(new Date());
  const [form, setForm] = React.useState({
    invoiceDate: invoice?.invoiceDate ?? today,
    dueDate: invoice?.dueDate ?? addDays(today, 30),
    partyName: invoice?.partyName ?? "",
    reference: invoice?.reference ?? "",
    accountId: invoice?.accountId ?? "",
    subtotal: invoice?.subtotal ?? "",
    taxAmount: invoice?.taxAmount ?? "0.00",
    description: invoice?.description ?? "",
    clientId: invoice?.clientId ?? "",
  });
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const { pending, run } = useErpAction();
  const set = <K extends keyof typeof form>(key: K, v: (typeof form)[K]) => setForm((f) => ({ ...f, [key]: v }));
  const total = parseMoney(form.subtotal) + parseMoney(form.taxAmount);
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-xl">
        <form
          className="grid gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            void run(() => saveInvoiceAction(kind, invoice?.id ?? null, { ...form, taxAmount: form.taxAmount || "0.00" }), {
              success: fs.saved,
              onOk: onClose,
              onFail: (r) => setErrors(r.fieldErrors ?? {}),
            });
          }}
        >
          <DialogHeader>
            <DialogTitle>{invoice ? iv.editTitle : k.newInvoice}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField label={k.party} htmlFor="inv-party" error={errors.partyName} required>
              <Input id="inv-party" value={form.partyName} onChange={(e) => set("partyName", e.target.value)} />
            </FormField>
            <FormField label={k.reference} htmlFor="inv-ref" error={errors.reference}>
              <Input id="inv-ref" value={form.reference} onChange={(e) => set("reference", e.target.value)} />
            </FormField>
            <FormField label={iv.invoiceDate} error={errors.invoiceDate} required>
              <DatePicker value={form.invoiceDate} onChange={(v) => set("invoiceDate", v)} />
            </FormField>
            <FormField label={iv.dueDate} error={errors.dueDate} required>
              <DatePicker value={form.dueDate} onChange={(v) => set("dueDate", v)} />
            </FormField>
          </div>
          <FormField label={k.account} error={errors.accountId} hint={fs.mappingDefault}>
            <AccountPicker accounts={accounts} filter={(a) => !a.isCashBank} value={form.accountId} onChange={(v) => set("accountId", v)} clearable />
          </FormField>
          <div className="grid gap-4 sm:grid-cols-3">
            <FormField label={iv.subtotal} error={errors.subtotal} required>
              <MoneyInput value={form.subtotal} onChange={(v) => set("subtotal", v)} invalid={!!errors.subtotal} />
            </FormField>
            <FormField label={iv.tax} error={errors.taxAmount}>
              <MoneyInput value={form.taxAmount} onChange={(v) => set("taxAmount", v)} />
            </FormField>
            <FormField label={iv.totalAmount}>
              <div className="flex h-9 items-center justify-end rounded-md border bg-muted/40 px-3 font-semibold tabular">{formatRupiah(total)}</div>
            </FormField>
          </div>
          <FormField label={fs.description} htmlFor="inv-desc">
            <Textarea id="inv-desc" rows={2} value={form.description} onChange={(e) => set("description", e.target.value)} />
          </FormField>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={onClose}>
              {fs.cancel}
            </Button>
            <Button type="submit" disabled={pending}>
              {pending ? <Loader2 className="size-4 animate-spin" /> : null}
              {fs.saveDraft}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

export function InvoicesTable({
  kind,
  invoices,
  page,
  accounts,
  parties,
  perms,
}: {
  kind: LedgerKind;
  invoices: Invoice[];
  page: ServerPage;
  accounts: PostableAccount[];
  /** party names for the party filter */
  parties: string[];
  perms: FinancePerms;
}) {
  const k = iv[kind];
  const [dialog, setDialog] = React.useState<{ invoice?: Invoice } | null>(null);
  const { run } = useErpAction();
  const columns: ColumnDef<Invoice, unknown>[] = [
    {
      accessorKey: "invoiceNo",
      header: iv.invoiceNo,
      cell: ({ row }) => (
        <span>
          <span className="block font-mono text-xs font-medium">{row.original.invoiceNo || "–"}</span>
          {row.original.reference ? <span className="text-xs text-muted-foreground">{row.original.reference}</span> : null}
        </span>
      ),
    },
    { accessorKey: "partyName", header: k.party },
    { accessorKey: "invoiceDate", header: iv.invoiceDate, cell: ({ row }) => formatDate(row.original.invoiceDate) },
    {
      accessorKey: "dueDate",
      header: iv.dueDate,
      cell: ({ row }) => (
        <span className={cn(row.original.paymentStatus === "overdue" && "text-negative")}>{formatDate(row.original.dueDate)}</span>
      ),
    },
    { accessorKey: "totalAmount", header: iv.totalAmount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.totalAmount} /> },
    {
      accessorKey: "outstandingAmount",
      header: iv.outstanding,
      meta: { align: "right" },
      cell: ({ row }) => <Money value={row.original.outstandingAmount} muteZero />,
    },
    {
      accessorKey: "status",
      header: fs.status,
      cell: ({ row }) => (
        <span className="flex flex-wrap gap-1">
          <ErpStatusBadge status={row.original.status} />
          {row.original.status === "POSTED" ? <PaymentStatusBadge status={row.original.paymentStatus} /> : null}
        </span>
      ),
    },
    {
      id: "actions",
      header: fs.actions,
      enableSorting: false,
      meta: { align: "right" },
      cell: ({ row }) => {
        const inv = row.original;
        return (
          <span className="flex justify-end gap-0.5">
            {inv.journalEntryId ? (
              <Button asChild size="xs" variant="ghost">
                <Link href={`/erp/finance/journals/${inv.journalEntryId}`}>{iv.journal}</Link>
              </Button>
            ) : null}
            {inv.status === "DRAFT" && perms.canWrite ? (
              <>
                <Button variant="ghost" size="icon-xs" aria-label={fs.edit} onClick={() => setDialog({ invoice: inv })}>
                  <Pencil />
                </Button>
                <ConfirmAction
                  trigger={
                    <Button variant="ghost" size="icon-xs" aria-label={fs.delete} className="text-destructive">
                      <Trash2 />
                    </Button>
                  }
                  title={fs.confirmDeleteTitle}
                  description={iv.deleteDesc}
                  confirmLabel={fs.delete}
                  destructive
                  onConfirm={async () => (await run(() => deleteInvoiceAction(kind, inv.id), { success: fs.deleted })).ok}
                />
              </>
            ) : null}
            {inv.status === "DRAFT" && perms.canApprove ? (
              <ConfirmAction
                trigger={
                  <Button size="xs" variant="outline">
                    <CheckCheck className="size-3.5" /> {iv.post}
                  </Button>
                }
                title={iv.postTitle}
                description={iv.postDesc}
                confirmLabel={iv.post}
                onConfirm={async () => {
                  const res = await run(() => postInvoiceAction(kind, inv.id), { success: iv.posted });
                  if (res.ok && res.data.note) toast.info(res.data.note);
                  return res.ok;
                }}
              />
            ) : null}
          </span>
        );
      },
    },
  ];
  return (
    <>
      <ErpDataTable
        columns={columns}
        data={invoices}
        pagination={page}
        emptyMessage={iv.empty}
        toolbar={
          <>
            <SearchFilter placeholder={iv.searchPh} />
            <SelectFilter
              param="status"
              label={fs.status}
              allLabel={fs.status}
              options={[
                { value: "DRAFT", label: "Draf" },
                { value: "POSTED", label: "Diposting" },
                { value: "VOID", label: "Dibatalkan" },
              ]}
            />
            <SelectFilter
              param="pay"
              label={iv.paymentStatus}
              allLabel={iv.allPayStatus}
              options={Object.entries(iv.payStatus).map(([value, label]) => ({ value, label }))}
            />
            <DateRangeFilter placeholder={iv.dateRange} />
            {parties.length ? (
              <SearchSelectFilter
                param="party"
                placeholder={k.party}
                options={parties.map((p) => ({ value: p, label: p }))}
                className="w-56"
              />
            ) : null}
            <ResetFilters params={["q", "status", "pay", "from", "to", "party"]} />
            {perms.canWrite ? (
              <Button size="sm" className="ml-auto" onClick={() => setDialog({})}>
                <Plus className="size-4" /> {k.newInvoice}
              </Button>
            ) : null}
          </>
        }
      />
      {dialog ? <InvoiceDialog kind={kind} invoice={dialog.invoice} accounts={accounts} onClose={() => setDialog(null)} /> : null}
    </>
  );
}
