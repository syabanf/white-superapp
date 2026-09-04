import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { DemoBanner } from "@/components/dashboard/banners";
import { Composer } from "@/features/publishing/components/composer/composer";
import { getBestTimes, getOwnAccounts, listMedia } from "@/features/publishing/queries";
import { todayIn } from "@/features/publishing/time";
import { p } from "@/features/publishing/strings";
import { isPublisherConfigured } from "@/lib/providers/social-publisher";
import { requireClientAccess } from "@/lib/rbac";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/publish/new">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.nav.compose}` };
}

export default async function PublishNewPage(props: PageProps<"/clients/[slug]/publish/new">) {
  const { slug } = await props.params;
  const { client, role } = await requireClientAccess(slug);
  const [accounts, library, bestTimes] = await Promise.all([getOwnAccounts(client.id), listMedia(client.id), getBestTimes(client.id, client.timezone)]);

  return (
    <>
      <PageHeader eyebrow={t.nav.publish} title={p.composeTitle} description={p.composeSubtitle} />
      {!isPublisherConfigured() ? <DemoBanner message={p.demoBanner} settingsHref={`/clients/${slug}/settings`} /> : null}
      <Composer
        clientId={client.id}
        slug={slug}
        timezone={client.timezone}
        today={todayIn(client.timezone)}
        accounts={accounts}
        library={library}
        bestTimes={bestTimes}
        initial={null}
        role={role}
      />
    </>
  );
}
