import Link from "next/link";
import { Money } from "@/features/erp/shared/money-display";
import { parseMoney } from "@/features/erp/shared/money";
import { fs } from "../strings";
import type { SubledgerRecon } from "../types";
import { ReportPaper } from "./report-views";

const s = fs.subledgerRecon;

export function SubledgerReconView({ data }: { data: SubledgerRecon }) {
  const rows = data.rows ?? [];
  const pending = data.pendingPostings ?? 0;
  return (
    <ReportPaper title={s.title} subtitle={s.subtitle}>
      <table className="w-full min-w-[640px] text-sm">
        <thead className="text-xs text-muted-foreground">
          <tr className="border-b">
            <th className="py-2 text-left font-medium">{s.ledger}</th>
            <th className="py-2 text-right font-medium">{s.generalLedger}</th>
            <th className="py-2 text-right font-medium">{s.subledger}</th>
            <th className="py-2 text-right font-medium">{s.difference}</th>
            <th className="py-2 text-right font-medium">{s.unjournaled}</th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {rows.map((row) => (
            <tr key={row.kind}>
              <td className="py-2">
                <span className="block font-medium">{s.kinds[row.kind]}</span>
                <span className="block text-xs text-muted-foreground">{(row.accounts ?? []).join(", ") || s.noAccount}</span>
                <span className="block text-xs text-muted-foreground">
                  {s.opening} <Money value={row.opening} /> · {s.recon} <Money value={row.recon} />
                </span>
              </td>
              <td className="py-2 text-right"><Money value={row.generalLedger} /></td>
              <td className="py-2 text-right"><Money value={row.subledger} /></td>
              <td className={parseMoney(row.difference) === 0 ? "py-2 text-right" : "py-2 text-right font-semibold text-negative"}>
                <Money value={row.difference} signed />
              </td>
              <td className="py-2 text-right">
                {row.unjournaledCount ? (
                  <span>
                    {row.unjournaledCount} · <Money value={row.unjournaledAmount} />
                  </span>
                ) : (
                  "–"
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="flex flex-col gap-1 py-3 text-xs text-muted-foreground">
        <p>{s.note}</p>
        {pending ? (
          <p>
            {s.pending(pending)}{" "}
            <Link href="/erp/finance/journals?tab=logs" className="underline underline-offset-4">
              {s.openLogs}
            </Link>
          </p>
        ) : null}
      </div>
    </ReportPaper>
  );
}
