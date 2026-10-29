import type { Metadata } from "next";
import Link from "next/link";
import { Palette, Wand2 } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { EmptyState } from "@/components/dashboard/empty-state";
import { InfoHint } from "@/components/dashboard/info-hint";
import { Button } from "@/components/ui/button";
import { BrandKitForm } from "@/features/brand/components/brand-kit-form";
import { getBrandKit } from "@/features/brand/queries";
import { b } from "@/features/brand/strings";
import { ds } from "@/features/design-studio/strings";
import { requireClientAccess } from "@/lib/rbac";
import { withClientMode } from "@/lib/client-mode";
import { formatRelative } from "@/lib/format";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content/brand">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${b.title}` };
}

async function BrandKitPageInner(props: PageProps<"/clients/[slug]/content/brand">) {
  const { slug } = await props.params;
  const { client, canManage } = await requireClientAccess(slug);
  const kit = await getBrandKit(client.id);
  const studioHref = `/clients/${slug}/content/studio`;

  return (
    <>
      <PageHeader
        eyebrow={b.eyebrow}
        title={
          <span className="inline-flex items-center gap-2">
            {b.title} <InfoHint className="mt-1">{b.pageHint}</InfoHint>
          </span>
        }
        description={
          kit
            ? `${b.lastUpdated} ${formatRelative(kit.updatedAt)}${kit.updatedByName ? ` · ${kit.updatedByName}` : ""}`
            : b.description
        }
        actions={
          <Button asChild variant="secondary" size="sm">
            <Link href={studioHref}>
              <Wand2 /> {ds.title}
            </Link>
          </Button>
        }
      />

      {!kit ? (
        <EmptyState compact icon={<Palette className="size-5" />} title={b.emptyTitle} description={canManage ? b.emptyDesc : `${b.emptyDesc} ${b.readOnly}`} />
      ) : !canManage ? (
        <p className="text-sm text-muted-foreground">{b.readOnly}</p>
      ) : null}

      <BrandKitForm slug={slug} clientName={client.name} initial={kit} canManage={canManage} />
    </>
  );
}

/** Klien demo dirender dalam mode demo, sama seperti halaman lain di bawah klien. */
export default async function BrandKitPage(props: Parameters<typeof BrandKitPageInner>[0]) {
  const { slug } = await props.params;
  return withClientMode({ slug }, () => BrandKitPageInner(props));
}
