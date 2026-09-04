"use client";

import type { ColumnDef } from "@tanstack/react-table";
import { ChartCard } from "@/components/dashboard/chart-card";
import { TimeSeriesChart } from "@/components/dashboard/charts/time-series-chart";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { DataTable, type ColMeta } from "@/components/dashboard/data-table";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { PlatformIcon } from "@/features/social/components/platform-icon";
import { formatCompact, formatCurrency, formatDate, formatNumber, formatPercent } from "@/lib/format";
import { t } from "@/i18n/id";
import { rs } from "@/features/reports/strings";
import type { ShareAdsView, ShareSeoView, ShareSocialView } from "@/features/share/queries";

/** Read-only module sections for the public share dashboard (same primitives as the app). */

type PostRow = ShareSocialView["topPosts"][number];

const postColumns: ColumnDef<PostRow, unknown>[] = [
  {
    accessorKey: "caption",
    header: "Caption",
    cell: ({ row }) => (
      <span className="flex min-w-0 items-center gap-2">
        <PlatformIcon platform={row.original.platform} className="size-3.5" />
        <span className="block max-w-[320px] truncate" title={row.original.caption}>
          {row.original.caption || "–"}
        </span>
      </span>
    ),
  },
  { accessorKey: "type", header: t.social.type },
  { accessorKey: "publishedAt", header: t.social.published, cell: ({ getValue }) => formatDate(String(getValue())) },
  { accessorKey: "likes", header: t.social.likes, cell: ({ getValue }) => formatNumber(Number(getValue())), meta: { align: "right" } satisfies ColMeta },
  { accessorKey: "comments", header: t.social.comments, cell: ({ getValue }) => formatNumber(Number(getValue())), meta: { align: "right" } satisfies ColMeta },
  { accessorKey: "engagements", header: t.social.engagements, cell: ({ getValue }) => formatNumber(Number(getValue())), meta: { align: "right" } satisfies ColMeta },
];

export function ShareSocialSection({ data }: { data: ShareSocialView }) {
  return (
    <div className="space-y-4">
      <KpiGrid cols={4}>
        <KpiTile label={t.social.followers} value={formatCompact(data.followers)} delta={data.followersDelta} />
        <KpiTile label={t.social.engagementRate} value={formatPercent(data.engagementRate)} delta={data.engagementRateDelta} />
        <KpiTile label={t.social.reach} value={formatCompact(data.reach)} delta={data.reachDelta} />
        <KpiTile label={t.social.postsPublished} value={formatNumber(data.posts)} delta={data.postsDelta} />
      </KpiGrid>
      <ChartCard title={t.social.growthChart} description={t.social.growthChartDesc}>
        <TimeSeriesChart
          data={data.daily}
          series={[{ key: "followers", label: t.social.followers, type: "area", format: (v) => formatNumber(v) }]}
          height={220}
          yDomain={["dataMin", "auto"]}
        />
      </ChartCard>
      <ChartCard title={rs.share.topPosts} description={t.social.topPostsDesc}>
        <DataTable columns={postColumns} data={data.topPosts} bare pageSize={10} dense getRowId={(r) => r.id} />
      </ChartCard>
    </div>
  );
}

type QueryRow = ShareSeoView["topQueries"][number];

const queryColumns: ColumnDef<QueryRow, unknown>[] = [
  { accessorKey: "query", header: t.seo.query },
  { accessorKey: "clicks", header: t.seo.clicks, cell: ({ getValue }) => formatNumber(Number(getValue())), meta: { align: "right" } satisfies ColMeta },
  { accessorKey: "impressions", header: t.seo.impressions, cell: ({ getValue }) => formatCompact(Number(getValue())), meta: { align: "right" } satisfies ColMeta },
  { accessorKey: "ctr", header: t.seo.ctr, cell: ({ getValue }) => formatPercent(Number(getValue())), meta: { align: "right" } satisfies ColMeta },
  { accessorKey: "position", header: t.seo.position, cell: ({ getValue }) => formatNumber(Number(getValue()), 1), meta: { align: "right" } satisfies ColMeta },
];

export function ShareSeoSection({ data }: { data: ShareSeoView }) {
  return (
    <div className="space-y-4">
      <KpiGrid cols={4}>
        <KpiTile label={t.seo.clicks} value={formatCompact(data.clicks)} delta={data.clicksDelta} />
        <KpiTile label={t.seo.impressions} value={formatCompact(data.impressions)} delta={data.impressionsDelta} />
        <KpiTile label={t.seo.ctr} value={formatPercent(data.ctr)} delta={data.ctrDelta} />
        <KpiTile label={t.seo.position} value={formatNumber(data.position, 1)} delta={data.positionDelta} lowerIsBetter />
      </KpiGrid>
      {/* Klik vs impresi differ ~30×: small multiples, one y-axis each (chart rule). */}
      <div className="grid gap-4 lg:grid-cols-2">
        <ChartCard title={t.seo.clicks} description={t.seo.trendChartDesc}>
          <TimeSeriesChart data={data.daily} series={[{ key: "clicks", label: t.seo.clicks, type: "area", format: (v) => formatNumber(v) }]} height={200} />
        </ChartCard>
        <ChartCard title={t.seo.impressions} description={t.seo.trendChartDesc}>
          <TimeSeriesChart
            data={data.daily}
            series={[{ key: "impressions", label: t.seo.impressions, type: "line", color: "var(--chart-2)", format: (v) => formatCompact(v) }]}
            height={200}
          />
        </ChartCard>
      </div>
      <ChartCard title={rs.share.topQueries}>
        <DataTable columns={queryColumns} data={data.topQueries} bare pageSize={10} dense getRowId={(r) => r.query} />
      </ChartCard>
    </div>
  );
}

export function ShareAdsSection({ data }: { data: ShareAdsView }) {
  const cur = data.currency;
  const campaignColumns: ColumnDef<ShareAdsView["campaigns"][number], unknown>[] = [
    { accessorKey: "name", header: t.ads.campaign },
    { accessorKey: "spend", header: t.ads.spend, cell: ({ getValue }) => formatCurrency(Number(getValue()), cur, { compact: true }), meta: { align: "right" } satisfies ColMeta },
    { accessorKey: "results", header: t.ads.results, cell: ({ getValue }) => formatNumber(Number(getValue())), meta: { align: "right" } satisfies ColMeta },
    {
      accessorKey: "cpr",
      header: t.ads.cpr,
      cell: ({ row }) => (row.original.isConversion && row.original.cpr > 0 ? formatCurrency(row.original.cpr, cur) : "–"),
      meta: { align: "right" } satisfies ColMeta,
    },
  ];
  return (
    <div className="space-y-4">
      <KpiGrid cols={4}>
        <KpiTile label={t.ads.spend} value={formatCurrency(data.spend, cur, { compact: true })} delta={data.spendDelta} />
        <KpiTile label={t.ads.results} value={formatNumber(data.results)} delta={data.resultsDelta} />
        <KpiTile label={t.ads.cpr} value={formatCurrency(data.cpr, cur)} delta={data.cprDelta} lowerIsBetter />
        <KpiTile label={t.ads.ctr} value={formatPercent(data.ctr)} />
      </KpiGrid>
      <ChartCard title={t.ads.spendChart} description={t.ads.spendResultsDesc}>
        <TimeSeriesChart
          data={data.daily}
          series={[{ key: "spend", label: t.ads.spend, type: "bar", color: "var(--chart-2)", format: (v) => formatCurrency(v, cur, { compact: true }) }]}
          height={220}
          yFormat={(v) => formatCompact(v)}
        />
      </ChartCard>
      <div className="grid gap-4 lg:grid-cols-2">
        <ChartCard title={rs.share.campaigns} description={t.ads.spend}>
          <HBarChart
            items={data.campaigns.slice(0, 6).map((c) => ({ label: c.name, value: c.spend }))}
            format={(v) => formatCurrency(v, cur, { compact: true })}
            showShare
          />
        </ChartCard>
        <ChartCard title={t.ads.breakdown}>
          <DataTable columns={campaignColumns} data={data.campaigns} bare pageSize={8} dense getRowId={(r) => r.id} />
        </ChartCard>
      </div>
    </div>
  );
}
