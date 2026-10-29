"use client";

import * as React from "react";
import { ImageOff, Library, Loader2, Trash2 } from "lucide-react";
import { toast } from "@/lib/toast";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { deleteDesignDraft } from "@/features/design-studio/actions";
import { FORMAT_LABEL, aspectRatio, type DraftRow } from "@/features/design-studio/lib";
import { ds } from "@/features/design-studio/strings";
import { formatRelative } from "@/lib/format";
import { cn } from "@/lib/utils";

function Thumb({ draft }: { draft: DraftRow }) {
  const [broken, setBroken] = React.useState(false);
  return draft.imageUrl && !broken ? (
    // eslint-disable-next-line @next/next/no-img-element
    <img src={draft.imageUrl} alt="" loading="lazy" onError={() => setBroken(true)} className="size-full object-cover" />
  ) : (
    <span className="flex size-full items-center justify-center text-muted-foreground">
      <ImageOff className="size-5" />
    </span>
  );
}

function DeleteButton({ draft, onDeleted }: { draft: DraftRow; onDeleted: (id: string) => void }) {
  const [pending, start] = React.useTransition();
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button type="button" variant="ghost" size="icon-xs" aria-label={ds.delete} disabled={pending}>
          {pending ? <Loader2 className="animate-spin" /> : <Trash2 />}
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{ds.delete}</AlertDialogTitle>
          <AlertDialogDescription>{ds.deleteConfirm}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{ds.cancel}</AlertDialogCancel>
          <AlertDialogAction
            onClick={() =>
              start(async () => {
                const res = await deleteDesignDraft(draft.id);
                if (res.ok) {
                  onDeleted(draft.id);
                  toast.success(ds.deleted);
                } else toast.error(res.error);
              })
            }
          >
            {ds.delete}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

/** Grid of past drafts; clicking one opens it in the result panel above. */
export function DraftHistory({
  drafts,
  selectedId,
  canManage,
  onSelect,
  onDeleted,
}: {
  drafts: DraftRow[];
  selectedId: string | null;
  canManage: boolean;
  onSelect: (d: DraftRow) => void;
  onDeleted: (id: string) => void;
}) {
  if (drafts.length === 0) return <p className="text-sm text-muted-foreground">{ds.historyEmpty}</p>;
  return (
    <ul className="grid grid-cols-2 gap-5 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6">
      {drafts.map((d) => (
        <li key={d.id} className="min-w-0">
          <div
            className={cn(
              "lift flex h-full flex-col overflow-hidden rounded-[1.25rem] bg-card shadow-(--card-shadow)",
              selectedId === d.id && "ring-2 ring-brand",
            )}
          >
            <button type="button" onClick={() => onSelect(d)} className="block w-full bg-muted/50 text-left outline-none focus-visible:ring-2 focus-visible:ring-ring">
              <span className="block w-full" style={{ aspectRatio: aspectRatio(d.format) }}>
                <Thumb draft={d} />
              </span>
            </button>
            <div className="flex items-start gap-1 px-3 pt-2 pb-3">
              <button type="button" onClick={() => onSelect(d)} className="min-w-0 flex-1 text-left outline-none">
                <span className="block truncate text-[13px] font-medium" title={d.title}>
                  {d.title}
                </span>
                <span className="flex items-center gap-1 text-xs text-muted-foreground">
                  {d.mediaAssetId ? <Library className="size-3 shrink-0" aria-label={ds.inLibrary} /> : null}
                  <span className="truncate">
                    {FORMAT_LABEL[d.format]} · {formatRelative(d.createdAt)}
                  </span>
                </span>
              </button>
              {canManage ? <DeleteButton draft={d} onDeleted={onDeleted} /> : null}
            </div>
          </div>
        </li>
      ))}
    </ul>
  );
}
