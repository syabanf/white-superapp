import { Money } from "@/features/erp/shared/money-display";
import { formatDate } from "@/lib/format";
import { ss } from "../strings";
import type { DocumentParty, Quotation } from "../types";
import { DocSection, DocTotals, SalesDocument, type DocRow } from "./sales-document";

const s = ss.quotations;
const d = ss.doc;

/** Info block of a quotation; `q` is null while a new one is being drafted. */
export function quotationRows(party: DocumentParty | null, q: Quotation | null): { left: DocRow[]; right: DocRow[] } {
  return {
    left: [
      { label: d.program, value: party?.deal.title ?? q?.dealTitle },
      { label: d.client, value: party?.deal.clientName ?? q?.clientName },
      { label: d.pic, value: party?.pic },
      { label: d.phone, value: party?.phone },
      { label: d.email, value: party?.email },
    ],
    right: [
      { label: d.date, value: formatDate(q ? q.sentAt || q.createdAt : new Date().toISOString()) },
      { label: d.validUntil, value: q?.validUntil ? formatDate(q.validUntil) : null },
      { label: d.version, value: q ? `v${q.version}` : "v1" },
      { label: d.city, value: party?.city },
      { label: d.account, value: party?.owner },
    ],
  };
}

/** Read-only quotation (sent, accepted, rejected or superseded) in document form. */
export function QuotationView({ q, party }: { q: Quotation; party: DocumentParty | null }) {
  const rows = quotationRows(party, q);
  return (
    <SalesDocument title={d.quotation} number={q.quoteNumber} left={rows.left} right={rows.right}>
      <DocSection title={s.items}>
        <div className="overflow-x-auto rounded-2xl border">
          <table className="w-full text-sm">
            <thead className="border-b bg-muted/40 text-xs text-muted-foreground">
              <tr>
                <th className="px-4 py-2.5 text-left font-medium">{s.description}</th>
                <th className="px-3 py-2.5 text-right font-medium">{s.qty}</th>
                <th className="px-3 py-2.5 text-right font-medium">{s.price}</th>
                <th className="px-4 py-2.5 text-right font-medium">{s.lineTotal}</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {q.items.map((it) => (
                <tr key={it.id}>
                  <td className="px-4 py-3 align-top">
                    <div className="font-medium">{it.description}</div>
                    {it.scope ? <div className="mt-0.5 text-xs whitespace-pre-line text-muted-foreground">{it.scope}</div> : null}
                  </td>
                  <td className="px-3 py-3 text-right align-top tabular whitespace-nowrap">
                    {it.qty} {it.unit}
                  </td>
                  <td className="px-3 py-3 text-right align-top">
                    <Money value={it.unitPrice} />
                  </td>
                  <td className="px-4 py-3 text-right align-top">
                    <Money value={it.lineTotal} className="font-medium" />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </DocSection>

      <div className="grid gap-(--gap) lg:grid-cols-[minmax(0,1fr)_minmax(0,22rem)]">
        <div className="flex flex-col gap-6">
          <DocSection title={d.specialNotes}>
            <p className="text-sm whitespace-pre-line text-muted-foreground">{q.notes || "-"}</p>
          </DocSection>
          <DocSection title={s.termsTitle}>
            <ul className="divide-y rounded-2xl border text-sm">
              {q.terms.map((t) => (
                <li key={t.id} className="flex items-center justify-between gap-3 px-4 py-2.5">
                  <span className="min-w-0">
                    <span className="block">{t.label}</span>
                    <span className="text-xs text-muted-foreground">
                      {t.percent}% · {formatDate(t.dueDate || null)}
                    </span>
                  </span>
                  <Money value={t.amount} />
                </li>
              ))}
            </ul>
          </DocSection>
        </div>
        <DocTotals>
          <Line label={s.subtotal} value={q.subtotal} />
          <Line label={`${s.discountNominal} ${q.discountPercent}%`} value={`-${q.discountNominal}`} muted />
          {q.usePpn ? <Line label={`${s.ppnNominal} ${q.ppnPercent}%`} value={q.ppnNominal} muted /> : null}
          <div className="mt-1 border-t pt-2.5 text-base font-semibold">
            <Line label={s.total} value={q.total} />
          </div>
        </DocTotals>
      </div>
    </SalesDocument>
  );
}

export function Line({ label, value, muted }: { label: string; value: string; muted?: boolean }) {
  const neg = value.startsWith("-");
  return (
    <div className={`flex justify-between gap-3 ${muted ? "text-muted-foreground" : ""}`}>
      <span>{label}</span>
      <span>
        {neg ? "-" : ""}
        <Money value={neg ? value.slice(1) : value} />
      </span>
    </div>
  );
}
