import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { PageHeader } from "@/components/dashboard/page-header";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load } from "@/features/erp/shared/load";
import { formatRupiah, parseMoney } from "@/features/erp/shared/money";
import { Money } from "@/features/erp/shared/money-display";
import { PrintButton } from "@/features/erp/shared/print";
import { ErpError } from "@/features/erp/shared/states";
import { InvoiceActions } from "@/features/erp/sales/components/billing";
import { Line } from "@/features/erp/sales/components/quotation-view";
import { DocSection, DocTotals, SalesDocument } from "@/features/erp/sales/components/sales-document";
import { SalesBadge } from "@/features/erp/sales/components/ui";
import { salesCaps } from "@/features/erp/sales/lib";
import { getDocumentParty, getInvoice, getQuotation } from "@/features/erp/sales/queries";
import { INVOICE_STATUS, PAYMENT_METHOD, lbl, ss } from "@/features/erp/sales/strings";
import { formatDate } from "@/lib/format";

const s = ss.invoices;
const d = ss.doc;
export const metadata: Metadata = { title: s.title };

export default async function InvoicePage(props: PageProps<"/erp/sales/invoices/[id]">) {
  const { id } = await props.params;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const res = await load(() => getInvoice(id));
  if (res.error?.code === "NOT_FOUND") notFound();
  if (!res.data) return <ErpError message={res.error?.message} />;
  const inv = res.data;
  const [party, quotation] = await Promise.all([
    load(() => getDocumentParty(inv.dealId)),
    inv.quotationId ? load(() => getQuotation(inv.quotationId)) : null,
  ]);
  const p = party.data;
  const quoteNumber = quotation?.data?.quoteNumber ?? "";
  const payments = (p?.payments ?? []).filter((x) => x.invoiceId === inv.id);
  const remaining = parseMoney(inv.amount) - parseMoney(inv.paidTotal);

  return (
    <>
      <PageHeader
        back={{ href: `/erp/sales/pipeline/${inv.dealId}`, label: inv.dealTitle }}
        eyebrow={s.title}
        title={inv.invoiceNumber}
        description={inv.clientName}
        actions={
          <>
            <PrintButton />
            <InvoiceActions inv={inv} caps={caps} />
          </>
        }
      />
      <SalesDocument
        title={d.invoice}
        number={inv.invoiceNumber}
        left={[
          { label: d.program, value: inv.dealTitle },
          { label: d.client, value: inv.clientName },
          { label: d.pic, value: p?.pic },
          { label: d.phone, value: p?.phone },
          { label: d.email, value: p?.email },
        ]}
        right={[
          { label: d.issuedOn, value: formatDate(inv.sentAt || inv.createdAt) },
          { label: d.due, value: inv.dueDate ? formatDate(inv.dueDate) : null },
          { label: d.reference, value: quoteNumber },
          { label: d.term, value: inv.label },
          { label: d.status, value: <SalesBadge value={inv.status} map={INVOICE_STATUS} /> },
        ]}
      >
        <DocSection title={s.label}>
          <div className="overflow-x-auto rounded-2xl border">
            <table className="w-full text-sm">
              <thead className="border-b bg-muted/40 text-xs text-muted-foreground">
                <tr>
                  <th className="px-4 py-2.5 text-left font-medium">{s.label}</th>
                  <th className="px-4 py-2.5 text-right font-medium">{s.amount}</th>
                </tr>
              </thead>
              <tbody>
                <tr>
                  <td className="px-4 py-3">
                    <div className="font-medium">{d.invoiceLine(inv.label, quoteNumber)}</div>
                    {inv.cancelReason ? <div className="mt-0.5 text-xs text-destructive">{inv.cancelReason}</div> : null}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Money value={inv.amount} className="font-medium" />
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </DocSection>

        <div className="grid gap-(--gap) lg:grid-cols-[minmax(0,1fr)_minmax(0,22rem)]">
          <DocSection title={s.payments}>
            {payments.length ? (
              <ul className="divide-y rounded-2xl border text-sm">
                {payments.map((x) => (
                  <li key={x.id} className="flex items-center justify-between gap-3 px-4 py-2.5">
                    <span className="min-w-0">
                      <span className="block">{lbl(PAYMENT_METHOD, x.method)}</span>
                      <span className="text-xs text-muted-foreground">
                        {formatDate(x.paidOn)}
                        {x.note ? ` · ${x.note}` : ""}
                      </span>
                    </span>
                    <Money value={x.amount} />
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-muted-foreground">{s.noPayments}</p>
            )}
          </DocSection>
          <DocTotals>
            <Line label={d.billed} value={inv.amount} />
            <Line label={s.paid} value={inv.paidTotal} muted />
            <div className="mt-1 flex justify-between gap-3 border-t pt-2.5 text-base font-semibold">
              <span>{d.remaining}</span>
              <span className="tabular">{formatRupiah(Math.max(0, remaining))}</span>
            </div>
          </DocTotals>
        </div>
      </SalesDocument>
    </>
  );
}
