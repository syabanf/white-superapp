import { Card } from "@/components/ui/card";
import { Money } from "@/features/erp/shared/money-display";
import { formatDate } from "@/lib/format";
import { ss } from "../strings";
import type { Quotation } from "../types";

const s = ss.quotations;

/** Read-only quotation body (sent, accepted, rejected or superseded). */
export function QuotationView({ q }: { q: Quotation }) {
  return (
    <div className="flex flex-col gap-4">
      <Card className="gap-0 overflow-hidden py-0">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="border-b bg-muted/40 text-xs text-muted-foreground">
              <tr>
                <th className="px-(--card-pad) py-2 text-left font-medium">{s.description}</th>
                <th className="px-3 py-2 text-right font-medium">{s.qty}</th>
                <th className="px-3 py-2 text-right font-medium">{s.price}</th>
                <th className="px-(--card-pad) py-2 text-right font-medium">{s.lineTotal}</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {q.items.map((it) => (
                <tr key={it.id}>
                  <td className="px-(--card-pad) py-3 align-top">
                    <div className="font-medium">{it.description}</div>
                    {it.scope ? <div className="mt-0.5 text-xs whitespace-pre-line text-muted-foreground">{it.scope}</div> : null}
                  </td>
                  <td className="px-3 py-3 text-right align-top tabular whitespace-nowrap">
                    {it.qty} {it.unit}
                  </td>
                  <td className="px-3 py-3 text-right align-top">
                    <Money value={it.unitPrice} />
                  </td>
                  <td className="px-(--card-pad) py-3 text-right align-top">
                    <Money value={it.lineTotal} className="font-medium" />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
      <div className="grid gap-(--gap) lg:grid-cols-2">
        <Card className="gap-3 p-(--card-pad)">
          <h2 className="text-[15px] font-semibold">{s.termsTitle}</h2>
          <ul className="divide-y text-sm">
            {q.terms.map((t) => (
              <li key={t.id} className="flex items-center justify-between gap-3 py-2">
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
          {q.notes ? <p className="border-t pt-3 text-xs whitespace-pre-line text-muted-foreground">{q.notes}</p> : null}
        </Card>
        <Card className="gap-2 text-sm p-(--card-pad)">
          <Line label={s.subtotal} value={q.subtotal} />
          <Line label={`${s.discountNominal} ${q.discountPercent}%`} value={`-${q.discountNominal}`} muted />
          {q.usePpn ? <Line label={`${s.ppnNominal} ${q.ppnPercent}%`} value={q.ppnNominal} muted /> : null}
          <div className="mt-1 border-t pt-2 text-base font-semibold">
            <Line label={s.total} value={q.total} />
          </div>
          <p className="text-xs text-muted-foreground">
            {s.validUntil} {formatDate(q.validUntil || null)}
          </p>
        </Card>
      </div>
    </div>
  );
}

function Line({ label, value, muted }: { label: string; value: string; muted?: boolean }) {
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
