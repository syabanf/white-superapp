import type { ReactNode } from "react";
import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { Wallet } from "lucide-react";
import { DetailLayout } from "@/components/dashboard/layout";
import { PageHeader } from "@/components/dashboard/page-header";
import { Card } from "@/components/ui/card";
import { requireErp } from "@/features/erp/access";
import { FakturPajakPanel, ReviseInvoiceDialog } from "@/features/erp/finance/components/invoice-revision";
import { fs } from "@/features/erp/finance/strings";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { InvoiceActions, PaymentDialog } from "@/features/erp/sales/components/billing";
import { SalesBadge } from "@/features/erp/sales/components/ui";
import { salesCaps } from "@/features/erp/sales/lib";
import { getInvoice, listDealPayments } from "@/features/erp/sales/queries";
import { INVOICE_STATUS, PAYMENT_METHOD, lbl } from "@/features/erp/sales/strings";
import { load } from "@/features/erp/shared/load";
import { parseMoney } from "@/features/erp/shared/money";
import { Money } from "@/features/erp/shared/money-display";
import { ErpLoadError } from "@/features/erp/shared/states";
import { formatDate } from "@/lib/format";

const s = fs.salesInvoices;
export const metadata: Metadata = { title: s.title };

function Row({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-4 py-2 text-sm">
      <dt className="shrink-0 text-muted-foreground">{label}</dt>
      <dd className="min-w-0 text-right break-words">{children}</dd>
    </div>
  );
}

export default async function FinanceInvoicePage(props: PageProps<"/erp/finance/invoices/[id]">) {
  const { id } = await props.params;
  const { access, offline } = await requireErp("finance.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const res = await load(() => getInvoice(id));
  if (res.error?.code === "NOT_FOUND") notFound();
  if (res.error) return <ErpLoadError error={res.error} />;
  const inv = res.data;
  const payments = await load(() => listDealPayments(inv.dealId));
  const own = (payments.data ?? []).filter((p) => p.invoiceId === inv.id);
  const remaining = parseMoney(inv.amount) - parseMoney(inv.paidTotal);

  return (
    <>
      <PageHeader
        back={{ href: "/erp/finance/invoices", label: s.title }}
        eyebrow={fs.eyebrow}
        title={inv.invoiceNumber}
        description={inv.clientName}
        actions={
          <div className="flex items-center gap-1">
            {caps.finance && inv.status === "diajukan" && parseMoney(inv.paidTotal) === 0 ? <ReviseInvoiceDialog inv={inv} /> : null}
            <InvoiceActions inv={inv} caps={caps} />
          </div>
        }
      />
      <DetailLayout
        aside={
          <Card className="gap-3 p-(--card-pad)">
            <h2 className="text-sm font-semibold">{s.faktur}</h2>
            <FakturPajakPanel inv={inv} canWrite={caps.finance} />
          </Card>
        }
      >
        <Card className="gap-0 p-(--card-pad)">
          <h2 className="mb-2 text-sm font-semibold">{s.detail}</h2>
          <dl className="divide-y">
            <Row label={fs.status}><SalesBadge value={inv.status} map={INVOICE_STATUS} /></Row>
            <Row label={s.client}>{inv.clientName}</Row>
            <Row label={s.deal}>{inv.dealTitle}</Row>
            {inv.quoteNumber ? <Row label={s.quotation}>{inv.quoteNumber}</Row> : null}
            {inv.termLabel ? <Row label={s.term}>{inv.termLabel}</Row> : null}
            <Row label={s.label}>{inv.label}</Row>
            <Row label={s.due}>{formatDate(inv.dueDate || null)}</Row>
            {inv.note ? <Row label={s.note}>{inv.note}</Row> : null}
            {inv.cancelReason ? <Row label={s.cancelReason}>{inv.cancelReason}</Row> : null}
            <Row label={s.amount}><Money value={inv.amount} className="font-semibold" /></Row>
            <Row label={s.paid}><Money value={inv.paidTotal} muteZero /></Row>
            <Row label={s.outstanding}><Money value={String(Math.max(0, remaining))} /></Row>
          </dl>
        </Card>
        <Card className="gap-3 p-(--card-pad)">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h2 className="text-sm font-semibold">{s.payments}</h2>
            {caps.finance && inv.status === "terkirim" && remaining > 0 ? (
              <PaymentDialog dealId={inv.dealId} invoices={[inv]} preset={{ invoiceId: inv.id, amount: String(remaining) }} />
            ) : null}
          </div>
          {payments.error ? <ErpLoadError error={payments.error} /> : null}
          {own.length ? (
            <ul className="divide-y rounded-lg border text-sm">
              {own.map((p) => (
                <li key={p.id} className="flex items-center gap-3 px-3 py-2">
                  <Wallet className="size-4 shrink-0 text-muted-foreground" />
                  <span className="min-w-0 flex-1">
                    <span className="tabular">{formatDate(p.paidOn)}</span>
                    <span className="text-muted-foreground">
                      {" "}· {lbl(PAYMENT_METHOD, p.method)}
                      {p.note ? ` · ${p.note}` : ""}
                    </span>
                  </span>
                  <Money value={p.amount} className="font-medium" />
                </li>
              ))}
            </ul>
          ) : !payments.error ? (
            <p className="text-sm text-muted-foreground">{inv.status === "diajukan" ? s.paymentNeedsIssue : s.noPayments}</p>
          ) : null}
          {own.length ? <p className="text-xs text-muted-foreground">{s.paymentCorrectHint}</p> : null}
        </Card>
      </DetailLayout>
    </>
  );
}
