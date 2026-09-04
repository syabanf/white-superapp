import type { Metadata } from "next";
import { Bot, Globe, Music2, Sparkles } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { StatusBadge } from "@/components/dashboard/status-badge";
import { PageHeader } from "@/components/dashboard/page-header";
import { WizardShell } from "@/features/clients/components/wizard/wizard-shell";
import { PlatformIcon } from "@/features/social/components/platform-icon";
import { CredentialCard } from "@/features/setup/components/credential-card";
import { MetaReviewChecklist } from "@/features/setup/components/meta-review-checklist";
import { SeoCostEstimator } from "@/features/setup/components/seo-cost-estimator";
import { ApifyCostEstimator } from "@/features/setup/components/apify-cost-estimator";
import { SetupNav } from "@/features/setup/components/setup-nav";
import { StepDone } from "@/features/setup/components/step-done";
import { getSetupOverview } from "@/features/setup/queries";
import { SETUP_META, SETUP_STEPS, resolveSetupStep, setupHref } from "@/features/setup/steps";
import { ts } from "@/features/setup/strings";
import { requireAdmin } from "@/lib/rbac";

export const metadata: Metadata = { title: ts.title };

/**
 * Workspace setup wizard (ADMIN). One pass per workspace: app-level credentials
 * for Meta / Google / DataForSEO / OpenRouter, the Meta App Review checklist,
 * and a cost estimate for paid SEO data. Everything is skippable — demo mode
 * keeps the whole product usable. Reachable later from Admin → Setup awal.
 */
export default async function SetupPage(props: PageProps<"/setup">) {
  const [sp] = await Promise.all([props.searchParams, requireAdmin()]);
  const step = resolveSetupStep(typeof sp.step === "string" ? sp.step : undefined);
  const overview = await getSetupOverview();
  const { statuses, appUrl } = overview;
  const steps = SETUP_STEPS.map((k) => SETUP_META[k]);
  const usdIdr = Number(process.env.DATAFORSEO_USD_IDR ?? 16_000);

  return (
    <>
      <PageHeader eyebrow={ts.eyebrow} title={ts.title} description={ts.subtitle} />
      <WizardShell steps={steps} current={step} hrefFor={(k) => setupHref(k as (typeof SETUP_STEPS)[number])} eyebrow={ts.eyebrow}>
        <div className="space-y-6">
          {step === "meta" ? (
            <>
              <CredentialCard
                integration="META"
                status={statuses.META}
                icon={<PlatformIcon platform="FACEBOOK" className="size-4" />}
                labels={{
                  title: SETUP_META.meta.title,
                  description: ts.metaWhere,
                  publicLabel: ts.metaAppId,
                  secretLabel: ts.metaAppSecret,
                  publicPlaceholder: "1234567890123456",
                  where: ts.metaRedirectHint,
                }}
                redirectUri={`${appUrl}/api/connections/meta/callback`}
              />
              <MetaReviewChecklist initial={overview.metaReview} />
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2 text-sm font-semibold">
                    <Music2 className="size-4 text-muted-foreground" /> {ts.tiktokTitle}
                    <StatusBadge kind="neutral">{ts.modeDemo}</StatusBadge>
                  </CardTitle>
                  <CardDescription className="text-xs">{ts.tiktokDesc}</CardDescription>
                </CardHeader>
                <CardContent className="text-xs text-muted-foreground">
                  <PlatformIcon platform="TIKTOK" className="mr-1.5 inline size-3.5" />
                  {ts.tiktokMode}
                </CardContent>
              </Card>
            </>
          ) : null}

          {step === "google" ? (
            <CredentialCard
              integration="GOOGLE"
              status={statuses.GOOGLE}
              icon={<Globe className="size-4 text-muted-foreground" />}
              labels={{
                title: SETUP_META.google.title,
                description: ts.googleWhere,
                publicLabel: ts.googleClientId,
                secretLabel: ts.googleClientSecret,
                publicPlaceholder: "xxxx.apps.googleusercontent.com",
                where: ts.googleRedirectHint,
              }}
              redirectUri={`${appUrl}/api/connections/google/callback`}
            />
          ) : null}

          {step === "seo-data" ? (
            <>
              <CredentialCard
                integration="DATAFORSEO"
                status={statuses.DATAFORSEO}
                labels={{
                  title: SETUP_META["seo-data"].title,
                  description: ts.dfsWhere,
                  publicLabel: ts.dfsLogin,
                  secretLabel: ts.dfsPassword,
                  publicPlaceholder: "nama@agency.id",
                  where: ts.costDesc,
                }}
              />
              <SeoCostEstimator trackedKeywords={overview.counts.trackedKeywords} competitorDomains={overview.counts.competitorDomains} usdIdr={usdIdr} />
            </>
          ) : null}

          {step === "apify" ? (
            <>
              <CredentialCard
                integration="APIFY"
                status={statuses.APIFY}
                icon={<Bot className="size-4 text-muted-foreground" />}
                labels={{
                  title: SETUP_META.apify.title,
                  description: ts.apifyWhere,
                  publicLabel: ts.apifyActors,
                  secretLabel: ts.apifyToken,
                  publicPlaceholder: '{"serp":"apify/google-search-scraper"}',
                  where: ts.apifyPriority,
                  publicOptional: true,
                }}
              />
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm font-semibold">{ts.apifyReplaces}</CardTitle>
                </CardHeader>
                <CardContent>
                  <ul className="space-y-2 text-sm text-muted-foreground">
                    <li className="flex gap-2"><span className="label-mono mt-1 shrink-0 text-brand">SERP</span><span>{ts.apifySerp}</span></li>
                    <li className="flex gap-2"><span className="label-mono mt-1 shrink-0 text-brand">SOCIAL</span><span>{ts.apifySocial}</span></li>
                    <li className="flex gap-2"><span className="label-mono mt-1 shrink-0 text-brand">TIKTOK</span><span>{ts.apifyTiktok}</span></li>
                  </ul>
                </CardContent>
              </Card>
              <ApifyCostEstimator trackedKeywords={overview.counts.trackedKeywords} socialAccounts={overview.counts.socialAccounts} usdIdr={usdIdr} />
            </>
          ) : null}

          {step === "ai" ? (
            <CredentialCard
              integration="OPENROUTER"
              status={statuses.OPENROUTER}
              icon={<Sparkles className="size-4 text-muted-foreground" />}
              labels={{
                title: SETUP_META.ai.title,
                description: ts.aiWhere,
                publicLabel: ts.aiModel,
                secretLabel: ts.aiKey,
                publicPlaceholder: "anthropic/claude-sonnet-4.5",
                where: SETUP_META.ai.description,
              }}
            />
          ) : null}

          {step === "selesai" ? <StepDone overview={overview} /> : <SetupNav step={step} />}
        </div>
      </WizardShell>
    </>
  );
}
