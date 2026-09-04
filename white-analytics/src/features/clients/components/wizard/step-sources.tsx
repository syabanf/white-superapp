import Link from "next/link";
import { ArrowRight, Plug } from "lucide-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { ConnectionCard } from "@/features/clients/components/connection-card";
import { AdAccountsList, SeoPropertiesList, SocialAccountsList } from "@/features/clients/components/resource-lists";
import { WizardNav } from "@/features/clients/components/wizard/wizard-nav";
import { sourceSectionsFor } from "@/features/clients/wizard";
import type { SettingsData } from "@/features/clients/queries";
import { tc } from "@/features/clients/strings";
import { wizardHref } from "@/features/clients/wizard";

/**
 * Step 3 — only shows the source sections for the channels picked in step 2.
 * Everything here is optional: the dashboard opens on demo data until a real
 * connection exists, so the operator can always come back later.
 */
export function StepSources({
  clientId,
  slug,
  modules,
  settings,
  metaConfigured,
  googleConfigured,
}: {
  clientId: string;
  slug: string;
  modules: string[];
  settings: SettingsData;
  metaConfigured: boolean;
  googleConfigured: boolean;
}) {
  const show = sourceSectionsFor(modules);
  const needsMeta = show.social || show.ads;

  return (
    <div className="space-y-7">
      <p className="text-sm text-muted-foreground">{tc.wizard.sourcesHint}</p>

      {(needsMeta && !metaConfigured) || (show.seo && !googleConfigured) ? (
        <Alert>
          <Plug className="size-4" />
          <AlertTitle>{tc.wizard.setupNeededTitle}</AlertTitle>
          <AlertDescription className="flex flex-wrap items-center justify-between gap-2">
            <span>{tc.wizard.setupNeededDesc}</span>
            <Button asChild size="xs" variant="outline">
              <Link href="/setup">{tc.wizard.setupOpen}</Link>
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}

      {needsMeta ? (
        <ConnectionCard
          provider="META"
          clientId={clientId}
          slug={slug}
          connections={settings.connections.filter((c) => c.provider === "META")}
          configured={metaConfigured}
          canManage
        >
          {show.social ? (
            <SocialAccountsList clientId={clientId} accounts={settings.socialAccounts} canManage />
          ) : null}
          {show.ads ? <AdAccountsList clientId={clientId} accounts={settings.adAccounts} canManage /> : null}
        </ConnectionCard>
      ) : null}

      {show.seo ? (
        <ConnectionCard
          provider="GOOGLE"
          clientId={clientId}
          slug={slug}
          connections={settings.connections.filter((c) => c.provider === "GOOGLE")}
          configured={googleConfigured}
          canManage
        >
          <SeoPropertiesList clientId={clientId} properties={settings.seoProperties} canManage />
        </ConnectionCard>
      ) : null}

      <WizardNav
        slug={slug}
        back={null}
        submit={
          <Button asChild className="group/nudge">
            <Link href={wizardHref("akses", { slug })}>
              {tc.wizard.next} <ArrowRight className="nudge nudge-x size-4" />
            </Link>
          </Button>
        }
      />
    </div>
  );
}
