"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Play, Plus, Power, Trash2 } from "lucide-react";
import { toast } from "@/lib/toast";
import type { ActionResult } from "@/lib/action-result";
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
import {
  createAssetAction,
  createRecurringAction,
  deleteAssetAction,
  deleteRecurringAction,
  runSchedulesAction,
  setAssetActiveAction,
  setRecurringActiveAction,
  updateAssetAction,
  updateRecurringAction,
  type AssetValues,
  type RecurringValues,
} from "../actions";
import { fs } from "../strings";
import type { FixedAsset, PostableAccount, RecurringJournal } from "../types";
import { AccountPicker } from "./account-picker";

const s = fs.schedules;

export function RunSchedulesButton() {
  const { pending, run } = useErpAction();
  return (
    <Button size="sm" variant="outline" disabled={pending} onClick={() => void run(runSchedulesAction, { onOk: (r) => toast.success(s.ran(r.posted, r.failed)) })}>
      <Play className="size-4" /> {s.run}
    </Button>
  );
}

function RowError({ message }: { message: string }) {
  return message ? <span className="block max-w-xs text-xs text-negative">{message}</span> : null;
}

const options = (map: Record<string, string>) => Object.entries(map).map(([value, label]) => ({ value, label }));

function ToggleDelete({ edit, active, onToggle, canDelete, onDelete }: { edit: React.ReactNode; active: boolean; onToggle: () => Promise<ActionResult<unknown>>; canDelete: boolean; onDelete: () => Promise<ActionResult<unknown>> }) {
  return (
    <div className="flex justify-end gap-1">
      {edit}
      <ActionButton size="xs" variant="ghost" action={onToggle} success={s.saved}>
        <Power className="size-3.5" /> {active ? s.deactivate : s.activate}
      </ActionButton>
      {canDelete ? (
        <ConfirmAction
          title={s.remove}
          action={onDelete}
          trigger={
            <Button size="icon-xs" variant="ghost" aria-label={s.remove}>
              <Trash2 className="size-3.5" />
            </Button>
          }
        />
      ) : null}
    </div>
  );
}

export function RecurringTable({ rows, accounts, canApprove }: { rows: RecurringJournal[]; accounts: PostableAccount[]; canApprove: boolean }) {
  const columns: ColumnDef<RecurringJournal, unknown>[] = [
    {
      accessorKey: "name",
      header: s.name,
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="font-medium">{row.original.name}</div>
          <RowError message={row.original.lastError} />
        </div>
      ),
    },
    { accessorKey: "frequency", header: s.frequency, cell: ({ row }) => s.frequencies[row.original.frequency] },
    { accessorKey: "total", header: s.amount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.total} /> },
    { accessorKey: "nextRun", header: s.nextRun, cell: ({ row }) => (row.original.isActive ? formatDate(row.original.nextRun) : s.inactive) },
    {
      id: "last",
      header: fs.journals.title,
      enableSorting: false,
      cell: ({ row }) =>
        row.original.lastEntryId ? (
          <Link href={`/erp/finance/journals/${row.original.lastEntryId}`} className="text-xs underline-offset-4 hover:underline">
            {fs.journals.title}
          </Link>
        ) : null,
    },
    ...(canApprove
      ? [
          {
            id: "actions",
            header: "",
            enableSorting: false,
            cell: ({ row }: { row: { original: RecurringJournal } }) => (
              <ToggleDelete
                edit={<RecurringDialog accounts={accounts} recurring={row.original} />}
                active={row.original.isActive}
                onToggle={() => setRecurringActiveAction(row.original.id, !row.original.isActive)}
                canDelete
                onDelete={() => deleteRecurringAction(row.original.id)}
              />
            ),
          } satisfies ColumnDef<RecurringJournal, unknown>,
        ]
      : []),
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={rows}
      getRowId={(r) => r.id}
      emptyMessage={s.emptyRecurring}
      exportName="jurnal-berulang"
      toolbar={
        <>
          <SearchFilter placeholder={s.search} />
          <SelectFilter param="frequency" label={s.frequency} allLabel={s.allFrequencies} options={options(s.frequencies)} />
          <SelectFilter param="status" label={s.status} allLabel={s.allStatus} options={options(s.statuses)} />
          <ResetFilters params={["q", "frequency", "status"]} />
        </>
      }
    />
  );
}

export function AssetsTable({ rows, accounts, canApprove }: { rows: FixedAsset[]; accounts: PostableAccount[]; canApprove: boolean }) {
  const columns: ColumnDef<FixedAsset, unknown>[] = [
    {
      accessorKey: "name",
      header: s.assetName,
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="font-medium">{row.original.name}</div>
          <div className="text-xs text-muted-foreground">{formatDate(row.original.acquiredOn)}</div>
          <RowError message={row.original.lastError} />
        </div>
      ),
    },
    { accessorKey: "cost", header: s.cost, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.cost} /> },
    { accessorKey: "monthlyAmount", header: s.monthly, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.monthlyAmount} /> },
    { id: "progress", header: s.progress, cell: ({ row }) => `${row.original.depreciatedMonths} / ${row.original.lifeMonths}` },
    { accessorKey: "bookValue", header: s.book, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.bookValue} /> },
    ...(canApprove
      ? [
          {
            id: "actions",
            header: "",
            enableSorting: false,
            cell: ({ row }: { row: { original: FixedAsset } }) => (
              <ToggleDelete
                edit={<AssetDialog accounts={accounts} asset={row.original} />}
                active={row.original.isActive}
                onToggle={() => setAssetActiveAction(row.original.id, !row.original.isActive)}
                canDelete={row.original.depreciatedMonths === 0}
                onDelete={() => deleteAssetAction(row.original.id)}
              />
            ),
          } satisfies ColumnDef<FixedAsset, unknown>,
        ]
      : []),
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={rows}
      getRowId={(a) => a.id}
      emptyMessage={s.emptyAssets}
      exportName="aset"
      toolbar={
        <>
          <SearchFilter placeholder={s.search} />
          <SelectFilter param="status" label={s.status} allLabel={s.allStatus} options={options(s.statuses)} />
          <ResetFilters params={["q", "status"]} />
        </>
      }
    />
  );
}

const EditTrigger = ({ label }: { label: string }) => (
  <Button size="icon-xs" variant="ghost" aria-label={label}>
    <Pencil className="size-3.5" />
  </Button>
);

export function RecurringDialog({ accounts, recurring }: { accounts: PostableAccount[]; recurring?: RecurringJournal }) {
  return (
    <FormDialog
      title={recurring ? s.editRecurring : s.newRecurring}
      description={recurring ? s.editRecurringHint : undefined}
      trigger={
        recurring ? (
          <EditTrigger label={s.editRecurring} />
        ) : (
          <Button size="sm">
            <Plus className="size-4" /> {s.newRecurring}
          </Button>
        )
      }
    >
      {(close) => <RecurringForm accounts={accounts} recurring={recurring} close={close} />}
    </FormDialog>
  );
}

function recurringValues(r?: RecurringJournal): RecurringValues {
  if (!r) return { name: "", description: "", frequency: "MONTHLY", startDate: toIsoDate(new Date()), endDate: "", debitAccountId: "", creditAccountId: "", amount: "" };
  const lines = r.lines ?? [];
  return {
    name: r.name,
    description: r.description,
    frequency: r.frequency,
    startDate: r.nextRun,
    endDate: r.endDate,
    debitAccountId: lines.find((l) => l.entrySide === "ENTRY_SIDE_DEBIT")?.accountId ?? "",
    creditAccountId: lines.find((l) => l.entrySide === "ENTRY_SIDE_CREDIT")?.accountId ?? "",
    amount: r.total,
  };
}

function RecurringForm({ accounts, recurring, close }: { accounts: PostableAccount[]; recurring?: RecurringJournal; close: () => void }) {
  const [v, setV] = React.useState<RecurringValues>(() => recurringValues(recurring));
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  const set = (k: keyof RecurringValues, x: string) => setV((p) => ({ ...p, [k]: x }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => (recurring ? updateRecurringAction(recurring.id, v) : createRecurringAction(v)), { success: s.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.name} htmlFor="rj-name" className="sm:col-span-2" error={errors?.name}>
        <Input id="rj-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
      <Fld label={s.frequency} htmlFor="rj-freq">
        <SimpleSelect id="rj-freq" value={v.frequency} onChange={(x) => set("frequency", x)} options={options(s.frequencies)} />
      </Fld>
      <Fld label={s.amount} htmlFor="rj-amount">
        <MoneyInput id="rj-amount" value={v.amount} onChange={(x) => set("amount", x)} />
      </Fld>
      <Fld label={recurring ? s.nextRun : s.startDate} htmlFor="rj-start" error={errors?.start_date}>
        <DatePicker id="rj-start" value={v.startDate} onChange={(x) => set("startDate", x)} />
      </Fld>
      <Fld label={s.endDate} htmlFor="rj-end" error={errors?.end_date}>
        <DatePicker id="rj-end" value={v.endDate} onChange={(x) => set("endDate", x)} clearable />
      </Fld>
      <Fld label={s.debit} htmlFor="rj-debit" className="sm:col-span-2">
        <AccountPicker accounts={accounts} value={v.debitAccountId} onChange={(x) => set("debitAccountId", x)} />
      </Fld>
      <Fld label={s.credit} htmlFor="rj-credit" className="sm:col-span-2">
        <AccountPicker accounts={accounts} value={v.creditAccountId} onChange={(x) => set("creditAccountId", x)} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

export function AssetDialog({ accounts, asset }: { accounts: PostableAccount[]; asset?: FixedAsset }) {
  return (
    <FormDialog
      title={asset ? s.editAsset : s.newAsset}
      description={asset?.depreciatedMonths ? s.assetLocked : s.assetHint}
      trigger={
        asset ? (
          <EditTrigger label={s.editAsset} />
        ) : (
          <Button size="sm">
            <Plus className="size-4" /> {s.newAsset}
          </Button>
        )
      }
    >
      {(close) => <AssetForm accounts={accounts} asset={asset} close={close} />}
    </FormDialog>
  );
}

function AssetForm({ accounts, asset, close }: { accounts: PostableAccount[]; asset?: FixedAsset; close: () => void }) {
  const [v, setV] = React.useState<AssetValues>(
    asset
      ? { name: asset.name, acquiredOn: asset.acquiredOn, cost: asset.cost, salvage: asset.salvage, lifeMonths: asset.lifeMonths, expenseAccountId: asset.expenseAccountId, accumAccountId: asset.accumAccountId }
      : { name: "", acquiredOn: toIsoDate(new Date()), cost: "", salvage: "", lifeMonths: 48, expenseAccountId: "", accumAccountId: "" },
  );
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  const set = <K extends keyof AssetValues>(k: K, x: AssetValues[K]) => setV((p) => ({ ...p, [k]: x }));
  const locked = Boolean(asset?.depreciatedMonths);
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => (asset ? updateAssetAction(asset.id, v) : createAssetAction(v)), { success: s.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.assetName} htmlFor="fa-name" className="sm:col-span-2" error={errors?.name}>
        <Input id="fa-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
      {locked ? null : (
        <>
          <Fld label={s.acquiredOn} htmlFor="fa-date">
            <DatePicker id="fa-date" value={v.acquiredOn} onChange={(x) => set("acquiredOn", x)} />
          </Fld>
          <Fld label={s.life} htmlFor="fa-life" error={errors?.life_months}>
            <Input id="fa-life" type="number" min={1} max={600} value={v.lifeMonths} onChange={(e) => set("lifeMonths", Number(e.target.value))} />
          </Fld>
          <Fld label={s.cost} htmlFor="fa-cost" error={errors?.cost}>
            <MoneyInput id="fa-cost" value={v.cost} onChange={(x) => set("cost", x)} />
          </Fld>
          <Fld label={s.salvage} htmlFor="fa-salvage" error={errors?.salvage}>
            <MoneyInput id="fa-salvage" value={v.salvage} onChange={(x) => set("salvage", x)} />
          </Fld>
        </>
      )}
      <Fld label={s.expense} htmlFor="fa-exp" className="sm:col-span-2" error={errors?.expense_account_id}>
        <AccountPicker accounts={accounts} value={v.expenseAccountId} onChange={(x) => set("expenseAccountId", x)} />
      </Fld>
      <Fld label={s.accum} htmlFor="fa-acc" className="sm:col-span-2" error={errors?.accum_account_id}>
        <AccountPicker accounts={accounts} value={v.accumAccountId} onChange={(x) => set("accumAccountId", x)} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}
