import type { ReactNode } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import type { Delta } from "@/lib/metrics";

export type PreviewKpi = {
  label: string;
  value: string;
  delta?: Delta | null;
  lowerIsBetter?: boolean;
};

/** Server-rendered KPI block for the builder preview (one card per module). */
export function ModulePreviewCard({ title, icon, kpis }: { title: string; icon?: ReactNode; kpis: PreviewKpi[] }) {
  return (
    <Card className="gap-3 py-0">
      <CardHeader className="px-5 pt-4 pb-0">
        <CardTitle className="flex items-center gap-2 text-sm font-semibold">
          {icon} {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="px-5 pb-4">
        <KpiGrid cols={4}>
          {kpis.map((k) => (
            <KpiTile key={k.label} label={k.label} value={k.value} delta={k.delta} lowerIsBetter={k.lowerIsBetter} className="shadow-none" />
          ))}
        </KpiGrid>
      </CardContent>
    </Card>
  );
}
