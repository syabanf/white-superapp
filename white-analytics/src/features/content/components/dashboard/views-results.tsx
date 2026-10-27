"use client";

import * as React from "react";
import { ExternalLink, ImageOff } from "lucide-react";
import { ChartCard } from "@/components/dashboard/chart-card";
import { HBarChart } from "@/components/dashboard/charts/hbar-chart";
import { slotColor } from "@/components/dashboard/charts/chart-primitives";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { Section } from "@/components/dashboard/page-header";
import { PlatformIcon, platformLabel } from "@/features/social/components/platform-icon";
import { dayLabel } from "@/features/publishing/time";
import { engagementRate, perPost, topWithOthers, type DashBreakdown, type DashPost } from "@/features/content/dashboard";
import { contentFiles, FORMATS, num } from "@/features/content/lib";
import { c, cd } from "@/features/content/strings";
import type { AssetKind, ContentFormat } from "@/features/content/types";
import { formatCompact, formatDateTime, formatDecimal, formatNumber, formatPercent } from "@/lib/format";
import { useRouter } from "next/navigation";
import { CategoryBars, PeriodChart, PostScatter } from "./charts";
import { useContentDrill } from "./drill";
import { contentBoardHref, periodRange } from "./links";
import { bucketRange } from "@/features/erp/shared/drill/time";
import { DetailTable, TextLink, bucketLabel, overviewHref, personName, spark, useDelta, type DashCtx, type TabProps } from "./shared";

const pct = (v: number) => formatPercent(v, 1);

function BreakdownCard({
  title,
  rows,
  label,
  param,
  ctx,
  topN = 6,
}: {
  title: string;
  rows: DashBreakdown[];
  label: (k: string) => string;
  param: string;
  ctx: DashCtx;
  topN?: number;
}) {
  const measured = rows.filter((r) => r.measured > 0);
  const { top, others, othersCount } = topWithOthers(measured, topN, (r) => perPost(r.engagements, r.measured));
  const items = top.map((r) => ({
    label: label(r.key),
    value: Math.round(perPost(r.engagements, r.measured)),
    secondary: `${cd.erShort} ${pct(engagementRate(r.engagements, r.reach))}`,
    href: overviewHref(ctx, { [param]: r.key }, "performa"),
  }));
  if (othersCount > 0) items.push({ label: cd.others(othersCount), value: Math.round(others / othersCount), secondary: "", href: "" });
  return (
    <ChartCard
      title={title}
      description={cd.breakdownDesc}
      table={
        <DetailTable
          rows={rows}
          rowKey={(r) => r.key}
          minWidth={420}
          initialSort={{ key: "eng", desc: true }}
          columns={[
            { key: "key", label: cd.segment, value: (r) => label(r.key) },
            { key: "published", label: cd.published, align: "right", value: (r) => r.published },
            { key: "eng", label: cd.engagement, align: "right", value: (r) => num(r.engagements), render: (r) => formatNumber(num(r.engagements)) },
            { key: "avg", label: cd.perPostShort, align: "right", value: (r) => perPost(r.engagements, r.measured), render: (r) => formatNumber(Math.round(perPost(r.engagements, r.measured))) },
            { key: "er", label: cd.erShort, align: "right", value: (r) => engagementRate(r.engagements, r.reach), render: (r) => pct(engagementRate(r.engagements, r.reach)) },
          ]}
          showTotals={false}
        />
      }
    >
      <HBarChart
        items={items.map((it) => ({ ...it, href: it.href || undefined, muted: !it.href }))}
        format={(v) => formatNumber(v)}
        emptyLabel={cd.noEngagement}
      />
      <p className="mt-3 text-xs text-muted-foreground">{cd.perPostUnit}</p>
    </ChartCard>
  );
}

function PostsTable({ posts, ctx }: { posts: DashPost[]; ctx: DashCtx }) {
  return (
    <DetailTable
      rows={posts}
      rowKey={(p) => p.itemId}
      initialSort={{ key: "eng", desc: true }}
      showTotals={false}
      minWidth={720}
      columns={[
        {
          key: "title",
          label: cd.post,
          value: (p) => p.title,
          render: (p) => (
            <div className="flex min-w-0 items-center gap-3">
              {p.thumbnailFileId ? (
                // eslint-disable-next-line @next/next/no-img-element -- authenticated proxy file, not optimizable
                <img src={contentFiles.designFile(ctx.clientId, p.thumbnailFileId)} alt="" loading="lazy" className="size-10 shrink-0 rounded-lg bg-muted object-cover" />
              ) : (
                <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                  <ImageOff className="size-4" />
                </span>
              )}
              <span className="min-w-0">
                <TextLink href={`${ctx.base}/board/${p.itemId}`} className="block max-w-[34ch] truncate">
                  {p.title}
                </TextLink>
                <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <PlatformIcon platform={p.platform} className="size-3" /> {c.format[p.format]} · {p.publishedAt ? formatDateTime(p.publishedAt) : dayLabel(p.plannedDate)}
                  {p.permalink ? (
                    <a href={p.permalink} target="_blank" rel="noreferrer" aria-label={c.permalink} className="text-brand">
                      <ExternalLink className="size-3" />
                    </a>
                  ) : null}
                </span>
              </span>
            </div>
          ),
        },
        { key: "eng", label: cd.engagement, align: "right", value: (p) => num(p.engagements), render: (p) => (p.hasMetrics ? formatNumber(num(p.engagements)) : "–") },
        { key: "reach", label: cd.reach, align: "right", value: (p) => num(p.reach), render: (p) => (p.hasMetrics ? formatCompact(num(p.reach)) : "–") },
        { key: "er", label: cd.erShort, align: "right", value: (p) => engagementRate(p.engagements, p.reach), render: (p) => (p.hasMetrics ? pct(engagementRate(p.engagements, p.reach)) : "–") },
        { key: "saves", label: c.saves, align: "right", value: (p) => num(p.saves), render: (p) => formatNumber(num(p.saves)) },
        { key: "shares", label: c.shares, align: "right", value: (p) => num(p.shares), render: (p) => formatNumber(num(p.shares)) },
      ]}
    />
  );
}

export function PerformanceTab({ data, ctx }: TabProps) {
  const cur = data.current;
  const prev = data.previous;
  const k = cur.kpis;
  const p = prev.kpis;
  const d = useDelta(ctx);
  const gran = cur.granularity;
  const buckets = cur.buckets ?? [];
  const prevBuckets = prev.buckets ?? [];
  const posts = cur.posts ?? [];
  const measured = posts.filter((x) => x.hasMetrics);
  const formats = FORMATS.filter((f) => measured.some((x) => x.format === f));
  const perPostNow = perPost(k.engagements, k.measuredPosts);
  const perPostPrev = perPost(p.engagements, p.measuredPosts);
  const bottom = measured.toSorted((a, b) => num(a.engagements) - num(b.engagements)).slice(0, 5);
  const drill = useContentDrill(ctx);
  const publishedBoard = contentBoardHref(ctx, { ...periodRange(ctx), status: ["PUBLISHED"] });
  const openBucket = (row: Record<string, unknown>) => {
    const { from, to } = bucketRange(String(row.start), gran);
    drill({ dateFrom: from, dateTo: to, dateBy: "published", status: ["PUBLISHED"] });
  };

  return (
    <div className="page-stack">
      <KpiGrid>
        <KpiTile label={cd.engagement} value={formatCompact(num(k.engagements))} delta={d(k.engagements, p.engagements)} previous={formatCompact(num(p.engagements))} spark={spark(buckets, (b) => b.engagements)} href={publishedBoard} hint={cd.hEngagement} />
        <KpiTile label={cd.reach} value={formatCompact(num(k.reach))} delta={d(k.reach, p.reach)} previous={formatCompact(num(p.reach))} spark={spark(buckets, (b) => b.reach)} href={publishedBoard} hint={cd.hReach} />
        <KpiTile label={cd.er} value={pct(k.engagementRate)} delta={d(k.engagementRate, p.engagementRate)} previous={pct(p.engagementRate)} spark={spark(buckets, (b) => engagementRate(b.engagements, b.reach))} href={publishedBoard} hint={cd.hEr} />
        <KpiTile label={cd.perPost} value={formatNumber(Math.round(perPostNow))} delta={d(perPostNow, perPostPrev)} caption={cd.measuredPosts(k.measuredPosts)} href={publishedBoard} hint={cd.hPerPost} />
      </KpiGrid>

      <div className="grid gap-(--gap) [&>*]:min-w-0 lg:grid-cols-2">
        <ChartCard title={cd.engagementTrend} description={cd.engagementTrendDesc}>
          <PeriodChart
            onSelect={openBucket}
            data={buckets.map((b, i) => ({ start: b.start, label: bucketLabel(b.start, gran), engagements: num(b.engagements), previous: ctx.compare ? num(prevBuckets[i]?.engagements) : null }))}
            series={[{ key: "engagements", label: cd.engagement, type: "bar", format: (v) => formatNumber(v) }]}
            previousKey={ctx.compare ? "previous" : undefined}
            summary={cd.engagementTrend}
          />
        </ChartCard>
        <ChartCard title={cd.erTrend} description={cd.erTrendDesc}>
          <PeriodChart
            onSelect={openBucket}
            data={buckets.map((b, i) => ({
              start: b.start,
              label: bucketLabel(b.start, gran),
              er: Number(engagementRate(b.engagements, b.reach).toFixed(2)),
              previous: ctx.compare && prevBuckets[i] ? Number(engagementRate(prevBuckets[i]!.engagements, prevBuckets[i]!.reach).toFixed(2)) : null,
            }))}
            series={[{ key: "er", label: cd.er, type: "line", format: (v) => pct(v) }]}
            previousKey={ctx.compare ? "previous" : undefined}
            yFormat={(v) => `${formatDecimal(v)}%`}
            summary={cd.erTrend}
          />
        </ChartCard>
      </div>

      <ChartCard title={cd.scatterTitle} description={cd.scatterDesc}>
        <PostScatter
          points={measured.map((x) => ({ x: num(x.reach), y: num(x.engagements), title: x.title, group: x.format, href: `${ctx.base}/board/${x.itemId}` }))}
          groups={formats}
          groupLabel={(g) => c.format[g as ContentFormat] ?? g}
          summary={cd.scatterTitle}
        />
      </ChartCard>

      <div className="grid gap-(--gap) [&>*]:min-w-0 lg:grid-cols-3">
        <BreakdownCard title={cd.byPlatform} rows={cur.byPlatform ?? []} label={platformLabel} param="platform" ctx={ctx} />
        <BreakdownCard title={cd.byFormat} rows={cur.byFormat ?? []} label={(k) => c.format[k as ContentFormat] ?? k} param="format" ctx={ctx} />
        <BreakdownCard title={cd.byPillar} rows={cur.byPillar ?? []} label={(k) => k} param="pillar" ctx={ctx} topN={5} />
      </div>

      <div className="grid gap-(--gap) [&>*]:min-w-0">
        <Section title={cd.topPosts} description={cd.topPostsDesc}>
          <div className="rounded-[1.5rem] bg-card p-(--card-pad) shadow-(--card-shadow)">
            <PostsTable posts={measured.slice(0, 5)} ctx={ctx} />
          </div>
        </Section>
        <Section title={cd.bottomPosts} description={cd.bottomPostsDesc}>
          <div className="rounded-[1.5rem] bg-card p-(--card-pad) shadow-(--card-shadow)">
            <PostsTable posts={bottom} ctx={ctx} />
          </div>
        </Section>
      </div>
      <Section title={cd.allPosts(posts.length)}>
        <div className="max-h-[560px] overflow-y-auto rounded-[1.5rem] bg-card p-(--card-pad) shadow-(--card-shadow) scrollbar-thin">
          <PostsTable posts={posts} ctx={ctx} />
        </div>
      </Section>
    </div>
  );
}

export function AssetsTab({ data, ctx }: TabProps) {
  const cur = data.current;
  const prev = data.previous;
  const k = cur.kpis;
  const p = prev.kpis;
  const d = useDelta(ctx);
  const gran = cur.granularity;
  const buckets = cur.buckets ?? [];
  const designers = (cur.people ?? []).filter((x) => x.role === "DESIGNER" && x.designTasks > 0);
  const kinds = cur.assetUploadsByKind ?? [];
  const router = useRouter();
  const designHref = `${ctx.base}/design`;
  const assetsHref = `${ctx.base}/assets`;

  return (
    <div className="page-stack">
      <KpiGrid>
        <KpiTile label={cd.designTasks} value={formatNumber(k.designTasks)} delta={d(k.designTasks, p.designTasks)} caption={cd.approvedOf(k.designApproved)} href={designHref} hint={cd.hDesignTasks} />
        <KpiTile
          label={cd.turnaround}
          value={cd.days(formatDecimal(k.designTurnaroundDays))}
          delta={d(k.designTurnaroundDays, p.designTurnaroundDays)}
          lowerIsBetter
          previous={cd.days(formatDecimal(p.designTurnaroundDays))}
          spark={spark(buckets, (b) => b.designTurnaroundDays)}
          href={designHref}
          hint={cd.hTurnaround}
        />
        <KpiTile label={cd.firstPass} value={pct(k.firstPassRate)} delta={d(k.firstPassRate, p.firstPassRate)} previous={pct(p.firstPassRate)} href={designHref} hint={cd.hFirstPass} />
        <KpiTile label={cd.assetUploads} value={formatNumber(k.assetUploads)} delta={d(k.assetUploads, p.assetUploads)} spark={spark(buckets, (b) => b.assetUploads)} href={assetsHref} hint={cd.hAssetUploads} />
      </KpiGrid>

      <div className="grid gap-(--gap) [&>*]:min-w-0 lg:grid-cols-5">
        <ChartCard className="lg:col-span-3" title={cd.turnaroundTrend} description={cd.turnaroundTrendDesc}>
          <PeriodChart
            onSelect={() => router.push(designHref)}
            data={buckets.map((b) => ({ label: bucketLabel(b.start, gran), turnaround: b.designTurnaroundDays || null }))}
            series={[{ key: "turnaround", label: cd.turnaround, type: "line", format: (v) => cd.days(formatDecimal(v)) }]}
            yFormat={(v) => formatDecimal(v)}
            summary={cd.turnaroundTrend}
          />
        </ChartCard>
        <ChartCard className="lg:col-span-2" title={cd.versionsTitle} description={cd.versionsDesc}>
          <CategoryBars
            data={(cur.designVersions ?? []).map((r) => ({ label: cd.versionsLabel(r.key), value: r.count, href: designHref }))}
            label={cd.designTasks}
            color={slotColor(2)}
            summary={cd.versionsTitle}
          />
        </ChartCard>
      </div>

      <div className="grid gap-(--gap) [&>*]:min-w-0 lg:grid-cols-5">
        <ChartCard className="lg:col-span-3" title={cd.uploadsTrend} description={cd.uploadsTrendDesc}>
          <PeriodChart
            onSelect={() => router.push(assetsHref)}
            data={buckets.map((b) => ({ label: bucketLabel(b.start, gran), uploads: b.assetUploads }))}
            series={[{ key: "uploads", label: cd.assetUploads, type: "bar", color: slotColor(3), format: (v) => formatNumber(v) }]}
            yFormat={(v) => formatNumber(v)}
            summary={cd.uploadsTrend}
          />
        </ChartCard>
        <ChartCard className="lg:col-span-2" title={cd.uploadsByKind} description={cd.uploadsByKindDesc}>
          <HBarChart
            items={kinds.map((r) => ({ label: c.assetKind[r.key as AssetKind] ?? r.key, value: r.count, href: `${ctx.base}/assets?kind=${r.key}` }))}
            format={(v) => formatNumber(v)}
            showShare
            color={slotColor(3)}
            emptyLabel={cd.noData}
          />
        </ChartCard>
      </div>

      <div className="grid gap-(--gap) [&>*]:min-w-0 xl:grid-cols-2">
        <Section title={cd.designerTable} description={cd.designerTableDesc}>
          <div className="rounded-[1.5rem] bg-card p-(--card-pad) shadow-(--card-shadow)">
            <DetailTable
              rows={designers}
              rowKey={(x) => x.userId}
              initialSort={{ key: "tasks", desc: true }}
              minWidth={480}
              columns={[
                { key: "name", label: cd.person, value: (x) => personName(ctx, x.userId), render: (x) => <TextLink href={`${ctx.base}/design?designer=${x.userId}`}>{personName(ctx, x.userId)}</TextLink> },
                { key: "tasks", label: cd.designTasks, align: "right", value: (x) => x.designTasks, total: formatNumber(k.designTasks) },
                { key: "approved", label: cd.approved, align: "right", value: (x) => x.designApproved, total: formatNumber(k.designApproved) },
                { key: "turn", label: cd.turnaround, align: "right", value: (x) => x.turnaroundDays, render: (x) => cd.days(formatDecimal(x.turnaroundDays)), total: cd.days(formatDecimal(k.designTurnaroundDays)) },
                { key: "first", label: cd.firstPass, align: "right", value: (x) => x.firstPassRate, render: (x) => pct(x.firstPassRate), total: pct(k.firstPassRate) },
              ]}
            />
          </div>
        </Section>
        <Section title={cd.assetTable} description={cd.assetTableDesc}>
          <div className="rounded-[1.5rem] bg-card p-(--card-pad) shadow-(--card-shadow)">
            <DetailTable
              rows={cur.topAssets ?? []}
              rowKey={(a) => a.assetId}
              initialSort={{ key: "uploads", desc: true }}
              minWidth={480}
              columns={[
                { key: "name", label: cd.asset, value: (a) => a.name, render: (a) => <TextLink href={`${ctx.base}/assets/${a.assetId}`}>{a.name}</TextLink> },
                { key: "kind", label: cd.kind, value: (a) => c.assetKind[a.kind] ?? a.kind },
                { key: "uploads", label: cd.uploads, align: "right", value: (a) => a.uploads, total: formatNumber(k.assetUploads) },
                { key: "version", label: cd.version, align: "right", value: (a) => a.currentVersion, render: (a) => `v${a.currentVersion}` },
                { key: "last", label: cd.lastUpload, align: "right", value: (a) => a.lastUploadedAt, render: (a) => dayLabel(a.lastUploadedAt.slice(0, 10), true) },
              ]}
            />
          </div>
        </Section>
      </div>
    </div>
  );
}
