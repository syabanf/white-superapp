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
import { RankDrillSheet } from "@/features/seo-suite/components/rank-drill";
import { BUCKET_LABELS, filterRankRows, parseRankBucket, parseRankChange } from "@/features/seo-suite/lib";
import { FilterChips } from "@/features/seo/components/filter-chips";
import { PERIOD_KEYS, pickQuery, withQuery } from "@/lib/drill-query";

export async function generateMetadata(props: PageProps<"/clients/[slug]/seo/rank">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.nav.rank}` };
}

export default async function SeoRankPage(props: PageProps<"/clients/[slug]/seo/rank">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, canManage } = await requireClientAccess(slug);
  const { range, previous, compare } = getRange(sp);
  const bucket = parseRankBucket(sp.bucket);
  const change = parseRankChange(sp.change);
  const basePath = `/clients/${slug}/seo/rank`;
  const keep = pickQuery(sp, [...PERIOD_KEYS, "bucket", "change"]);
  const table = (extra: Record<string, string | null>) => withQuery(basePath, keep, extra, "keywords");

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
  const rows = filterRankRows(data.rows, { bucket, change });
  const chips = [
    bucket ? { label: t.seo.position, value: BUCKET_LABELS[bucket], removeHref: table({ bucket: null }) } : null,
    change ? { label: s.deltaCol, value: change === "improved" ? s.improved : s.declinedKw, removeHref: table({ change: null }) } : null,
  ].filter((c) => c != null);

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
            <KpiTile label={s.visibility} value={formatPercent(data.kpis.visibility, 1)} delta={d(data.deltas.visibility)} hint={s.visibilityHint} href={table({ bucket: null, change: null })} />
            <KpiTile label={t.seo.position} value={data.kpis.avgPosition == null ? "–" : formatNumber(data.kpis.avgPosition, 1)} delta={d(data.deltas.avgPosition)} lowerIsBetter hint={s.avgPositionHint} href={table({ bucket: null, change: null })} />
            <KpiTile label={s.top3} value={formatNumber(data.kpis.top3)} delta={d(data.deltas.top3)} hint={s.top3Hint} href={table({ bucket: "1-3", change: null })} />
            <KpiTile label={s.top10} value={formatNumber(data.kpis.top10)} delta={d(data.deltas.top10)} hint={s.top10Hint} href={table({ bucket: "4-10", change: null })} />
          </KpiGrid>

          <StatStrip
            items={[
              { label: s.trackedKeywords, value: formatNumber(data.strip.tracked), href: table({ bucket: null, change: null }) },
              { label: s.ranked, value: formatNumber(data.strip.ranked), href: table({ bucket: null, change: null }) },
              { label: s.unranked, value: formatNumber(data.strip.unranked), href: table({ bucket: "unranked", change: null }) },
              { label: s.improved, value: formatNumber(data.strip.improved), hint: s.deltaColHint, href: table({ bucket: null, change: "improved" }) },
              { label: s.declinedKw, value: formatNumber(data.strip.declined), hint: s.deltaColHint, href: table({ bucket: null, change: "declined" }) },
            ]}
          />

          <RankDrillSheet slug={slug} listHref={table({ bucket: null, change: null })} />
          <RankTrendCards daily={data.daily} compare={compare} />
          <RankBreakdownCards distribution={data.distribution} sov={data.sov} bucketHref={table({ bucket: null, change: null })} />
          <div id="keywords" className="scroll-mt-24 space-y-3">
            <FilterChips chips={chips} clearHref={table({ bucket: null, change: null })} />
            <RankTable rows={rows} tags={data.tags} clientId={client.id} canManage={canManage} />
          </div>
        </>
      )}
    </>
  );
}
