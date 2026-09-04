import type { Metadata } from "next";
import Link from "next/link";
import { Users } from "lucide-react";
import { Button } from "@/components/ui/button";
import { DemoBanner } from "@/components/dashboard/banners";
import { EmptyState } from "@/components/dashboard/empty-state";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { AddCompetitorDialog } from "@/features/social/components/add-competitor-dialog";
import { CompetitorGrowthCard } from "@/features/social/components/competitor-growth-chart";
import { CompetitorTable } from "@/features/social/components/competitor-table";
import { TopPostsGrid } from "@/features/social/components/post-cards";
import { getCompetitorDashboard } from "@/features/social/queries";
import { s } from "@/features/social/strings";
import { requireClientAccess } from "@/lib/rbac";
import { getRange } from "@/lib/range-params";
import { formatDateRange } from "@/lib/format";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/social/competitors">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.nav.competitors}` };
}

export default async function CompetitorsPage(props: PageProps<"/clients/[slug]/social/competitors">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, canManage } = await requireClientAccess(slug);
  const { range, previous, compare } = getRange(sp);
  const base = `/clients/${slug}`;

  const data = await getCompetitorDashboard(client.id, range, previous);
  const competitorCount = data.rows.filter((r) => !r.isOwn).length;
  const isEmpty = data.rows.length === 0 || (!data.hasOwn && competitorCount === 0);

  return (
    <>
      <PageHeader
        eyebrow={`${t.nav.social} · ${client.name}`}
        title={t.social.competitors}
        description={`${t.social.competitorsSubtitle} · ${formatDateRange(range.from, range.to)}`}
        actions={canManage ? <AddCompetitorDialog clientId={client.id} /> : undefined}
      />

      <DemoBanner message={t.social.demoBanner} settingsHref={`${base}/settings`} />

      {isEmpty ? (
        <EmptyState
          icon={<Users className="size-5" />}
          title={s.noCompetitors}
          description={s.noCompetitorsDesc}
          action={
            canManage ? (
              <AddCompetitorDialog clientId={client.id} />
            ) : (
              <Button asChild size="sm" variant="outline">
                <Link href={`${base}/settings`}>{t.common.settings}</Link>
              </Button>
            )
          }
        />
      ) : (
        <>
          {!data.hasOwn ? <p className="text-sm text-muted-foreground">{s.ownAccountMissing}</p> : null}

          <Section title={t.social.compareTable}>
            <CompetitorTable
              rows={data.rows}
              canManage={canManage}
              compare={compare}
              exportName={`kompetitor-${slug}`}
            />
          </Section>

          <CompetitorGrowthCard chart={data.chart} />

          <Section title={s.topCompetitorPosts} description={s.topCompetitorPostsDesc}>
            <TopPostsGrid posts={data.topPosts} />
          </Section>
        </>
      )}
    </>
  );
}
