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
        const [social, seo, ads] = await Promise.all([
          getOverviewSocial(c.id, range, previous),
          getOverviewSeo(c.id, range, previous),
          getOverviewAds(c.id, range, previous),
        ]);
        return { client: c, social, seo, ads };
      }),
    ),
  ]);

  const healths = perClient.map((p) => p.seo.health).filter((h): h is number => h != null);
  const avgHealth = healths.length > 0 ? Math.round(mean(healths)) : null;
  const d = (x: Delta): Delta | null => (compare ? x : null);

  const cards: PortfolioCardData[] = perClient.map(({ client, social, seo, ads }) => ({
    id: client.id,
    name: client.name,
    slug: client.slug,
    industry: client.industry,
    logoUrl: client.logoUrl,
    currency: client.currency,
    followers: social.followers,
    followersDelta: d(social.followersDelta),
    clicks: seo.clicks,
    clicksDelta: d(seo.clicksDelta),
    spend: ads.kpis.spend,
    spendDelta: d(ads.spendDelta),
    health: seo.health,
    modules: { social: social.hasData, seo: seo.hasData, ads: ads.hasData },
    lastSyncLabel: lastSync[client.id] ? formatRelative(lastSync[client.id]) : null,
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
          { label: t.portfolio.followers, value: formatCompact(totals.followers), delta: d(totals.followersDelta), hint: tc.portfolio.followersHint },
          { label: t.portfolio.organicClicks, value: formatCompact(totals.clicks), delta: d(totals.clicksDelta), hint: tc.portfolio.clicksHint },
          { label: t.portfolio.adSpend, value: formatCurrency(totals.spend, "IDR", { compact: true }), delta: d(totals.spendDelta), hint: tc.portfolio.spendHint },
          {
            label: t.portfolio.healthScore,
            value: avgHealth != null ? formatNumber(avgHealth) : "–",
            caption: avgHealth != null ? tc.portfolio.healthOutOf : tc.portfolio.noAudit,
            hint: tc.portfolio.healthHint,
          },
        ]}
      />
      {showUnauthorized ? <UnauthorizedNotice /> : null}

      <PortfolioGrid cards={cards} />
    </>
  );
}

function UnauthorizedNotice() {
  return (
    <Alert variant="destructive" className="py-2">
      <ShieldAlert className="size-4" />
      <AlertTitle className="text-xs font-medium">{t.auth.unauthorized}</AlertTitle>
    </Alert>
  );
}
