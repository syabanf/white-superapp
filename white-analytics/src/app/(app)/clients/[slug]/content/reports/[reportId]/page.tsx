import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft, Download, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { MarkdownLite } from "@/features/reports/components/ai-insight-card";
import { monthLabel, todayIn } from "@/features/publishing/time";
import type { DashCtx } from "@/features/content/components/dashboard/shared";
import { contentBoardHref } from "@/features/content/components/dashboard/links";
import { ContentDrillSheet } from "@/features/content/components/dashboard/drill";
import { bucketRange } from "@/features/erp/shared/drill/time";
import { RegenerateButton } from "@/features/content/components/reports/report-list";
import { ReportPostsTable } from "@/features/content/components/reports/report-parts";
import { ReportCharts } from "@/features/content/components/reports/report-charts";
import { kpiDelta } from "@/features/content/dashboard";
import { ContentOffline } from "@/features/content/components/offline";
import { contentFiles, num, shiftMonth } from "@/features/content/lib";
import { contentContext, getReport, listReports, loadContent } from "@/features/content/queries";
import { c, cd } from "@/features/content/strings";
import { formatCompact, formatDateTime, formatDecimal, formatNumber } from "@/lib/format";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content/reports/[reportId]">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await contentContext(slug);
  return { title: `${client.name} · ${t.nav.contentReports}` };
}

const card = "rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow)";

export default async function ContentReportPage(props: PageProps<"/clients/[slug]/content/reports/[reportId]">) {
  const { slug, reportId } = await props.params;
  const { client, canWrite, names } = await contentContext(slug);
  const back = (
    <Button asChild variant="outline" size="sm" className="group/nudge">
      <Link href={`/clients/${slug}/content/reports`}>
        <ArrowLeft className="nudge nudge-back size-3.5" /> {t.nav.contentReports}
      </Link>
    </Button>
  );
  const res = await loadContent(() => Promise.all([getReport(client.id, reportId), listReports(client.id)]));
  if (!res.ok) {
    return (
      <>
        <PageHeader eyebrow={c.reportEyebrow} title={c.reportsTitle} actions={back} />
        <ContentOffline message={res.message} />
      </>
    );
  }

  const [r, reports] = res.data;
  const prevMonth = shiftMonth(r.month, -1);
  const prev = reports.find((x) => x.month === prevMonth);
  const d = (cur: number | string, before: number | string | undefined) => (prev && before !== undefined ? kpiDelta(cur, before) : null);
  const prevNote = prev ? cd.vsPrevMonth(monthLabel(prevMonth)) : undefined;
  const { from, to } = bucketRange(r.month, "month");
  // drill scope: the report month (items planned in it, like the report counts)
  const ctx: DashCtx = {
    base: `/clients/${slug}/content`,
    clientId: client.id,
    query: "",
    names,
    compare: false,
    plans: [],
    from,
    to,
    periodLabel: monthLabel(r.month),
    today: todayIn(client.timezone),
  };
  const board = (patch: Parameters<typeof contentBoardHref>[1] = {}) => contentBoardHref(ctx, { dateFrom: from, dateTo: to, ...patch });

  return (
    <>
      <PageHeader
        eyebrow={c.reportEyebrow}
        title={monthLabel(r.month)}
        description={`${c.colUpdated} ${formatDateTime(r.updatedAt, client.timezone)}`}
        actions={
          <>
            {canWrite ? <RegenerateButton slug={slug} month={r.month} /> : null}
            <Button asChild size="sm">
              <a href={contentFiles.reportPdf(client.id, r.id)} target="_blank" rel="noreferrer">
                <Download className="size-3.5" /> {c.pdf}
              </a>
            </Button>
            {back}
          </>
        }
      />

      <KpiGrid>
        <KpiTile
          label={c.published}
          value={`${r.published} / ${r.planned}`}
          delta={d(r.published, prev?.published)}
          href={board({ status: ["PUBLISHED"] })}
          caption={r.planned ? `${formatDecimal((r.published / r.planned) * 100)}% · ${prevNote ?? ""}` : prevNote}
        />
        <KpiTile label={c.onTime} value={`${formatDecimal(r.onTimeRate)}%`} delta={d(r.onTimeRate, prev?.onTimeRate)} href={board()} caption={c.onTimeCaption(r.deadlinesMet, r.deadlinesTotal)} />
        <KpiTile
          label={c.revisionsTotal}
          value={formatNumber(r.revisionsTotal)}
          delta={d(r.revisionsTotal, prev?.revisionsTotal)}
          lowerIsBetter
          href={board({ rounds: "any" })}
          caption={c.revisionAvg(formatDecimal(r.revisionAvg))}
        />
        <KpiTile label={c.failed} value={formatNumber(r.failed)} delta={d(r.failed, prev?.failed)} lowerIsBetter href={board({ status: ["FAILED"] })} caption={`${c.captionVersions}: ${r.captionVersions}`} />
        {r.engagementAvailable ? (
          <>
            <KpiTile label={c.engagement} value={formatCompact(num(r.totalEngagements))} delta={d(r.totalEngagements, prev?.totalEngagements)} href={board({ status: ["PUBLISHED"] })} caption={prevNote} />
            <KpiTile label={c.reach} value={formatCompact(num(r.totalReach))} delta={d(r.totalReach, prev?.totalReach)} href={board({ status: ["PUBLISHED"] })} caption={prevNote} />
            <KpiTile
              label={cd.er}
              value={`${formatDecimal(num(r.totalReach) ? (num(r.totalEngagements) / num(r.totalReach)) * 100 : 0)}%`}
              caption={cd.measuredPosts((r.posts ?? []).filter((p) => p.hasMetrics).length)}
              href={board({ status: ["PUBLISHED"] })}
            />
            <KpiTile label={c.scheduledCount} value={formatNumber(r.scheduled)} href={board({ status: ["SCHEDULED"] })} caption={`${c.inProgress} ${r.inProgress}`} />
          </>
        ) : null}
      </KpiGrid>

      <ReportCharts report={r} ctx={ctx} />
      <ContentDrillSheet ctx={ctx} posts={r.posts ?? []} granularity="week" />

      <div className="grid gap-6 [&>*]:min-w-0">
        <div className="min-w-0 space-y-6">
          <Section index="01" title={c.summary}>
            <div className={card}>
              {r.summary ? (
                <div className="prose prose-sm max-w-none text-sm text-foreground/90 dark:prose-invert [&_h2]:mt-3 [&_h2]:text-sm [&_h2]:font-semibold [&_ol]:pl-4 [&_ul]:pl-4">
                  <MarkdownLite text={r.summary} />
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">{c.none}</p>
              )}
              {r.aiModel ? (
                <p className="mt-3 flex items-center gap-1 text-xs text-muted-foreground">
                  <Sparkles className="size-3" /> {r.aiModel === "mock" ? c.summaryMock : c.summaryModel(r.aiModel)}
                </p>
              ) : null}
            </div>
          </Section>

          {r.engagementAvailable ? (
            <>
              <Section index="02" title={c.topPosts}>
                <div className="rounded-[1.5rem] bg-card py-2 shadow-(--card-shadow)">
                  <ReportPostsTable posts={r.top ?? []} slug={slug} />
                </div>
              </Section>
              <Section index="03" title={c.bottomPosts}>
                <div className="rounded-[1.5rem] bg-card py-2 shadow-(--card-shadow)">
                  <ReportPostsTable posts={r.bottom ?? []} slug={slug} />
                </div>
              </Section>
            </>
          ) : (
            <p className="rounded-xl border border-dashed px-3 py-6 text-center text-sm text-muted-foreground">{c.engagementUnavailable}</p>
          )}
          <Section index="04" title={c.allPosts}>
            <div className="rounded-[1.5rem] bg-card py-2 shadow-(--card-shadow)">
              <ReportPostsTable posts={r.posts ?? []} slug={slug} />
            </div>
          </Section>
        </div>
      </div>
    </>
  );
}
