import type { Metadata } from "next";
import Link from "next/link";
import { Palette } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { DemoBanner } from "@/components/dashboard/banners";
import { InfoHint } from "@/components/dashboard/info-hint";
import { Button } from "@/components/ui/button";
import { DesignStudio } from "@/features/design-studio/components/studio";
import { hasBrandKit, listDesignDrafts } from "@/features/design-studio/queries";
import { ds } from "@/features/design-studio/strings";
import { isAiStudioConfigured } from "@/lib/providers/ai-studio";
import { requireClientAccess } from "@/lib/rbac";
import { withClientMode } from "@/lib/client-mode";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content/studio">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${ds.title}` };
}

async function DesignStudioPageInner(props: PageProps<"/clients/[slug]/content/studio">) {
  const { slug } = await props.params;
  const { client, canManage } = await requireClientAccess(slug);
  const [drafts, brandKit] = await Promise.all([listDesignDrafts(client.id), hasBrandKit(client.id)]);
  // Runs inside withClientMode, so demo clients always report "not configured".
  const demo = !isAiStudioConfigured();
  const brandHref = `/clients/${slug}/content/brand`;

  return (
    <>
      <PageHeader
        eyebrow={ds.eyebrow}
        title={
          <span className="inline-flex items-center gap-2">
            {ds.title} <InfoHint className="mt-1">{ds.pageHint}</InfoHint>
          </span>
        }
        description={brandKit ? ds.description : ds.noBrandKit}
        actions={
          <Button asChild variant="secondary" size="sm">
            <Link href={brandHref}>
              <Palette /> {ds.brandKitLink}
            </Link>
          </Button>
        }
      />
      {demo ? <DemoBanner message={ds.demoBanner} /> : null}
      <DesignStudio slug={slug} drafts={drafts} canManage={canManage} />
    </>
  );
}

/** Klien demo dirender dalam mode demo: AI mock + DemoBanner walau kunci OpenRouter terpasang. */
export default async function DesignStudioPage(props: Parameters<typeof DesignStudioPageInner>[0]) {
  const { slug } = await props.params;
  return withClientMode({ slug }, () => DesignStudioPageInner(props));
}
