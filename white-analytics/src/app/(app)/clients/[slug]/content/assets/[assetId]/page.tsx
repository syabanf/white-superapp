import type { Metadata } from "next";
import { Download } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { DetailLayout, KeyValueList } from "@/components/dashboard/layout";
import { AssetEditor } from "@/features/content/components/assets/asset-editor";
import { FilePreview } from "@/features/content/components/file-preview";
import { ContentOffline } from "@/features/content/components/offline";
import { contentFiles, formatBytes } from "@/features/content/lib";
import { contentContext, getAsset, listFolders, loadContent } from "@/features/content/queries";
import { c } from "@/features/content/strings";
import { formatDateTime } from "@/lib/format";
import { t } from "@/i18n/id";
import { withClientMode } from "@/lib/client-mode";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content/assets/[assetId]">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await contentContext(slug);
  return { title: `${client.name} · ${t.nav.contentAssets}` };
}

async function AssetDetailPageInner(props: PageProps<"/clients/[slug]/content/assets/[assetId]">) {
  const { slug, assetId } = await props.params;
  const { client, canWrite, names } = await contentContext(slug);
  const back = { href: `/clients/${slug}/content/assets`, label: t.nav.contentAssets };
  const res = await loadContent(() => Promise.all([getAsset(client.id, assetId), listFolders(client.id)]));
  if (!res.ok) {
    return (
      <>
        <PageHeader eyebrow={c.assetEyebrow} title={c.assetsTitle} back={back} />
        <ContentOffline message={res.message} />
      </>
    );
  }

  const [{ asset, versions = [] }, folders] = res.data;
  const tz = client.timezone;
  const folder = folders.find((f) => f.id === asset.folderId)?.name ?? c.rootFolder;
  const currentHref = contentFiles.asset(client.id, asset.id);

  return (
    <>
      <PageHeader
        back={back}
        eyebrow={c.assetEyebrow}
        title={asset.name}
        description={asset.description || `${c.assetKind[asset.kind]} · ${folder}`}
        actions={
          <>
            <Badge variant="secondary">{c.assetKind[asset.kind]}</Badge>
            <Button asChild variant="outline">
              <a href={currentHref} download={asset.current?.fileName}>
                <Download /> {c.download}
              </a>
            </Button>
          </>
        }
      />

      <DetailLayout
        aside={
          <>
            <Section index="03" title={c.details}>
              {canWrite ? (
                <AssetEditor key={asset.updatedAt} slug={slug} clientId={client.id} asset={asset} folders={folders} />
              ) : (
                <div className="rounded-[1.5rem] bg-card p-(--card-pad) shadow-(--card-shadow)">
                  <KeyValueList
                    items={[
                      { key: "folder", label: c.folder, value: folder },
                      {
                        key: "tags",
                        label: c.tags,
                        value: (asset.tags ?? []).length ? (
                          <span className="flex flex-wrap gap-1">
                            {asset.tags!.map((tg) => (
                              <Badge key={tg} variant="secondary">
                                #{tg}
                              </Badge>
                            ))}
                          </span>
                        ) : (
                          c.none
                        ),
                      },
                    ]}
                  />
                </div>
              )}
            </Section>
          </>
        }
      >
        <Section index="01" title={c.preview}>
          <div className="aspect-[4/3] overflow-hidden rounded-[1.5rem] bg-card shadow-(--card-shadow)">
            <FilePreview src={currentHref} mime={asset.current?.mimeType} name={asset.current?.fileName ?? asset.name} />
          </div>
        </Section>
        <Section index="02" title={c.versionHistory}>
          <ol className="divide-y rounded-[1.5rem] bg-card shadow-(--card-shadow)">
            {versions.map((v) => (
              <li key={v.id} className="flex flex-wrap items-center gap-x-3 gap-y-1 px-(--card-pad) py-3 text-sm">
                <Badge variant={v.version === asset.currentVersion ? "default" : "secondary"} className="tabular">
                  {c.version(v.version)}
                </Badge>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-[13px] font-medium">{v.fileName}</span>
                  <span className="block text-xs text-muted-foreground">
                    {formatBytes(v.sizeBytes)} · {v.uploadedBy ? (names[v.uploadedBy] ?? c.none) : c.system} · {formatDateTime(v.createdAt, tz)}
                    {v.note ? ` · ${v.note}` : ""}
                  </span>
                </span>
                <Button asChild size="icon-sm" variant="ghost">
                  <a href={contentFiles.asset(client.id, asset.id, v.version)} download={v.fileName} aria-label={`${c.download} ${c.version(v.version)}`}>
                    <Download />
                  </a>
                </Button>
              </li>
            ))}
          </ol>
        </Section>
      </DetailLayout>
    </>
  );
}

/** Klien demo dirender dalam mode demo: adapter mock + DemoBanner walau kredensial asli terpasang. */
export default async function AssetDetailPage(props: Parameters<typeof AssetDetailPageInner>[0]) {
  const { slug } = await props.params;
  return withClientMode({ slug }, () => AssetDetailPageInner(props));
}
