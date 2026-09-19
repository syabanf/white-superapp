import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { MembersSection } from "@/features/clients/components/members-section";
import { ShareSection } from "@/features/clients/components/share-section";
import { WizardNav } from "@/features/clients/components/wizard/wizard-nav";
import { wizardHref } from "@/features/clients/wizard";
import type { SettingsData } from "@/features/clients/queries";
import { tc } from "@/features/clients/strings";

/** Step 4 — team access and the read-only client link. Both optional. */
export function StepAccess({
  clientId,
  slug,
  settings,
  candidates,
  shareEnabled,
  hasPin,
  shareModules,
  appUrl,
}: {
  clientId: string;
  slug: string;
  settings: SettingsData;
  candidates: { id: string; name: string; email: string }[];
  shareEnabled: boolean;
  hasPin: boolean;
  shareModules: string[];
  appUrl: string;
}) {
  return (
    <div className="space-y-7">
      <p className="text-sm text-muted-foreground">{tc.wizard.accessHint}</p>

      <MembersSection clientId={clientId} members={settings.members} candidates={candidates} canManage />

      <ShareSection
        clientId={clientId}
        slug={slug}
        shareEnabled={shareEnabled}
        hasPin={hasPin}
        shareModules={shareModules}
        appUrl={appUrl}
        canManage
      />

      <WizardNav
        slug={slug}
        back="sumber"
        submit={
          <Button asChild className="group/nudge">
            <Link href={wizardHref("selesai", { slug })}>
              {tc.wizard.reviewSetup} <ArrowRight className="nudge nudge-x size-4" />
            </Link>
          </Button>
        }
      />
    </div>
  );
}
