"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { Play, Plus, Power, Trash2 } from "lucide-react";
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
import { formatDate } from "@/lib/format";
import {
  createAssetAction,
  createRecurringAction,
  deleteAssetAction,
  deleteRecurringAction,
  runSchedulesAction,
  setAssetActiveAction,
  setRecurringActiveAction,
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

function ToggleDelete({ active, onToggle, canDelete, onDelete }: { active: boolean; onToggle: () => Promise<ActionResult<unknown>>; canDelete: boolean; onDelete: () => Promise<ActionResult<unknown>> }) {
  return (
    <div className="flex justify-end gap-1">
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

export function RecurringTable({ rows, canApprove }: { rows: RecurringJournal[]; canApprove: boolean }) {
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
  return <ErpDataTable columns={columns} data={rows} getRowId={(r) => r.id} emptyMessage={s.emptyRecurring} />;
}

export function AssetsTable({ rows, canApprove }: { rows: FixedAsset[]; canApprove: boolean }) {
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
  return <ErpDataTable columns={columns} data={rows} getRowId={(a) => a.id} emptyMessage={s.emptyAssets} />;
}

export function RecurringDialog({ accounts }: { accounts: PostableAccount[] }) {
  return (
    <FormDialog
      title={s.newRecurring}
      trigger={
        <Button size="sm">
          <Plus className="size-4" /> {s.newRecurring}
        </Button>
      }
    >
      {(close) => <RecurringForm accounts={accounts} close={close} />}
    </FormDialog>
  );
}

function RecurringForm({ accounts, close }: { accounts: PostableAccount[]; close: () => void }) {
  const [v, setV] = React.useState<RecurringValues>({ name: "", description: "", frequency: "MONTHLY", startDate: toIsoDate(new Date()), endDate: "", debitAccountId: "", creditAccountId: "", amount: "" });
  const { pending, run } = useErpAction();
  const set = (k: keyof RecurringValues, x: string) => setV((p) => ({ ...p, [k]: x }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => createRecurringAction(v), { success: s.saved, onOk: close });
      }}
    >
      <Fld label={s.name} htmlFor="rj-name" className="sm:col-span-2">
        <Input id="rj-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
      <Fld label={s.frequency} htmlFor="rj-freq">
        <SimpleSelect id="rj-freq" value={v.frequency} onChange={(x) => set("frequency", x)} options={Object.entries(s.frequencies).map(([value, label]) => ({ value, label }))} />
      </Fld>
      <Fld label={s.amount} htmlFor="rj-amount">
        <MoneyInput id="rj-amount" value={v.amount} onChange={(x) => set("amount", x)} />
      </Fld>
      <Fld label={s.startDate} htmlFor="rj-start">
        <DatePicker id="rj-start" value={v.startDate} onChange={(x) => set("startDate", x)} />
      </Fld>
      <Fld label={s.endDate} htmlFor="rj-end">
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

export function AssetDialog({ accounts }: { accounts: PostableAccount[] }) {
  return (
    <FormDialog
      title={s.newAsset}
      description={s.assetHint}
      trigger={
        <Button size="sm">
          <Plus className="size-4" /> {s.newAsset}
        </Button>
      }
    >
      {(close) => <AssetForm accounts={accounts} close={close} />}
    </FormDialog>
  );
}

function AssetForm({ accounts, close }: { accounts: PostableAccount[]; close: () => void }) {
  const [v, setV] = React.useState<AssetValues>({ name: "", acquiredOn: toIsoDate(new Date()), cost: "", salvage: "", lifeMonths: 48, expenseAccountId: "", accumAccountId: "" });
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  const set = <K extends keyof AssetValues>(k: K, x: AssetValues[K]) => setV((p) => ({ ...p, [k]: x }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => createAssetAction(v), { success: s.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.assetName} htmlFor="fa-name" className="sm:col-span-2" error={errors?.name}>
        <Input id="fa-name" value={v.name} onChange={(e) => set("name", e.target.value)} />
      </Fld>
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
