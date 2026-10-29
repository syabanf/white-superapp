"use client";

import * as React from "react";
import Link from "next/link";
import { Check, Download, ImageOff, Library, Loader2, RefreshCw } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { saveDraftToLibrary } from "@/features/design-studio/actions";
import { FORMAT_LABEL, FORMAT_SIZE, aspectRatio, designFilename, formatSizeLabel, isSvgDataUrl, type DraftRow } from "@/features/design-studio/lib";
import { ds } from "@/features/design-studio/strings";
import { formatRelative } from "@/lib/format";
import { LayoutBrief } from "./layout-brief";
import { rasterizeToPng } from "./rasterize";

/** Generated image at the format's aspect ratio + designer brief + library/regenerate actions. */
export function DesignResult({
  slug,
  draft,
  canManage,
  regenerating,
  onRegenerate,
  onUpdated,
}: {
  slug: string;
  draft: DraftRow;
  canManage: boolean;
  regenerating: boolean;
  onRegenerate: () => void;
  onUpdated: (d: DraftRow) => void;
}) {
  const [saving, start] = React.useTransition();
  const [brokenUrl, setBrokenUrl] = React.useState<string | null>(null);
  const broken = brokenUrl !== null && brokenUrl === draft.imageUrl;
  const { width, height } = FORMAT_SIZE[draft.format];
  const portrait = height > width;

  const save = () =>
    start(async () => {
      if (!draft.imageUrl) return;
      try {
        const fd = new FormData();
        fd.set("draftId", draft.id);
        if (isSvgDataUrl(draft.imageUrl)) {
          fd.set("file", await rasterizeToPng(draft.imageUrl, width, height, designFilename(draft.title, draft.format, "png")));
        }
        const res = await saveDraftToLibrary(fd);
        if (res.ok) {
          onUpdated(res.data.draft);
          toast.success(ds.savedToLibrary);
        } else toast.error(res.error);
      } catch (e) {
        console.error(e);
        toast.error(ds.rasterizeFailed);
      }
    });

  return (
    <div className="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
      <section className="flex flex-col gap-4 rounded-[1.5rem] bg-card p-6 shadow-(--card-shadow)">
        <header className="min-w-0">
          <h3 className="truncate text-base font-semibold tracking-[-0.01em]" title={draft.title}>
            {draft.title}
          </h3>
          <p className="text-xs text-muted-foreground">
            {FORMAT_LABEL[draft.format]} · {formatSizeLabel(draft.format)} · {formatRelative(draft.createdAt)} · {ds.model} {draft.model}
          </p>
        </header>
        <div className="flex justify-center rounded-[1.25rem] bg-muted/50 p-3">
          <div className={portrait ? "w-full max-w-[18rem]" : "w-full max-w-[26rem]"} style={{ aspectRatio: aspectRatio(draft.format) }}>
            {draft.imageUrl && !broken ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={draft.imageUrl}
                alt={draft.layout.headline || draft.title}
                className="size-full rounded-xl object-cover shadow-(--card-shadow)"
                onError={() => setBrokenUrl(draft.imageUrl)}
              />
            ) : (
              <div className="flex size-full items-center justify-center rounded-xl bg-card text-muted-foreground">
                <ImageOff className="size-6" />
              </div>
            )}
          </div>
        </div>
        <div className="mt-auto flex flex-wrap gap-2">
          {canManage ? (
            draft.mediaAssetId ? (
              <Button asChild variant="secondary" size="sm">
                <Link href={`/clients/${slug}/publish/media`}>
                  <Check /> {ds.inLibrary}
                </Link>
              </Button>
            ) : (
              <Button type="button" size="sm" onClick={save} disabled={saving || !draft.imageUrl}>
                {saving ? <Loader2 className="animate-spin" /> : <Library />} {ds.saveToLibrary}
              </Button>
            )
          ) : null}
          {canManage ? (
            <Button type="button" variant="secondary" size="sm" onClick={onRegenerate} disabled={regenerating}>
              {regenerating ? <Loader2 className="animate-spin" /> : <RefreshCw />} {ds.regenerate}
            </Button>
          ) : null}
          {draft.imageUrl ? (
            <Button asChild variant="ghost" size="sm">
              <a href={draft.imageUrl} download={designFilename(draft.title, draft.format, isSvgDataUrl(draft.imageUrl) ? "svg" : "png")}>
                <Download /> {ds.download}
              </a>
            </Button>
          ) : null}
        </div>
      </section>
      <LayoutBrief draft={draft} />
    </div>
  );
}
