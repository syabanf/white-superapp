import type { Metadata } from "next";
import { FolderOpen } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { EmptyState } from "@/components/dashboard/empty-state";
import { AssetLibrary } from "@/features/content/components/assets/asset-library";
import { ContentOffline } from "@/features/content/components/offline";
import { isAssetKind } from "@/features/content/lib";
import { contentContext, listFolders, loadContent, searchAssets } from "@/features/content/queries";
import { c } from "@/features/content/strings";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content/assets">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await contentContext(slug);
  return { title: `${client.name} · ${t.nav.contentAssets}` };
}

const one = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v) ?? "";

export default async function ContentAssetsPage(props: PageProps<"/clients/[slug]/content/assets">) {
  const [{ slug }, sp] = await Promise.all([props.params, props.searchParams]);
  const { client, canWrite } = await contentContext(slug);
  const rawKind = one(sp.kind);
  const filters = { q: one(sp.q), kind: isAssetKind(rawKind) ? rawKind : ("" as const), folder: one(sp.folder), tag: one(sp.tag) };
  const filtered = Boolean(filters.q || filters.kind || filters.folder || filters.tag);
  const res = await loadContent(() =>
    Promise.all([
      listFolders(client.id),
      searchAssets(client.id),
      filtered ? searchAssets(client.id, { query: filters.q, kind: filters.kind, folderId: filters.folder, tag: filters.tag }) : null,
    ]),
  );

  return (
    <>
      <PageHeader eyebrow={t.nav.content} title={c.assetsTitle} description={c.assetsDesc} />
      {!res.ok ? (
        <ContentOffline message={res.message} />
      ) : res.data[1].length === 0 && res.data[0].length === 0 && !canWrite ? (
        <EmptyState icon={<FolderOpen />} title={c.noAssets} description={c.noAssetsDesc} />
      ) : (
        <AssetLibrary
          slug={slug}
          clientId={client.id}
          folders={res.data[0]}
          assets={res.data[2] ?? res.data[1]}
          tags={Array.from(new Set(res.data[1].flatMap((a) => a.tags ?? []))).sort()}
          filters={filters}
          canWrite={canWrite}
        />
      )}
    </>
  );
}
