import type { Metadata } from "next";
import Link from "next/link";
import { FileText, Settings, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { HeroStage } from "@/components/dashboard/hero-stage";
import { AdsModuleCard, SeoModuleCard, SocialModuleCard } from "@/features/overview/components/module-cards";
import { SyncActivity } from "@/features/overview/components/sync-activity";
import { AiInsightCard } from "@/features/reports/components/ai-insight-card";
import {
  getOverviewAds,
  getOverviewSeo,
  getOverviewSocial,
  getRecentSyncJobs,
} from "@/features/overview/queries";
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
  const globalQuery = pickGlobalQuery(sp);
  const detailHref = (path: string) => `${base}/${path}${globalQuery ? `?${globalQuery}` : ""}`;

  const [social, seo, ads, jobs, insight] = await Promise.all([
    getOverviewSocial(client.id, range, previous),
    getOverviewSeo(client.id, range, previous),
    getOverviewAds(client.id, range, previous),
    getRecentSyncJobs(client.id),
    getLatestInsight(client.id, "OVERVIEW", range),
  ]);

  const d = (x: { pct: number | null; abs: number; direction: "up" | "down" | "flat" }, isDemo: boolean) =>
    compare && !isDemo ? x : null;

  return (
    <>
      <HeroStage
        eyebrow={t.nav.overview}
        title={
          <>
            {client.name} <span className="text-muted-foreground">dalam satu corong</span>
          </>
        }
        description={`${t.overview.subtitle} · ${formatDateRange(range.from, range.to)}`}
        actions={
          <>
            <Button asChild variant="outline" size="sm">
              <Link href={detailHref("settings")}>
                <Settings className="size-4" /> {t.common.settings}
              </Link>
            </Button>
            <Button asChild size="sm">
              <Link href={detailHref("reports")}>
                <FileText className="size-4" /> {t.reports.builder}
              </Link>
            </Button>
          </>
        }
        stats={[
          {
            label: t.social.followers,
            value: formatCompact(social.followers),
            delta: d(social.followersDelta, social.isDemo),
            href: detailHref("social"),
            hint: "Total followers semua akun sendiri (IG, FB, TikTok). Delta = pertumbuhan periode ini vs periode sebelumnya.",
          },
          {
            label: `${t.seo.clicks} organik`,
            value: formatCompact(seo.clicks),
            delta: d(seo.clicksDelta, seo.isDemo),
            href: detailHref("seo"),
            hint: "Klik organik dari Google Search Console.",
          },
          {
            label: t.ads.spend,
            value: formatCurrency(ads.kpis.spend, client.currency, { compact: true }),
            delta: d(ads.spendDelta, ads.isDemo),
            href: detailHref("ads"),
          },
          {
            label: t.ads.cpr,
            value: formatCurrency(ads.kpis.cpr, client.currency),
            delta: d(ads.cprDelta, ads.isDemo),
            lowerIsBetter: true,
            href: detailHref("ads"),
            hint: "Belanja iklan / hasil konversi. Lebih rendah lebih baik.",
          },
        ]}
      />

      <div className="grid gap-5 lg:grid-cols-3">
        <SocialModuleCard data={social} href={detailHref("social")} settingsHref={detailHref("settings")} />
        <SeoModuleCard data={seo} href={detailHref("seo")} settingsHref={detailHref("settings")} />
        <AdsModuleCard
          data={ads}
          href={detailHref("ads")}
          settingsHref={detailHref("settings")}
          currency={client.currency}
        />
      </div>

      <div className="grid gap-5 lg:grid-cols-3">
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
          <SyncActivity jobs={jobs} baseHref={base} />
        </div>
      </div>
    </>
  );
}

function pickGlobalQuery(sp: Record<string, string | string[] | undefined>): string {
  const query = new URLSearchParams();
  for (const key of ["from", "to", "preset", "compare"]) {
    const value = sp[key];
    if (typeof value === "string" && value) query.set(key, value);
  }
  return query.toString();
}
