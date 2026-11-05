import type { Metadata } from "next";
import type { ReactNode } from "react";
import Link from "next/link";
import { notFound } from "next/navigation";
import { PageHeader } from "@/components/dashboard/page-header";
import { Card } from "@/components/ui/card";
import { requireErp } from "@/features/erp/access";
import { DocumentActions } from "@/features/erp/finance/components/documents";
import { financePerms } from "@/features/erp/finance/perms";
import { getDocument, listChargeTypes, listPostableAccounts, listTaxCodes, listVendors } from "@/features/erp/finance/queries";
import { fs } from "@/features/erp/finance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { Money } from "@/features/erp/shared/money-display";
import { parseMoney } from "@/features/erp/shared/money";
import { ErpStatusBadge } from "@/features/erp/shared/status-badge";
import { ErpLoadError } from "@/features/erp/shared/states";
import { formatDate } from "@/lib/format";
import { requireUser } from "@/lib/rbac";

const s = fs.documents;
export const metadata: Metadata = { title: s.detailTitle };

function Meta({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="min-w-0">
      <dt className="label-mono text-muted-foreground">{label}</dt>
      <dd className="mt-0.5 text-sm break-words">{children}</dd>
    </div>
  );
}

export default async function DocumentPage(props: PageProps<"/erp/finance/documents/[id]">) {
  const { id } = await props.params;
  const { access, offline } = await requireErp("finance.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const res = await load(() => getDocument(id));
  if (res.error?.code === "NOT_FOUND") notFound();
  if (res.error) return <ErpLoadError error={res.error} />;
  const doc = res.data;
  const user = await requireUser();
  const { canWrite, canApprove } = financePerms(access);
  const [vendors, accounts, taxes, charges] = await Promise.all([
    load(() => listVendors({ activeOnly: true })),
    canWrite ? load(listPostableAccounts) : null,
    load(() => listTaxCodes(true)),
    load(() => listChargeTypes(true)),
  ]);
  const purchase = doc.kind === "PR" || doc.kind === "PO";
  const ledger = doc.side === "SUPPLIER" ? "payables" : "receivables";
  return (
    <>
      <PageHeader
        back={{ href: purchase ? `/erp/finance/purchasing?tab=${doc.kind}` : `/erp/finance/documents?tab=${doc.kind}`, label: purchase ? s.purchasingTitle : s.commercialTitle }}
        eyebrow={s.kinds[doc.kind]}
        title={doc.number}
        description={doc.title}
        actions={<DocumentActions doc={doc} userId={user.id} isAdmin={access.role === "ADMIN"} canWrite={canWrite} canApprove={canApprove} vendors={vendors.data ?? []} accounts={accounts?.data ?? []} options={{ taxes: taxes.data ?? [], charges: charges.data ?? [] }} />}
      />
      {doc.status === "SUBMITTED" && doc.createdBy === user.id && access.role !== "ADMIN" ? <p className="text-sm text-muted-foreground">{s.ownDocument}</p> : null}
      {doc.rejectReason ? (
        <p role="status" className="rounded-lg border border-negative/30 bg-negative/5 px-3 py-2 text-sm">
          {s.rejectedBecause}: {doc.rejectReason}
        </p>
      ) : null}
      <Card className="p-(--card-pad)">
        <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Meta label={s.status}><ErpStatusBadge status={doc.status} labels={s.statuses} /></Meta>
          {doc.partyName ? <Meta label={s.party}>{doc.partyName}</Meta> : null}
          {doc.picName ? <Meta label={s.pic}>{doc.picName}</Meta> : null}
          {doc.phone ? <Meta label={s.phone}>{doc.phone}</Meta> : null}
          <Meta label={s.date}>{formatDate(doc.issueDate)}</Meta>
          {doc.dueDate ? <Meta label={s.due}>{formatDate(doc.dueDate)}</Meta> : null}
          {doc.externalNumber ? <Meta label={s.externalNumber}>{doc.externalNumber}</Meta> : null}
          {doc.sourceId ? <Meta label={s.openSource}><Link className="underline-offset-4 hover:underline" href={`/erp/finance/documents/${doc.sourceId}`}>{s.openSource}</Link></Meta> : null}
          {doc.ledgerInvoiceId ? <Meta label={s.openLedger}><Link className="underline-offset-4 hover:underline" href={`/erp/finance/${ledger}`}>{s.openLedger}</Link></Meta> : null}
        </dl>
      </Card>
      <Card className="gap-0 overflow-hidden py-0">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[560px] text-sm">
            <thead className="bg-muted/40 text-xs text-muted-foreground">
              <tr>
                <th className="px-4 py-2 text-left font-medium">{s.description}</th>
                <th className="px-4 py-2 text-right font-medium">{s.qty}</th>
                <th className="px-4 py-2 text-left font-medium">{s.unit}</th>
                <th className="px-4 py-2 text-right font-medium">{s.price}</th>
                <th className="px-4 py-2 text-right font-medium">{s.lineTotal}</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {(doc.items ?? []).map((it, i) => (
                <tr key={i}>
                  <td className="px-4 py-2">{it.description}</td>
                  <td className="px-4 py-2 text-right tabular">{parseMoney(it.qty)}</td>
                  <td className="px-4 py-2">{it.unit}</td>
                  <td className="px-4 py-2 text-right"><Money value={it.unitPrice} /></td>
                  <td className="px-4 py-2 text-right"><Money value={it.lineTotal ?? "0"} /></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <dl className="ml-auto flex w-full max-w-xs flex-col gap-1 border-t p-4 text-sm">
          <div className="flex justify-between"><dt className="text-muted-foreground">{s.subtotal}</dt><dd><Money value={doc.subtotal} /></dd></div>
          {parseMoney(doc.discountAmount) > 0 ? <div className="flex justify-between"><dt className="text-muted-foreground">{s.discountAmount}</dt><dd><Money value={doc.discountAmount} /></dd></div> : null}
          {(doc.charges ?? []).map((c) => (
            <div key={c.chargeId} className="flex justify-between"><dt className="text-muted-foreground">{c.name}</dt><dd><Money value={c.amount} signed /></dd></div>
          ))}
          {(doc.taxes ?? []).filter((t) => t.kind === "VAT").map((t) => (
            <div key={t.taxId} className="flex justify-between gap-3"><dt className="text-muted-foreground">{t.name}</dt><dd><Money value={t.amount} /></dd></div>
          ))}
          <div className="flex justify-between font-semibold"><dt>{s.total}</dt><dd><Money value={doc.totalAmount} /></dd></div>
          {(doc.taxes ?? []).filter((t) => t.kind === "WITHHOLDING").map((t) => (
            <div key={t.taxId} className="flex justify-between gap-3"><dt className="text-muted-foreground">{s.withheld(t.name)}</dt><dd><Money value={t.amount} /></dd></div>
          ))}
          {parseMoney(doc.withholdingAmount) > 0 ? <div className="flex justify-between font-semibold"><dt>{s.net}</dt><dd><Money value={doc.netDue} /></dd></div> : null}
        </dl>
      </Card>
      {doc.notes ? <Card className="p-(--card-pad) text-sm whitespace-pre-line">{doc.notes}</Card> : null}
    </>
  );
}
