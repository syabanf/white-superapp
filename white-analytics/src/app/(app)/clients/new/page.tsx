import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { db } from "@/lib/db";
import { PageHeader } from "@/components/dashboard/page-header";
import { WizardShell } from "@/features/clients/components/wizard/wizard-shell";
import { StepProfile } from "@/features/clients/components/wizard/step-profile";
import { StepTypes } from "@/features/clients/components/wizard/step-types";
import { StepSources } from "@/features/clients/components/wizard/step-sources";
import { StepAccess } from "@/features/clients/components/wizard/step-access";
import { StepDone } from "@/features/clients/components/wizard/step-done";
import { getClientSettings, listUserOptions } from "@/features/clients/queries";
import { WIZARD_META, WIZARD_STEPS, parseTypes, resolveStep, wizardHref, type WizardStep } from "@/features/clients/wizard";
import { googleConfigured } from "@/lib/providers/google-oauth";
import { metaConfigured } from "@/lib/providers/meta-oauth";
import { requireUser } from "@/lib/rbac";
import { tc } from "@/features/clients/strings";

export const metadata: Metadata = { title: tc.wizard.title };

/**
 * Client onboarding wizard. The client record is created at step 1 and every
 * later step writes straight through, so the flow is resumable from its URL
 * (`/clients/new?step=…&client=<slug>`) and nothing is lost on abandon.
 */
export default async function NewClientWizardPage(props: PageProps<"/clients/new">) {
  const [sp, user] = await Promise.all([props.searchParams, requireUser()]);
  if (user.role !== "ADMIN") redirect("/clients?unauthorized=1");

  const slugParam = typeof sp.client === "string" ? sp.client : undefined;
  const stepParam = typeof sp.step === "string" ? sp.step : undefined;
  const types = parseTypes(typeof sp.types === "string" ? sp.types : undefined);

  const client = slugParam
    ? await db.client.findUnique({
        where: { slug: slugParam },
        select: {
          id: true,
          name: true,
          slug: true,
          shareEnabled: true,
          sharePinHash: true,
          shareModules: true,
        },
      })
    : null;

  const step = resolveStep(stepParam, { hasClient: !!client, hasTypes: types.length > 0 });
  const steps = WIZARD_STEPS.map((k) => WIZARD_META[k]);

  // Type and profile happen before the client row exists.
  if (!client) {
    return (
      <>
        <PageHeader eyebrow={tc.wizard.eyebrow} title={tc.wizard.title} description={tc.wizard.subtitle} />
        <WizardShell steps={steps} current={step}>
          {step === "jenis" ? <StepTypes initial={types} /> : <StepProfile types={types} />}
        </WizardShell>
      </>
    );
  }

  const modules = client.shareModules as string[];
  const needsSettings = step === "sumber" || step === "akses" || step === "selesai";
  const [settings, candidates] = await Promise.all([
    needsSettings ? getClientSettings(client.id) : Promise.resolve(null),
    step === "akses" ? listUserOptions() : Promise.resolve([]),
  ]);

  const sourceCount = settings
    ? settings.socialAccounts.length + settings.seoProperties.length + settings.adAccounts.length
    : 0;

  return (
    <>
      <PageHeader eyebrow={tc.wizard.eyebrow} title={client.name} description={tc.wizard.subtitle} />
      <WizardShell steps={steps} current={step} hrefFor={(k) => wizardHref(k as WizardStep, { slug: client.slug })}>
        {step === "sumber" && settings ? (
          <StepSources
            clientId={client.id}
            slug={client.slug}
            modules={modules}
            settings={settings}
            metaConfigured={metaConfigured()}
            googleConfigured={googleConfigured()}
          />
        ) : null}

        {step === "akses" && settings ? (
          <StepAccess
            clientId={client.id}
            slug={client.slug}
            settings={settings}
            candidates={candidates}
            shareEnabled={client.shareEnabled}
            hasPin={!!client.sharePinHash}
            shareModules={modules}
            appUrl={process.env.NEXT_PUBLIC_APP_URL ?? ""}
          />
        ) : null}

        {step === "selesai" ? (
          <StepDone
            slug={client.slug}
            clientName={client.name}
            modules={modules}
            sourceCount={sourceCount}
            shareEnabled={client.shareEnabled}
          />
        ) : null}
      </WizardShell>
    </>
  );
}
