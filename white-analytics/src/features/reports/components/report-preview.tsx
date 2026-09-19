import type { ReactNode } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import type { Delta } from "@/lib/metrics";
import type { ReportSection } from "@/features/reports/extras";

export type PreviewKpi = {
  label: string;
  value: string;
  delta?: Delta | null;
  lowerIsBetter?: boolean;
};

function PreviewKpis({ kpis }: { kpis: PreviewKpi[] }) {
  return (
    <KpiGrid cols={4}>
      {kpis.map((k) => (
        <KpiTile key={k.label} label={k.label} value={k.value} delta={k.delta} lowerIsBetter={k.lowerIsBetter} className="shadow-none" />
      ))}
    </KpiGrid>
  );
}

/** Same sub-block the PDF prints under a module: heading, KPI tiles, optional table. */
function SectionPreview({ section }: { section: ReportSection }) {
  const table = section.table && section.table.rows.length > 0 ? section.table : null;
  return (
    <div className="space-y-3 border-t pt-4 first:border-t-0 first:pt-0">
      <h4 className="label-mono">{section.title}</h4>
      <PreviewKpis kpis={section.kpis} />
      {table ? (
        <div className="space-y-2">
          <p className="text-xs font-semibold">{table.title}</p>
          <Table>
            <TableHeader>
              <TableRow>
                {table.columns.map((c) => (
                  <TableHead key={c.label} className={c.align === "right" ? "text-right" : undefined}>
                    {c.label}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {table.rows.map((row, ri) => (
                <TableRow key={ri}>
                  {row.map((cell, ci) => (
                    <TableCell key={ci} className={table.columns[ci]?.align === "right" ? "tabular text-right" : "max-w-[18rem] truncate"}>
                      {cell}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : null}
    </div>
  );
}

/** Server-rendered KPI block for the builder preview (one card per module). */
export function ModulePreviewCard({ title, icon, kpis, sections = [] }: { title: string; icon?: ReactNode; kpis: PreviewKpi[]; sections?: ReportSection[] }) {
  return (
    <Card className="gap-3 py-0">
      <CardHeader className="px-5 pt-4 pb-0">
        <CardTitle className="flex items-center gap-2 text-sm font-semibold">
          {icon} {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4 px-5 pb-4">
        {kpis.length > 0 ? <PreviewKpis kpis={kpis} /> : null}
        {sections.map((s) => (
          <SectionPreview key={s.key} section={s} />
        ))}
      </CardContent>
    </Card>
  );
}
