import type { Metadata } from "next";
import Link from "next/link";
import { Users } from "lucide-react";
import { Button } from "@/components/ui/button";
import { DemoBanner } from "@/components/dashboard/banners";
import { EmptyState } from "@/components/dashboard/empty-state";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { PlatformTabs } from "@/features/social/components/platform-tabs";
import { PostsTable } from "@/features/social/components/posts-table";
import { SyncButton } from "@/features/social/components/sync-button";
import { TopPostsGrid } from "@/features/social/components/post-cards";
import {
  ErByTypeCard,
  GrowthChartCard,
  PostingHeatmapCard,
  ReachChartCard,
} from "@/features/social/components/social-charts";
import { parsePlatform } from "@/features/social/lib";
import { getSocialDashboard } from "@/features/social/queries";
import { s } from "@/features/social/strings";
import { requireClientAccess } from "@/lib/rbac";
import { getRange } from "@/lib/range-params";
import {
  formatCompact,
  formatDateRange,
  formatDeltaNumber,
  formatDeltaPercent,
  formatNumber,
  formatPercent,
} from "@/lib/format";
import type { Delta } from "@/lib/metrics";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/social">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.nav.social}` };
}

export default async function SocialPage(props: PageProps<"/clients/[slug]/social">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client } = await requireClientAccess(slug);
  const { range, previous, compare } = getRange(sp);
  const platform = parsePlatform(sp.platform);
  const base = `/clients/${slug}`;

  const data = await getSocialDashboard(client.id, platform, range, previous);
  const d = (x: Delta) => (compare && !data.isDemo ? x : null);
  const kpis = data.kpis;

  return (
    <>
      <PageHeader
        eyebrow={t.nav.social}
        title={client.name}
        description={`${t.social.subtitle} · ${formatDateRange(range.from, range.to)}`}
        actions={
          <>
            <Button asChild variant="outline" size="sm">
              <Link href={`${base}/social/competitors`}>
                <Users className="size-4" /> {t.social.competitors}
              </Link>
            </Button>
            <SyncButton clientId={client.id} platform={platform} />
          </>
        }
      />

      <PlatformTabs current={platform} />

      {data.isDemo ? (
        <DemoBanner
          message={
            platform === "TIKTOK" ? `${t.social.demoBanner} ${t.social.tiktokMock}` : t.social.demoBanner
          }
          settingsHref={`${base}/settings`}
        />
      ) : null}

      {!data.account || !kpis ? (
        <EmptyState
          title={t.social.noAccount}
          description={t.social.noAccountDesc}
          action={
            <Button asChild size="sm">
              <Link href={`${base}/settings`}>{t.social.addAccount}</Link>
            </Button>
          }
        />
      ) : (
        <>
          <KpiGrid>
            <KpiTile
              label={t.social.followers}
              value={formatCompact(kpis.followers)}
              delta={d(kpis.followersDelta)}
              spark={kpis.followersSpark}
              hint={s.followersHint}
            />
            <KpiTile
              label={t.social.followerGrowth}
              value={formatDeltaNumber(kpis.growthAbs)}
              caption={kpis.growthPct == null ? undefined : formatDeltaPercent(kpis.growthPct)}
              hint={s.growthHint}
            />
            <KpiTile
              label={t.social.engagementRate}
              value={formatPercent(kpis.er)}
              delta={d(kpis.erDelta)}
              hint={s.erHint}
            />
            <KpiTile
              label={t.social.reach}
              value={formatCompact(kpis.reach)}
              delta={d(kpis.reachDelta)}
              hint={s.reachHint}
            />
          </KpiGrid>

          <StatStrip
            items={[
              {
                label: t.social.impressions,
                value: formatCompact(kpis.impressions),
                hint: s.impressionsHint,
              },
              {
                label: t.social.postsPublished,
                value: formatNumber(kpis.posts),
                caption: `${formatNumber(kpis.postsPerWeek, 1)}${t.common.perWeek}`,
                hint: s.postsHint,
              },
            ]}
          />

          <div className="grid gap-5 lg:grid-cols-2">
            <GrowthChartCard data={data.growthSeries} />
            <ReachChartCard data={data.reachSeries} />
          </div>

          <div className="grid gap-5 lg:grid-cols-2">
            <ErByTypeCard items={data.erByType} />
            <PostingHeatmapCard cells={data.heatmap} />
          </div>

          <Section title={t.social.topPosts} description={t.social.topPostsDesc}>
            <TopPostsGrid posts={data.topPosts} />
          </Section>

          <Section title={t.social.recentPosts}>
            <PostsTable rows={data.posts} exportName={`postingan-${slug}-${platform.toLowerCase()}`} />
          </Section>
        </>
      )}
    </>
  );
}
