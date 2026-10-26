import Link from "next/link";
import { Card } from "@/components/ui/card";
import { formatDate } from "@/lib/format";
import { Money } from "@/features/erp/shared/money-display";
import type { Ledger } from "../types";
import { fs } from "../strings";

/** Account ledger with opening, running and closing balance. Server-safe and print-friendly. */
export function LedgerTable({ ledger }: { ledger: Ledger }) {
  return (
    <Card className="gap-0 overflow-hidden p-0 print:shadow-none">
      <div className="grid grid-cols-2 gap-3 border-b px-5 py-3 text-sm sm:grid-cols-4">
        <div>
          <p className="label-mono text-muted-foreground">{fs.cash.opening}</p>
          <Money value={ledger.openingBalance} signed className="font-semibold" />
        </div>
        <div>
          <p className="label-mono text-muted-foreground">{fs.reports.totalDebit}</p>
          <Money value={ledger.totalDebit} className="font-semibold" />
        </div>
        <div>
          <p className="label-mono text-muted-foreground">{fs.reports.totalCredit}</p>
          <Money value={ledger.totalCredit} className="font-semibold" />
        </div>
        <div>
          <p className="label-mono text-muted-foreground">{fs.cash.closing}</p>
          <Money value={ledger.closingBalance} signed className="font-semibold" />
        </div>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[760px] text-sm">
          <thead className="bg-muted/40 text-xs text-muted-foreground">
            <tr>
              <th className="px-4 py-2 text-left font-medium">{fs.date}</th>
              <th className="px-4 py-2 text-left font-medium">{fs.journals.entryNo}</th>
              <th className="px-4 py-2 text-left font-medium">{fs.description}</th>
              <th className="px-4 py-2 text-right font-medium">{fs.debit}</th>
              <th className="px-4 py-2 text-right font-medium">{fs.credit}</th>
              <th className="px-4 py-2 text-right font-medium">{fs.cash.running}</th>
            </tr>
          </thead>
          <tbody>
            <tr className="border-t bg-muted/20 text-muted-foreground">
              <td className="px-4 py-2" colSpan={5}>
                {fs.cash.opening}
              </td>
              <td className="px-4 py-2 text-right">
                <Money value={ledger.openingBalance} signed />
              </td>
            </tr>
            {ledger.lines.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-muted-foreground">
                  {fs.cash.empty}
                </td>
              </tr>
            ) : (
              ledger.lines.map((l) => (
                <tr key={l.lineId} className="border-t">
                  <td className="px-4 py-2 whitespace-nowrap">{formatDate(l.entryDate)}</td>
                  <td className="px-4 py-2">
                    <Link href={`/erp/finance/journals/${l.entryId}`} className="font-mono text-xs hover:underline">
                      {l.entryNo}
                    </Link>
                  </td>
                  <td className="px-4 py-2">
                    {l.description}
                    {l.memo ? <span className="block text-xs text-muted-foreground">{l.memo}</span> : null}
                  </td>
                  <td className="px-4 py-2 text-right">
                    <Money value={l.debit} muteZero />
                  </td>
                  <td className="px-4 py-2 text-right">
                    <Money value={l.credit} muteZero />
                  </td>
                  <td className="px-4 py-2 text-right">
                    <Money value={l.runningBalance} signed />
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </Card>
  );
}
