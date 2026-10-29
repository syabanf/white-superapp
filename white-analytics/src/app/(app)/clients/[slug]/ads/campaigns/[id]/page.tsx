import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { PageHeader } from "@/components/dashboard/page-header";
import { CampaignBuilder } from "@/features/campaigns/components/builder";
import { DraftStatusBadge, SimulatedBadge } from "@/features/campaigns/components/status";
import { getBuilderContext, getDraft } from "@/features/campaigns/queries";
import { todayKey } from "@/features/campaigns/lib";
import { c } from "@/features/campaigns/strings";
import { requireClientAccess } from "@/lib/rbac";
import { withClientMode } from "@/lib/client-mode";

export async function generateMetadata(props: PageProps<"/clients/[slug]/ads/campaigns/[id]">): Promise<Metadata> {
  const { slug, id } = await props.params;
  const { client } = await requireClientAccess(slug);
  const draft = await getDraft(client.id, id);
  return { title: `${client.name} · ${draft?.name || c.editTitle}` };
}

async function CampaignDraftPageInner(props: PageProps<"/clients/[slug]/ads/campaigns/[id]">) {
  const { slug, id } = await props.params;
  const { client, canManage } = await requireClientAccess(slug);
  const [draft, ctx] = await Promise.all([getDraft(client.id, id), getBuilderContext(client.id)]);
  if (!draft) notFound();
  const base = `/clients/${slug}/ads/campaigns`;

  return (
    <>
      <PageHeader
        back={{ href: base, label: c.backToList }}
        eyebrow={c.listTitle}
        title={draft.name || c.editTitle}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <DraftStatusBadge status={draft.status} />
            {draft.status === "SUBMITTED" && draft.simulated ? <SimulatedBadge /> : null}
          </div>
        }
      />
      <CampaignBuilder
        key={draft.updatedAt}
        clientId={client.id}
        slug={slug}
        today={todayKey(new Date(), client.timezone)}
        clientCurrency={client.currency}
        adAccounts={ctx.adAccounts}
        images={ctx.images}
        brand={ctx.brand}
        noAccountGate={ctx.noAccountGate}
        initial={draft}
        canManage={canManage}
      />
    </>
  );
}

/** Klien demo dirender dalam mode demo: AI mock dan pengiriman simulasi. */
export default async function CampaignDraftPage(props: Parameters<typeof CampaignDraftPageInner>[0]) {
  const { slug } = await props.params;
  return withClientMode({ slug }, () => CampaignDraftPageInner(props));
}
