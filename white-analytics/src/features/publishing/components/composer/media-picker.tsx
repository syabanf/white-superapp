"use client";

import * as React from "react";
import { ArrowDown, ArrowUp, Film, ImagePlus, Search, Upload, X } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { uploadMedia } from "@/features/publishing/actions-media";
import type { MediaRow } from "@/features/publishing/queries";
import { p } from "@/features/publishing/strings";
import { cn } from "@/lib/utils";

/* eslint-disable @next/next/no-img-element */

export function MediaThumb({ asset, className }: { asset: Pick<MediaRow, "url" | "thumbnailUrl" | "kind" | "altText" | "filename">; className?: string }) {
  const src = asset.thumbnailUrl ?? asset.url;
  return (
    <div className={cn("relative overflow-hidden bg-muted", className)}>
      {asset.kind === "VIDEO" && !asset.thumbnailUrl ? (
        <video src={asset.url} muted playsInline preload="metadata" className="size-full object-cover" />
      ) : (
        <img src={src} alt={asset.altText ?? asset.filename} loading="lazy" className="size-full object-cover" />
      )}
      {asset.kind === "VIDEO" ? (
        <span className="absolute right-1 bottom-1 inline-flex items-center gap-1 rounded-full bg-background/90 px-1.5 py-0.5 text-[10px] font-medium">
          <Film className="size-3" /> Video
        </span>
      ) : null}
    </div>
  );
}

/** Hidden file input + button; uploads through the server action and returns the created rows. */
export function UploadButton({
  clientId,
  onUploaded,
  size = "sm",
  variant = "outline",
  disabled,
}: {
  clientId: string;
  onUploaded: (assets: MediaRow[]) => void;
  size?: "xs" | "sm" | "default";
  variant?: "outline" | "default" | "secondary";
  disabled?: boolean;
}) {
  const ref = React.useRef<HTMLInputElement>(null);
  const [pending, start] = React.useTransition();
  const onFiles = (files: FileList | null) => {
    if (!files || files.length === 0) return;
    const fd = new FormData();
    fd.set("clientId", clientId);
    for (const f of Array.from(files)) fd.append("files", f);
    start(async () => {
      const res = await uploadMedia(fd);
      if (res.ok) {
        toast.success(p.toast.mediaUploaded(res.data.assets.length));
        onUploaded(res.data.assets);
      } else toast.error(res.error);
      if (ref.current) ref.current.value = "";
    });
  };
  return (
    <>
      <input ref={ref} type="file" multiple accept="image/jpeg,image/png,image/webp,image/gif,video/mp4,video/quicktime" className="hidden" onChange={(e) => onFiles(e.target.files)} />
      <Button type="button" size={size} variant={variant} onClick={() => ref.current?.click()} disabled={pending || disabled}>
        {pending ? <Spinner className="size-3.5" /> : <Upload className="size-3.5" />}
        {pending ? p.uploading : p.upload}
      </Button>
    </>
  );
}

export function MediaPicker({
  clientId,
  library,
  onLibraryAdd,
  selected,
  onChange,
  alt,
  onAlt,
  disabled,
}: {
  clientId: string;
  library: MediaRow[];
  onLibraryAdd: (assets: MediaRow[]) => void;
  selected: MediaRow[];
  onChange: (assets: MediaRow[]) => void;
  alt: Record<string, string>;
  onAlt: (id: string, v: string) => void;
  disabled?: boolean;
}) {
  const [open, setOpen] = React.useState(false);
  const [q, setQ] = React.useState("");
  const [picked, setPicked] = React.useState<string[]>([]);

  const openDialog = () => {
    setPicked(selected.map((s) => s.id));
    setQ("");
    setOpen(true);
  };
  const confirm = () => {
    const byId = new Map(library.map((m) => [m.id, m]));
    const kept = selected.filter((s) => picked.includes(s.id));
    const added = picked.filter((id) => !selected.some((s) => s.id === id)).map((id) => byId.get(id)).filter((m): m is MediaRow => Boolean(m));
    onChange([...kept, ...added].slice(0, 10));
    setOpen(false);
  };
  const move = (i: number, dir: -1 | 1) => {
    const j = i + dir;
    if (j < 0 || j >= selected.length) return;
    const next = [...selected];
    [next[i], next[j]] = [next[j]!, next[i]!];
    onChange(next);
  };
  const filtered = library.filter((m) => {
    if (!q.trim()) return true;
    const needle = q.toLowerCase();
    return m.filename.toLowerCase().includes(needle) || m.tags.some((tg) => tg.toLowerCase().includes(needle)) || (m.altText ?? "").toLowerCase().includes(needle);
  });

  return (
    <div className="space-y-3">
      {selected.length === 0 ? (
        <p className="rounded-lg border border-dashed px-3 py-4 text-sm text-muted-foreground">{p.noMediaSelected}</p>
      ) : (
        <ul className="space-y-2">
          {selected.map((m, i) => (
            <li key={m.id} className="flex items-start gap-3 rounded-lg border p-2">
              <MediaThumb asset={m} className="size-16 shrink-0 rounded-md" />
              <div className="min-w-0 flex-1 space-y-1.5">
                <p className="truncate text-xs font-medium">{m.filename}</p>
                <Input
                  value={alt[m.id] ?? m.altText ?? ""}
                  onChange={(e) => onAlt(m.id, e.target.value)}
                  placeholder={p.altPlaceholder}
                  aria-label={p.altText}
                  disabled={disabled}
                  className="h-8 text-xs"
                />
              </div>
              <div className="flex shrink-0 flex-col gap-0.5">
                <Button type="button" variant="ghost" size="icon-xs" onClick={() => move(i, -1)} disabled={disabled || i === 0} aria-label={p.moveUp}>
                  <ArrowUp />
                </Button>
                <Button type="button" variant="ghost" size="icon-xs" onClick={() => move(i, 1)} disabled={disabled || i === selected.length - 1} aria-label={p.moveDown}>
                  <ArrowDown />
                </Button>
                <Button type="button" variant="ghost" size="icon-xs" onClick={() => onChange(selected.filter((s) => s.id !== m.id))} disabled={disabled} aria-label={p.remove}>
                  <X />
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}
      <div className="flex flex-wrap items-center gap-2">
        <Button type="button" size="sm" variant="outline" onClick={openDialog} disabled={disabled}>
          <ImagePlus className="size-3.5" /> {p.pickFromLibrary}
        </Button>
        <UploadButton
          clientId={clientId}
          disabled={disabled}
          onUploaded={(assets) => {
            onLibraryAdd(assets);
            onChange([...selected, ...assets].slice(0, 10));
          }}
        />
        <span className="text-xs text-muted-foreground">{p.uploadHint}</span>
      </div>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-h-[85vh] overflow-hidden sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>{p.selectMedia}</DialogTitle>
            <DialogDescription>{p.selected(picked.length)}</DialogDescription>
          </DialogHeader>
          <div className="relative">
            <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input value={q} onChange={(e) => setQ(e.target.value)} placeholder={p.librarySearch} className="h-8 pl-8 text-[13px]" />
          </div>
          <div className="max-h-[50vh] overflow-y-auto scrollbar-thin">
            {filtered.length === 0 ? (
              <p className="py-10 text-center text-sm text-muted-foreground">{p.libraryEmpty}</p>
            ) : (
              <ul className="grid grid-cols-3 gap-2 sm:grid-cols-4">
                {filtered.map((m) => {
                  const on = picked.includes(m.id);
                  return (
                    <li key={m.id}>
                      <button
                        type="button"
                        onClick={() => setPicked((cur) => (on ? cur.filter((x) => x !== m.id) : [...cur, m.id]))}
                        className={cn("lift block w-full overflow-hidden rounded-lg border text-left", on && "ring-2 ring-brand ring-offset-1")}
                        aria-pressed={on}
                      >
                        <MediaThumb asset={m} className="aspect-square" />
                        <span className="block truncate px-1.5 py-1 text-[11px] text-muted-foreground">{m.filename}</span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
          <DialogFooter className="flex-row items-center justify-between sm:justify-between">
            <UploadButton clientId={clientId} size="xs" onUploaded={(assets) => { onLibraryAdd(assets); setPicked((cur) => [...cur, ...assets.map((a) => a.id)]); }} />
            <Button type="button" size="sm" onClick={confirm}>
              {p.done}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
