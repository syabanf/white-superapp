import { CheckCircle2, CircleAlert, Loader2, Clock } from "lucide-react";
import Link from "next/link";
import { formatRelative } from "@/lib/format";
import { t } from "@/i18n/id";
import type { OverviewSyncJob } from "@/features/overview/queries";

const KIND_LABEL: Record<string, string> = {
  SOCIAL_SNAPSHOT: "Social media",
  SEO_GSC: "Search Console",
  SEO_GA4: "Google Analytics 4",
  SEO_AUDIT: "Audit situs",
  SEO_SUITE_DAILY: "SEO suite harian",
  SEO_SUITE_RANKS: "Peringkat kata kunci",
  SEO_SUITE_BACKLINKS: "Backlink",
  SEO_SUITE_DOMAINS: "Domain kompetitor",
  ADS_INSIGHTS: "Meta Ads",
  ADS_CSV_IMPORT: "Impor CSV Ads",
  PUBLISH: "Publikasi terjadwal",
};

/** Job detail line. "mock" runs read as demo mode; a trailing "(mock)" note is dropped. */
function detailOf(j: OverviewSyncJob): string {
  if (j.error) return j.error;
  if (!j.message || j.message === "mock") return "Mode demo";
  return j.message.replace(/\s*\(mock\)\s*$/i, "");
}

export function SyncActivity({ jobs, baseHref }: { jobs: OverviewSyncJob[]; baseHref: string }) {
  return (
    <section className="border-t pt-5" aria-labelledby="sync-activity-title">
      <h2 id="sync-activity-title" className="text-[15px] font-semibold tracking-[-0.015em]">
        {t.overview.syncActivity}
      </h2>
      <div className="mt-3">
        {jobs.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t.overview.noSync}</p>
        ) : (
          <ul className="divide-y">
            {jobs.map((j) => {
              const Icon =
                j.status === "SUCCESS"
                  ? CheckCircle2
                  : j.status === "FAILED"
                    ? CircleAlert
                    : j.status === "RUNNING"
                      ? Loader2
                      : Clock;
              const color =
                j.status === "SUCCESS"
                  ? "var(--status-good)"
                  : j.status === "FAILED"
                    ? "var(--status-critical)"
                    : "var(--muted-foreground)";
              return (
                <li key={j.id} className="flex items-start gap-3 py-2 text-sm">
                  <Icon
                    className={j.status === "RUNNING" ? "mt-0.5 size-4 animate-spin" : "mt-0.5 size-4"}
                    style={{ color }}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-medium">{KIND_LABEL[j.kind] ?? j.kind}</span>
                      <span className="shrink-0 text-xs text-muted-foreground">
                        {formatRelative(j.startedAt)}
                      </span>
                    </div>
                    <div className="flex items-center justify-between gap-2">
                      <p className="truncate text-xs text-muted-foreground">{detailOf(j)}</p>
                      {j.status === "FAILED" ? (
                        <Link
                          href={`${baseHref}/${modulePath(j.kind)}`}
                          className="shrink-0 text-xs font-medium text-brand-ink hover:underline"
                        >
                          Coba sinkronkan ulang
                        </Link>
                      ) : null}
                    </div>
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </section>
  );
}

function modulePath(kind: string): string {
  if (kind === "PUBLISH") return "publish";
  if (kind.startsWith("SOCIAL")) return "social";
  if (kind.startsWith("ADS")) return "ads";
  return "seo";
}
