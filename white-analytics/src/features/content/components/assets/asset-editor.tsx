"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2, Trash2, Upload } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { deleteAsset, updateAsset } from "@/features/content/actions";
import { ASSET_KINDS, contentFiles, isAssetKind, splitList } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import type { Asset, AssetFolder, AssetKind } from "@/features/content/types";
import { Field } from "../fields";
import { UploadDialog } from "../upload-dialog";

const ROOT = "__root";

/** Metadata form, new version upload and soft delete. Members who may write only. */
export function AssetEditor({ slug, clientId, asset, folders }: { slug: string; clientId: string; asset: Asset; folders: AssetFolder[] }) {
  const router = useRouter();
  const [name, setName] = React.useState(asset.name);
  const [kind, setKind] = React.useState<AssetKind>(asset.kind);
  const [description, setDescription] = React.useState(asset.description);
  const [tags, setTags] = React.useState((asset.tags ?? []).join(", "));
  const [folderId, setFolderId] = React.useState(asset.folderId ?? "");
  const [dialog, setDialog] = React.useState<"version" | "delete" | null>(null);
  const [pending, start] = React.useTransition();

  const save = (e: React.FormEvent) => {
    e.preventDefault();
    start(async () => {
      const res = await updateAsset({ slug, assetId: asset.id, name, kind, description, tags: splitList(tags), folderId });
      if (res.ok) {
        toast.success(c.assetSaved);
        router.refresh();
      } else toast.error(res.error);
    });
  };

  const remove = () =>
    start(async () => {
      const res = await deleteAsset({ slug, assetId: asset.id });
      if (res.ok) {
        toast.success(c.assetDeleted);
        router.push(`/clients/${slug}/content/assets`);
      } else toast.error(res.error);
    });

  return (
    <>
      <form onSubmit={save} className="space-y-4 rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow)">
        <Field id="ae-name" label={c.assetName}>
          <Input id="ae-name" value={name} onChange={(e) => setName(e.target.value)} />
        </Field>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field id="ae-kind" label={c.kind}>
            <Select value={kind} onValueChange={(v) => isAssetKind(v) && setKind(v)}>
              <SelectTrigger id="ae-kind" className="w-full">
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
          <Field id="ae-folder" label={c.folder}>
            <Select value={folderId || ROOT} onValueChange={(v) => setFolderId(v === ROOT ? "" : v)}>
              <SelectTrigger id="ae-folder" className="w-full">
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
        <Field id="ae-desc" label={c.description}>
          <Textarea id="ae-desc" rows={3} value={description} onChange={(e) => setDescription(e.target.value)} />
        </Field>
        <Field id="ae-tags" label={c.tags} hint={c.tagsHint}>
          <Input id="ae-tags" value={tags} onChange={(e) => setTags(e.target.value)} />
        </Field>
        <div className="flex flex-wrap items-center gap-2">
          <Button type="submit" size="sm" disabled={pending}>
            {pending ? <Loader2 className="size-3.5 animate-spin" /> : null} {c.save}
          </Button>
          <Button type="button" size="sm" variant="outline" onClick={() => setDialog("version")}>
            <Upload className="size-3.5" /> {c.newVersion}
          </Button>
          <Button type="button" size="sm" variant="destructive" className="ml-auto" onClick={() => setDialog("delete")}>
            <Trash2 className="size-3.5" /> {c.deleteAsset}
          </Button>
        </div>
      </form>

      <UploadDialog
        open={dialog === "version"}
        onOpenChange={(o) => setDialog(o ? "version" : null)}
        title={c.newVersion}
        description={c.newVersionDesc}
        url={contentFiles.assetVersion(clientId, asset.id)}
        fileField="file"
        noteLabel={c.versionNote}
        success={c.versionUploaded}
      />

      <Dialog open={dialog === "delete"} onOpenChange={(o) => !o && setDialog(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{c.deleteAsset}</DialogTitle>
            <DialogDescription>{c.deleteAssetDesc}</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" size="sm" onClick={() => setDialog(null)}>
              {c.cancel}
            </Button>
            <Button size="sm" variant="destructive" disabled={pending} onClick={remove}>
              {c.delete}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
