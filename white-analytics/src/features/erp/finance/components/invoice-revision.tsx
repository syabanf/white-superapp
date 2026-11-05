"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { FileText, Pencil, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { useErpAction } from "@/features/erp/shared/client";
import { FileUpload } from "@/features/erp/shared/file-upload";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { deleteFakturPajakAction, updateInvoiceAction, type InvoiceRevision } from "@/features/erp/sales/actions";
import { ConfirmAction, type FieldErrors, Fld, FormDialog, FormFooter } from "@/features/erp/sales/components/ui";
import type { Invoice } from "@/features/erp/sales/types";
import { fs } from "../strings";

const s = fs.salesInvoices;

export function ReviseInvoiceDialog({ inv }: { inv: Invoice }) {
  return (
    <FormDialog
      title={s.reviseTitle}
      description={s.reviseHint}
      trigger={
        <Button size="xs" variant="outline">
          <Pencil className="size-3.5" /> {s.revise}
        </Button>
      }
    >
      {(close) => <ReviseForm inv={inv} close={close} />}
    </FormDialog>
  );
}

function ReviseForm({ inv, close }: { inv: Invoice; close: () => void }) {
  const [v, setV] = React.useState<InvoiceRevision>({ label: inv.label, amount: inv.amount, dueDate: inv.dueDate, note: inv.note });
  const [errors, setErrors] = React.useState<FieldErrors>();
  const { pending, run } = useErpAction();
  const set = (k: keyof InvoiceRevision, val: string) => setV((p) => ({ ...p, [k]: val }));
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        void run(() => updateInvoiceAction(inv.id, v), { success: s.revised, onOk: close, onFail: (r) => setErrors(r.fieldErrors) });
      }}
    >
      <Fld label={s.label} htmlFor="rev-label" error={errors?.label} className="sm:col-span-2">
        <Input id="rev-label" value={v.label} maxLength={150} onChange={(e) => set("label", e.target.value)} />
      </Fld>
      <Fld label={s.amount} htmlFor="rev-amount" error={errors?.amount} hint={inv.termId ? s.termAmountHint : undefined}>
        <MoneyInput id="rev-amount" value={v.amount} onChange={(x) => set("amount", x)} disabled={Boolean(inv.termId)} />
      </Fld>
      <Fld label={s.due} htmlFor="rev-due" error={errors?.dueDate}>
        <DatePicker id="rev-due" value={v.dueDate} onChange={(x) => set("dueDate", x)} clearable />
      </Fld>
      <Fld label={s.note} htmlFor="rev-note" error={errors?.note} className="sm:col-span-2">
        <Textarea id="rev-note" value={v.note} maxLength={300} onChange={(e) => set("note", e.target.value)} />
      </Fld>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} />
      </div>
    </form>
  );
}

/** One tax invoice per invoice: anyone who can read the invoice opens it, finance.write changes it. */
export function FakturPajakPanel({ inv, canWrite }: { inv: Invoice; canWrite: boolean }) {
  const router = useRouter();
  const url = `/api/erp/api/sales/invoices/${encodeURIComponent(inv.id)}/faktur-pajak`;
  return (
    <div className="flex flex-col gap-3">
      {inv.fakturPajakName ? (
        <div className="flex items-center gap-2 rounded-lg border px-3 py-2 text-sm">
          <FileText className="size-4 shrink-0 text-muted-foreground" />
          <a href={url} target="_blank" rel="noreferrer" className="min-w-0 flex-1 truncate underline-offset-4 hover:underline" title={s.fakturOpen}>
            {inv.fakturPajakName}
          </a>
          {canWrite ? (
            <ConfirmAction
              title={s.fakturDelete}
              success={s.fakturDeleted}
              action={() => deleteFakturPajakAction(inv.id)}
              trigger={
                <Button size="icon-xs" variant="ghost" aria-label={s.fakturDeleteLabel}>
                  <Trash2 className="size-3.5" />
                </Button>
              }
            />
          ) : null}
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">{s.fakturEmpty}</p>
      )}
      {canWrite && inv.status !== "batal" ? (
        <>
          {inv.fakturPajakName ? <p className="text-xs text-muted-foreground">{s.fakturReplace}</p> : null}
          <FileUpload
            url={url}
            accept="application/pdf,image/jpeg,image/png,image/webp"
            acceptLabel="PDF, JPG, PNG, WebP"
            multiple={false}
            onUploaded={() => router.refresh()}
          />
        </>
      ) : null}
    </div>
  );
}
