"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Plus, Power, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ActionButton, ConfirmAction, Fld, FormDialog, FormFooter, SimpleSelect } from "@/features/erp/sales/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker, toIsoDate } from "@/features/erp/shared/period-picker";
import { ResetFilters, SearchFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate } from "@/lib/format";
import { deleteRecurringInvoiceAction, saveRecurringInvoiceAction, setRecurringInvoiceActiveAction, type RecurringInvoiceValues } from "../actions";
import { fs } from "../strings";
import type { PostableAccount, ProjectTag, RecurringInvoice, TaxCode } from "../types";
import { AccountPicker } from "./account-picker";
import { TagPicker } from "./tag-picker";

const s = fs.recurringInvoices;
const options = (map: Record<string, string>) => Object.entries(map).map(([value, label]) => ({ value, label }));

export type RecurringInvoiceOptions = { accounts: PostableAccount[]; taxes: TaxCode[]; tags: ProjectTag[] };

function values(r?: RecurringInvoice): RecurringInvoiceValues {
  if (!r) return { kind: "AR", name: "", partyName: "", clientId: "", accountId: "", tagId: "", description: "", subtotal: "", taxAmount: "0.00", taxIds: [], dueDays: 30, frequency: "MONTHLY", startDate: toIsoDate(new Date()), endDate: "", autoPost: true };
  return { kind: r.kind, name: r.name, partyName: r.partyName, clientId: r.clientId, accountId: r.accountId, tagId: r.tagId, description: r.description, subtotal: r.subtotal, taxAmount: r.taxAmount, taxIds: r.taxIds ?? [], dueDays: r.dueDays, frequency: r.frequency, startDate: r.nextRun, endDate: r.endDate, autoPost: r.autoPost };
}

function RecurringInvoiceForm({ invoice, opts, close }: { invoice?: RecurringInvoice; opts: RecurringInvoiceOptions; close: () => void }) {
  const [v, setV] = React.useState<RecurringInvoiceValues>(() => values(invoice));
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  const set = <K extends keyof RecurringInvoiceValues>(k: K, x: RecurringInvoiceValues[K]) => setV((p) => ({ ...p, [k]: x }));
  const scope = v.kind === "AR" ? "SALE" : "PURCHASE";
  const taxChoices = opts.taxes.filter((t) => t.scope === "BOTH" || t.scope === scope || v.taxIds.includes(t.id));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveRecurringInvoiceAction(invoice?.id ?? null, v), { success: s.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={`${s.name} *`} htmlFor="ri-name" className="sm:col-span-2" error={errors?.name}>
        <Input id="ri-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
      <Fld label={s.kind} htmlFor="ri-kind">
        <SimpleSelect id="ri-kind" value={v.kind} onChange={(x) => set("kind", x as RecurringInvoiceValues["kind"])} options={options(s.kinds)} />
      </Fld>
      <Fld label={`${s.party} *`} htmlFor="ri-party" error={errors?.party_name}>
        <Input id="ri-party" value={v.partyName} onChange={(e) => set("partyName", e.target.value)} />
      </Fld>
      <Fld label={s.account} htmlFor="ri-account" hint={fs.mappingDefault} className="sm:col-span-2">
        <AccountPicker accounts={opts.accounts} filter={(a) => !a.isCashBank} value={v.accountId} onChange={(x) => set("accountId", x)} clearable />
      </Fld>
      <Fld label={`${s.subtotal} *`} htmlFor="ri-subtotal" error={errors?.subtotal}>
        <MoneyInput id="ri-subtotal" value={v.subtotal} onChange={(x) => set("subtotal", x)} />
      </Fld>
      <Fld label={fs.projects.tag} htmlFor="ri-tag" error={errors?.tag_id}>
        <TagPicker id="ri-tag" tags={opts.tags} value={v.tagId} onChange={(x) => set("tagId", x)} />
      </Fld>
      {taxChoices.length ? (
        <fieldset className="flex flex-col gap-1.5 sm:col-span-2">
          <legend className="text-sm font-medium">{fs.invoices.taxCodes}</legend>
          {errors?.tax_ids ? <p className="text-xs text-negative">{errors.tax_ids[0]}</p> : null}
          <div className="flex flex-wrap gap-x-4 gap-y-1.5">
            {taxChoices.map((t) => (
              <label key={t.id} className="flex items-center gap-2 text-sm">
                <input type="checkbox" checked={v.taxIds.includes(t.id)} onChange={() => set("taxIds", v.taxIds.includes(t.id) ? v.taxIds.filter((x) => x !== t.id) : [...v.taxIds, t.id])} /> {t.name}
              </label>
            ))}
          </div>
        </fieldset>
      ) : null}
      <Fld label={s.frequency} htmlFor="ri-freq">
        <SimpleSelect id="ri-freq" value={v.frequency} onChange={(x) => set("frequency", x as RecurringInvoiceValues["frequency"])} options={options(fs.schedules.frequencies)} />
      </Fld>
      <Fld label={s.dueDays} htmlFor="ri-due" error={errors?.due_days}>
        <Input id="ri-due" type="number" min={0} max={365} value={v.dueDays} onChange={(e) => set("dueDays", Number(e.target.value))} />
      </Fld>
      <Fld label={invoice ? fs.schedules.nextRun : fs.schedules.startDate} htmlFor="ri-start" error={errors?.start_date}>
        <DatePicker id="ri-start" value={v.startDate} onChange={(x) => set("startDate", x)} />
      </Fld>
      <Fld label={fs.schedules.endDate} htmlFor="ri-end" error={errors?.end_date}>
        <DatePicker id="ri-end" value={v.endDate} onChange={(x) => set("endDate", x)} clearable />
      </Fld>
      <Fld label={fs.description} htmlFor="ri-desc" className="sm:col-span-2">
        <Input id="ri-desc" value={v.description} onChange={(e) => set("description", e.target.value)} />
      </Fld>
      <label className="flex items-center gap-2 text-sm sm:col-span-2">
        <input type="checkbox" checked={v.autoPost} onChange={(e) => set("autoPost", e.target.checked)} /> {s.autoPost}
      </label>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

export function RecurringInvoiceDialog({ invoice, opts }: { invoice?: RecurringInvoice; opts: RecurringInvoiceOptions }) {
  const title = invoice ? s.edit : s.new;
  return (
    <FormDialog
      title={title}
      description={s.hint}
      size="lg"
      trigger={
        invoice ? (
          <Button size="icon-xs" variant="ghost" aria-label={title}>
            <Pencil className="size-3.5" />
          </Button>
        ) : (
          <Button size="sm">
            <Plus className="size-4" /> {title}
          </Button>
        )
      }
    >
      {(close) => <RecurringInvoiceForm invoice={invoice} opts={opts} close={close} />}
    </FormDialog>
  );
}

export function RecurringInvoicesTable({ rows, opts, canApprove }: { rows: RecurringInvoice[]; opts: RecurringInvoiceOptions; canApprove: boolean }) {
  const columns: ColumnDef<RecurringInvoice, unknown>[] = [
    {
      accessorKey: "name",
      header: s.name,
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="font-medium">{row.original.name}</div>
          <div className="text-xs text-muted-foreground">{row.original.partyName}</div>
          {row.original.lastError ? <span className="block max-w-xs text-xs text-negative">{row.original.lastError}</span> : null}
        </div>
      ),
    },
    { accessorKey: "kind", header: s.kind, cell: ({ row }) => s.kinds[row.original.kind] },
    { accessorKey: "frequency", header: s.frequency, cell: ({ row }) => fs.schedules.frequencies[row.original.frequency] },
    { accessorKey: "subtotal", header: s.subtotal, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.subtotal} /> },
    { accessorKey: "nextRun", header: fs.schedules.nextRun, cell: ({ row }) => (row.original.isActive ? formatDate(row.original.nextRun) : fs.schedules.inactive) },
    {
      id: "last",
      header: s.last,
      enableSorting: false,
      cell: ({ row }) =>
        row.original.lastInvoiceId ? (
          <Link href={`/erp/finance/${row.original.kind === "AR" ? "receivables" : "payables"}`} className="text-xs underline-offset-4 hover:underline">
            {s.lastLink}
          </Link>
        ) : null,
    },
    ...(canApprove
      ? [
          {
            id: "actions",
            header: "",
            enableSorting: false,
            cell: ({ row }: { row: { original: RecurringInvoice } }) => (
              <div className="flex justify-end gap-1">
                <RecurringInvoiceDialog invoice={row.original} opts={opts} />
                <ActionButton size="xs" variant="ghost" action={() => setRecurringInvoiceActiveAction(row.original.id, !row.original.isActive)} success={s.saved}>
                  <Power className="size-3.5" /> {row.original.isActive ? fs.schedules.deactivate : fs.schedules.activate}
                </ActionButton>
                <ConfirmAction
                  title={s.remove}
                  description={s.removeHint}
                  action={() => deleteRecurringInvoiceAction(row.original.id)}
                  trigger={
                    <Button size="icon-xs" variant="ghost" aria-label={s.remove}>
                      <Trash2 className="size-3.5" />
                    </Button>
                  }
                />
              </div>
            ),
          } satisfies ColumnDef<RecurringInvoice, unknown>,
        ]
      : []),
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={rows}
      getRowId={(r) => r.id}
      emptyMessage={s.empty}
      exportName="tagihan-berulang"
      toolbar={
        <>
          <SearchFilter placeholder={s.search} />
          <SelectFilter param="kind" label={s.kind} allLabel={s.allKinds} options={options(s.kinds)} />
          <SelectFilter param="frequency" label={s.frequency} allLabel={fs.schedules.allFrequencies} options={options(fs.schedules.frequencies)} />
          <SelectFilter param="status" label={fs.schedules.status} allLabel={fs.schedules.allStatus} options={options(fs.schedules.statuses)} />
          <ResetFilters params={["q", "kind", "frequency", "status"]} />
        </>
      }
    />
  );
}
