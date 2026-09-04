import type { Metadata } from "next";
import Link from "next/link";
import { TrendingUp } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { DemoBanner } from "@/components/dashboard/banners";
import { seoDataMode } from "@/lib/providers/dataforseo";
import { EmptyState } from "@/components/dashboard/empty-state";
import { requireClientAccess } from "@/lib/rbac";
import { getRange } from "@/lib/range-params";
import { formatDate, formatDateRange, formatNumber, formatPercent } from "@/lib/format";
import { t } from "@/i18n/id";
import { s } from "@/features/seo-suite/strings";
import { getRankData, getSuiteProperty, isSuiteDemo, lastJobAt } from "@/features/seo-suite/queries";
import { RankBreakdownCards, RankTrendCards } from "@/features/seo-suite/components/rank-charts";
import { RankTable } from "@/features/seo-suite/components/rank-table";
import { AddKeywordsDialog } from "@/features/seo-suite/components/add-keywords-dialog";
import { RefreshButton } from "@/features/seo-suite/components/refresh-button";

export async function generateMetadata(props: PageProps<"/clients/[slug]/seo/rank">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.nav.rank}` };
}

export default async function SeoRankPage(props: PageProps<"/clients/[slug]/seo/rank">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, canManage } = await requireClientAccess(slug);
  const { range, previous, compare } = getRange(sp);

  const property = await getSuiteProperty(client.id);
  if (!property) {
    return (
      <>
        <PageHeader eyebrow={t.nav.seo} title={t.nav.rank} description={s.rankSubtitle} />
        <EmptyState
          icon={<TrendingUp />}
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

  const [demo, data, lastRun] = await Promise.all([isSuiteDemo(), getRankData(property.id, range, previous), lastJobAt(client.id, ["SEO_SUITE_RANKS", "SEO_SUITE_DAILY"])]);
  const d = <T,>(x: T) => (compare ? x : null);

  return (
    <>
      <PageHeader
        eyebrow={t.nav.seo}
        title={t.nav.rank}
        description={`${property.host} · ${formatDateRange(range.from, range.to)}${data.latestDate ? ` · ${s.lastUpdated} ${formatDate(data.latestDate)}` : ""}`}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            {canManage ? <AddKeywordsDialog clientId={client.id} propertyId={property.id} variant="outline" /> : null}
            {data.hasKeywords ? <RefreshButton kind="ranks" clientId={client.id} propertyId={property.id} lastRunAt={lastRun} /> : null}
          </div>
        }
      />
      {demo ? <DemoBanner message={seoDataMode() === "apify" ? s.apifyBanner : s.demoBanner} /> : null}

      {!data.hasKeywords ? (
        <EmptyState icon={<TrendingUp />} title={s.noKeywordsTitle} description={s.noKeywordsDesc} action={canManage ? <AddKeywordsDialog clientId={client.id} propertyId={property.id} /> : undefined} />
      ) : (
        <>
          <KpiGrid>
            <KpiTile label={s.visibility} value={formatPercent(data.kpis.visibility, 1)} delta={d(data.deltas.visibility)} hint={s.visibilityHint} />
            <KpiTile label={t.seo.position} value={data.kpis.avgPosition == null ? "–" : formatNumber(data.kpis.avgPosition, 1)} delta={d(data.deltas.avgPosition)} lowerIsBetter hint={s.avgPositionHint} />
            <KpiTile label={s.top3} value={formatNumber(data.kpis.top3)} delta={d(data.deltas.top3)} hint={s.top3Hint} />
            <KpiTile label={s.top10} value={formatNumber(data.kpis.top10)} delta={d(data.deltas.top10)} hint={s.top10Hint} />
          </KpiGrid>

          <StatStrip
            items={[
              { label: s.trackedKeywords, value: formatNumber(data.strip.tracked) },
              { label: s.ranked, value: formatNumber(data.strip.ranked) },
              { label: s.unranked, value: formatNumber(data.strip.unranked) },
              { label: s.improved, value: formatNumber(data.strip.improved), hint: s.deltaColHint },
              { label: s.declinedKw, value: formatNumber(data.strip.declined), hint: s.deltaColHint },
            ]}
          />

          <RankTrendCards daily={data.daily} compare={compare} />
          <RankBreakdownCards distribution={data.distribution} sov={data.sov} />
          <RankTable rows={data.rows} tags={data.tags} clientId={client.id} canManage={canManage} />
        </>
      )}
    </>
  );
}
