import type { Metadata } from "next";
import { PageHeader } from "@/components/dashboard/page-header";
import { MediaLibrary } from "@/features/publishing/components/media/media-library";
import { listMedia } from "@/features/publishing/queries";
import { p } from "@/features/publishing/strings";
import { requireClientAccess } from "@/lib/rbac";
import { formatNumber } from "@/lib/format";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/publish/media">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.nav.media}` };
}

export default async function PublishMediaPage(props: PageProps<"/clients/[slug]/publish/media">) {
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
