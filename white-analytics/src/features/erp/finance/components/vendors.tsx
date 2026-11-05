"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Pencil, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Fld, FormDialog, FormFooter } from "@/features/erp/sales/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { SearchFilter } from "@/features/erp/shared/url-filters";
import { saveVendorAction, type VendorValues } from "../actions";
import { fs } from "../strings";
import type { Vendor } from "../types";

const s = fs.vendors;

const BLANK: VendorValues = { name: "", picName: "", phone: "", email: "", address: "", npwp: "", bankName: "", bankAccountNo: "", bankAccountName: "", paymentTermsDays: 0, notes: "", isActive: true };

function VendorForm({ vendor, close }: { vendor?: Vendor; close: () => void }) {
  const [v, setV] = React.useState<VendorValues>(vendor ?? BLANK);
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  const text = (k: keyof VendorValues, label: string, cls?: string) => (
    <Fld label={label} htmlFor={`v-${k}`} error={errors?.[k]} className={cls}>
      <Input id={`v-${k}`} value={String(v[k] ?? "")} onChange={(e) => setV((p) => ({ ...p, [k]: e.target.value }))} />
    </Fld>
  );
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => saveVendorAction(v), { success: s.saved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      {text("name", `${s.name} *`, "sm:col-span-2")}
      {text("picName", s.pic)}
      {text("phone", s.phone)}
      {text("email", s.email)}
      {text("npwp", s.npwp)}
      {text("address", s.address, "sm:col-span-2")}
      {text("bankName", s.bankName)}
      {text("bankAccountNo", s.bankAccountNo)}
      {text("bankAccountName", s.bankAccountName)}
      <Fld label={s.terms} htmlFor="v-terms" error={errors?.payment_terms_days}>
        <Input id="v-terms" type="number" min={0} max={365} value={v.paymentTermsDays} onChange={(e) => setV((p) => ({ ...p, paymentTermsDays: Number(e.target.value) }))} />
      </Fld>
      {text("notes", s.notes, "sm:col-span-2")}
      <label className="flex items-center gap-2 text-sm sm:col-span-2">
        <input type="checkbox" checked={v.isActive} onChange={(e) => setV((p) => ({ ...p, isActive: e.target.checked }))} /> {s.active}
      </label>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

export function VendorDialog({ vendor }: { vendor?: Vendor }) {
  return (
    <FormDialog
      title={vendor ? s.edit : s.new}
      size="lg"
      trigger={
        vendor ? (
          <Button size="xs" variant="ghost" aria-label={s.edit}>
            <Pencil className="size-3.5" />
          </Button>
        ) : (
          <Button size="sm">
            <Plus className="size-4" /> {s.new}
          </Button>
        )
      }
    >
      {(close) => <VendorForm vendor={vendor} close={close} />}
    </FormDialog>
  );
}

export function VendorsTable({ vendors, canWrite }: { vendors: Vendor[]; canWrite: boolean }) {
  const columns: ColumnDef<Vendor, unknown>[] = [
    {
      accessorKey: "name",
      header: s.name,
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="font-medium">{row.original.name}</div>
          {row.original.isActive ? null : <div className="text-xs text-muted-foreground">{s.inactive}</div>}
        </div>
      ),
    },
    { accessorKey: "picName", header: s.pic },
    { accessorKey: "phone", header: s.phone },
    {
      id: "bank",
      header: s.bankName,
      cell: ({ row }) => (row.original.bankName ? `${row.original.bankName} ${row.original.bankAccountNo}` : "–"),
    },
    { accessorKey: "paymentTermsDays", header: s.terms, meta: { align: "right" } },
    ...(canWrite
      ? [{ id: "actions", header: "", enableSorting: false, cell: ({ row }: { row: { original: Vendor } }) => <VendorDialog vendor={row.original} /> } satisfies ColumnDef<Vendor, unknown>]
      : []),
  ];
  return <ErpDataTable columns={columns} data={vendors} getRowId={(v) => v.id} emptyMessage={s.empty} exportName="vendor" toolbar={<SearchFilter placeholder={s.search} />} />;
}
