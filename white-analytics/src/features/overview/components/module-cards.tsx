"use client";

import type { ReactNode } from "react";
import Link from "next/link";
import { ArrowUpRight, Clock3, Database, Megaphone, Search, Share2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { TimeSeriesChart } from "@/components/dashboard/charts/time-series-chart";
import { DeltaBadge } from "@/components/dashboard/delta-badge";
import { EmptyState } from "@/components/dashboard/empty-state";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { cn } from "@/lib/utils";
import {
  formatCompact,
  formatCurrency,
  formatDateRange,
  formatNumber,
  formatPercent,
  formatRelative,
} from "@/lib/format";
import { t } from "@/i18n/id";
import type { Delta } from "@/lib/metrics";
import type { DataQuality, OverviewAds, OverviewSeo, OverviewSocial } from "@/features/overview/queries";
import { PlatformIcon } from "@/features/social/components/platform-icon";

/**
 * One shape for all three module panels: header · flex-1 body · two-stat footer.
 * Equal heights and aligned baselines across the row — no ragged bottoms.
 */
function ModulePanel({
  icon,
  title,
  meta,
  href,
  children,
  footer,
  index,
  quality,
}: {
  icon: ReactNode;
  title: string;
  meta: string;
  href: string;
  children: ReactNode;
  footer?: ReactNode;
  index: string;
  quality: DataQuality;
}) {
  return (
    <Card className="lift flex min-h-[330px] flex-col gap-0 overflow-hidden py-0 hover:ring-brand/25">
      <div className="flex items-start justify-between gap-3 px-6 pt-5 pb-3">
        <div className="min-w-0">
          <span className="label-mono text-brand">{index}</span>
          <h3 className="mt-1 flex items-center gap-2 text-[15px] font-semibold tracking-[-0.02em]">
            {icon}
            {title}
          </h3>
          <p className="mt-1 truncate text-xs text-muted-foreground">{meta}</p>
        </div>
        <Button asChild variant="ghost" size="xs" className="group/nudge shrink-0">
          <Link href={href}>
            {t.common.open} <ArrowUpRight className="nudge size-3.5" />
          </Link>
        </Button>
      </div>

      <DataQualityLine quality={quality} />

      <div className="flex flex-1 flex-col justify-center px-6 pb-6">{children}</div>

      {footer ? <div className="grid grid-cols-2 divide-x border-t">{footer}</div> : null}
    </Card>
  );
}

const QUALITY_COPY: Record<DataQuality["state"], { label: string; kind: "good" | "warning" | "neutral" }> = {
  healthy: { label: "Koneksi sehat", kind: "good" },
  stale: { label: "Data terlambat", kind: "warning" },
  demo: { label: "Data demo", kind: "neutral" },
  empty: { label: "Belum ada data", kind: "warning" },
  not_connected: { label: "Belum terhubung", kind: "neutral" },
};

function DataQualityLine({ quality }: { quality: DataQuality }) {
  const copy = QUALITY_COPY[quality.state];
  return (
    <div className="mx-6 mb-4 flex flex-wrap items-center gap-x-3 gap-y-1 border-b pb-3 text-xs text-muted-foreground">
      <StatusBadge kind={copy.kind}>{copy.label}</StatusBadge>
      <span className="inline-flex items-center gap-1">
        <Clock3 className="size-3.5" />
        {quality.lastSyncedAt
          ? `Sinkron ${formatRelative(quality.lastSyncedAt)}`
          : "Belum pernah disinkronkan"}
      </span>
      {quality.dataFrom && quality.dataTo ? (
        <span className="inline-flex items-center gap-1">
          <Database className="size-3.5" />
          Data {formatDateRange(new Date(quality.dataFrom), new Date(quality.dataTo))}
        </span>
      ) : null}
    </div>
  );
}

function FooterStat({
  label,
  value,
  delta,
  lowerIsBetter,
}: {
  label: string;
  value: string;
  delta?: Delta | null;
  lowerIsBetter?: boolean;
}) {
  return (
    <div className="min-w-0 px-6 py-3.5">
      <div className="label-mono truncate text-muted-foreground">{label}</div>
      <div className="mt-1.5 flex flex-wrap items-center gap-2">
        <span className="text-[17px] font-bold tracking-[-0.02em] tabular">{value}</span>
        {delta ? <DeltaBadge delta={delta} lowerIsBetter={lowerIsBetter} /> : null}
      </div>
    </div>
  );
}

function ConnectEmpty({ title, description, href }: { title: string; description: string; href: string }) {
  return (
    <EmptyState
      compact
      title={title}
      description={description}
      action={
        <Button asChild size="sm">
          <Link href={href}>{t.common.connect}</Link>
        </Button>
      }
    />
  );
}

function NoDataEmpty({ href }: { href: string }) {
  return (
    <EmptyState
      compact
      title="Belum ada data pada periode ini"
      description="Sumber data sudah tersedia, tetapi periode yang dipilih belum memiliki metrik. Coba periode lain atau sinkronkan ulang."
      action={
        <Button asChild size="sm" variant="outline">
          <Link href={href}>Buka modul</Link>
        </Button>
      }
    />
  );
}

export function SocialModuleCard({
  data,
  href,
  settingsHref,
}: {
  data: OverviewSocial;
  href: string;
  settingsHref: string;
}) {
  const max = Math.max(1, ...data.accounts.map((a) => a.followers));
  return (
    <ModulePanel
      index="01 · Social"
      icon={<Share2 className="size-4 text-muted-foreground" />}
      title={t.overview.socialCard}
      meta={`${formatNumber(data.followers)} followers · ${data.posts} ${t.social.posts.toLowerCase()}`}
      href={href}
      quality={data.quality}
      footer={
        data.hasData ? (
          <>
            <FooterStat
              label={t.social.engagementRate}
              value={formatPercent(data.engagementRate)}
              delta={data.isDemo ? null : data.engagementRateDelta}
            />
            <FooterStat
              label={t.social.reach}
              value={formatCompact(data.reach)}
              delta={data.isDemo ? null : data.reachDelta}
            />
          </>
        ) : undefined
      }
    >
      {!data.hasSource ? (
        <ConnectEmpty title={t.social.noAccount} description={t.social.noAccountDesc} href={settingsHref} />
      ) : !data.hasData ? (
        <NoDataEmpty href={href} />
      ) : (
        <ul className="space-y-3.5">
          {data.accounts.map((a) => (
            <li key={`${a.platform}-${a.username}`}>
              <div className="mb-1.5 flex items-center justify-between gap-2 text-sm">
                <span className="flex min-w-0 items-center gap-2">
                  <PlatformIcon platform={a.platform} className="size-4 shrink-0" />
                  <span className="truncate text-muted-foreground">@{a.username}</span>
                </span>
                <span className="shrink-0 font-semibold tabular">{formatCompact(a.followers)}</span>
              </div>
              <div className="h-1.5 w-full rounded-full bg-muted">
                <div
                  className="h-1.5 rounded-full"
                  style={{ width: `${(a.followers / max) * 100}%`, background: "var(--chart-1)" }}
                />
              </div>
            </li>
          ))}
        </ul>
      )}
    </ModulePanel>
  );
}

export function SeoModuleCard({
  data,
  href,
  settingsHref,
}: {
  data: OverviewSeo;
  href: string;
  settingsHref: string;
}) {
  return (
    <ModulePanel
      index="02 · SEO"
      icon={<Search className="size-4 text-muted-foreground" />}
      title={t.overview.seoCard}
      meta={`${formatNumber(data.clicks)} ${t.seo.clicks.toLowerCase()} · ${formatCompact(data.impressions)} ${t.seo.impressions.toLowerCase()}`}
      href={href}
      quality={data.quality}
      footer={
        data.hasData ? (
          <>
            <FooterStat
              label={t.seo.ctr}
              value={formatPercent(data.ctr)}
              delta={data.isDemo ? null : data.ctrDelta}
            />
            <FooterStat
              label={t.seo.position}
              value={formatNumber(data.position, 1)}
              delta={data.isDemo ? null : data.positionDelta}
              lowerIsBetter
            />
          </>
        ) : undefined
      }
    >
      {!data.hasSource ? (
        <ConnectEmpty title={t.seo.noProperty} description={t.seo.noPropertyDesc} href={settingsHref} />
      ) : !data.hasData ? (
        <NoDataEmpty href={href} />
      ) : (
        <TimeSeriesChart
          data={data.daily}
          series={[{ key: "clicks", label: t.seo.clicks, type: "area", format: (v) => formatNumber(v) }]}
          height={168}
        />
      )}
    </ModulePanel>
  );
}

export function AdsModuleCard({
  data,
  href,
  settingsHref,
  currency,
}: {
  data: OverviewAds;
  href: string;
  settingsHref: string;
  currency: string;
}) {
  const ctrDelta: Delta | null =
    data.prev.ctr === 0
      ? null
      : {
          abs: data.kpis.ctr - data.prev.ctr,
          pct: ((data.kpis.ctr - data.prev.ctr) / data.prev.ctr) * 100,
          direction: data.kpis.ctr >= data.prev.ctr ? "up" : "down",
        };
  return (
    <ModulePanel
      index="03 · Ads"
      icon={<Megaphone className="size-4 text-muted-foreground" />}
      title={t.overview.adsCard}
      meta={`${formatCurrency(data.kpis.spend, currency, { compact: true })} · ${formatNumber(data.kpis.results)} ${t.ads.results.toLowerCase()}`}
      href={href}
      quality={data.quality}
      footer={
        data.hasData ? (
          <>
            <FooterStat
              label={t.ads.cpr}
              value={formatCurrency(data.kpis.cpr, currency)}
              delta={data.isDemo ? null : data.cprDelta}
              lowerIsBetter
            />
            <FooterStat
              label={t.ads.ctr}
              value={formatPercent(data.kpis.ctr)}
              delta={data.isDemo ? null : ctrDelta}
            />
          </>
        ) : undefined
      }
    >
      {!data.hasSource ? (
        <ConnectEmpty title={t.ads.noAccount} description={t.ads.noAccountDesc} href={settingsHref} />
      ) : !data.hasData ? (
        <NoDataEmpty href={href} />
      ) : (
        <TimeSeriesChart
          data={data.daily}
          series={[
            {
              key: "spend",
              label: t.ads.spend,
              type: "bar",
              color: "var(--chart-2)",
              format: (v) => formatCurrency(v, currency, { compact: true }),
            },
          ]}
          height={168}
          yFormat={(v) => formatCompact(v)}
        />
      )}
    </ModulePanel>
  );
}

export { ModulePanel as OverviewModulePanel };
export const moduleCardClass = cn();
