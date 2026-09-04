import type { Metadata } from "next";
import Link from "next/link";
import { ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { ChartCard } from "@/components/dashboard/chart-card";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { EmptyState } from "@/components/dashboard/empty-state";
import { StatusBadge, type StatusKind } from "@/components/dashboard/status-badge";
import { requireClientAccess } from "@/lib/rbac";
import { formatDateTime, formatDuration, formatNumber } from "@/lib/format";
import { t } from "@/i18n/id";
import { s } from "@/features/seo/strings";
import { getAuditData, getIssueTrend, getSeoProperty } from "@/features/seo/queries";
import { AuditStrategyCard } from "@/features/seo/components/audit-strategy-card";
import { ScoreHistoryChart } from "@/features/seo/components/score-history-chart";
import { AuditDetailTabs } from "@/features/seo/components/audit-detail-tabs";
import { RunAuditButton } from "@/features/seo/components/run-audit-button";
import { AuditCadenceSelect } from "@/features/seo/components/audit-cadence-select";
import { IssueTrendChart } from "@/features/seo/components/issue-trend-chart";

export async function generateMetadata(props: PageProps<"/clients/[slug]/seo/audit">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.seo.audit}` };
}

const CRAWL_STATUS: Record<string, { kind: StatusKind; label: string }> = {
  SUCCESS: { kind: "good", label: "Sukses" },
  FAILED: { kind: "critical", label: "Gagal" },
  RUNNING: { kind: "info", label: "Berjalan" },
  PENDING: { kind: "neutral", label: "Menunggu" },
};

export default async function SeoAuditPage(props: PageProps<"/clients/[slug]/seo/audit">) {
  const { slug } = await props.params;
  const { client, canManage } = await requireClientAccess(slug);

  const property = await getSeoProperty(client.id);
  if (!property) {
    return (
      <>
        <PageHeader eyebrow={t.nav.seo} title={t.seo.audit} description={t.seo.auditSubtitle} />
        <EmptyState
          icon={<ShieldCheck />}
          title={t.seo.noProperty}
          description={t.seo.noPropertyDesc}
          action={
            <Button asChild size="sm">
              <Link href={`/clients/${slug}/settings`}>{t.seo.addProperty}</Link>
            </Button>
          }
        />
      </>
    );
  }

  const [data, issueTrend] = await Promise.all([getAuditData(property.id), getIssueTrend(property.id)]);
  const hasAudit = Boolean(data.latestMobile || data.latestDesktop);
  const crawlStatus = data.crawl ? (CRAWL_STATUS[data.crawl.status] ?? CRAWL_STATUS.PENDING!) : null;
  const crawlDurationSec =
    data.crawl?.finishedAt != null ? (new Date(data.crawl.finishedAt).getTime() - new Date(data.crawl.startedAt).getTime()) / 1000 : null;

  return (
    <>
      <PageHeader
        eyebrow={t.nav.seo}
        title={t.seo.audit}
        description={t.seo.auditSubtitle}
        actions={
          <div className="flex flex-wrap items-center gap-3">
            {canManage ? <AuditCadenceSelect clientId={client.id} propertyId={property.id} value={property.auditCadence} /> : null}
            <RunAuditButton clientId={client.id} propertyId={property.id} lastRunAt={data.lastRunAt} />
          </div>
        }
      />

      {!hasAudit ? (
        <EmptyState icon={<ShieldCheck />} title={t.seo.noAudit} description={t.seo.noAuditDesc} />
      ) : (
        <>
          <div className="grid gap-4 lg:grid-cols-2">
            <AuditStrategyCard audit={data.latestMobile} strategy="MOBILE" />
            <AuditStrategyCard audit={data.latestDesktop} strategy="DESKTOP" />
          </div>

          {data.history.length > 1 ? <ScoreHistoryChart history={data.history} /> : null}
        </>
      )}

      <Section
        title={t.seo.issues}
        description={t.seo.issuesDesc}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <StatusBadge kind="critical">{formatNumber(data.severityCounts.errors)} Error</StatusBadge>
            <StatusBadge kind="warning">
              {formatNumber(data.severityCounts.warnings)} {t.common.warning}
            </StatusBadge>
            <StatusBadge kind="neutral">
              {formatNumber(data.severityCounts.notices)} {t.common.notice}
            </StatusBadge>
          </div>
        }
      >
        <div className="grid gap-4 lg:grid-cols-3">
          <Card className="gap-3 py-0">
            <CardHeader className="px-5 pt-4 pb-0">
              <CardTitle className="text-sm font-semibold">{s.crawlSummary}</CardTitle>
              <CardDescription className="text-xs">
                {data.crawl ? `${t.seo.lastAudit}: ${formatDateTime(data.crawl.startedAt)}` : s.noCrawl}
              </CardDescription>
            </CardHeader>
            <CardContent className="px-5 pb-5">
              {data.crawl ? (
                <dl className="space-y-2.5 text-sm">
                  <div className="flex items-center justify-between gap-2">
                    <dt className="text-muted-foreground">{t.seo.crawlStatus}</dt>
                    <dd>{crawlStatus ? <StatusBadge kind={crawlStatus.kind}>{crawlStatus.label}</StatusBadge> : "–"}</dd>
                  </div>
                  <div className="flex items-center justify-between gap-2">
                    <dt className="text-muted-foreground">{t.seo.pagesCrawled}</dt>
                    <dd className="font-medium tabular">
                      {formatNumber(data.crawl.pagesCrawled)} / {formatNumber(data.crawl.maxPages)}
                    </dd>
                  </div>
                  <div className="flex items-center justify-between gap-2">
                    <dt className="text-muted-foreground">{s.duration}</dt>
                    <dd className="font-medium tabular">{crawlDurationSec == null ? "–" : formatDuration(crawlDurationSec)}</dd>
                  </div>
                  {data.crawl.error ? <p className="text-xs text-negative">{data.crawl.error}</p> : null}
                </dl>
              ) : (
                <p className="py-4 text-sm text-muted-foreground">{s.noCrawl}</p>
              )}
            </CardContent>
          </Card>
          <div className="lg:col-span-2">
            <ChartCard title={s.topCodes} description={s.topCodesDesc}>
              <HBarChart items={data.codeCounts.slice(0, 8).map((c) => ({ label: c.code, value: c.count }))} className="py-1" emptyLabel={t.common.noData} />
            </ChartCard>
          </div>
        </div>

        {issueTrend.length > 1 ? <IssueTrendChart history={issueTrend} /> : null}

        <AuditDetailTabs issues={data.issues} pages={data.pages} />
      </Section>
    </>
  );
}
