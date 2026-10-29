import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { MediaLibrary } from "@/features/publishing/components/media/media-library";
import { listMedia } from "@/features/publishing/queries";
import { p } from "@/features/publishing/strings";
import { requireClientAccess } from "@/lib/rbac";
import { formatNumber } from "@/lib/format";
import { t } from "@/i18n/id";
import { withClientMode } from "@/lib/client-mode";

export async function generateMetadata(props: PageProps<"/clients/[slug]/publish/media">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.nav.media}` };
}

async function PublishMediaPageInner(props: PageProps<"/clients/[slug]/publish/media">) {
  const { slug } = await props.params;
  const { client, role } = await requireClientAccess(slug);
  const assets = await listMedia(client.id);
  const images = assets.filter((a) => a.kind === "IMAGE").length;

  return (
    <>
      <PageHeader
        eyebrow={t.nav.publish}
        title={p.mediaTitle}
        description={`${p.mediaSubtitle} · ${formatNumber(images)} ${p.kindImage.toLowerCase()} · ${formatNumber(assets.length - images)} ${p.kindVideo.toLowerCase()}`}
      />
      <MediaLibrary clientId={client.id} assets={assets} role={role} />
    </>
  );
}

/** Klien demo dirender dalam mode demo: adapter mock + DemoBanner walau kredensial asli terpasang. */
export default async function PublishMediaPage(props: Parameters<typeof PublishMediaPageInner>[0]) {
  const { slug } = await props.params;
  return withClientMode({ slug }, () => PublishMediaPageInner(props));
}
