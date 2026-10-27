"use client";

import * as React from "react";
import { useRouter } from "next/navigation";
import { Loader2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { PlatformIcon, platformLabel } from "@/features/social/components/platform-icon";
import { fromZoned, toZoned } from "@/features/publishing/time";
import { createItem, updateItem } from "@/features/content/actions";
import { FORMATS, PLATFORMS, parseHashtags } from "@/features/content/lib";
import { c } from "@/features/content/strings";
import type { ContentFormat, ItemInput, Person, Platform } from "@/features/content/types";
import { Field, PersonSelect } from "./fields";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  slug: string;
  timezone: string;
  people: Person[];
  pillars: string[];
  /** Create: plan id and a default date. Edit: item id with its current input. */
  mode: { kind: "create"; planId: string; date: string } | { kind: "edit"; itemId: string; initial: ItemInput };
};

function blank(date: string): ItemInput {
  return {
    plannedDate: date,
    platform: "INSTAGRAM",
    format: "FEED",
    pillar: "",
    title: "",
    visualBrief: "",
    copywriterId: "",
    designerId: "",
    copyDue: "",
    designDue: "",
    reviewDue: "",
  };
}

export function ItemFormDialog({ open, onOpenChange, slug, timezone, people, pillars, mode }: Props) {
  const router = useRouter();
  const [pending, start] = React.useTransition();
  const initial = mode.kind === "edit" ? mode.initial : blank(mode.date);
  const [v, setV] = React.useState<ItemInput>(initial);
  const [time, setTime] = React.useState(initial.publishAt ? toZoned(new Date(initial.publishAt), timezone).time : "");
  const [hook, setHook] = React.useState("");
  const [caption, setCaption] = React.useState("");
  const [hashtags, setHashtags] = React.useState("");
  const [errors, setErrors] = React.useState<Record<string, string[]>>();
  const set = <K extends keyof ItemInput>(k: K, val: ItemInput[K]) => setV((s) => ({ ...s, [k]: val }));

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const item: ItemInput = {
      ...v,
      publishAt: time && v.plannedDate ? fromZoned(v.plannedDate, time, timezone).toISOString() : undefined,
    };
    start(async () => {
      const res =
        mode.kind === "create"
          ? await createItem({ slug, planId: mode.planId, item, hook, caption, hashtags: parseHashtags(hashtags) })
          : await updateItem({ slug, itemId: mode.itemId, item });
      if (!res.ok) {
        setErrors(res.fieldErrors);
        toast.error(res.error);
        return;
      }
      toast.success(mode.kind === "create" ? c.itemCreated : c.itemSaved);
      onOpenChange(false);
      router.refresh();
    });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{mode.kind === "create" ? c.itemNewTitle : c.itemEditTitle}</DialogTitle>
          <DialogDescription>{c.itemDesc}</DialogDescription>
        </DialogHeader>
        <form id="item-form" onSubmit={submit} className="grid items-start gap-4 sm:grid-cols-2">
          <Field id="it-title" label={c.itemTitle} error={errors?.title} className="sm:col-span-2">
            <Input id="it-title" value={v.title} onChange={(e) => set("title", e.target.value)} autoFocus />
          </Field>
          <Field id="it-date" label={c.plannedDate} error={errors?.plannedDate}>
            <Input id="it-date" type="date" value={v.plannedDate} onChange={(e) => set("plannedDate", e.target.value)} />
          </Field>
          <Field id="it-time" label={c.publishAt}>
            <Input id="it-time" type="time" step={300} value={time} onChange={(e) => setTime(e.target.value)} />
          </Field>
          <Field id="it-platform" label={c.platform}>
            <Select value={v.platform} onValueChange={(x) => set("platform", x as Platform)}>
              <SelectTrigger id="it-platform" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PLATFORMS.map((p) => (
                  <SelectItem key={p} value={p}>
                    <PlatformIcon platform={p} className="size-3.5" /> {platformLabel(p)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field id="it-format" label={c.formatLabel}>
            <Select value={v.format} onValueChange={(x) => set("format", x as ContentFormat)}>
              <SelectTrigger id="it-format" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {FORMATS.map((f) => (
                  <SelectItem key={f} value={f}>
                    {c.format[f]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </Field>
          <Field id="it-pillar" label={c.pillar} className="sm:col-span-2">
            <Input id="it-pillar" list="it-pillars" value={v.pillar} onChange={(e) => set("pillar", e.target.value)} />
            <datalist id="it-pillars">
              {pillars.map((p) => (
                <option key={p} value={p} />
              ))}
            </datalist>
          </Field>
          <Field id="it-brief" label={c.visualBrief} className="sm:col-span-2">
            <Textarea id="it-brief" rows={2} value={v.visualBrief} onChange={(e) => set("visualBrief", e.target.value)} />
          </Field>
          {mode.kind === "create" ? (
            <>
              <Field id="it-hook" label={c.hook} className="sm:col-span-2">
                <Input id="it-hook" value={hook} onChange={(e) => setHook(e.target.value)} />
              </Field>
              <Field id="it-caption" label={c.caption} className="sm:col-span-2">
                <Textarea id="it-caption" rows={3} value={caption} onChange={(e) => setCaption(e.target.value)} />
              </Field>
              <Field id="it-tags" label={c.hashtags} hint={c.hashtagsHint} className="sm:col-span-2">
                <Input id="it-tags" value={hashtags} onChange={(e) => setHashtags(e.target.value)} />
              </Field>
            </>
          ) : null}
          <Field id="it-copywriter" label={c.copywriter}>
            <PersonSelect id="it-copywriter" value={v.copywriterId} onChange={(x) => set("copywriterId", x)} people={people} />
          </Field>
          <Field id="it-designer" label={c.designer}>
            <PersonSelect id="it-designer" value={v.designerId} onChange={(x) => set("designerId", x)} people={people} />
          </Field>
          <div className="grid gap-4 sm:col-span-2 sm:grid-cols-3">
            <Field id="it-copydue" label={c.copyDue}>
              <Input id="it-copydue" type="date" value={v.copyDue} onChange={(e) => set("copyDue", e.target.value)} />
            </Field>
            <Field id="it-designdue" label={c.designDue}>
              <Input id="it-designdue" type="date" value={v.designDue} onChange={(e) => set("designDue", e.target.value)} />
            </Field>
            <Field id="it-reviewdue" label={c.reviewDue}>
              <Input id="it-reviewdue" type="date" value={v.reviewDue} onChange={(e) => set("reviewDue", e.target.value)} />
            </Field>
          </div>
        </form>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {c.cancel}
          </Button>
          <Button type="submit" form="item-form" disabled={pending}>
            {pending ? <Loader2 className="size-3.5 animate-spin" /> : null} {c.save}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
