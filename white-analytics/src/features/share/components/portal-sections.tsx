"use client";

import * as React from "react";
import { useActionState } from "react";
import { Download, FileText, Loader2, Send } from "lucide-react";
import { StatusBadge, type StatusKind } from "@/components/dashboard/status-badge";
import { EmptyState } from "@/components/dashboard/empty-state";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { formatCurrency, formatDate } from "@/lib/format";
import { toast } from "@/lib/toast";
import type { ActionResult } from "@/lib/action-result";
import { decideContent, submitPaymentProof } from "../portal-actions";
import type { PortalContentItem, PortalContract, PortalData, PortalInvoice, PortalProof } from "../portal";
import { pt } from "../strings";

const rupiah = (v: string | number) => formatCurrency(Number(v));

function invoiceState(inv: PortalInvoice, today: string): { kind: StatusKind; label: string } {
  const remaining = Number(inv.amount) - Number(inv.paidTotal);
  if (remaining <= 0) return { kind: "good", label: pt.billing.status.paid };
  if (inv.dueDate && inv.dueDate < today) return { kind: "critical", label: pt.billing.status.overdue };
  if (Number(inv.paidTotal) > 0) return { kind: "warning", label: pt.billing.status.partial };
  return { kind: "neutral", label: pt.billing.status.open };
}

const proofKind: Record<PortalProof["status"], StatusKind> = { PENDING: "warning", APPROVED: "good", REJECTED: "critical" };

export function PortalBilling({ slug, data, today }: { slug: string; data: PortalData; today: string }) {
  const outstanding = data.invoices.reduce((sum, i) => sum + Math.max(0, Number(i.amount) - Number(i.paidTotal)), 0);
  return (
    <div className="space-y-4">
      <div className="grid gap-4 md:grid-cols-[1fr_2fr]">
        <Card className="p-(--card-pad)">
          <p className="text-sm text-muted-foreground">{pt.billing.outstanding}</p>
          <p className="mt-1 text-2xl font-semibold tabular-nums">{rupiah(outstanding)}</p>
        </Card>
        <Card className="p-(--card-pad)">
          <p className="text-sm text-muted-foreground">{pt.billing.bank}</p>
          <p className="mt-1 text-sm">{data.bankLine}</p>
        </Card>
      </div>
      {data.invoices.length === 0 ? (
        <EmptyState title={pt.billing.empty} />
      ) : (
        data.invoices.map((inv) => (
          <InvoiceCard key={inv.id} slug={slug} inv={inv} today={today} proofs={data.proofs.filter((p) => p.invoiceId === inv.id)} />
        ))
      )}
    </div>
  );
}

function InvoiceCard({ slug, inv, today, proofs }: { slug: string; inv: PortalInvoice; today: string; proofs: PortalProof[] }) {
  const state = invoiceState(inv, today);
  const remaining = Math.max(0, Number(inv.amount) - Number(inv.paidTotal));
  return (
    <Card className="space-y-3 p-(--card-pad)">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="font-medium">
            {inv.invoiceNumber} · {inv.label}
          </p>
          <p className="truncate text-sm text-muted-foreground">{inv.dealTitle}</p>
        </div>
        <StatusBadge kind={state.kind}>{state.label}</StatusBadge>
      </div>
      <dl className="grid grid-cols-2 gap-2 text-sm sm:grid-cols-4">
        <div>
          <dt className="text-muted-foreground">{pt.billing.total}</dt>
          <dd className="tabular-nums">{rupiah(inv.amount)}</dd>
        </div>
        <div>
          <dt className="text-muted-foreground">{pt.billing.paid}</dt>
          <dd className="tabular-nums">{rupiah(inv.paidTotal)}</dd>
        </div>
        <div>
          <dt className="text-muted-foreground">{pt.billing.outstanding}</dt>
          <dd className="font-medium tabular-nums">{rupiah(remaining)}</dd>
        </div>
        <div>
          <dt className="text-muted-foreground">{pt.billing.due}</dt>
          <dd>{inv.dueDate ? formatDate(inv.dueDate) : pt.billing.noDue}</dd>
        </div>
      </dl>
      <div className="flex flex-wrap gap-2">
        <Button variant="outline" size="sm" asChild>
          <a href={`/api/share/${slug}/invoices/${inv.id}/pdf`} target="_blank" rel="noopener">
            <Download className="size-4" /> {pt.billing.pdf}
          </a>
        </Button>
        {remaining > 0 ? <ProofDialog slug={slug} invoice={inv} remaining={remaining} today={today} /> : null}
      </div>
      {proofs.length > 0 ? (
        <ul className="space-y-1.5 border-t pt-3 text-sm">
          {proofs.map((p) => (
            <li key={p.id} className="flex flex-wrap items-center justify-between gap-2">
              <span className="text-muted-foreground">
                {formatDate(p.paidOn)} · {rupiah(p.amount)} · {p.payerName}
              </span>
              <span className="flex flex-col items-end gap-0.5">
                <StatusBadge kind={proofKind[p.status]}>{pt.billing.proofStatus[p.status]}</StatusBadge>
                {p.status === "REJECTED" && p.rejectReason ? <span className="text-xs text-muted-foreground">{p.rejectReason}</span> : null}
              </span>
            </li>
          ))}
        </ul>
      ) : null}
    </Card>
  );
}

function ProofDialog({ slug, invoice, remaining, today }: { slug: string; invoice: PortalInvoice; remaining: number; today: string }) {
  const [open, setOpen] = React.useState(false);
  const [state, action, pending] = useActionState<ActionResult | null, FormData>(async (prev, fd) => {
    const res = await submitPaymentProof(slug, prev, fd);
    if (res.ok) {
      toast.success(pt.proof.sent);
      setOpen(false);
    }
    return res;
  }, null);
  const err = (k: string) => (state && !state.ok ? state.fieldErrors?.[k]?.[0] : undefined);
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm">
          <Send className="size-4" /> {pt.billing.sendProof}
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {pt.proof.title} · {invoice.invoiceNumber}
          </DialogTitle>
          <DialogDescription>{pt.proof.desc}</DialogDescription>
        </DialogHeader>
        <form action={action} className="space-y-3">
          <input type="hidden" name="invoiceId" value={invoice.id} />
          <FieldRow id="amount" label={pt.proof.amount} error={err("amount")}>
            <Input id="amount" name="amount" inputMode="decimal" defaultValue={remaining.toFixed(2).replace(".", ",").replace(/,00$/, "")} required />
          </FieldRow>
          <FieldRow id="paidOn" label={pt.proof.paidOn} error={err("paidOn")}>
            <Input id="paidOn" name="paidOn" type="date" max={today} defaultValue={today} required />
          </FieldRow>
          <FieldRow id="payerName" label={pt.proof.payerName} error={err("payerName")}>
            <Input id="payerName" name="payerName" required maxLength={120} />
          </FieldRow>
          <FieldRow id="file" label={pt.proof.file} error={err("file")}>
            <Input id="file" name="file" type="file" accept="application/pdf,image/png,image/jpeg,image/webp" required />
          </FieldRow>
          <FieldRow id="note" label={pt.proof.note}>
            <Textarea id="note" name="note" rows={2} maxLength={500} />
          </FieldRow>
          {state && !state.ok && !state.fieldErrors ? <p className="text-sm text-destructive">{state.error}</p> : null}
          <Button type="submit" className="w-full" disabled={pending}>
            {pending ? <Loader2 className="size-4 animate-spin" /> : null} {pt.proof.submit}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function FieldRow({ id, label, error, children }: { id: string; label: string; error?: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      {children}
      {error ? <p className="text-xs text-destructive">{error}</p> : null}
    </div>
  );
}

const contractKind: Record<string, StatusKind> = { sent: "warning", signed: "good", active: "good", ended: "neutral" };

export function PortalContracts({ contracts }: { contracts: PortalContract[] }) {
  if (contracts.length === 0) return <EmptyState title={pt.contracts.empty} />;
  return (
    <div className="space-y-3">
      {contracts.map((c) => (
        <Card key={c.id} className="flex flex-wrap items-center justify-between gap-3 p-(--card-pad)">
          <div className="min-w-0">
            <p className="font-medium">{c.title}</p>
            <p className="text-sm text-muted-foreground">
              {c.contractNumber}
              {c.startDate ? ` · ${formatDate(c.startDate)} – ${c.endDate ? formatDate(c.endDate) : "…"}` : ""}
            </p>
          </div>
          <div className="flex items-center gap-2">
            <StatusBadge kind={contractKind[c.status] ?? "neutral"}>{pt.contracts.status[c.status as keyof typeof pt.contracts.status] ?? c.status}</StatusBadge>
            <Button size="sm" variant={c.status === "sent" ? "default" : "outline"} asChild>
              <a href={`/kontrak/${c.signToken}`}>
                <FileText className="size-4" /> {c.status === "sent" ? pt.contracts.sign : pt.contracts.open}
              </a>
            </Button>
          </div>
        </Card>
      ))}
    </div>
  );
}

export function PortalContent({ slug, items }: { slug: string; items: PortalContentItem[] }) {
  const [signer, setSigner] = React.useState("");
  if (items.length === 0) return <EmptyState title={pt.content.empty} />;
  return (
    <div className="space-y-4">
      <div className="max-w-xs space-y-1.5">
        <Label htmlFor="signer">{pt.content.signer}</Label>
        <Input id="signer" value={signer} onChange={(e) => setSigner(e.target.value)} maxLength={120} />
      </div>
      {items.map((it) => (
        <ContentCard key={it.id} slug={slug} item={it} signer={signer} />
      ))}
    </div>
  );
}

function ContentCard({ slug, item, signer }: { slug: string; item: PortalContentItem; signer: string }) {
  const [note, setNote] = React.useState("");
  const [revising, setRevising] = React.useState(false);
  const [pending, start] = React.useTransition();
  const decide = (approve: boolean) =>
    start(async () => {
      const res = await decideContent(slug, { itemId: item.id, approve, signerName: signer, note: approve ? "" : note });
      if (res.ok) toast.success(approve ? pt.content.approved : pt.content.revised);
      else toast.error(res.fieldErrors?.signerName?.[0] ?? res.fieldErrors?.note?.[0] ?? res.error);
    });
  const images = item.files.filter((f) => f.mimeType.startsWith("image/"));
  return (
    <Card className="space-y-3 p-(--card-pad)">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <p className="font-medium">{item.title}</p>
          <p className="text-sm text-muted-foreground">
            {item.platform.toLowerCase()} · {item.format.toLowerCase()} · {pt.content.planned} {formatDate(item.plannedDate)}
            {item.revisionRound > 0 ? ` · ${pt.content.round}${item.revisionRound}` : ""}
          </p>
        </div>
      </div>
      {images.length > 0 ? (
        <div className="flex gap-2 overflow-x-auto">
          {images.map((f) => (
            // eslint-disable-next-line @next/next/no-img-element -- private, cookie-gated file; next/image cannot fetch it
            <img key={f.id} src={`/api/share/${slug}/files/${f.id}`} alt={f.fileName} className="h-56 w-auto shrink-0 rounded-xl border object-contain" />
          ))}
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">{pt.content.noVisual}</p>
      )}
      {item.hook ? <p className="font-medium">{item.hook}</p> : null}
      <p className="text-sm whitespace-pre-line">{item.caption}</p>
      {item.hashtags.length > 0 ? <p className="text-sm text-muted-foreground">{item.hashtags.join(" ")}</p> : null}
      {revising ? (
        <div className="space-y-1.5">
          <Label htmlFor={`note-${item.id}`}>{pt.content.note}</Label>
          <Textarea id={`note-${item.id}`} value={note} onChange={(e) => setNote(e.target.value)} rows={3} maxLength={1000} />
        </div>
      ) : null}
      <div className="flex flex-wrap gap-2">
        <Button size="sm" disabled={pending} onClick={() => decide(true)}>
          {pt.content.approve}
        </Button>
        <Button size="sm" variant="outline" disabled={pending} onClick={() => (revising ? decide(false) : setRevising(true))}>
          {pt.content.revise}
        </Button>
      </div>
    </Card>
  );
}
