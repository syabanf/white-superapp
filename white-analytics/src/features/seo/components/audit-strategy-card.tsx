import { Monitor, Smartphone } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { ScoreRing, RATING_LABEL } from "@/components/dashboard/charts/score-ring";
import { StatusBadge, ratingToKind } from "@/components/dashboard/status-badge";
import { rateCwv, type CwvMetric } from "@/lib/metrics";
import { formatDateTime, formatMs, formatNumber } from "@/lib/format";
import { t } from "@/i18n/id";
import type { AuditRow } from "@/features/seo/queries";

const CWV_ITEMS: { metric: CwvMetric; label: string; field: keyof Pick<AuditRow, "lcpMs" | "inpMs" | "cls" | "fcpMs" | "ttfbMs" | "tbtMs">; isMs: boolean }[] = [
  { metric: "lcpMs", label: t.seo.lcp, field: "lcpMs", isMs: true },
  { metric: "inpMs", label: t.seo.inp, field: "inpMs", isMs: true },
  { metric: "cls", label: t.seo.cls, field: "cls", isMs: false },
  { metric: "fcpMs", label: t.seo.fcp, field: "fcpMs", isMs: true },
  { metric: "ttfbMs", label: t.seo.ttfb, field: "ttfbMs", isMs: true },
  { metric: "tbtMs", label: t.seo.tbt, field: "tbtMs", isMs: true },
];

/** One strategy column (MOBILE/DESKTOP): 4 Lighthouse rings + Core Web Vitals grid. */
export function AuditStrategyCard({ audit, strategy }: { audit: AuditRow | null; strategy: "MOBILE" | "DESKTOP" }) {
  const Icon = strategy === "MOBILE" ? Smartphone : Monitor;
  const label = strategy === "MOBILE" ? t.seo.mobile : t.seo.desktop;
  return (
    <Card className="gap-3 py-0">
      <CardHeader className="flex flex-row items-start justify-between gap-3 space-y-0 px-5 pt-4 pb-0">
        <div>
          <CardTitle className="flex items-center gap-2 text-sm font-semibold">
            <Icon className="size-4 text-muted-foreground" /> {label}
          </CardTitle>
          <CardDescription className="mt-0.5 text-xs">
            {audit ? `${t.seo.lastAudit}: ${formatDateTime(audit.runAt)}` : t.seo.noAudit}
          </CardDescription>
        </div>
      </CardHeader>
      <CardContent className="space-y-4 px-5 pb-5">
        <div className="grid grid-cols-4 gap-2">
          <ScoreRing score={audit?.performance} label={t.seo.performance} size={72} />
          <ScoreRing score={audit?.seo} label={t.seo.seoScore} size={72} />
          <ScoreRing score={audit?.accessibility} label={t.seo.accessibility} size={72} />
          <ScoreRing score={audit?.bestPractices} label={t.seo.bestPractices} size={72} />
        </div>
        <div>
          <div className="mb-2 text-xs font-medium text-muted-foreground">{t.seo.cwv}</div>
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
            {CWV_ITEMS.map(({ metric, label: metricLabel, field, isMs }) => {
              const value = audit ? audit[field] : null;
              const rating = rateCwv(metric, value);
              return (
                <div key={metric} className="rounded-lg border bg-muted/30 px-3 py-2">
                  <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">{metricLabel}</div>
                  <div className="mt-1 flex flex-wrap items-center justify-between gap-1">
                    <span className="text-sm font-semibold tabular">{value == null ? "–" : isMs ? formatMs(value) : formatNumber(value, 2)}</span>
                    {rating ? <StatusBadge kind={ratingToKind(rating)}>{RATING_LABEL[rating]}</StatusBadge> : null}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
