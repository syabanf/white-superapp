import type { Metadata } from "next";
import Link from "next/link";
import { FileText, Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/dashboard/page-header";
import { EmptyState } from "@/components/dashboard/empty-state";
import { DemoBanner } from "@/components/dashboard/banners";
import { PostsTable } from "@/features/publishing/components/posts-table";
import { isPostsTab } from "@/features/publishing/posts-tabs";
import { listPosts } from "@/features/publishing/queries";
import { p } from "@/features/publishing/strings";
import { isPublisherConfigured } from "@/lib/providers/social-publisher";
import { requireClientAccess } from "@/lib/rbac";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/publish/posts">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await requireClientAccess(slug);
  return { title: `${client.name} · ${t.nav.posts}` };
}

export default async function PublishPostsPage(props: PageProps<"/clients/[slug]/publish/posts">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, canManage } = await requireClientAccess(slug);
  const raw = (Array.isArray(sp.status) ? sp.status[0] : sp.status)?.toUpperCase();
  const tab = isPostsTab(raw) ? raw : "ALL";
  const { rows, counts } = await listPosts(client.id);
  const base = `/clients/${slug}/publish`;

  return (
    <>
      <PageHeader
        eyebrow={t.nav.publish}
        title={p.postsTitle}
        description={p.postsSubtitle}
        actions={
          canManage ? (
            <Button asChild size="sm">
              <Link href={`${base}/new`}>
                <Plus className="size-3.5" /> {t.nav.compose}
              </Link>
            </Button>
          ) : null
        }
      />
      {!isPublisherConfigured() ? <DemoBanner message={p.demoBanner} settingsHref={`/clients/${slug}/settings`} /> : null}
      {rows.length === 0 ? (
        <EmptyState
          icon={<FileText />}
          title={p.noPosts}
          description={p.noPostsDesc}
          action={
            canManage ? (
              <Button asChild size="sm">
                <Link href={`${base}/new`}>{t.nav.compose}</Link>
              </Button>
            ) : undefined
          }
        />
      ) : (
        <PostsTable rows={rows} counts={counts} tab={tab} slug={slug} timezone={client.timezone} />
      )}
    </>
  );
}
