"use client";

import * as React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Folder, FolderPlus, Loader2, MoreHorizontal, Search, Upload } from "lucide-react";
import { toast } from "@/lib/toast";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { createFolder, deleteFolder, renameFolder } from "@/features/content/actions";
import { ASSET_KINDS, contentFiles, formatBytes, isAssetKind } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import type { Asset, AssetFolder, AssetKind } from "@/features/content/types";
import { cn } from "@/lib/utils";
import { FilePreview } from "../file-preview";
import { Field, OptionSelect } from "../fields";
import { useSetParam } from "../use-set-param";
import { UploadDialog } from "../upload-dialog";

type Filters = { q: string; kind: AssetKind | ""; folder: string; tag: string };

type Props = {
  slug: string;
  clientId: string;
  assets: Asset[];
  folders: AssetFolder[];
  tags: string[];
  filters: Filters;
  canWrite: boolean;
};

const ROOT = "__root";

export function AssetLibrary({ slug, clientId, assets, folders, tags, filters, canWrite }: Props) {
  const router = useRouter();
  const [query, setQuery] = React.useState(filters.q);
  const [upload, setUpload] = React.useState(false);
  const [folderDialog, setFolderDialog] = React.useState<{ id?: string; name: string } | null>(null);
  const [pending, start] = React.useTransition();
  const folderName = new Map(folders.map((f) => [f.id, f.name]));

  const setParam = useSetParam();

  React.useEffect(() => {
    if (query === filters.q) return;
    const h = setTimeout(() => setParam("q", query.trim()), 350);
    return () => clearTimeout(h);
  }, [query, filters.q, setParam]);

  const saveFolder = () => {
    if (!folderDialog) return;
    start(async () => {
      const res = folderDialog.id
        ? await renameFolder({ slug, folderId: folderDialog.id, name: folderDialog.name })
        : await createFolder({ slug, name: folderDialog.name });
      if (res.ok) {
        toast.success(folderDialog.id ? c.folderRenamed : c.folderCreated);
        setFolderDialog(null);
        router.refresh();
      } else toast.error(res.error);
    });
  };

  const removeFolder = (id: string) =>
    start(async () => {
      const res = await deleteFolder({ slug, folderId: id });
      if (res.ok) {
        toast.success(c.folderDeleted);
        if (filters.folder === id) setParam("folder", "");
        else router.refresh();
      } else toast.error(res.error);
    });

  // upload form state
  const [name, setName] = React.useState("");
  const [kind, setKind] = React.useState<AssetKind>("PHOTO");
  const [description, setDescription] = React.useState("");
  const [tagText, setTagText] = React.useState("");
  const [folderId, setFolderId] = React.useState(filters.folder);

  return (
    <div className="grid gap-6 lg:grid-cols-[220px_minmax(0,1fr)]">
      <nav aria-label={c.folders} className="space-y-1">
        <div className="flex items-center justify-between px-2 pb-1">
          <span className="label-mono text-muted-foreground">{c.folders}</span>
          {canWrite ? (
            <Button size="icon-xs" variant="ghost" onClick={() => setFolderDialog({ name: "" })} aria-label={c.newFolder}>
              <FolderPlus />
            </Button>
          ) : null}
        </div>
        <ul className="flex gap-1 overflow-x-auto pb-1 scrollbar-thin lg:flex-col lg:overflow-visible">
          {[{ id: "", name: c.allFolders, assetCount: undefined as number | undefined }, ...folders].map((f) => (
            <li key={f.id || "all"} className="group/folder flex shrink-0 items-center">
              <button
                type="button"
                onClick={() => setParam("folder", f.id)}
                className={cn(
                  "flex min-w-0 flex-1 items-center gap-2 rounded-lg px-2 py-1.5 text-left text-[13px] hover:bg-muted",
                  filters.folder === f.id && "bg-muted font-medium",
                )}
                aria-current={filters.folder === f.id ? "true" : undefined}
              >
                <Folder className="size-3.5 shrink-0 text-muted-foreground" />
                <span className="truncate">{f.name}</span>
                {f.assetCount !== undefined ? <span className="tabular ml-auto text-xs text-muted-foreground">{f.assetCount}</span> : null}
              </button>
              {canWrite && f.id ? (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button size="icon-xs" variant="ghost" className="opacity-60 group-hover/folder:opacity-100" aria-label={f.name}>
                      <MoreHorizontal />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onSelect={() => setFolderDialog({ id: f.id, name: f.name })}>{c.renameFolder}</DropdownMenuItem>
                    <DropdownMenuItem variant="destructive" onSelect={() => removeFolder(f.id)}>
                      {c.deleteFolder}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              ) : null}
            </li>
          ))}
        </ul>
      </nav>

      <div className="min-w-0 space-y-4">
        <div className="flex flex-wrap items-center gap-2">
          <div className="relative min-w-0 flex-1 sm:max-w-xs">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input value={query} onChange={(e) => setQuery(e.target.value)} placeholder={c.searchAssets} className="h-8 pl-8" aria-label={c.search} />
          </div>
          <OptionSelect
            label={c.allKinds}
            value={filters.kind}
            onChange={(v) => setParam("kind", v)}
            allLabel={c.allKinds}
            options={ASSET_KINDS.map((k) => ({ value: k, label: c.assetKind[k] }))}
            className="w-[150px]"
          />
          <OptionSelect
            label={c.allTags}
            value={filters.tag}
            onChange={(v) => setParam("tag", v)}
            allLabel={c.allTags}
            options={tags.map((t) => ({ value: t, label: `#${t}` }))}
            className="w-[150px]"
          />
          {pending ? <Loader2 className="size-4 animate-spin text-muted-foreground" /> : null}
          {canWrite ? (
            <Button size="sm" className="ml-auto" onClick={() => setUpload(true)}>
              <Upload className="size-3.5" /> {c.uploadAsset}
            </Button>
          ) : null}
        </div>

        {assets.length === 0 ? (
          <p className="rounded-xl border border-dashed px-3 py-10 text-center text-sm text-muted-foreground">{c.noAssetsFound}</p>
        ) : (
          <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-4">
            {assets.map((a) => (
              <li key={a.id}>
                <Link
                  href={`/clients/${slug}/content/assets/${a.id}`}
                  className="group block overflow-hidden rounded-xl bg-card shadow-(--card-shadow) outline-none focus-visible:ring-2 focus-visible:ring-ring"
                >
                  <div className="aspect-[4/3] overflow-hidden border-b">
                    <FilePreview
                      src={contentFiles.asset(clientId, a.id)}
                      mime={a.current?.mimeType}
                      name={a.current?.fileName ?? a.name}
                      controls={false}
                      className="transition-transform duration-200 group-hover:scale-[1.02]"
                    />
                  </div>
                  <div className="space-y-1 p-3">
                    <p className="truncate text-[13px] font-medium">{a.name}</p>
                    <p className="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
                      <Badge variant="secondary" className="h-4 px-1.5 text-[10px]">{c.assetKind[a.kind] ?? a.kind}</Badge>
                      <span className="tabular">{c.version(a.currentVersion)}</span>
                      {a.current ? <span className="tabular">· {formatBytes(a.current.sizeBytes)}</span> : null}
                    </p>
                    <p className="truncate text-xs text-muted-foreground">
                      {a.folderId ? folderName.get(a.folderId) : c.rootFolder}
                      {(a.tags ?? []).length ? ` · ${(a.tags ?? []).map((t) => `#${t}`).join(" ")}` : ""}
                    </p>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>

      <Dialog open={!!folderDialog} onOpenChange={(o) => !o && setFolderDialog(null)}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>{folderDialog?.id ? c.renameFolder : c.newFolder}</DialogTitle>
            <DialogDescription className="sr-only">{c.folderName}</DialogDescription>
          </DialogHeader>
          <Field id="folder-name" label={c.folderName}>
            <Input
              id="folder-name"
              value={folderDialog?.name ?? ""}
              onChange={(e) => setFolderDialog((s) => (s ? { ...s, name: e.target.value } : s))}
              autoFocus
            />
          </Field>
          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => setFolderDialog(null)}>
              {c.cancel}
            </Button>
            <Button size="sm" disabled={pending || !folderDialog?.name.trim()} onClick={saveFolder}>
              {c.save}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <UploadDialog
        open={upload}
        onOpenChange={setUpload}
        title={c.uploadAsset}
        description={c.uploadAssetDesc}
        url={contentFiles.createAsset(clientId)}
        fileField="file"
        success={c.assetUploaded}
        extra={() =>
          name.trim()
            ? { name: name.trim(), kind, description: description.trim(), tags: tagText, folderId }
            : c.assetNameRequired
        }
        onDone={(body) => {
          const id = (body as { asset?: { id: string } }).asset?.id;
          setName("");
          setDescription("");
          setTagText("");
          if (id) router.push(`/clients/${slug}/content/assets/${id}`);
          else router.refresh();
        }}
      >
        <Field id="as-name" label={c.assetName}>
          <Input id="as-name" value={name} onChange={(e) => setName(e.target.value)} />
        </Field>
        <div className="grid grid-cols-2 gap-3">
          <Field id="as-kind" label={c.kind}>
            <Select value={kind} onValueChange={(v) => isAssetKind(v) && setKind(v)}>
              <SelectTrigger id="as-kind" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {ASSET_KINDS.map((k) => (
                  <SelectItem key={k} value={k}>
                    {c.assetKind[k]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field id="as-folder" label={c.folder}>
            <Select value={folderId || ROOT} onValueChange={(v) => setFolderId(v === ROOT ? "" : v)}>
              <SelectTrigger id="as-folder" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ROOT}>{c.rootFolder}</SelectItem>
                {folders.map((f) => (
                  <SelectItem key={f.id} value={f.id}>
                    {f.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
        </div>
        <Field id="as-desc" label={c.description}>
          <Textarea id="as-desc" rows={2} value={description} onChange={(e) => setDescription(e.target.value)} />
        </Field>
        <Field id="as-tags" label={c.tags} hint={c.tagsHint}>
          <Input id="as-tags" value={tagText} onChange={(e) => setTagText(e.target.value)} />
        </Field>
      </UploadDialog>
    </div>
  );
}
