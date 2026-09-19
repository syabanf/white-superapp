import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";
import { Briefcase, Plus, ShieldAlert } from "lucide-react";
import { Alert, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { EmptyState } from "@/components/dashboard/empty-state";
import { HeroStage } from "@/components/dashboard/hero-stage";
import { PageHeader } from "@/components/dashboard/page-header";
import { PortfolioGrid, type PortfolioCardData } from "@/features/clients/components/portfolio-grid";
import { getLatestSyncByClient, getPortfolioTotals } from "@/features/clients/queries";
import { getOverviewAds, getOverviewSeo, getOverviewSocial } from "@/features/overview/queries";
import { getPublishingStats } from "@/features/publishing/queries";
import { listAccessibleClients, requireUser } from "@/lib/rbac";
import { getSetupState } from "@/lib/integrations";
import { getRange } from "@/lib/range-params";
import { formatCompact, formatCurrency, formatDateRange, formatNumber, formatRelative } from "@/lib/format";
import { mean, type Delta } from "@/lib/metrics";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";

export const metadata: Metadata = { title: t.portfolio.title };

export default async function PortfolioPage(props: PageProps<"/">) {
  const sp = await props.searchParams;
  const [user, clients] = await Promise.all([requireUser(), listAccessibleClients()]);
  const { range, previous, compare } = getRange(sp);
  const isAdmin = user.role === "ADMIN";
  const showUnauthorized = sp.unauthorized === "1";

  // First run: an admin with no clients and an untouched setup lands in the setup wizard once.
  if (isAdmin && clients.length === 0) {
    const setup = await getSetupState();
    if (!setup.completedAt && !setup.skippedAt) redirect("/setup");
  }

  const newClientAction = isAdmin ? (
    <Button asChild size="sm">
      <Link href="/clients/new">
        <Plus className="size-4" /> {t.portfolio.newClient}
      </Link>
    </Button>
  ) : undefined;

  const header = (
    <PageHeader
      eyebrow={t.nav.portfolio}
      title={t.portfolio.title}
      description={`${t.portfolio.subtitle} · ${clients.length} ${t.portfolio.clientsCount} · ${formatDateRange(range.from, range.to)}`}
      actions={newClientAction}
    />
  );

  if (clients.length === 0) {
    return (
      <>
        {header}
        {showUnauthorized ? <UnauthorizedNotice /> : null}
        <EmptyState
          icon={<Briefcase className="size-5" />}
          title={t.portfolio.emptyTitle}
          description={t.portfolio.emptyDesc}
          action={
            isAdmin ? (
              <Button asChild>
                <Link href="/clients/new">
                  <Plus className="size-4" /> {t.portfolio.newClient}
                </Link>
              </Button>
            ) : undefined
          }
        />
      </>
    );
  }

  const clientIds = clients.map((c) => c.id);
  const [totals, lastSync, perClient] = await Promise.all([
    getPortfolioTotals(clientIds, range, previous),
    getLatestSyncByClient(clientIds),
    Promise.all(
      clients.map(async (c) => {
        const [social, seo, ads, publishing] = await Promise.all([
          getOverviewSocial(c.id, range, previous),
          getOverviewSeo(c.id, range, previous),
          getOverviewAds(c.id, range, previous),
          getPublishingStats(c.id),
        ]);
        return { client: c, social, seo, ads, publishing };
      }),
    ),
  ]);

  const healths = perClient.map((p) => p.seo.health).filter((h): h is number => h != null);
  const avgHealth = healths.length > 0 ? Math.round(mean(healths)) : null;
  const allDemo = perClient.every((p) => p.social.isDemo && p.seo.isDemo && p.ads.isDemo);
  const d = (x: Delta, isDemo = allDemo): Delta | null => (compare && !isDemo ? x : null);

  const globalQuery = pickGlobalQuery(sp);
  const cards: PortfolioCardData[] = perClient.map(({ client, social, seo, ads, publishing }) => ({
    id: client.id,
    name: client.name,
    slug: client.slug,
    industry: client.industry,
    logoUrl: client.logoUrl,
    currency: client.currency,
    followers: social.followers,
    followersDelta: d(social.followersDelta, social.isDemo),
    clicks: seo.clicks,
    clicksDelta: d(seo.clicksDelta, seo.isDemo),
    spend: ads.kpis.spend,
    spendDelta: d(ads.spendDelta, ads.isDemo),
    health: seo.health,
    modules: { social: social.hasData, seo: seo.hasData, ads: ads.hasData },
    lastSyncLabel: lastSync[client.id] ? formatRelative(lastSync[client.id]) : null,
    contextQuery: globalQuery,
    attention: [
      ...(publishing.failed > 0
        ? [
            {
              label: `${publishing.failed} publikasi gagal`,
              href: "publish/posts?status=FAILED",
              kind: "critical" as const,
            },
          ]
        : []),
      ...(publishing.inReview > 0
        ? [
            {
              label: `${publishing.inReview} post menunggu review`,
              href: "publish/posts?status=IN_REVIEW",
              kind: "warning" as const,
            },
          ]
        : []),
      ...(seo.health != null && seo.health < 60
        ? [{ label: "Audit SEO perlu perhatian", href: "seo/audit", kind: "warning" as const }]
        : []),
      ...(!lastSync[client.id]
        ? [{ label: "Belum pernah disinkronkan", href: "settings", kind: "warning" as const }]
        : []),
    ].slice(0, 2),
  }));

  return (
    <>
      <HeroStage
        eyebrow={t.nav.portfolio}
        title={
          <>
            Halo, {user.name.split(" ")[0]} 👋{" "}
            <span className="text-muted-foreground">bagaimana performa</span> klien hari ini?
          </>
        }
        description={`${clients.length} ${t.portfolio.clientsCount} · ${formatDateRange(range.from, range.to)}`}
        actions={newClientAction}
        stats={[
          {
            label: t.portfolio.followers,
            value: formatCompact(totals.followers),
            delta: d(totals.followersDelta),
            href: "#clients",
            hint: tc.portfolio.followersHint,
          },
          {
            label: t.portfolio.organicClicks,
            value: formatCompact(totals.clicks),
            delta: d(totals.clicksDelta),
            href: "#clients",
            hint: tc.portfolio.clicksHint,
          },
          {
            label: t.portfolio.adSpend,
            value: formatCurrency(totals.spend, "IDR", { compact: true }),
            delta: d(totals.spendDelta),
            href: "#clients",
            hint: tc.portfolio.spendHint,
          },
          {
            label: t.portfolio.healthScore,
            value: avgHealth != null ? formatNumber(avgHealth) : "–",
            caption: avgHealth != null ? tc.portfolio.healthOutOf : tc.portfolio.noAudit,
            hint: tc.portfolio.healthHint,
            href: "#clients",
          },
        ]}
      />
      {showUnauthorized ? <UnauthorizedNotice /> : null}

      <PortfolioGrid cards={cards} />
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

function UnauthorizedNotice() {
  return (
    <Alert variant="destructive" className="py-2">
      <ShieldAlert className="size-4" />
      <AlertTitle className="text-xs font-medium">{t.auth.unauthorized}</AlertTitle>
    </Alert>
  );
}
