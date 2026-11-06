"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { Check, X } from "lucide-react";
import { StatusBadge, type StatusKind } from "@/components/dashboard/status-badge";
import { Button } from "@/components/ui/button";
import { FormDialog, FormFooter, Fld } from "@/features/erp/sales/components/ui";
import { useErpAction } from "@/features/erp/shared/client";
import { ReasonAction } from "@/features/erp/shared/confirm-action";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { formatDate } from "@/lib/format";
import { approvePaymentProofAction, rejectPaymentProofAction } from "../actions";
import { fs } from "../strings";
import type { PaymentProof } from "../types";

const s = fs.proofs;
const kind: Record<string, StatusKind> = { PENDING: "warning", APPROVED: "good", REJECTED: "critical" };

function ApproveForm({ proof, close }: { proof: PaymentProof; close: () => void }) {
  const [v, setV] = React.useState({ amount: proof.amount, paidOn: proof.paidOn, bankFee: "", withheld: "" });
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  const set = (p: Partial<typeof v>) => setV((x) => ({ ...x, ...p }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => approvePaymentProofAction(proof.id, v), { success: s.approved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.amount} htmlFor="pp-amount" error={errors?.amount}>
        <MoneyInput id="pp-amount" value={v.amount} onChange={(x) => set({ amount: x })} />
      </Fld>
      <Fld label={s.paidOn} htmlFor="pp-date" error={errors?.paidOn}>
        <DatePicker id="pp-date" value={v.paidOn} onChange={(x) => set({ paidOn: x })} />
      </Fld>
      <Fld label={s.bankFee} htmlFor="pp-fee" error={errors?.bankFee}>
        <MoneyInput id="pp-fee" value={v.bankFee} onChange={(x) => set({ bankFee: x })} />
      </Fld>
      <Fld label={s.withheld} htmlFor="pp-wht" error={errors?.withheld}>
        <MoneyInput id="pp-wht" value={v.withheld} onChange={(x) => set({ withheld: x })} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

export function PaymentProofsTable({ rows, canWrite, status }: { rows: PaymentProof[]; canWrite: boolean; status: string }) {
  const { run } = useErpAction();
  const columns: ColumnDef<PaymentProof>[] = [
    {
      accessorKey: "invoiceNumber",
      header: s.invoice,
      cell: ({ row: { original: p } }) => (
        <span>
          <Link href={`/erp/sales/pipeline/${p.dealId}`} className="hover:underline">{p.invoiceNumber}</Link>
          <span className="block text-xs text-muted-foreground">{p.clientName}</span>
        </span>
      ),
    },
    { accessorKey: "payerName", header: s.payer, cell: ({ row: { original: p } }) => <span>{p.payerName}{p.note ? <span className="block text-xs text-muted-foreground">{p.note}</span> : null}</span> },
    { accessorKey: "paidOn", header: s.paidOn, cell: ({ row }) => formatDate(row.original.paidOn) },
    { accessorKey: "amount", header: s.amount, meta: { align: "right" }, cell: ({ row }) => <Money value={row.original.amount} /> },
    {
      accessorKey: "status",
      header: s.status,
      cell: ({ row: { original: p } }) => (
        <span className="flex flex-col gap-0.5">
          <StatusBadge kind={kind[p.status] ?? "neutral"}>{s.statuses[p.status]}</StatusBadge>
          {p.rejectReason ? <span className="text-xs text-muted-foreground">{p.rejectReason}</span> : null}
        </span>
      ),
    },
    {
      id: "actions",
      header: "",
      cell: ({ row: { original: p } }) => (
        <span className="flex justify-end gap-0.5">
          <DownloadButton href={`/api/erp/api/sales/payment-proofs/${p.id}/file`} fileName={p.fileName} size="xs" variant="ghost">{s.file}</DownloadButton>
          {canWrite && p.status === "PENDING" ? (
            <>
              <FormDialog title={s.approveTitle} description={s.approveHint} trigger={<Button size="xs" variant="ghost"><Check className="size-3.5" /> {s.approve}</Button>}>
                {(close) => <ApproveForm proof={p} close={close} />}
              </FormDialog>
              <ReasonAction
                trigger={<Button size="xs" variant="ghost"><X className="size-3.5" /> {s.reject}</Button>}
                title={s.rejectTitle}
                description={s.rejectHint}
                reasonLabel={s.reason}
                confirmLabel={s.reject}
                onConfirm={async (reason) => (await run(() => rejectPaymentProofAction(p.id, reason), { success: s.rejected })).ok}
              />
            </>
          ) : null}
        </span>
      ),
    },
  ];
  return (
    <div className="flex flex-col gap-3">
      <div className="flex gap-2">
        {Object.entries(s.filters).map(([value, label]) => (
          <Button key={value || "all"} size="sm" variant={status === value ? "default" : "outline"} asChild>
            <Link href={value ? `?status=${value}` : "?status=all"}>{label}</Link>
          </Button>
        ))}
      </div>
      <ErpDataTable columns={columns} data={rows} getRowId={(p) => p.id} emptyMessage={s.empty} exportName="bukti-transfer" />
    </div>
  );
}
