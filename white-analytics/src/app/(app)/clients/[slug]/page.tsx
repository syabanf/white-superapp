import type { Metadata } from "next";
import Link from "next/link";
import { FileText, Settings, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { KpiGrid, KpiTile } from "@/components/dashboard/kpi-tile";
import { AdsModuleCard, SeoModuleCard, SocialModuleCard } from "@/features/overview/components/module-cards";
import { SyncActivity } from "@/features/overview/components/sync-activity";
import { AiInsightCard } from "@/features/reports/components/ai-insight-card";
import { getOverviewAds, getOverviewSeo, getOverviewSocial, getRecentSyncJobs } from "@/features/overview/queries";
import { getLatestInsight } from "@/features/reports/queries";
import { requireClientAccess } from "@/lib/rbac";
import { getRange } from "@/lib/range-params";
import { formatCompact, formatCurrency, formatDateRange } from "@/lib/format";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.nav.overview}` };
}

export default async function ClientOverviewPage(props: PageProps<"/clients/[slug]">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client } = await requireClientAccess(slug);
  const { range, previous, compare } = getRange(sp);
  const base = `/clients/${slug}`;

  const [social, seo, ads, jobs, insight] = await Promise.all([
    getOverviewSocial(client.id, range, previous),
    getOverviewSeo(client.id, range, previous),
    getOverviewAds(client.id, range, previous),
    getRecentSyncJobs(client.id),
    getLatestInsight(client.id, "OVERVIEW", range),
  ]);

  const d = (x: { pct: number | null; abs: number; direction: "up" | "down" | "flat" }) => (compare ? x : null);

  return (
    <>
      <PageHeader
        eyebrow={t.nav.overview}
        title={client.name}
        description={`${t.overview.subtitle} · ${formatDateRange(range.from, range.to)}`}
        actions={
          <>
            <Button asChild variant="outline" size="sm">
              <Link href={`${base}/settings`}>
                <Settings className="size-4" /> {t.common.settings}
              </Link>
            </Button>
            <Button asChild size="sm">
              <Link href={`${base}/reports`}>
                <FileText className="size-4" /> {t.reports.builder}
              </Link>
            </Button>
          </>
        }
      />

      <KpiGrid>
        <KpiTile
          label={t.social.followers}
          value={formatCompact(social.followers)}
          delta={d(social.followersDelta)}
          spark={social.followersSpark}
          hint="Total followers semua akun sendiri (IG, FB, TikTok). Delta = pertumbuhan periode ini vs periode sebelumnya."
        />
        <KpiTile
          label={`${t.seo.clicks} organik`}
          value={formatCompact(seo.clicks)}
          delta={d(seo.clicksDelta)}
          spark={seo.clicksSpark}
          hint="Klik organik dari Google Search Console."
        />
        <KpiTile
          label={t.ads.spend}
          value={formatCurrency(ads.kpis.spend, client.currency, { compact: true })}
          delta={d(ads.spendDelta)}
          spark={ads.spendSpark}
        />
        <KpiTile
          label={t.ads.cpr}
          value={formatCurrency(ads.kpis.cpr, client.currency)}
          delta={d(ads.cprDelta)}
          lowerIsBetter
          hint="Belanja iklan / hasil konversi. Lebih rendah lebih baik."
        />
      </KpiGrid>

      <div className="grid gap-4 lg:grid-cols-3">
        <SocialModuleCard data={social} href={`${base}/social`} settingsHref={`${base}/settings`} />
        <SeoModuleCard data={seo} href={`${base}/seo`} settingsHref={`${base}/settings`} />
        <AdsModuleCard data={ads} href={`${base}/ads`} settingsHref={`${base}/settings`} currency={client.currency} />
      </div>

      <div className="grid gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2 [&>div]:h-full">
          <AiInsightCard
            content={insight?.content}
            createdAt={insight?.createdAt}
            isMock={insight?.model === "mock"}
            action={
              <Button asChild variant="outline" size="xs">
                <Link href={`${base}/reports?tab=insight`}>
                  <Sparkles className="size-3.5" /> {insight ? t.reports.aiRegenerate : t.reports.aiGenerate}
                </Link>
              </Button>
            }
          />
        </div>
        <div className="[&>div]:h-full">
          <SyncActivity jobs={jobs} />
        </div>
      </div>
    </>
  );
}
