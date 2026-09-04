import { CheckCircle2, CircleAlert, Loader2, Clock } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { formatRelative } from "@/lib/format";
import { t } from "@/i18n/id";
import type { OverviewSyncJob } from "@/features/overview/queries";

const KIND_LABEL: Record<string, string> = {
  SOCIAL_SNAPSHOT: "Snapshot sosial",
  SEO_GSC: "Search Console",
  SEO_GA4: "GA4",
  SEO_AUDIT: "Audit situs",
  ADS_INSIGHTS: "Meta Ads insight",
  ADS_CSV_IMPORT: "Impor CSV Ads",
};

export function SyncActivity({ jobs }: { jobs: OverviewSyncJob[] }) {
  return (
    <Card className="gap-3 py-0">
      <CardHeader className="px-5 pt-4 pb-0">
        <CardTitle className="text-sm font-semibold">{t.overview.syncActivity}</CardTitle>
        <CardDescription className="text-xs">{t.common.lastSynced}</CardDescription>
      </CardHeader>
      <CardContent className="px-5 pb-4">
        {jobs.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t.overview.noSync}</p>
        ) : (
          <ul className="divide-y">
            {jobs.map((j) => {
              const Icon = j.status === "SUCCESS" ? CheckCircle2 : j.status === "FAILED" ? CircleAlert : j.status === "RUNNING" ? Loader2 : Clock;
              const color = j.status === "SUCCESS" ? "var(--status-good)" : j.status === "FAILED" ? "var(--status-critical)" : "var(--muted-foreground)";
              return (
                <li key={j.id} className="flex items-start gap-3 py-2 text-sm">
                  <Icon className={j.status === "RUNNING" ? "mt-0.5 size-4 animate-spin" : "mt-0.5 size-4"} style={{ color }} />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-medium">{KIND_LABEL[j.kind] ?? j.kind}</span>
                      <span className="shrink-0 text-xs text-muted-foreground">{formatRelative(j.startedAt)}</span>
                    </div>
                    <p className="truncate text-xs text-muted-foreground">{j.error ?? j.message ?? "–"}</p>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}
