import { Money } from "@/features/erp/shared/money-display";
import { parseMoney, toMoneyString } from "@/features/erp/shared/money";
import { formatDate } from "@/lib/format";
import { fs } from "../strings";
import type { ProjectProfit, ProjectProfitRow } from "../types";
import { ReportPaper } from "./report-views";

const s = fs.projects;

/** Merges projects into one row per client. */
export function byClient(rows: ProjectProfitRow[]): ProjectProfitRow[] {
  const merged = new Map<string, ProjectProfitRow>();
  for (const row of rows) {
    const key = row.tagId ? row.clientName || row.tagName : "";
    const seen = merged.get(key);
    merged.set(key, {
      tagId: row.tagId,
      tagName: key,
      clientName: "",
      revenue: toMoneyString(parseMoney(seen?.revenue ?? "0") + parseMoney(row.revenue)),
      cost: toMoneyString(parseMoney(seen?.cost ?? "0") + parseMoney(row.cost)),
      profit: toMoneyString(parseMoney(seen?.profit ?? "0") + parseMoney(row.profit)),
    });
  }
  return [...merged.values()];
}

const margin = (r: Pick<ProjectProfitRow, "revenue" | "profit">) => {
  const revenue = parseMoney(r.revenue);
  return revenue > 0 ? `${((parseMoney(r.profit) / revenue) * 100).toFixed(1)}%` : "–";
};

export function ProjectProfitView({ data, grouped }: { data: ProjectProfit; grouped: boolean }) {
  const rows = grouped ? byClient(data.rows ?? []) : (data.rows ?? []);
  const first = grouped ? s.client : s.name;
  return (
    <ReportPaper title={s.profitTitle} subtitle={`${formatDate(data.dateFrom)} – ${formatDate(data.dateTo)}`}>
      {rows.length === 0 ? (
        <p className="py-6 text-center text-sm text-muted-foreground">{s.noProfit}</p>
      ) : (
        <>
          <table className="w-full min-w-[640px] text-sm">
            <thead className="text-xs text-muted-foreground">
              <tr className="border-b">
                <th className="py-2 text-left font-medium">{first}</th>
                {grouped ? null : <th className="py-2 text-left font-medium">{s.client}</th>}
                <th className="py-2 text-right font-medium">{s.revenue}</th>
                <th className="py-2 text-right font-medium">{s.cost}</th>
                <th className="py-2 text-right font-medium">{s.profit}</th>
                <th className="py-2 text-right font-medium">{s.margin}</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {rows.map((row) => (
                <tr key={row.tagId || row.tagName || "none"}>
                  <td className="py-2">{row.tagName || <span className="text-muted-foreground">{s.untagged}</span>}</td>
                  {grouped ? null : <td className="py-2 text-muted-foreground">{row.clientName || "–"}</td>}
                  <td className="py-2 text-right"><Money value={row.revenue} muteZero /></td>
                  <td className="py-2 text-right"><Money value={row.cost} muteZero /></td>
                  <td className="py-2 text-right"><Money value={row.profit} signed /></td>
                  <td className="py-2 text-right tabular">{margin(row)}</td>
                </tr>
              ))}
            </tbody>
            <tfoot className="border-t font-semibold">
              <tr>
                <td className="py-2" colSpan={grouped ? 1 : 2}>{s.total}</td>
                <td className="py-2 text-right"><Money value={data.totalRevenue} /></td>
                <td className="py-2 text-right"><Money value={data.totalCost} /></td>
                <td className="py-2 text-right"><Money value={data.totalProfit} signed /></td>
                <td className="py-2 text-right tabular">{margin({ revenue: data.totalRevenue, profit: data.totalProfit })}</td>
              </tr>
            </tfoot>
          </table>
          <p className="py-3 text-xs text-muted-foreground">{s.profitNote}</p>
        </>
      )}
    </ReportPaper>
  );
}
