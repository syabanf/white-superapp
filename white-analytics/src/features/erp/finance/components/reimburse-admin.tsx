"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { Loader2, Pencil } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { formatDate } from "@/lib/format";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { FormField } from "@/features/erp/shared/form-field";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { formatRupiah } from "@/features/erp/shared/money";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { updatePolicyAction } from "../actions";
import type { MonthlySummary, MonthlySummaryRow, PostableAccount, ReimbursementBatch, ReimbursementPolicy } from "../types";
import { fs } from "../strings";
import { AccountPicker } from "./account-picker";
import { categoryLabel } from "./claims";

const rb = fs.reimburse;

export function BatchesTable({ batches }: { batches: ReimbursementBatch[] }) {
  const columns: ColumnDef<ReimbursementBatch, unknown>[] = [
    { accessorKey: "batchNo", header: rb.batchNo, meta: { mono: true } },
    { accessorKey: "paidAt", header: rb.paidAt, cell: ({ row }) => formatDate(row.original.paidAt) },
    { accessorKey: "cashAccountName", header: rb.cashAccount },
    { accessorKey: "claimCount", header: rb.claimCount, meta: { align: "right" } },
    { accessorKey: "totalAmount", header: fs.total, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.totalAmount} /> },
    {
      accessorKey: "journalStatus",
      header: rb.journalStatus,
      cell: ({ row }) => (
        <span className="flex flex-col gap-0.5">
          <span className="flex items-center gap-1.5">
            <ErpStatusBadge
              status={row.original.journalStatus.toUpperCase()}
              labels={{ POSTED: fs.journals.logs.status.posted, DRAFT: fs.journals.logs.status.draft, SKIPPED: fs.journals.logs.status.skipped }}
              kinds={{ DRAFT: "warning", SKIPPED: "warning" }}
            />
            {row.original.journalEntryId ? (
              <Link href={`/erp/finance/journals/${row.original.journalEntryId}`} className="text-xs underline">
                {fs.detail}
              </Link>
            ) : null}
          </span>
          {row.original.journalReason ? <span className="max-w-64 text-xs text-muted-foreground">{row.original.journalReason}</span> : null}
        </span>
      ),
    },
    { accessorKey: "notes", header: rb.notes, cell: ({ row }) => <span className="text-xs text-muted-foreground">{row.original.notes || "–"}</span> },
  ];
  return <ErpDataTable columns={columns} data={batches} emptyMessage={rb.emptyBatches} />;
}

export function MonthlySummaryView({ summary, showEmployee = true }: { summary: MonthlySummary; showEmployee?: boolean }) {
  const columns: ColumnDef<MonthlySummaryRow, unknown>[] = [
    ...(showEmployee
      ? ([
          {
            accessorKey: "employeeName",
            header: rb.employee,
            cell: ({ row }) => (
              <span>
                <span className="block">{row.original.employeeName}</span>
                <span className="text-xs text-muted-foreground">{row.original.departmentName}</span>
              </span>
            ),
          },
        ] as ColumnDef<MonthlySummaryRow, unknown>[])
      : []),
    { accessorKey: "category", header: rb.category, cell: ({ row }) => categoryLabel(row.original.category) },
    { accessorKey: "claimCount", header: rb.claimCount, meta: { align: "right" } },
    { accessorKey: "submittedAmount", header: rb.submittedAmount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.submittedAmount} muteZero /> },
    { accessorKey: "approvedAmount", header: rb.approvedAmount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.approvedAmount} muteZero /> },
    { accessorKey: "paidAmount", header: rb.paidAmount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.paidAmount} muteZero /> },
  ];
  return (
    <div className="page-stack">
      <KpiGrid cols={3}>
        <KpiTile label={rb.submittedAmount} value={formatRupiah(summary.totalSubmitted)} size="sm" />
        <KpiTile label={rb.approvedAmount} value={formatRupiah(summary.totalApproved)} size="sm" />
        <KpiTile label={rb.paidAmount} value={formatRupiah(summary.totalPaid)} size="sm" />
      </KpiGrid>
      <ErpDataTable columns={columns} data={summary.rows} emptyMessage={rb.emptySummary} exportName={`reimburse-${summary.month}`} />
    </div>
  );
}

function PolicyDialog({ policy, accounts, onClose }: { policy: ReimbursementPolicy; accounts: PostableAccount[]; onClose: () => void }) {
  const [form, setForm] = React.useState({
    category: policy.category,
    monthlyLimit: policy.monthlyLimit,
    perClaimLimit: policy.perClaimLimit,
    expenseAccountId: policy.expenseAccountId,
    isActive: policy.isActive,
    autoApprove: policy.autoApprove,
  });
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const { pending, run } = useErpAction();
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <form
          className="grid gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            void run(() => updatePolicyAction(form), {
              success: rb.policy.saved,
              onOk: onClose,
              onFail: (r) => setErrors(r.fieldErrors ?? {}),
            });
          }}
        >
          <DialogHeader>
            <DialogTitle>{rb.policy.editTitle(policy.label || categoryLabel(policy.category))}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField label={rb.policy.perClaimLimit} error={errors.perClaimLimit} hint={rb.policy.emptyMeansNoLimit}>
              <MoneyInput value={form.perClaimLimit} onChange={(v) => setForm({ ...form, perClaimLimit: v })} />
            </FormField>
            <FormField label={rb.policy.monthlyLimit} error={errors.monthlyLimit} hint={rb.policy.emptyMeansNoLimit}>
              <MoneyInput value={form.monthlyLimit} onChange={(v) => setForm({ ...form, monthlyLimit: v })} />
            </FormField>
          </div>
          <FormField label={rb.policy.expenseAccount} error={errors.expenseAccountId} hint={fs.mappingDefault}>
            <AccountPicker accounts={accounts} filter={(a) => !a.isCashBank} value={form.expenseAccountId} onChange={(v) => setForm({ ...form, expenseAccountId: v })} clearable />
          </FormField>
          <div className="flex items-start justify-between gap-3 rounded-lg border px-3 py-2.5">
            <div>
              <Label htmlFor="pol-auto">{rb.policy.autoApprove}</Label>
              <p className="text-xs text-muted-foreground">{rb.policy.autoApproveHint}</p>
            </div>
            <Switch id="pol-auto" checked={form.autoApprove} onCheckedChange={(v) => setForm({ ...form, autoApprove: v })} />
          </div>
          <div className="flex items-center gap-2">
            <Switch id="pol-active" checked={form.isActive} onCheckedChange={(v) => setForm({ ...form, isActive: v })} />
            <Label htmlFor="pol-active">{fs.active}</Label>
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

export function PoliciesTable({ policies, accounts, canEdit }: { policies: ReimbursementPolicy[]; accounts: PostableAccount[]; canEdit: boolean }) {
  const [editing, setEditing] = React.useState<ReimbursementPolicy | null>(null);
  const limit = (v: string) => (v ? formatRupiah(v) : <span className="text-muted-foreground">{rb.policy.noLimit}</span>);
  const columns: ColumnDef<ReimbursementPolicy, unknown>[] = [
    { accessorKey: "category", header: rb.category, cell: ({ row }) => row.original.label || categoryLabel(row.original.category) },
    { accessorKey: "perClaimLimit", header: rb.policy.perClaimLimit, meta: { align: "right" }, cell: ({ row }) => limit(row.original.perClaimLimit) },
    { accessorKey: "monthlyLimit", header: rb.policy.monthlyLimit, meta: { align: "right" }, cell: ({ row }) => limit(row.original.monthlyLimit) },
    { accessorKey: "expenseAccountName", header: rb.policy.expenseAccount, cell: ({ row }) => row.original.expenseAccountName || "–" },
    { accessorKey: "autoApprove", header: rb.policy.autoApprove, cell: ({ row }) => (row.original.autoApprove ? fs.yes : fs.no) },
    { accessorKey: "isActive", header: fs.status, cell: ({ row }) => <ErpStatusBadge status={row.original.isActive ? "ACTIVE" : "INACTIVE"} /> },
    {
      id: "actions",
      header: fs.actions,
      enableSorting: false,
      meta: { align: "right" },
      cell: ({ row }) =>
        canEdit ? (
          <Button variant="ghost" size="icon-xs" aria-label={fs.edit} onClick={() => setEditing(row.original)}>
            <Pencil />
          </Button>
        ) : null,
    },
  ];
  return (
    <>
      <ErpDataTable columns={columns} data={policies} />
      {editing ? <PolicyDialog policy={editing} accounts={accounts} onClose={() => setEditing(null)} /> : null}
    </>
  );
}
