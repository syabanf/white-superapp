"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { Ban, Plus, ReceiptText } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Fld, FormDialog, FormFooter, SimpleSelect } from "@/features/erp/sales/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ReasonAction } from "@/features/erp/shared/confirm-action";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { formatRupiah, parseMoney } from "@/features/erp/shared/money";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker, toIsoDate } from "@/features/erp/shared/period-picker";
import { ResetFilters, SearchFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import { advanceState } from "../advance-state";
import { applyAdvanceAction, createAdvanceAction, voidAdvanceAction } from "../actions";
import { fs } from "../strings";
import type { Advance, Invoice, LedgerKind, PostableAccount } from "../types";
import { AccountPicker } from "./account-picker";

const s = fs.advances;

function AdvanceForm({ kind, accounts, close }: { kind: LedgerKind; accounts: PostableAccount[]; close: () => void }) {
  const defaultAccount = accounts.find((a) => a.code === (kind === "AR" ? "2105001" : "1106001"))?.id ?? "";
  const [v, setV] = React.useState({ kind, advanceDate: toIsoDate(new Date()), partyName: "", amount: "", cashAccountId: "", advanceAccountId: defaultAccount, reference: "", notes: "" });
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  const set = <K extends keyof typeof v>(k: K, x: (typeof v)[K]) => setV((p) => ({ ...p, [k]: x }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => createAdvanceAction(v), { success: s.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={`${s.party[kind]} *`} htmlFor="adv-party" error={errors?.party_name}>
        <Input id="adv-party" value={v.partyName} onChange={(e) => set("partyName", e.target.value)} />
      </Fld>
      <Fld label={`${s.amount} *`} htmlFor="adv-amount" error={errors?.amount}>
        <MoneyInput id="adv-amount" value={v.amount} onChange={(x) => set("amount", x)} />
      </Fld>
      <Fld label={s.date} htmlFor="adv-date" error={errors?.advance_date}>
        <DatePicker id="adv-date" value={v.advanceDate} onChange={(x) => set("advanceDate", x)} />
      </Fld>
      <Fld label={s.reference} htmlFor="adv-ref">
        <Input id="adv-ref" value={v.reference} onChange={(e) => set("reference", e.target.value)} />
      </Fld>
      <Fld label={`${s.cashAccount} *`} htmlFor="adv-cash" className="sm:col-span-2" error={errors?.cash_account_id}>
        <AccountPicker accounts={accounts} filter={(a) => a.isCashBank} value={v.cashAccountId} onChange={(x) => set("cashAccountId", x)} />
      </Fld>
      <Fld label={`${s.advanceAccount} *`} htmlFor="adv-account" hint={s.accountHint[kind]} className="sm:col-span-2" error={errors?.advance_account_id}>
        <AccountPicker accounts={accounts} filter={(a) => !a.isCashBank} value={v.advanceAccountId} onChange={(x) => set("advanceAccountId", x)} />
      </Fld>
      <Fld label={s.notes} htmlFor="adv-notes" className="sm:col-span-2">
        <Input id="adv-notes" value={v.notes} onChange={(e) => set("notes", e.target.value)} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

export function AdvanceDialog({ kind, accounts }: { kind: LedgerKind; accounts: PostableAccount[] }) {
  return (
    <FormDialog
      title={s.new}
      description={s.hint}
      size="lg"
      trigger={
        <Button size="sm">
          <Plus className="size-4" /> {s.new}
        </Button>
      }
    >
      {(close) => <AdvanceForm kind={kind} accounts={accounts} close={close} />}
    </FormDialog>
  );
}

function ApplyForm({ advance, invoices, close }: { advance: Advance; invoices: Invoice[]; close: () => void }) {
  const choices = invoices.filter((i) => i.partyName.toLowerCase() === advance.partyName.toLowerCase());
  const amountFor = (invoiceId: string) => {
    const invoice = choices.find((i) => i.id === invoiceId);
    return invoice && parseMoney(invoice.outstandingAmount) < parseMoney(advance.remaining) ? invoice.outstandingAmount : advance.remaining;
  };
  const [v, setV] = React.useState({ invoiceId: choices[0]?.id ?? "", amount: amountFor(choices[0]?.id ?? ""), date: toIsoDate(new Date()) });
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  if (!choices.length) return <p className="text-sm text-muted-foreground">{s.noInvoices}</p>;
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => applyAdvanceAction(advance.kind, advance.id, v), { success: s.applied, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.invoice} htmlFor="apply-invoice">
        <SimpleSelect
          id="apply-invoice"
          value={v.invoiceId}
          onChange={(x) => setV((p) => ({ ...p, invoiceId: x, amount: amountFor(x) }))}
          options={choices.map((i) => ({ value: i.id, label: `${i.invoiceNo} · ${formatRupiah(parseMoney(i.outstandingAmount))}` }))}
        />
      </Fld>
      <Fld label={s.amount} htmlFor="apply-amount" error={errors?.amount}>
        <MoneyInput id="apply-amount" value={v.amount} onChange={(x) => setV((p) => ({ ...p, amount: x }))} />
      </Fld>
      <Fld label={s.date} htmlFor="apply-date" error={errors?.payment_date}>
        <DatePicker id="apply-date" value={v.date} onChange={(x) => setV((p) => ({ ...p, date: x }))} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

export function AdvancesTable({ kind, rows, openInvoices, canApprove }: { kind: LedgerKind; rows: Advance[]; openInvoices: Invoice[]; canApprove: boolean }) {
  const { run } = useErpAction();
  const columns: ColumnDef<Advance, unknown>[] = [
    {
      accessorKey: "advanceNo",
      header: s.number,
      cell: ({ row }) => (
        <span>
          <span className="block font-mono text-xs font-medium">{row.original.advanceNo}</span>
          {row.original.reference ? <span className="text-xs text-muted-foreground">{row.original.reference}</span> : null}
        </span>
      ),
    },
    { accessorKey: "partyName", header: s.party[kind] },
    { accessorKey: "advanceDate", header: s.date, cell: ({ row }) => formatDate(row.original.advanceDate) },
    { accessorKey: "amount", header: s.amount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.amount} /> },
    { accessorKey: "applied", header: s.appliedCol, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.applied} muteZero /> },
    { accessorKey: "remaining", header: s.remaining, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.remaining} muteZero /> },
    {
      id: "status",
      header: s.status,
      cell: ({ row }) => (
        <span title={row.original.voidReason || undefined}>{s.statuses[advanceState(row.original)]}</span>
      ),
    },
    {
      id: "actions",
      header: fs.actions,
      enableSorting: false,
      meta: { align: "right" },
      cell: ({ row }) => {
        const a = row.original;
        return (
          <span className="flex justify-end gap-0.5">
            {a.journalEntryId ? (
              <Button asChild size="xs" variant="ghost">
                <Link href={`/erp/finance/journals/${a.journalEntryId}`}>{fs.invoices.journal}</Link>
              </Button>
            ) : null}
            {canApprove && advanceState(a) === "open" ? (
              <FormDialog
                title={s.applyTitle}
                description={s.applyHint}
                trigger={
                  <Button size="xs" variant="ghost">
                    <ReceiptText className="size-3.5" /> {s.apply}
                  </Button>
                }
              >
                {(close) => <ApplyForm advance={a} invoices={openInvoices} close={close} />}
              </FormDialog>
            ) : null}
            {canApprove && a.status === "POSTED" && parseMoney(a.applied) === 0 ? (
              <ReasonAction
                trigger={
                  <Button size="xs" variant="ghost" className="text-destructive">
                    <Ban className="size-3.5" /> {s.void}
                  </Button>
                }
                title={s.voidTitle}
                description={s.voidHint}
                confirmLabel={s.void}
                reasonLabel={s.voidReason}
                onConfirm={async (reason: string) => (await run(() => voidAdvanceAction(kind, a.id, reason), { success: s.voided })).ok}
              />
            ) : null}
          </span>
        );
      },
    },
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={rows}
      getRowId={(a) => a.id}
      emptyMessage={s.empty}
      exportName="uang-muka"
      toolbar={
        <>
          <SearchFilter placeholder={s.search} />
          <SelectFilter param="status" label={s.status} allLabel={s.allStatus} options={Object.entries(s.statuses).map(([value, label]) => ({ value, label }))} />
          <ResetFilters params={["q", "status"]} />
        </>
      }
    />
  );
}
