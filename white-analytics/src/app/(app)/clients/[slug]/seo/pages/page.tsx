import type { Metadata } from "next";
import Link from "next/link";
import { Globe } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { EmptyState } from "@/components/dashboard/empty-state";
import { requireClientAccess } from "@/lib/rbac";
import { getRange } from "@/lib/range-params";
import { formatDateRange } from "@/lib/format";
import { t } from "@/i18n/id";
import { s } from "@/features/seo/strings";
import { getPageStats, getSeoProperty } from "@/features/seo/queries";
import { filterDimRows, parseBucket } from "@/features/seo/aggregate";
import { FilterChips } from "@/features/seo/components/filter-chips";
import { BUCKET_LABELS } from "@/features/seo/components/buckets";
import { PERIOD_KEYS, pickQuery, withQuery } from "@/lib/drill-query";
import { ClicksPerPageCard, PagesTable } from "@/features/seo/components/pages-explorer";

export async function generateMetadata(props: PageProps<"/clients/[slug]/seo/pages">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.seo.pages}` };
}

export default async function SeoPagesPage(props: PageProps<"/clients/[slug]/seo/pages">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client } = await requireClientAccess(slug);
  const { range, previous, compare } = getRange(sp);
  const q = typeof sp.q === "string" ? sp.q.trim() || null : null;
  const bucket = parseBucket(typeof sp.bucket === "string" ? sp.bucket : undefined);
  const basePath = `/clients/${slug}/seo/pages`;
  const without = (k: string) => withQuery(basePath, pickQuery(sp, [...PERIOD_KEYS, "q", "bucket"]), { [k]: null });

  const property = await getSeoProperty(client.id);
  if (!property) {
    return (
      <>
        <PageHeader eyebrow={t.nav.seo} title={t.seo.pages} description={t.seo.pagesSubtitle} />
        <EmptyState
          icon={<Globe />}
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

  const all = await getPageStats(property.id, range, previous);
  const rows = filterDimRows(all, { q, bucket });
  const chips = [
    q ? { label: s.filterPage, value: q, removeHref: without("q") } : null,
    bucket ? { label: s.filterBucket, value: BUCKET_LABELS[bucket], removeHref: without("bucket") } : null,
  ].filter((c) => c != null);

  return (
    <>
      <PageHeader
        eyebrow={t.nav.seo}
        title={t.seo.pages}
        description={`${t.seo.pagesSubtitle} · ${formatDateRange(range.from, range.to)}`}
      />
      <FilterChips chips={chips} clearHref={withQuery(basePath, pickQuery(sp))} />
      <ClicksPerPageCard rows={rows} />
      <Section title={s.allPages}>
        <PagesTable rows={rows} compare={compare} />
      </Section>
    </>
  );
}
