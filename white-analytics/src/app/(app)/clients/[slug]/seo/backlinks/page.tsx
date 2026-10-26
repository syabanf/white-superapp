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
import { BacklinkDrillSheet } from "@/features/seo-suite/components/backlink-drill";
import { FilterChips } from "@/features/seo/components/filter-chips";
import { PERIOD_KEYS, pickQuery, withQuery } from "@/lib/drill-query";
import type { BacklinkView } from "@/lib/metrics/seo-suite";

const BACKLINK_VIEWS: BacklinkView[] = ["all", "new", "lost", "toxic"];

export async function generateMetadata(props: PageProps<"/clients/[slug]/seo/backlinks">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.nav.backlinks}` };
}

export default async function SeoBacklinksPage(props: PageProps<"/clients/[slug]/seo/backlinks">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client } = await requireClientAccess(slug);
  const { range, compare } = getRange(sp);
  const one = (k: string) => (typeof sp[k] === "string" ? (sp[k] as string) : null);
  const view = BACKLINK_VIEWS.find((v) => v === one("view")) ?? "all";
  const domain = one("domain");
  const anchor = one("anchor");
  const basePath = `/clients/${slug}/seo/backlinks`;
  const keep = pickQuery(sp, [...PERIOD_KEYS, "view", "domain", "anchor"]);
  const list = (extra: Record<string, string | null>) => withQuery(basePath, keep, extra, "backlinks");

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
  // same keys as topCounts() so a bar lands on exactly its rows
  const label = (v: string) => v.trim() || "(kosong)";
  const rows = data.rows.filter((r) => (!domain || label(r.sourceDomain) === domain) && (!anchor || label(r.anchor) === anchor));
  const chips = [
    domain ? { label: s.domainLabel, value: domain, removeHref: list({ domain: null }) } : null,
    anchor ? { label: s.topAnchors, value: anchor, removeHref: list({ anchor: null }) } : null,
  ].filter((c) => c != null);
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
            <KpiTile label={s.backlinksKpi} value={formatCompact(data.latest.backlinks)} delta={d(data.deltas?.backlinks)} caption={s.vs30d} href={list({ view: null, domain: null, anchor: null })} />
            <KpiTile label={s.referringDomains} value={formatCompact(data.latest.referringDomains)} delta={d(data.deltas?.referringDomains)} caption={s.vs30d} href={list({ view: null, domain: null, anchor: null })} />
            <KpiTile label={s.domainRank} value={data.latest.domainRank == null ? "–" : formatNumber(data.latest.domainRank)} delta={d(data.deltas?.domainRank)} hint={s.domainRankHint} caption={s.vs30d} />
            <KpiTile label={s.dofollowShare} value={formatPercent(data.dofollowShare, 1)} delta={d(data.deltas?.dofollowShare)} hint={s.dofollowShareHint} caption={s.vs30d} href={list({ view: null, domain: null, anchor: null })} />
          </KpiGrid>

          <StatStrip
            items={[
              { label: s.dofollow, value: formatCompact(data.latest.dofollow) },
              { label: s.nofollow, value: formatCompact(data.latest.nofollow) },
              { label: s.newLast30, href: list({ view: "new" }), value: formatNumber(data.rows.filter((r) => !r.isLost && new Date(data.today).getTime() - new Date(r.firstSeen).getTime() <= 30 * 86_400_000).length) },
              { label: s.lostLast30, href: list({ view: "lost" }), value: formatNumber(data.rows.filter((r) => r.isLost && new Date(data.today).getTime() - new Date(r.lastSeen).getTime() <= 30 * 86_400_000).length) },
              { label: s.toxicShare, value: data.latest.toxicShare == null ? "–" : formatPercent(data.latest.toxicShare * 100, 1), hint: s.toxicShareHint, href: list({ view: "toxic" }) },
            ]}
          />

          <BacklinkDrillSheet rows={data.rows} listHref={list({ view: null, domain: null, anchor: null })} />
          <BacklinkTrendCards daily={data.daily} weekly={data.weekly} />
          <div id="backlinks" className="scroll-mt-24 space-y-3">
            <FilterChips chips={chips} clearHref={list({ view: null, domain: null, anchor: null })} />
            <BacklinkTable key={view} rows={rows} today={data.today} initialView={view} />
          </div>
          <BacklinkTopCards topAnchors={data.topAnchors} topDomains={data.topDomains} listHref={list({ view: null, domain: null, anchor: null })} />
        </>
      )}
    </>
  );
}
