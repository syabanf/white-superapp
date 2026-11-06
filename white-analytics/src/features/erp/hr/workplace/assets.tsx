"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { History, Pencil, Plus, Undo2, UserPlus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { formatDate } from "@/lib/format";
import { EmployeePicker, Fld, FormDialog, FormFooter, SimpleSelect, type EmployeeOption, type FieldErrors } from "../components/ui";
import { assetHistoryAction, assignAssetAction, returnAssetAction, saveAssetAction } from "./actions";
import { tw } from "./strings";
import type { Asset, AssetAssignment } from "./types";

const s = tw.assets;
const statusKinds = { available: "good", assigned: "info", repair: "warning", retired: "neutral" } as const;
const opts = (r: Record<string, string>) => Object.entries(r).map(([value, label]) => ({ value, label }));

function AssetForm({ asset, close }: { asset?: Asset; close: () => void }) {
  const [v, setV] = React.useState({
    id: asset?.id ?? "",
    assetCode: asset?.assetCode ?? "",
    name: asset?.name ?? "",
    category: asset?.category ?? "laptop",
    serialNo: asset?.serialNo ?? "",
    purchaseDate: asset?.purchaseDate ?? "",
    purchaseValue: asset?.purchaseValue ?? "",
    notes: asset?.notes ?? "",
    status: "",
  });
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  const set = (p: Partial<typeof v>) => setV((x) => ({ ...x, ...p }));
  const statusOptions = opts(s.statuses).filter((o) => o.value !== "assigned");
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveAssetAction(v), { success: s.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.code} htmlFor="as-code" error={errors?.assetCode}>
        <Input id="as-code" value={v.assetCode} onChange={(e) => set({ assetCode: e.target.value })} required />
      </Fld>
      <Fld label={s.name} htmlFor="as-name" error={errors?.name}>
        <Input id="as-name" value={v.name} onChange={(e) => set({ name: e.target.value })} required />
      </Fld>
      <Fld label={s.category} htmlFor="as-cat" error={errors?.category}>
        <SimpleSelect id="as-cat" value={v.category} onChange={(x) => set({ category: x as Asset["category"] })} options={opts(s.categories)} />
      </Fld>
      <Fld label={s.serial} htmlFor="as-serial">
        <Input id="as-serial" value={v.serialNo} onChange={(e) => set({ serialNo: e.target.value })} />
      </Fld>
      <Fld label={s.purchaseDate} htmlFor="as-date" error={errors?.purchaseDate}>
        <DatePicker id="as-date" value={v.purchaseDate} onChange={(x) => set({ purchaseDate: x })} clearable />
      </Fld>
      <Fld label={s.value} htmlFor="as-value" error={errors?.purchaseValue}>
        <MoneyInput id="as-value" value={v.purchaseValue} onChange={(x) => set({ purchaseValue: x })} />
      </Fld>
      {asset && asset.status !== "assigned" ? (
        <Fld label={s.status} htmlFor="as-status" error={errors?.status}>
          <SimpleSelect id="as-status" value={v.status || asset.status} onChange={(x) => set({ status: x })} options={statusOptions} />
        </Fld>
      ) : null}
      <Fld label={s.notes} htmlFor="as-notes" className="sm:col-span-2">
        <Input id="as-notes" value={v.notes} onChange={(e) => set({ notes: e.target.value })} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

function HandOverForm({ asset, employees, today, close }: { asset: Asset; employees?: EmployeeOption[]; today: string; close: () => void }) {
  const lending = asset.status === "available";
  const [v, setV] = React.useState({ employeeId: "", on: today, condition: "" });
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  const set = (p: Partial<typeof v>) => setV((x) => ({ ...x, ...p }));
  return (
    <form
      className="grid gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        const action = lending
          ? () => assignAssetAction(asset.id, { employeeId: v.employeeId, assignedOn: v.on, condition: v.condition })
          : () => returnAssetAction(asset.id, { returnedOn: v.on, condition: v.condition });
        void run(action, { success: lending ? s.lent : s.returned, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      {lending ? (
        <Fld label={s.employee} htmlFor="ho-emp" error={errors?.employeeId}>
          <EmployeePicker id="ho-emp" value={v.employeeId} onChange={(x) => set({ employeeId: x })} options={employees ?? []} />
        </Fld>
      ) : null}
      <Fld label={lending ? s.assignedOn : s.returnedOn} htmlFor="ho-date" error={errors?.assignedOn ?? errors?.returnedOn}>
        <DatePicker id="ho-date" value={v.on} onChange={(x) => set({ on: x })} />
      </Fld>
      <Fld label={lending ? s.conditionOut : s.conditionIn} htmlFor="ho-cond">
        <Input id="ho-cond" value={v.condition} onChange={(e) => set({ condition: e.target.value })} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} />
    </form>
  );
}

function HistoryList({ assetId }: { assetId: string }) {
  const [rows, setRows] = React.useState<AssetAssignment[] | null>(null);
  React.useEffect(() => {
    void assetHistoryAction(assetId).then((r) => setRows(r.ok ? r.data : []));
  }, [assetId]);
  if (!rows) return <p className="text-sm text-muted-foreground">…</p>;
  if (rows.length === 0) return <p className="text-sm text-muted-foreground">{s.historyEmpty}</p>;
  return (
    <ul className="flex flex-col gap-2 text-sm">
      {rows.map((r) => (
        <li key={r.id} className="rounded-lg border p-2.5">
          <p className="font-medium">{r.employee.fullName}</p>
          <p className="text-muted-foreground">
            {formatDate(r.assignedOn)} – {r.returnedOn ? formatDate(r.returnedOn) : s.statuses.assigned}
          </p>
          {r.conditionOut || r.conditionIn ? (
            <p className="text-xs text-muted-foreground">
              {[r.conditionOut && `${s.conditionOut}: ${r.conditionOut}`, r.conditionIn && `${s.conditionIn}: ${r.conditionIn}`].filter(Boolean).join(" · ")}
            </p>
          ) : null}
        </li>
      ))}
    </ul>
  );
}

export function AssetsAdmin({ assets, employees, canWrite, today }: { assets: Asset[]; employees: EmployeeOption[]; canWrite: boolean; today: string }) {
  const columns: ColumnDef<Asset>[] = [
    { accessorKey: "assetCode", header: s.code, cell: ({ row: { original: a } }) => <span>{a.assetCode}<span className="block text-xs text-muted-foreground">{a.serialNo}</span></span> },
    { accessorKey: "name", header: s.name, cell: ({ row: { original: a } }) => <span>{a.name}<span className="block text-xs text-muted-foreground">{s.categories[a.category]}</span></span> },
    { id: "holder", accessorFn: (a) => a.holder?.fullName ?? "", header: s.holder },
    { accessorKey: "purchaseValue", header: s.value, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.purchaseValue} muteZero /> },
    { accessorKey: "status", header: s.status, cell: ({ row }) => <ErpStatusBadge status={row.original.status} labels={s.statuses} kinds={statusKinds} /> },
    {
      id: "actions",
      header: "",
      cell: ({ row: { original: a } }) => (
        <span className="flex justify-end gap-0.5">
          <FormDialog title={`${s.history} · ${a.assetCode}`} trigger={<Button size="icon-xs" variant="ghost" aria-label={s.history}><History className="size-3.5" /></Button>}>
            {() => <HistoryList assetId={a.id} />}
          </FormDialog>
          {canWrite ? (
            <>
              <FormDialog title={s.edit} wide trigger={<Button size="icon-xs" variant="ghost" aria-label={s.edit}><Pencil className="size-3.5" /></Button>}>
                {(close) => <AssetForm asset={a} close={close} />}
              </FormDialog>
              {a.status === "available" || a.status === "assigned" ? (
                <FormDialog
                  title={a.status === "available" ? s.assignTitle : s.returnTitle}
                  trigger={
                    <Button size="xs" variant="ghost">
                      {a.status === "available" ? <UserPlus className="size-3.5" /> : <Undo2 className="size-3.5" />}
                      {a.status === "available" ? s.assign : s.giveBack}
                    </Button>
                  }
                >
                  {(close) => <HandOverForm asset={a} employees={employees} today={today} close={close} />}
                </FormDialog>
              ) : null}
            </>
          ) : null}
        </span>
      ),
    },
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={assets}
      getRowId={(a) => a.id}
      emptyMessage={s.empty}
      exportName="aset-karyawan"
      toolbar={
        canWrite ? (
          <FormDialog title={s.add} wide trigger={<Button size="sm"><Plus className="size-4" /> {s.add}</Button>}>
            {(close) => <AssetForm close={close} />}
          </FormDialog>
        ) : undefined
      }
    />
  );
}
