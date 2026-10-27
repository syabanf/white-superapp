"use client";

import { ChartCard } from "@/components/dashboard/chart-card";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { platformLabel } from "@/features/social/components/platform-icon";
import { dayLabel } from "@/features/publishing/time";
import { GroupedBars, PeriodChart, WaterfallChart } from "@/features/content/components/dashboard/charts";
import { engagementRate } from "@/features/content/dashboard";
import { num } from "@/features/content/lib";
import { c, cd } from "@/features/content/strings";
import type { ContentFormat, ContentReport, ReportBreakdown, ReportPost } from "@/features/content/types";
import { formatNumber, formatPercent } from "@/lib/format";
import type { ItemStatus, Platform } from "@/features/content/types";
import type { DashCtx } from "@/features/content/components/dashboard/shared";
import { useContentDrill } from "@/features/content/components/dashboard/drill";
import { periodRange } from "@/features/content/components/dashboard/links";

/** Engagement per post by a key of the post, for the measured posts. */
function engagementBy(posts: ReportPost[], key: (p: ReportPost) => string, label: (k: string) => string) {
  const acc = new Map<string, { eng: number; reach: number; n: number }>();
  for (const p of posts) {
    if (!p.hasMetrics) continue;
    const a = acc.get(key(p)) ?? { eng: 0, reach: 0, n: 0 };
    acc.set(key(p), { eng: a.eng + num(p.engagements), reach: a.reach + num(p.reach), n: a.n + 1 });
  }
  return [...acc.entries()]
    .map(([k, a]) => ({ key: k, label: label(k), value: Math.round(a.eng / a.n), secondary: `${cd.erShort} ${formatPercent(engagementRate(a.eng, a.reach), 1)}` }))
    .toSorted((a, b) => b.value - a.value);
}

/** Days of the report month, with engagement of the posts published that day. */
function dailyEngagement(month: string, posts: ReportPost[]) {
  const [y, m] = month.split("-").map(Number) as [number, number];
  const last = new Date(Date.UTC(y, m, 0)).getUTCDate();
  const byDay = new Map<string, { eng: number }>();
  for (const p of posts) {
    if (!p.hasMetrics || !p.publishedAt) continue;
    const day = new Date(new Date(p.publishedAt).getTime() + 7 * 3600_000).toISOString().slice(0, 10);
    byDay.set(day, { eng: (byDay.get(day)?.eng ?? 0) + num(p.engagements) });
  }
  return Array.from({ length: last }, (_, i) => {
    const day = `${month}-${String(i + 1).padStart(2, "0")}`;
    return { day, label: dayLabel(day), engagements: byDay.get(day)?.eng ?? 0 };
  });
}

function breakdown(rows: ReportBreakdown[], label: (k: string) => string) {
  return rows.map((r) => ({ key: r.key, label: label(r.key), planned: r.planned, published: r.published }));
}

const IN_PROGRESS: ItemStatus[] = ["IDEA", "COPYWRITING", "DESIGN", "REVIEW", "REVISION", "APPROVED"];
const WATERFALL_STATUS: (ItemStatus[] | undefined)[] = [undefined, IN_PROGRESS, ["SCHEDULED"], ["FAILED"], ["PUBLISHED"]];

/** Report charts; every mark opens the content drill sheet scoped to the report month. */
export function ReportCharts({ report: r, ctx }: { report: ContentReport; ctx: DashCtx }) {
  const posts = r.posts ?? [];
  const formatName = (k: string) => c.format[k as ContentFormat] ?? k;
  const daily = dailyEngagement(r.month, posts);
  const drill = useContentDrill(ctx);
  const month = periodRange(ctx);
  const published = { ...month, status: ["PUBLISHED" as ItemStatus] };
  return (
    <div className="page-stack">
      <div className="grid gap-(--gap) [&>*]:min-w-0 lg:grid-cols-2">
        <ChartCard title={cd.waterfallTitle} description={cd.waterfallDesc}>
          <WaterfallChart
            steps={[
              { label: cd.planned, value: r.planned, kind: "total" },
              { label: c.inProgress, value: r.inProgress, kind: "minus" },
              { label: c.scheduledCount, value: r.scheduled, kind: "minus" },
              { label: c.failed, value: r.failed, kind: "minus" },
              { label: cd.published, value: r.published, kind: "total" },
            ]}
            summary={cd.publishedSummary(r.published, r.planned)}
            onSelect={(_s, i) => drill({ ...month, status: WATERFALL_STATUS[i] })}
          />
        </ChartCard>
        <ChartCard title={cd.dailyEngagement} description={cd.dailyEngagementDesc}>
          {r.engagementAvailable ? (
            <PeriodChart
              data={daily}
              onSelect={(row) => drill({ dateFrom: String(row.day), dateTo: String(row.day), dateBy: "published", status: ["PUBLISHED"] })}
              series={[{ key: "engagements", label: cd.engagement, type: "bar", format: (v) => formatNumber(v) }]}
              summary={cd.dailyEngagement}
            />
          ) : (
            <p className="py-10 text-center text-sm text-muted-foreground">{c.engagementUnavailable}</p>
          )}
        </ChartCard>
      </div>
      <div className="grid gap-(--gap) [&>*]:min-w-0 lg:grid-cols-2">
        <ChartCard title={c.perPlatform} description={cd.breakdownPlanDesc}>
          <GroupedBars
            data={breakdown(r.perPlatform ?? [], platformLabel)}
            summary={c.perPlatform}
            onSelect={(row, key) => drill({ ...(key === "published" ? published : month), platform: row.key as Platform })}
          />
          <div className="mt-5 border-t pt-4">
            <p className="mb-3 text-xs font-medium text-muted-foreground">{cd.perPost}</p>
            <HBarChart
              items={engagementBy(posts, (p) => p.platform, platformLabel)}
              format={(v) => formatNumber(v)}
              emptyLabel={cd.noEngagement}
              onSelect={(it) => drill({ ...published, platform: it.key as Platform })}
            />
          </div>
        </ChartCard>
        <ChartCard title={c.perFormat} description={cd.breakdownPlanDesc}>
          <GroupedBars
            data={breakdown(r.perFormat ?? [], formatName)}
            summary={c.perFormat}
            onSelect={(row, key) => drill({ ...(key === "published" ? published : month), format: row.key as ContentFormat })}
          />
          <div className="mt-5 border-t pt-4">
            <p className="mb-3 text-xs font-medium text-muted-foreground">{cd.perPost}</p>
            <HBarChart
              items={engagementBy(posts, (p) => p.format, formatName)}
              format={(v) => formatNumber(v)}
              color={slotColor(2)}
              emptyLabel={cd.noEngagement}
              onSelect={(it) => drill({ ...published, format: it.key as ContentFormat })}
            />
          </div>
        </ChartCard>
      </div>
    </div>
  );
}
