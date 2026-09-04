import type { Metadata } from "next";
import { CheckCircle2, CircleAlert, FlaskConical } from "lucide-react";
import { Alert, AlertTitle } from "@/components/ui/alert";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { ClientProfileForm } from "@/features/clients/components/client-profile-form";
import { ConnectionCard } from "@/features/clients/components/connection-card";
import { DangerZone } from "@/features/clients/components/danger-zone";
import { MembersSection } from "@/features/clients/components/members-section";
import { AdAccountsList, SeoPropertiesList, SocialAccountsList } from "@/features/clients/components/resource-lists";
import { ShareSection } from "@/features/clients/components/share-section";
import { getClientSettings, listUserOptions } from "@/features/clients/queries";
import { googleConfigured } from "@/lib/providers/google-oauth";
import { metaConfigured } from "@/lib/providers/meta-oauth";
import { requireClientAccess, requireUser } from "@/lib/rbac";
import { t } from "@/i18n/id";
import { tc } from "@/features/clients/strings";

export async function generateMetadata(props: PageProps<"/clients/[slug]/settings">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.settings.title}` };
}

const ERROR_MESSAGES: Record<string, string> = {
  oauth_state: tc.settings.oauthState,
  oauth_denied: tc.settings.oauthDenied,
  unauthorized: t.auth.unauthorized,
  RATE_LIMIT: t.errors.RATE_LIMIT,
  TOKEN_EXPIRED: t.errors.TOKEN_EXPIRED,
  PERMISSION: t.errors.PERMISSION,
  NOT_FOUND: t.errors.NOT_FOUND,
  NETWORK: t.errors.NETWORK,
  UNKNOWN: t.errors.UNKNOWN,
};

export default async function ClientSettingsPage(props: PageProps<"/clients/[slug]/settings">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const [{ client, canManage }, user] = await Promise.all([requireClientAccess(slug), requireUser()]);
  const isAdmin = user.role === "ADMIN";

  const [settings, userOptions] = await Promise.all([
    getClientSettings(client.id),
    canManage ? listUserOptions() : Promise.resolve([]),
  ]);

  const metaConnections = settings.connections.filter((c) => c.provider === "META");
  const googleConnections = settings.connections.filter((c) => c.provider === "GOOGLE");
  const configured = { meta: metaConfigured(), google: googleConfigured() };
  const appUrl = process.env.NEXT_PUBLIC_APP_URL ?? "";

  const connected = typeof sp.connected === "string" ? sp.connected : undefined;
  const errorKey = typeof sp.error === "string" ? sp.error : undefined;
  const showDemoNotice = sp.demo === "1";

  return (
    <>
      <PageHeader eyebrow={client.name} title={t.settings.title} description={t.settings.subtitle} />

      {connected === "meta" || connected === "google" ? (
        <Alert className="border-[color-mix(in_oklab,var(--status-good)_35%,transparent)] py-2 [&>svg]:text-[var(--status-good)]">
          <CheckCircle2 className="size-4" />
          <AlertTitle className="text-xs font-medium">
            {connected === "meta" ? tc.settings.connectedMeta : tc.settings.connectedGoogle}
          </AlertTitle>
        </Alert>
      ) : null}
      {showDemoNotice ? (
        <Alert className="border-dashed bg-muted/40 py-2">
          <FlaskConical className="size-4" />
          <AlertTitle className="text-xs font-medium">{t.settings.notConfigured}</AlertTitle>
        </Alert>
      ) : null}
      {errorKey ? (
        <Alert variant="destructive" className="py-2">
          <CircleAlert className="size-4" />
          <AlertTitle className="text-xs font-medium">{ERROR_MESSAGES[errorKey] ?? t.errors.UNKNOWN}</AlertTitle>
        </Alert>
      ) : null}

      <Section title={t.settings.profile} description={tc.settings.profileDesc}>
        <ClientProfileForm
          client={{
            id: client.id,
            name: client.name,
            slug: client.slug,
            description: client.description,
            industry: client.industry ?? "",
            websiteUrl: client.websiteUrl ?? "",
            currency: client.currency,
            timezone: client.timezone,
          }}
          canManage={canManage}
        />
      </Section>

      <Section title={t.settings.connections} description={t.settings.connectionsDesc}>
        <div className="grid items-start gap-4 lg:grid-cols-2">
          <ConnectionCard
            provider="META"
            clientId={client.id}
            slug={client.slug}
            connections={metaConnections}
            configured={configured.meta}
            canManage={canManage}
          >
            <SocialAccountsList clientId={client.id} accounts={settings.socialAccounts} canManage={canManage} />
            <AdAccountsList clientId={client.id} accounts={settings.adAccounts} canManage={canManage} />
          </ConnectionCard>
          <ConnectionCard
            provider="GOOGLE"
            clientId={client.id}
            slug={client.slug}
            connections={googleConnections}
            configured={configured.google}
            canManage={canManage}
          >
            <SeoPropertiesList clientId={client.id} properties={settings.seoProperties} canManage={canManage} />
          </ConnectionCard>
        </div>
      </Section>

      {canManage ? (
        <Section title={t.settings.members} description={t.settings.membersDesc}>
          <MembersSection clientId={client.id} members={settings.members} candidates={userOptions} canManage={canManage} />
        </Section>
      ) : null}

      <Section title={t.reports.share} description={t.reports.shareDesc}>
        <ShareSection
          clientId={client.id}
          slug={client.slug}
          shareEnabled={client.shareEnabled}
          hasPin={client.sharePinHash != null}
          shareModules={client.shareModules}
          appUrl={appUrl}
          canManage={canManage}
        />
      </Section>

      {isAdmin ? (
        <Section title={t.settings.danger} description={t.settings.dangerDesc}>
          <DangerZone clientId={client.id} slug={client.slug} name={client.name} />
        </Section>
      ) : null}
    </>
  );
}
