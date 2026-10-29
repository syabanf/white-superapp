import type { Metadata } from "next";
import Link from "next/link";
import { Megaphone, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { EmptyState } from "@/components/dashboard/empty-state";
import { DraftList } from "@/features/campaigns/components/draft-list";
import { SimulatedBadge } from "@/features/campaigns/components/status";
import { getBuilderContext, listDrafts } from "@/features/campaigns/queries";
import { c } from "@/features/campaigns/strings";
import { requireClientAccess } from "@/lib/rbac";
import { withClientMode } from "@/lib/client-mode";

export async function generateMetadata(props: PageProps<"/clients/[slug]/ads/campaigns">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${c.listTitle}` };
}

async function CampaignsPageInner(props: PageProps<"/clients/[slug]/ads/campaigns">) {
  const { slug } = await props.params;
  const { client, canManage } = await requireClientAccess(slug);
  const [drafts, ctx] = await Promise.all([listDrafts(client.id, client.currency), getBuilderContext(client.id)]);
  const base = `/clients/${slug}/ads/campaigns`;
  const allSimulated = [ctx.noAccountGate, ...ctx.adAccounts.map((a) => a.gate)].every((g) => g.mode === "simulation");

  return (
    <>
      <PageHeader
        back={{ href: `/clients/${slug}/ads`, label: c.backToAds }}
        eyebrow={c.eyebrow}
        title={c.listTitle}
        description={c.listSubtitle}
        actions={
          canManage ? (
            <Button asChild size="sm">
              <Link href={`${base}/new`}>
                <Plus /> {c.newDraft}
              </Link>
            </Button>
          ) : undefined
        }
      />
      {allSimulated ? (
        <div className="flex items-center gap-2" title={c.simulationBanner}>
          <SimulatedBadge />
        </div>
      ) : null}
      {drafts.length ? (
        <DraftList slug={slug} drafts={drafts} />
      ) : (
        <EmptyState
          icon={<Megaphone />}
          title={c.emptyTitle}
          description={canManage ? c.emptyDesc : c.emptyViewer}
          action={
            canManage ? (
              <Button asChild>
                <Link href={`${base}/new`}>
                  <Plus /> {c.openBuilder}
                </Link>
              </Button>
            ) : undefined
          }
        />
      )}
    </>
  );
}

/** Klien demo dirender dalam mode demo: adapter mock walau kredensial asli terpasang. */
export default async function CampaignsPage(props: Parameters<typeof CampaignsPageInner>[0]) {
  const { slug } = await props.params;
  return withClientMode({ slug }, () => CampaignsPageInner(props));
}
