"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2, Pencil, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { useErpAction } from "@/features/erp/shared/client";
import { ConfirmAction } from "@/features/erp/shared/confirm-action";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { FormField } from "@/features/erp/shared/form-field";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { deleteAccountTypeAction, saveAccountTypeAction } from "../actions";
import type { AccountType } from "../types";
import { fs } from "../strings";

const t = fs.accounts.types;

function TypeDialog({ type, onClose }: { type?: AccountType; onClose: () => void }) {
  const [form, setForm] = React.useState({
    code: type?.code ?? "",
    name: type?.name ?? "",
    normalBalance: (type?.normalBalance === "CREDIT" ? "CREDIT" : "DEBIT") as "DEBIT" | "CREDIT",
    sortOrder: type?.sortOrder ?? 0,
    isActive: type?.isActive ?? true,
  });
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const { pending, run } = useErpAction();
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-md">
        <form
          className="grid gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            void run(() => saveAccountTypeAction(type?.id ?? null, form), {
              success: fs.saved,
              onOk: onClose,
              onFail: (r) => setErrors(r.fieldErrors ?? {}),
            });
          }}
        >
          <DialogHeader>
            <DialogTitle>{type ? t.editTitle : t.new}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 sm:grid-cols-[120px_1fr]">
            <FormField label={fs.code} htmlFor="at-code" error={errors.code} required>
              <Input id="at-code" value={form.code} onChange={(e) => setForm({ ...form, code: e.target.value.toUpperCase() })} />
            </FormField>
            <FormField label={fs.name} htmlFor="at-name" error={errors.name} required>
              <Input id="at-name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </FormField>
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField label={t.normalBalance}>
              <Select value={form.normalBalance} onValueChange={(v) => setForm({ ...form, normalBalance: v as "DEBIT" | "CREDIT" })}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="DEBIT">{fs.debit}</SelectItem>
                  <SelectItem value="CREDIT">{fs.credit}</SelectItem>
                </SelectContent>
              </Select>
            </FormField>
            <FormField label={t.sortOrder} htmlFor="at-sort">
              <Input
                id="at-sort"
                type="number"
                min={0}
                value={form.sortOrder}
                onChange={(e) => setForm({ ...form, sortOrder: Number(e.target.value) || 0 })}
              />
            </FormField>
          </div>
          <div className="flex items-center gap-2">
            <Switch id="at-active" checked={form.isActive} onCheckedChange={(v) => setForm({ ...form, isActive: v })} />
            <Label htmlFor="at-active">{fs.active}</Label>
          </div>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={onClose}>
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

export function AccountTypesTable({ types, canWrite }: { types: AccountType[]; canWrite: boolean }) {
  const [dialog, setDialog] = React.useState<{ type?: AccountType } | null>(null);
  const { run } = useErpAction();
  const columns: ColumnDef<AccountType, unknown>[] = [
    { accessorKey: "sortOrder", header: t.sortOrder, meta: { align: "right" } },
    { accessorKey: "code", header: fs.code, meta: { mono: true } },
    { accessorKey: "name", header: fs.name },
    { accessorKey: "normalBalance", header: t.normalBalance, cell: ({ row }) => fs.side[row.original.normalBalance] ?? "–" },
    { accessorKey: "accountCount", header: t.accountCount, meta: { align: "right" } },
    {
      accessorKey: "isActive",
      header: fs.status,
      cell: ({ row }) => <ErpStatusBadge status={row.original.isActive ? "ACTIVE" : "INACTIVE"} />,
    },
    {
      id: "actions",
      header: fs.actions,
      enableSorting: false,
      meta: { align: "right" },
      cell: ({ row }) =>
        canWrite ? (
          <span className="flex justify-end gap-0.5">
            <Button variant="ghost" size="icon-xs" aria-label={fs.edit} onClick={() => setDialog({ type: row.original })}>
              <Pencil />
            </Button>
            <ConfirmAction
              trigger={
                <Button variant="ghost" size="icon-xs" aria-label={fs.delete} className="text-destructive" disabled={row.original.accountCount > 0}>
                  <Trash2 />
                </Button>
              }
              title={fs.confirmDeleteTitle}
              description={t.deleteDesc(row.original.name)}
              confirmLabel={fs.delete}
              destructive
              onConfirm={async () => (await run(() => deleteAccountTypeAction(row.original.id), { success: fs.deleted })).ok}
            />
          </span>
        ) : null,
    },
  ];
  return (
    <>
      <ErpDataTable
        columns={columns}
        data={[...types].sort((x, y) => x.sortOrder - y.sortOrder)}
        emptyMessage={t.empty}
        toolbar={
          canWrite ? (
            <Button size="sm" className="ml-auto" onClick={() => setDialog({})}>
              <Plus className="size-4" /> {t.new}
            </Button>
          ) : null
        }
      />
      {dialog ? <TypeDialog type={dialog.type} onClose={() => setDialog(null)} /> : null}
    </>
  );
}
