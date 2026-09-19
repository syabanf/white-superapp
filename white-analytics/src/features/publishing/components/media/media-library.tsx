"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Film, ImageIcon, Link2, Loader2, Search, Trash2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { EmptyState } from "@/components/dashboard/empty-state";
import { addExternalMedia, deleteMedia, updateMedia } from "@/features/publishing/actions-media";
import { MediaThumb, UploadButton } from "@/features/publishing/components/composer/media-picker";
import type { MediaRow } from "@/features/publishing/queries";
import type { PostRole } from "@/features/publishing/lib";
import { p } from "@/features/publishing/strings";
import { formatCompact, formatDateShort } from "@/lib/format";
import { t } from "@/i18n/id";

type Kind = "ALL" | "IMAGE" | "VIDEO";

export function MediaLibrary({
  clientId,
  assets,
  role,
}: {
  clientId: string;
  assets: MediaRow[];
  role: PostRole;
}) {
  const router = useRouter();
  const canManage = role !== "VIEWER";
  const [kind, setKind] = React.useState<Kind>("ALL");
  const [q, setQ] = React.useState("");
  const [editing, setEditing] = React.useState<MediaRow | null>(null);
  const [urlOpen, setUrlOpen] = React.useState(false);

  const filtered = assets.filter((a) => {
    if (kind !== "ALL" && a.kind !== kind) return false;
    if (!q.trim()) return true;
    const needle = q.toLowerCase();
    return (
      a.filename.toLowerCase().includes(needle) ||
      a.tags.some((tg) => tg.toLowerCase().includes(needle)) ||
      (a.altText ?? "").toLowerCase().includes(needle)
    );
  });

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div className="flex flex-wrap items-center gap-2">
          <Tabs value={kind} onValueChange={(v) => setKind(v as Kind)}>
            <TabsList className="h-8">
              <TabsTrigger value="ALL">{p.kindAll}</TabsTrigger>
              <TabsTrigger value="IMAGE">
                <ImageIcon className="size-3.5" /> {p.kindImage}
              </TabsTrigger>
              <TabsTrigger value="VIDEO">
                <Film className="size-3.5" /> {p.kindVideo}
              </TabsTrigger>
            </TabsList>
          </Tabs>
          <div className="relative">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder={p.librarySearch}
              className="h-8 w-56 pl-8 text-[13px]"
            />
          </div>
        </div>
        {canManage ? (
          <div className="flex items-center gap-2">
            <Button size="sm" variant="outline" onClick={() => setUrlOpen(true)}>
              <Link2 className="size-3.5" /> {p.addByUrl}
            </Button>
            <UploadButton clientId={clientId} variant="default" onUploaded={() => router.refresh()} />
          </div>
        ) : null}
      </div>

      {filtered.length === 0 ? (
        <EmptyState
          icon={<ImageIcon />}
          title={assets.length === 0 ? p.mediaEmpty : t.common.noResults}
          description={assets.length === 0 ? p.mediaEmptyDesc : undefined}
          compact
        />
      ) : (
        <ul className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
          {filtered.map((a) => (
            <li key={a.id}>
              <button
                type="button"
                onClick={() => setEditing(a)}
                className="lift block w-full overflow-hidden rounded-[1.5rem] bg-card shadow-(--card-shadow) text-left"
              >
                <MediaThumb asset={a} className="aspect-square" />
                <div className="space-y-1 px-2.5 py-2">
                  <p className="truncate text-xs font-medium">{a.filename}</p>
                  <p className="label-mono text-muted-foreground">
                    {formatCompact(a.sizeBytes / 1024, 0)} KB · {formatDateShort(a.createdAt)}
                  </p>
                  <p className="text-xs text-muted-foreground">{p.usedIn(a.usageCount)}</p>
                  {a.tags.length ? (
                    <div className="flex flex-wrap gap-1 pt-0.5">
                      {a.tags.slice(0, 3).map((tg) => (
                        <Badge key={tg} variant="secondary" className="min-h-5 px-1.5 text-xs">
                          {tg}
                        </Badge>
                      ))}
                    </div>
                  ) : null}
                </div>
              </button>
            </li>
          ))}
        </ul>
      )}

      {editing ? (
        <EditMediaDialog asset={editing} canManage={canManage} onClose={() => setEditing(null)} />
      ) : null}
      <AddUrlDialog clientId={clientId} open={urlOpen} onClose={() => setUrlOpen(false)} />
    </div>
  );
}

function EditMediaDialog({
  asset,
  canManage,
  onClose,
}: {
  asset: MediaRow;
  canManage: boolean;
  onClose: () => void;
}) {
  const router = useRouter();
  const [alt, setAlt] = React.useState(asset.altText ?? "");
  const [tags, setTags] = React.useState(asset.tags.join(", "));
  const [pending, start] = React.useTransition();
  const save = () =>
    start(async () => {
      const res = await updateMedia({
        id: asset.id,
        altText: alt,
        tags: tags
          .split(",")
          .map((s) => s.trim())
          .filter(Boolean),
      });
      if (res.ok) {
        toast.success(p.toast.mediaUpdated);
        router.refresh();
        onClose();
      } else toast.error(res.error);
    });
  const remove = () =>
    start(async () => {
      const res = await deleteMedia(asset.id);
      if (res.ok) {
        toast.success(p.toast.mediaDeleted);
        router.refresh();
        onClose();
      } else toast.error(res.error);
    });
  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{p.editMedia}</DialogTitle>
          <DialogDescription className="truncate">{asset.filename}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-5 sm:grid-cols-[160px_1fr]">
          <MediaThumb asset={asset} className="aspect-square rounded-lg" />
          <div className="space-y-3">
            <div className="space-y-1">
              <Label htmlFor="media-alt">{p.altText}</Label>
              <Input
                id="media-alt"
                value={alt}
                onChange={(e) => setAlt(e.target.value)}
                placeholder={p.altPlaceholder}
                disabled={!canManage || pending}
                className="h-8 text-sm"
              />
            </div>
            <div className="space-y-1">
              <Label htmlFor="media-tags">{p.tags}</Label>
              <Input
                id="media-tags"
                value={tags}
                onChange={(e) => setTags(e.target.value)}
                placeholder={p.tagsPlaceholder}
                disabled={!canManage || pending}
                className="h-8 text-sm"
              />
            </div>
            <p className="text-xs text-muted-foreground">
              {asset.mimeType} · {asset.width && asset.height ? `${asset.width}×${asset.height} · ` : ""}
              {p.usedIn(asset.usageCount)}
            </p>
            <a
              href={asset.url}
              target="_blank"
              rel="noreferrer"
              className="block truncate text-xs text-brand underline-offset-4 hover:underline"
            >
              {asset.url}
            </a>
          </div>
        </div>
        {canManage ? (
          <DialogFooter className="flex-row items-center justify-between sm:justify-between">
            <Button
              variant="destructive"
              size="sm"
              onClick={remove}
              disabled={pending || asset.usageCount > 0}
              title={asset.usageCount > 0 ? p.mediaInUse : undefined}
            >
              <Trash2 className="size-3.5" /> {t.common.delete}
            </Button>
            <Button size="sm" onClick={save} disabled={pending}>
              {pending ? <Loader2 className="size-3.5 animate-spin" /> : null} {t.common.save}
            </Button>
          </DialogFooter>
        ) : null}
        {canManage && asset.usageCount > 0 ? (
          <p className="text-xs text-muted-foreground">{p.mediaInUse}</p>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function AddUrlDialog({ clientId, open, onClose }: { clientId: string; open: boolean; onClose: () => void }) {
  const router = useRouter();
  const [url, setUrl] = React.useState("");
  const [kind, setKind] = React.useState<"IMAGE" | "VIDEO">("IMAGE");
  const [alt, setAlt] = React.useState("");
  const [pending, start] = React.useTransition();
  const submit = () =>
    start(async () => {
      const res = await addExternalMedia({ clientId, url: url.trim(), kind, altText: alt });
      if (res.ok) {
        toast.success(p.toast.mediaAdded);
        setUrl("");
        setAlt("");
        router.refresh();
        onClose();
      } else toast.error(res.error);
    });
  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{p.addByUrl}</DialogTitle>
          <DialogDescription>{p.addByUrlDesc}</DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <div className="space-y-1">
            <Label htmlFor="ext-url">{p.urlLabel}</Label>
            <Input
              id="ext-url"
              type="url"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://…"
              className="h-9"
              autoFocus
            />
          </div>
          <div className="grid gap-3 sm:grid-cols-[140px_1fr]">
            <div className="space-y-1">
              <Label>{p.kindLabel}</Label>
              <Select value={kind} onValueChange={(v) => setKind(v as "IMAGE" | "VIDEO")}>
                <SelectTrigger size="sm" className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="IMAGE">{p.kindImage}</SelectItem>
                  <SelectItem value="VIDEO">{p.kindVideo}</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label htmlFor="ext-alt">{p.altText}</Label>
              <Input
                id="ext-alt"
                value={alt}
                onChange={(e) => setAlt(e.target.value)}
                placeholder={p.altPlaceholder}
                className="h-8 text-sm"
              />
            </div>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" size="sm" onClick={onClose}>
            {t.common.cancel}
          </Button>
          <Button size="sm" onClick={submit} disabled={pending || !/^https:\/\//.test(url.trim())}>
            {pending ? <Loader2 className="size-3.5 animate-spin" /> : null} {t.common.add}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
