import type { Metadata } from "next";
import Link from "next/link";
import { Megaphone } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { StatStrip } from "@/components/dashboard/stat-strip";
import { DemoBanner } from "@/components/dashboard/banners";
import { EmptyState } from "@/components/dashboard/empty-state";
import { CampaignFilter } from "@/features/ads/components/campaign-filter";
import { CsvImportDialog } from "@/features/ads/components/csv-import-dialog";
import { SyncButton } from "@/features/ads/components/sync-button";
import { CampaignShareCard, ResultsDailyCard, SpendDailyCard } from "@/features/ads/components/ads-charts";
import { DemographicsCard } from "@/features/ads/components/demographics-card";
import { CampaignBreakdown } from "@/features/ads/components/campaign-breakdown";
import { TopAds } from "@/features/ads/components/top-ads";
import { getAdsDashboard } from "@/features/ads/queries";
import { resultTypeLabel, s } from "@/features/ads/strings";
import { requireClientAccess } from "@/lib/rbac";
import { getRange } from "@/lib/range-params";
import {
  formatCompact,
  formatCurrency,
  formatDateRange,
  formatNumber,
  formatPercent,
  formatRelative,
} from "@/lib/format";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/ads">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.ads.title}` };
}

export default async function AdsPage(props: PageProps<"/clients/[slug]/ads">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, canManage } = await requireClientAccess(slug);
  const { range, previous, compare } = getRange(sp);
  const campaignParam = Array.isArray(sp.campaign) ? sp.campaign[0] : sp.campaign;
  const base = `/clients/${slug}`;

  const data = await getAdsDashboard(client.id, range, previous, campaignParam);
  const { kpis, deltas } = data;

  const d = (x: { pct: number | null; abs: number; direction: "up" | "down" | "flat" }) =>
    compare && !data.isDemo ? x : null;

  return (
    <>
      <PageHeader
        eyebrow={t.nav.ads}
        title={t.ads.title}
        description={`${t.ads.subtitle} · ${formatDateRange(range.from, range.to)}`}
        actions={
          canManage ? (
            <>
              <CsvImportDialog clientId={client.id} />
              <SyncButton clientId={client.id} />
            </>
          ) : undefined
        }
      />

      {data.isDemo && data.account ? (
        <DemoBanner message={t.ads.demoBanner} settingsHref={`${base}/settings`} />
      ) : null}

      {!data.account ? (
        <EmptyState
          icon={<Megaphone className="size-5" />}
          title={t.ads.noAccount}
          description={t.ads.noAccountDesc}
          action={
            <Button asChild size="sm">
              <Link href={`${base}/settings`}>{t.common.connect}</Link>
            </Button>
          }
        />
      ) : (
        <>
          {/* Baris filter (satu baris, di bawah header, di atas KPI) */}
          <div className="flex flex-wrap items-center justify-between gap-2">
            <CampaignFilter options={data.campaignOptions} />
            <p className="text-xs text-muted-foreground">
              {data.account.name}
              {data.account.lastSyncedAt
                ? ` · ${t.common.lastSynced} ${formatRelative(data.account.lastSyncedAt)}`
                : ` · ${t.common.neverSynced}`}
            </p>
          </div>

          <KpiGrid>
            <KpiTile
              label={t.ads.spend}
              value={formatCurrency(kpis.spend, client.currency, { compact: true })}
              delta={d(deltas.spend)}
              spark={data.spendSpark}
            />
            <KpiTile
              label={t.ads.results}
              value={formatNumber(kpis.results)}
              delta={d(deltas.results)}
              spark={data.resultsSpark}
              caption={
                kpis.results > 0 && data.dominantResultType
                  ? resultTypeLabel(data.dominantResultType)
                  : s.noConversionResults
              }
              hint={s.resultsHint}
            />
            <KpiTile
              label={t.ads.cpr}
              value={kpis.results > 0 ? formatCurrency(kpis.cpr, client.currency) : "–"}
              delta={kpis.results > 0 ? d(deltas.cpr) : null}
              lowerIsBetter
              hint={s.cprHint}
            />
            <KpiTile
              label={t.ads.ctr}
              value={formatPercent(kpis.ctr)}
              delta={d(deltas.ctr)}
              hint={s.ctrHint}
            />
          </KpiGrid>

          <StatStrip
            items={[
              {
                label: t.ads.impressions,
                value: formatCompact(kpis.impressions),
                delta: d(deltas.impressions),
              },
              {
                label: t.ads.reach,
                value: formatCompact(kpis.reach),
                delta: d(deltas.reach),
                hint: s.reachHint,
              },
              { label: t.ads.linkClicks, value: formatCompact(kpis.linkClicks), delta: d(deltas.linkClicks) },
              {
                label: t.ads.cpc,
                value: kpis.linkClicks > 0 ? formatCurrency(kpis.cpc, client.currency) : "–",
                delta: kpis.linkClicks > 0 ? d(deltas.cpc) : null,
                lowerIsBetter: true,
                hint: s.cpcHint,
              },
              {
                label: t.ads.cpm,
                value: kpis.impressions > 0 ? formatCurrency(kpis.cpm, client.currency) : "–",
                delta: kpis.impressions > 0 ? d(deltas.cpm) : null,
                lowerIsBetter: true,
                hint: s.cpmHint,
              },
              {
                label: t.ads.frequency,
                value: formatNumber(kpis.frequency, 1),
                delta: d(deltas.frequency),
                lowerIsBetter: true,
                hint: s.frequencyHint,
              },
              ...(kpis.roas != null
                ? [
                    {
                      label: t.ads.roas,
                      value: `${formatNumber(kpis.roas, 2)}×`,
                      delta: deltas.roas ? d(deltas.roas) : null,
                      hint: s.roasHint,
                    },
                  ]
                : []),
            ]}
          />

          {/* Dua small multiples terpisah — bukan dual axis */}
          <div className="grid gap-5 lg:grid-cols-2">
            <SpendDailyCard daily={data.daily} currency={client.currency} compare={compare} />
            <ResultsDailyCard daily={data.daily} />
          </div>

          <div className="grid gap-5 lg:grid-cols-2">
            <CampaignShareCard items={data.campaignShare} currency={client.currency} />
            <DemographicsCard
              byAge={data.demographics.byAge}
              byGender={data.demographics.byGender}
              matrix={data.demographics.matrix}
              currency={client.currency}
              hasData={data.demographics.hasData}
            />
          </div>

          <Section title={t.ads.breakdown} description={t.ads.breakdownDesc}>
            <CampaignBreakdown campaigns={data.breakdown} currency={client.currency} />
          </Section>

          <Section title={t.ads.topAds} description={t.ads.topAdsDesc}>
            <TopAds ads={data.topAds} currency={client.currency} />
          </Section>
        </>
      )}
    </>
  );
}
