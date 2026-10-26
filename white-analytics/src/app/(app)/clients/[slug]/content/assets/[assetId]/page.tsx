import type { Metadata } from "next";
import Link from "next/link";
import { ArrowLeft, Download } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { PageHeader, Section } from "@/components/dashboard/page-header";
import { AssetEditor } from "@/features/content/components/assets/asset-editor";
import { FilePreview } from "@/features/content/components/file-preview";
import { ContentOffline } from "@/features/content/components/offline";
import { contentFiles, formatBytes } from "@/features/content/lib";
import { contentContext, getAsset, listFolders, loadContent } from "@/features/content/queries";
import { c } from "@/features/content/strings";
import { formatDateTime } from "@/lib/format";
import { t } from "@/i18n/id";

export async function generateMetadata(props: PageProps<"/clients/[slug]/content/assets/[assetId]">): Promise<Metadata> {
  const { slug } = await props.params;
  const { client } = await contentContext(slug);
  return { title: `${client.name} · ${t.nav.contentAssets}` };
}

export default async function AssetDetailPage(props: PageProps<"/clients/[slug]/content/assets/[assetId]">) {
  const { slug, assetId } = await props.params;
  const { client, canWrite, names } = await contentContext(slug);
  const back = (
    <Button asChild variant="outline" size="sm" className="group/nudge">
      <Link href={`/clients/${slug}/content/assets`}>
        <ArrowLeft className="nudge nudge-back size-3.5" /> {t.nav.contentAssets}
      </Link>
    </Button>
  );
  const res = await loadContent(() => Promise.all([getAsset(client.id, assetId), listFolders(client.id)]));
  if (!res.ok) {
    return (
      <>
        <PageHeader eyebrow={c.assetEyebrow} title={c.assetsTitle} actions={back} />
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
        eyebrow={c.assetEyebrow}
        title={asset.name}
        description={asset.description || `${c.assetKind[asset.kind]} · ${folder}`}
        actions={
          <>
            <Badge variant="secondary">{c.assetKind[asset.kind]}</Badge>
            <Button asChild size="sm" variant="outline">
              <a href={currentHref} download={asset.current?.fileName}>
                <Download className="size-3.5" /> {c.download}
              </a>
            </Button>
            {back}
          </>
        }
      />

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_380px]">
        <div className="min-w-0 space-y-6">
          <div className="aspect-[4/3] overflow-hidden rounded-[1.5rem] bg-card shadow-(--card-shadow)">
            <FilePreview src={currentHref} mime={asset.current?.mimeType} name={asset.current?.fileName ?? asset.name} />
          </div>
          <Section index="01" title={c.versionHistory}>
            <ol className="divide-y rounded-[1.5rem] bg-card shadow-(--card-shadow)">
              {versions.map((v) => (
                <li key={v.id} className="flex flex-wrap items-center gap-x-3 gap-y-1 px-5 py-3 text-sm">
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
        </div>
        <aside className="min-w-0 space-y-6">
          <Section index="02" title={c.details}>
            {canWrite ? (
              <AssetEditor key={asset.updatedAt} slug={slug} clientId={client.id} asset={asset} folders={folders} />
            ) : (
              <dl className="space-y-3 rounded-[1.5rem] bg-card p-6 text-sm shadow-(--card-shadow)">
                <div>
                  <dt className="label-mono text-muted-foreground">{c.folder}</dt>
                  <dd className="mt-0.5">{folder}</dd>
                </div>
                <div>
                  <dt className="label-mono text-muted-foreground">{c.tags}</dt>
                  <dd className="mt-1 flex flex-wrap gap-1">
                    {(asset.tags ?? []).length ? asset.tags!.map((tg) => <Badge key={tg} variant="secondary">#{tg}</Badge>) : c.none}
                  </dd>
                </div>
              </dl>
            )}
          </Section>
        </aside>
      </div>
    </>
  );
}
