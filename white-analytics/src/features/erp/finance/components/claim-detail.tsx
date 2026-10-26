"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { ExternalLink, FileText, Loader2, Pencil, Plus, Send, Trash2, XCircle } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { formatDate, formatDateTime } from "@/lib/format";
import { useErpAction } from "@/features/erp/shared/client";
import { ConfirmAction } from "@/features/erp/shared/confirm-action";
import { FileUpload } from "@/features/erp/shared/file-upload";
import { FormField } from "@/features/erp/shared/form-field";
import { Money } from "@/features/erp/shared/money-display";
import { MoneyInput } from "@/features/erp/shared/money-input";
import { DatePicker, toIsoDate } from "@/features/erp/shared/period-picker";
import { claimTransitionAction, deleteReceiptAction, saveClaimAction } from "../actions";
import type { Claim, ClaimCategory, ClaimReceipt } from "../types";
import { fs } from "../strings";
import { ClaimDecision, ClaimStatusBadge, categoryLabel } from "./claims";

const rb = fs.reimburse;
const receiptUrl = (id: string) => `/api/erp/api/finance/receipts/${encodeURIComponent(id)}`;

/** Create or edit a claim. A new claim opens its detail page so receipts can be uploaded. */
export function ClaimFormDialog({ claim, trigger }: { claim?: Claim; trigger: React.ReactNode }) {
  const router = useRouter();
  const [open, setOpen] = React.useState(false);
  const blank = {
    category: (claim?.category && claim.category !== "CLAIM_CATEGORY_UNSPECIFIED" ? claim.category : "") as ClaimCategory | "",
    transactionDate: claim?.transactionDate ?? toIsoDate(new Date()),
    amount: claim?.amount ?? "",
    description: claim?.description ?? "",
    clientId: claim?.clientId ?? "",
    project: claim?.project ?? "",
  };
  const [form, setForm] = React.useState(blank);
  const [errors, setErrors] = React.useState<Record<string, string[]>>({});
  const { pending, run } = useErpAction();
  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o);
        if (o) {
          setForm(blank);
          setErrors({});
        }
      }}
    >
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <form
          className="grid gap-4"
          onSubmit={(e) => {
            e.preventDefault();
            void run(() => saveClaimAction(claim?.id ?? null, { ...form, category: form.category as ClaimCategory }), {
              success: claim ? fs.saved : rb.createdThenUpload,
              onOk: (saved) => {
                setOpen(false);
                if (!claim) router.push(`/erp/me/reimbursements/${saved.id}`);
              },
              onFail: (r) => setErrors(r.fieldErrors ?? {}),
            });
          }}
        >
          <DialogHeader>
            <DialogTitle>{claim ? rb.editTitle : rb.newTitle}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField label={rb.category} error={errors.category} required>
              <Select value={form.category} onValueChange={(v) => setForm({ ...form, category: v as ClaimCategory })}>
                <SelectTrigger className="w-full" aria-invalid={!!errors.category || undefined}>
                  <SelectValue placeholder={rb.category} />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(rb.categories).map(([v, label]) => (
                    <SelectItem key={v} value={v}>
                      {label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </FormField>
            <FormField label={rb.transactionDate} error={errors.transactionDate} required>
              <DatePicker value={form.transactionDate} onChange={(v) => setForm({ ...form, transactionDate: v })} />
            </FormField>
          </div>
          <FormField label={fs.amount} error={errors.amount} required>
            <MoneyInput value={form.amount} onChange={(v) => setForm({ ...form, amount: v })} invalid={!!errors.amount} />
          </FormField>
          <FormField label={fs.description} htmlFor="cl-desc" error={errors.description} required>
            <Textarea id="cl-desc" rows={3} value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          </FormField>
          <div className="grid gap-4 sm:grid-cols-2">
            <FormField label={rb.project} htmlFor="cl-project" error={errors.project}>
              <Input id="cl-project" value={form.project} onChange={(e) => setForm({ ...form, project: e.target.value })} />
            </FormField>
            <FormField label={rb.clientId} htmlFor="cl-client" error={errors.clientId}>
              <Input id="cl-client" value={form.clientId} onChange={(e) => setForm({ ...form, clientId: e.target.value })} />
            </FormField>
          </div>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
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

export function NewClaimButton() {
  return (
    <ClaimFormDialog
      trigger={
        <Button size="sm">
          <Plus className="size-4" /> {rb.new}
        </Button>
      }
    />
  );
}

function ReceiptTile({ receipt, claimId, canDelete }: { receipt: ClaimReceipt; claimId: string; canDelete: boolean }) {
  const { run } = useErpAction();
  const isImage = receipt.contentType.startsWith("image/");
  return (
    <div className="group relative overflow-hidden rounded-xl border bg-muted/30">
      <a href={receiptUrl(receipt.id)} target="_blank" rel="noreferrer" className="block" aria-label={`${rb.openReceipt} ${receipt.filename}`}>
        {isImage ? (
          // Receipts stream through the authenticated proxy, so next/image optimisation does not apply.
          // eslint-disable-next-line @next/next/no-img-element
          <img src={receiptUrl(receipt.id)} alt={receipt.filename} className="aspect-[4/3] w-full object-cover" loading="lazy" />
        ) : (
          <span className="flex aspect-[4/3] w-full flex-col items-center justify-center gap-2 text-muted-foreground">
            <FileText className="size-8" />
            <span className="text-xs">PDF</span>
          </span>
        )}
      </a>
      <div className="flex items-center gap-1 border-t bg-card px-2 py-1.5 text-xs">
        <span className="min-w-0 flex-1 truncate" title={receipt.filename}>
          {receipt.filename}
        </span>
        <a href={receiptUrl(receipt.id)} target="_blank" rel="noreferrer" className="text-muted-foreground hover:text-foreground" aria-label={rb.openReceipt}>
          <ExternalLink className="size-3.5" />
        </a>
        {canDelete ? (
          <ConfirmAction
            trigger={
              <button type="button" className="text-muted-foreground hover:text-destructive" aria-label={rb.deleteReceipt}>
                <Trash2 className="size-3.5" />
              </button>
            }
            title={rb.deleteReceipt}
            description={receipt.filename}
            confirmLabel={fs.delete}
            destructive
            onConfirm={async () => (await run(() => deleteReceiptAction(claimId, receipt.id), { success: rb.receiptDeleted })).ok}
          />
        ) : null}
      </div>
    </div>
  );
}

function Meta({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="min-w-0">
      <dt className="label-mono text-muted-foreground">{label}</dt>
      <dd className="mt-1 text-sm">{children}</dd>
    </div>
  );
}

/**
 * Claim detail. `mode="mine"` shows the claimant's edit, upload, submit and
 * cancel controls; `mode="review"` shows approve/reject when allowed.
 */
export function ClaimDetail({ claim, mode }: { claim: Claim; mode: "mine" | "review" }) {
  const router = useRouter();
  const { run } = useErpAction();
  const editable = mode === "mine" && claim.canEdit;
  const cancellable = mode === "mine" && (claim.status === "CLAIM_DRAFT" || claim.status === "SUBMITTED");
  return (
    <div className="grid gap-5 lg:grid-cols-[1fr_340px]">
      <div className="flex flex-col gap-5">
        <Card className="p-5">
          <div className="mb-4 flex flex-wrap items-center gap-2">
            <ClaimStatusBadge status={claim.status} />
            <span className="ml-auto flex flex-wrap gap-2">
              {editable ? (
                <ClaimFormDialog
                  claim={claim}
                  trigger={
                    <Button size="sm" variant="outline">
                      <Pencil className="size-3.5" /> {fs.edit}
                    </Button>
                  }
                />
              ) : null}
              {cancellable ? (
                <ConfirmAction
                  trigger={
                    <Button size="sm" variant="ghost" className="text-destructive">
                      <XCircle className="size-3.5" /> {rb.cancelClaim}
                    </Button>
                  }
                  title={rb.cancelTitle}
                  description={rb.cancelDesc}
                  confirmLabel={rb.cancelClaim}
                  destructive
                  onConfirm={async () => (await run(() => claimTransitionAction(claim.id, "cancel"), { success: rb.cancelled })).ok}
                />
              ) : null}
              {editable ? (
                <ConfirmAction
                  trigger={
                    <Button size="sm" disabled={claim.receipts.length === 0}>
                      <Send className="size-3.5" /> {rb.submit}
                    </Button>
                  }
                  title={rb.submitTitle}
                  description={rb.submitDesc}
                  confirmLabel={rb.submit}
                  onConfirm={async () => (await run(() => claimTransitionAction(claim.id, "submit"), { success: rb.submitted })).ok}
                />
              ) : null}
              {mode === "review" ? <ClaimDecision claim={claim} size="sm" /> : null}
            </span>
          </div>
          {claim.status === "REJECTED" && claim.rejectReason ? (
            <Alert variant="destructive" className="mb-4 rounded-xl">
              <AlertDescription>
                {claim.rejectedStage ? `${rb.rejectedAt(claim.rejectedStage)}: ` : ""}
                {claim.rejectReason}
              </AlertDescription>
            </Alert>
          ) : null}
          {claim.overLimit && claim.limitNote ? (
            <Alert className="mb-4 rounded-xl">
              <AlertDescription>
                {rb.overLimit}: {claim.limitNote}
              </AlertDescription>
            </Alert>
          ) : null}
          <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <Meta label={rb.employee}>
              {claim.employeeName}
              {claim.departmentName ? <span className="block text-xs text-muted-foreground">{claim.departmentName}</span> : null}
            </Meta>
            <Meta label={rb.category}>{categoryLabel(claim.category)}</Meta>
            <Meta label={rb.transactionDate}>{formatDate(claim.transactionDate)}</Meta>
            <Meta label={fs.amount}>
              <Money value={claim.amount} className="text-lg font-semibold" />
            </Meta>
            {claim.project ? <Meta label={rb.project}>{claim.project}</Meta> : null}
            {claim.paidAt ? <Meta label={rb.paidAt}>{formatDateTime(claim.paidAt)}</Meta> : null}
            <div className="sm:col-span-2 lg:col-span-3">
              <dt className="label-mono text-muted-foreground">{fs.description}</dt>
              <dd className="mt-1 text-sm whitespace-pre-wrap">{claim.description}</dd>
            </div>
          </dl>
        </Card>

        <Card className="gap-4 p-5">
          <h2 className="text-sm font-semibold">
            {rb.receipts} <span className="font-normal text-muted-foreground">({claim.receipts.length})</span>
          </h2>
          {claim.receipts.length === 0 ? <p className="text-sm text-muted-foreground">{rb.noReceipts}</p> : null}
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
            {claim.receipts.map((r) => (
              <ReceiptTile key={r.id} receipt={r} claimId={claim.id} canDelete={editable} />
            ))}
          </div>
          {editable ? (
            <FileUpload url={`/api/erp/api/finance/reimbursements/${encodeURIComponent(claim.id)}/receipts`} onUploaded={() => router.refresh()} />
          ) : null}
        </Card>
      </div>

      <Card className="h-fit gap-3 p-5">
        <h2 className="text-sm font-semibold">{rb.history}</h2>
        <ol className="relative space-y-4 border-l pl-4">
          {[...claim.history].reverse().map((h, i) => (
            <li key={`${h.createdAt}-${i}`} className="relative">
              <span className="absolute top-1.5 -left-[21px] size-2.5 rounded-full border-2 border-card bg-brand" aria-hidden />
              <p className="text-sm font-medium">{rb.historyActions[h.action] ?? h.action}</p>
              <p className="text-xs text-muted-foreground">
                {h.actorName || "–"} · {formatDateTime(h.createdAt)}
              </p>
              {h.note ? <p className="mt-1 text-xs whitespace-pre-wrap">{h.note}</p> : null}
            </li>
          ))}
        </ol>
      </Card>
    </div>
  );
}
