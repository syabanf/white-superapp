"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { History, Loader2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { reviseCaption } from "@/features/content/actions";
import { parseHashtags } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import type { CaptionRevision, Item } from "@/features/content/types";
import { formatRelative } from "@/lib/format";
import { CaptionAiDialog } from "@/features/brand/components/caption-ai-dialog";
import { formatHashtags } from "@/features/brand/lib";
import { Field } from "../fields";

/** Caption, hook and hashtags. Each save is a new revision; older versions can be loaded back into the editor. */
export function CaptionEditor({
  slug,
  item,
  revisions,
  editable,
}: {
  slug: string;
  item: Item;
  revisions: CaptionRevision[];
  editable: boolean;
}) {
  const router = useRouter();
  const [hook, setHook] = React.useState(item.hook);
  const [caption, setCaption] = React.useState(item.caption);
  const [tags, setTags] = React.useState((item.hashtags ?? []).join(" "));
  const [note, setNote] = React.useState("");
  const [pending, start] = React.useTransition();
  const dirty = hook !== item.hook || caption !== item.caption || tags !== (item.hashtags ?? []).join(" ");

  const save = (e: React.FormEvent) => {
    e.preventDefault();
    start(async () => {
      const res = await reviseCaption({ slug, itemId: item.id, hook, caption, hashtags: parseHashtags(tags), note });
      if (res.ok) {
        setNote("");
        toast.success(c.revisionSaved);
        router.refresh();
      } else toast.error(res.error);
    });
  };

  const load = (r: CaptionRevision) => {
    setHook(r.hook);
    setCaption(r.caption);
    setTags((r.hashtags ?? []).join(" "));
    setNote(`${c.restore}: ${c.version(r.version)}`);
  };

  return (
    <div className="page-stack">
      <form onSubmit={save} className="space-y-4 rounded-[1.5rem] bg-card p-(--card-pad) shadow-(--card-shadow)">
        {editable ? (
          <div className="flex justify-end">
            <CaptionAiDialog
              slug={slug}
              platform={item.platform}
              read={() => ({ hook, caption, hashtags: parseHashtags(tags) })}
              onApply={(v) => {
                setHook(v.hook);
                setCaption(v.caption);
                setTags(formatHashtags(v.hashtags));
              }}
              disabled={pending}
            />
          </div>
        ) : null}
        <Field id="cap-hook" label={c.hook}>
          <Input id="cap-hook" value={hook} onChange={(e) => setHook(e.target.value)} disabled={!editable} />
        </Field>
        <Field id="cap-body" label={`${c.caption} · ${c.chars(caption.length)}`}>
          <Textarea id="cap-body" rows={7} value={caption} onChange={(e) => setCaption(e.target.value)} disabled={!editable} className="leading-relaxed" />
        </Field>
        <Field id="cap-tags" label={c.hashtags} hint={c.hashtagsHint}>
          <Input id="cap-tags" value={tags} onChange={(e) => setTags(e.target.value)} disabled={!editable} />
        </Field>
        {editable ? (
          <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
            <Field id="cap-note" label={c.revisionNote} className="flex-1">
              <Input id="cap-note" value={note} onChange={(e) => setNote(e.target.value)} placeholder={c.revisionNotePlaceholder} />
            </Field>
            <Button type="submit" disabled={pending || !dirty}>
              {pending ? <Loader2 className="animate-spin" /> : null} {c.saveRevision}
            </Button>
          </div>
        ) : null}
      </form>

      <div className="rounded-[1.5rem] bg-card p-(--card-pad) shadow-(--card-shadow)">
        <h3 className="mb-3 flex items-center gap-1.5 text-sm font-semibold">
          <History className="size-3.5 text-muted-foreground" /> {c.revisions}
        </h3>
        {revisions.length === 0 ? (
          <p className="text-sm text-muted-foreground">{c.none}</p>
        ) : (
          <ol className="space-y-3">
            {revisions.map((r) => (
              <li key={r.id} className="rounded-lg border px-3 py-2">
                <div className="flex flex-wrap items-center gap-2 text-xs">
                  <Badge variant="secondary" className="tabular">{c.version(r.version)}</Badge>
                  {r.version === item.captionVersion ? <Badge variant="outline">{c.current}</Badge> : null}
                  <span className="font-medium">{r.createdByName || c.system}</span>
                  <span className="text-muted-foreground">· {formatRelative(r.createdAt)}</span>
                  {editable && r.version !== item.captionVersion ? (
                    <Button type="button" size="xs" variant="ghost" className="ml-auto" onClick={() => load(r)}>
                      {c.restore}
                    </Button>
                  ) : null}
                </div>
                {r.note ? <p className="mt-1 text-xs text-muted-foreground">{r.note}</p> : null}
                <p className="mt-1.5 line-clamp-3 text-[13px] whitespace-pre-wrap">{r.caption}</p>
              </li>
            ))}
          </ol>
        )}
      </div>
    </div>
  );
}
