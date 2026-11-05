"use client";

import * as React from "react";
import Link from "next/link";
import type { ColumnDef } from "@tanstack/react-table";
import { MessageCircle, SkipForward } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { approveFollowupAction, skipFollowupAction } from "@/features/erp/sales/actions";
import { ConfirmAction, Fld, FormDialog, FormFooter } from "@/features/erp/sales/components/ui";
import type { InvoiceFollowup } from "@/features/erp/sales/types";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { fs } from "../strings";

const s = fs.followups;

function ApproveForm({ row, close }: { row: InvoiceFollowup; close: () => void }) {
  const [phone, setPhone] = React.useState(row.phone);
  const [message, setMessage] = React.useState(row.message);
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const { pending, run } = useErpAction();
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => approveFollowupAction(row.id, phone, message), { success: s.approved, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.phone} htmlFor="fu-phone" error={errors?.phone}>
        <Input id="fu-phone" value={phone} onChange={(e) => setPhone(e.target.value)} />
      </Fld>
      <Fld label={s.message} htmlFor="fu-message">
        <Textarea id="fu-message" rows={5} value={message} onChange={(e) => setMessage(e.target.value)} />
      </Fld>
      <FormFooter onCancel={close} pending={pending} submitLabel={s.approve} />
    </form>
  );
}

export function FollowupQueue({ rows, canWrite, whatsappEnabled }: { rows: InvoiceFollowup[]; canWrite: boolean; whatsappEnabled: boolean }) {
  const columns: ColumnDef<InvoiceFollowup, unknown>[] = [
    {
      accessorKey: "invoiceNumber",
      header: s.invoice,
      cell: ({ row }) => (
        <Link href={`/erp/finance/invoices/${row.original.invoiceId}`} className="font-medium tabular underline-offset-4 hover:underline">
          {row.original.invoiceNumber}
        </Link>
      ),
    },
    { accessorKey: "clientName", header: s.client },
    { accessorKey: "stage", header: s.stage, cell: ({ row }) => s.stages[String(row.original.stage)] ?? row.original.stage },
    {
      accessorKey: "message",
      header: s.message,
      cell: ({ row }) => (
        <div className="max-w-md min-w-0">
          <div className="line-clamp-2 text-xs">{row.original.message}</div>
          {row.original.lastError ? <div className="text-xs text-muted-foreground">{row.original.lastError}</div> : null}
        </div>
      ),
    },
    { accessorKey: "status", header: s.status, cell: ({ row }) => <ErpStatusBadge status={row.original.status} labels={s.statuses} /> },
    {
      id: "actions",
      header: "",
      enableSorting: false,
      cell: ({ row }) =>
        canWrite && (row.original.status === "PENDING" || row.original.status === "APPROVED") ? (
          <div className="flex justify-end gap-1">
            {row.original.status === "PENDING" ? (
              <FormDialog
                title={s.approveTitle}
                description={`${row.original.invoiceNumber} · ${row.original.clientName}`}
                trigger={
                  <Button size="xs">
                    <MessageCircle className="size-3.5" /> {s.approve}
                  </Button>
                }
              >
                {(close) => <ApproveForm row={row.original} close={close} />}
              </FormDialog>
            ) : null}
            <ConfirmAction
              title={s.skipTitle}
              description=""
              success={s.skipped}
              destructive={false}
              action={() => skipFollowupAction(row.original.id)}
              trigger={
                <Button size="xs" variant="ghost">
                  <SkipForward className="size-3.5" /> {s.skip}
                </Button>
              }
            />
          </div>
        ) : null,
    },
  ];
  return (
    <>
      <p className="text-sm text-muted-foreground">{s.hint}</p>
      {whatsappEnabled ? null : <p className="rounded-lg border border-brand/25 bg-brand/5 px-3 py-2 text-sm">{s.waOff}</p>}
      <ErpDataTable columns={columns} data={rows} getRowId={(r) => r.id} emptyMessage={s.empty} />
    </>
  );
}
