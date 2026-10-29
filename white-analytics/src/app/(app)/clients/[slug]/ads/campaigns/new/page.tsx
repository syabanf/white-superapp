import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { PageHeader } from "@/components/dashboard/page-header";
import { CampaignBuilder } from "@/features/campaigns/components/builder";
import { getBuilderContext } from "@/features/campaigns/queries";
import { todayKey } from "@/features/campaigns/lib";
import { c } from "@/features/campaigns/strings";
import { requireClientAccess } from "@/lib/rbac";
import { withClientMode } from "@/lib/client-mode";

export async function generateMetadata(props: PageProps<"/clients/[slug]/ads/campaigns/new">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${c.newTitle}` };
}

async function NewCampaignPageInner(props: PageProps<"/clients/[slug]/ads/campaigns/new">) {
  const { slug } = await props.params;
  const { client, canManage } = await requireClientAccess(slug);
  const base = `/clients/${slug}/ads/campaigns`;
  if (!canManage) redirect(base);
  const ctx = await getBuilderContext(client.id);

  return (
    <>
      <PageHeader back={{ href: base, label: c.backToList }} eyebrow={c.listTitle} title={c.newTitle} />
      <CampaignBuilder
        clientId={client.id}
        slug={slug}
        today={todayKey(new Date(), client.timezone)}
        clientCurrency={client.currency}
        adAccounts={ctx.adAccounts}
        images={ctx.images}
        brand={ctx.brand}
        noAccountGate={ctx.noAccountGate}
        initial={null}
        canManage={canManage}
      />
    </>
  );
}

/** Klien demo dirender dalam mode demo: AI mock dan pengiriman simulasi. */
export default async function NewCampaignPage(props: Parameters<typeof NewCampaignPageInner>[0]) {
  const { slug } = await props.params;
  return withClientMode({ slug }, () => NewCampaignPageInner(props));
}
