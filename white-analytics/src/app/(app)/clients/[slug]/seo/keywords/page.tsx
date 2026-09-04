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
import { getQueryStats, getSeoProperty } from "@/features/seo/queries";
import { buildKeywordRows, normalizeKeywordView, type KeywordViewKey } from "@/features/seo/aggregate";
import { KeywordsExplorer } from "@/features/seo/components/keywords-explorer";

export async function generateMetadata(props: PageProps<"/clients/[slug]/seo/keywords">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.seo.keywords}` };
}

export default async function SeoKeywordsPage(props: PageProps<"/clients/[slug]/seo/keywords">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client } = await requireClientAccess(slug);
  const { range, previous, compare } = getRange(sp);
  const view = normalizeKeywordView(Array.isArray(sp.view) ? sp.view[0] : sp.view);

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

  const stats = await getQueryStats(property.id, range, previous);
  const rows = buildKeywordRows(stats.rows, stats.opportunities, view);
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
      <KeywordsExplorer
        view={view}
        rows={rows}
        counts={counts}
        distribution={stats.distribution.map((d) => ({ bucket: d.bucket, queries: d.queries }))}
        compare={compare}
        basePath={`/clients/${slug}/seo/keywords`}
      />
    </>
  );
}
