import type { Metadata } from "next";
import Link from "next/link";
import { ExternalLink, FileText, History, Megaphone, Search, Share2, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { AiInsightCard } from "@/features/reports/components/ai-insight-card";
import { InsightGenerator } from "@/features/reports/components/insight-generator";
import { ModulePreviewCard, type PreviewKpi } from "@/features/reports/components/report-preview";
import { ReportBuilderForm } from "@/features/reports/components/report-builder-form";
import { ReportHistoryTable, type ReportRow } from "@/features/reports/components/report-history-table";
import { ShareSummaryCard } from "@/features/reports/components/share-summary-card";
import { getOverviewAds, getOverviewSeo, getOverviewSocial } from "@/features/overview/queries";
import { getReportSections } from "@/features/reports/sections";
import { getLatestInsight, listReports, type InsightModule } from "@/features/reports/queries";
import { requireClientAccess } from "@/lib/rbac";
import { getRange } from "@/lib/range-params";
import { toISODate } from "@/lib/dates";
import { formatCompact, formatCurrency, formatDateRange, formatNumber, formatPercent } from "@/lib/format";
import { t } from "@/i18n/id";
import { rs } from "@/features/reports/strings";

const TABS = ["builder", "history", "insight", "share"] as const;
type TabKey = (typeof TABS)[number];

export async function generateMetadata(props: PageProps<"/clients/[slug]/reports">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.reports.title}` };
}

function monthTitle(d: Date): string {
  return new Intl.DateTimeFormat("id-ID", { month: "long", year: "numeric", timeZone: "UTC" }).format(d);
}

export default async function ReportsPage(props: PageProps<"/clients/[slug]/reports">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, canManage } = await requireClientAccess(slug);
  const { range, previous, compare } = getRange(sp);

  const tabRaw = Array.isArray(sp.tab) ? sp.tab[0] : sp.tab;
  const tab: TabKey = TABS.includes(tabRaw as TabKey) ? (tabRaw as TabKey) : "builder";
  const from = toISODate(range.from);
  const to = toISODate(range.to);
  const rangeLabel = formatDateRange(range.from, range.to);

  // Preserve the global range params in tab links.
  const keep = new URLSearchParams();
  for (const key of ["from", "to", "preset", "compare"]) {
    const v = Array.isArray(sp[key]) ? sp[key]?.[0] : (sp[key] as string | undefined);
    if (v) keep.set(key, v);
  }
  const tabHref = (k: TabKey) => {
    const params = new URLSearchParams(keep);
    params.set("tab", k);
    return `/clients/${slug}/reports?${params.toString()}`;
  };

  const shareUrl = `${process.env.NEXT_PUBLIC_APP_URL ?? "http://localhost:3000"}/share/${slug}`;

  return (
    <>
      <PageHeader
        eyebrow={t.nav.reports}
        title={t.reports.title}
        description={`${client.name} · ${t.reports.subtitle}`}
        actions={
          <Button asChild variant="outline" size="sm">
            <a href={`/share/${slug}`} target="_blank" rel="noreferrer">
              <ExternalLink className="size-4" /> {t.reports.openShare}
            </a>
          </Button>
        }
      />

      <Tabs value={tab} className="min-w-0 overflow-x-auto pb-1 scrollbar-thin">
        <TabsList className="min-w-max">
          <TabsTrigger value="builder" asChild>
            <Link href={tabHref("builder")}>
              <FileText className="size-3.5" /> {t.reports.builder}
            </Link>
          </TabsTrigger>
          <TabsTrigger value="history" asChild>
            <Link href={tabHref("history")}>
              <History className="size-3.5" /> {t.reports.history}
            </Link>
          </TabsTrigger>
          <TabsTrigger value="insight" asChild>
            <Link href={tabHref("insight")}>
              <Sparkles className="size-3.5" /> {t.reports.aiInsight}
            </Link>
          </TabsTrigger>
          <TabsTrigger value="share" asChild>
            <Link href={tabHref("share")}>
              <Share2 className="size-3.5" /> {t.reports.share}
            </Link>
          </TabsTrigger>
        </TabsList>
      </Tabs>

      {tab === "builder" ? (
        <BuilderTab
          slug={slug}
          clientId={client.id}
          clientName={client.name}
          currency={client.currency}
          range={range}
          previous={previous}
          compare={compare}
          from={from}
          to={to}
          rangeLabel={rangeLabel}
        />
      ) : null}
      {tab === "history" ? <HistoryTab slug={slug} clientId={client.id} canManage={canManage} /> : null}
      {tab === "insight" ? (
        <InsightTab
          slug={slug}
          clientId={client.id}
          range={range}
          from={from}
          to={to}
          rangeLabel={rangeLabel}
        />
      ) : null}
      {tab === "share" ? (
        <ShareSummaryCard
          slug={slug}
          enabled={client.shareEnabled}
          pinSet={Boolean(client.sharePinHash)}
          modules={client.shareModules}
          shareUrl={shareUrl}
        />
      ) : null}
    </>
  );
}

async function BuilderTab({
  slug,
  clientId,
  clientName,
  currency,
  range,
  previous,
  compare,
  from,
  to,
  rangeLabel,
}: {
  slug: string;
  clientId: string;
  clientName: string;
  currency: string;
  range: { from: Date; to: Date };
  previous: { from: Date; to: Date };
  compare: boolean;
  from: string;
  to: string;
  rangeLabel: string;
}) {
  const [social, seo, ads, insight, extras] = await Promise.all([
    getOverviewSocial(clientId, range, previous),
    getOverviewSeo(clientId, range, previous),
    getOverviewAds(clientId, range, previous),
    getLatestInsight(clientId, "OVERVIEW", range),
    getReportSections({ clientId, range, previous, compare, modules: ["SOCIAL", "SEO"], lang: "id" }),
  ]);
  const d = <T,>(x: T, isDemo: boolean) => (compare && !isDemo ? x : null);

  const socialKpis: PreviewKpi[] = [
    {
      label: t.social.followers,
      value: formatCompact(social.followers),
      delta: d(social.followersDelta, social.isDemo),
    },
    {
      label: t.social.engagementRate,
      value: formatPercent(social.engagementRate),
      delta: d(social.engagementRateDelta, social.isDemo),
    },
    { label: t.social.reach, value: formatCompact(social.reach), delta: d(social.reachDelta, social.isDemo) },
    {
      label: t.social.postsPublished,
      value: formatNumber(social.posts),
      delta: d(social.postsDelta, social.isDemo),
    },
  ];
  const seoKpis: PreviewKpi[] = [
    { label: t.seo.clicks, value: formatCompact(seo.clicks), delta: d(seo.clicksDelta, seo.isDemo) },
    {
      label: t.seo.impressions,
      value: formatCompact(seo.impressions),
      delta: d(seo.impressionsDelta, seo.isDemo),
    },
    { label: t.seo.ctr, value: formatPercent(seo.ctr), delta: d(seo.ctrDelta, seo.isDemo) },
    {
      label: t.seo.position,
      value: formatNumber(seo.position, 1),
      delta: d(seo.positionDelta, seo.isDemo),
      lowerIsBetter: true,
    },
  ];
  const adsKpis: PreviewKpi[] = [
    {
      label: t.ads.spend,
      value: formatCurrency(ads.kpis.spend, currency, { compact: true }),
      delta: d(ads.spendDelta, ads.isDemo),
    },
    { label: t.ads.results, value: formatNumber(ads.kpis.results), delta: d(ads.resultsDelta, ads.isDemo) },
    {
      label: t.ads.cpr,
      value: formatCurrency(ads.kpis.cpr, currency),
      delta: d(ads.cprDelta, ads.isDemo),
      lowerIsBetter: true,
    },
    { label: t.ads.ctr, value: formatPercent(ads.kpis.ctr) },
  ];

  return (
    <ReportBuilderForm
      slug={slug}
      defaultTitle={`Laporan ${monthTitle(range.to)} — ${clientName}`}
      from={from}
      to={to}
      rangeLabel={rangeLabel}
      compareDefault={compare}
      previews={{
        SOCIAL:
          social.hasData || extras.SOCIAL.length > 0 ? (
            <ModulePreviewCard
              title={t.overview.socialCard}
              icon={<Share2 className="size-4 text-muted-foreground" />}
              kpis={social.hasData ? socialKpis : []}
              sections={extras.SOCIAL}
            />
          ) : null,
        SEO:
          seo.hasData || extras.SEO.length > 0 ? (
            <ModulePreviewCard
              title={t.overview.seoCard}
              icon={<Search className="size-4 text-muted-foreground" />}
              kpis={seo.hasData ? seoKpis : []}
              sections={extras.SEO}
            />
          ) : null,
        ADS: ads.hasData ? (
          <ModulePreviewCard
            title={t.overview.adsCard}
            icon={<Megaphone className="size-4 text-muted-foreground" />}
            kpis={adsKpis}
          />
        ) : null,
      }}
      insightPreview={
        <AiInsightCard
          content={insight?.content}
          createdAt={insight?.createdAt}
          isMock={insight?.model === "mock"}
          action={
            <InsightGenerator
              slug={slug}
              module="OVERVIEW"
              from={from}
              to={to}
              hasExisting={Boolean(insight)}
            />
          }
        />
      }
      moduleStatus={{
        SOCIAL: { hasData: social.hasData || extras.SOCIAL.length > 0, isDemo: social.isDemo },
        SEO: { hasData: seo.hasData || extras.SEO.length > 0, isDemo: seo.isDemo },
        ADS: { hasData: ads.hasData, isDemo: ads.isDemo },
      }}
    />
  );
}

async function HistoryTab({
  slug,
  clientId,
  canManage,
}: {
  slug: string;
  clientId: string;
  canManage: boolean;
}) {
  const reports = await listReports(clientId);
  const rows: ReportRow[] = reports.map((r) => ({
    id: r.id,
    title: r.title,
    from: r.dateFrom.toISOString(),
    to: r.dateTo.toISOString(),
    modules: r.modules,
    language: r.language,
    hasAiSummary: r.hasAiSummary,
    createdByName: r.createdByName,
    createdAt: r.createdAt.toISOString(),
  }));
  return (
    <Section title={t.reports.history}>
      <ReportHistoryTable slug={slug} rows={rows} canManage={canManage} />
    </Section>
  );
}

const INSIGHT_MODULES: { key: InsightModule; label: string; desc: string }[] = [
  { key: "OVERVIEW", label: rs.insight.overviewCard, desc: rs.insight.overviewDesc },
  { key: "SOCIAL", label: rs.insight.socialCard, desc: rs.insight.socialDesc },
  { key: "SEO", label: rs.insight.seoCard, desc: rs.insight.seoDesc },
  { key: "ADS", label: rs.insight.adsCard, desc: rs.insight.adsDesc },
];

async function InsightTab({
  slug,
  clientId,
  range,
  from,
  to,
  rangeLabel,
}: {
  slug: string;
  clientId: string;
  range: { from: Date; to: Date };
  from: string;
  to: string;
  rangeLabel: string;
}) {
  const insights = await Promise.all(INSIGHT_MODULES.map((m) => getLatestInsight(clientId, m.key, range)));
  return (
    <Section
      title={t.reports.aiInsight}
      description={`${t.reports.aiInsightDesc} · ${rs.insight.forRange} ${rangeLabel}`}
    >
      <div className="grid gap-5 lg:grid-cols-2">
        {INSIGHT_MODULES.map((m, i) => {
          const insight = insights[i];
          return (
            <div key={m.key} className="space-y-2">
              <h3 className="flex items-baseline gap-2 text-sm font-semibold">
                {m.label}
                <span className="text-xs font-normal text-muted-foreground">{m.desc}</span>
              </h3>
              <AiInsightCard
                content={insight?.content}
                createdAt={insight?.createdAt}
                isMock={insight?.model === "mock"}
                action={
                  <InsightGenerator
                    slug={slug}
                    module={m.key}
                    from={from}
                    to={to}
                    hasExisting={Boolean(insight)}
                  />
                }
              />
            </div>
          );
        })}
      </div>
    </Section>
  );
}
