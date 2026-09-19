import type { Metadata } from "next";
import Link from "next/link";
import { Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { DemoBanner } from "@/components/dashboard/banners";
import { SeoSyncButton } from "@/features/seo/components/seo-sync-button";
import { EmptyState } from "@/components/dashboard/empty-state";
import { requireClientAccess } from "@/lib/rbac";
import { getRange } from "@/lib/range-params";
import { formatCompact, formatDateRange, formatNumber, formatPercent } from "@/lib/format";
import { t } from "@/i18n/id";
import { s } from "@/features/seo/strings";
import {
  getDimBreakdown,
  getPageStats,
  getQueryStats,
  getSeoOverview,
  getSeoProperty,
  hasGoogleConnection,
} from "@/features/seo/queries";
import { SearchTrendCards, PositionTrendCard, Ga4SessionsCard } from "@/features/seo/components/overview-charts";
import { PositionDistributionCard, DeviceCountryCards } from "@/features/seo/components/breakdown-cards";
import { TopQueriesCard, TopPagesCard } from "@/features/seo/components/top-tables";

export async function generateMetadata(props: PageProps<"/clients/[slug]/seo">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.seo.title}` };
}

export default async function SeoOverviewPage(props: PageProps<"/clients/[slug]/seo">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client } = await requireClientAccess(slug);
  const { range, previous, compare } = getRange(sp);
  const base = `/clients/${slug}/seo`;
  const settingsHref = `/clients/${slug}/settings`;

  const property = await getSeoProperty(client.id);
  if (!property) {
    return (
      <>
        <PageHeader eyebrow={t.nav.seo} title={t.nav.seoOverview} description={t.seo.subtitle} />
        <EmptyState
          icon={<Search />}
          title={t.seo.noProperty}
          description={t.seo.noPropertyDesc}
          action={
            <Button asChild size="sm">
              <Link href={settingsHref}>{t.seo.addProperty}</Link>
            </Button>
          }
        />
      </>
    );
  }

  const [connected, overview, queryStats, pageStats, breakdown] = await Promise.all([
    hasGoogleConnection(client.id),
    getSeoOverview(property.id, range, previous),
    getQueryStats(property.id, range, previous),
    getPageStats(property.id, range, previous),
    getDimBreakdown(property.id, range),
  ]);

  const d = (x: { pct: number | null; abs: number; direction: "up" | "down" | "flat" }) => (compare ? x : null);

  return (
    <>
      <PageHeader
        eyebrow={t.nav.seo}
        title={t.nav.seoOverview}
        description={`${property.siteUrl} · ${formatDateRange(range.from, range.to)}`}
        actions={<SeoSyncButton clientId={client.id} />}
      />

      {!connected ? <DemoBanner message={t.seo.demoBanner} settingsHref={settingsHref} /> : null}

      <KpiGrid>
        <KpiTile label={t.seo.clicks} value={formatCompact(overview.kpis.clicks)} delta={d(overview.deltas.clicks)} spark={overview.sparks.clicks} hint={s.clicksHint} />
        <KpiTile label={t.seo.impressions} value={formatCompact(overview.kpis.impressions)} delta={d(overview.deltas.impressions)} spark={overview.sparks.impressions} hint={s.impressionsHint} />
        <KpiTile label={t.seo.ctr} value={formatPercent(overview.kpis.ctr)} delta={d(overview.deltas.ctr)} />
        <KpiTile label={t.seo.position} value={formatNumber(overview.kpis.position, 1)} delta={d(overview.deltas.position)} lowerIsBetter hint={s.positionHint} />
      </KpiGrid>

      <StatStrip
        items={[
          {
            label: t.seo.organicSessions,
            value: formatCompact(overview.ga4.organicSessions),
            delta: d(overview.ga4.organicDelta),
            hint: s.organicHint,
          },
          {
            label: t.seo.conversions,
            value: formatCompact(overview.ga4.conversions),
            delta: d(overview.ga4.conversionsDelta),
            hint: s.conversionsHint,
          },
          {
            label: t.seo.healthScore,
            value: overview.health == null ? "–" : `${formatNumber(overview.health)} / 100`,
            hint: s.healthHint,
          },
        ]}
      />

      <SearchTrendCards daily={overview.daily} compare={compare} />

      <div className="grid gap-5 lg:grid-cols-2">
        <PositionTrendCard daily={overview.daily} compare={compare} />
        <PositionDistributionCard distribution={queryStats.distribution} />
      </div>

      <div className="grid gap-5 lg:grid-cols-2">
        <DeviceCountryCards devices={breakdown.devices} countries={breakdown.countries} />
      </div>

      <Ga4SessionsCard daily={overview.ga4Daily} />

      <div className="grid gap-5 lg:grid-cols-2">
        <TopQueriesCard rows={queryStats.rows} href={`${base}/keywords`} compare={compare} />
        <TopPagesCard rows={pageStats} href={`${base}/pages`} compare={compare} />
      </div>
    </>
  );
}
