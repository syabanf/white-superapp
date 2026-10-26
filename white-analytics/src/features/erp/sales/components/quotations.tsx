"use client";

import * as React from "react";
import type { ColumnDef } from "@tanstack/react-table";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { CheckCircle2, FilePlus2, GitBranch, Plus, Receipt, Send, Sparkles, Trash2, XCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { useErpAction } from "@/features/erp/shared/client";
import { ErpDataTable } from "@/features/erp/shared/data-table";
import { DownloadButton } from "@/features/erp/shared/download-button";
import { Money } from "@/features/erp/shared/money-display";
import { DatePicker } from "@/features/erp/shared/period-picker";
import { SelectFilter } from "@/features/erp/shared/url-filters";
import { formatDate, formatDateTime } from "@/lib/format";
import { toast } from "@/lib/toast";
import {
  decideApprovalAction,
  deleteQuotationAction,
  generateQuotationAction,
  invoicesFromTermsAction,
  reviseQuotationAction,
  setQuotationStatusAction,
} from "../actions";
import { nameOf, type SalesCaps } from "../lib";
import { APPROVAL_STATUS, QUOTATION_STATUS, SERVICE_LINE, TIER, lbl, opts, ss } from "../strings";
import type { ApprovalRequest, Quotation } from "../types";
import { ActionButton, ConfirmAction, Fld, FormDialog, FormFooter, type Option, Picker, ReasonDialog, SalesBadge, SimpleSelect } from "./ui";

const s = ss.quotations;

export function QuotationsTable({ quotations, caps, deals }: { quotations: Quotation[]; caps: SalesCaps; deals: Option[] }) {
  const router = useRouter();
  const columns: ColumnDef<Quotation, unknown>[] = [
    {
      accessorKey: "quoteNumber",
      header: s.number,
      cell: ({ row }) => (
        <span className="font-medium tabular">
          {row.original.quoteNumber}
          {row.original.version > 1 ? <span className="ml-1 text-xs text-muted-foreground">v{row.original.version}</span> : null}
        </span>
      ),
    },
    {
      accessorKey: "clientName",
      header: s.client,
      cell: ({ row }) => (
        <div className="min-w-0">
          <div className="truncate">{row.original.clientName}</div>
          <div className="truncate text-xs text-muted-foreground">{row.original.dealTitle}</div>
        </div>
      ),
    },
    {
      accessorKey: "total",
      header: s.total,
      accessorFn: (q) => Number(q.total),
      cell: ({ row }) => <Money value={row.original.total} />,
      meta: { align: "right" },
    },
    { accessorKey: "status", header: s.status, cell: ({ row }) => <SalesBadge value={row.original.status} map={QUOTATION_STATUS} /> },
    {
      accessorKey: "approvalStatus",
      header: s.approval,
      cell: ({ row }) =>
        row.original.approvalStatus === "none" ? null : <SalesBadge value={row.original.approvalStatus} map={APPROVAL_STATUS} />,
    },
    { accessorKey: "validUntil", header: s.validUntil, cell: ({ row }) => formatDate(row.original.validUntil || null) },
  ];
  return (
    <ErpDataTable
      columns={columns}
      data={quotations}
      getRowId={(q) => q.id}
      onRowClick={(row) => router.push(`/erp/sales/quotations/${row.original.id}`)}
      exportName="quotation"
      toolbar={
        <>
          <SelectFilter param="status" label={s.status} allLabel={s.allStatus} options={opts(QUOTATION_STATUS)} />
          {caps.write ? (
            <div className="flex gap-2">
              <GenerateQuotationDialog deals={deals} />
              <NewQuotationDialog deals={deals} />
            </div>
          ) : null}
        </>
      }
    />
  );
}

/** Picks a deal, then opens the empty editor. */
function NewQuotationDialog({ deals }: { deals: Option[] }) {
  return (
    <FormDialog
      title={s.newTitle}
      trigger={
        <Button size="sm" variant="outline">
          <Plus className="size-3.5" /> {s.new}
        </Button>
      }
    >
      {(close) => <NewQuotationForm deals={deals} close={close} />}
    </FormDialog>
  );
}

function NewQuotationForm({ deals, close }: { deals: Option[]; close: () => void }) {
  const router = useRouter();
  const [deal, setDeal] = React.useState("");
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault();
        if (!deal) return;
        close();
        router.push(`/erp/sales/quotations/new?deal=${deal}`);
      }}
    >
      <Fld label={s.pickDeal} htmlFor="nq-deal">
        <Picker id="nq-deal" value={deal} onChange={setDeal} options={deals} />
      </Fld>
      <FormFooter onCancel={close} pending={false} submitLabel={ss.create} disabled={!deal} />
    </form>
  );
}

/** "Generate from deal": draft from service lines, catalog tiers and retainer discounts. */
export function GenerateQuotationDialog({ deals, dealId }: { deals?: Option[]; dealId?: string }) {
  return (
    <FormDialog
      title={s.generateTitle}
      description={s.generateHint}
      trigger={
        <Button size="sm">
          <Sparkles className="size-3.5" /> {s.generate}
        </Button>
      }
    >
      {(close) => <GenerateForm deals={deals ?? []} dealId={dealId} close={close} />}
    </FormDialog>
  );
}

function GenerateForm({ deals, dealId, close }: { deals: Option[]; dealId?: string; close: () => void }) {
  const router = useRouter();
  const [deal, setDeal] = React.useState(dealId ?? "");
  const [tier, setTier] = React.useState("standard");
  const [months, setMonths] = React.useState(3);
  const [useAi, setUseAi] = React.useState(true);
  const [usePpn, setUsePpn] = React.useState(true);
  const [startDate, setStartDate] = React.useState("");
  const { pending, run } = useErpAction();
  return (
    <form
      className="grid gap-4 sm:grid-cols-2"
      onSubmit={(e) => {
        e.preventDefault();
        if (!deal) return;
        void run(() => generateQuotationAction(deal, { tier, retainerMonths: months, useAi, usePpn, startDate }), {
          success: s.generated,
          onOk: (r) => {
            if (r.unpricedServiceLines.length)
              toast.warning(s.unpriced(r.unpricedServiceLines.map((l) => lbl(SERVICE_LINE, l)).join(", ")));
            close();
            router.push(`/erp/sales/quotations/${r.quotation.id}`);
          },
        });
      }}
    >
      {!dealId ? (
        <Fld label={s.pickDeal} htmlFor="gq-deal" className="sm:col-span-2">
          <Picker id="gq-deal" value={deal} onChange={setDeal} options={deals} />
        </Fld>
      ) : null}
      <Fld label={s.tier} htmlFor="gq-tier">
        <SimpleSelect id="gq-tier" value={tier} onChange={setTier} options={opts(TIER)} />
      </Fld>
      <Fld label={s.retainerMonths} htmlFor="gq-months">
        <Input id="gq-months" type="number" min={1} max={36} value={months} onChange={(e) => setMonths(Number(e.target.value) || 1)} />
      </Fld>
      <Fld label={s.startDate} htmlFor="gq-start">
        <DatePicker id="gq-start" value={startDate} onChange={setStartDate} clearable />
      </Fld>
      <div className="flex flex-col justify-end gap-2 text-sm">
        <label className="flex items-center gap-2">
          <Checkbox checked={useAi} onCheckedChange={(c) => setUseAi(c === true)} /> {s.useAi}
        </label>
        <label className="flex items-center gap-2">
          <Checkbox checked={usePpn} onCheckedChange={(c) => setUsePpn(c === true)} /> {s.usePpn}
        </label>
      </div>
      <div className="sm:col-span-2">
        <FormFooter onCancel={close} pending={pending} submitLabel={s.generate} disabled={!deal} />
      </div>
    </form>
  );
}

/** Status, revise, PDF, invoices and delete for one quotation. */
export function QuotationActions({ quotation: q, caps }: { quotation: Quotation; caps: SalesCaps }) {
  const router = useRouter();
  const pdf = <DownloadButton href={`/api/erp/api/sales/quotations/${q.id}/pdf`} fileName={`${q.quoteNumber}.pdf`}>{s.pdf}</DownloadButton>;
  if (!caps.write) return pdf;
  const status = (to: string, label: string, icon: React.ReactNode, variant: "default" | "outline" = "outline") => (
    <ActionButton size="sm" variant={variant} action={() => setQuotationStatusAction(q.id, to)} success={s.statusChanged}>
      {icon} {label}
    </ActionButton>
  );
  return (
    <>
      {q.status === "draft" ? (
        <ConfirmAction
          title={s.deleteTitle}
          action={() => deleteQuotationAction(q.id)}
          onDone={() => router.push("/erp/sales/quotations")}
          trigger={
            <Button size="sm" variant="ghost">
              <Trash2 className="size-3.5" /> {ss.delete}
            </Button>
          }
        />
      ) : null}
      {pdf}
      {q.status !== "draft" && q.status !== "superseded" ? (
        <ActionButton
          size="sm"
          variant="outline"
          action={() => reviseQuotationAction(q.id)}
          success={s.revised}
          onOk={(d) => router.push(`/erp/sales/quotations/${(d as Quotation).id}`)}
        >
          <GitBranch className="size-3.5" /> {s.revise}
        </ActionButton>
      ) : null}
      {q.status === "terkirim" ? (
        <>
          {status("ditolak", s.reject, <XCircle className="size-3.5" />)}
          {status("diterima", s.accept, <CheckCircle2 className="size-3.5" />, "default")}
        </>
      ) : null}
      {q.status === "draft"
        ? status("terkirim", s.send, <Send className="size-3.5" />, "default")
        : null}
      {q.status === "diterima" ? (
        <>
          <ActionButton size="sm" variant="outline" action={() => invoicesFromTermsAction(q.id)} success={s.invoicesMade}>
            <Receipt className="size-3.5" /> {s.makeInvoices}
          </ActionButton>
          <Button size="sm" asChild>
            <Link href={`/erp/sales/contracts?generate=${q.id}`}>
              <FilePlus2 className="size-3.5" /> {s.makeContract}
            </Link>
          </Button>
        </>
      ) : null}
    </>
  );
}

/** Discount approval inbox (NüHabit approvals-inbox-page). */
export function ApprovalsInbox({ requests, owners }: { requests: ApprovalRequest[]; owners: Option[] }) {
  const names = Object.fromEntries(owners.map((o) => [o.value, o.label]));
  const a = ss.approvals;
  if (!requests.length) return <p className="rounded-xl border border-dashed py-10 text-center text-sm text-muted-foreground">{a.empty}</p>;
  return (
    <ul className="flex flex-col gap-3">
      {requests.map((r) => (
        <li key={r.id} className="rounded-xl border bg-card p-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="min-w-0">
              <Link href={`/erp/sales/quotations/${r.quotationId}`} className="font-medium hover:underline">
                {r.quoteNumber} · {r.clientName}
              </Link>
              <p className="text-xs text-muted-foreground">
                {r.dealTitle} · {a.requestedBy} {nameOf(names, r.requestedBy)} · {formatDateTime(r.createdAt)}
              </p>
            </div>
            <div className="flex items-center gap-3 text-sm">
              <span className="tabular">
                {s.discount} <strong>{r.discountPercent}%</strong>
              </span>
              <Money value={r.amount} className="font-semibold" />
              <SalesBadge value={r.status} map={APPROVAL_STATUS} />
            </div>
          </div>
          <ol className="mt-3 flex flex-wrap gap-2 text-xs">
            {r.steps.map((st) => (
              <li key={st.level} className="rounded-full border px-2.5 py-1">
                {a.level(st.level)} · {st.approverUserId ? nameOf(names, st.approverUserId) : st.approverRole} ·{" "}
                {lbl(APPROVAL_STATUS, st.status)}
                {st.comment ? ` · "${st.comment}"` : ""}
              </li>
            ))}
          </ol>
          {r.canDecide && r.status === "pending" ? (
            <div className="mt-3 flex justify-end gap-2">
              <ReasonDialog
                title={a.reject}
                label={a.comment}
                submitLabel={a.reject}
                success={a.decided}
                onSubmit={(c) => decideApprovalAction(r.id, "reject", c)}
                trigger={
                  <Button size="sm" variant="outline">
                    {a.reject}
                  </Button>
                }
              />
              <ReasonDialog
                title={a.approve}
                label={a.comment}
                submitLabel={a.approve}
                success={a.decided}
                optional
                destructive={false}
                onSubmit={(c) => decideApprovalAction(r.id, "approve", c)}
                trigger={<Button size="sm">{a.approve}</Button>}
              />
            </div>
          ) : null}
        </li>
      ))}
    </ul>
  );
}
