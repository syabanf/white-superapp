import type { Metadata } from "next";
import Link from "next/link";
import { KeyRound } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { EmptyState } from "@/components/dashboard/empty-state";
import { requireClientAccess } from "@/lib/rbac";
import { getRange } from "@/lib/range-params";
import { formatDateRange } from "@/lib/format";
import { t } from "@/i18n/id";
import { s } from "@/features/seo/strings";
import { getDimensionDetail, getQueryStats, getSeoProperty } from "@/features/seo/queries";
import {
  buildKeywordRows,
  countryLabel,
  deviceLabel,
  filterDimRows,
  normalizeKeywordView,
  parseBucket,
  type KeywordViewKey,
} from "@/features/seo/aggregate";
import { KeywordsExplorer } from "@/features/seo/components/keywords-explorer";
import { DimensionFocus } from "@/features/seo/components/dimension-focus";
import { FilterChips } from "@/features/seo/components/filter-chips";
import { BUCKET_LABELS } from "@/features/seo/components/buckets";
import { PERIOD_KEYS, pickQuery, withQuery } from "@/lib/drill-query";
import { withClientMode } from "@/lib/client-mode";

export async function generateMetadata(props: PageProps<"/clients/[slug]/seo/keywords">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.seo.keywords}` };
}

async function SeoKeywordsPageInner(props: PageProps<"/clients/[slug]/seo/keywords">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client } = await requireClientAccess(slug);
  const { range, previous, compare } = getRange(sp);
  const one = (k: string) => (typeof sp[k] === "string" ? (sp[k] as string) : undefined);
  const view = normalizeKeywordView(one("view"));
  const q = one("q")?.trim() || null;
  const bucket = parseBucket(one("bucket"));
  const device = one("device")?.toUpperCase() || null;
  const country = one("country")?.toLowerCase() || null;
  const basePath = `/clients/${slug}/seo/keywords`;
  const filterKeys = ["q", "bucket", "device", "country"];
  const query = pickQuery(sp, [...PERIOD_KEYS, ...filterKeys]);
  const without = (k: string) => withQuery(basePath, pickQuery(sp, [...PERIOD_KEYS, "view", ...filterKeys]), { [k]: null });

  const property = await getSeoProperty(client.id);
  if (!property) {
    return (
      <>
        <PageHeader eyebrow={t.nav.seo} title={t.seo.keywords} description={t.seo.keywordsSubtitle} />
        <EmptyState
          icon={<KeyRound />}
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

  const focus = device ? ("DEVICE" as const) : country ? ("COUNTRY" as const) : null;
  const [stats, detail] = await Promise.all([
    getQueryStats(property.id, range, previous),
    focus ? getDimensionDetail(property.id, focus, focus === "DEVICE" ? device! : country!, range, previous) : null,
  ]);
  const rows = filterDimRows(buildKeywordRows(stats.rows, stats.opportunities, view), { q, bucket });
  const chips = [
    q ? { label: s.filterQuery, value: q, removeHref: without("q") } : null,
    bucket ? { label: s.filterBucket, value: BUCKET_LABELS[bucket], removeHref: without("bucket") } : null,
    device ? { label: s.filterDevice, value: deviceLabel(device), removeHref: without("device") } : null,
    country ? { label: s.filterCountry, value: countryLabel(country), removeHref: without("country") } : null,
  ].filter((c) => c != null);
  const counts: Record<KeywordViewKey, number> = {
    all: stats.rows.length,
    striking: stats.opportunities.striking.length,
    low_ctr: stats.opportunities.low_ctr.length,
    declining: stats.opportunities.declining.length,
    rising: stats.opportunities.rising.length,
  };

  return (
    <>
      <PageHeader
        eyebrow={t.nav.seo}
        title={t.seo.keywords}
        description={`${t.seo.keywordsSubtitle} · ${formatDateRange(range.from, range.to)}`}
      />
      <FilterChips chips={chips} clearHref={withQuery(basePath, pickQuery(sp, [...PERIOD_KEYS, "view"]))} />
      {detail ? (
        <DimensionFocus
          detail={detail}
          label={`${detail.dimension === "DEVICE" ? s.filterDevice : s.filterCountry}: ${detail.dimension === "DEVICE" ? deviceLabel(detail.key) : countryLabel(detail.key)}`}
          compare={compare}
        />
      ) : null}
      <KeywordsExplorer
        view={view}
        rows={rows}
        counts={counts}
        distribution={stats.distribution.map((d) => ({ bucket: d.bucket, queries: d.queries }))}
        compare={compare}
        basePath={basePath}
        query={query}
      />
    </>
  );
}

/** Klien demo dirender dalam mode demo: adapter mock + DemoBanner walau kredensial asli terpasang. */
export default async function SeoKeywordsPage(props: Parameters<typeof SeoKeywordsPageInner>[0]) {
  const { slug } = await props.params;
  return withClientMode({ slug }, () => SeoKeywordsPageInner(props));
}
