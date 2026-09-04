import type { Metadata } from "next";
import Link from "next/link";
import { Link2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { DemoBanner } from "@/components/dashboard/banners";
import { EmptyState } from "@/components/dashboard/empty-state";
import { requireClientAccess } from "@/lib/rbac";
import { getRange } from "@/lib/range-params";
import { formatCompact, formatDate, formatDateRange, formatNumber, formatPercent } from "@/lib/format";
import { t } from "@/i18n/id";
import { s } from "@/features/seo-suite/strings";
import { getBacklinkData, getSuiteProperty, isSuiteDemo, lastJobAt } from "@/features/seo-suite/queries";
import { BacklinkTopCards, BacklinkTrendCards } from "@/features/seo-suite/components/backlink-charts";
import { BacklinkTable } from "@/features/seo-suite/components/backlink-table";
import { RefreshButton } from "@/features/seo-suite/components/refresh-button";

export async function generateMetadata(props: PageProps<"/clients/[slug]/seo/backlinks">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.nav.backlinks}` };
}

export default async function SeoBacklinksPage(props: PageProps<"/clients/[slug]/seo/backlinks">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client } = await requireClientAccess(slug);
  const { range, compare } = getRange(sp);

  const property = await getSuiteProperty(client.id);
  if (!property) {
    return (
      <>
        <PageHeader eyebrow={t.nav.seo} title={t.nav.backlinks} description={s.backlinksSubtitle} />
        <EmptyState
          icon={<Link2 />}
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

  const [demo, data, lastRun] = await Promise.all([isSuiteDemo(), getBacklinkData(property.id, range), lastJobAt(client.id, ["SEO_SUITE_BACKLINKS", "SEO_SUITE_DAILY"])]);
  const d = <T,>(x: T | undefined) => (compare && x ? x : null);
  const refresh = <RefreshButton kind="backlinks" clientId={client.id} propertyId={property.id} lastRunAt={lastRun} />;

  return (
    <>
      <PageHeader
        eyebrow={t.nav.seo}
        title={t.nav.backlinks}
        description={`${property.host} · ${formatDateRange(range.from, range.to)}${data.latest ? ` · ${s.lastUpdated} ${formatDate(data.latest.date)}` : ""}`}
        actions={refresh}
      />
      {demo ? <DemoBanner message={s.demoBanner} /> : null}

      {!data.latest ? (
        <EmptyState icon={<Link2 />} title={s.noBacklinksTitle} description={s.noBacklinksDesc} action={refresh} />
      ) : (
        <>
          <KpiGrid>
            <KpiTile label={s.backlinksKpi} value={formatCompact(data.latest.backlinks)} delta={d(data.deltas?.backlinks)} caption={s.vs30d} />
            <KpiTile label={s.referringDomains} value={formatCompact(data.latest.referringDomains)} delta={d(data.deltas?.referringDomains)} caption={s.vs30d} />
            <KpiTile label={s.domainRank} value={data.latest.domainRank == null ? "–" : formatNumber(data.latest.domainRank)} delta={d(data.deltas?.domainRank)} hint={s.domainRankHint} caption={s.vs30d} />
            <KpiTile label={s.dofollowShare} value={formatPercent(data.dofollowShare, 1)} delta={d(data.deltas?.dofollowShare)} hint={s.dofollowShareHint} caption={s.vs30d} />
          </KpiGrid>

          <StatStrip
            items={[
              { label: s.dofollow, value: formatCompact(data.latest.dofollow) },
              { label: s.nofollow, value: formatCompact(data.latest.nofollow) },
              { label: s.newLast30, value: formatNumber(data.rows.filter((r) => !r.isLost && new Date(data.today).getTime() - new Date(r.firstSeen).getTime() <= 30 * 86_400_000).length) },
              { label: s.lostLast30, value: formatNumber(data.rows.filter((r) => r.isLost && new Date(data.today).getTime() - new Date(r.lastSeen).getTime() <= 30 * 86_400_000).length) },
              { label: s.toxicShare, value: data.latest.toxicShare == null ? "–" : formatPercent(data.latest.toxicShare * 100, 1), hint: s.toxicShareHint },
            ]}
          />

          <BacklinkTrendCards daily={data.daily} weekly={data.weekly} />
          <BacklinkTable rows={data.rows} today={data.today} />
          <BacklinkTopCards topAnchors={data.topAnchors} topDomains={data.topDomains} />
        </>
      )}
    </>
  );
}
